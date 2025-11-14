package models

type CreateTeamRequest struct {
    TeamName string       `json:"team_name"`
    Members  []TeamMember `json:"members"`
}

type SetUserActiveRequest struct {
    UserID   string `json:"user_id"`
    IsActive bool   `json:"is_active"`
}

type CreatePRRequest struct {
    PullRequestID   string `json:"pull_request_id"`
    PullRequestName string `json:"pull_request_name"`
    AuthorID        string `json:"author_id"`
}

type MergePRRequest struct {
    PullRequestID string `json:"pull_request_id"`
}

type ReassignReviewerRequest struct {
    PullRequestID string `json:"pull_request_id"`
    OldUserID     string `json:"old_user_id"`  
}

type GetUserPRsRequest struct {
    UserID string `json:"user_id"` 
}
 
type GetTeamRequest struct {
    TeamName string `json:"team_name"`  
}