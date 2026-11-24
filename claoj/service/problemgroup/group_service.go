// Package problemgroup provides problem group management services.
package problemgroup

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// ProblemGroupService provides problem group management operations.
type ProblemGroupService struct{}

// NewProblemGroupService creates a new ProblemGroupService instance.
func NewProblemGroupService() *ProblemGroupService {
	return &ProblemGroupService{}
}

// ListGroups retrieves a list of all problem groups.
func (s *ProblemGroupService) ListGroups(req ListGroupsRequest) (*ListGroupsResponse, error) {
	query := db.DB.Model(&models.ProblemGroup{})
	if req.Search != "" {
		p := db.ContainsPattern(req.Search)
		query = query.Where("(name LIKE ? ESCAPE '!' OR full_name LIKE ? ESCAPE '!')", p, p)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var groups []models.ProblemGroup
	if err := query.Order("name ASC").Find(&groups).Error; err != nil {
		return nil, err
	}

	result := make([]ProblemGroup, len(groups))
	for i, g := range groups {
		result[i] = groupToModel(g)
	}

	return &ListGroupsResponse{
		Groups: result,
		Total:  total,
	}, nil
}

// GetGroup retrieves a problem group by ID.
func (s *ProblemGroupService) GetGroup(req GetGroupRequest) (*ProblemGroup, error) {
	if req.GroupID == 0 {
		return nil, ErrInvalidGroupID
	}

	var group models.ProblemGroup
	if err := db.DB.First(&group, req.GroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	result := groupToModel(group)
	if err := db.DB.Model(&models.Problem{}).Where("group_id = ?", group.ID).Count(&result.ProblemCount).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateGroup creates a new problem group.
func (s *ProblemGroupService) CreateGroup(req CreateGroupRequest) (*ProblemGroup, error) {
	if req.Name == "" {
		return nil, ErrEmptyGroupName
	}
	if req.FullName == "" {
		return nil, ErrEmptyGroupFullName
	}

	// Check if name already exists
	var existing models.ProblemGroup
	if err := db.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, ErrGroupNameExists
	}

	group := models.ProblemGroup{
		Name:     req.Name,
		FullName: req.FullName,
	}

	if err := db.DB.Create(&group).Error; err != nil {
		return nil, err
	}

	result := groupToModel(group)
	return &result, nil
}

// UpdateGroup updates an existing problem group.
func (s *ProblemGroupService) UpdateGroup(req UpdateGroupRequest) (*ProblemGroup, error) {
	if req.GroupID == 0 {
		return nil, ErrInvalidGroupID
	}

	var group models.ProblemGroup
	if err := db.DB.First(&group, req.GroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrGroupNotFound
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
		if err := db.DB.Model(&group).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	result := groupToModel(group)
	return &result, nil
}

// DeleteGroup deletes a problem group.
func (s *ProblemGroupService) DeleteGroup(req DeleteGroupRequest) error {
	if req.GroupID == 0 {
		return ErrInvalidGroupID
	}

	var group models.ProblemGroup
	if err := db.DB.First(&group, req.GroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrGroupNotFound
		}
		return err
	}

	return db.DB.Delete(&group).Error
}

// Helper functions

func groupToModel(g models.ProblemGroup) ProblemGroup {
	return ProblemGroup{
		ID:       g.ID,
		Name:     g.Name,
		FullName: g.FullName,
	}
}
