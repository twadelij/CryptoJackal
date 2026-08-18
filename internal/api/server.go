package api

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/twadelij/cryptojackal/internal/api/handlers"
	"github.com/twadelij/cryptojackal/internal/api/middleware"
	"github.com/twadelij/cryptojackal/internal/config"
	"github.com/twadelij/cryptojackal/internal/discovery"
	"github.com/twadelij/cryptojackal/internal/paper"
	"github.com/twadelij/cryptojackal/internal/storage"
	"github.com/twadelij/cryptojackal/internal/trading"
	"go.uber.org/zap"
)

//go:embed templates/index.html
var indexHTML embed.FS

// Server is the HTTP API server
type Server struct {
	config  *config.Config
	router  *gin.Engine
	server  *http.Server
	handler *handlers.Handler
	logger  *zap.Logger
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, engine *trading.Engine, disc *discovery.Service, paperSvc *paper.Service, store *storage.Storage, logger *zap.Logger) *Server {
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

	handler := handlers.NewHandler(cfg, engine, disc, paperSvc, store, logger)

	// Routes
	api := router.Group("/api")
	{
		// Public routes (no auth required)
		api.GET("/health", handler.Health)
		api.POST("/auth/login", handler.Login)

		// Protected routes (JWT required)
		protected := api.Group("/")
		protected.Use(middleware.JWTAuth(cfg.JWTSecret))
		{
			// Config
			protected.GET("/config", handler.GetConfig)
			protected.POST("/config", handler.UpdateConfig)

			// Bot control
			protected.GET("/bot/status", handler.GetStatus)
			protected.POST("/bot/start", handler.StartBot)
			protected.POST("/bot/stop", handler.StopBot)

			// Trading
			protected.GET("/trading/opportunities", handler.GetOpportunities)
			protected.POST("/trading/execute", handler.ExecuteTrade)
			protected.GET("/trading/history", handler.GetTradingHistory)

			// Discovery
			protected.GET("/discovery/trending", handler.GetTrendingTokens)
			protected.GET("/discovery/new", handler.GetNewTokens)
			protected.GET("/discovery/analyze/:address", handler.AnalyzeToken)

			// Paper trading
			protected.GET("/paper/balance", handler.GetPaperBalance)
			protected.POST("/paper/reset", handler.ResetPaperBalance)
			protected.POST("/paper/trade", handler.ExecutePaperTrade)
			protected.GET("/paper/history", handler.GetTradingHistory)
			protected.GET("/paper/export", handler.ExportPaperTrades)

			// Metrics
			protected.GET("/metrics", handler.GetMetrics)

			// External API health
			protected.GET("/health/external", handler.GetExternalHealth)

			// Positions
			protected.GET("/positions", handler.GetPositions)
			protected.POST("/positions/:id/close", handler.ClosePosition)

			// Strategies
			protected.GET("/strategies", handler.GetStrategies)

			// ML
			protected.GET("/ml/status", handler.GetMLStatus)

			// Datasource status
			protected.GET("/datasources/status", handler.GetDatasourceStatus)

			// Backtesting
			protected.POST("/backtest/run", handler.RunBacktest)
			protected.GET("/backtest/history", handler.GetBacktestHistory)
			protected.GET("/indicators/:pair", handler.GetIndicators)
		}
	}

	// Serve built React frontend if web/dist exists, otherwise fallback to embedded template
	if _, err := os.Stat("web/dist"); err == nil {
		router.Static("/assets", "web/dist/assets")
		router.StaticFile("/favicon.ico", "web/dist/favicon.ico")
		router.StaticFile("/index.html", "web/dist/index.html")
		router.GET("/", func(c *gin.Context) {
			c.File("web/dist/index.html")
		})
	} else {
		// Fallback: serve embedded template
		router.GET("/", func(c *gin.Context) {
			data, _ := indexHTML.ReadFile("templates/index.html")
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		})
	}

	router.NoRoute(func(c *gin.Context) {
		// API 404s should return JSON
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "not found"})
			return
		}
		// All other 404s serve the React frontend (SPA routing)
		if _, err := os.Stat("web/dist/index.html"); err == nil {
			c.File("web/dist/index.html")
		} else {
			data, _ := indexHTML.ReadFile("templates/index.html")
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		}
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
