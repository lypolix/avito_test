package models

type CreateTeamRequest struct {
	TeamName string       `json:"team_name" binding:"required"`
	Members  []TeamMember `json:"members" binding:"required"`
}

type SetUserActiveRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	IsActive bool   `json:"is_active"`
}

type BulkDeactivateRequest struct {
	TeamName string   `json:"team_name" binding:"required"`
	UserIDs  []string `json:"user_ids" binding:"required"`
}

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id" binding:"required"`
	PullRequestName string `json:"pull_request_name" binding:"required"`
	AuthorID        string `json:"author_id" binding:"required"`
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
}

type ReassignReviewerRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
	OldUserID     string `json:"old_user_id" binding:"required"`
}

type GetUserPRsRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

type GetTeamRequest struct {
	TeamName string `json:"team_name" binding:"required"`
}
