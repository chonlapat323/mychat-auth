package handlers

import (
	"context"
	"encoding/json"
	"mychat-auth/database"
	"mychat-auth/models"
	"mychat-auth/shared/contextkey"
	"mychat-auth/utils"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GET /rooms
func GetRoomsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := database.RoomCollection.Find(ctx, bson.M{}) //  ใช้ collection จาก database package
	if err != nil {
		http.Error(w, "Failed to fetch rooms", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var rooms []models.Room
	if err := cursor.All(ctx, &rooms); err != nil {
		http.Error(w, "Failed to decode rooms", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rooms)
}

func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := utils.ValidateToken(tokenStr)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	if claims.Role != "admin" {
		http.Error(w, "Forbidden: admin only", http.StatusForbidden)
		return
	}

	var req models.Room
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || (req.Type != "public" && req.Type != "private") {
		http.Error(w, "Invalid room data", http.StatusBadRequest)
		return
	}

	count, _ := database.RoomCollection.CountDocuments(context.TODO(), bson.M{"name": req.Name})
	if count > 0 {
		http.Error(w, "Room name already exists", http.StatusConflict)
		return
	}

	req.ID = primitive.NewObjectID()
	req.CreatedAt = time.Now()
	if req.Type == "public" {
		req.Members = []primitive.ObjectID{}
	}

	_, err = database.RoomCollection.InsertOne(context.TODO(), req)
	if err != nil {
		http.Error(w, "Failed to create room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/rooms/")
	roomID = strings.TrimSuffix(roomID, "/join")

	userID, ok := r.Context().Value(contextkey.UserID).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roomObjID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	userObjID := models.StringToObjectID(userID)

	filter := bson.M{
		"_id":     roomObjID,
		"type":    "private",
		"members": bson.M{"$ne": userObjID},
	}

	update := bson.M{
		"$push": bson.M{"members": userObjID},
	}

	res, err := database.RoomCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil || res.ModifiedCount == 0 {
		http.Error(w, "Unable to join room", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Joined room"})
}
