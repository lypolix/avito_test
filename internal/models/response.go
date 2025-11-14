package models

type ErrorDetail struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type ErrorResponse struct {
    Error ErrorDetail `json:"error"`
}

type TeamResponse struct {
    Team *Team `json:"team"`
}

type UserResponse struct {
    User *User `json:"user"`
}

type PRResponse struct {
    PR *PullRequest `json:"pr"`
}

type ReassignResponse struct {
    PR         *PullRequest `json:"pr"`
    ReplacedBy string       `json:"replaced_by"`
}

type UserPRsResponse struct {
    UserID       string             `json:"user_id"`
    PullRequests []PullRequestShort `json:"pull_requests"`
}