package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type KubeEPResources struct {
	DB            *gorm.DB
	ValidatorInst *validator.Validate
	Redis         *redis.Client
}
