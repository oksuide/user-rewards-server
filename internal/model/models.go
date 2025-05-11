package model

import "time"

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Points    int       `json:"points"`
	Referrer  *string   `json:"referrer,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Task struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Points      int    `json:"points"`
}

type UserStatus struct {
	User           User   `json:"user"`
	CompletedTasks []Task `json:"completed_tasks"`
	Referrals      []User `json:"referrals,omitempty"`
}

type LeaderboardEntry struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Points   int    `json:"points"`
	Position int    `json:"position"`
}

type Referral struct {
	ReferrerID string    `json:"referrer_id" db:"referrer_id"`
	RefereeID  string    `json:"referee_id" db:"referee_id"`
	Date       time.Time `json:"date" db:"date"`
}
