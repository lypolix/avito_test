package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/lypolix/avito_test/internal/config"
	"github.com/lypolix/avito_test/internal/database"
	"github.com/lypolix/avito_test/internal/handlers"
	"github.com/lypolix/avito_test/internal/repository"
	"github.com/lypolix/avito_test/internal/server"
	"github.com/lypolix/avito_test/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.ConnectWithRetry(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to database")

	repo := repository.NewRepository(db)
	service := services.NewService(repo)
	handler := handlers.NewHandler(service)

	server := server.New(cfg.Server)
	server.SetupRoutes(func(router *gin.Engine) {
		handler.SetupRoutesWithRouter(router)
	})

	startServerWithShutdown(server, cfg)
}

func startServerWithShutdown(server *server.Server, cfg *config.Config) {
	go func() {
		log.Printf("Starting server on :%s", cfg.Server.Port)
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
