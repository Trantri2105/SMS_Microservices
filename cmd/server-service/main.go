package main

import (
	"VCS_SMS_Microservice/internal/server-service/api/handler"
	"VCS_SMS_Microservice/internal/server-service/api/routes"
	"VCS_SMS_Microservice/internal/server-service/config"
	"VCS_SMS_Microservice/internal/server-service/repository"
	"VCS_SMS_Microservice/internal/server-service/service"
	"VCS_SMS_Microservice/pkg/infra"
	"VCS_SMS_Microservice/pkg/logger"
	"VCS_SMS_Microservice/pkg/mail"
	"VCS_SMS_Microservice/pkg/middleware"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func main() {
	appConfig, err := config.LoadConfig("./.env")
	if err != nil {
		log.Fatal(fmt.Sprintf("load config error: %v", err))
	}

	// set up logger
	fileSyncer, err := logger.NewReopenableWriteSyncer("./log/server-service.log")
	zapLogger := logger.NewLogger(appConfig.Server.LogLevel, fileSyncer).With(zap.String("service.name", "server-service"))
	defer zapLogger.Sync()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for {
			<-c
			zapLogger.Info("receive logrotate SIGHUP, reloading log file")
			if e := fileSyncer.Reload(); e != nil {
				zapLogger.Error("failed to reload log file", zap.Error(e))
			} else {
				zapLogger.Info("successfully reloaded log file")
			}
		}
	}()

	//set up database
	db, err := infra.NewPostgresConnection(infra.PostgresConfig{
		Host:     appConfig.Postgres.Host,
		Port:     appConfig.Postgres.Port,
		User:     appConfig.Postgres.User,
		Password: appConfig.Postgres.Password,
		DBName:   appConfig.Postgres.DBName,
	})
	if err != nil {
		zapLogger.Fatal("failed to connect to postgres", zap.Error(err))
	} else {
		zapLogger.Info("connected to postgres successfully")
	}
	sqlDB, err := db.DB()
	if err != nil {
		zapLogger.Fatal("failed to get sql.DB from gorm:", zap.Error(err))
	}
	defer sqlDB.Close()

	//set up elasticsearch
	esClient, err := infra.NewElasticSearchConnection(infra.ElasticsearchConfig{
		Addresses: appConfig.Elasticsearch.Addresses,
	})
	if err != nil {
		zapLogger.Fatal("failed to connect to elasticsearch", zap.Error(err))
	} else {
		zapLogger.Info("connected to elasticsearch successfully")
	}

	// set up dependencies
	serverRepo := repository.NewServerRepository(db)
	healthcheckRepo := repository.NewHealthCheckRepository(db, esClient)
	mailSender := mail.NewMailSender(appConfig.Mail.Email, appConfig.Mail.Password, appConfig.Mail.Host, appConfig.Mail.Port)
	serverService := service.NewServerService(serverRepo, healthcheckRepo, mailSender)
	serverHandler := handler.NewServerHandler(zapLogger, serverService)

	m := middleware.NewAuthMiddleware()

	// Create cronjob for daily report
	cronJob := cron.New()
	_, err = cronJob.AddFunc("0 0 * * *", func() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
		zapLogger.Info("cronjob called")
		e := serverService.ReportServersInformation(ctx2, time.Now().Add(-time.Hour*24), time.Now(), appConfig.Mail.AdminMailAddress)
		cancel2()
		if e != nil {
			zapLogger.Error("failed to generate daily report", zap.Error(e))
		}
	})
	if err != nil {
		zapLogger.Fatal("failed to create cron job for daily report", zap.Error(err))
	}
	cronJob.Start()

	// Set up http server
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	routes.SetUpServerRoutes(r, serverHandler, m)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", appConfig.Server.Port),
		Handler: r,
	}
	go func() {
		zapLogger.Info(fmt.Sprintf("starting server on %s", srv.Addr))
		if e := srv.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			zapLogger.Fatal("failed to start server", zap.Error(e))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zapLogger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = srv.Shutdown(ctx); err != nil {
		zapLogger.Error("server forced to shutdown:", zap.Error(err))
	}
	zapLogger.Info("server exiting")
}
