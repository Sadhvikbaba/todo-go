package todocontrollers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sadhvikbaba/go-todo/database"
	"github.com/Sadhvikbaba/go-todo/middleware"
	"github.com/Sadhvikbaba/go-todo/models"
	"github.com/Sadhvikbaba/go-todo/redis"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SetupTodoRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Protected todo routes
	todos := api.Group("/todos", middleware.RequireAuth)

	todos.Post("/", CreateTodo)
	todos.Get("/", GetTodos)
	todos.Put("/:id", UpdateTodo)
	todos.Patch("/toggle/:id", ToggleCompleteTodo)
	todos.Delete("/:id", DeleteTodo)
}

func CreateTodo(c *fiber.Ctx) error {
	// Parse request body into Todo struct
	var todo models.Todo
	if err := c.BodyParser(&todo); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid todo format"})
	}

	// Extract user ID from JWT
	userClaims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	// Check if _id exists in claims and if it's a string
	userIDStr, ok := userClaims["_id"].(string)
	if !ok || userIDStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "User ID not found in token"})
	}

	// Convert to ObjectID
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Prepare todo object
	todo.ID = primitive.NewObjectID()
	todo.UserID = userID
	todo.CreatedAt = time.Now()
	todo.UpdatedAt = time.Now()

	// Insert into DB
	_, err = database.DB.Collection("todos").InsertOne(context.TODO(), todo)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create todo"})
	}

	// Return success response
	return c.JSON(fiber.Map{"message": "Todo created successfully", "todo": todo})
}

func GetTodos(c *fiber.Ctx) error {
	userClaims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	userIDStr, ok := userClaims["_id"].(string)
	if !ok || userIDStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "User ID not found in token"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	cacheKey := fmt.Sprintf("todos:%s", userIDStr)

	cachedTodos, err := redis.RedisClient.Get(redis.Ctx, cacheKey).Result()
	if err == nil {
		var todos []models.Todo
		if err := json.Unmarshal([]byte(cachedTodos), &todos); err == nil {
			return c.JSON(fiber.Map{"todos": todos, "cached": true})
		}
	}

	var todos []models.Todo
	cursor, err := database.DB.Collection("todos").Find(context.TODO(), bson.M{"userId": userID})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve todos"})
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var todo models.Todo
		if err := cursor.Decode(&todo); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to decode todo"})
		}
		todos = append(todos, todo)
	}
	if err := cursor.Err(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Cursor error"})
	}

	todoBytes, err := json.Marshal(todos)
	if err == nil {
		redis.RedisClient.Set(redis.Ctx, cacheKey, todoBytes, 5*time.Minute)
	}

	return c.JSON(fiber.Map{"todos": todos, "cached": false})
}

func UpdateTodo(c *fiber.Ctx) error {
	todoIDParam := c.Params("id")
	if todoIDParam == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Todo ID is required"})
	}

	todoID, err := primitive.ObjectIDFromHex(todoIDParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid Todo ID"})
	}

	// Parse only updatable fields (not ID or UserID)
	var updatedData struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&updatedData); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid todo format"})
	}

	// Get user ID from JWT
	userClaims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid token claims"})
	}
	userIDStr, ok := userClaims["_id"].(string)
	if !ok || userIDStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "User ID not found in token"})
	}

	cacheKey := "todos:" + userIDStr

	// Update Redis cache if exists
	cachedTodos, err := redis.RedisClient.Get(redis.Ctx, cacheKey).Result()
	if err == nil {
		var todos []models.Todo
		if err := json.Unmarshal([]byte(cachedTodos), &todos); err == nil {
			for i, t := range todos {
				if t.ID == todoID {
					todos[i].Title = updatedData.Title
					todos[i].Description = updatedData.Description
					todos[i].UpdatedAt = time.Now()
					break
				}
			}
			todoBytes, _ := json.Marshal(todos)
			redis.RedisClient.Set(redis.Ctx, cacheKey, todoBytes, 5*time.Minute)
		}
	}

	// Update MongoDB
	filter := bson.M{"_id": todoID, "userId": userIDStr}
	update := bson.M{
		"$set": bson.M{
			"title":       updatedData.Title,
			"description": updatedData.Description,
			"updatedAt":   time.Now(),
		},
	}

	_, err = database.DB.Collection("todos").UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update todo in MongoDB"})
	}

	return c.JSON(fiber.Map{"message": "Todo updated successfully"})
}

func ToggleCompleteTodo(c *fiber.Ctx) error {
	todoIDParam := c.Params("id")
	if todoIDParam == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Todo ID is required"})
	}

	todoID, err := primitive.ObjectIDFromHex(todoIDParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid Todo ID"})
	}

	// Extract user ID from JWT
	userClaims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid token claims"})
	}
	userIDStr, ok := userClaims["_id"].(string)
	if !ok || userIDStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "User ID not found in token"})
	}

	cacheKey := "todos:" + userIDStr
	var newIsCompleted bool

	// Check Redis cache
	cachedTodos, err := redis.RedisClient.Get(redis.Ctx, cacheKey).Result()
	if err == nil {
		var todos []models.Todo
		if err := json.Unmarshal([]byte(cachedTodos), &todos); err == nil {
			for i, t := range todos {
				if t.ID == todoID {
					todos[i].IsCompleted = !t.IsCompleted
					newIsCompleted = todos[i].IsCompleted
					break
				}
			}
			todoBytes, _ := json.Marshal(todos)
			redis.RedisClient.Set(redis.Ctx, cacheKey, todoBytes, 5*time.Minute)
		}
	}

	// Update in MongoDB
	filter := bson.M{"_id": todoID, "userId": userIDStr}
	update := bson.M{
		"$set": bson.M{
			"isCompleted": newIsCompleted,
			"updatedAt":   time.Now(),
		},
	}

	_, err = database.DB.Collection("todos").UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update todo in MongoDB"})
	}

	return c.JSON(fiber.Map{"message": "Todo completion status toggled", "isCompleted": newIsCompleted})
}

func DeleteTodo(c *fiber.Ctx) error {
	// Extract todo ID from params
	todoIDStr := c.Params("id")
	todoID, err := primitive.ObjectIDFromHex(todoIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid todo ID"})
	}

	// Extract user ID from JWT claims
	userClaims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	userIDStr, ok := userClaims["_id"].(string)
	if !ok || userIDStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "User ID not found in token"})
	}

	// MongoDB delete filter
	filter := bson.M{"_id": todoID, "userId": userIDStr}
	_, err = database.DB.Collection("todos").DeleteOne(context.TODO(), filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete todo from MongoDB"})
	}

	// Redis cache key
	cacheKey := fmt.Sprintf("todos:%s", userIDStr)

	// Check if the user's todo list is cached
	cachedTodos, err := redis.RedisClient.Get(redis.Ctx, cacheKey).Result()
	if err == nil {
		var todos []models.Todo
		if err := json.Unmarshal([]byte(cachedTodos), &todos); err == nil {
			// Remove the specific todo from the cached list
			for i, t := range todos {
				if t.ID == todoID {
					// Remove the todo from the slice
					todos = append(todos[:i], todos[i+1:]...)
					break
				}
			}

			// Update the Redis cache with the modified todo list
			updatedTodosBytes, err := json.Marshal(todos)
			if err == nil {
				redis.RedisClient.Set(redis.Ctx, cacheKey, updatedTodosBytes, 5*time.Minute)
			}
		}
	}

	return c.JSON(fiber.Map{"message": "Todo deleted successfully"})
}
