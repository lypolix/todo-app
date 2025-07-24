package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/lypolix/todo-app"
	"github.com/lypolix/todo-app/pkg/cache"
	"github.com/lypolix/todo-app/pkg/handler"
	"github.com/lypolix/todo-app/pkg/repository"
	"github.com/lypolix/todo-app/pkg/service"
	"github.com/lypolix/todo-app/pkg/websocket"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	
	logrus.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})

	if err := initConfig(); err != nil {
		logrus.Fatalf("init config: %v", err)
	}
	if err := godotenv.Load(); err != nil {
		logrus.Fatalf("load .env: %v", err)
	}

	db, err := repository.NewPostgresDB(repository.Config{
		Host:     viper.GetString("db.host"),
		Port:     viper.GetString("db.port"),
		Username: viper.GetString("db.username"),
		DBName:   viper.GetString("db.dbname"),
		SSLMode:  viper.GetString("db.sslmode"),
		Password: os.Getenv("DB_PASSWORD"),
	})
	if err != nil {
		logrus.Fatalf("postgres: %v", err)
	}

	redisClient, err := cache.NewRedisClient(cache.RedisConfig{
		Host:     viper.GetString("redis.host"),
		Port:     viper.GetString("redis.port"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       viper.GetInt("redis.db"),
	})
	if err != nil {
		logrus.Fatalf("redis: %v", err)
	}
	cacheService := cache.NewCacheService(redisClient)

	
	wsHub := websocket.NewHub()
	go wsHub.Run()

	
	repos := repository.NewRepository(db)
	services := service.NewService(repos, cacheService, wsHub)
	handlers := handler.NewHandler(services, wsHub)

	
	srv := new(todo.Server)
	go func() {
		if err := srv.Run(viper.GetString("port"), handlers.InitRoutes()); err != nil {
			logrus.Fatalf("http server: %v", err)
		}
	}()
	logrus.Info("TodoApp started")

	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("shutting down")

	if err := srv.Shutdown(context.Background()); err != nil {
		logrus.Errorf("server shutdown: %v", err)
	}
	if err := db.Close(); err != nil {
		logrus.Errorf("db close: %v", err)
	}
	if err := redisClient.Close(); err != nil {
		logrus.Errorf("redis close: %v", err)
	}
}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}