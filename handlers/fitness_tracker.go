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

// FitnessTrackerPageHandler displays the fitness tracking dashboard
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

	// Get user info
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

	// Get fitness profile
	fitnessCollection, err := database.GetCollection("SSE", "fitness_profiles")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	var profile models.FitnessProfile
	err = fitnessCollection.FindOne(context.TODO(), bson.M{"user_id": objID}).Decode(&profile)

	// Get recent activities
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
		Authenticated: true, // Since this handler requires authentication
	}

	fmt.Println("Template data - UserName:", data.UserName, "HasProfile:", data.HasProfile)

	err = tmpl.Execute(w, data)
	if err != nil {
		fmt.Println("Template execution error:", err)
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// CreateFitnessProfileHandler creates or updates user's fitness profile
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

	// Validate required fields
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

	// Check if profile already exists
	var existingProfile models.FitnessProfile
	err = collection.FindOne(context.TODO(), bson.M{"user_id": objID}).Decode(&existingProfile)
	if err == nil {
		// Update existing profile
		profile.ID = existingProfile.ID
		profile.CreatedAt = existingProfile.CreatedAt
		_, err = collection.ReplaceOne(context.TODO(), bson.M{"_id": existingProfile.ID}, profile)
	} else {
		// Create new profile
		_, err = collection.InsertOne(context.TODO(), profile)
	}

	if err != nil {
		http.Error(w, "Failed to save profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Fitness profile saved successfully"})
}

// AddActivityHandler adds a new activity entry
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

// GetFitnessRecommendationsHandler gets AI-powered fitness recommendations
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

	// Get user's fitness profile
	collection, err := database.GetCollection("SSE", "fitness_profiles")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	var profile models.FitnessProfile
	err = collection.FindOne(context.TODO(), bson.M{"user_id": objID}).Decode(&profile)
	if err != nil {
		// If no profile exists, return a general recommendation
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"recommendations": "Please create your fitness profile first to get personalized recommendations. Go to the profile section above and fill in your age, gender, height, weight, activity level, and fitness goals.",
		})
		return
	}

	var req struct {
		RequestType      string `json:"request_type"` // "workout", "nutrition", "general"
		SpecificQuestion string `json:"specific_question,omitempty"`
	}
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Println("Calling GetGeminiFitnessRecommendations...")
	recommendations, err := GetGeminiFitnessRecommendations(profile, req.RequestType, req.SpecificQuestion)
	if err != nil {
		fmt.Println("Gemini API error:", err)
		http.Error(w, "Failed to get recommendations", http.StatusInternalServerError)
		return
	}

	fmt.Println("Recommendations received successfully")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"recommendations": recommendations})
}

// GetGeminiFitnessRecommendations gets personalized fitness recommendations from Gemini AI
func GetGeminiFitnessRecommendations(profile models.FitnessProfile, requestType, specificQuestion string) (string, error) {
	fmt.Println("GetGeminiFitnessRecommendations started")
	apiKey := "AIzaSyCdIzPAdPKzHc9-g8h4l9RKZg_xP5sMQDI"
	url := "https://generativelanguage.googleapis.com/v1/models/gemini-2.5-flash:generateContent?key=" + apiKey
	fmt.Println("API URL:", url)

	// Create personalized prompt based on user's profile
	prompt := fmt.Sprintf(`You are a professional fitness trainer and nutritionist. Based on the following user profile, provide personalized recommendations:

User Profile:
- Age: %d years
- Gender: %s
- Height: %.1f cm
- Weight: %.1f kg
- BMI: %.1f
- Activity Level: %s
- Goal: %s
- Target Calories: %d per day
- BMR: %.1f calories
- TDEE: %.1f calories

Request Type: %s

`, profile.Age, profile.Gender, profile.Height, profile.Weight, profile.BMI,
		profile.ActivityLevel, profile.Goal, profile.TargetCalories, profile.BMR, profile.TDEE, requestType)

	if specificQuestion != "" {
		prompt += fmt.Sprintf("Specific Question: %s\n\n", specificQuestion)
	}

	switch requestType {
	case "workout":
		prompt += "Provide a detailed workout plan including exercises, sets, reps, and progression. Consider the user's goal and activity level."
	case "nutrition":
		prompt += "Provide detailed nutrition advice including meal planning, macro distribution, and food recommendations to support their goal."
	case "general":
		prompt += "Provide general fitness and health advice tailored to their profile and goals."
	default:
		prompt += "Provide comprehensive fitness and health recommendations."
	}

	prompt += "\n\nKeep recommendations practical, safe, and achievable. Include specific actionable steps."

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("JSON marshal error:", err)
		return "", err
	}
	fmt.Println("Request payload created, length:", len(body))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("HTTP request creation error:", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	fmt.Println("Sending request to Gemini API...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("HTTP request error:", err)
		return "", err
	}
	defer resp.Body.Close()
	fmt.Println("Response status:", resp.Status)

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return "", err
	}
	fmt.Println("Response body length:", len(respBody))
	fmt.Println("Response body:", string(respBody))

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Println("JSON unmarshal error:", err)
		return "", err
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		fmt.Println("No candidates in response")
		return "", fmt.Errorf("No response from Gemini")
	}

	fmt.Println("Successfully parsed Gemini response")
	return result.Candidates[0].Content.Parts[0].Text, nil
}

// TestFitnessHandler - Simple test handler
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
