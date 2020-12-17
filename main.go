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

	"github.com/sirupsen/logrus"

	"github.com/maknahar/alpha-flow/internal/configs"
	"github.com/maknahar/alpha-flow/internal/routes"
)

func main() {
	config, err := configs.Configure(context.Background())
	if err != nil {
		logrus.WithError(err).Panic("Unable to start the application. Error in configuration.")
	}

	server := &http.Server{
		Addr:    config.Host,
		Handler: routes.Get(config),
	}

	config.Logger.Info("Starting the service on ", config.Host)

	go func() {
		if sErr := server.ListenAndServe(); sErr != nil && !errors.Is(sErr, http.ErrServerClosed) {
			config.Logger.Fatal(sErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	config.Logger.Info("Service is shutting down")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err = server.Shutdown(timeoutCtx); err != nil {
		config.Logger.Errorln("Error during HTTP server shutdown:", err)
	}

	log.Println("Service Stopped")
}
