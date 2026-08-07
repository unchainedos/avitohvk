package domain

import "time"

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Email        *string
	TG           *string
	PhoneNumber  *string
	IsBanned     bool
	CreatedAt    time.Time
}

type UserUpdate struct {
	Username    *string
	PasswordHash *string
	Email       *string
	TG          *string
	PhoneNumber *string
}
