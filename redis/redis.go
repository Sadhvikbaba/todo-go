package redis

import (
	"context"
	"crypto/tls"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var Ctx = context.Background()

func ConnectRedis() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	// Set TLS config BEFORE creating the client!
	opt.TLSConfig = &tls.Config{
		InsecureSkipVerify: true, // Only for dev; see note above
	}

	RedisClient = redis.NewClient(opt)

	log.Println("✅ Redis connected successfully")
}
