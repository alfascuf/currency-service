package validator

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validation for date range
	_ = validate.RegisterValidation("date_range", validateDateRange)
}

// Validate validates struct fields using validator tags
func Validate(data interface{}) error {
	err := validate.Struct(data)
	if err == nil {
		return nil
	}

	// Format validation errors into readable messages
	var errMessages []string
	for _, err := range err.(validator.ValidationErrors) {
		errMessages = append(errMessages, formatValidationError(err))
	}

	return fmt.Errorf("%s", strings.Join(errMessages, "; "))
}

// ValidateHistoryDates checks that start_date is before or equal to end_date
func ValidateHistoryDates(startDate, endDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("invalid start_date format: must be YYYY-MM-DD")
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("invalid end_date format: must be YYYY-MM-DD")
	}

	if start.After(end) {
		return fmt.Errorf("start_date must be before or equal to end_date")
	}

	return nil
}

// formatValidationError converts validator error to human-readable message
func formatValidationError(err validator.FieldError) string {
	field := strings.ToLower(err.Field())

	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", field, err.Param())
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", field)
	case "datetime":
		return fmt.Sprintf("%s must be in format YYYY-MM-DD", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// validateDateRange validates date format (custom validator)
func validateDateRange(fl validator.FieldLevel) bool {
	dateStr := fl.Field().String()
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}
