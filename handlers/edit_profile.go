package handlers

import (
	"SSE/auth"
	"SSE/database"
	"SSE/models"
	"SSE/sessions"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"html/template"
	"log"
	"net/http"
	"time"
)

func EditProfile(w http.ResponseWriter, r *http.Request) {
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

	switch r.Method {
	case http.MethodGet:
		var user models.User
		if err := collection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user); err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		tmpl, err := template.ParseFiles("templates/edit_profile.html")
		if err != nil {
			log.Printf("template parse error: %v", err)
			http.Error(w, "Failed to load template: "+err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			Name  string
			Email string
			ID    string
		}{
			Name:  user.Name,
			Email: user.Email,
			ID:    userID,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("template execute error: %v", err)
			http.Error(w, "Failed to render page: "+err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		email := r.FormValue("email")
		newPassword := r.FormValue("password")
		currentPassword := r.FormValue("current_password")

		updateFields := bson.M{}

		if name != "" {
			updateFields["name"] = name
		}
		if email != "" {
			updateFields["email"] = email
		}
		if newPassword != "" {
			if currentPassword == "" {
				http.Redirect(w, r, "/edit-profile?error=required_current_password", http.StatusSeeOther)
				return
			}
			var user models.User
			err := collection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
			if err != nil {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			if !user.CheckPassword(currentPassword) {
				http.Redirect(w, r, "/edit-profile?error=incorrect_password", http.StatusSeeOther)
				return
			}
			hashed, err := auth.HashPassword(newPassword)
			if err != nil {
				http.Error(w, "Failed to hash password", http.StatusInternalServerError)
				return
			}
			updateFields["password"] = hashed
		}

		if len(updateFields) == 0 {
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}

		updateFields["updated_at"] = time.Now()

		_, err := collection.UpdateOne(r.Context(), bson.M{"_id": objID}, bson.M{"$set": updateFields})
		if err != nil {
			http.Error(w, "Failed to update profile", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
