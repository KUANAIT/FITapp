package handlers

import (
	"SSE/database"
	"SSE/models"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"
)

func CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var userData struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if userData.Name == "" || userData.Email == "" || userData.Password == "" {
		http.Error(w, "Name, Email, and Password are required fields", http.StatusBadRequest)
		return
	}

	user := models.User{
		Name:             userData.Name,
		Email:            userData.Email,
		Password:         userData.Password,
		MembershipStatus: models.StatusActive,
		JoinDate:         time.Now(),
		TotalVisits:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	user.GenerateMemberID()

	if err := user.HashPassword(); err != nil {
		http.Error(w, "Failed to hash password: "+err.Error(), http.StatusInternalServerError)
		return
	}

	collection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Error(w, "Failed to get database collection: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var existingUser models.User
	err = collection.FindOne(r.Context(), bson.M{"email": user.Email}).Decode(&existingUser)
	if err == nil {
		http.Error(w, "User with this email already exists", http.StatusConflict)
		return
	}

	result, err := collection.InsertOne(r.Context(), user)
	if err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	user.ID = result.InsertedID.(primitive.ObjectID)

	responseUser := struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		ID:        user.ID.Hex(),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(responseUser)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid customer ID format", http.StatusBadRequest)
		return
	}

	collection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	var user models.User
	err = collection.FindOne(r.Context(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error retrieving customer", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	var updateData struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updateFields := bson.M{}

	if updateData.Name != "" {
		updateFields["name"] = updateData.Name
	}

	if updateData.Email != "" {
		updateFields["email"] = updateData.Email
	}

	if updateData.Password != "" {
		tempUser := models.User{Password: updateData.Password}
		if err := tempUser.HashPassword(); err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		updateFields["password"] = tempUser.Password
	}

	if len(updateFields) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	updateFields["updatedat"] = time.Now()

	collection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	result, err := collection.UpdateOne(r.Context(), bson.M{"_id": objectID}, bson.M{"$set": updateFields})
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User updated successfully"})
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid customer ID format", http.StatusBadRequest)
		return
	}

	collection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	result, err := collection.DeleteOne(r.Context(), bson.M{"_id": objectID})
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User deleted successfully"})
}
