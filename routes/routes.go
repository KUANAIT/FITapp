package routes

import (
	"SSE/handlers"
	"SSE/middleware"
	"net/http"
)

func RegisterRoutes() {
	http.HandleFunc("/", handlers.HomePage)
	http.HandleFunc("/users", handlers.CreateUser)
	http.HandleFunc("/users/get", handlers.GetUser)
	http.HandleFunc("/users/update", handlers.UpdateUser)
	http.HandleFunc("/users/delete", handlers.DeleteUser)
	http.HandleFunc("/loginuser", handlers.LoginCustomer)
	http.HandleFunc("/profile", handlers.Profile)
	http.HandleFunc("/edit-profile", middleware.AuthRequired(handlers.EditProfile))

	http.HandleFunc("/membership/plans", handlers.GetMembershipPlans)
	http.HandleFunc("/membership/select", handlers.SelectMembershipPlan)
	http.HandleFunc("/membership/user", handlers.GetUserMembership)
	http.HandleFunc("/membership", handlers.MembershipPlansPage)
	http.HandleFunc("/membership/update-plans", handlers.UpdateMembershipPlans)
	http.HandleFunc("/fitness-chat", handlers.FitnessChatPageHandler)
	http.HandleFunc("/ask-fitness", handlers.AskFitnessHandler)

	http.HandleFunc("/fitness-tracker", middleware.AuthRequired(handlers.FitnessTrackerPageHandler))
	http.HandleFunc("/fitness/profile", middleware.AuthRequired(handlers.CreateFitnessProfileHandler))
	http.HandleFunc("/fitness/activity", middleware.AuthRequired(handlers.AddActivityHandler))
	http.HandleFunc("/fitness/recommendations", middleware.AuthRequired(handlers.GetFitnessRecommendationsHandler))

}
