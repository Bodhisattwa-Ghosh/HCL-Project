package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/config"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/httpapi"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/realtime"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/repository"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Docker and hosted providers inject environment variables directly. This
	// optional load makes `go run` pleasant for local development too.
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("connect MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("ping MongoDB: %v", err)
	}

	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("ping Redis: %v", err)
	}

	store := repository.New(mongoClient, cfg.MongoDatabase)
	if err := store.EnsureIndexes(ctx); err != nil {
		log.Fatalf("create MongoDB indexes: %v", err)
	}
	broker := realtime.New(redisClient)
	auth := service.NewAuth(store, cfg.JWTSecret, cfg.JWTTTL)
	polls := service.NewPolls(store, redisClient, broker)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), httpapi.SecurityHeaders(), httpapi.CORS(cfg.FrontendOrigin))
	httpapi.New(auth, polls, broker, cfg.CookieSecure, cfg.CookieSameSite).Register(router)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0, // SSE responses stay open; per-request contexts still cancel on disconnect.
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("PulsePoll API listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve HTTP: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}
