package v2

import (
	"encoding/xml"
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// RSS Feed structures
type RSSFeed struct {
	XMLName        xml.Name    `xml:"rss"`
	Version        string      `xml:"version,attr"`
	Channel        RSSChannel  `xml:"channel"`
}

type RSSChannel struct {
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	Description string      `xml:"description"`
	Language    string      `xml:"language,omitempty"`
	LastBuild   string      `xml:"lastBuildDate,omitempty"`
	Items       []RSSItem   `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid,omitempty"`
	Author      string `xml:"author,omitempty"`
}

// Atom Feed structures
type AtomFeed struct {
	XMLName     xml.Name     `xml:"feed"`
	Xmlns       string       `xml:"xmlns,attr"`
	Title       string       `xml:"title"`
	Link        AtomLink     `xml:"link"`
	Updated     string       `xml:"updated"`
	ID          string       `xml:"id"`
	Description string       `xml:"subtitle,omitempty"`
	Entries     []AtomEntry  `xml:"entry"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
}

type AtomEntry struct {
	Title       string   `xml:"title"`
	Link        AtomLink `xml:"link"`
	ID          string   `xml:"id"`
	Updated     string   `xml:"updated"`
	Published   string   `xml:"published,omitempty"`
	Author      AtomAuthor `xml:"author,omitempty"`
	Summary     string   `xml:"summary,omitempty"`
	Content     string   `xml:"content,omitempty"`
}

type AtomAuthor struct {
	Name string `xml:"name"`
}

const (
	baseURL = "https://beta.claoj.edu.vn"
	feedTitle = "CLAOJ - Competitive Programming Online Judge"
)

// ProblemFeedRSS - GET /api/v2/problems/feed/rss
// Returns RSS feed for recent problems
func ProblemFeedRSS(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var problems []models.Problem
	if err := db.DB.
		Preload("Group").
		Where("is_public = ?", true).
		Order("date DESC").
		Limit(limit).
		Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	items := make([]RSSItem, 0, len(problems))
	for _, p := range problems {
		groupName := getProblemGroupName(p.Group)
		item := RSSItem{
			Title:   p.Name,
			Link:    baseURL + "/problems/" + p.Code,
			PubDate: p.Date.Format(time.RFC1123Z),
			GUID:    baseURL + "/problems/" + p.Code,
			Description: "[" + groupName + "] " + p.Name,
		}
		items = append(items, item)
	}

	feed := RSSFeed{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       feedTitle,
			Link:        baseURL + "/problems",
			Description: "Recent problems on CLAOJ Online Judge",
			Language:    "en-us",
			LastBuild:   time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	}

	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	c.XML(http.StatusOK, feed)
}

// ProblemFeedAtom - GET /api/v2/problems/feed/atom
// Returns Atom feed for recent problems
func ProblemFeedAtom(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var problems []models.Problem
	if err := db.DB.
		Preload("Group").
		Where("is_public = ?", true).
		Order("date DESC").
		Limit(limit).
		Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	entries := make([]AtomEntry, 0, len(problems))
	for _, p := range problems {
		groupName := getProblemGroupName(p.Group)
		entry := AtomEntry{
			Title: p.Name,
			Link: AtomLink{
				Href: baseURL + "/problems/" + p.Code,
				Rel:  "alternate",
			},
			ID:        baseURL + "/problems/" + p.Code,
			Updated:   p.Date.Format(time.RFC3339),
			Published: p.Date.Format(time.RFC3339),
			Summary:   "[" + groupName + "] " + p.Name,
		}
		entries = append(entries, entry)
	}

	feed := AtomFeed{
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: feedTitle,
		Link: AtomLink{
			Href: baseURL + "/problems",
			Rel:  "alternate",
		},
		Updated:     time.Now().Format(time.RFC3339),
		ID:          baseURL + "/problems",
		Description: "Recent problems on CLAOJ Online Judge",
		Entries:     entries,
	}

	c.Header("Content-Type", "application/atom+xml; charset=utf-8")
	c.XML(http.StatusOK, feed)
}

// CommentFeedRSS - GET /api/v2/comments/feed/rss
// Returns RSS feed for recent comments
func CommentFeedRSS(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var comments []models.Comment
	if err := db.DB.
		Preload("Author.User").
		Where("hidden = ?", false).
		Order("time DESC").
		Limit(limit).
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	items := make([]RSSItem, 0, len(comments))
	for _, cmt := range comments {
		username := ""
		if cmt.Author.UserID != 0 {
			username = cmt.Author.User.Username
		}

		item := RSSItem{
			Title:   "Comment by " + username + " on " + cmt.Page,
			Link:    baseURL + cmt.Page + "#comment-" + strconv.Itoa(int(cmt.ID)),
			PubDate: cmt.Time.Format(time.RFC1123Z),
			GUID:    baseURL + "/comments/" + strconv.Itoa(int(cmt.ID)),
			Author:  username,
			Description: truncateString(stripHTML(cmt.Body), 500),
		}
		items = append(items, item)
	}

	feed := RSSFeed{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       "CLAOJ - Recent Comments",
			Link:        baseURL + "/comments",
			Description: "Recent comments on CLAOJ",
			Language:    "en-us",
			LastBuild:   time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	}

	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	c.XML(http.StatusOK, feed)
}

// CommentFeedAtom - GET /api/v2/comments/feed/atom
// Returns Atom feed for recent comments
func CommentFeedAtom(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var comments []models.Comment
	if err := db.DB.
		Preload("Author.User").
		Where("hidden = ?", false).
		Order("time DESC").
		Limit(limit).
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	entries := make([]AtomEntry, 0, len(comments))
	for _, cmt := range comments {
		username := "Anonymous"
		if cmt.Author.UserID != 0 {
			username = cmt.Author.User.Username
		}

		entry := AtomEntry{
			Title: "Comment by " + username + " on " + cmt.Page,
			Link: AtomLink{
				Href: baseURL + cmt.Page + "#comment-" + strconv.Itoa(int(cmt.ID)),
			},
			ID:        baseURL + "/comments/" + strconv.Itoa(int(cmt.ID)),
			Updated:   cmt.Time.Format(time.RFC3339),
			Published: cmt.Time.Format(time.RFC3339),
			Author: AtomAuthor{
				Name: username,
			},
			Summary: truncateString(stripHTML(cmt.Body), 500),
		}
		entries = append(entries, entry)
	}

	feed := AtomFeed{
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: "CLAOJ - Recent Comments",
		Link: AtomLink{
			Href: baseURL + "/comments",
			Rel:  "alternate",
		},
		Updated:     time.Now().Format(time.RFC3339),
		ID:          baseURL + "/comments",
		Description: "Recent comments on CLAOJ",
		Entries:     entries,
	}

	c.Header("Content-Type", "application/atom+xml; charset=utf-8")
	c.XML(http.StatusOK, feed)
}

// BlogFeedRSS - GET /api/v2/blogs/feed/rss
// Returns RSS feed for recent blog posts
func BlogFeedRSS(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var posts []models.BlogPost
	if err := db.DB.
		Preload("Author.User").
		Where("visible = ? AND publish_on <= ?", true, time.Now()).
		Order("publish_on DESC").
		Limit(limit).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	items := make([]RSSItem, 0, len(posts))
	for _, post := range posts {
		username := ""
		if post.Author.UserID != 0 {
			username = post.Author.User.Username
		}

		item := RSSItem{
			Title:   post.Title,
			Link:    baseURL + "/blog/" + strconv.Itoa(int(post.ID)),
			PubDate: post.PublishOn.Format(time.RFC1123Z),
			GUID:    baseURL + "/blog/" + strconv.Itoa(int(post.ID)),
			Author:  username,
			Description: truncateString(stripHTML(post.Content), 500),
		}
		items = append(items, item)
	}

	feed := RSSFeed{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       "CLAOJ - Blog Posts",
			Link:        baseURL + "/blog",
			Description: "Recent blog posts on CLAOJ",
			Language:    "en-us",
			LastBuild:   time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	}

	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	c.XML(http.StatusOK, feed)
}

// BlogFeedAtom - GET /api/v2/blogs/feed/atom
// Returns Atom feed for recent blog posts
func BlogFeedAtom(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var posts []models.BlogPost
	if err := db.DB.
		Preload("Author.User").
		Where("visible = ? AND publish_on <= ?", true, time.Now()).
		Order("publish_on DESC").
		Limit(limit).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	entries := make([]AtomEntry, 0, len(posts))
	for _, post := range posts {
		username := ""
		if post.Author.UserID != 0 {
			username = post.Author.User.Username
		}

		entry := AtomEntry{
			Title: post.Title,
			Link: AtomLink{
				Href: baseURL + "/blog/" + strconv.Itoa(int(post.ID)),
				Rel:  "alternate",
			},
			ID:        baseURL + "/blog/" + strconv.Itoa(int(post.ID)),
			Updated:   post.PublishOn.Format(time.RFC3339),
			Published: post.PublishOn.Format(time.RFC3339),
			Author: AtomAuthor{
				Name: username,
			},
			Summary: truncateString(stripHTML(post.Content), 500),
		}
		entries = append(entries, entry)
	}

	feed := AtomFeed{
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: "CLAOJ - Blog Posts",
		Link: AtomLink{
			Href: baseURL + "/blog",
			Rel:  "alternate",
		},
		Updated:     time.Now().Format(time.RFC3339),
		ID:          baseURL + "/blog",
		Description: "Recent blog posts on CLAOJ",
		Entries:     entries,
	}

	c.Header("Content-Type", "application/atom+xml; charset=utf-8")
	c.XML(http.StatusOK, feed)
}

// Helper functions

func getProblemGroupName(group models.ProblemGroup) string {
	if group.FullName != "" {
		return group.FullName
	}
	return group.Name
}

func stripHTML(html string) string {
	// Simple HTML tag removal
	result := html
	// Remove common HTML tags
	tags := []string{"<p>", "</p>", "<br>", "<br/>", "<br />", "<div>", "</div>", "<span>", "</span>"}
	for _, tag := range tags {
		result = replaceAll(result, tag, " ")
	}
	// Remove any remaining tags (simple approach)
	clean := ""
	inTag := false
	for _, ch := range result {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
		} else if !inTag {
			clean += string(ch)
		}
	}
	return clean
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func replaceAll(s, old, new string) string {
	result := s
	for stringContains(result, old) {
		result = replaceFirst(result, old, new)
	}
	return result
}

func replaceFirst(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}
