package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/flash-deals/product/internal/config"
	grpcserver "github.com/flash-deals/product/internal/grpc"
	"github.com/flash-deals/product/internal/handler"
	"github.com/flash-deals/product/internal/repository"
	"github.com/flash-deals/product/internal/telemetry"
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

	// Initialize Repository
	dbRepo := repository.NewDBRepository(pool)
	var repo repository.ProductRepository = dbRepo

	// Initialize Redis and Cached Repository if enabled
	if cfg.EnableCache {
		redisClient := redis.NewClient(&redis.Options{
			Addr: cfg.RedisAddr(),
		})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Printf("Warning: Failed to connect to Redis: %v", err)
		} else {
			repo = repository.NewCachedRepository(dbRepo, redisClient, cfg.CacheTTL)
			log.Println("Cache enabled with Redis")
		}
	}

	// Initialize Handler
	productHandler := handler.NewProductHandler(repo)

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
	e.GET("/health", productHandler.Health)

	// Products
	e.POST("/products", productHandler.CreateProduct)
	e.GET("/products", productHandler.ListProducts)
	e.GET("/products/:id", productHandler.GetProduct)
	e.PATCH("/products/:id", productHandler.UpdateProduct)

	// Stock
	e.GET("/products/:id/stock", productHandler.GetStock)
	e.PATCH("/products/:id/stock", productHandler.UpdateStock)

	// Deals
	e.POST("/deals", productHandler.CreateDeal)
	e.GET("/deals", productHandler.ListActiveDeals)
	e.GET("/deals/:id", productHandler.GetDeal)

	// Start gRPC server if enabled
	if cfg.GRPCEnabled {
		go func() {
			if err := grpcserver.StartGRPCServer(cfg, repo, cfg.OTelEnabled); err != nil {
				log.Printf("gRPC server error: %v", err)
			}
		}()
	}

	// Start HTTP server
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("HTTP server starting on %s", addr)
	if err := e.Start(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
