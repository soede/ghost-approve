package redis

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"strconv"
)

type Store struct {
	client *redis.Client
	ctx    context.Context
}

var rdb *Store

func InitRedis(ctx context.Context) *Store {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		log.Fatalf("Ошибка преобразования REDIS_DB в число: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       redisDB,
	})

	_, err = client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Ошибка преобразования REDIS_DB в число: %v", err)
	}
	rdb = &Store{client, ctx}

	log.Println("Connected to Redis successfully")
	return rdb
}
