package handlers

import "github.com/gin-gonic/gin"

func (h *Handler) setupRoutes(router *gin.Engine) {
	router.POST("/team/add", h.CreateTeam)
	router.GET("/team/get", h.GetTeam)

	router.POST("/users/setIsActive", h.SetUserActive)
	router.POST("/users/bulkDeactivate", h.BulkDeactivateUsers)
	router.GET("/users/getReview", h.GetUserPRs)

	router.POST("/pullRequest/create", h.CreatePR)
	router.POST("/pullRequest/merge", h.MergePR)
	router.POST("/pullRequest/reassign", h.ReassignReviewer)

	router.GET("/stats", h.GetStats)

	router.GET("/health", h.HealthCheck)
}
