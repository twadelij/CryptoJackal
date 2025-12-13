package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/twadelij/cryptojackal/internal/api/handlers"
	"github.com/twadelij/cryptojackal/internal/api/middleware"
	"github.com/twadelij/cryptojackal/internal/config"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/paper"
	"github.com/twadelij/cryptojackal/internal/trading"
	"go.uber.org/zap"
)

// Server is the HTTP API server
type Server struct {
	config  *config.Config
	router  *gin.Engine
	server  *http.Server
	handler *handlers.Handler
	logger  *zap.Logger
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, engine *trading.Engine, disc *discovery.Service, paperSvc *paper.Service, logger *zap.Logger) *Server {
	// Set gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	
	// Middleware
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.Logger(logger))
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	handler := handlers.NewHandler(engine, disc, paperSvc, logger)

	// Routes
	api := router.Group("/api")
	{
		// Health
		api.GET("/health", handler.Health)

		// Bot control
		api.GET("/bot/status", handler.GetStatus)
		api.POST("/bot/start", handler.StartBot)
		api.POST("/bot/stop", handler.StopBot)

		// Trading
		api.GET("/trading/opportunities", handler.GetOpportunities)
		api.POST("/trading/execute", handler.ExecuteTrade)
		api.GET("/trading/history", handler.GetTradingHistory)

		// Discovery
		api.GET("/discovery/trending", handler.GetTrendingTokens)
		api.GET("/discovery/new", handler.GetNewTokens)
		api.GET("/discovery/analyze/:address", handler.AnalyzeToken)

		// Paper trading
		api.GET("/paper/balance", handler.GetPaperBalance)
		api.POST("/paper/reset", handler.ResetPaperBalance)

		// Metrics
		api.GET("/metrics", handler.GetMetrics)
	}

	// Serve static files for frontend
	router.Static("/assets", "./web/dist/assets")
	router.StaticFile("/", "./web/dist/index.html")
	router.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	return &Server{
		config:  cfg,
		router:  router,
		handler: handler,
		logger:  logger,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.config.ServerHost, s.config.ServerPort)
	
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("starting HTTP server", zap.String("addr", addr))
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")
	return s.server.Shutdown(ctx)
}
