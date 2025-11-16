package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/lypolix/avito_test/internal/config"
	"github.com/lypolix/avito_test/internal/database"
	"github.com/lypolix/avito_test/internal/handlers"
	"github.com/lypolix/avito_test/internal/repository"
	"github.com/lypolix/avito_test/internal/server"
	"github.com/lypolix/avito_test/internal/services"

	_ "github.com/lib/pq"
)

type TestSuite struct {
	DB         *sql.DB
	Repo       *repository.Repository
	Service    *services.Service
	Handler    *handlers.Handler
	Server     *server.Server
	Config     *config.Config
	HTTPClient *http.Client
	BaseURL    string
}

type IntegrationTestSuite struct {
	DB      *sql.DB
	Repo    *repository.Repository
	Service *services.Service
	Config  *config.Config
	Ctx     context.Context
}

func SetupTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         getEnv("TEST_SERVER_PORT", "8090"),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Database: config.DatabaseConfig{
			Host:            getEnv("TEST_DB_HOST", "localhost"),
			Port:            getEnv("TEST_DB_PORT", "55432"),
			User:            getEnv("TEST_DB_USER", "test_user"),
			Password:        getEnv("TEST_DB_PASSWORD", "test_pass"),
			Name:            getEnv("TEST_DB_NAME", "appdb_test"),
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			MaxRetries:      3,
			RetryInterval:   1 * time.Second,
		},
		App: config.AppConfig{
			Env:             "test",
			ShutdownTimeout: 5 * time.Second,
		},
	}

	db, err := database.ConnectWithRetry(cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	repo := repository.NewRepository(db)
	service := services.NewService(repo)
	handler := handlers.NewHandler(service)

	srv := server.New(cfg.Server)
	srv.SetupRoutes(handler.SetupRoutesWithRouter)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("Test server error: %v", err)
		}
	}()
	time.Sleep(500 * time.Millisecond)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	baseURL := getEnv("TEST_SERVER_URL", "http://localhost:8090")

	suite := &TestSuite{
		DB:         db,
		Repo:       repo,
		Service:    service,
		Handler:    handler,
		Server:     srv,
		Config:     cfg,
		HTTPClient: client,
		BaseURL:    baseURL,
	}

	suite.CleanDatabase(t)
	return suite
}

func SetupIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	t.Helper()

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:            getEnv("TEST_DB_HOST", "localhost"),
			Port:            getEnv("TEST_DB_PORT", "55432"),
			User:            getEnv("TEST_DB_USER", "test_user"),
			Password:        getEnv("TEST_DB_PASSWORD", "test_pass"),
			Name:            getEnv("TEST_DB_NAME", "appdb_test"),
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			MaxRetries:      3,
			RetryInterval:   1 * time.Second,
		},
	}

	db, err := database.ConnectWithRetry(cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	repo := repository.NewRepository(db)
	service := services.NewService(repo)

	suite := &IntegrationTestSuite{
		DB:      db,
		Repo:    repo,
		Service: service,
		Config:  cfg,
		Ctx:     context.Background(),
	}

	suite.CleanDatabase(t)
	return suite
}

func (ts *TestSuite) CleanDatabase(t *testing.T) {
	cleanDatabase(t, ts.DB)
}

func (its *IntegrationTestSuite) CleanDatabase(t *testing.T) {
	cleanDatabase(t, its.DB)
}

func cleanDatabase(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []string{"pr_reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Fatalf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (ts *TestSuite) TearDown(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), ts.Config.App.ShutdownTimeout)
	defer cancel()

	if err := ts.Server.Shutdown(ctx); err != nil {
		t.Logf("Error shutting down test server: %v", err)
	}

	if err := ts.DB.Close(); err != nil {
		t.Logf("Error closing database connection: %v", err)
	}
}

func (its *IntegrationTestSuite) TearDown(t *testing.T) {
	t.Helper()

	if err := its.DB.Close(); err != nil {
		t.Logf("Error closing database connection: %v", err)
	}
}

func (ts *TestSuite) CreateTestTeam(t *testing.T, teamName string) {
	createTestTeam(t, ts.DB, teamName)
}

func (its *IntegrationTestSuite) CreateTestTeam(t *testing.T, teamName string) {
	createTestTeam(t, its.DB, teamName)
}

func createTestTeam(t *testing.T, db *sql.DB, teamName string) {
	t.Helper()
	_, err := db.Exec("INSERT INTO teams (team_name) VALUES ($1)", teamName)
	if err != nil {
		t.Fatalf("Failed to create test team: %v", err)
	}
}

func (ts *TestSuite) CreateTestUser(t *testing.T, userID, username, teamName string, isActive bool) {
	createTestUser(t, ts.DB, userID, username, teamName, isActive)
}

func (its *IntegrationTestSuite) CreateTestUser(t *testing.T, userID, username, teamName string, isActive bool) {
	createTestUser(t, its.DB, userID, username, teamName, isActive)
}

func createTestUser(t *testing.T, db *sql.DB, userID, username, teamName string, isActive bool) {
	t.Helper()
	_, err := db.Exec(
		"INSERT INTO users (user_id, username, team_name, is_active) VALUES ($1, $2, $3, $4)",
		userID, username, teamName, isActive,
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
}

func (ts *TestSuite) CreateTestPR(t *testing.T, prID, prName, authorID string, reviewers []string) {
	createTestPR(t, ts.DB, prID, prName, authorID, reviewers)
}

func (its *IntegrationTestSuite) CreateTestPR(t *testing.T, prID, prName, authorID string, reviewers []string) {
	createTestPR(t, its.DB, prID, prName, authorID, reviewers)
}

func createTestPR(t *testing.T, db *sql.DB, prID, prName, authorID string, reviewers []string) {
	t.Helper()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES ($1, $2, $3, $4)",
		prID, prName, authorID, "OPEN",
	)
	if err != nil {
		t.Fatalf("Failed to create test PR: %v", err)
	}

	for _, reviewerID := range reviewers {
		_, err = tx.Exec(
			"INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)",
			prID, reviewerID,
		)
		if err != nil {
			t.Fatalf("Failed to add reviewer to PR: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
