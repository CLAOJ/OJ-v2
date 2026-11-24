// Django's contrib flatpages and redirects tables. Like django_auth.go, v2
// reads and writes ROWS here but never creates or alters the schema — it is
// owned by Django migrations in the OJ repository.
package models

// CMSSiteID is the django_site row v2 uses for scoping. This is a single-site
// deployment: v1 sets SITE_ID=1 and every flatpage/redirect belongs to it, so
// v2 stores and matches site 1 rather than modelling django_site at all.
const CMSSiteID uint = 1

// FlatPage mirrors django_flatpage (Django 4.2: no site column; sites are the
// django_flatpage_sites m2m, and url is unique per SITE, not globally).
type FlatPage struct {
	ID                   uint   `gorm:"primaryKey;column:id"`
	URL                  string `gorm:"column:url;size:100;not null;index"`
	Title                string `gorm:"column:title;size:200;not null"`
	Content              string `gorm:"column:content;type:longtext;not null"`
	EnableComments       bool   `gorm:"column:enable_comments;not null;default:0"`
	TemplateName         string `gorm:"column:template_name;size:70;not null"`
	RegistrationRequired bool   `gorm:"column:registration_required;not null;default:0"`
}

func (FlatPage) TableName() string { return "django_flatpage" }

// FlatPageSite is Django's auto-created django_flatpage_sites join row.
type FlatPageSite struct {
	ID         uint `gorm:"primaryKey;column:id"`
	FlatPageID uint `gorm:"column:flatpage_id;not null;uniqueIndex:uq_flatpage_site"`
	SiteID     uint `gorm:"column:site_id;not null;uniqueIndex:uq_flatpage_site"`
}

func (FlatPageSite) TableName() string { return "django_flatpage_sites" }

// Redirect mirrors django_redirect: old_path is unique per site.
type Redirect struct {
	ID      uint   `gorm:"primaryKey;column:id"`
	SiteID  uint   `gorm:"column:site_id;not null;uniqueIndex:uq_redirect_site_old"`
	OldPath string `gorm:"column:old_path;size:200;not null;uniqueIndex:uq_redirect_site_old"`
	NewPath string `gorm:"column:new_path;size:200;not null"`
}

func (Redirect) TableName() string { return "django_redirect" }
