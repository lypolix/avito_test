package integration

import (
	"context"
	"testing"

	"github.com/lypolix/avito_test/internal/services"
	"github.com/lypolix/avito_test/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UserIntegrationTestSuite struct {
	suite.Suite
	suite    *testutils.IntegrationTestSuite
	testData testutils.TestData
	ctx      context.Context
}

func TestUserIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UserIntegrationTestSuite))
}

func (ts *UserIntegrationTestSuite) SetupTest() {
	ts.suite = testutils.SetupIntegrationTestSuite(ts.T())
	ts.testData = testutils.GetTestData()
	ts.ctx = context.Background()
}

func (ts *UserIntegrationTestSuite) TearDownTest() {
	ts.suite.TearDown(ts.T())
}

func (ts *UserIntegrationTestSuite) TestSetUserActive_Success() {
	ts.suite.CreateTestTeam(ts.T(), ts.testData.Team1)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User1, "Alice", ts.testData.Team1, true)

	updatedUser, err := ts.suite.Service.SetUserActive(ts.testData.User1, false)
	assert.NoError(ts.T(), err)
	assert.NotNil(ts.T(), updatedUser)
	assert.Equal(ts.T(), ts.testData.User1, updatedUser.UserID)
	assert.False(ts.T(), updatedUser.IsActive)

	userFromDB, err := ts.suite.Repo.GetUser(ts.testData.User1)
	assert.NoError(ts.T(), err)
	assert.False(ts.T(), userFromDB.IsActive)
}

func (ts *UserIntegrationTestSuite) TestSetUserActive_NotFound() {
	user, err := ts.suite.Service.SetUserActive("nonexistent-user", false)
	assert.Error(ts.T(), err)
	assert.Nil(ts.T(), user)

	if businessErr, ok := err.(*services.BusinessError); ok {
		assert.Equal(ts.T(), "NOT_FOUND", businessErr.Code)
	} else {
		ts.T().Errorf("Expected BusinessError, got %T", err)
	}
}

func (ts *UserIntegrationTestSuite) TestGetUserPRs_Success() {
	ts.suite.CreateTestTeam(ts.T(), ts.testData.Team1)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User1, "Author", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User2, "Reviewer", ts.testData.Team1, true)
	ts.suite.CreateTestPR(ts.T(), ts.testData.PR1, "Test PR", ts.testData.User1, []string{ts.testData.User2})

	userPRs, err := ts.suite.Service.GetUserPRs(ts.testData.User2)
	assert.NoError(ts.T(), err)
	assert.NotNil(ts.T(), userPRs)
	assert.Equal(ts.T(), ts.testData.User2, userPRs.UserID)
	assert.Len(ts.T(), userPRs.PullRequests, 1)
	assert.Equal(ts.T(), ts.testData.PR1, userPRs.PullRequests[0].PullRequestID)
}

func (ts *UserIntegrationTestSuite) TestBulkDeactivateUsers_Success() {
	ts.suite.CreateTestTeam(ts.T(), ts.testData.Team1)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User1, "User1", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User2, "User2", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User3, "User3", ts.testData.Team1, true)

	response, err := ts.suite.Service.BulkDeactivateUsers(ts.testData.Team1, []string{ts.testData.User1, ts.testData.User2})
	assert.NoError(ts.T(), err)
	assert.NotNil(ts.T(), response)
	assert.Equal(ts.T(), ts.testData.Team1, response.TeamName)
	assert.Len(ts.T(), response.DeactivatedUsers, 2)

	user1, _ := ts.suite.Repo.GetUser(ts.testData.User1)
	user2, _ := ts.suite.Repo.GetUser(ts.testData.User2)
	user3, _ := ts.suite.Repo.GetUser(ts.testData.User3)

	assert.False(ts.T(), user1.IsActive)
	assert.False(ts.T(), user2.IsActive)
	assert.True(ts.T(), user3.IsActive)
}
