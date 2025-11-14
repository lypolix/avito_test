package handlers

import (
	"github.com/lypolix/avito_test/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) SetUserActive(c *gin.Context) {
	var req models.SetUserActiveRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "BAD_REQUEST",
				Message: "Invalid request body",
			},
		})
		return
	}

	user, err := h.service.SetUserActive(req.UserID, req.IsActive)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.UserResponse{User: user})
}

func (h *Handler) GetUserPRs(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "BAD_REQUEST",
				Message: "user_id query parameter is required",
			},
		})
		return
	}

	response, err := h.service.GetUserPRs(userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}