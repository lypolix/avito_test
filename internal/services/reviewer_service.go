package services

import (
	"math/rand"

	"github.com/lypolix/avito_test/internal/models"
)

func (s *Service) autoAssignReviewers(author *models.User) ([]string, error) {
	teamUsers, err := s.repo.GetActiveUsersByTeam(author.TeamName)
	if err != nil {
		return nil, err
	}

	var candidates []string
	for _, user := range teamUsers {
		if user.UserID != author.UserID {
			candidates = append(candidates, user.UserID)
		}
	}
	shuffle(candidates)

	maxReviewers := 2
	if len(candidates) < maxReviewers {
		maxReviewers = len(candidates)
	}

	return candidates[:maxReviewers], nil
}

func (s *Service) findReplacementReviewer(oldUserID string, currentReviewers []string, authorID string) (string, error) {
	oldReviewer, err := s.repo.GetUser(oldUserID)
	if err != nil {
		return "", err
	}
	if oldReviewer == nil {
		return "", NewBusinessError(ErrorNotFound, "old reviewer not found")
	}

	teamUsers, err := s.repo.GetActiveUsersByTeam(oldReviewer.TeamName)
	if err != nil {
		return "", err
	}

	var candidates []string
	for _, user := range teamUsers {
		if user.UserID != oldUserID && 
		   user.UserID != authorID && 
		   !contains(currentReviewers, user.UserID) {
			candidates = append(candidates, user.UserID)
		}
	}

	if len(candidates) == 0 {
		return "", NewBusinessError(ErrorNoCandidate, "no active replacement candidate in team")
	}

	return candidates[rand.Intn(len(candidates))], nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func replaceInSlice(slice []string, old, new string) []string {
	result := make([]string, len(slice))
	for i, item := range slice {
		if item == old {
			result[i] = new
		} else {
			result[i] = item
		}
	}
	return result
}

func shuffle(slice []string) {
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}