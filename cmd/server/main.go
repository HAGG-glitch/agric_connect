package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agriconnect-ai/internal/ai"
	"github.com/agriconnect-ai/internal/auth"
	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/database"
	"github.com/agriconnect-ai/internal/diagnosis"
	"github.com/agriconnect-ai/internal/handlers"
	"github.com/agriconnect-ai/internal/middleware"
	"github.com/agriconnect-ai/internal/repositories"
	"github.com/agriconnect-ai/internal/services"
	"github.com/agriconnect-ai/internal/storage"
	"github.com/agriconnect-ai/internal/transcription"
	"github.com/agriconnect-ai/internal/weather"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	migrationsDir := findMigrationsDir()
	migrator := database.NewMigrationRunner(db, migrationsDir)
	if err := migrator.Up(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Repositories
	convRepo := repositories.NewConversationRepository(db)
	msgRepo := repositories.NewMessageRepository(db)
	knowledgeRepo := repositories.NewKnowledgeRepository(db)
	weatherRepo := repositories.NewWeatherRepository(db)
	diagnosisRepo := diagnosis.NewRepository(db)

	// Weather client
	weatherClient := weather.NewClient(cfg.OpenMeteoBaseURL)

	// AI client
	aiClient := ai.NewClient(cfg.GroqAPIKey, cfg.GroqBaseURL, cfg.GroqChatModel, cfg.GroqRequestTimeoutSecs)

	// Services
	knowledgeSvc := services.NewKnowledgeService(knowledgeRepo)
	weatherSvc := services.NewWeatherService(weatherRepo, weatherClient, cfg.WeatherCacheMinutes)
	chatSvc := services.NewChatService(convRepo, msgRepo, cfg.GroqChatModel)

	// Orchestrator
	orchestrator := ai.NewOrchestrator(aiClient, knowledgeSvc, weatherSvc)

	// Storage
	var objStore storage.ObjectStorage
	switch cfg.StorageDriver {
	case "supabase":
		secretKey := cfg.SupabaseSecretKey
		if secretKey == "" {
			secretKey = cfg.SupabaseServiceRoleKey
		}
		if cfg.SupabaseURL == "" || secretKey == "" {
			log.Fatalf("SUPABASE_URL and SUPABASE_SECRET_KEY must be set when STORAGE_DRIVER=supabase")
		}
		objStore = storage.NewSupabaseStorage(cfg.SupabaseURL, secretKey, cfg.SupabaseStorageBucket)
	default:
		ls, err := storage.NewLocalStorage(cfg.LocalUploadDir)
		if err != nil {
			log.Fatalf("Failed to init local storage: %v", err)
		}
		objStore = ls
	}

	// Diagnosis
	diagnosisAI := ai.NewCropDiagnosisAI(aiClient, cfg.GroqVisionModel)
	diagnosisSvc := diagnosis.NewService(diagnosisRepo, objStore, diagnosisAI, knowledgeSvc, cfg)

	// Transcription
	audioTranscriber := ai.NewAudioTranscriber(cfg.GroqAPIKey, cfg.GroqBaseURL, cfg.GroqTranscriptionModel)
	transcriptionSvc := transcription.NewService(audioTranscriber)

	// Auth
	accessDur, _ := time.ParseDuration(cfg.JWTAccessDuration)
	refreshDur, _ := time.ParseDuration(cfg.JWTRefreshDuration)
	authSvc := auth.NewService(db, cfg.JWTAccessSecret, cfg.JWTRefreshSecret, accessDur, refreshDur)

	// Handlers
	pageHandler := handlers.NewPageHandler(cfg)
	convHandler := handlers.NewConversationHandler(chatSvc)
	chatHandler := handlers.NewChatHandler(chatSvc, orchestrator, cfg)
	weatherHandler := handlers.NewWeatherHandler(weatherSvc)
	healthHandler := handlers.NewHealthHandler(db)
	diagnosisHandler := handlers.NewDiagnosisHandler(diagnosisSvc, cfg, objStore, chatSvc)
	transcriptionHandler := handlers.NewTranscriptionHandler(transcriptionSvc, cfg)
	authHandler := handlers.NewAuthHandler(authSvc, cfg.CookieSecure, cfg.CookieDomain, cfg.CookieSameSite, cfg.JWTRefreshSecret)
	officerHandler := handlers.NewOfficerHandler(db, diagnosisSvc)
	adminHandler := handlers.NewAdminHandler(db)
	notifHandler := handlers.NewNotificationHandler(db)

	router := gin.Default()
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery())
	router.Use(middleware.AnonymousUser(cfg.CookieSecure, cfg.CookieDomain, cfg.CookieSameSite))
	router.Use(middleware.RateLimit(cfg.RateLimitPerMinute))

	router.SetTrustedProxies(nil)

	// Static files
	router.Static("/static", "./web/static")

	// Templates
	assetVersion := fmt.Sprintf("%x", time.Now().Unix())
	router.SetFuncMap(template.FuncMap{
		"json": func(v any) (template.HTML, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return template.HTML(b), nil
		},
		"RainProbability": func(d weather.DailyForecast) int {
			return d.RainProbabilityPercent
		},
		"assetVersion": func() string {
			return assetVersion
		},
	})
	router.LoadHTMLGlob("web/templates/**/*.html")

	// Health
	router.GET("/health", healthHandler.Check)

	// Public pages (with optional auth to recognize logged-in users)
	publicPages := router.Group("")
	publicPages.Use(middleware.OptionalAuth(cfg.JWTAccessSecret, db))
	{
		publicPages.GET("/", pageHandler.AssistantPage)
		publicPages.GET("/assistant", pageHandler.AssistantPage)
		publicPages.GET("/diagnose", diagnosisHandler.DiagnosePage)
		publicPages.GET("/diagnoses", diagnosisHandler.HistoryPage)
		publicPages.GET("/diagnoses/:id", diagnosisHandler.DetailPage)
	}

	// Auth pages (with optional auth to redirect already-logged-in users)
	authPages := router.Group("")
	authPages.Use(middleware.OptionalAuth(cfg.JWTAccessSecret, db))
	{
		authPages.GET("/login", authHandler.LoginPage)
		authPages.GET("/register", authHandler.RegisterPage)
	}

	// Officer pages (require auth)
	officerPages := router.Group("")
	officerPages.Use(middleware.OptionalAuth(cfg.JWTAccessSecret, db))
	officerPages.Use(middleware.RequireRole("officer", "admin"))
	{
		officerPages.GET("/officer", officerHandler.OfficerPage)
		officerPages.GET("/officer/diagnoses", officerHandler.OfficerDiagnosesPage)
		officerPages.GET("/officer/diagnoses/:id", officerHandler.OfficerDiagnosisDetailPage)
	}

	// Admin pages
	adminPages := router.Group("")
	adminPages.Use(middleware.OptionalAuth(cfg.JWTAccessSecret, db))
	adminPages.Use(middleware.RequireRole("admin"))
	{
		adminPages.GET("/admin/users", adminHandler.AdminPage)
	}

	// API v1 — auth flows that must work without a prior session
	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/register", authHandler.Register)
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/auth/refresh", authHandler.Refresh)
	}

	// API v1 — routes that recognise authenticated users (OptionalAuth)
	v1User := router.Group("/api/v1")
	v1User.Use(middleware.OptionalAuth(cfg.JWTAccessSecret, db))
	{
		// Auth
		v1User.POST("/auth/logout", authHandler.Logout)
		v1User.GET("/auth/me", authHandler.Me)

		// Conversations
		v1User.POST("/conversations", convHandler.Create)
		v1User.GET("/conversations", convHandler.List)
		v1User.GET("/conversations/:id", convHandler.Get)
		v1User.DELETE("/conversations/:id", convHandler.Delete)

		v1User.POST("/conversations/:id/messages", chatHandler.SendMessage)
		v1User.POST("/conversations/:id/messages/stream", chatHandler.StreamMessage)

		// Weather
		v1User.GET("/weather", weatherHandler.GetWeather)

		// Diagnoses
		v1User.POST("/diagnoses", diagnosisHandler.Create)
		v1User.GET("/diagnoses", diagnosisHandler.List)
		v1User.GET("/diagnoses/:id", diagnosisHandler.Get)
		v1User.DELETE("/diagnoses/:id", diagnosisHandler.Delete)
		v1User.GET("/diagnoses/:id/image", diagnosisHandler.ServeImage)
		v1User.POST("/diagnoses/:id/continue-in-chat", diagnosisHandler.ContinueInChat)

		// Transcription
		v1User.POST("/ai/transcribe", transcriptionHandler.Transcribe)
	}

	// API v1 — officer (auth + role check)
	officerAPI := router.Group("/api/v1/officer")
	officerAPI.Use(middleware.OptionalAuth(cfg.JWTAccessSecret, db))
	officerAPI.Use(middleware.RequireRole("officer", "admin"))
	{
		officerAPI.GET("/diagnoses", officerHandler.ListDiagnoses)
		officerAPI.GET("/diagnoses/:id", officerHandler.GetDiagnosis)
		officerAPI.POST("/diagnoses/:id/reviews", officerHandler.CreateReview)
		officerAPI.PUT("/diagnoses/:id/reviews/:reviewID", officerHandler.UpdateReview)
	}

	// API v1 — admin (auth + role check)
	adminAPI := router.Group("/api/v1/admin")
	adminAPI.Use(middleware.OptionalAuth(cfg.JWTAccessSecret, db))
	adminAPI.Use(middleware.RequireRole("admin"))
	{
		adminAPI.GET("/users", adminHandler.ListUsers)
		adminAPI.PATCH("/users/:userId/role", adminHandler.UpdateRole)
		adminAPI.PATCH("/users/:userId/status", adminHandler.UpdateStatus)
	}

	// API v1 — notifications (auth, any role)
	notifAPI := router.Group("/api/v1/notifications")
	notifAPI.Use(middleware.OptionalAuth(cfg.JWTAccessSecret, db))
	{
		notifAPI.GET("", notifHandler.List)
		notifAPI.PATCH("/:id/read", notifHandler.MarkRead)
	}

	// Determine port: Render provides PORT, fall back to APP_PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.AppPort
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("AgriConnect AI starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Close database connection
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited cleanly")
}

func findMigrationsDir() string {
	return "migrations"
}
