// Package organization provides organization management services.
package organization

import (
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/sanitization"
	"gorm.io/gorm"
)

// OrganizationProfile represents an organization with related data.
type OrganizationProfile struct {
	ID                uint
	Name              string
	Slug              string
	ShortName         string
	About             string
	IsOpen            bool
	IsUnlisted        bool
	Slots             *int
	AccessCode        *string
	MemberCount       int
	PerformancePoints float64
	AdminIDs          []uint
	MemberIDs         []uint
}

// CreateOrganizationRequest holds the parameters for creating an organization.
type CreateOrganizationRequest struct {
	Name       string
	Slug       string
	ShortName  string
	About      string
	IsOpen     bool
	IsUnlisted bool
	Slots      *int
	AccessCode *string
}

// UpdateOrganizationRequest holds the parameters for updating an organization.
type UpdateOrganizationRequest struct {
	OrganizationID uint
	Name           *string
	Slug           *string
	ShortName      *string
	About          *string
	IsOpen         *bool
	IsUnlisted     *bool
	Slots          *int
	AccessCode     *string
}

// DeleteOrganizationRequest holds the parameters for deleting an organization.
type DeleteOrganizationRequest struct {
	OrganizationID uint
}

// GetOrganizationRequest holds the parameters for getting an organization.
type GetOrganizationRequest struct {
	OrganizationID uint
}

// ListOrganizationsRequest holds the parameters for listing organizations.
type ListOrganizationsRequest struct {
	Page     int
	PageSize int
}

// ListOrganizationsResponse holds the response for listing organizations.
type ListOrganizationsResponse struct {
	Organizations []OrganizationProfile
	Total         int64
	Page          int
	PageSize      int
}

// JoinOrganizationRequest holds the parameters for joining an organization.
type JoinOrganizationRequest struct {
	OrganizationID uint
	UserID         uint
}

// LeaveOrganizationRequest holds the parameters for leaving an organization.
type LeaveOrganizationRequest struct {
	OrganizationID uint
	UserID         uint
}

// KickUserRequest holds the parameters for kicking a user from an organization.
type KickUserRequest struct {
	OrganizationID uint
	UserID         uint
}

// OrganizationService provides organization management operations.
type OrganizationService struct{}

// NewOrganizationService creates a new OrganizationService instance.
func NewOrganizationService() *OrganizationService {
	return &OrganizationService{}
}

// ListOrganizations retrieves a paginated list of organizations.
func (s *OrganizationService) ListOrganizations(req ListOrganizationsRequest) (*ListOrganizationsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var organizations []models.Organization
	query := db.DB.Order("name ASC")

	// Get total count
	var total int64
	if err := db.DB.Model(&models.Organization{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&organizations).Error; err != nil {
		return nil, err
	}

	result := make([]OrganizationProfile, len(organizations))
	for i, o := range organizations {
		result[i] = organizationToProfile(o)
	}

	return &ListOrganizationsResponse{
		Organizations: result,
		Total:         total,
		Page:          req.Page,
		PageSize:      req.PageSize,
	}, nil
}

// GetOrganization retrieves an organization by ID with full details.
func (s *OrganizationService) GetOrganization(req GetOrganizationRequest) (*OrganizationProfile, error) {
	var org models.Organization
	if err := db.DB.Preload("Admins").Preload("Members").First(&org, req.OrganizationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}

	profile := organizationToProfile(org)
	return &profile, nil
}

// CreateOrganization creates a new organization.
func (s *OrganizationService) CreateOrganization(req CreateOrganizationRequest) (*OrganizationProfile, error) {
	org := models.Organization{
		Name:       sanitization.SanitizeTitle(req.Name),
		Slug:       req.Slug,
		ShortName:  sanitization.SanitizeTitle(req.ShortName),
		About:      sanitization.SanitizeBlogContent(req.About),
		IsOpen:     req.IsOpen,
		IsUnlisted: req.IsUnlisted,
		Slots:      req.Slots,
		AccessCode: req.AccessCode,
	}

	if err := db.DB.Create(&org).Error; err != nil {
		return nil, err
	}

	profile := organizationToProfile(org)
	return &profile, nil
}

// UpdateOrganization updates an existing organization.
func (s *OrganizationService) UpdateOrganization(req UpdateOrganizationRequest) (*OrganizationProfile, error) {
	var org models.Organization
	if err := db.DB.First(&org, req.OrganizationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = sanitization.SanitizeTitle(*req.Name)
	}
	if req.Slug != nil {
		updates["slug"] = *req.Slug
	}
	if req.ShortName != nil {
		updates["short_name"] = sanitization.SanitizeTitle(*req.ShortName)
	}
	if req.About != nil {
		updates["about"] = sanitization.SanitizeBlogContent(*req.About)
	}
	if req.IsOpen != nil {
		updates["is_open"] = *req.IsOpen
	}
	if req.IsUnlisted != nil {
		updates["is_unlisted"] = *req.IsUnlisted
	}
	if req.Slots != nil {
		updates["slots"] = *req.Slots
	}
	if req.AccessCode != nil {
		updates["access_code"] = *req.AccessCode
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&org).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	profile := organizationToProfile(org)
	return &profile, nil
}

// DeleteOrganization performs a soft delete by marking the organization as closed and unlisted.
func (s *OrganizationService) DeleteOrganization(req DeleteOrganizationRequest) error {
	var org models.Organization
	if err := db.DB.First(&org, req.OrganizationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrOrganizationNotFound
		}
		return err
	}

	// Soft delete: mark as not open and unlisted
	return db.DB.Model(&org).Updates(map[string]interface{}{
		"is_open":     false,
		"is_unlisted": true,
	}).Error
}

// JoinOrganization adds a user to an organization.
func (s *OrganizationService) JoinOrganization(req JoinOrganizationRequest) error {
	var org models.Organization
	if err := db.DB.First(&org, req.OrganizationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrOrganizationNotFound
		}
		return err
	}

	var profile models.Profile
	if err := db.DB.First(&profile, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrUserNotFound
		}
		return err
	}

	// Check if already a member
	var count int64
	if err := db.DB.Model(&org).Where("id = ? AND member_count > 0", org.ID).Count(&count).Error; err != nil {
		return err
	}

	// Check slots
	if org.Slots != nil && int(count) >= *org.Slots {
		return ErrOrganizationFull
	}

	return db.DB.Model(&org).Association("Members").Append(&profile)
}

// LeaveOrganization removes a user from an organization.
func (s *OrganizationService) LeaveOrganization(req LeaveOrganizationRequest) error {
	var org models.Organization
	if err := db.DB.First(&org, req.OrganizationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrOrganizationNotFound
		}
		return err
	}

	var profile models.Profile
	if err := db.DB.First(&profile, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrUserNotFound
		}
		return err
	}

	return db.DB.Model(&org).Association("Members").Delete(&profile)
}

// KickUser removes a user from an organization (admin action).
func (s *OrganizationService) KickUser(req KickUserRequest) error {
	// Same as leave, but could have different permissions
	return s.LeaveOrganization(LeaveOrganizationRequest{
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
	})
}

// Helper functions

func organizationToProfile(o models.Organization) OrganizationProfile {
	adminIDs := getProfileIDs(o.Admins)
	memberIDs := getProfileIDs(o.Members)

	return OrganizationProfile{
		ID:                o.ID,
		Name:              o.Name,
		Slug:              o.Slug,
		ShortName:         o.ShortName,
		About:             o.About,
		IsOpen:            o.IsOpen,
		IsUnlisted:        o.IsUnlisted,
		Slots:             o.Slots,
		AccessCode:        o.AccessCode,
		MemberCount:       o.MemberCount,
		PerformancePoints: o.PerformancePoints,
		AdminIDs:          adminIDs,
		MemberIDs:         memberIDs,
	}
}

func getProfileIDs(profiles []models.Profile) []uint {
	ids := make([]uint, len(profiles))
	for i, p := range profiles {
		ids[i] = p.ID
	}
	return ids
}
