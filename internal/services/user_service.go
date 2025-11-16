package services

import (
	"database/sql"

	"github.com/lypolix/avito_test/internal/models"
)

func (s *Service) SetUserActive(userID string, isActive bool) (*models.User, error) {
	user, err := s.repo.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, NewBusinessError(ErrorNotFound, "user not found")
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

func (s *Service) BulkDeactivateUsers(teamName string, userIDs []string) (*models.BulkDeactivateResponse, error) {
	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	response, err := s.processBulkDeactivationInTx(tx, teamName, userIDs)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *Service) processBulkDeactivationInTx(tx *sql.Tx, teamName string, userIDs []string) (*models.BulkDeactivateResponse, error) {
	teamExists, err := s.repo.TeamExists(teamName)
	if err != nil {
		return nil, err
	}
	if !teamExists {
		return nil, NewBusinessError(ErrorNotFound, "team not found")
	}

	activeTeamUsers, err := s.repo.GetActiveUsersByTeamInTx(tx, teamName)
	if err != nil {
		return nil, err
	}

	activeUsersMap := make(map[string]bool)
	var activeUserIDs []string
	for _, user := range activeTeamUsers {
		activeUsersMap[user.UserID] = true
		activeUserIDs = append(activeUserIDs, user.UserID)
	}

	for _, userID := range userIDs {
		if !activeUsersMap[userID] {
			return nil, NewBusinessError(ErrorNotFound, "user not found or not active: "+userID)
		}
	}

	response := &models.BulkDeactivateResponse{
		TeamName:            teamName,
		DeactivatedUsers:    userIDs,
		ReassignedPRs:       []models.ReassignedPRDetail{},
		FailedReassignments: []models.FailedReassignment{},
	}

	openPRs, err := s.repo.GetAllOpenPRs()
	if err != nil {
		return nil, err
	}

	deactivatingMap := make(map[string]bool)
	for _, userID := range userIDs {
		deactivatingMap[userID] = true
	}

	var prsToProcess []*models.PullRequest
	for _, pr := range openPRs {
		for _, reviewer := range pr.AssignedReviewers {
			if deactivatingMap[reviewer] {
				prsToProcess = append(prsToProcess, pr)
				break
			}
		}
	}

	for _, pr := range prsToProcess {
		reassignedPR, failedReplacements, err := s.reassignDeactivatedReviewersInTx(tx, pr, userIDs, activeUserIDs)
		if err != nil {
			return nil, err
		}

		if len(reassignedPR.Replacements) > 0 || len(failedReplacements) > 0 {
			response.ReassignedPRs = append(response.ReassignedPRs, reassignedPR)
		}

		response.FailedReassignments = append(response.FailedReassignments, failedReplacements...)
	}

	for _, userID := range userIDs {
		if err := s.repo.UpdateUserActiveInTx(tx, userID, false); err != nil {
			return nil, err
		}
	}

	return response, nil
}

func (s *Service) reassignDeactivatedReviewersInTx(tx *sql.Tx, pr *models.PullRequest, deactivatingUserIDs []string, activeUserIDs []string) (models.ReassignedPRDetail, []models.FailedReassignment, error) {
	reassignedPR := models.ReassignedPRDetail{
		PullRequestID: pr.PullRequestID,
		Replacements:  []models.UserReplacement{},
	}

	var failedReplacements []models.FailedReassignment

	newReviewers := make([]string, len(pr.AssignedReviewers))
	copy(newReviewers, pr.AssignedReviewers)

	currentReviewersMap := make(map[string]bool)
	for _, reviewer := range newReviewers {
		currentReviewersMap[reviewer] = true
	}

	deactivatingMap := make(map[string]bool)
	for _, userID := range deactivatingUserIDs {
		deactivatingMap[userID] = true
	}

	availableCandidates := s.findAvailableCandidates(activeUserIDs, pr.AuthorID, currentReviewersMap, deactivatingMap)

	for _, oldUserID := range deactivatingUserIDs {
		if !currentReviewersMap[oldUserID] {
			continue
		}

		if len(availableCandidates) == 0 {
			newReviewers = s.removeFromSlice(newReviewers, oldUserID)
			failedReplacements = append(failedReplacements, models.FailedReassignment{
				PullRequestID: pr.PullRequestID,
				OldUserID:     oldUserID,
				Reason:        "no active replacement candidate available",
			})
			continue
		}

		newUserID := availableCandidates[0]
		availableCandidates = availableCandidates[1:]

		newReviewers = s.replaceInSlice(newReviewers, oldUserID, newUserID)

		availableCandidates = s.removeFromSlice(availableCandidates, newUserID)

		reassignedPR.Replacements = append(reassignedPR.Replacements, models.UserReplacement{
			OldUserID: oldUserID,
			NewUserID: newUserID,
		})
	}

	if len(reassignedPR.Replacements) > 0 || len(failedReplacements) > 0 {
		if err := s.repo.UpdatePRReviewersInTx(tx, pr.PullRequestID, newReviewers); err != nil {
			return reassignedPR, failedReplacements, err
		}
	}

	return reassignedPR, failedReplacements, nil
}

func (s *Service) findAvailableCandidates(activeUserIDs []string, authorID string, currentReviewersMap map[string]bool, deactivatingMap map[string]bool) []string {
	var candidates []string

	for _, userID := range activeUserIDs {
		if userID != authorID &&
			!currentReviewersMap[userID] &&
			!deactivatingMap[userID] {
			candidates = append(candidates, userID)
		}
	}

	return candidates
}

func (s *Service) removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

func (s *Service) replaceInSlice(slice []string, old, new string) []string {
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
