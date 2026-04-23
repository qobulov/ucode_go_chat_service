package handlers

import (
	"errors"
	"net/http"
	"ucode/ucode_go_chat_service/models"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func (h *handler) RoomCreate(c *gin.Context) {
	req := &models.CreateRoom{}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, err)
		return
	}

	room, err := h.storage.Postgres().RoomCreate(
		c.Request.Context(), req,
	)
	if err != nil {
		handleResponse(c, http.StatusInternalServerError, err)
		return
	}

	_, err = h.storage.Postgres().RoomMemberCreate(
		c.Request.Context(),
		&models.CreateRoomMember{
			RoomId: room.Id,
			RowId:  req.RowId,
			ToName: req.ToName,
		},
	)
	if err != nil {
		handleResponse(c, http.StatusInternalServerError, err)
		return
	}

	toRowId := cast.ToString(req.ToRowId)

	if toRowId != "" && req.FromName != "" {
		_, err = h.storage.Postgres().RoomMemberCreate(
			c.Request.Context(),
			&models.CreateRoomMember{
				RoomId: room.Id,
				RowId:  toRowId,
				ToName: req.FromName,
			},
		)
		if err != nil {
			handleResponse(c, http.StatusInternalServerError, err)
			return
		}
	}

	handleResponse(c, http.StatusCreated, room)
}

func (h *handler) RoomGetList(c *gin.Context) {
	offset, err := ParseOffsetQueryParam(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, err)
		return
	}

	limit, err := ParseLimitQueryParam(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, err)
		return
	}
	if limit > 100 {
		limit = 100
	}

	rowId := c.Query("row_id")
	if rowId == "" {
		handleResponse(c, http.StatusBadRequest, errors.New("Row is required"))
		return
	}

	typeParam := c.Query("type")
	searchParam := c.Query("search")
	projectIDParam := c.Query("project_id")

	req := &models.GetListRoomReq{
		Offset:    uint64(offset),
		Limit:     uint64(limit),
		RowId:     rowId,
		Type:      typeParam,
		Search:    searchParam,
		ProjectId: projectIDParam,
	}

	rooms, err := h.storage.Postgres().RoomGetList(
		c.Request.Context(), req,
	)
	if err != nil {
		handleResponse(c, http.StatusInternalServerError, err)
		return
	}

	handleResponse(c, http.StatusOK, rooms)
}

func (h *handler) RoomExists(c *gin.Context) {
	req := &models.ExistsRoom{}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, err)
		return
	}

	resp, err := h.storage.Postgres().RoomExists(
		c.Request.Context(), req,
	)
	if err != nil {
		handleResponse(c, http.StatusInternalServerError, err)
		return
	}

	handleResponse(c, http.StatusOK, resp)
}

func (h *handler) RoomDelete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handleResponse(c, http.StatusBadRequest, "id is required")
		return
	}

	err := h.storage.Postgres().RoomDelete(c.Request.Context(), id)
	if err != nil {
		handleResponse(c, http.StatusInternalServerError, err)
		return
	}

	handleResponse(c, http.StatusOK, "room deleted")
}

func (h *handler) RoomIdByItemId(c *gin.Context) {
	itemId := c.Param("item_id")
	if itemId == "" {
		handleResponse(c, http.StatusBadRequest, "item_id is required")
		return
	}

	projectId := c.Query("project_id")
	if projectId == "" {
		handleResponse(c, http.StatusBadRequest, "project_id is required")
		return
	}

	req := &models.GetRoomIdByItemIdReq{
		ItemId:    itemId,
		ProjectId: projectId,
	}

	resp, err := h.storage.Postgres().RoomIdByItemId(
		c.Request.Context(), req,
	)
	if err != nil {
		handleResponse(c, http.StatusInternalServerError, err)
		return
	}

	handleResponse(c, http.StatusOK, resp)
}
