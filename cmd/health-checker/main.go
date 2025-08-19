package main

import (
	health_checker "VCS_SMS_Microservice/internal/health-checker"
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
	appConfig, err := health_checker.LoadConfig("./.env")
	if err != nil {
		log.Fatal(fmt.Sprintf("load config error: %v", err))
	}

	// set up logger
	fileSyncer, err := logger.NewReopenableWriteSyncer("./log/health-checker.log")
	zapLogger := logger.NewLogger(appConfig.Server.LogLevel, fileSyncer).With(zap.String("service.name", "health-checker"))
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

	serverClient := health_checker.NewServerClient(appConfig.Server.MaxRetries, appConfig.Server.RequestTimeout, appConfig.Server.InitialBackoff)
	kafkaWriter := infra.NewKafkaWriter(appConfig.Kafka.Brokers, appConfig.Kafka.ProducerTopic)
	defer kafkaWriter.Close()

	consumers := make([]health_checker.Consumer, appConfig.Kafka.ConsumerCnt)
	for i := 0; i < appConfig.Kafka.ConsumerCnt; i++ {
		consumers[i] = health_checker.NewConsumer(infra.NewKafkaReader(appConfig.Kafka.Brokers, appConfig.Kafka.ConsumerGroupID, appConfig.Kafka.ConsumerTopic),
			kafkaWriter, serverClient, zapLogger)
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
