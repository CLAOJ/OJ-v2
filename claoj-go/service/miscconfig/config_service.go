// Package miscconfig provides miscellaneous configuration management services.
package miscconfig

import (
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"gorm.io/gorm"
)

// MiscConfigService provides miscellaneous configuration management operations.
type MiscConfigService struct{}

// NewMiscConfigService creates a new MiscConfigService instance.
func NewMiscConfigService() *MiscConfigService {
	return &MiscConfigService{}
}

// ListConfig retrieves a paginated list of configurations.
func (s *MiscConfigService) ListConfig(req ListConfigRequest) (*ListConfigResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var configs []models.MiscConfig
	query := db.DB.Model(&models.MiscConfig{})

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
		Find(&configs).Error; err != nil {
		return nil, err
	}

	result := make([]MiscConfig, len(configs))
	for i, c := range configs {
		result[i] = configToModel(c)
	}

	return &ListConfigResponse{
		Configs:  result,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetConfig retrieves a configuration by ID.
func (s *MiscConfigService) GetConfig(req GetConfigRequest) (*MiscConfig, error) {
	if req.ConfigID == 0 {
		return nil, ErrInvalidConfigID
	}

	var config models.MiscConfig
	if err := db.DB.First(&config, req.ConfigID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	result := configToModel(config)
	return &result, nil
}

// GetConfigByKey retrieves a configuration by key.
func (s *MiscConfigService) GetConfigByKey(req GetConfigByKeyRequest) (*MiscConfig, error) {
	if req.Key == "" {
		return nil, ErrEmptyKey
	}

	var config models.MiscConfig
	if err := db.DB.Where("key = ?", req.Key).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	result := configToModel(config)
	return &result, nil
}

// CreateConfig creates a new configuration.
func (s *MiscConfigService) CreateConfig(req CreateConfigRequest) (*MiscConfig, error) {
	if req.Key == "" {
		return nil, ErrEmptyKey
	}

	// Check if key already exists
	var existing models.MiscConfig
	if err := db.DB.Where("key = ?", req.Key).First(&existing).Error; err == nil {
		return nil, ErrKeyExists
	}

	config := models.MiscConfig{
		Key:   req.Key,
		Value: req.Value,
	}

	if err := db.DB.Create(&config).Error; err != nil {
		return nil, err
	}

	result := configToModel(config)
	return &result, nil
}

// UpdateConfig updates an existing configuration by ID.
func (s *MiscConfigService) UpdateConfig(req UpdateConfigRequest) (*MiscConfig, error) {
	if req.ConfigID == 0 {
		return nil, ErrInvalidConfigID
	}

	var config models.MiscConfig
	if err := db.DB.First(&config, req.ConfigID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	if err := db.DB.Model(&config).Update("value", req.Value).Error; err != nil {
		return nil, err
	}

	result := configToModel(config)
	return &result, nil
}

// UpdateConfigByKey updates an existing configuration by key.
func (s *MiscConfigService) UpdateConfigByKey(req UpdateConfigByKeyRequest) (*MiscConfig, error) {
	if req.Key == "" {
		return nil, ErrEmptyKey
	}

	var config models.MiscConfig
	if err := db.DB.Where("key = ?", req.Key).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	if err := db.DB.Model(&config).Update("value", req.Value).Error; err != nil {
		return nil, err
	}

	result := configToModel(config)
	return &result, nil
}

// DeleteConfig deletes a configuration by ID.
func (s *MiscConfigService) DeleteConfig(req DeleteConfigRequest) error {
	if req.ConfigID == 0 {
		return ErrInvalidConfigID
	}

	var config models.MiscConfig
	if err := db.DB.First(&config, req.ConfigID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrConfigNotFound
		}
		return err
	}

	return db.DB.Delete(&config).Error
}

// DeleteConfigByKey deletes a configuration by key.
func (s *MiscConfigService) DeleteConfigByKey(req DeleteConfigByKeyRequest) error {
	if req.Key == "" {
		return ErrEmptyKey
	}

	var config models.MiscConfig
	if err := db.DB.Where("key = ?", req.Key).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrConfigNotFound
		}
		return err
	}

	return db.DB.Delete(&config).Error
}

// Helper functions

func configToModel(c models.MiscConfig) MiscConfig {
	return MiscConfig{
		ID:    c.ID,
		Key:   c.Key,
		Value: c.Value,
	}
}
