// Package license provides license management services.
package license

import (
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"gorm.io/gorm"
)

// LicenseService provides license management operations.
type LicenseService struct{}

// NewLicenseService creates a new LicenseService instance.
func NewLicenseService() *LicenseService {
	return &LicenseService{}
}

// ListLicenses retrieves a paginated list of licenses.
func (s *LicenseService) ListLicenses(req ListLicensesRequest) (*ListLicensesResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var licenses []models.License
	query := db.DB.Model(&models.License{})

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Order("key ASC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&licenses).Error; err != nil {
		return nil, err
	}

	result := make([]License, len(licenses))
	for i, l := range licenses {
		result[i] = licenseToModel(l)
	}

	return &ListLicensesResponse{
		Licenses: result,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetLicense retrieves a license by ID.
func (s *LicenseService) GetLicense(req GetLicenseRequest) (*License, error) {
	if req.LicenseID == 0 {
		return nil, ErrInvalidLicenseID
	}

	var lic models.License
	if err := db.DB.First(&lic, req.LicenseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrLicenseNotFound
		}
		return nil, err
	}

	result := licenseToModel(lic)
	return &result, nil
}

// CreateLicense creates a new license.
func (s *LicenseService) CreateLicense(req CreateLicenseRequest) (*License, error) {
	if req.Key == "" {
		return nil, ErrEmptyKey
	}
	if req.Name == "" {
		return nil, ErrEmptyName
	}
	if req.Link == "" {
		return nil, ErrEmptyLink
	}

	// Check if key already exists
	var existing models.License
	if err := db.DB.Where("key = ?", req.Key).First(&existing).Error; err == nil {
		return nil, ErrKeyExists
	}

	lic := models.License{
		Key:     req.Key,
		Link:    req.Link,
		Name:    req.Name,
		Display: req.Display,
		Icon:    req.Icon,
		Text:    req.Text,
	}

	if err := db.DB.Create(&lic).Error; err != nil {
		return nil, err
	}

	result := licenseToModel(lic)
	return &result, nil
}

// UpdateLicense updates an existing license.
func (s *LicenseService) UpdateLicense(req UpdateLicenseRequest) (*License, error) {
	if req.LicenseID == 0 {
		return nil, ErrInvalidLicenseID
	}

	var lic models.License
	if err := db.DB.First(&lic, req.LicenseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrLicenseNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Link != nil {
		updates["link"] = *req.Link
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Display != nil {
		updates["display"] = *req.Display
	}
	if req.Icon != nil {
		updates["icon"] = *req.Icon
	}
	if req.Text != nil {
		updates["text"] = *req.Text
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&lic).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	result := licenseToModel(lic)
	return &result, nil
}

// DeleteLicense deletes a license.
func (s *LicenseService) DeleteLicense(req DeleteLicenseRequest) error {
	if req.LicenseID == 0 {
		return ErrInvalidLicenseID
	}

	var lic models.License
	if err := db.DB.First(&lic, req.LicenseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrLicenseNotFound
		}
		return err
	}

	return db.DB.Delete(&lic).Error
}

// Helper functions

func licenseToModel(l models.License) License {
	return License{
		ID:      l.ID,
		Key:     l.Key,
		Link:    l.Link,
		Name:    l.Name,
		Display: l.Display,
		Icon:    l.Icon,
		Text:    l.Text,
	}
}
