package handlers

import (
	"github.com/lypolix/avito_test/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateTeam(c *gin.Context) {
	var team models.Team
	if err := c.ShouldBindJSON(&team); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "BAD_REQUEST",
				Message: "Invalid request body",
			},
		})
		return
	}

	if err := h.service.CreateTeam(&team); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.TeamResponse{Team: &team})
}

func (h *Handler) GetTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "BAD_REQUEST",
				Message: "team_name query parameter is required",
			},
		})
		return
	}

	team, err := h.service.GetTeam(teamName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, team)
}