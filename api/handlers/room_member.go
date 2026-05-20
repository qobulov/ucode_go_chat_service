package handlers

import (
	"net/http"
	"ucode/ucode_go_chat_service/models"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func (h *handler) RoomMemberCreate(c *gin.Context) {
	req := &models.CreateRoomMember{}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, err)
		return
	}

	roomMember, err := h.storage.Postgres().RoomMemberCreate(
		c.Request.Context(),
		req,
	)
	if err != nil {
		handleResponse(c, http.StatusInternalServerError, err)
		return
	}

	handleResponse(c, http.StatusCreated, roomMember)
}

func (h *handler) RoomMemberUpdate(c *gin.Context) {
	var body struct {
		models.UpdateRoomMember
		FromName string `json:"from_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, http.StatusBadRequest, err)
		return
	}
	if body.RoomId == "" || body.RowId == "" {
		handleResponse(c, http.StatusBadRequest, "room_id and row_id are required")
		return
	}

	ctx := c.Request.Context()

	member, err := h.storage.Postgres().RoomMemberUpdate(ctx, &body.UpdateRoomMember)
	if err != nil {
		handleResponse(c, http.StatusInternalServerError, err)
		return
	}

	toRowId := cast.ToString(body.ToRowId)
	if body.FromName != "" && toRowId != "" {
		_, err = h.storage.Postgres().RoomMemberUpdate(ctx, &models.UpdateRoomMember{
			RoomId: body.RoomId,
			RowId:  toRowId,
			ToName: body.FromName,
		})
		if err != nil {
			handleResponse(c, http.StatusInternalServerError, err)
			return
		}
	}

	handleResponse(c, http.StatusOK, member)
}
