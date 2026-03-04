// Package models mirrors the Django judge app's database schema using GORM.
// Table names match Django's convention: judge_<modelname> (lowercase).
// We intentionally do NOT use AutoMigrate — Django manages the schema.
package models

import "time"

// Organization mirrors judge_organization.
type Organization struct {
	ID                uint      `gorm:"primaryKey;column:id"`
	Name              string    `gorm:"column:name;size:128;not null"`
	Slug              string    `gorm:"column:slug;size:128;not null"`
	ShortName         string    `gorm:"column:short_name;size:20;not null"`
	About             string    `gorm:"column:about;type:longtext;not null"`
	CreationDate      time.Time `gorm:"column:creation_date;not null"`
	IsOpen            bool      `gorm:"column:is_open;not null;default:0"`
	IsUnlisted        bool      `gorm:"column:is_unlisted;not null;default:1"`
	Slots             *int      `gorm:"column:slots"`
	AccessCode        *string   `gorm:"column:access_code;size:7"`
	LogoOverrideImage string    `gorm:"column:logo_override_image;size:150;not null;default:''"`
	MemberCount       int       `gorm:"column:member_count;not null;default:0"`
	PerformancePoints float64   `gorm:"column:performance_points;not null;default:0"`

	// relations
	Admins  []Profile `gorm:"many2many:judge_organization_admins;joinForeignKey:organization_id;joinReferences:profile_id"`
	Members []Profile `gorm:"many2many:judge_profile_organizations;joinForeignKey:organization_id;joinReferences:profile_id"`
}

func (Organization) TableName() string { return "judge_organization" }

// Profile mirrors judge_profile (the user-extended info; auth_user is Django's built-in).
type Profile struct {
	ID                      uint       `gorm:"primaryKey;column:id"`
	UserID                  uint       `gorm:"column:user_id;uniqueIndex;not null"`
	About                   *string    `gorm:"column:about;type:longtext"`
	Timezone                string     `gorm:"column:timezone;size:50;not null"`
	LanguageID              uint       `gorm:"column:language_id;not null"`
	Points                  float64    `gorm:"column:points;not null;index;default:0"`
	PerformancePoints       float64    `gorm:"column:performance_points;not null;index;default:0"`
	ContributionPoints      int        `gorm:"column:contribution_points;not null;index;default:0"`
	ProblemCount            int        `gorm:"column:problem_count;not null;index;default:0"`
	AceTheme                string     `gorm:"column:ace_theme;size:30;not null;default:'auto'"`
	SiteTheme               string     `gorm:"column:site_theme;size:10;not null;default:'auto'"`
	LastAccess              time.Time  `gorm:"column:last_access;not null"`
	IP                      *string    `gorm:"column:ip;size:39"`
	DisplayRank             string     `gorm:"column:display_rank;size:10;not null;default:'user'"`
	Mute                    bool       `gorm:"column:mute;not null;default:0"`
	IsUnlisted              bool       `gorm:"column:is_unlisted;not null;default:0"`
	BanReason               *string    `gorm:"column:ban_reason;type:longtext"`
	Rating                  *int       `gorm:"column:rating"`
	UserScript              string     `gorm:"column:user_script;type:longtext;not null"`
	CurrentContestID        *uint      `gorm:"column:current_contest_id;uniqueIndex"`
	MathEngine              string     `gorm:"column:math_engine;size:4;not null"`
	IsTotpEnabled           bool       `gorm:"column:is_totp_enabled;not null;default:0"`
	IsWebauthnEnabled       bool       `gorm:"column:is_webauthn_enabled;not null;default:0"`
	TotpKey                 *string    `gorm:"column:totp_key;size:255"`      // encrypted
	ScratchCodes            *string    `gorm:"column:scratch_codes;size:255"` // encrypted
	LastTotpTimecode        int        `gorm:"column:last_totp_timecode;not null;default:0"`
	ApiToken                *string    `gorm:"column:api_token;size:64"`
	Notes                   *string    `gorm:"column:notes;type:longtext"`
	DataLastDownloaded      *time.Time `gorm:"column:data_last_downloaded"`
	UsernameDisplayOverride string     `gorm:"column:username_display_override;size:100;not null;default:''"`

	// relations
	Language      Language       `gorm:"foreignKey:LanguageID"`
	User          AuthUser       `gorm:"foreignKey:UserID"`
	Organizations []Organization `gorm:"many2many:judge_profile_organizations;joinForeignKey:profile_id;joinReferences:organization_id"`
	Roles         []Role         `gorm:"many2many:judge_profile_roles;joinForeignKey:profile_id;joinReferences:role_id"`
}

func (Profile) TableName() string { return "judge_profile" }

// OrganizationRequest mirrors judge_organizationrequest.
type OrganizationRequest struct {
	ID             uint         `gorm:"primaryKey;column:id"`
	UserID         uint         `gorm:"column:user_id;not null;index"`
	OrganizationID uint         `gorm:"column:organization_id;not null;index"`
	Time           time.Time    `gorm:"column:time;not null"`
	State          string       `gorm:"column:state;size:1;not null"` // P=Pending, A=Approved, R=Rejected
	Reason         string       `gorm:"column:reason;type:longtext;not null"`
	User           Profile      `gorm:"foreignKey:UserID"`
	Organization   Organization `gorm:"foreignKey:OrganizationID"`
}

func (OrganizationRequest) TableName() string { return "judge_organizationrequest" }

// WebAuthnCredential mirrors judge_webauthncredential.
type WebAuthnCredential struct {
	ID        uint    `gorm:"primaryKey;column:id"`
	UserID    uint    `gorm:"column:user_id;not null;index"`
	Name      string  `gorm:"column:name;size:100;not null"`
	CredID    string  `gorm:"column:cred_id;size:255;not null;uniqueIndex"`
	PublicKey string  `gorm:"column:public_key;type:longtext;not null"`
	Counter   int64   `gorm:"column:counter;not null"`
	User      Profile `gorm:"foreignKey:UserID"`
}

func (WebAuthnCredential) TableName() string { return "judge_webauthncredential" }

// AuthUser mirrors Django's auth_user table (read-only from Go).
type AuthUser struct {
	ID          uint      `gorm:"primaryKey;column:id"`
	Username    string    `gorm:"column:username;size:150;not null;uniqueIndex"`
	Password    string    `gorm:"column:password;size:128;not null"`
	FirstName   string    `gorm:"column:first_name;size:150;not null"`
	LastName    string    `gorm:"column:last_name;size:150;not null"`
	Email       string    `gorm:"column:email;size:254;not null"`
	IsStaff     bool      `gorm:"column:is_staff;not null;default:0"`
	IsActive    bool      `gorm:"column:is_active;not null;default:1"`
	IsSuperuser bool      `gorm:"column:is_superuser;not null;default:0"`
	DateJoined  time.Time `gorm:"column:date_joined;not null"`
}

func (AuthUser) TableName() string { return "auth_user" }
