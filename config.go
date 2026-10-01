package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	RedisAddr   string
}

func LoadConfig() (Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using environment variables")
	}

	jwtSecret, ok := os.LookupEnv("JWT_SECRET")
	if !ok || jwtSecret == "" {
		log.Println("JWT_SECRET is not set")
		return Config{}, fmt.Errorf("JWT_SECRET is not set")
	}
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok || databaseUrl == "" {
		log.Println("DATABASE_URL is not set")
		return Config{}, fmt.Errorf("DATABASE_URL is not set")
	}
	redisAddr, ok := os.LookupEnv("REDIS_ADDR")
	if !ok || redisAddr == "" {
		log.Println("REDIS_ADDR is not set")
		return Config{}, fmt.Errorf("REDIS_ADDR is not set")
	}
	return Config{
		DatabaseURL: databaseUrl,
		JWTSecret:   jwtSecret,
		RedisAddr:   redisAddr,
	}, nil

}
