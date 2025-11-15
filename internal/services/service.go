package services

import (

	"github.com/lypolix/avito_test/internal/repository"
)


const (
	ErrorTeamExists   = "TEAM_EXISTS"
	ErrorPRExists     = "PR_EXISTS"
	ErrorPRMerged     = "PR_MERGED"
	ErrorNotAssigned  = "NOT_ASSIGNED"
	ErrorNoCandidate  = "NO_CANDIDATE"
	ErrorNotFound     = "NOT_FOUND"
	ErrorInvalidTeam  = "INVALID_TEAM"
	ErrorUserInOtherTeam = "USER_IN_OTHER_TEAM"
)

type Service struct {
	repo *repository.Repository
}

func NewService(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}


type BusinessError struct {
	Code    string
	Message string
}

func (e *BusinessError) Error() string {
	return e.Message
}

func NewBusinessError(code, message string) error {
	return &BusinessError{Code: code, Message: message}
}