package handlers

import (
	"context"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"

	"github.com/Sadhvikbaba/todo-go/database"
	"github.com/Sadhvikbaba/todo-go/models"
)

func Signup(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Check if user already exists
	var existingUser models.User
	err := database.DB.Collection("users").FindOne(context.TODO(), bson.M{"email": user.Email}).Decode(&existingUser)
	if err == nil {
		return c.Status(400).JSON(fiber.Map{"error": "User already registered"})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to hash password"})
	}
	user.Password = string(hashedPassword)

	// Insert user
	_, err = database.DB.Collection("users").InsertOne(context.TODO(), user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "User creation failed"})
	}

	return c.JSON(fiber.Map{"message": "User created"})
}

func Login(c *fiber.Ctx) error {
	var body models.User
	// Parse the request body into the User struct
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
	}

	var user models.User
	// Find the user by email in the database
	err := database.DB.Collection("users").FindOne(context.TODO(), bson.M{"email": body.Email}).Decode(&user)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "User not found"})
	}

	// Compare the password from the request body with the stored password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Wrong password"})
	}

	// Generate JWT token with the user's _id and email
	claims := jwt.MapClaims{
		"_id":   user.ID.Hex(),                         // Convert ObjectID to string and add to claims
		"email": user.Email,                            // Optionally include the user's email
		"exp":   time.Now().Add(time.Hour * 24).Unix(), // Set expiration time (1 day)
	}

	// Create a new JWT token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtToken, err := token.SignedString([]byte(os.Getenv("JWT_SECRET"))) // Sign the token
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Token generation failed"})
	}

	// Return the generated token in the response
	return c.JSON(fiber.Map{
		"token": jwtToken,
	})
}
