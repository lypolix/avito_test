package repository

import (
	"database/sql"

	"github.com/lypolix/avito_test/internal/models"
)

func (r *Repository) CreateUser(user *models.User) error {
	query := `INSERT INTO users (user_id, username, team_name, is_active) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, user.UserID, user.Username, user.TeamName, user.IsActive)
	return err
}

func (r *Repository) GetUser(userID string) (*models.User, error) {
	query := `SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1`
	var user models.User
	err := r.db.QueryRow(query, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *Repository) UpdateUserActive(userID string, isActive bool) error {
	query := `UPDATE users SET is_active = $1 WHERE user_id = $2`
	_, err := r.db.Exec(query, isActive, userID)
	return err
}

func (r *Repository) UpdateUserActiveInTx(tx *sql.Tx, userID string, isActive bool) error {
	query := `UPDATE users SET is_active = $1 WHERE user_id = $2`
	_, err := tx.Exec(query, isActive, userID)
	return err
}

func (r *Repository) GetActiveUsersByTeam(teamName string) ([]models.User, error) {
	query := `SELECT user_id, username, team_name, is_active 
	          FROM users WHERE team_name = $1 AND is_active = true`
	rows, err := r.db.Query(query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *Repository) GetUsersByTeam(teamName string) ([]models.User, error) {
	query := `SELECT user_id, username, team_name, is_active FROM users WHERE team_name = $1`
	rows, err := r.db.Query(query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *Repository) GetActiveUsersByTeamInTx(tx *sql.Tx, teamName string) ([]models.User, error) {
	query := `SELECT user_id, username, team_name, is_active 
	          FROM users WHERE team_name = $1 AND is_active = true`
	rows, err := tx.Query(query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *Repository) IsUserInOtherTeam(userID, teamName string) (bool, error) {
	query := `SELECT team_name FROM users WHERE user_id = $1`
	var existingTeamName string
	err := r.db.QueryRow(query, userID).Scan(&existingTeamName)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return existingTeamName != teamName, nil
}
