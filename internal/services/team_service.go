package services

import (
	"github.com/lypolix/avito_test/internal/models"
)

func (s *Service) CreateTeam(team *models.Team) error {
    exists, err := s.repo.TeamExists(team.TeamName)
    if err != nil {
        return err
    }
    if exists {
        return NewBusinessError(ErrorTeamExists, "team already exists")
    }

    for _, member := range team.Members {
        inOtherTeam, err := s.repo.IsUserInOtherTeam(member.UserID, team.TeamName)
        if err != nil {
            return err
        }
        if inOtherTeam {
            return NewBusinessError(ErrorUserInOtherTeam, "user " + member.UserID + " already belongs to another team")
        }
    }
    if err := s.repo.CreateTeam(team.TeamName); err != nil {
        return err
    }

    for _, member := range team.Members {
        user := &models.User{
            UserID:   member.UserID,
            Username: member.Username,
            TeamName: team.TeamName,
            IsActive: member.IsActive,
        }
        if err := s.repo.CreateUser(user); err != nil {
            return err
        }
    }

    return nil
}

func (s *Service) GetTeam(teamName string) (*models.Team, error) {
	team, err := s.repo.GetTeam(teamName)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, NewBusinessError(ErrorNotFound, "resource not found")
	}
	return team, nil
}
