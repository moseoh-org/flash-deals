package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/flash-deals/order/internal/config"
	"github.com/flash-deals/order/internal/handler"
	"github.com/flash-deals/order/internal/product"
	"github.com/flash-deals/order/internal/queue"
	"github.com/flash-deals/order/internal/service"
	"github.com/flash-deals/order/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	// Initialize OpenTelemetry
	if cfg.OTelEnabled {
		endpoint := strings.TrimPrefix(cfg.OTelExporterEndpoint, "http://")
		shutdown, err := telemetry.InitTracer(ctx, cfg.OTelServiceName, endpoint)
		if err != nil {
			log.Printf("Warning: Failed to initialize tracer: %v", err)
		} else {
			defer shutdown(ctx)
			log.Println("OpenTelemetry initialized")
		}
	}

	// Initialize Database
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	// Initialize Product gRPC Client
	if err := product.InitClient(cfg.ProductGRPCAddr(), cfg.OTelEnabled); err != nil {
		log.Fatalf("Failed to initialize product client: %v", err)
	}
	defer product.Close()
	log.Printf("Product gRPC client initialized: %s", cfg.ProductGRPCAddr())

	// Initialize Service and Handler
	orderService := service.NewOrderService(pool)
	orderHandler := handler.NewOrderHandler(orderService)

	// Initialize Redis and Async Order Processing if enabled
	if cfg.AsyncOrderEnabled {
		redisClient := redis.NewClient(&redis.Options{
			Addr: cfg.RedisAddr(),
		})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}
		defer redisClient.Close()
		log.Printf("Connected to Redis: %s", cfg.RedisAddr())

		// Setup order queue
		orderQueue := queue.NewOrderQueue(redisClient)
		orderService.SetOrderQueue(orderQueue)
		orderHandler.SetAsyncEnabled(true)

		// Start order worker
		orderWorker := queue.NewOrderWorker(redisClient, pool)
		go orderWorker.Start(ctx)

		log.Println("Async order processing enabled (Redis List)")
	} else {
		log.Println("Sync order processing (DB)")
	}

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	if cfg.OTelEnabled {
		e.Use(otelecho.Middleware(cfg.OTelServiceName))
	}

	// Routes
	e.GET("/health", orderHandler.Health)
	e.GET("/orders/health", orderHandler.Health)

	// Orders
	e.GET("/orders", orderHandler.ListOrders)
	e.POST("/orders", orderHandler.CreateOrder)
	e.GET("/orders/:id", orderHandler.GetOrder)
	e.POST("/orders/:id/confirm", orderHandler.ConfirmOrder)
	e.POST("/orders/:id/cancel", orderHandler.CancelOrder)

	// Start HTTP server
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("HTTP server starting on %s", addr)
	if err := e.Start(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
