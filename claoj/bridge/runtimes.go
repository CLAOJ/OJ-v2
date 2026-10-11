package bridge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// maxRuntimeField is the width of judge_runtimeversion.name/.version.
const maxRuntimeField = 64

// onJudgeRuntimes rewrites judge_runtimeversion for this judge from the
// executor list tiervnoj announced in its handshake. v1 parity: delete every
// row for the judge, then insert one row per entry of each executor key that
// matches a judge_language.key — version the components joined with ".",
// priority the entry's index within its language. Keys without a language
// row are skipped, not rejected. Delete-then-insert is what makes a
// replayed packet a no-op instead of a duplicate.
func onJudgeRuntimes(judge *models.Judge, pkt Packet) error {
	execs, ok := pkt["executors"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("%w: executors must be an object", ErrBadPacket)
	}

	langIDs, err := languageIDs(execs)
	if err != nil {
		return err
	}

	var rows []models.RuntimeVersion
	for key, raw := range execs {
		langID, known := langIDs[key]
		if !known {
			continue
		}
		entries, ok := raw.([]interface{})
		if !ok {
			continue
		}
		for idx, entry := range entries {
			name, version, ok := executorEntry(entry)
			if !ok {
				continue
			}
			rows = append(rows, models.RuntimeVersion{
				LanguageID: langID,
				JudgeID:    judge.ID,
				Name:       truncate(name, maxRuntimeField),
				Version:    truncate(version, maxRuntimeField),
				Priority:   idx,
			})
		}
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("judge_id = ?", judge.ID).Delete(&models.RuntimeVersion{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}

// languageIDs maps each announced executor key to judge_language.id; keys
// with no language row are simply absent from the result.
func languageIDs(execs map[string]interface{}) (map[string]uint, error) {
	keys := make([]string, 0, len(execs))
	for key := range execs {
		keys = append(keys, key)
	}
	ids := make(map[string]uint, len(keys))
	if len(keys) == 0 {
		return ids, nil
	}
	var langs []models.Language
	if err := db.DB.Where("key IN ?", keys).Find(&langs).Error; err != nil {
		return nil, err
	}
	for _, l := range langs {
		ids[l.Key] = l.ID
	}
	return ids, nil
}

// executorEntry reads one element of the handshake's executor list: either
// ["GCC 12", [12, 2, 0]] or a bare name string. Anything else is skipped.
func executorEntry(raw interface{}) (name, version string, ok bool) {
	switch e := raw.(type) {
	case string:
		return e, "", true
	case []interface{}:
		if len(e) != 2 {
			return "", "", false
		}
		name, ok = e[0].(string)
		if !ok {
			return "", "", false
		}
		return name, joinVersion(e[1]), true
	}
	return "", "", false
}

// joinVersion renders tiervnoj's version tuple the way v1 does:
// ".".join(map(str, version)) → "12.2.0".
func joinVersion(raw interface{}) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return v
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, c := range v {
			parts = append(parts, versionComponent(c))
		}
		return strings.Join(parts, ".")
	}
	return versionComponent(raw)
}

// versionComponent formats one version component, never in scientific
// notation ("1e+06"), which strconv would otherwise emit for large ints.
func versionComponent(c interface{}) string {
	switch v := c.(type) {
	case nil:
		return ""
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	}
	return fmt.Sprintf("%v", c)
}
