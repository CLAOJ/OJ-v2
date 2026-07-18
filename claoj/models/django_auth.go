// Django's built-in auth tables. OJ-v2 reads and writes ROWS in these tables
// (exactly like Django admin does) but never creates or alters them — the
// schema is owned by Django migrations in the OJ repository.
package models

// AuthGroup is Django's auth_group — the "role" unit shared with the Django site.
type AuthGroup struct {
	ID          uint             `gorm:"primaryKey;column:id"`
	Name        string           `gorm:"column:name;size:150;uniqueIndex"`
	Permissions []AuthPermission `gorm:"many2many:auth_group_permissions;joinForeignKey:group_id;joinReferences:permission_id"`
}

func (AuthGroup) TableName() string { return "auth_group" }

// DjangoContentType is django_content_type; app_label+model qualify permission codenames.
type DjangoContentType struct {
	ID       uint   `gorm:"primaryKey;column:id"`
	AppLabel string `gorm:"column:app_label;size:100"`
	Model    string `gorm:"column:model;size:100"`
}

func (DjangoContentType) TableName() string { return "django_content_type" }

// AuthPermission is auth_permission. Full permission string = "{app_label}.{codename}".
type AuthPermission struct {
	ID            uint              `gorm:"primaryKey;column:id"`
	Name          string            `gorm:"column:name;size:255"`
	ContentTypeID uint              `gorm:"column:content_type_id"`
	Codename      string            `gorm:"column:codename;size:100"`
	ContentType   DjangoContentType `gorm:"foreignKey:ContentTypeID"`
}

func (AuthPermission) TableName() string { return "auth_permission" }

// AuthUserGroup is the auth_user_groups join row (user ↔ group).
type AuthUserGroup struct {
	ID      uint `gorm:"primaryKey;column:id"`
	UserID  uint `gorm:"column:user_id"`
	GroupID uint `gorm:"column:group_id"`
}

func (AuthUserGroup) TableName() string { return "auth_user_groups" }

// AuthGroupPermission is the auth_group_permissions join row (group ↔ permission).
type AuthGroupPermission struct {
	ID           uint `gorm:"primaryKey;column:id"`
	GroupID      uint `gorm:"column:group_id"`
	PermissionID uint `gorm:"column:permission_id"`
}

func (AuthGroupPermission) TableName() string { return "auth_group_permissions" }

// AuthUserPermission is the auth_user_user_permissions join row (user ↔ direct permission).
type AuthUserPermission struct {
	ID           uint `gorm:"primaryKey;column:id"`
	UserID       uint `gorm:"column:user_id"`
	PermissionID uint `gorm:"column:permission_id"`
}

func (AuthUserPermission) TableName() string { return "auth_user_user_permissions" }
