// Package sanitization provides HTML sanitization for user-generated content.
package sanitization

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var (
	// StrictPolicy - No HTML allowed, for plain text
	StrictPolicy = bluemonday.StrictPolicy()

	// UGCPolicy - Safe HTML for comments, tickets, blogs
	// Allows basic formatting but no scripts or dangerous elements
	UGCPolicy = createUGCPolicy()

	// Content length limits
	MaxCommentLength = 10000
	MaxBlogLength    = 50000
	MaxTicketLength  = 5000
)

func createUGCPolicy() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()

	// Allow safe HTML elements for rich text
	policy.AllowElements(
		"p", "br", "strong", "b", "em", "i", "u", "s", "strike",
		"code", "pre", "blockquote",
		"ul", "ol", "li",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"a", "img",
		"table", "thead", "tbody", "tr", "th", "td",
		"div", "span", "hr",
	)

	// Allow specific attributes on safe elements
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")
	policy.AllowAttrs("class").OnElements("code", "pre", "span", "div")
	policy.AllowAttrs("target", "rel").OnElements("a")

	// Require rel="noopener noreferrer" on external links
	policy.RequireNoReferrerOnLinks(true)

	// Allow data URLs for images (for embedded images)
	policy.AllowDataURIImages()

	return policy
}

// SanitizeComment sanitizes comment body content
func SanitizeComment(input string) string {
	// Enforce length limit
	if len(input) > MaxCommentLength {
		input = input[:MaxCommentLength]
	}
	return UGCPolicy.Sanitize(strings.TrimSpace(input))
}

// SanitizeBlogContent sanitizes blog post content
func SanitizeBlogContent(input string) string {
	if len(input) > MaxBlogLength {
		input = input[:MaxBlogLength]
	}
	return UGCPolicy.Sanitize(strings.TrimSpace(input))
}

// SanitizeBlogSummary sanitizes blog summary (stricter, no HTML)
func SanitizeBlogSummary(input string) string {
	if len(input) > 500 {
		input = input[:500]
	}
	return StrictPolicy.Sanitize(strings.TrimSpace(input))
}

// SanitizeTicketBody sanitizes ticket message body
func SanitizeTicketBody(input string) string {
	if len(input) > MaxTicketLength {
		input = input[:MaxTicketLength]
	}
	return UGCPolicy.Sanitize(strings.TrimSpace(input))
}

// SanitizeTitle sanitizes titles (no HTML allowed)
func SanitizeTitle(input string) string {
	if len(input) > 200 {
		input = input[:200]
	}
	return StrictPolicy.Sanitize(strings.TrimSpace(input))
}

// SanitizeProblemContent sanitizes problem statement content
// Allows more HTML for problem statements including math
func SanitizeProblemContent(input string) string {
	if len(input) > 100000 {
		input = input[:100000]
	}

	// Create a more permissive policy for problem statements
	policy := bluemonday.UGCPolicy()
	policy.AllowElements(
		"p", "br", "strong", "b", "em", "i", "u",
		"code", "pre", "blockquote",
		"ul", "ol", "li",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"table", "thead", "tbody", "tr", "th", "td",
		"div", "span", "hr",
		"sup", "sub", // For mathematical notation
	)
	policy.AllowAttrs("class").OnElements("code", "pre", "span", "div")

	return policy.Sanitize(strings.TrimSpace(input))
}
