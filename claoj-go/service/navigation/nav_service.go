// Package navigation provides navigation bar management services.
package navigation

import (
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"gorm.io/gorm"
)

// NavigationService provides navigation bar management operations.
type NavigationService struct{}

// NewNavigationService creates a new NavigationService instance.
func NewNavigationService() *NavigationService {
	return &NavigationService{}
}

// ListNavigation retrieves a paginated list of navigation entries.
func (s *NavigationService) ListNavigation(req ListNavigationRequest) (*ListNavigationResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var entries []models.NavigationBar
	query := db.DB.Model(&models.NavigationBar{})

	if req.TreeID != nil {
		query = query.Where("tree_id = ?", *req.TreeID)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Order("tree_id ASC, lft ASC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&entries).Error; err != nil {
		return nil, err
	}

	result := make([]NavigationEntry, len(entries))
	for i, e := range entries {
		result[i] = navToModel(e)
	}

	return &ListNavigationResponse{
		Entries:  result,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetNavigation retrieves a navigation entry by ID.
func (s *NavigationService) GetNavigation(req GetNavigationRequest) (*NavigationEntry, error) {
	if req.NavID == 0 {
		return nil, ErrInvalidNavID
	}

	var nav models.NavigationBar
	if err := db.DB.First(&nav, req.NavID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNavNotFound
		}
		return nil, err
	}

	result := navToModel(nav)
	return &result, nil
}

// CreateNavigation creates a new navigation entry.
func (s *NavigationService) CreateNavigation(req CreateNavigationRequest) (*NavigationEntry, error) {
	if req.Key == "" {
		return nil, ErrEmptyKey
	}
	if req.Label == "" {
		return nil, ErrEmptyLabel
	}
	if req.Path == "" {
		return nil, ErrEmptyPath
	}

	// Check if key already exists
	var existing models.NavigationBar
	if err := db.DB.Where("key = ?", req.Key).First(&existing).Error; err == nil {
		return nil, ErrKeyExists
	}

	nav := models.NavigationBar{
		Key:      req.Key,
		Label:    req.Label,
		Path:     req.Path,
		ParentID: req.ParentID,
		Order:    req.Order,
	}

	if err := db.DB.Create(&nav).Error; err != nil {
		return nil, err
	}

	result := navToModel(nav)
	return &result, nil
}

// UpdateNavigation updates an existing navigation entry.
func (s *NavigationService) UpdateNavigation(req UpdateNavigationRequest) (*NavigationEntry, error) {
	if req.NavID == 0 {
		return nil, ErrInvalidNavID
	}

	var nav models.NavigationBar
	if err := db.DB.First(&nav, req.NavID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNavNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.Path != nil {
		updates["path"] = *req.Path
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}
	if req.Order != nil {
		updates["order"] = *req.Order
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&nav).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	result := navToModel(nav)
	return &result, nil
}

// DeleteNavigation deletes a navigation entry.
func (s *NavigationService) DeleteNavigation(req DeleteNavigationRequest) error {
	if req.NavID == 0 {
		return ErrInvalidNavID
	}

	var nav models.NavigationBar
	if err := db.DB.First(&nav, req.NavID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNavNotFound
		}
		return err
	}

	return db.DB.Delete(&nav).Error
}

// Helper functions

func navToModel(n models.NavigationBar) NavigationEntry {
	return NavigationEntry{
		ID:       n.ID,
		Key:      n.Key,
		Label:    n.Label,
		Path:     n.Path,
		ParentID: n.ParentID,
		Order:    n.Order,
		Lft:      n.Lft,
		Rght:     n.Rght,
		TreeID:   n.TreeID,
		Level:    n.Level,
	}
}
