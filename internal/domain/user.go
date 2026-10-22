package domain

import "time"

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID           int64
	FullName     string
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserUpdate struct {
	FullName     *string
	Email        *string
	PasswordHash *string
}
