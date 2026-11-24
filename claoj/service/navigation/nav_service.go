// Package navigation provides navigation bar management services.
package navigation

import (
	"fmt"
	"regexp"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
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
	query := db.DB.Model(&models.NavigationBar{}).Preload("Parent")

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
	if err := db.DB.Preload("Parent").First(&nav, req.NavID).Error; err != nil {
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
	if err := validateRegex(req.Regex); err != nil {
		return nil, err
	}

	var created models.NavigationBar
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// Check if key already exists
		var existing models.NavigationBar
		if err := tx.Where("`key` = ?", req.Key).First(&existing).Error; err == nil {
			return ErrKeyExists
		}

		if req.ParentID != nil {
			if err := requireNavExists(tx, *req.ParentID); err != nil {
				return err
			}
		}

		created = models.NavigationBar{
			Key:      req.Key,
			Label:    req.Label,
			Path:     req.Path,
			Regex:    req.Regex,
			ParentID: req.ParentID,
			Order:    req.Order,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		return rebuildTree(tx)
	})
	if err != nil {
		return nil, err
	}

	return s.GetNavigation(GetNavigationRequest{NavID: created.ID})
}

// UpdateNavigation updates an existing navigation entry.
func (s *NavigationService) UpdateNavigation(req UpdateNavigationRequest) (*NavigationEntry, error) {
	if req.NavID == 0 {
		return nil, ErrInvalidNavID
	}
	if req.Regex != nil {
		if err := validateRegex(*req.Regex); err != nil {
			return nil, err
		}
	}
	if req.Label != nil && *req.Label == "" {
		return nil, ErrEmptyLabel
	}
	if req.Path != nil && *req.Path == "" {
		return nil, ErrEmptyPath
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var nav models.NavigationBar
		if err := tx.First(&nav, req.NavID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrNavNotFound
			}
			return err
		}

		if req.SetParent && req.ParentID != nil {
			if err := checkParentMove(tx, req.NavID, *req.ParentID); err != nil {
				return err
			}
		}

		updates := make(map[string]interface{})
		if req.Label != nil {
			updates["label"] = *req.Label
		}
		if req.Path != nil {
			updates["path"] = *req.Path
		}
		if req.Regex != nil {
			updates["regex"] = *req.Regex
		}
		if req.SetParent {
			// nil moves the entry to the root (parent_id NULL).
			if req.ParentID == nil {
				updates["parent_id"] = nil
			} else {
				updates["parent_id"] = *req.ParentID
			}
		}
		if req.Order != nil {
			updates["order"] = *req.Order
		}

		if len(updates) == 0 {
			return nil
		}
		if err := tx.Model(&nav).Updates(updates).Error; err != nil {
			return err
		}
		return rebuildTree(tx)
	})
	if err != nil {
		return nil, err
	}

	return s.GetNavigation(GetNavigationRequest{NavID: req.NavID})
}

// DeleteNavigation deletes a navigation entry. Children are deleted with it,
// mirroring v1's TreeForeignKey(on_delete=CASCADE).
func (s *NavigationService) DeleteNavigation(req DeleteNavigationRequest) error {
	if req.NavID == 0 {
		return ErrInvalidNavID
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		var nav models.NavigationBar
		if err := tx.First(&nav, req.NavID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrNavNotFound
			}
			return err
		}

		ids, err := subtreeIDs(tx, req.NavID)
		if err != nil {
			return err
		}
		// Django's CASCADE runs in Python; at the DB level the self-FK is
		// RESTRICT, so a single multi-row DELETE trips InnoDB 1451 in
		// arbitrary row order. subtreeIDs is breadth-first (parents before
		// children), so delete in reverse: leaves first.
		for i := len(ids) - 1; i >= 0; i-- {
			if err := tx.Delete(&models.NavigationBar{}, ids[i]).Error; err != nil {
				return err
			}
		}
		return rebuildTree(tx)
	})
}

// Helper functions

func navToModel(n models.NavigationBar) NavigationEntry {
	e := NavigationEntry{
		ID:       n.ID,
		Key:      n.Key,
		Label:    n.Label,
		Path:     n.Path,
		Regex:    n.Regex,
		ParentID: n.ParentID,
		Order:    n.Order,
		Lft:      n.Lft,
		Rght:     n.Rght,
		TreeID:   n.TreeID,
		Level:    n.Level,
	}
	if n.Parent != nil {
		e.ParentLabel = n.Parent.Label
	}
	return e
}

// validateRegex rejects patterns that do not compile. The empty pattern is
// allowed: v1 stores NOT NULL with an empty default meaning "never match".
func validateRegex(pattern string) error {
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRegex, err)
	}
	return nil
}

func requireNavExists(tx *gorm.DB, id uint) error {
	var n int64
	if err := tx.Model(&models.NavigationBar{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return ErrParentNotFound
	}
	return nil
}

// checkParentMove rejects a parent that is the entry itself or one of its
// descendants, which would detach a cycle from the tree.
func checkParentMove(tx *gorm.DB, navID, parentID uint) error {
	if navID == parentID {
		return ErrParentCycle
	}
	if err := requireNavExists(tx, parentID); err != nil {
		return err
	}
	seen := map[uint]bool{}
	for cur := parentID; ; {
		var node models.NavigationBar
		if err := tx.Select("id", "parent_id").First(&node, cur).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrParentNotFound
			}
			return err
		}
		if node.ParentID == nil {
			return nil
		}
		if *node.ParentID == navID {
			return ErrParentCycle
		}
		if seen[cur] {
			return ErrParentCycle
		}
		seen[cur] = true
		cur = *node.ParentID
	}
}

// subtreeIDs returns the entry ID plus every descendant ID.
func subtreeIDs(tx *gorm.DB, rootID uint) ([]uint, error) {
	ids := []uint{rootID}
	seen := map[uint]bool{rootID: true}
	frontier := []uint{rootID}
	for len(frontier) > 0 {
		var next []uint
		if err := tx.Model(&models.NavigationBar{}).
			Where("parent_id IN ?", frontier).
			Pluck("id", &next).Error; err != nil {
			return nil, err
		}
		frontier = nil
		for _, id := range next {
			if seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
			frontier = append(frontier, id)
		}
	}
	return ids, nil
}

// rebuildTree recomputes lft/rght/level/tree_id for every row from
// parent_id/order. The table is tiny, so a deterministic DFS is simpler and
// safer than incremental nested-set shifts, which corrupt on reorder/move.
func rebuildTree(tx *gorm.DB) error {
	var nodes []models.NavigationBar
	if err := tx.Order("`order` ASC, id ASC").Find(&nodes).Error; err != nil {
		return err
	}

	index := make(map[uint]int, len(nodes))
	for i, n := range nodes {
		index[n.ID] = i
	}

	children := make(map[uint][]int, len(nodes))
	var roots []int
	for i, n := range nodes {
		if n.ParentID == nil {
			roots = append(roots, i)
			continue
		}
		if _, ok := index[*n.ParentID]; !ok {
			roots = append(roots, i) // orphan: keep it addressable as a root
			continue
		}
		children[*n.ParentID] = append(children[*n.ParentID], i)
	}

	visited := make([]bool, len(nodes))
	counter, treeID := 1, 0 // 1-based lft/rght, as Django MPTT writes them
	var walk func(i, level int)
	walk = func(i, level int) {
		if visited[i] {
			return
		}
		visited[i] = true
		nodes[i].Lft = counter
		nodes[i].Level = level
		nodes[i].TreeID = treeID
		counter++
		for _, c := range children[nodes[i].ID] {
			walk(c, level+1)
		}
		nodes[i].Rght = counter
		counter++
	}
	for _, r := range roots {
		treeID++
		walk(r, 0)
	}
	// Cycles in legacy data: hang whatever is left off a fresh root so every
	// row still gets a valid lft/rght pair.
	for i := range nodes {
		if !visited[i] {
			treeID++
			walk(i, 0)
		}
	}

	for _, n := range nodes {
		if err := tx.Model(&models.NavigationBar{}).
			Where("id = ?", n.ID).
			Updates(map[string]interface{}{
				"lft":     n.Lft,
				"rght":    n.Rght,
				"level":   n.Level,
				"tree_id": n.TreeID,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}
