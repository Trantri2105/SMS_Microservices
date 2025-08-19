package main

import (
	health_check_consumer "VCS_SMS_Microservice/internal/health-check-consumer"
	"VCS_SMS_Microservice/internal/server-service/repository"
	"VCS_SMS_Microservice/pkg/infra"
	"VCS_SMS_Microservice/pkg/logger"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	appConfig, err := health_check_consumer.LoadConfig("./.env")
	if err != nil {
		log.Fatal(fmt.Sprintf("load config error: %v", err))
	}

	// set up logger
	fileSyncer, err := logger.NewReopenableWriteSyncer("./log/health-check-consumer.log")
	zapLogger := logger.NewLogger(appConfig.Server.LogLevel, fileSyncer).With(zap.String("service.name", "health-check-consumer"))
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

	consumers := make([]health_check_consumer.HealthCheckConsumer, appConfig.Kafka.ConsumerCnt)
	for i := 0; i < appConfig.Kafka.ConsumerCnt; i++ {
		consumers[i] = health_check_consumer.NewHealthCheckConsumer(infra.NewKafkaReader(appConfig.Kafka.Brokers, appConfig.Kafka.ConsumerGroupID, appConfig.Kafka.ConsumerTopic), serverRepo, zapLogger)
		consumers[i].Start()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zapLogger.Info("shutting down server...")
	for i := 0; i < appConfig.Kafka.ConsumerCnt; i++ {
		consumers[i].Stop()
	}
	zapLogger.Info("server exiting")
}
