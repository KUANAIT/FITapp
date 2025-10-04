package handlers

import (
	"SSE/database"
	"SSE/models"
	"SSE/sessions"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"html/template"
	"net/http"
)

func HomePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	session, err := sessions.Get(r)
	if err != nil {
		http.Error(w, "Failed to retrieve session", http.StatusInternalServerError)
		return
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		data := struct {
			Authenticated bool
			UserName      string
		}{
			Authenticated: false,
			UserName:      "",
		}
		tmpl.Execute(w, data)
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

	_, isAuthenticated := session.Values["authenticated"].(bool)

	data := struct {
		Authenticated bool
		UserName      string
	}{
		Authenticated: isAuthenticated,
		UserName:      user.Name,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}
