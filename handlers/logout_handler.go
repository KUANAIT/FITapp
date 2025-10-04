package handlers

import (
	"SSE/sessions"
	"net/http"
)

func LogoutCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := sessions.ClearSession(w, r)
	if err != nil {
		http.Error(w, "Failed to clear session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	http.Redirect(w, r, "/", http.StatusSeeOther)

}
