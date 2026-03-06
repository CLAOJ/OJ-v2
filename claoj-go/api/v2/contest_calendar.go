package v2

import (
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

// ContestCalendarRequest represents the calendar query parameters
type ContestCalendarRequest struct {
	Year  int `form:"year" binding:"required,min=2000,max=2100"`
	Month int `form:"month" binding:"required,min=1,max=12"`
}

// ContestCalendarItem represents a contest in the calendar view
type ContestCalendarItem struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	IsRated   bool      `json:"is_rated"`
	Format    string    `json:"format"`
	Day       int       `json:"day"` // Day of month (1-31)
}

// ContestCalendarResponse contains calendar data for a specific month
type ContestCalendarResponse struct {
	Year      int                 `json:"year"`
	Month     int                 `json:"month"`
	MonthName string              `json:"month_name"`
	DaysInMonth int               `json:"days_in_month"`
	FirstDayOfWeek int            `json:"first_day_of_week"` // 0=Sunday, 1=Monday, etc.
	Contests  []ContestCalendarItem `json:"contests"`
}

// ContestCalendar – GET /api/v2/contests/calendar
// Returns contest data for a specific month for calendar display
func ContestCalendar(c *gin.Context) {
	var req ContestCalendarRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid year or month"))
		return
	}

	// Calculate month boundaries
	year := req.Year
	month := time.Month(req.Month)

	// First day of the month
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)

	// Last day of the month (first day of next month minus 1 second)
	lastOfMonth := firstOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	// Get day of week for first day (0 = Sunday)
	firstDayOfWeek := int(firstOfMonth.Weekday())

	// Get contests that start or end within this month
	var contests []models.Contest

	// Use raw SQL to handle reserved word 'key'
	sql := `SELECT * FROM judge_contest
		WHERE is_visible = ?
		AND (
			(start_time >= ? AND start_time <= ?) OR
			(end_time >= ? AND end_time <= ?) OR
			(start_time <= ? AND end_time >= ?)
		)
		ORDER BY start_time ASC`

	args := []interface{}{
		true,
		firstOfMonth, lastOfMonth, // contests starting in month
		firstOfMonth, lastOfMonth, // contests ending in month
		firstOfMonth, lastOfMonth, // contests spanning entire month
	}

	if err := db.DB.Raw(sql, args...).Scan(&contests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Build calendar items
	items := make([]ContestCalendarItem, 0, len(contests))
	for _, ct := range contests {
		// Determine which day to show the contest on
		// If contest starts in this month, use start day
		// If contest started before this month but is ongoing, show on day 1
		day := 1
		if ct.StartTime.Month() == month && ct.StartTime.Year() == year {
			day = ct.StartTime.Day()
		} else if ct.StartTime.Before(firstOfMonth) && ct.EndTime.After(firstOfMonth) {
			day = 1 // Ongoing contest that started before this month
		} else if ct.StartTime.After(lastOfMonth) {
			continue // Contest starts after this month
		}

		items = append(items, ContestCalendarItem{
			Key:       ct.Key,
			Name:      ct.Name,
			StartTime: ct.StartTime,
			EndTime:   ct.EndTime,
			IsRated:   ct.IsRated,
			Format:    ct.FormatName,
			Day:       day,
		})
	}

	// Get month name
	monthName := month.String()

	response := ContestCalendarResponse{
		Year:           year,
		Month:          req.Month,
		MonthName:      monthName,
		DaysInMonth:    lastOfMonth.Day(),
		FirstDayOfWeek: firstDayOfWeek,
		Contests:       items,
	}

	c.JSON(http.StatusOK, response)
}
