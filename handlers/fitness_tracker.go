package handlers

import (
	"SSE/database"
	"SSE/models"
	"SSE/sessions"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"html/template"
	"io/ioutil"
	"net/http"
	"time"
)

func FitnessTrackerPageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("FitnessTrackerPageHandler called")
	tmpl, err := template.ParseFiles("templates/fitness_tracker.html")
	if err != nil {
		fmt.Println("Template error:", err)
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := sessions.Get(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	userCollection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	var user models.User
	err = userCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	fitnessCollection, err := database.GetCollection("SSE", "fitness_profiles")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	var profile models.FitnessProfile
	err = fitnessCollection.FindOne(context.TODO(), bson.M{"user_id": objID}).Decode(&profile)

	activityCollection, err := database.GetCollection("SSE", "activities")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	cursor, err := activityCollection.Find(context.TODO(), bson.M{"user_id": objID}, options.Find().SetSort(bson.M{"date": -1}).SetLimit(10))
	var activities []models.Activity
	if err == nil {
		cursor.All(context.TODO(), &activities)
	}

	data := struct {
		UserName      string
		Profile       models.FitnessProfile
		Activities    []models.Activity
		HasProfile    bool
		Authenticated bool
	}{
		UserName:      user.Name,
		Profile:       profile,
		Activities:    activities,
		HasProfile:    err == nil,
		Authenticated: true,
	}

	fmt.Println("Template data - UserName:", data.UserName, "HasProfile:", data.HasProfile)

	err = tmpl.Execute(w, data)
	if err != nil {
		fmt.Println("Template execution error:", err)
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func CreateFitnessProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := sessions.Get(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var profile models.FitnessProfile
	err = json.NewDecoder(r.Body).Decode(&profile)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if profile.Age <= 0 || profile.Height <= 0 || profile.Weight <= 0 {
		http.Error(w, "Age, height, and weight must be positive", http.StatusBadRequest)
		return
	}

	profile.UserID = objID
	profile.CalculateBMI()
	profile.CalculateTargetCalories()
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	collection, err := database.GetCollection("SSE", "fitness_profiles")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var existingProfile models.FitnessProfile
	err = collection.FindOne(context.TODO(), bson.M{"user_id": objID}).Decode(&existingProfile)
	if err == nil {
		profile.ID = existingProfile.ID
		profile.CreatedAt = existingProfile.CreatedAt
		_, err = collection.ReplaceOne(context.TODO(), bson.M{"_id": existingProfile.ID}, profile)
	} else {
		_, err = collection.InsertOne(context.TODO(), profile)
	}

	if err != nil {
		http.Error(w, "Failed to save profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Fitness profile saved successfully"})
}

func AddActivityHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("AddActivityHandler called")
	if r.Method != http.MethodPost {
		fmt.Println("Method not allowed:", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := sessions.Get(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var activity models.Activity
	err = json.NewDecoder(r.Body).Decode(&activity)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	activity.UserID = objID
	activity.CreatedAt = time.Now()
	if activity.Date.IsZero() {
		activity.Date = time.Now()
	}

	collection, err := database.GetCollection("SSE", "activities")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	_, err = collection.InsertOne(context.TODO(), activity)
	if err != nil {
		http.Error(w, "Failed to save activity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Activity added successfully"})
}

func GetFitnessRecommendationsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GetFitnessRecommendationsHandler called")
	if r.Method != http.MethodPost {
		fmt.Println("Method not allowed:", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := sessions.Get(r)
	if err != nil {
		fmt.Println("Session error:", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	collection, err := database.GetCollection("SSE", "fitness_profiles")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	var profile models.FitnessProfile
	err = collection.FindOne(context.TODO(), bson.M{"user_id": objID}).Decode(&profile)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"recommendations": "Please create your fitness profile first to get personalized recommendations. Go to the profile section above and fill in your age, gender, height, weight, activity level, and fitness goals.",
		})
		return
	}

	var req struct {
		RequestType      string `json:"request_type"`
		SpecificQuestion string `json:"specific_question,omitempty"`
	}
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Println("Calling fitness recommendation AI API...")
	recommendations, err := GetAIFitnessRecommendations(profile, req.RequestType, req.SpecificQuestion)
	if err != nil {
		fmt.Println("AI API error:", err)
		http.Error(w, "Failed to get recommendations", http.StatusInternalServerError)
		return
	}

	fmt.Println("Recommendations received successfully")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"recommendations": recommendations})
}

func GetAIFitnessRecommendations(profile models.FitnessProfile, requestType, specificQuestion string) (string, error) {
	fmt.Println("GetAIFitnessRecommendations started (REST version)")

	type PlanRequest struct {
		Age                int      `json:"age"`
		Sex                string   `json:"sex"`
		HeightCM           float64  `json:"height_cm"`
		WeightKG           float64  `json:"weight_kg"`
		Goal               string   `json:"goal"`
		SessionsPerWeek    int      `json:"sessions_per_week"`
		BudgetKzt          string   `json:"budget_kzt"`
		Location           string   `json:"location"`
		HealthConditions   string   `json:"health_conditions"`
		ExistingActivities []string `json:"existing_activities"`
		ActivityLevel      int      `json:"activity_level"`
	}

	sex := profile.Gender
	if sex == "male" || sex == "Male" {
		sex = "Male"
	} else if sex == "female" || sex == "Female" {
		sex = "Female"
	} else {
		sex = "Other"
	}

	reqBody := PlanRequest{
		Age:                profile.Age,
		Sex:                sex,
		HeightCM:           profile.Height,
		WeightKG:           profile.Weight,
		Goal:               profile.Goal,
		SessionsPerWeek:    3,
		BudgetKzt:          profile.BudgetKzt,
		Location:           profile.Location,
		HealthConditions:   profile.HealthConditions,
		ExistingActivities: []string{},
		ActivityLevel:      mapActivityLevel(profile.ActivityLevel),
	}

	if specificQuestion != "" {
	}

	jsonPayload, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Println("JSON marshal error:", err)
		return "", err
	}

	apiURL := "https://meatiest-inkier-miriam.ngrok-free.dev/plan"
	request, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonPayload))
	if err != nil {
		fmt.Println("HTTP request creation error:", err)
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		fmt.Println("HTTP request error:", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("API error status:", resp.Status, string(body))
		return "", fmt.Errorf("REST API error: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return "", err
	}

	type ApiResponse struct {
		UserSummary struct {
			Age                int      `json:"age"`
			Sex                string   `json:"sex"`
			HeightCM           float64  `json:"height_cm"`
			WeightKG           float64  `json:"weight_kg"`
			BMI                float64  `json:"bmi"`
			SessionsPerWeek    int      `json:"sessions_per_week"`
			BudgetTier         string   `json:"budget_tier"`
			HealthConditions   string   `json:"health_conditions"`
			ExistingActivities []string `json:"existing_activities"`
			PredictedLabel     string   `json:"predicted_label"`
		} `json:"user_summary"`
		TrainingSetting struct {
			Where, Reason string
		} `json:"training_setting"`
		WeeklyWorkouts []struct {
			DayIndex       int    `json:"day_index"`
			Focus          string `json:"focus"`
			SessionMinutes int    `json:"session_minutes"`
			Exercises      []struct {
				Name string `json:"name"`
				Sets string `json:"sets"`
				Reps string `json:"reps"`
			} `json:"exercises"`
		} `json:"weekly_workouts"`
		Diet struct {
			EstimatedDailyKcal int `json:"estimated_daily_kcal"`
			Macros             struct {
				ProteinG int         `json:"protein_g"`
				CarbsG   interface{} `json:"carbs_g"`
				FatG     interface{} `json:"fat_g"`
			} `json:"macros"`
			Advice       string `json:"advice"`
			BudgetAdvice string `json:"budget_advice"`
		} `json:"diet"`
		GymRecommendationText *string `json:"gym_recommendation_text"`
		Notes                 string  `json:"notes"`
	}

	var apiResp ApiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		fmt.Println("JSON unmarshal error:", err)
		return "", err
	}

	var result bytes.Buffer
	result.WriteString("**Personalized Fitness Plan**\n\n")
	result.WriteString(fmt.Sprintf("**User**: %d y/o %s, %.1fcm, %.1fkg, BMI=%.1f\n\n", apiResp.UserSummary.Age, apiResp.UserSummary.Sex, apiResp.UserSummary.HeightCM, apiResp.UserSummary.WeightKG, apiResp.UserSummary.BMI))
	result.WriteString(fmt.Sprintf("**Goal:** %s. Sessions/week: %d. Location: %s\n\n", apiResp.UserSummary.PredictedLabel, apiResp.UserSummary.SessionsPerWeek, profile.Location))
	if apiResp.UserSummary.HealthConditions != "" {
		result.WriteString(fmt.Sprintf("**Health Conditions:** %s\n\n", apiResp.UserSummary.HealthConditions))
	}

	if apiResp.TrainingSetting.Where != "" {
		result.WriteString(fmt.Sprintf("**Training Location:** %s\n_Reason: %s_\n\n", apiResp.TrainingSetting.Where, apiResp.TrainingSetting.Reason))
	}

	if len(apiResp.WeeklyWorkouts) > 0 {
		result.WriteString("### Weekly Workouts Plan\n")
		for _, wk := range apiResp.WeeklyWorkouts {
			result.WriteString(fmt.Sprintf("- **Day %d** (%s, %d min): ", wk.DayIndex+1, wk.Focus, wk.SessionMinutes))
			for _, ex := range wk.Exercises {
				result.WriteString(fmt.Sprintf("%s (%s x %s); ", ex.Name, ex.Sets, ex.Reps))
			}
			result.WriteString("\n")
		}
		result.WriteString("\n")
	}

	result.WriteString("### Diet Guidance\n")
	result.WriteString(fmt.Sprintf("- **Calories:** %d kcal/day\n", apiResp.Diet.EstimatedDailyKcal))
	result.WriteString(fmt.Sprintf("- **Protein:** %dg ", apiResp.Diet.Macros.ProteinG))
	result.WriteString(fmt.Sprintf("- **Carbs:** %v ", apiResp.Diet.Macros.CarbsG))
	result.WriteString(fmt.Sprintf("- **Fat:** %v\n", apiResp.Diet.Macros.FatG))
	result.WriteString(fmt.Sprintf("- %s\n", apiResp.Diet.Advice))
	if apiResp.Diet.BudgetAdvice != "" {
		result.WriteString(fmt.Sprintf("- _%s_\n", apiResp.Diet.BudgetAdvice))
	}

	if apiResp.GymRecommendationText != nil {
		result.WriteString(fmt.Sprintf("**Gym Advice:** %s\n\n", *apiResp.GymRecommendationText))
	}
	if apiResp.Notes != "" {
		result.WriteString(fmt.Sprintf("**Notes:** %s\n", apiResp.Notes))
	}

	return result.String(), nil
}

func mapActivityLevel(level string) int {
	switch level {
	case "sedentary":
		return 1
	case "light":
		return 2
	case "moderate":
		return 3
	case "active":
		return 4
	case "very_active":
		return 5
	default:
		return 2
	}
}

func TestFitnessHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/test_fitness.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		UserName   string
		HasProfile bool
	}{
		UserName:   "Test User",
		HasProfile: false,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
