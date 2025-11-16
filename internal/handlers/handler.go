package handlers

import (
	"net/http"

	"github.com/lypolix/avito_test/internal/models"
	"github.com/lypolix/avito_test/internal/services"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *services.Service
}

func NewHandler(service *services.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SetupRoutes() *gin.Engine {
	router := gin.Default()
	h.setupRoutes(router)
	return router
}

func (h *Handler) SetupRoutesWithRouter(router *gin.Engine) {
	h.setupRoutes(router)
}

func (h *Handler) handleError(c *gin.Context, err error) {
	if bizErr, ok := err.(*services.BusinessError); ok {
		c.JSON(h.getHTTPStatus(bizErr.Code), models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    bizErr.Code,
				Message: bizErr.Message,
			},
		})
		return
	}

	c.JSON(http.StatusInternalServerError, models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: "Internal server error",
		},
	})
}

func (h *Handler) getHTTPStatus(errorCode string) int {
	switch errorCode {
	case services.ErrorTeamExists:
		return http.StatusBadRequest
	case services.ErrorNotFound:
		return http.StatusNotFound
	case services.ErrorInvalidTeam, services.ErrorUserInOtherTeam:
		return http.StatusBadRequest
	case services.ErrorPRMerged, services.ErrorNotAssigned, services.ErrorNoCandidate, services.ErrorPRExists:
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}
