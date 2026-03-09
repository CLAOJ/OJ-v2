// Package language provides language management services.
package language

import "errors"

// Service errors
var (
	ErrLanguageNotFound  = errors.New("language: language not found")
	ErrInvalidLanguageID = errors.New("language: invalid language ID")
	ErrEmptyLanguageKey  = errors.New("language: language key cannot be empty")
	ErrEmptyLanguageName = errors.New("language: language name cannot be empty")
	ErrLanguageInUse     = errors.New("language: cannot delete language with existing submissions")
	ErrLanguageKeyExists = errors.New("language: language key already exists")
)
