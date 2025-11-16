package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/lypolix/avito_test/internal/models"
	"github.com/lypolix/avito_test/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FullWorkflowTestSuite struct {
	suite.Suite
	suite    *testutils.TestSuite
	testData testutils.TestData
}

func TestFullWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(FullWorkflowTestSuite))
}

func (ts *FullWorkflowTestSuite) SetupTest() {
	ts.suite = testutils.SetupTestSuite(ts.T())
	ts.testData = testutils.GetTestData()
}

func (ts *FullWorkflowTestSuite) TearDownTest() {
	ts.suite.TearDown(ts.T())
}

func (ts *FullWorkflowTestSuite) TestFullPRWorkflow() {
	teamReq := models.CreateTeamRequest{
		TeamName: ts.testData.Team1,
		Members: []models.TeamMember{
			{UserID: ts.testData.User1, Username: "Product Owner", IsActive: true},
			{UserID: ts.testData.User2, Username: "Developer 1", IsActive: true},
			{UserID: ts.testData.User3, Username: "Developer 2", IsActive: true},
			{UserID: ts.testData.User4, Username: "QA Engineer", IsActive: true},
		},
	}

	body, _ := json.Marshal(teamReq)
	resp, err := ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/team/add", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusCreated, resp.StatusCode)
	defer resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   ts.testData.PR1,
		PullRequestName: "Implement new feature",
		AuthorID:        ts.testData.User2,
	}

	body, _ = json.Marshal(prReq)
	resp, err = ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusCreated, resp.StatusCode)

	var prResponse models.PRResponse
	err = json.NewDecoder(resp.Body).Decode(&prResponse)
	resp.Body.Close()
	assert.NoError(ts.T(), err)

	assert.Len(ts.T(), prResponse.PR.AssignedReviewers, 2)
	assert.Equal(ts.T(), "OPEN", prResponse.PR.Status)

	deactivateReq := models.SetUserActiveRequest{
		UserID:   prResponse.PR.AssignedReviewers[0],
		IsActive: false,
	}

	body, _ = json.Marshal(deactivateReq)
	resp, err = ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/users/setIsActive", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	reassignReq := models.ReassignReviewerRequest{
		PullRequestID: ts.testData.PR1,
		OldUserID:     prResponse.PR.AssignedReviewers[0],
	}

	body, _ = json.Marshal(reassignReq)
	resp, err = ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusOK, resp.StatusCode)

	var reassignResponse models.ReassignResponse
	err = json.NewDecoder(resp.Body).Decode(&reassignResponse)
	resp.Body.Close()
	assert.NoError(ts.T(), err)

	assert.NotContains(ts.T(), reassignResponse.PR.AssignedReviewers, prResponse.PR.AssignedReviewers[0])
	assert.NotEmpty(ts.T(), reassignResponse.ReplacedBy)

	mergeReq := models.MergePRRequest{
		PullRequestID: ts.testData.PR1,
	}

	body, _ = json.Marshal(mergeReq)
	resp, err = ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/pullRequest/merge", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp, err = ts.suite.HTTPClient.Get(ts.suite.BaseURL + "/stats")
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusOK, resp.StatusCode)

	var statsResponse models.StatsResponse
	err = json.NewDecoder(resp.Body).Decode(&statsResponse)
	resp.Body.Close()
	assert.NoError(ts.T(), err)

	assert.Greater(ts.T(), statsResponse.Summary.TotalUsers, 0)
	assert.Greater(ts.T(), statsResponse.Summary.TotalPRs, 0)
	assert.Greater(ts.T(), statsResponse.Summary.TotalAssignments, 0)
}

func (ts *FullWorkflowTestSuite) TestGetTeam() {
	teamReq := models.CreateTeamRequest{
		TeamName: ts.testData.Team1,
		Members: []models.TeamMember{
			{UserID: ts.testData.User1, Username: "Developer 1", IsActive: true},
			{UserID: ts.testData.User2, Username: "Developer 2", IsActive: true},
			{UserID: ts.testData.User3, Username: "QA Engineer", IsActive: false},
		},
	}

	body, _ := json.Marshal(teamReq)
	resp, err := ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/team/add", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp, err = ts.suite.HTTPClient.Get(ts.suite.BaseURL + "/team/get?team_name=" + ts.testData.Team1)
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	var teamResponse struct {
		TeamName string              `json:"team_name"`
		Members  []models.TeamMember `json:"members"`
	}
	err = json.NewDecoder(resp.Body).Decode(&teamResponse)
	assert.NoError(ts.T(), err)

	assert.Equal(ts.T(), ts.testData.Team1, teamResponse.TeamName)
	assert.Len(ts.T(), teamResponse.Members, 3)

	membersMap := make(map[string]models.TeamMember)
	for _, member := range teamResponse.Members {
		membersMap[member.UserID] = member
	}

	assert.Contains(ts.T(), membersMap, ts.testData.User1)
	assert.Equal(ts.T(), "Developer 1", membersMap[ts.testData.User1].Username)
	assert.True(ts.T(), membersMap[ts.testData.User1].IsActive)

	assert.Contains(ts.T(), membersMap, ts.testData.User2)
	assert.Equal(ts.T(), "Developer 2", membersMap[ts.testData.User2].Username)
	assert.True(ts.T(), membersMap[ts.testData.User2].IsActive)

	assert.Contains(ts.T(), membersMap, ts.testData.User3)
	assert.Equal(ts.T(), "QA Engineer", membersMap[ts.testData.User3].Username)
	assert.False(ts.T(), membersMap[ts.testData.User3].IsActive)
}

func (ts *FullWorkflowTestSuite) TestGetTeamNotFound() {
	resp, err := ts.suite.HTTPClient.Get(ts.suite.BaseURL + "/team/get?team_name=nonexistent-team")
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusNotFound, resp.StatusCode)
	defer resp.Body.Close()

	var errorResponse models.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), "NOT_FOUND", errorResponse.Error.Code)
}

func (ts *FullWorkflowTestSuite) TestGetTeamWithoutName() {
	resp, err := ts.suite.HTTPClient.Get(ts.suite.BaseURL + "/team/get")
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusBadRequest, resp.StatusCode)
	defer resp.Body.Close()
}

func (ts *FullWorkflowTestSuite) TestGetUserPRs() {
	teamReq := models.CreateTeamRequest{
		TeamName: ts.testData.Team1,
		Members: []models.TeamMember{
			{UserID: ts.testData.User1, Username: "Author", IsActive: true},
			{UserID: ts.testData.User2, Username: "Reviewer", IsActive: true},
			{UserID: ts.testData.User3, Username: "Reviewer 2", IsActive: true},
		},
	}

	body, _ := json.Marshal(teamReq)
	resp, err := ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/team/add", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   ts.testData.PR1,
		PullRequestName: "Test Feature",
		AuthorID:        ts.testData.User1,
	}

	body, _ = json.Marshal(prReq)
	resp, err = ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	resp.Body.Close()

	resp, err = ts.suite.HTTPClient.Get(ts.suite.BaseURL + "/users/getReview?user_id=" + ts.testData.User2)
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	var userPRsResponse models.UserPRsResponse
	err = json.NewDecoder(resp.Body).Decode(&userPRsResponse)
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), ts.testData.User2, userPRsResponse.UserID)
	assert.Len(ts.T(), userPRsResponse.PullRequests, 1)
	assert.Equal(ts.T(), ts.testData.PR1, userPRsResponse.PullRequests[0].PullRequestID)
}

func (ts *FullWorkflowTestSuite) TestHealthCheck() {
	resp, err := ts.suite.HTTPClient.Get(ts.suite.BaseURL + "/health")
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	var healthResponse map[string]string
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), "OK", healthResponse["status"])
}

func (ts *FullWorkflowTestSuite) TestErrorCases() {
	teamReq := models.CreateTeamRequest{
		TeamName: ts.testData.Team1,
		Members:  []models.TeamMember{},
	}

	body, _ := json.Marshal(teamReq)

	resp, err := ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/team/add", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp, err = ts.suite.HTTPClient.Post(ts.suite.BaseURL+"/team/add", "application/json", bytes.NewBuffer(body))
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), http.StatusBadRequest, resp.StatusCode)
	defer resp.Body.Close()

	var errorResponse models.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	assert.NoError(ts.T(), err)
	assert.Equal(ts.T(), "TEAM_EXISTS", errorResponse.Error.Code)
}

type TestData struct {
	Team1 string
	Team2 string
	User1 string
	User2 string
	User3 string
	User4 string
	PR1   string
	PR2   string
}

func GetTestData() TestData {
	return TestData{
		Team1: "team-alpha",
		Team2: "team-beta",
		User1: "user1",
		User2: "user2",
		User3: "user3",
		User4: "user4",
		PR1:   "pr-001",
		PR2:   "pr-002",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
