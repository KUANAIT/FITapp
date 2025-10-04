package models

import (
	"SSE/auth"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	ID               primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name             string             `json:"name" bson:"name"`
	Email            string             `json:"email" bson:"email"`
	Password         string             `json:"password" bson:"password"`
	MembershipPlanID primitive.ObjectID `json:"membership_plan_id,omitempty" bson:"membership_plan_id,omitempty"`
	MembershipStatus MembershipStatus   `json:"membership_status" bson:"membership_status"`
	MembershipExpiry time.Time          `json:"membership_expiry" bson:"membership_expiry"`
	MemberID         string             `json:"member_id" bson:"member_id"`
	JoinDate         time.Time          `json:"join_date" bson:"join_date"`
	LastCheckIn      time.Time          `json:"last_check_in,omitempty" bson:"last_check_in,omitempty"`
	TotalVisits      int                `json:"total_visits" bson:"total_visits"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" bson:"updated_at"`
}

func (u *User) HashPassword() error {
	hashedPassword, err := auth.HashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = hashedPassword
	return nil
}

func (u *User) CheckPassword(providedPassword string) bool {
	return auth.CheckPassword(u.Password, providedPassword)
}

func (u *User) GenerateMemberID() {
	year := time.Now().Year()
	timestamp := time.Now().Unix()
	u.MemberID = fmt.Sprintf("GYM-%d-%06d", year, timestamp%1000000)
}

func (u *User) IsActiveMember() bool {
	return u.MembershipStatus == StatusActive && time.Now().Before(u.MembershipExpiry)
}

func (u *User) DaysUntilExpiry() int {
	if u.MembershipExpiry.IsZero() {
		return 0
	}
	days := int(time.Until(u.MembershipExpiry).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
