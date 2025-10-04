package handlers

import (
	"SSE/database"
	"SSE/models"
	"SSE/sessions"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

func LoginCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	collection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	var user models.User
	err = collection.FindOne(r.Context(), bson.M{"email": credentials.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if !user.CheckPassword(credentials.Password) {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	userID := user.ID.Hex()

	if err := sessions.SetUserSession(w, r, userID); err != nil {
		http.Error(w, "Failed to set session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}
