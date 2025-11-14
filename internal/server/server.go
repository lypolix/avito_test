package server

import (
	"context"
	"net/http"

	"github.com/lypolix/avito_test/internal/config"
	"github.com/gin-gonic/gin"
)

type Server struct {
	httpServer *http.Server
	router     *gin.Engine
}

func New(serverConfig config.ServerConfig) *Server {
	router := gin.Default()
	
	return &Server{
		httpServer: &http.Server{
			Addr:         ":" + serverConfig.Port,
			Handler:      router,
			ReadTimeout:  serverConfig.ReadTimeout,
			WriteTimeout: serverConfig.WriteTimeout,
			IdleTimeout:  serverConfig.IdleTimeout,
		},
		router: router,
	}
}

func (s *Server) SetupRoutes(setupFunc func(*gin.Engine)) {
	setupFunc(s.router)
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}