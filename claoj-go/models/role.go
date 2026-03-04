package models

import "time"

// Role represents a user role with associated permissions
// Table: judge_role
type Role struct {
	ID          uint       `gorm:"primaryKey;column:id"`
	Name        string     `gorm:"column:name;size:50;not null;uniqueIndex"`
	DisplayName string     `gorm:"column:display_name;size:100;not null"`
	Description string     `gorm:"column:description;type:text"`
	Color       string     `gorm:"column:color;size:20;default:'#6b7280'"`
	IsDefault   bool       `gorm:"column:is_default;not null;default:0"` // If true, assigned to new users
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	Permissions []Permission `gorm:"many2many:judge_role_permissions;joinForeignKey:role_id;joinReferences:permission_id"`
	Profiles    []Profile    `gorm:"many2many:judge_profile_roles;joinForeignKey:role_id;joinReferences:profile_id"`
}

func (Role) TableName() string { return "judge_role" }

// Permission represents a single permission that can be assigned to roles
// Table: judge_permission
type Permission struct {
	ID          uint      `gorm:"primaryKey;column:id"`
	Code        string    `gorm:"column:code;size:100;not null;uniqueIndex"` // e.g., "problems.create"
	Name        string    `gorm:"column:name;size:200;not null"`
	Description string    `gorm:"column:description;type:text"`
	Category    string    `gorm:"column:category;size:50"` // problems, contests, users, etc.
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (Permission) TableName() string { return "judge_permission" }

// RolePermission is the join table for roles and permissions
// This is managed automatically by GORM's many2many relationship
type RolePermission struct {
	RoleID       uint `gorm:"primaryKey;column:role_id;uniqueIndex:idx_role_perm"`
	PermissionID uint `gorm:"primaryKey;column:permission_id;uniqueIndex:idx_role_perm"`
}

func (RolePermission) TableName() string { return "judge_role_permissions" }

// ProfileRole is the join table for profiles and roles
type ProfileRole struct {
	ProfileID uint `gorm:"primaryKey;column:profile_id;uniqueIndex:idx_profile_role"`
	RoleID    uint `gorm:"primaryKey;column:role_id;uniqueIndex:idx_profile_role"`
}

func (ProfileRole) TableName() string { return "judge_profile_roles" }
