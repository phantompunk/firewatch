package model

import "time"

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

type AdminUser struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Role        Role       `json:"role"`
	Status      Status     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
}
