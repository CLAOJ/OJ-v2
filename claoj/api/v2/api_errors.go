package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// apiBindError turns a request-binding failure (ShouldBind*, json.Unmarshal)
// into a message naming the field, e.g. "publish on is required", instead of
// the generic "An error occurred".
func apiBindError(err error) gin.H {
	log.Printf("API bind error: %v", err)
	return gin.H{"error": bindErrorMessage(err)}
}

func bindErrorMessage(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		f := ve[0]
		name := fieldLabel(f.Field())
		unit := ""
		switch f.Kind().String() {
		case "string":
			unit = " characters"
		case "slice", "array", "map":
			unit = " items"
		}
		switch f.Tag() {
		case "required":
			return name + " is required"
		case "min", "gte":
			return fmt.Sprintf("%s must be at least %s%s", name, f.Param(), unit)
		case "max", "lte":
			return fmt.Sprintf("%s must be at most %s%s", name, f.Param(), unit)
		case "gt":
			return fmt.Sprintf("%s must be greater than %s%s", name, f.Param(), unit)
		case "lt":
			return fmt.Sprintf("%s must be less than %s%s", name, f.Param(), unit)
		case "len":
			return fmt.Sprintf("%s must be exactly %s%s", name, f.Param(), unit)
		case "email":
			return name + " must be a valid email address"
		case "url":
			return name + " must be a valid URL"
		case "oneof":
			return name + " must be one of: " + strings.ReplaceAll(f.Param(), " ", ", ")
		}
		return name + " is invalid"
	}
	var te *json.UnmarshalTypeError
	if errors.As(err, &te) && te.Field != "" {
		return fieldLabel(te.Field) + " has the wrong type (expected " + te.Type.String() + ")"
	}
	return "invalid request data"
}

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// fieldLabel: "PublishOn" -> "publish on", "author_ids" -> "author ids".
func fieldLabel(field string) string {
	if i := strings.LastIndex(field, "."); i >= 0 {
		field = field[i+1:]
	}
	field = camelBoundary.ReplaceAllString(field, "$1 $2")
	return strings.ToLower(strings.ReplaceAll(field, "_", " "))
}

// internalError matches error text that comes from the database or the
// network rather than from our own validation.
var internalError = regexp.MustCompile(`(?i)error \d{4}|sql|gorm|mysql|redis|dial |connection|timeout|deadlock|duplicate entry|foreign key|constraint|syntax|record not found|no such|panic|runtime`)

// apiClientError answers a 4xx whose error came from a service or
// transaction: deliberate messages ("you are already a member of this
// organization") reach the user, anything that looks internal does not.
// ponytail: pattern-based; give services typed errors if a leak shows up.
func apiClientError(err error) gin.H {
	msg := err.Error()
	if internalError.MatchString(msg) {
		return apiError(msg)
	}
	// Service sentinels are prefixed with their package ("blogpost: slug ...").
	if i := strings.Index(msg, ": "); i > 0 && !strings.Contains(msg[:i], " ") {
		msg = msg[i+2:]
	}
	return apiUserError(msg)
}
