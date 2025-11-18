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

	if profile.Location != "" {
		result.WriteString("\n#### Recommended Fitness Spots in ")
		result.WriteString(profile.Location + " district")
		result.WriteString("\n")

		switch profile.Location {
		case "Almaty":
			appendParkWithMap(&result,
				"Zheruiyk Park",
				"https://maps.app.goo.gl/u7jXA2QsDuYJZ8159",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d20026.501284176888!2d71.48262!3d51.1395852!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x42458164bd64bdd7%3A0xb6455d1f92027b0c!2z0J_QsNGA0LogwqvQltC10YDSsdC50YvSm8K7!5e0!3m2!1sru!2skz!4v1763485095800!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
			appendParkWithMap(&result,
				"Park named after B. Momyshuly",
				"https://maps.app.goo.gl/7ywzBg8QMSm5gLm28",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d80118.66676408233!2d71.3551787!3d51.1322877!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x424583fe49ec2f05%3A0xbfdaf9e955678c0e!2z0L_QsNGA0Log0LjQvC4g0JHQsNGD0YvRgNC20LDQvdCwINCc0L7QvNGL0YjSsdC70Ys!5e0!3m2!1sru!2skz!4v1763485377075!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
			appendParkWithMap(&result,
				"Triathlon Park",
				"https://maps.app.goo.gl/qaiKPwsi9pDLk8Du8",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d20029.973216144732!2d71.4444481!3d51.131581!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x4245870067b1517d%3A0x6a00eb76b363a4e5!2sTriathlon%20Park!5e0!3m2!1sru!2skz!4v1763485430182!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
		case "Baikonur":
			appendParkWithMap(&result,
				"Atyrau Bridge / Embankment Area",
				"https://maps.app.goo.gl/Bm6S1RuvuWsUHWzm9",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d40045.51154599432!2d71.4119242!3d51.1482191!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x424587987fa8fedd%3A0x555f7a5fccb9e949!2z0JDRgtGL0YDQsNGDINCa06nQv9GW0YDRlg!5e0!3m2!1sru!2skz!4v1763485689999!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
		case "Esil":
			appendParkWithMap(&result,
				"Botanical Garden",
				"https://maps.app.goo.gl/brmo6SAZyDAphAne7",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m18!1m12!1m3!1d40081.73841599552!2d71.33993334863281!3d51.10645550000002!2m3!1f0!2f0!3f0!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x42458435e1300fc1%3A0xfd44a581cbd7eb4e!2z0JHQvtGC0LDQvdC40YfQtdGB0LrQuNC5INGB0LDQtA!5e0!3m2!1sru!2skz!4v1763485772326!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
			appendParkWithMap(&result,
				"Expo Park",
				"https://maps.app.goo.gl/YWq5SCAEQtivUvf19",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m18!1m12!1m3!1d2506.0302686086943!2d71.4128179!3d51.0894489!2m3!1f0!2f0!3f0!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x4245857b9fa89d21%3A0xc4c832ddd885b44d!2z0J_QsNGA0LogRXhwbyAyMDE3!5e0!3m2!1sru!2skz!4v1763485823987!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
			appendParkWithMap(&result,
				"Linear Park",
				"https://maps.app.goo.gl/vuSPejUBT8E4yiJr7",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d80195.53565010101!2d71.3947841!3d51.0879684!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x424585d08682705b%3A0x32db93640e0d489b!2z0JvQuNC90LXQudC90YvQuSDQv9Cw0YDQuiDQkNGB0YLQsNC90LA!5e0!3m2!1sru!2skz!4v1763485851784!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
			appendParkWithMap(&result,
				"Lovers' Park",
				"https://maps.app.goo.gl/KDKH5vH6nDuA5hPr7",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d80116.97619318932!2d71.2721729!3d51.1332621!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x4245869c77fa4883%3A0x6aaf637c7b90145b!2z0pLQsNGI0YvSm9GC0LDRgCDRgdCw0Y_QsdCw0pPRiw!5e0!3m2!1sru!2skz!4v1763485960762!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
		case "Nura":
			appendParkWithMap(&result,
				"Turan Ave.",
				"https://maps.app.goo.gl/1z8fPZaw2i4sVvmf6",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d20043.2937688575!2d71.3914396!3d51.1008634!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x42458695c93afaaf%3A0xef0ec94aac7cadb3!2z0L_RgNC-0YHQvy4g0KLRg9GA0LDQvSwgMDIwMDAwINCQ0YHRgtCw0L3QsA!5e0!3m2!1sru!2skz!4v1763485999172!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
			appendParkWithMap(&result,
				"Central Park",
				"https://maps.app.goo.gl/x1GR7mPatBcNvpLA9",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d40065.99948832376!2d71.3808382!3d51.1246029!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x424586c58884bee3%3A0x3c695a5bf1c2492b!2z0KbQtdC90YLRgNCw0LvRjNC90YvQuSDQn9Cw0YDQug!5e0!3m2!1sru!2skz!4v1763486015368!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
		case "Saryarka":
			appendParkWithMap(&result,
				"Korgalzhyn Park",
				"https://maps.app.goo.gl/DEomXwXn27bNJ5bNA",
				`<<iframe src="https://www.google.com/maps/embed?pb=!1m18!1m12!1m3!1d56590.97603169474!2d71.30637580720908!3d51.18238084809854!2m3!1f0!2f0!3f0!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x4245879cbfbe3f17%3A0x25861b90d4b416da!2z0KHQutCy0LXRgCAi0JrQvtGA0LPQsNC70LbRi9C9Ig!5e0!3m2!1sru!2skz!4v1763486356881!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
			appendParkWithMap(&result,
				"Koktal Park",
				"https://maps.app.goo.gl/BzXxPNA8HnQBZpWf7",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d20010.411010311265!2d71.3518228!3d51.176668!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x424587a617c49225%3A0xca32e5f49d063e98!2z0JrQvtC60YLQsNC7!5e0!3m2!1sru!2skz!4v1763486107213!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
		case "Sarayshyq":
			appendParkWithMap(&result,
				"President's Park",
				"https://maps.app.goo.gl/BT2u9wQwyskF28T7A",
				`<iframe src="https://www.google.com/maps/embed?pb=!1m14!1m8!1m3!1d80143.17587666509!2d71.390861!3d51.1181598!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x4245840f6e6b75a7%3A0x9a8f0fe5c39da89c!2z0J_RgNC10LfQuNC00LXQvdGC0YHQutC40Lkg0L_QsNGA0Lo!5e0!3m2!1sru!2skz!4v1763486139314!5m2!1sru!2skz" width="100%" height="300" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`)
		default:
			result.WriteString("Stay At Home\n")
		}
	}
	return result.String(), nil
}

func appendParkWithMap(buf *bytes.Buffer, name, link, iframeHTML string) {
	buf.WriteString("- [")
	buf.WriteString(name)
	buf.WriteString("](")
	buf.WriteString(link)
	buf.WriteString(")\n")
	if iframeHTML != "" {
		buf.WriteString(iframeHTML)
		buf.WriteString("\n\n")
	}
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
