package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email     string             `bson:"email" validate:"required,email"`
	Password  string             `bson:"password" validate:"required,min=6"`
	Role      string             `bson:"role" json:"-"`
	ImageURL  string             `bson:"image_url" json:"image_url"`
	CreatedAt time.Time          `bson:"created_at"`
}

func StringToObjectID(id string) primitive.ObjectID {
	oid, _ := primitive.ObjectIDFromHex(id)
	return oid
}
