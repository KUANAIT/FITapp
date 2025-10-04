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

func GetMembershipPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	collection, err := database.GetCollection("SSE", "membership_plans")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	var plans []models.MembershipPlan
	cursor, err := collection.Find(r.Context(), bson.M{"is_active": true})
	if err != nil {
		http.Error(w, "Failed to fetch membership plans", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	if err = cursor.All(r.Context(), &plans); err != nil {
		http.Error(w, "Failed to decode membership plans", http.StatusInternalServerError)
		return
	}

	if len(plans) == 0 {
		defaultPlans := models.GetDefaultPlans()
		var interfaces []interface{}
		for _, plan := range defaultPlans {
			interfaces = append(interfaces, plan)
		}

		_, err = collection.InsertMany(r.Context(), interfaces)
		if err != nil {
			http.Error(w, "Failed to create default plans", http.StatusInternalServerError)
			return
		}
		plans = defaultPlans
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plans)
}

func SelectMembershipPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		UserID           string `json:"user_id"`
		MembershipPlanID string `json:"membership_plan_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(requestData.UserID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	planObjID, err := primitive.ObjectIDFromHex(requestData.MembershipPlanID)
	if err != nil {
		http.Error(w, "Invalid membership plan ID", http.StatusBadRequest)
		return
	}

	planCollection, err := database.GetCollection("SSE", "membership_plans")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	var plan models.MembershipPlan
	err = planCollection.FindOne(r.Context(), bson.M{"_id": planObjID, "is_active": true}).Decode(&plan)
	if err != nil {
		http.Error(w, "Membership plan not found", http.StatusNotFound)
		return
	}

	userCollection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	expiryDate := now.AddDate(0, plan.Duration, 0)

	updateFields := bson.M{
		"membership_plan_id": planObjID,
		"membership_status":  models.StatusActive,
		"membership_expiry":  expiryDate,
		"join_date":          now,
		"updated_at":         now,
	}

	result, err := userCollection.UpdateOne(r.Context(), bson.M{"_id": userObjID}, bson.M{"$set": updateFields})
	if err != nil {
		http.Error(w, "Failed to update user membership", http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Membership plan selected successfully",
		"plan_name":   plan.Name,
		"expiry_date": expiryDate.Format("2006-01-02"),
	})
}

func GetUserMembership(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	userCollection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	var user models.User
	err = userCollection.FindOne(r.Context(), bson.M{"_id": userObjID}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var plan models.MembershipPlan
	if !user.MembershipPlanID.IsZero() {
		planCollection, err := database.GetCollection("SSE", "membership_plans")
		if err == nil {
			planCollection.FindOne(r.Context(), bson.M{"_id": user.MembershipPlanID}).Decode(&plan)
		}
	}

	membershipInfo := map[string]interface{}{
		"member_id":         user.MemberID,
		"membership_status": user.MembershipStatus,
		"membership_expiry": user.MembershipExpiry.Format("2006-01-02"),
		"join_date":         user.JoinDate.Format("2006-01-02"),
		"total_visits":      user.TotalVisits,
		"days_until_expiry": user.DaysUntilExpiry(),
		"is_active":         user.IsActiveMember(),
		"plan":              plan,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(membershipInfo)
}
