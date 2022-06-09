package main

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/config"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/cron"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository"
	useCase "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/usecase"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

func runService(configData *config.Config) {
	ctx := context.Background()

	// Bootstrap DB
	newDBLogger := logger.New(
		log.StandardLogger(),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	postgresConfig := configData.Database.Postgres
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		postgresConfig.Host,
		postgresConfig.Username,
		postgresConfig.Password,
		postgresConfig.DBName,
		postgresConfig.Port,
	)
	db, err := gorm.Open(
		postgres.Open(dsn), &gorm.Config{
			Logger: newDBLogger,
		},
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	dbSQL, err := db.DB()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = dbSQL.Ping()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer dbSQL.Close()

	err = repository.Migrate(db)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Bootsrap Redis
	redisClient := redis.NewClient(
		&redis.Options{
			Addr: fmt.Sprintf(
				"%s:%s",
				configData.Database.Redis.Host,
				configData.Database.Redis.Port,
			),
			Password: configData.Database.Redis.Password,
			DB:       0,
		},
	)
	if status := redisClient.Ping(ctx); status.Err() != nil {
		log.Fatal(status.Err().Error())
	}
	defer redisClient.Close()

	//Bootstrap Validator
	validatorInst := validator.New()

	// Boostrap Dependencies
	resources := &config.KubeEPResources{
		DB:            db,
		ValidatorInst: validatorInst,
		Redis:         redisClient,
	}

	repositories := repository.BuildRepositories(resources)
	useCases := useCase.BuildUseCases(resources, repositories)
	cronInst := cron.BuildCron(useCases, resources)
	cronInst.Start()
}
