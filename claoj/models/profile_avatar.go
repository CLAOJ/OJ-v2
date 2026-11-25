package models

import "time"

// ProfileAvatar is a v2-only uploaded avatar (on R2) overriding the Gravatar
// identicon. v1 never reads it; see scripts/v2_runtime_tables.sql.
type ProfileAvatar struct {
	ProfileID uint      `gorm:"primaryKey;column:profile_id;autoIncrement:false"`
	URL       string    `gorm:"column:url;size:255;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (ProfileAvatar) TableName() string { return "profile_avatar" }
