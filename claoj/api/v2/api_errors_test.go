package v2

import (
	"errors"
	"testing"

	"github.com/gin-gonic/gin/binding"
)

func TestBindErrorMessage(t *testing.T) {
	var req struct {
		Title     string `json:"title" binding:"required,max=5"`
		PublishOn string `json:"publish_on" binding:"required"`
	}
	req.Title = "toolong"
	req.PublishOn = "x"
	if got := bindErrorMessage(binding.Validator.ValidateStruct(&req)); got != "title must be at most 5 characters" {
		t.Errorf("max: got %q", got)
	}
	req.Title, req.PublishOn = "ok", ""
	if got := bindErrorMessage(binding.Validator.ValidateStruct(&req)); got != "publish on is required" {
		t.Errorf("required: got %q", got)
	}
	if got := bindErrorMessage(errors.New("unexpected EOF")); got != "invalid request data" {
		t.Errorf("other: got %q", got)
	}
}

func TestAPIClientError(t *testing.T) {
	for in, want := range map[string]string{
		"you are already a member of this organization":   "you are already a member of this organization",
		"blogpost: slug cannot be empty":                  "slug cannot be empty",
		"Error 1062 (23000): Duplicate entry 'x' for key": "An error occurred. Please try again.",
		"record not found":                                "An error occurred. Please try again.",
	} {
		if got := apiClientError(errors.New(in))["error"]; got != want {
			t.Errorf("apiClientError(%q) = %q, want %q", in, got, want)
		}
	}
}
