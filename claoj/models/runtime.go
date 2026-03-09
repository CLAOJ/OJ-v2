package models

import "time"

// Language mirrors judge_language.
type Language struct {
	ID               uint    `gorm:"primaryKey;column:id"`
	Key              string  `gorm:"column:key;size:10;not null;uniqueIndex"`
	Name             string  `gorm:"column:name;size:20;not null"`
	ShortName        *string `gorm:"column:short_name;size:10"`
	CommonName       string  `gorm:"column:common_name;size:20;not null"`
	Ace              string  `gorm:"column:ace;size:20;not null"`
	Pygments         string  `gorm:"column:pygments;size:20;not null"`
	Template         string  `gorm:"column:template;type:longtext;not null"`
	Info             string  `gorm:"column:info;size:50;not null;default:''"`
	Description      string  `gorm:"column:description;type:longtext;not null"`
	Extension        string  `gorm:"column:extension;size:10;not null"`
	FileOnly         bool    `gorm:"column:file_only;not null;default:0"`
	FileSizeLimit    int     `gorm:"column:file_size_limit;not null;default:0"`
	IncludeInProblem bool    `gorm:"column:include_in_problem;not null;default:0"`
}

func (Language) TableName() string { return "judge_language" }

// Judge mirrors judge_judge.
type Judge struct {
	ID          uint       `gorm:"primaryKey;column:id"`
	Name        string     `gorm:"column:name;size:50;not null;uniqueIndex"`
	Created     time.Time  `gorm:"column:created;not null"`
	AuthKey     string     `gorm:"column:auth_key;size:100;not null"`
	IsBlocked   bool       `gorm:"column:is_blocked;not null;default:0"`
	IsDisabled  bool       `gorm:"column:is_disabled;not null;default:0"`
	Online      bool       `gorm:"column:online;not null;default:0"`
	StartTime   *time.Time `gorm:"column:start_time"`
	Ping        *float64   `gorm:"column:ping"`
	Load        *float64   `gorm:"column:load"`
	Description string     `gorm:"column:description;type:longtext;not null"`
	LastIP      *string    `gorm:"column:last_ip;size:39"`

	// M2M
	Problems []Problem  `gorm:"many2many:judge_judge_problems;joinForeignKey:judge_id;joinReferences:problem_id"`
	Runtimes []Language `gorm:"many2many:judge_judge_runtimes;joinForeignKey:judge_id;joinReferences:language_id"`
}

func (Judge) TableName() string { return "judge_judge" }

// RuntimeVersion mirrors judge_runtimeversion.
type RuntimeVersion struct {
	ID         uint     `gorm:"primaryKey;column:id"`
	LanguageID uint     `gorm:"column:language_id;not null;index"`
	JudgeID    uint     `gorm:"column:judge_id;not null;index"`
	Name       string   `gorm:"column:name;size:64;not null"`
	Version    string   `gorm:"column:version;size:64;not null;default:''"`
	Priority   int      `gorm:"column:priority;not null;default:0"`
	Language   Language `gorm:"foreignKey:LanguageID"`
	Judge      Judge    `gorm:"foreignKey:JudgeID"`
}

func (RuntimeVersion) TableName() string { return "judge_runtimeversion" }

// PasswordResetToken stores password reset tokens
type PasswordResetToken struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	UserID    uint      `gorm:"column:user_id;not null;index"`
	Token     string    `gorm:"column:token;size:64;not null;uniqueIndex"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null;index"`
}

func (PasswordResetToken) TableName() string { return "password_reset_token" }

// EmailVerificationToken stores email verification tokens
type EmailVerificationToken struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	UserID    uint      `gorm:"column:user_id;not null;index"`
	Token     string    `gorm:"column:token;size:64;not null;uniqueIndex"`
	Email     string    `gorm:"column:email;size:254;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null;index"`
}

func (EmailVerificationToken) TableName() string { return "email_verification_token" }

// TotpDevice stores TOTP configuration for users
type TotpDevice struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	UserID    uint      `gorm:"column:user_id;not null;uniqueIndex"`
	Secret    string    `gorm:"column:secret;size:255;not null"` // encrypted
	Confirmed bool      `gorm:"column:confirmed;not null;default:0"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (TotpDevice) TableName() string { return "totp_device" }

// BackupCode stores backup codes for 2FA
type BackupCode struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	UserID    uint      `gorm:"column:user_id;not null;index"`
	Code      string    `gorm:"column:code;size:64;not null"` // hashed
	Used      bool      `gorm:"column:used;not null;default:0"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (BackupCode) TableName() string { return "backup_code" }

// RefreshToken stores refresh tokens for revocation
type RefreshToken struct {
	ID           uint      `gorm:"primaryKey;column:id"`
	UserID       uint      `gorm:"column:user_id;not null;index"`
	Token        string    `gorm:"column:token;size:512;not null;uniqueIndex"`
	Revoked      bool      `gorm:"column:revoked;not null;default:0"`
	RevokedAt    *time.Time `gorm:"column:revoked_at"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;index"`
	ExpiresAt    time.Time `gorm:"column:expires_at;not null;index"`
	UserAgent    *string   `gorm:"column:user_agent;size:512"`
	ClientIP     *string   `gorm:"column:client_ip;size:39"`
	FamilyID     string    `gorm:"column:family_id;size:64;not null;index"` // Token family for rotation
}

func (RefreshToken) TableName() string { return "refresh_token" }
