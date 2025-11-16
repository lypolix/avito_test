package repository

import (
	"database/sql"

	"github.com/lypolix/avito_test/internal/models"
)

func (r *Repository) GetUserAssignmentStats() ([]models.UserStat, error) {
	query := `
		SELECT 
			u.user_id,
			u.username,
			u.team_name,
			u.is_active,
			COUNT(prr.pr_id) as assignments_count
		FROM users u
		LEFT JOIN pr_reviewers prr ON u.user_id = prr.user_id
		LEFT JOIN pull_requests pr ON prr.pr_id = pr.pull_request_id AND pr.status = 'OPEN'
		GROUP BY u.user_id, u.username, u.team_name, u.is_active
		ORDER BY assignments_count DESC, u.user_id
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.UserStat
	for rows.Next() {
		var stat models.UserStat
		err := rows.Scan(&stat.UserID, &stat.Username, &stat.TeamName, &stat.IsActive, &stat.AssignmentsCount)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func (r *Repository) GetPRAssignmentStats() ([]models.PRStat, error) {
	query := `
		SELECT 
			pr.pull_request_id,
			pr.pull_request_name,
			pr.author_id,
			pr.status,
			COUNT(prr.user_id) as reviewers_count,
			u.team_name
		FROM pull_requests pr
		LEFT JOIN pr_reviewers prr ON pr.pull_request_id = prr.pr_id
		LEFT JOIN users u ON pr.author_id = u.user_id
		GROUP BY pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, u.team_name
		ORDER BY reviewers_count DESC, pr.pull_request_id
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.PRStat
	for rows.Next() {
		var stat models.PRStat
		var teamName sql.NullString
		err := rows.Scan(&stat.PullRequestID, &stat.PullRequestName, &stat.AuthorID, &stat.Status, &stat.ReviewersCount, &teamName)
		if err != nil {
			return nil, err
		}
		if teamName.Valid {
			stat.TeamName = teamName.String
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func (r *Repository) GetStatsSummary() (models.StatsSummary, error) {
	var summary models.StatsSummary

	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&summary.TotalUsers)
	if err != nil {
		return summary, err
	}

	err = r.db.QueryRow("SELECT COUNT(*) FROM pull_requests").Scan(&summary.TotalPRs)
	if err != nil {
		return summary, err
	}

	err = r.db.QueryRow("SELECT COUNT(*) FROM pr_reviewers").Scan(&summary.TotalAssignments)
	if err != nil {
		return summary, err
	}

	if summary.TotalPRs > 0 {
		summary.AvgReviewersPerPR = float64(summary.TotalAssignments) / float64(summary.TotalPRs)
	}

	query := `
		SELECT u.user_id 
		FROM users u 
		JOIN pr_reviewers prr ON u.user_id = prr.user_id 
		GROUP BY u.user_id 
		ORDER BY COUNT(prr.pr_id) DESC 
		LIMIT 1
	`
	var mostActiveUser sql.NullString
	err = r.db.QueryRow(query).Scan(&mostActiveUser)
	if err != nil && err != sql.ErrNoRows {
		return summary, err
	}
	if mostActiveUser.Valid {
		summary.MostActiveUser = mostActiveUser.String
	}

	query = `
		SELECT pr_id 
		FROM pr_reviewers 
		GROUP BY pr_id 
		ORDER BY COUNT(user_id) DESC 
		LIMIT 1
	`
	var mostReviewedPR sql.NullString
	err = r.db.QueryRow(query).Scan(&mostReviewedPR)
	if err != nil && err != sql.ErrNoRows {
		return summary, err
	}
	if mostReviewedPR.Valid {
		summary.MostReviewedPR = mostReviewedPR.String
	}

	return summary, nil
}
