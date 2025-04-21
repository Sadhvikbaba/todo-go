package main

import (
	"log"

	"github.com/Sadhvikbaba/todo-go/database"
	"github.com/Sadhvikbaba/todo-go/handlers"
	"github.com/Sadhvikbaba/todo-go/redis"
	"github.com/Sadhvikbaba/todo-go/todocontrollers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "https://todo-frontend-beta-three.vercel.app",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	// Define routes
	app.Post("/api/signup", handlers.Signup)
	app.Post("/api/login", handlers.Login)

	// Set up Todo routes
	todocontrollers.SetupTodoRoutes(app)

	// Start the app
	log.Fatal(app.Listen(":3000"))
}
