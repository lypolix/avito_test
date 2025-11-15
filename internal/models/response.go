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

type BulkDeactivateResponse struct {
    TeamName          string               `json:"team_name"`
    DeactivatedUsers  []string             `json:"deactivated_users"`
    ReassignedPRs     []ReassignedPRDetail `json:"reassigned_prs"`
    FailedReassignments []FailedReassignment `json:"failed_reassignments,omitempty"`
}

type ReassignedPRDetail struct {
    PullRequestID string            `json:"pull_request_id"`
    Replacements  []UserReplacement `json:"replacements"`
}

type UserReplacement struct {
    OldUserID string `json:"old_user_id"`
    NewUserID string `json:"new_user_id,omitempty"`
}

type FailedReassignment struct {
    PullRequestID string `json:"pull_request_id"`
    OldUserID     string `json:"old_user_id"`
    Reason        string `json:"reason"`
}