package handlers

import (
	"SSE/sessions"
	"html/template"
	"log"
	"net/http"
)

func MembershipPlansPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	session, err := sessions.Get(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/membership_plans.html")
	if err != nil {
		log.Printf("template parse error: %v", err)
		http.Error(w, "Failed to load template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		UserID string
	}{
		UserID: userID,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template execute error: %v", err)
		http.Error(w, "Failed to render page: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
