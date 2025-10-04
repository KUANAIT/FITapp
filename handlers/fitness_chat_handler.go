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
	"html/template"
	"io/ioutil"
	"net/http"
)

func FitnessChatPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/fitness_chat.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
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
	collection, err := database.GetCollection("SSE", "users")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	var user models.User
	err = collection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	data := struct {
		UserName string
	}{
		UserName: user.Name,
	}
	tmpl.Execute(w, data)
}

func AskFitnessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Question string `json:"question"`
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &req)
	if err != nil || req.Question == "" {
		http.Error(w, "Invalid JSON or empty question", http.StatusBadRequest)
		return
	}

	answer, err := GetFitnessAIAnswer(req.Question)
	if err != nil {
		fmt.Println("OpenAI error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"answer": "Sorry, there was an error contacting the AI."})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"answer": answer})
}

func GetFitnessAIAnswer(question string) (string, error) {
	apiKey := "AIzaSyCdIzPAdPKzHc9-g8h4l9RKZg_xP5sMQDI"
	url := "https://generativelanguage.googleapis.com/v1/models/gemini-2.5-flash:generateContent?key=" + apiKey

	prompt := "You are a helpful fitness assistant. Only answer questions related to fitness, exercise, nutrition, and health. If a question is not about fitness, politely say you can only answer fitness-related questions.\n\n" + question

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
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fmt.Println("Gemini raw response:", string(respBody))

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
		return "", err
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("No response from Gemini")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}
