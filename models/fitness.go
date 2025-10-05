package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// FitnessProfile represents user's physical and fitness data
type FitnessProfile struct {
	ID             primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID         primitive.ObjectID `json:"user_id" bson:"user_id"`
	Age            int                `json:"age" bson:"age"`
	Gender         string             `json:"gender" bson:"gender"`                 // "male", "female", "other"
	Height         float64            `json:"height" bson:"height"`                 // in cm
	Weight         float64            `json:"weight" bson:"weight"`                 // in kg
	ActivityLevel  string             `json:"activity_level" bson:"activity_level"` // "sedentary", "light", "moderate", "active", "very_active"
	Goal           string             `json:"goal" bson:"goal"`                     // "lose_weight", "gain_weight", "maintain", "build_muscle"
	TargetWeight   float64            `json:"target_weight,omitempty" bson:"target_weight,omitempty"`
	TargetCalories int                `json:"target_calories,omitempty" bson:"target_calories,omitempty"`
	BMI            float64            `json:"bmi" bson:"bmi"`
	BMR            float64            `json:"bmr" bson:"bmr"`   // Basal Metabolic Rate
	TDEE           float64            `json:"tdee" bson:"tdee"` // Total Daily Energy Expenditure
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}

// Activity represents a fitness activity entry
type Activity struct {
	ID             primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID         primitive.ObjectID `json:"user_id" bson:"user_id"`
	ActivityType   string             `json:"activity_type" bson:"activity_type"` // "cardio", "strength", "flexibility", "sports"
	Name           string             `json:"name" bson:"name"`
	Duration       int                `json:"duration" bson:"duration"` // in minutes
	CaloriesBurned int                `json:"calories_burned" bson:"calories_burned"`
	Intensity      string             `json:"intensity" bson:"intensity"` // "low", "moderate", "high"
	Date           time.Time          `json:"date" bson:"date"`
	Notes          string             `json:"notes,omitempty" bson:"notes,omitempty"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
}

// FitnessGoal represents user's fitness goals and progress
type FitnessGoal struct {
	ID           primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID       primitive.ObjectID `json:"user_id" bson:"user_id"`
	GoalType     string             `json:"goal_type" bson:"goal_type"` // "weight_loss", "weight_gain", "muscle_building", "endurance", "strength"
	TargetValue  float64            `json:"target_value" bson:"target_value"`
	CurrentValue float64            `json:"current_value" bson:"current_value"`
	TargetDate   time.Time          `json:"target_date" bson:"target_date"`
	IsCompleted  bool               `json:"is_completed" bson:"is_completed"`
	Progress     float64            `json:"progress" bson:"progress"` // percentage
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

// CalculateBMI calculates Body Mass Index
func (fp *FitnessProfile) CalculateBMI() {
	if fp.Height > 0 {
		fp.BMI = fp.Weight / ((fp.Height / 100) * (fp.Height / 100))
	}
}

// CalculateBMR calculates Basal Metabolic Rate using Mifflin-St Jeor Equation
func (fp *FitnessProfile) CalculateBMR() {
	if fp.Age > 0 && fp.Weight > 0 && fp.Height > 0 {
		if fp.Gender == "male" {
			fp.BMR = 10*fp.Weight + 6.25*fp.Height - 5*float64(fp.Age) + 5
		} else {
			fp.BMR = 10*fp.Weight + 6.25*fp.Height - 5*float64(fp.Age) - 161
		}
	}
}

// CalculateTDEE calculates Total Daily Energy Expenditure
func (fp *FitnessProfile) CalculateTDEE() {
	fp.CalculateBMR()

	activityMultipliers := map[string]float64{
		"sedentary":   1.2,
		"light":       1.375,
		"moderate":    1.55,
		"active":      1.725,
		"very_active": 1.9,
	}

	if multiplier, exists := activityMultipliers[fp.ActivityLevel]; exists {
		fp.TDEE = fp.BMR * multiplier
	} else {
		fp.TDEE = fp.BMR * 1.2 // default to sedentary
	}
}

// CalculateTargetCalories calculates target calories based on goal
func (fp *FitnessProfile) CalculateTargetCalories() {
	fp.CalculateTDEE()

	switch fp.Goal {
	case "lose_weight":
		fp.TargetCalories = int(fp.TDEE - 500) // 500 calorie deficit for ~1lb/week loss
	case "gain_weight":
		fp.TargetCalories = int(fp.TDEE + 500) // 500 calorie surplus for ~1lb/week gain
	case "build_muscle":
		fp.TargetCalories = int(fp.TDEE + 300) // 300 calorie surplus for muscle building
	default: // maintain
		fp.TargetCalories = int(fp.TDEE)
	}
}
