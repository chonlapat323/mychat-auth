package types

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SafeUser struct {
	ID        primitive.ObjectID `json:"id"`
	Email     string             `json:"email"`
	CreatedAt time.Time          `json:"created_at"`
}
