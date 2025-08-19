package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/lypolix/todo-app"
	"github.com/lypolix/todo-app/pkg/handler"
	"github.com/lypolix/todo-app/pkg/repository"
	"github.com/lypolix/todo-app/pkg/service"
	"github.com/lypolix/todo-app/pkg/wsserver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// @title Todo App API
// @version 1.0
// @description API Server for Todo

// @host localhost:8000
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization


func main(){

	server := wsserver.NewWsServer(":8000")
	logrus.Info("Started ws server")
	if err := server.Start(); err != nil {
		logrus.Errorf("Error with ws server: %v", err)
	}
  

	logrus.SetFormatter(new(logrus.JSONFormatter))
	if err := initConfig(); err != nil {
		logrus.Fatalf ("error initializing configs: %s", err.Error())
	}

	if err := godotenv.Load(); err != nil {
		logrus.Fatalf("error loading env variables: %s", err.Error())
	}

	db, err := repository.NewPostgresDB(repository.Config{
		Host: viper.GetString("db.host"),
		Port: viper.GetString("db.port"),
		Username: viper.GetString("db.username"),
		DBName: viper.GetString("db.dbname"),
		SSLMode: viper.GetString("db.sslmode"),
		Password: os.Getenv("DB_PASSWORD"),
	})

	if err != nil {
		logrus.Fatalf("failed to ititializedb: %s", err.Error())
	}

	repos := repository.NewRepository(db)
	services := service.NewService(repos)
	handlers := handler.NewHandler(services)

	srv := new(todo.Server)
	go func () {
		if err := srv.Run(viper.GetString("port"), handlers.InitRoutes()); err != nil {
			logrus.Fatalf("error occured while running http server: %s", err.Error())
		}
	}()

	logrus.Print("TodoApp Started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<- quit

	logrus.Print("TodoApp Shutting Down")

	if err := srv.Shutdown(context.Background()); err != nil {
		logrus.Errorf("error occured on server shutting down: %s", err.Error())
	}

	if err := db.Close(); err != nil {
		logrus.Errorf("error occured on db connection close: %s", err.Error())
	}
}



func initConfig() error{
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
