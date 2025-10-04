package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type MembershipPlan struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Price       float64            `json:"price" bson:"price"`
	Duration    int                `json:"duration" bson:"duration"`
	Features    []string           `json:"features" bson:"features"`
	IsActive    bool               `json:"is_active" bson:"is_active"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

type MembershipStatus string

const (
	StatusActive    MembershipStatus = "active"
	StatusExpired   MembershipStatus = "expired"
	StatusSuspended MembershipStatus = "suspended"
	StatusCancelled MembershipStatus = "cancelled"
)

func GetDefaultPlans() []MembershipPlan {
	now := time.Now()
	return []MembershipPlan{
		{
			Name:        "Basic",
			Description: "Access to gym equipment and basic facilities",
			Price:       13499,
			Duration:    1,
			Features: []string{
				"Access to gym equipment",
				"Locker room access",
				"Basic fitness assessment",
				"Mobile app access",
			},
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Name:        "Premium",
			Description: "All Basic features plus group classes and extended hours",
			Price:       22499,
			Duration:    1,
			Features: []string{
				"All Basic features",
				"Unlimited group classes",
				"Extended gym hours",
				"Guest pass (2 per month)",
				"Nutrition consultation",
				"Progress tracking",
			},
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Name:        "VIP",
			Description: "Premium features plus personal training and exclusive amenities",
			Price:       35999,
			Duration:    1,
			Features: []string{
				"All Premium features",
				"2 personal training sessions per month",
				"Priority class booking",
				"VIP locker access",
				"Unlimited guest passes",
				"Towel service",
				"Massage therapy discount",
				"Exclusive VIP area access",
			},
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}
