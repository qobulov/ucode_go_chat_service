package api

import (
	"context"
	"time"
	"ucode/ucode_go_chat_service/api/handlers"
	"ucode/ucode_go_chat_service/config"
	"ucode/ucode_go_chat_service/internal/socketio"
	"ucode/ucode_go_chat_service/pkg/logger"
	"ucode/ucode_go_chat_service/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(log *logger.Logger, cfg config.Config, strg storage.StorageI) *gin.Engine {
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	h := handlers.New(&handlers.HandlerConfig{
		Logger:   log,
		Cfg:      cfg,
		Postgres: strg,
	})

	router.Static("/static", "./public")
	router.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	api := router.Group("/v1")

	room := api.Group("/room")
	room.POST("", h.RoomCreate)
	room.GET("", h.RoomGetList)
	room.POST("/exist", h.RoomExists)
	room.GET("/:item_id", h.RoomIdByItemId)
	room.PATCH("/:id", h.RoomUpdate)
	room.DELETE("/:id", h.RoomDelete)

	roomMember := api.Group("/room-member")
	roomMember.POST("", h.RoomMemberCreate)
	roomMember.PATCH("", h.RoomMemberUpdate)

	message := api.Group("/message")
	message.GET("", h.MessageGetList)

	supervisor := api.Group("/supervisor")
	supervisor.GET("/rooms", h.SupervisorRooms)
	supervisor.GET("/messages", h.SupervisorMessages)

	io := socketio.New()
	socketIoHandle(io, strg, log)

	router.Any("/socket.io/*any", gin.WrapH(io.HttpHandler()))

	sweeperCtx := context.Background()
	startPresenceSweeper(sweeperCtx, strg, func(ev string, payload any) {
		io.Emit(ev, payload)
	})

	return router
}

func socketIoHandle(io *socketio.Io, strg storage.StorageI, log *logger.Logger) {
	handlers.NewSocketHandler(io, strg, log)
}

func startPresenceSweeper(
	ctx context.Context,
	strg storage.StorageI,
	emit func(event string, payload any),
) {
	const (
		interval   = 1 * time.Minute
		staleAfter = 1 * time.Minute
	)
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cutoff := time.Now().UTC().Add(-staleAfter)

				dbCtx, cancel := context.WithTimeout(ctx, time.Duration(config.DBTimeout)*time.Second)

				ids, err := strg.Postgres().PresenceSweepOffline(dbCtx, cutoff)
				cancel()

				if err != nil {
					continue
				}
				if len(ids) == 0 {
					continue
				}

				for _, id := range ids {
					emit("presence.updated", map[string]any{
						"row_id":       id,
						"status":       "offline",
						"last_seen_at": time.Now().UTC(),
					})
				}

			case <-ctx.Done():
				return
			}
		}
	}()
}
