package sanitization

import (
	"strings"
	"testing"
)

func TestSanitizeComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSafe bool
	}{
		{
			name:     "safe HTML allowed",
			input:    "<p>Hello <strong>world</strong></p>",
			wantSafe: true,
		},
		{
			name:     "script tag removed",
			input:    "<script>alert('xss')</script>Hello",
			wantSafe: true,
		},
		{
			name:     "img onerror removed",
			input:    `<img src="x" onerror="alert('xss')">`,
			wantSafe: true,
		},
		{
			name:     "svg onload removed",
			input:    `<svg onload="alert('xss')"></svg>`,
			wantSafe: true,
		},
		{
			name:     "long comment truncated",
			input:    strings.Repeat("a", 15000),
			wantSafe: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeComment(tt.input)

			if tt.wantSafe {
				// Check for dangerous patterns
				if strings.Contains(strings.ToLower(result), "<script>") {
					t.Errorf("SanitizeComment() failed to remove script tag")
				}
				if strings.Contains(strings.ToLower(result), "onerror") {
					t.Errorf("SanitizeComment() failed to remove onerror handler")
				}
				if strings.Contains(strings.ToLower(result), "onload") {
					t.Errorf("SanitizeComment() failed to remove onload handler")
				}
			}

			// Check length limit
			if len(result) > MaxCommentLength {
				t.Errorf("SanitizeComment() result exceeds max length: got %d, want <= %d", len(result), MaxCommentLength)
			}
		})
	}
}

func TestSanitizeBlogContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "script tag removed",
			input: "<script>alert('xss')</script>",
		},
		{
			name:  "safe formatting allowed",
			input: "<p>This is <strong>bold</strong> and <em>italic</em></p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeBlogContent(tt.input)

			// Check for dangerous patterns
			if strings.Contains(strings.ToLower(result), "<script>") {
				t.Errorf("SanitizeBlogContent() failed to remove script tag")
			}

			// Check length limit
			if len(result) > MaxBlogLength {
				t.Errorf("SanitizeBlogContent() result exceeds max length")
			}
		})
	}
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "HTML stripped from title",
			input: "<script>alert('xss')</script>Test Title",
		},
		{
			name:  "Long title truncated",
			input: strings.Repeat("a", 300),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeTitle(tt.input)

			// Titles should have no HTML
			if strings.Contains(result, "<") {
				t.Errorf("SanitizeTitle() should strip all HTML: got %q", result)
			}

			// Check length limit
			if len(result) > 200 {
				t.Errorf("SanitizeTitle() result exceeds max length")
			}
		})
	}
}

func TestSanitizeTicketBody(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "script tag removed",
			input: "<script>alert('xss')</script>",
		},
		{
			name:  "long content truncated",
			input: strings.Repeat("a", 10000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeTicketBody(tt.input)

			if strings.Contains(strings.ToLower(result), "<script>") {
				t.Errorf("SanitizeTicketBody() failed to remove script tag")
			}

			if len(result) > MaxTicketLength {
				t.Errorf("SanitizeTicketBody() result exceeds max length")
			}
		})
	}
}
