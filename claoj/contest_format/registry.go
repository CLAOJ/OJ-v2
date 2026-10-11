package contest_format

import (
	"fmt"

	"github.com/CLAOJ/claoj/models"
)

type Generator func(contest *models.Contest, config models.JSONField) ContestFormat

var registry = map[string]Generator{
	"default": NewDefaultFormat,
	"ioi16":   NewIOI16Format,
	"ioi":     NewIOI16Format, // Alias
	"icpc":    NewICPCFormat,
	"atcoder": NewAtCoderFormat,
	"ecoo":    NewECOOFormat,
}

// GetFormat returns a ContestFormat implementation based on the format name.
func GetFormat(name string, contest *models.Contest, config models.JSONField) ContestFormat {
	if gen, ok := registry[name]; ok {
		return gen(contest, config)
	}
	// Fallback to default
	return NewDefaultFormat(contest, config)
}

// Register adds a new contest format to the registry.
func Register(name string, gen Generator) {
	registry[name] = gen
}

// IsRegistered reports whether name is a known format. GetFormat silently
// falls back to "default", so validation has to ask explicitly.
func IsRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}

// configKeyType is the JSON type a config key must hold.
type configKeyType int

const (
	configNumber configKeyType = iota
	configBool
)

// allowedConfigKeys lists, per format, the config keys that format actually
// reads and the JSON type each must have. A format absent from this map
// (and any unregistered name that slips through) accepts no keys.
var allowedConfigKeys = map[string]map[string]configKeyType{
	"ioi16":   {"cumtime": configBool},
	"ioi":     {"cumtime": configBool},
	"icpc":    {"penalty": configNumber},
	"atcoder": {"penalty": configNumber},
	"ecoo":    {"first_ac_bonus": configNumber, "time_bonus": configNumber, "cumtime": configBool},
}

// ValidateConfig reports whether cfg is valid for the named format: every
// key must be one that format reads, holding the JSON type it expects.
func ValidateConfig(name string, cfg map[string]interface{}) error {
	allowed := allowedConfigKeys[name]
	for key, val := range cfg {
		want, ok := allowed[key]
		if !ok {
			return fmt.Errorf("contest_format: format %q has no config key %q", name, key)
		}
		switch want {
		case configNumber:
			if _, ok := val.(float64); !ok {
				return fmt.Errorf("contest_format: config key %q for format %q must be a number", key, name)
			}
		case configBool:
			if _, ok := val.(bool); !ok {
				return fmt.Errorf("contest_format: config key %q for format %q must be a boolean", key, name)
			}
		}
	}
	return nil
}
