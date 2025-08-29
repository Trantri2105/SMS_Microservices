package main

import (
	"VCS_SMS_Microservice/internal/auth-service/api/handler"
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"
	"VCS_SMS_Microservice/internal/auth-service/api/routes"
	"VCS_SMS_Microservice/internal/auth-service/config"
	"VCS_SMS_Microservice/internal/auth-service/jwt"
	"VCS_SMS_Microservice/internal/auth-service/repository"
	"VCS_SMS_Microservice/internal/auth-service/service"
	"VCS_SMS_Microservice/pkg/infra"
	"VCS_SMS_Microservice/pkg/logger"
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
	"go.uber.org/zap"
)

func main() {
	appConfig, err := config.LoadConfig("./.env")
	if err != nil {
		log.Fatal(fmt.Sprintf("load config error: %v", err))
	}

	// set up logger
	fileSyncer, err := logger.NewReopenableWriteSyncer("./log/auth-service.log")
	zapLogger := logger.NewLogger(appConfig.Server.LogLevel, fileSyncer).With(zap.String("service.name", "auth-service"))
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
	db = db.Debug()
	sqlDB, err := db.DB()
	if err != nil {
		zapLogger.Fatal("failed to get sql.DB from gorm:", zap.Error(err))
	}
	defer sqlDB.Close()

	// set up redis
	redisClient, err := infra.NewRedisConnection(infra.RedisConfig{
		Host: appConfig.Redis.Host,
		Port: appConfig.Redis.Port,
	})
	if err != nil {
		zapLogger.Fatal("failed to connect to redis", zap.Error(err))
	} else {
		zapLogger.Info("connected to redis successfully")
	}
	defer redisClient.Close()

	tokenRepo := repository.NewRefreshTokenRepository(redisClient)
	roleRepo := repository.NewRoleRepository(db)
	scopeRepo := repository.NewScopeRepository(db)
	userRepo := repository.NewUserRepository(db)

	jwtUtils := jwt.NewJwtUtils(appConfig.JWT.SecretKey, appConfig.JWT.AccessTokenTTL, appConfig.JWT.RefreshTokenTTL)

	scopeService := service.NewScopeService(scopeRepo)
	roleService := service.NewRoleService(roleRepo, scopeService)
	userService := service.NewUserService(userRepo, roleService)
	authService := service.NewAuthService(userService, jwtUtils, tokenRepo, appConfig.Server.UserSessionTTL)

	m := middleware.NewAuthMiddleware(jwtUtils)

	handlerLogger := handler.NewLogger(zapLogger)
	scopeHandler := handler.NewScopeHandler(scopeService, handlerLogger)
	roleHandler := handler.NewRoleHandler(roleService, handlerLogger)
	userHandler := handler.NewUserHandler(userService, handlerLogger)
	authHandler := handler.NewAuthHandler(authService, handlerLogger)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	routes.SetUpScopeRoutes(r, scopeHandler, m)
	routes.SetUpRoleRoutes(r, roleHandler, m)
	routes.SetUpUserRoutes(r, userHandler, m)
	routes.SetUpAuthRoutes(r, authHandler, m)

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
