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

type PRIntegrationTestSuite struct {
	suite.Suite
	suite    *testutils.IntegrationTestSuite
	testData testutils.TestData
	ctx      context.Context
}

func TestPRIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PRIntegrationTestSuite))
}

func (ts *PRIntegrationTestSuite) SetupTest() {
	ts.suite = testutils.SetupIntegrationTestSuite(ts.T())
	ts.testData = testutils.GetTestData()
	ts.ctx = context.Background()
}

func (ts *PRIntegrationTestSuite) TearDownTest() {
	ts.suite.TearDown(ts.T())
}

func (ts *PRIntegrationTestSuite) TestCreatePR_Success() {
	ts.suite.CreateTestTeam(ts.T(), ts.testData.Team1)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User1, "Author", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User2, "Reviewer1", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User3, "Reviewer2", ts.testData.Team1, true)

	prRequest := &models.CreatePRRequest{
		PullRequestID:   ts.testData.PR1,
		PullRequestName: "Test Feature",
		AuthorID:        ts.testData.User1,
	}

	pr, err := ts.suite.Service.CreatePR(prRequest)
	assert.NoError(ts.T(), err)
	assert.NotNil(ts.T(), pr)
	assert.Equal(ts.T(), ts.testData.PR1, pr.PullRequestID)
	assert.Equal(ts.T(), "OPEN", pr.Status)
	assert.Len(ts.T(), pr.AssignedReviewers, 2)
}

func (ts *PRIntegrationTestSuite) TestCreatePR_AuthorNotFound() {
	prRequest := &models.CreatePRRequest{
		PullRequestID:   ts.testData.PR1,
		PullRequestName: "Test Feature",
		AuthorID:        "nonexistent-author",
	}

	pr, err := ts.suite.Service.CreatePR(prRequest)
	assert.Error(ts.T(), err)
	assert.Nil(ts.T(), pr)

	if businessErr, ok := err.(*services.BusinessError); ok {
		assert.Equal(ts.T(), "NOT_FOUND", businessErr.Code)
	} else {
		ts.T().Errorf("Expected BusinessError, got %T", err)
	}
}

func (ts *PRIntegrationTestSuite) TestMergePR_Success() {
	ts.suite.CreateTestTeam(ts.T(), ts.testData.Team1)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User1, "Author", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User2, "Reviewer", ts.testData.Team1, true)
	ts.suite.CreateTestPR(ts.T(), ts.testData.PR1, "Test PR", ts.testData.User1, []string{ts.testData.User2})

	mergedPR, err := ts.suite.Service.MergePR(ts.testData.PR1)
	assert.NoError(ts.T(), err)
	assert.NotNil(ts.T(), mergedPR)
	assert.Equal(ts.T(), "MERGED", mergedPR.Status)

	mergedPR2, err := ts.suite.Service.MergePR(ts.testData.PR1)
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), "MERGED", mergedPR2.Status)
}

func (ts *PRIntegrationTestSuite) TestReassignReviewer_Success() {
	ts.suite.CreateTestTeam(ts.T(), ts.testData.Team1)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User1, "Author", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User2, "Reviewer1", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User3, "Reviewer2", ts.testData.Team1, true)
	ts.suite.CreateTestPR(ts.T(), ts.testData.PR1, "Test PR", ts.testData.User1, []string{ts.testData.User2})

	reassignResponse, err := ts.suite.Service.ReassignReviewer(ts.testData.PR1, ts.testData.User2)
	assert.NoError(ts.T(), err)
	assert.NotNil(ts.T(), reassignResponse)
	assert.NotContains(ts.T(), reassignResponse.PR.AssignedReviewers, ts.testData.User2)
	assert.NotEmpty(ts.T(), reassignResponse.ReplacedBy)
}

func (ts *PRIntegrationTestSuite) TestGetStats_Success() {
	ts.suite.CreateTestTeam(ts.T(), ts.testData.Team1)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User1, "User1", ts.testData.Team1, true)
	ts.suite.CreateTestUser(ts.T(), ts.testData.User2, "User2", ts.testData.Team1, true)
	ts.suite.CreateTestPR(ts.T(), ts.testData.PR1, "PR1", ts.testData.User1, []string{ts.testData.User2})

	stats, err := ts.suite.Service.GetStats()
	assert.NoError(ts.T(), err)
	assert.NotNil(ts.T(), stats)
	assert.Greater(ts.T(), stats.Summary.TotalUsers, 0)
	assert.Greater(ts.T(), stats.Summary.TotalPRs, 0)
	assert.Greater(ts.T(), stats.Summary.TotalAssignments, 0)
}
