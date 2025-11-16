package repository

import (
	"database/sql"

	"github.com/lypolix/avito_test/internal/models"
)

func (r *Repository) CreateTeam(teamName string) error {
	query := `INSERT INTO teams (team_name) VALUES ($1)`
	_, err := r.db.Exec(query, teamName)
	return err
}

func (r *Repository) TeamExists(teamName string) (bool, error) {
	query := `SELECT 1 FROM teams WHERE team_name = $1`
	var exists int
	err := r.db.QueryRow(query, teamName).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists == 1, err
}

func (r *Repository) GetTeam(teamName string) (*models.Team, error) {
	exists, err := r.TeamExists(teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	users, err := r.GetUsersByTeam(teamName)
	if err != nil {
		return nil, err
	}

	var members []models.TeamMember
	for _, user := range users {
		members = append(members, models.TeamMember{
			UserID:   user.UserID,
			Username: user.Username,
			IsActive: user.IsActive,
		})
	}

	return &models.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}
