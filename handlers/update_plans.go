package handlers

import (
	"SSE/database"
	"SSE/models"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

func UpdateMembershipPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	collection, err := database.GetCollection("SSE", "membership_plans")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	_, err = collection.DeleteMany(r.Context(), bson.M{})
	if err != nil {
		http.Error(w, "Failed to delete existing plans", http.StatusInternalServerError)
		return
	}

	defaultPlans := models.GetDefaultPlans()
	var interfaces []interface{}
	for _, plan := range defaultPlans {
		interfaces = append(interfaces, plan)
	}

	_, err = collection.InsertMany(r.Context(), interfaces)
	if err != nil {
		http.Error(w, "Failed to create new plans", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Membership plans updated successfully to Tenge prices"})
}
