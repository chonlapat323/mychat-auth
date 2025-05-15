package utils

import (
	"context"
	"log"
	"time"

	"mychat-auth/database"
	"mychat-auth/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SeedAdminUser() {
	adminEmail := "admin@admin.com"
	adminRole := "admin"
	adminPassword := "123123"
	adminAvatar := "https://cdn.example.com/avatars/admin.png"

	// ตรวจสอบว่ามี admin อยู่แล้วหรือยัง
	var existing models.User
	err := database.UserCollection.FindOne(context.TODO(), bson.M{"email": adminEmail}).Decode(&existing)
	if err == nil {
		log.Println("Admin already exists, skip seeding")
		return
	}

	hashed, _ := HashPassword(adminPassword)
	newAdmin := models.User{
		ID:        primitive.NewObjectID(),
		Email:     adminEmail,
		Password:  hashed,
		Role:      adminRole,
		ImageURL:  adminAvatar,
		CreatedAt: time.Now(),
	}

	_, err = database.UserCollection.InsertOne(context.TODO(), newAdmin)
	if err != nil {
		log.Println("Failed to seed admin:", err)
	} else {
		log.Println("Admin user seeded successfully")
	}
}

func SeedRoom() {
	var existing models.Room
	err := database.RoomCollection.FindOne(context.TODO(), bson.M{"name": "general"}).Decode(&existing)
	if err == nil {
		log.Println("Room 'general' already exists, skip seeding")
		return
	}

	room := models.Room{
		Name:      "general",
		Type:      "public",
		Members:   []models.SafeUser{},
		CreatedAt: time.Now(),
	}

	_, err = database.RoomCollection.InsertOne(context.TODO(), room)
	if err != nil {
		log.Println("Failed to seed room:", err)
	} else {
		log.Println("Room 'general' seeded successfully")
	}
}
