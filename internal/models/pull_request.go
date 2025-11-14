package models

import "time"

type PullRequest struct {
    PullRequestID    string     `json:"pull_request_id" db:"pull_request_id"`
    PullRequestName  string     `json:"pull_request_name" db:"pull_request_name"`
    AuthorID         string     `json:"author_id" db:"author_id"`
    Status           string     `json:"status" db:"status"`
    AssignedReviewers []string  `json:"assigned_reviewers"` 
    CreatedAt        time.Time  `json:"-" db:"created_at"`
    MergedAt         *time.Time `json:"mergedAt,omitempty" db:"merged_at"`
}


type PullRequestShort struct {
    PullRequestID   string `json:"pull_request_id"`
    PullRequestName string `json:"pull_request_name"`
    AuthorID        string `json:"author_id"`
    Status          string `json:"status"`
}

