package main

import (
	"VCS_SMS_Microservice/internal/scheduler/config"
	"VCS_SMS_Microservice/internal/scheduler/consumer"
	"VCS_SMS_Microservice/internal/scheduler/repository"
	"VCS_SMS_Microservice/internal/scheduler/scheduler"
	"VCS_SMS_Microservice/pkg/infra"
	"VCS_SMS_Microservice/pkg/logger"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	appConfig, err := config.LoadConfig("./.env")
	if err != nil {
		log.Fatal(fmt.Sprintf("load config error: %v", err))
	}

	// set up logger
	fileSyncer, err := logger.NewReopenableWriteSyncer("./log/server-service.log")
	zapLogger := logger.NewLogger(appConfig.Server.LogLevel, fileSyncer)
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

	serverRepo := repository.NewServerRepository(db)

	tw := scheduler.NewTimeWheel(3600, 500, 10, zapLogger, serverRepo, infra.NewKafkaWriter(appConfig.Kafka.Brokers, appConfig.Kafka.ProducerTopic))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	servers, err := serverRepo.GetAllServers(ctx)
	cancel()
	if err != nil {
		zapLogger.Fatal("failed to get all servers for schedule", zap.Error(err))
	}
	for _, server := range servers {
		tw.AddServer(server)
	}

	consumers := make([]consumer.ServerConsumer, appConfig.Kafka.ConsumerCnt)
	for i := 0; i < appConfig.Kafka.ConsumerCnt; i++ {
		consumers[i] = consumer.NewServerConsumer(serverRepo, tw, zapLogger, infra.NewKafkaReader(appConfig.Kafka.Brokers, appConfig.Kafka.ConsumerGroupID, appConfig.Kafka.ConsumerTopic))
		consumers[i].Start()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zapLogger.Info("shutting down server...")
	for i := 0; i < appConfig.Kafka.ConsumerCnt; i++ {
		consumers[i].Stop()
	}
	tw.Stop()
	zapLogger.Info("server exiting")
}
