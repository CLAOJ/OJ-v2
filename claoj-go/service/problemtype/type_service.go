// Package problemtype provides problem type management services.
package problemtype

import (
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"gorm.io/gorm"
)

// ProblemTypeService provides problem type management operations.
type ProblemTypeService struct{}

// NewProblemTypeService creates a new ProblemTypeService instance.
func NewProblemTypeService() *ProblemTypeService {
	return &ProblemTypeService{}
}

// ListTypes retrieves a list of all problem types.
func (s *ProblemTypeService) ListTypes() (*ListTypesResponse, error) {
	var types []models.ProblemType
	if err := db.DB.Order("name ASC").Find(&types).Error; err != nil {
		return nil, err
	}

	var total int64
	if err := db.DB.Model(&models.ProblemType{}).Count(&total).Error; err != nil {
		return nil, err
	}

	result := make([]ProblemType, len(types))
	for i, t := range types {
		result[i] = typeToModel(t)
	}

	return &ListTypesResponse{
		Types: result,
		Total: total,
	}, nil
}

// GetType retrieves a problem type by ID.
func (s *ProblemTypeService) GetType(req GetTypeRequest) (*ProblemType, error) {
	if req.TypeID == 0 {
		return nil, ErrInvalidTypeID
	}

	var t models.ProblemType
	if err := db.DB.First(&t, req.TypeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTypeNotFound
		}
		return nil, err
	}

	result := typeToModel(t)
	return &result, nil
}

// CreateType creates a new problem type.
func (s *ProblemTypeService) CreateType(req CreateTypeRequest) (*ProblemType, error) {
	if req.Name == "" {
		return nil, ErrEmptyTypeName
	}
	if req.FullName == "" {
		return nil, ErrEmptyTypeFullName
	}

	// Check if name already exists
	var existing models.ProblemType
	if err := db.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, ErrTypeNameExists
	}

	t := models.ProblemType{
		Name:     req.Name,
		FullName: req.FullName,
	}

	if err := db.DB.Create(&t).Error; err != nil {
		return nil, err
	}

	result := typeToModel(t)
	return &result, nil
}

// UpdateType updates an existing problem type.
func (s *ProblemTypeService) UpdateType(req UpdateTypeRequest) (*ProblemType, error) {
	if req.TypeID == 0 {
		return nil, ErrInvalidTypeID
	}

	var t models.ProblemType
	if err := db.DB.First(&t, req.TypeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTypeNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.FullName != nil {
		updates["full_name"] = *req.FullName
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&t).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	result := typeToModel(t)
	return &result, nil
}

// DeleteType deletes a problem type.
func (s *ProblemTypeService) DeleteType(req DeleteTypeRequest) error {
	if req.TypeID == 0 {
		return ErrInvalidTypeID
	}

	var t models.ProblemType
	if err := db.DB.First(&t, req.TypeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrTypeNotFound
		}
		return err
	}

	return db.DB.Delete(&t).Error
}

// Helper functions

func typeToModel(t models.ProblemType) ProblemType {
	return ProblemType{
		ID:       t.ID,
		Name:     t.Name,
		FullName: t.FullName,
	}
}
