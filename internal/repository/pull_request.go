package repository

import (
	"github.com/lypolix/avito_test/internal/models"
	"database/sql"
	"time"
)

func (r *Repository) CreatePR(pr *models.PullRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) 
	          VALUES ($1, $2, $3, $4)`
	_, err = tx.Exec(query, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status)
	if err != nil {
		return err
	}

	for _, reviewerID := range pr.AssignedReviewers {
		query = `INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)`
		_, err = tx.Exec(query, pr.PullRequestID, reviewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetPR(prID string) (*models.PullRequest, error) {
	query := `SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at 
	          FROM pull_requests WHERE pull_request_id = $1`
	var pr models.PullRequest
	var mergedAt sql.NullTime
	err := r.db.QueryRow(query, prID).Scan(
		&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, 
		&pr.CreatedAt, &mergedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	reviewers, err := r.GetPRReviewers(prID)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (r *Repository) GetPRReviewers(prID string) ([]string, error) {
	query := `SELECT user_id FROM pr_reviewers WHERE pr_id = $1`
	rows, err := r.db.Query(query, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, reviewerID)
	}
	return reviewers, nil
}

func (r *Repository) UpdatePRStatus(prID, status string) error {
	query := `UPDATE pull_requests SET status = $1, merged_at = $2 WHERE pull_request_id = $3`
	
	var mergedAt interface{}
	if status == "MERGED" {
		mergedAt = time.Now()
	} else {
		mergedAt = nil
	}

	_, err := r.db.Exec(query, status, mergedAt, prID)
	return err
}

func (r *Repository) PRExists(prID string) (bool, error) {
	query := `SELECT 1 FROM pull_requests WHERE pull_request_id = $1`
	var exists int
	err := r.db.QueryRow(query, prID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists == 1, err
}

func (r *Repository) UpdatePRReviewers(prID string, reviewers []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM pr_reviewers WHERE pr_id = $1", prID)
	if err != nil {
		return err
	}

	for _, reviewerID := range reviewers {
		_, err = tx.Exec("INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)", prID, reviewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetPRsByReviewer(userID string) ([]models.PullRequestShort, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN pr_reviewers prr ON pr.pull_request_id = prr.pr_id
		WHERE prr.user_id = $1
		ORDER BY pr.created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
		
	var prs []models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	return prs, nil
}