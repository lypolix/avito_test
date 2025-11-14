package services

import (
	"github.com/lypolix/avito_test/internal/models"
)

func (s *Service) SetUserActive(userID string, isActive bool) (*models.User, error) {
	user, err := s.repo.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, NewBusinessError(ErrorNotFound, "resource not found")
	}

	if err := s.repo.UpdateUserActive(userID, isActive); err != nil {
		return nil, err
	}

	updatedUser, err := s.repo.GetUser(userID)
	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}

func (s *Service) GetUserPRs(userID string) (*models.UserPRsResponse, error) {
	user, err := s.repo.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, NewBusinessError(ErrorNotFound, "user not found")
	}

	prs, err := s.repo.GetPRsByReviewer(userID)
	if err != nil {
		return nil, err
	}

	return &models.UserPRsResponse{
		UserID:       userID,
		PullRequests: prs,
	}, nil
}