package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/redora/redora/backend/internal/config"
	"github.com/redora/redora/backend/internal/crypto"
	"github.com/redora/redora/backend/internal/database"
	"github.com/redora/redora/backend/internal/handler"
	"github.com/redora/redora/backend/internal/middleware"
	"github.com/redora/redora/backend/internal/redis"
	"github.com/redora/redora/backend/internal/repository"
	"github.com/redora/redora/backend/internal/service"
)

//go:embed all:dist
var webFS embed.FS

func main() {
	cfg := config.Load()

	// Configure structured logger
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	slog.Info("Starting Redora backend", "port", cfg.Port, "db_path", cfg.DBPath)

	// Initialize SQLite Database
	db, err := database.InitDB(cfg.DBPath)
	if err != nil {
		slog.Error("Database initialization failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize Services & Dependencies
	encryptor := crypto.NewEncryptor(cfg.EncryptionKey)
	redisMgr := redis.NewConnectionManager()
	defer redisMgr.CloseAll()

	connRepo := repository.NewConnectionRepository(db)
	connSvc := service.NewConnectionService(connRepo, encryptor, redisMgr)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "HTTP_ERROR",
					"message": err.Error(),
				},
			})
		},
	})

	// Register global middleware
	app.Use(middleware.Recover())
	app.Use(middleware.Logger())

	// API Routes Group
	api := app.Group("/api")

	// Health Handler
	healthHandler := handler.NewHealthHandler("1.0.0")
	healthHandler.RegisterRoutes(api)

	// Connection Handler
	connHandler := handler.NewConnectionHandler(connSvc)
	connHandler.RegisterRoutes(api)

	// Serve Frontend (Embedded assets or fallback)
	distFS, err := fs.Sub(webFS, "dist")
	if err == nil {
		app.Use("/", filesystem.New(filesystem.Config{
			Root:         http.FS(distFS),
			Browse:       false,
			Index:        "index.html",
			NotFoundFile: "index.html",
		}))
	} else {
		app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("Redora API Server is running. (Frontend assets not embedded in dev mode)")
		})
	}

	// Channel for graceful shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		slog.Info("Redora server running", "addr", addr)
		if err := app.Listen(addr); err != nil {
			slog.Error("Server error", "error", err)
		}
	}()

	<-stop
	slog.Info("Shutting down Redora server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Redora server stopped gracefully")
}
