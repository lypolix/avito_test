package models

type User struct {
    UserID   string `json:"user_id" db:"user_id"`
    Username string `json:"username" db:"username"`
    TeamName string `json:"team_name" db:"team_name"`
    IsActive bool   `json:"is_active" db:"is_active"`
}

type TeamMember struct {
    UserID   string `json:"user_id" db:"user_id"`
    Username string `json:"username" db:"username"`
    IsActive bool   `json:"is_active" db:"is_active"`
}		