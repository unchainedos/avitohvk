package dto

import "time"

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type RegisterResponse struct {
	ID string `json:"id"`
}

type UserResponse struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Email       *string   `json:"email,omitempty"`
	TG          *string   `json:"tg,omitempty"`
	PhoneNumber *string   `json:"phone_number,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type UpdateUserRequest struct {
	Username    *string `json:"username"`
	Password    *string `json:"password"`
	Email       *string `json:"email"`
	TG          *string `json:"tg"`
	PhoneNumber *string `json:"phone_number"`
}
