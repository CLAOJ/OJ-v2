package db

import "strings"

var likeEscaper = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")

// ContainsPattern returns a LIKE pattern that matches term anywhere, with
// LIKE wildcards in term matched literally. Always pair it with
// `LIKE ? ESCAPE '!'`: '!' (not '\') is the escape character that behaves
// the same in MySQL and in the SQLite test database.
func ContainsPattern(term string) string {
	return "%" + likeEscaper.Replace(term) + "%"
}
