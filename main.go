package main

import (
	"log"

	"github.com/Sadhvikbaba/go-todo/database"
	"github.com/Sadhvikbaba/go-todo/handlers"
	"github.com/Sadhvikbaba/go-todo/redis"
	"github.com/Sadhvikbaba/go-todo/todocontrollers"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Connect to MongoDB
	database.ConnectMongo()

	// Connect to Redis
	redis.ConnectRedis()

	// Initialize Fiber app
	app := fiber.New()

	// Define routes
	app.Post("/api/signup", handlers.Signup)
	app.Post("/api/login", handlers.Login)

	// Set up Todo routes
	todocontrollers.SetupTodoRoutes(app)

	// Start the app
	log.Fatal(app.Listen(":3000"))
}
