package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var UserCollection *mongo.Collection
var RoomCollection *mongo.Collection
var MessageCollection *mongo.Collection

func InitMongo() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	Client, err = mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URL")))
	if err != nil {
		log.Fatal("❌ MongoDB connection error:", err)
	}

	db := Client.Database("mychat")
	UserCollection = db.Collection("users")
	RoomCollection = db.Collection("rooms")
	MessageCollection = db.Collection("messages")

	log.Println("✅ Connected to MongoDB and initialized collections")
}
