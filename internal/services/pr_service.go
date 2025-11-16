package services

import (
	"github.com/lypolix/avito_test/internal/models"
)

func (s *Service) CreatePR(prRequest *models.CreatePRRequest) (*models.PullRequest, error) {
	exists, err := s.repo.PRExists(prRequest.PullRequestID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, NewBusinessError(ErrorPRExists, "PR id already exists")
	}

	author, err := s.repo.GetUser(prRequest.AuthorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, NewBusinessError(ErrorNotFound, "resource not found")
	}

	reviewers, err := s.autoAssignReviewers(author)
	if err != nil {
		return nil, err
	}

	pr := &models.PullRequest{
		PullRequestID:     prRequest.PullRequestID,
		PullRequestName:   prRequest.PullRequestName,
		AuthorID:          prRequest.AuthorID,
		Status:            "OPEN",
		AssignedReviewers: reviewers,
	}

	if err := s.repo.CreatePR(pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) MergePR(prID string) (*models.PullRequest, error) {
	pr, err := s.repo.GetPR(prID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, NewBusinessError(ErrorNotFound, "resource not found")
	}

	if pr.Status == "MERGED" {
		return pr, nil
	}

	if err := s.repo.UpdatePRStatus(prID, "MERGED"); err != nil {
		return nil, err
	}

	mergedPR, err := s.repo.GetPR(prID)
	if err != nil {
		return nil, err
	}

	return mergedPR, nil
}

func (s *Service) ReassignReviewer(prID, oldUserID string) (*models.ReassignResponse, error) {
	pr, err := s.repo.GetPR(prID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, NewBusinessError(ErrorNotFound, "resource not found")
	}

	if pr.Status == "MERGED" {
		return nil, NewBusinessError(ErrorPRMerged, "cannot reassign on merged PR")
	}

	if !contains(pr.AssignedReviewers, oldUserID) {
		return nil, NewBusinessError(ErrorNotAssigned, "reviewer is not assigned to this PR")
	}

	newReviewerID, err := s.findReplacementReviewer(oldUserID, pr.AssignedReviewers, pr.AuthorID)
	if err != nil {
		return nil, err
	}

	newReviewers := replaceInSlice(pr.AssignedReviewers, oldUserID, newReviewerID)
	if err := s.repo.UpdatePRReviewers(prID, newReviewers); err != nil {
		return nil, err
	}

	updatedPR, err := s.repo.GetPR(prID)
	if err != nil {
		return nil, err
	}

	return &models.ReassignResponse{
		PR:         updatedPR,
		ReplacedBy: newReviewerID,
	}, nil
}
