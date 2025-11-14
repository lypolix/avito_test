package handlers

import (
	"github.com/lypolix/avito_test/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreatePR(c *gin.Context) {
	var req models.CreatePRRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "BAD_REQUEST",
				Message: "Invalid request body",
			},
		})
		return
	}

	pr, err := h.service.CreatePR(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.PRResponse{PR: pr})
}

func (h *Handler) MergePR(c *gin.Context) {
	var req models.MergePRRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "BAD_REQUEST",
				Message: "Invalid request body",
			},
		})
		return
	}

	pr, err := h.service.MergePR(req.PullRequestID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.PRResponse{PR: pr})
}

func (h *Handler) ReassignReviewer(c *gin.Context) {
	var req models.ReassignReviewerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "BAD_REQUEST",
				Message: "Invalid request body",
			},
		})
		return
	}

	response, err := h.service.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "OK",
	})
}