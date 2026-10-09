// Package relations applies many-to-many edits for the admin update
// handlers and renders related users as {id, username} refs.
package relations

import (
	"errors"
	"fmt"

	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// ErrUnknownID is returned when a full list names an id that has no row,
// so a stale form or typo cannot silently drop a relation.
var ErrUnknownID = errors.New("unknown id")

// Sync updates owner's association `name` (rows of type T).
//
//   - full != nil: replace the association with exactly those ids; an
//     empty list clears it. add/remove are ignored.
//   - full == nil: apply the legacy add/remove lists.
//
// Run it inside the caller's transaction.
func Sync[T any](tx *gorm.DB, owner any, name string, full *[]uint, add, remove []uint) error {
	if full != nil {
		ids := unique(*full)
		if len(ids) == 0 {
			return tx.Model(owner).Association(name).Clear()
		}
		rows, err := find[T](tx, ids)
		if err != nil {
			return err
		}
		if len(rows) != len(ids) {
			return fmt.Errorf("%w in %s", ErrUnknownID, name)
		}
		return tx.Model(owner).Association(name).Replace(&rows)
	}
	if len(add) > 0 {
		rows, err := find[T](tx, unique(add))
		if err != nil {
			return err
		}
		if err := tx.Model(owner).Association(name).Append(&rows); err != nil {
			return err
		}
	}
	if len(remove) > 0 {
		rows, err := find[T](tx, unique(remove))
		if err != nil {
			return err
		}
		if err := tx.Model(owner).Association(name).Delete(&rows); err != nil {
			return err
		}
	}
	return nil
}

func find[T any](tx *gorm.DB, ids []uint) ([]T, error) {
	var rows []T
	err := tx.Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func unique(ids []uint) []uint {
	seen := make(map[uint]bool, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// UserRef is a related profile as the admin forms display it.
type UserRef struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// UserRefs maps profiles (preloaded with User) to refs; never nil.
func UserRefs(profiles []models.Profile) []UserRef {
	refs := make([]UserRef, len(profiles))
	for i, p := range profiles {
		refs[i] = UserRef{ID: p.ID, Username: p.User.Username}
	}
	return refs
}
