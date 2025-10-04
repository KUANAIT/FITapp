package handlers

import (
	"SSE/database"
	"SSE/models"
	"SSE/sessions"
	"bytes"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"html/template"
	"log"
	"net/http"
)

func Profile(w http.ResponseWriter, r *http.Request) {
	session, err := sessions.Get(r)
	if err != nil {
		http.Error(w, "Failed to retrieve session", http.StatusInternalServerError)
		return
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusInternalServerError)
		return
	}

	collection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Error(w, "Failed to get database collection", http.StatusInternalServerError)
		return
	}

	var user models.User
	err = collection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var membershipPlan models.MembershipPlan
	if !user.MembershipPlanID.IsZero() {
		planCollection, err := database.GetCollection("SSE", "membership_plans")
		if err == nil {
			planCollection.FindOne(context.TODO(), bson.M{"_id": user.MembershipPlanID}).Decode(&membershipPlan)
		}
	}

	tmpl, err := template.ParseFiles("templates/profile.html")
	if err != nil {
		log.Printf("template parse error: %v", err)
		http.Error(w, "Failed to load template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Name             string
		Email            string
		ID               string
		MemberID         string
		MembershipStatus string
		MembershipPlan   string
		MembershipPrice  float64
		MembershipExpiry string
		JoinDate         string
		TotalVisits      int
		DaysUntilExpiry  int
		IsActiveMember   bool
		HasMembership    bool
		CreatedAt        string
		UpdatedAt        string
	}{
		Name:             user.Name,
		Email:            user.Email,
		ID:               userID,
		MemberID:         user.MemberID,
		MembershipStatus: string(user.MembershipStatus),
		MembershipPlan:   membershipPlan.Name,
		MembershipPrice:  membershipPlan.Price,
		MembershipExpiry: user.MembershipExpiry.Format("January 2, 2006"),
		JoinDate:         user.JoinDate.Format("January 2, 2006"),
		TotalVisits:      user.TotalVisits,
		DaysUntilExpiry:  user.DaysUntilExpiry(),
		IsActiveMember:   user.IsActiveMember(),
		HasMembership:    !user.MembershipPlanID.IsZero(),
		CreatedAt:        user.CreatedAt.Format("January 2, 2006 15:04:05"),
		UpdatedAt:        user.UpdatedAt.Format("January 2, 2006 15:04:05"),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("template execute error: %v", err)
		http.Error(w, "Failed to render page: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
