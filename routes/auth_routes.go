package routes

import (
	"SSE/handlers"
	"SSE/middleware"
	"net/http"
)

func RegisterAuthRoutes() {
	logoutHandler := middleware.AuthRequired(http.HandlerFunc(handlers.LogoutCustomer))
	http.Handle("/logout", logoutHandler)
}
