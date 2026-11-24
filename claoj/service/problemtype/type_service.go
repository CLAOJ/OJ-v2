// Package problemtype provides problem type management services.
package problemtype

import (
	"fmt"
	"sort"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// ProblemTypeService provides problem type management operations.
type ProblemTypeService struct{}

// NewProblemTypeService creates a new ProblemTypeService instance.
func NewProblemTypeService() *ProblemTypeService {
	return &ProblemTypeService{}
}

// ListTypes retrieves a list of all problem types.
func (s *ProblemTypeService) ListTypes(req ListTypesRequest) (*ListTypesResponse, error) {
	query := db.DB.Model(&models.ProblemType{})
	if req.Search != "" {
		p := db.ContainsPattern(req.Search)
		query = query.Where("(name LIKE ? ESCAPE '!' OR full_name LIKE ? ESCAPE '!')", p, p)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var types []models.ProblemType
	if err := query.Order("name ASC").Find(&types).Error; err != nil {
		return nil, err
	}

	ids := make([]uint, len(types))
	for i, t := range types {
		ids[i] = t.ID
	}
	counts, err := problemCounts(ids)
	if err != nil {
		return nil, err
	}

	result := make([]ProblemType, len(types))
	for i, t := range types {
		result[i] = typeToModel(t)
		result[i].ProblemCount = counts[t.ID]
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
	if err := typeProblemIDs(db.DB, t.ID, &result.ProblemIDs); err != nil {
		return nil, err
	}
	result.ProblemCount = int64(len(result.ProblemIDs))
	return &result, nil
}

// GetTypeProblems returns the id/code/name of every problem included in a type.
func (s *ProblemTypeService) GetTypeProblems(typeID uint) ([]ProblemRef, error) {
	var refs []ProblemRef
	err := db.DB.Table("judge_problem").
		Select("judge_problem.id, judge_problem.code, judge_problem.name").
		Joins("JOIN judge_problem_types ON judge_problem_types.problem_id = judge_problem.id").
		Where("judge_problem_types.problemtype_id = ?", typeID).
		Order("judge_problem.code ASC").
		Scan(&refs).Error
	return refs, err
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
	var problemIDs []uint

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&t).Error; err != nil {
			return err
		}
		if req.ProblemIDs != nil {
			ids, err := existingProblemIDs(tx, *req.ProblemIDs)
			if err != nil {
				return err
			}
			problemIDs = ids
			return replaceTypeProblems(tx, t.ID, ids)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	result := typeToModel(t)
	if req.ProblemIDs != nil {
		result.ProblemIDs = problemIDs
		result.ProblemCount = int64(len(problemIDs))
	}
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
	var problemIDs []uint

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&t).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.ProblemIDs != nil {
			ids, err := existingProblemIDs(tx, *req.ProblemIDs)
			if err != nil {
				return err
			}
			problemIDs = ids
			return replaceTypeProblems(tx, t.ID, ids)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	result := typeToModel(t)
	if req.ProblemIDs != nil {
		result.ProblemIDs = problemIDs
		result.ProblemCount = int64(len(problemIDs))
	}
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

	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM judge_problem_types WHERE problemtype_id = ?", t.ID).Error; err != nil {
			return err
		}
		return tx.Delete(&t).Error
	})
}

// Helper functions

func typeToModel(t models.ProblemType) ProblemType {
	return ProblemType{
		ID:       t.ID,
		Name:     t.Name,
		FullName: t.FullName,
	}
}

// uniqueIDs drops duplicate ids, keeping the request order.
func uniqueIDs(ids []uint) []uint {
	seen := make(map[uint]bool, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// existingProblemIDs validates that every problem id exists and returns the
// de-duplicated list.
func existingProblemIDs(tx *gorm.DB, ids []uint) ([]uint, error) {
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return ids, nil
	}

	var found []uint
	if err := tx.Model(&models.Problem{}).Where("id IN ?", ids).Pluck("id", &found).Error; err != nil {
		return nil, err
	}
	have := make(map[uint]bool, len(found))
	for _, id := range found {
		have[id] = true
	}
	var missing []uint
	for _, id := range ids {
		if !have[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
		return nil, fmt.Errorf("%w: %v", ErrUnknownProblemIDs, missing)
	}
	return ids, nil
}

// replaceTypeProblems rewrites judge_problem_types for one type only.
func replaceTypeProblems(tx *gorm.DB, typeID uint, ids []uint) error {
	if err := tx.Exec("DELETE FROM judge_problem_types WHERE problemtype_id = ?", typeID).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := tx.Exec("INSERT INTO judge_problem_types (problem_id, problemtype_id) VALUES (?, ?)", id, typeID).Error; err != nil {
			return err
		}
	}
	return nil
}

func typeProblemIDs(tx *gorm.DB, typeID uint, out *[]uint) error {
	return tx.Table("judge_problem_types").
		Where("problemtype_id = ?", typeID).
		Order("problem_id ASC").
		Pluck("problem_id", out).Error
}

// problemCounts maps type id -> number of included problems.
func problemCounts(typeIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(typeIDs))
	if len(typeIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		ProblemTypeID uint  `gorm:"column:problemtype_id"`
		Count         int64 `gorm:"column:count"`
	}
	if err := db.DB.Table("judge_problem_types").
		Select("problemtype_id, COUNT(*) AS count").
		Where("problemtype_id IN ?", typeIDs).
		Group("problemtype_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		counts[r.ProblemTypeID] = r.Count
	}
	return counts, nil
}