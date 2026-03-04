package contest_format

import (
	"github.com/CLAOJ/claoj-go/models"
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
