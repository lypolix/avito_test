package services

import (
    "github.com/lypolix/avito_test/internal/models"
)

func (s *Service) GetStats() (*models.StatsResponse, error) {
	userStats, err := s.repo.GetUserAssignmentStats()
	if err != nil {
		return nil, err
	}

	prStats, err := s.repo.GetPRAssignmentStats()
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.GetStatsSummary()
	if err != nil {
		return nil, err
	}

	return &models.StatsResponse{
		UserStats: userStats,
		PRStats:   prStats,
		Summary:   summary,
	}, nil
}
