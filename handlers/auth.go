package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"mychat-auth/database"
	"mychat-auth/models"
	"mychat-auth/shared/contextkey"
	"mychat-auth/types"
	"mychat-auth/utils"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var validate = validator.New()

// RegisterHandler รับ POST /register
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// validate email/password
	if err := validate.Struct(req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	// check duplicate email
	count, _ := database.UserCollection.CountDocuments(context.TODO(), bson.M{"email": req.Email})
	if count > 0 {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// hash password
	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Hash error", http.StatusInternalServerError)
		return
	}
	req.Password = hashedPwd
	req.CreatedAt = time.Now()

	// insert user
	res, err := database.UserCollection.InsertOne(context.TODO(), req)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// ดึง ObjectID กลับมา แล้วแปลงเป็น string
	userID := res.InsertedID.(primitive.ObjectID).Hex()

	// generate token
	token, err := utils.GenerateToken(userID, req.Email, req.Role)
	if err != nil {
		http.Error(w, "Token error", http.StatusInternalServerError)
		return
	}

	// return token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var user models.User
	err := database.UserCollection.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if !utils.CheckPassword(req.Password, user.Password) {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := utils.GenerateToken(user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	err := database.UserCollection.FindOne(context.TODO(), bson.M{"_id": models.StringToObjectID(userID)}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.Password = "" // ซ่อน password

	safeUser := types.SafeUser{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	json.NewEncoder(w).Encode(safeUser)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := utils.ValidateToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	if err := utils.BlacklistToken(token, claims.ExpiresAt.Time); err != nil {
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}
