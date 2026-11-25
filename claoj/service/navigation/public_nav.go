package navigation

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
)

// PublicNavigation returns the navbar as a tree in the order v1 renders it:
// MPTT tree order (`tree_id`, `lft`), children nested under their parent, as
// `mptt_tree(nav_bar)` does in templates/base.html. v1 shows every
// judge_navigationbar row (NavigationBar.objects.all()), so nothing is
// filtered here.
func (s *NavigationService) PublicNavigation() ([]PublicNavigationItem, error) {
	var rows []models.NavigationBar
	if err := db.DB.Order("tree_id ASC, lft ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	nodes := make(map[uint]PublicNavigationItem, len(rows))
	for _, r := range rows {
		nodes[r.ID] = PublicNavigationItem{Key: r.Key, Label: r.Label, Path: r.Path, Regex: r.Regex}
	}

	children := make(map[uint][]uint, len(rows))
	var roots []uint
	for _, r := range rows {
		if r.ParentID == nil {
			roots = append(roots, r.ID)
			continue
		}
		if _, ok := nodes[*r.ParentID]; !ok {
			roots = append(roots, r.ID) // orphan: keep it reachable
			continue
		}
		children[*r.ParentID] = append(children[*r.ParentID], r.ID)
	}

	var build func(id uint) PublicNavigationItem
	build = func(id uint) PublicNavigationItem {
		item := nodes[id]
		for _, child := range children[id] {
			item.Children = append(item.Children, build(child))
		}
		return item
	}

	items := make([]PublicNavigationItem, 0, len(roots))
	for _, id := range roots {
		items = append(items, build(id))
	}
	return items, nil
}
