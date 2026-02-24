package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	// Internal packages
	"github.com/seu-repo/sigec-ve/internal/adapter/ai/gemini"
	"github.com/seu-repo/sigec-ve/internal/adapter/cache"
	payment "github.com/seu-repo/sigec-ve/internal/adapter/external/payment"
	"github.com/seu-repo/sigec-ve/internal/adapter/grpc/server"
	"github.com/seu-repo/sigec-ve/internal/ports"
	"github.com/seu-repo/sigec-ve/internal/adapter/http/fiber/handlers"
	"github.com/seu-repo/sigec-ve/internal/adapter/http/fiber/middleware"
	v201 "github.com/seu-repo/sigec-ve/internal/adapter/ocpp/v201"
	"github.com/seu-repo/sigec-ve/internal/adapter/queue"
	nzdb "github.com/seu-repo/sigec-ve/internal/adapter/storage/nietzsche"
	wsAdapter "github.com/seu-repo/sigec-ve/internal/adapter/websocket"
	"github.com/seu-repo/sigec-ve/internal/observability/telemetry"
	"github.com/seu-repo/sigec-ve/internal/service/auth"
	"github.com/seu-repo/sigec-ve/internal/service/device"
	"github.com/seu-repo/sigec-ve/internal/service/transaction"
	"github.com/seu-repo/sigec-ve/internal/service/voice"
	"github.com/seu-repo/sigec-ve/pkg/config"

	// Import metrics to register them
	_ "github.com/seu-repo/sigec-ve/internal/observability/telemetry"
)

const (
	serviceName    = "sigec-ve-enterprise"
	serviceVersion = "v1.0.0"
)

func main() {
	// 1. Initialize Logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("Starting SIGEC-VE Enterprise",
		zap.String("service", serviceName),
		zap.String("version", serviceVersion),
	)

	// 2. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// 3. Initialize OpenTelemetry (Distributed Tracing)
	tracerProvider, err := telemetry.InitTracer(serviceName)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}()

	// 4. Initialize NietzscheDB Connection
	nietzscheAddr := os.Getenv("NIETZSCHE_ADDR")
	if nietzscheAddr == "" {
		nietzscheAddr = "34.123.223.50:50051"
	}
	db, err := nzdb.NewConnection(nietzscheAddr, logger)
	if err != nil {
		logger.Fatal("Failed to connect to NietzscheDB", zap.Error(err))
	}
	defer db.Close()

	// 5. Initialize Redis Cache
	redisCache, err := cache.NewRedisCache(cfg.Redis.URL, logger)
	if err != nil {
		logger.Warn("Redis not available, running without cache", zap.Error(err))
		redisCache = nil
	}
	if redisCache != nil {
		defer redisCache.Close()
	}

	// 6. Initialize Message Queue (NATS) - Optional
	messageQueue, err := queue.NewNATSQueue(cfg.NATS.URL, logger)
	if err != nil {
		logger.Warn("NATS not available, running without message queue", zap.Error(err))
		messageQueue = nil
	} else {
		defer messageQueue.Close()
	}

	// 7. Initialize Repositories (NietzscheDB-backed)
	chargePointRepo := nzdb.NewChargePointRepository(db, logger)
	transactionRepo := nzdb.NewTransactionRepository(db, logger)
	userRepo := nzdb.NewUserRepository(db, logger)

	// 8. Initialize Payment Gateway (Stripe)
	stripeGateway := payment.NewStripeService(cfg.Payment.Stripe.SecretKey, logger)

	// 9. Initialize Services (Business Logic Layer)
	authService := auth.NewService(userRepo, redisCache, cfg.JWT.Secret, logger)
	deviceService := device.NewService(chargePointRepo, redisCache, messageQueue, logger)
	transactionService := transaction.NewService(transactionRepo, deviceService, messageQueue, logger)
	billingService := transaction.NewBillingService(transactionRepo, messageQueue, transaction.DefaultPricingConfig(), logger)


	// 9. Initialize Gemini Live API Client (Voice)
	geminiClient := gemini.NewLiveClient(cfg.Gemini.APIKey, logger)
	voiceAssistant := voice.NewVoiceAssistant(geminiClient, deviceService, transactionService, logger)

	// 10. Initialize OCPP 2.0.1 Server
	ocppServer := v201.NewServer(deviceService, transactionService, logger)
	go func() {
		logger.Info("Starting OCPP WebSocket Server", zap.Int("port", cfg.OCPP.Port))
		if err := ocppServer.Start(cfg.OCPP.Port); err != nil {
			logger.Fatal("OCPP Server failed", zap.Error(err))
		}
	}()

	// 11. Initialize WebSocket Hub (for real-time updates)
	wsHub := wsAdapter.NewHub()
	go wsHub.Run()

	// 12. Initialize Voice Stream Handler
	voiceStreamHandler := wsAdapter.NewVoiceStreamHandler(voiceAssistant, logger)

	// 13. Initialize Fiber HTTP Server
	app := fiber.New(fiber.Config{
		AppName:               serviceName,
		ServerHeader:          serviceName,
		DisableStartupMessage: true,
		ErrorHandler:          middleware.ErrorHandler(logger),
	})

	// Global Middleware
	app.Use(recover.New())
	app.Use(fiberlogger.New()) // Fiber logger middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Join(cfg.HTTP.AllowedOrigins, ","),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))
	app.Use(middleware.RateLimit())
	app.Use(middleware.CircuitBreakerWithLogger(logger))
	// app.Use(middleware.RequestID()) // Assuming this exists or uses fiber's
	// app.Use(telemetry.HTTPMiddleware()) // Assuming this exists

	// Health Check Endpoints
	app.Get("/health/live", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		// Check NietzscheDB
		if err := db.Client.HealthCheck(context.Background()); err != nil {
			return c.Status(503).SendString("Database not ready")
		}
		if redisCache != nil {
			if err := redisCache.Ping(); err != nil {
				return c.Status(503).SendString("Cache not ready")
			}
		}
		return c.SendString("Ready")
	})

	// Metrics endpoint for Prometheus
	app.Get("/metrics", func(c *fiber.Ctx) error {
		// Adapt net/http handler to fasthttp for Fiber
		handler := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
		handler(c.Context())
		return nil
	})

	// API v1 Routes
	v1 := app.Group("/api/v1")

	// Auth routes (public)
	authHandler := handlers.NewAuthHandler(authService, logger)
	v1.Post("/auth/login", authHandler.Login)
	v1.Post("/auth/register", authHandler.Register)
	v1.Post("/auth/refresh", authHandler.RefreshToken)

	// Protected routes
	protected := v1.Group("", middleware.AuthRequired(authService))

	// Auth protected routes
	protected.Get("/auth/me", authHandler.Me)

	// Device routes (nearby MUST come before :id to avoid matching "nearby" as id param)
	deviceHandler := handlers.NewDeviceHandler(deviceService, logger)
	protected.Get("/devices", deviceHandler.List)
	protected.Get("/devices/nearby", deviceHandler.GetNearby)
	protected.Get("/devices/:id", deviceHandler.Get)
	protected.Patch("/devices/:id/status", deviceHandler.UpdateStatus)

	// Transaction routes (specific paths MUST come before :id to avoid matching as param)
	txHandler := handlers.NewTransactionHandler(transactionService, logger)
	protected.Post("/transactions/start", txHandler.Start)
	protected.Get("/transactions/history", txHandler.GetHistory)
	protected.Get("/transactions/active", txHandler.GetActive)
	protected.Post("/transactions/:id/stop", txHandler.Stop)
	protected.Get("/transactions/:id", txHandler.Get)

	// Voice routes
	voiceHandler := handlers.NewVoiceHandler(voiceAssistant, logger)
	protected.Post("/voice/command", voiceHandler.ProcessCommand)
	protected.Get("/voice/history", voiceHandler.GetHistory)

	// WebSocket routes
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Real-time updates WebSocket
	app.Get("/ws/updates", websocket.New(func(c *websocket.Conn) {
		// Extract userID from locals/token. For now assume "guest" or extract from query
		userID := c.Query("userId", "guest")
		wsHub.AddClient(c, userID)
	}))

	// Voice streaming WebSocket
	app.Get("/ws/voice", websocket.New(func(c *websocket.Conn) {
		voiceStreamHandler.HandleVoiceStream(c)
	}))

	// 14. Initialize gRPC Server (for internal microservices communication)
	grpcServer := server.NewGRPCServer(deviceService, transactionService, logger)
	go func() {
		logger.Info("Starting gRPC Server", zap.Int("port", cfg.GRPC.Port))
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
		if err != nil {
			logger.Fatal("Failed to listen for gRPC", zap.Error(err))
		}
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC Server failed", zap.Error(err))
		}
	}()

	// 15. Start Background Workers (only if NATS available)
	if messageQueue != nil {
		go startBackgroundWorkers(messageQueue, billingService, stripeGateway, transactionRepo, logger)
	}

	// 16. Start HTTP Server
	go func() {
		logger.Info("Starting HTTP Server", zap.Int("port", cfg.HTTP.Port))
		if err := app.Listen(fmt.Sprintf(":%d", cfg.HTTP.Port)); err != nil {
			logger.Fatal("HTTP Server failed", zap.Error(err))
		}
	}()

	// 17. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	ocppServer.Stop()

	logger.Info("Server exited gracefully")
}

// startBackgroundWorkers starts async jobs like billing, analytics, etc.
func startBackgroundWorkers(mq queue.MessageQueue, billing *transaction.BillingService, pg ports.PaymentGateway, txRepo ports.TransactionRepository, logger *zap.Logger) {
	logger.Info("Starting background workers")

	// Worker 1: Process billing payment events
	mq.Subscribe("billing.payment.required", func(msg []byte) error {
		logger.Info("Processing billing payment event", zap.ByteString("msg", msg))

		var event struct {
			TransactionID string  `json:"transaction_id"`
			UserID        string  `json:"user_id"`
			Amount        float64 `json:"amount"`
			Currency      string  `json:"currency"`
		}
		if err := json.Unmarshal(msg, &event); err != nil {
			logger.Error("Failed to unmarshal billing event", zap.Error(err))
			return err
		}

		// Create payment intent via Stripe
		piID, err := pg.CreatePaymentIntent(context.Background(), event.Amount, event.Currency, event.UserID)
		if err != nil {
			logger.Error("Failed to create payment intent", zap.Error(err), zap.String("tx_id", event.TransactionID))
			return err
		}
		logger.Info("Payment intent created for transaction",
			zap.String("tx_id", event.TransactionID),
			zap.String("payment_intent_id", piID),
		)
		return nil
	})

	// Worker 2: Process general billing events
	mq.Subscribe("billing.events", func(msg []byte) error {
		logger.Info("Processing billing event", zap.ByteString("msg", msg))
		return nil
	})

	// Worker 3: Send notifications
	mq.Subscribe("notifications.events", func(msg []byte) error {
		logger.Info("Sending notification", zap.ByteString("msg", msg))
		return nil
	})

	// Worker 4: Analytics aggregation
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		logger.Info("Running analytics aggregation")
	}
}
