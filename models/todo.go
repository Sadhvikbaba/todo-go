package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Todo struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"userId,omitempty" json:"userId"` // To associate the todo with a specific user
	Title       string             `bson:"title" json:"title"`             // Title of the todo/task
	Description string             `bson:"description" json:"description"` // Description of the todo/task
	IsCompleted bool               `bson:"isCompleted" json:"isCompleted"` // Whether the todo is completed
	Context     string             `bson:"context" json:"context"`         // Context of the todo/task
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`     // When the todo was created
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`     // When the todo was last modified
}
