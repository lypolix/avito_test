package integration

import (
	"context"
	"testing"

	"github.com/lypolix/avito_test/internal/models"
	"github.com/lypolix/avito_test/internal/services"
	"github.com/lypolix/avito_test/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TeamIntegrationTestSuite struct {
	suite.Suite
	suite    *testutils.IntegrationTestSuite
	testData testutils.TestData
	ctx      context.Context
}

func TestTeamIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TeamIntegrationTestSuite))
}

func (ts *TeamIntegrationTestSuite) SetupTest() {
	ts.suite = testutils.SetupIntegrationTestSuite(ts.T())
	ts.testData = testutils.GetTestData()
	ts.ctx = context.Background()
}

func (ts *TeamIntegrationTestSuite) TearDownTest() {
	ts.suite.TearDown(ts.T())
}

func (ts *TeamIntegrationTestSuite) TestCreateTeam_Success() {
	team := models.Team{
		TeamName: ts.testData.Team1,
		Members: []models.TeamMember{
			{UserID: ts.testData.User1, Username: "Alice", IsActive: true},
			{UserID: ts.testData.User2, Username: "Bob", IsActive: true},
		},
	}

	err := ts.suite.Service.CreateTeam(&team)
	assert.NoError(ts.T(), err)

	createdTeam, err := ts.suite.Repo.GetTeam(ts.testData.Team1)
	assert.NoError(ts.T(), err)
	assert.NotNil(ts.T(), createdTeam)
	assert.Equal(ts.T(), ts.testData.Team1, createdTeam.TeamName)
	assert.Len(ts.T(), createdTeam.Members, 2)

	user1, err := ts.suite.Repo.GetUser(ts.testData.User1)
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), "Alice", user1.Username)
	assert.Equal(ts.T(), ts.testData.Team1, user1.TeamName)
	assert.True(ts.T(), user1.IsActive)
}

func (ts *TeamIntegrationTestSuite) TestCreateTeam_AlreadyExists() {
	team := models.Team{
		TeamName: ts.testData.Team1,
		Members: []models.TeamMember{
			{UserID: ts.testData.User1, Username: "Alice", IsActive: true},
		},
	}

	err := ts.suite.Service.CreateTeam(&team)
	assert.NoError(ts.T(), err)

	err = ts.suite.Service.CreateTeam(&team)
	assert.Error(ts.T(), err)

	if businessErr, ok := err.(*services.BusinessError); ok {
		assert.Equal(ts.T(), "TEAM_EXISTS", businessErr.Code)
	} else {
		ts.T().Errorf("Expected BusinessError, got %T", err)
	}
}

func (ts *TeamIntegrationTestSuite) TestGetTeam_NotFound() {
	team, err := ts.suite.Service.GetTeam("nonexistent-team")
	assert.Error(ts.T(), err)
	assert.Nil(ts.T(), team)

	if businessErr, ok := err.(*services.BusinessError); ok {
		assert.Equal(ts.T(), "NOT_FOUND", businessErr.Code)
	} else {
		ts.T().Errorf("Expected BusinessError, got %T", err)
	}
}

func (ts *TeamIntegrationTestSuite) TestGetTeam_Success() {
	ts.suite.CreateTestTeam(ts.T(), ts.testData.Team1)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User1, "Alice", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User2, "Bob", ts.testData.Team1, false)

	team, err := ts.suite.Service.GetTeam(ts.testData.Team1)
	assert.NoError(ts.T(), err)
	assert.NotNil(ts.T(), team)
	assert.Equal(ts.T(), ts.testData.Team1, team.TeamName)
	assert.Len(ts.T(), team.Members, 2)

	memberMap := make(map[string]models.TeamMember)
	for _, member := range team.Members {
		memberMap[member.UserID] = member
	}

	assert.Contains(ts.T(), memberMap, ts.testData.User1)
	assert.Equal(ts.T(), "Alice", memberMap[ts.testData.User1].Username)
	assert.True(ts.T(), memberMap[ts.testData.User1].IsActive)

	assert.Contains(ts.T(), memberMap, ts.testData.User2)
	assert.Equal(ts.T(), "Bob", memberMap[ts.testData.User2].Username)
	assert.False(ts.T(), memberMap[ts.testData.User2].IsActive)
}
