package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/flash-deals/order/internal/config"
	"github.com/flash-deals/order/internal/handler"
	"github.com/flash-deals/order/internal/product"
	"github.com/flash-deals/order/internal/service"
	"github.com/flash-deals/order/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	e.POST("/orders/:id/cancel", orderHandler.CancelOrder)

	// Start HTTP server
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("HTTP server starting on %s", addr)
	if err := e.Start(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
