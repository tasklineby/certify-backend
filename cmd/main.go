package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/tasklineby/certify-backend/config"
	"github.com/tasklineby/certify-backend/db"
	"github.com/tasklineby/certify-backend/repository/pg"
	"github.com/tasklineby/certify-backend/repository/rdb"
	"github.com/tasklineby/certify-backend/service"
	"github.com/tasklineby/certify-backend/transport/rest/handlers"

	_ "github.com/tasklineby/certify-backend/docs" // swagger docs
)

// @title           Certify Backend API
// @version         1.0
// @description     Artemon loh
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

var programLevel = new(slog.LevelVar)
var configPath = "/app/.env"

func main() {
	programLevel.Set(slog.LevelInfo)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	})
	slog.SetDefault(slog.New(handler))
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		slog.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	dbConn, err := db.InitPostgresDB(cfg.Database.GetDSN())
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	redisClient, err := db.InitRedis(cfg.Redis.GetAddr(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	userRepo := pg.NewUserRepository(dbConn)
	companyRepo := pg.NewCompanyRepository(dbConn)
	documentRepo := pg.NewDocumentRepository(dbConn)
	historyRepo := pg.NewHistoryRepository(dbConn)
	tokenRepo := rdb.NewTokenRepository(redisClient)

	jwtService := service.NewJwtService(
		cfg.Jwt.AccessTokenSecret,
		cfg.Jwt.AccessTokenTTL*time.Minute,
		cfg.Jwt.RefreshTokenTTL*time.Hour,
	)
	userService := service.NewUserService(userRepo, companyRepo)
	authService := service.NewAuthService(userService, tokenRepo, jwtService)
	documentService := service.NewDocumentService(documentRepo, historyRepo)

	userHandler := handlers.NewUserHandler(userService, jwtService, tokenRepo)
	authHandler := handlers.NewAuthHandler(authService)
	documentHandler := handlers.NewDocumentHandler(documentService)

	router := handlers.InitRoutes(userHandler, authHandler, documentHandler, authService)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("Server start up", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server err: %w", err)
		}
	}()
	shutdownApp(server, dbConn, redisClient, errCh)
}

func shutdownApp(server *http.Server, db *sqlx.DB, redisClient *redis.Client, serverErrCh <-chan error) {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case err := <-serverErrCh:
		slog.Error("Server error", "error", err)
	case sig := <-signalCh:
		slog.Info("Received sys signal to begin graceful shutdown", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if server != nil {
		slog.Info("Shutting down the server...")
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("Failed to shut down the server", "error", err)
		}
	}

	if db != nil {
		slog.Info("Disconnecting from Postgres...")
		err := db.Close()
		if err != nil {
			slog.Error("Error disconnecting from Postgres", "error", err)
		}
	}

	if redisClient != nil {
		slog.Info("Disconnecting from Redis...")
		err := redisClient.Close()
		if err != nil {
			slog.Error("Error disconnecting from Redis", "error", err)
		}
	}

	slog.Info("Application shutdown complete")
}
