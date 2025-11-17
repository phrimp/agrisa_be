package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func ValidateEmail(email string) (bool, error) {
	email_regex_pattern := `^[a-zA-Z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+(?:\.[a-zA-Z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$`

	regex, err := regexp.Compile(email_regex_pattern)
	if err != nil {
		return false, fmt.Errorf("error: compiling regex: %s", err)
	}

	if !regex.MatchString(email) {
		return false, fmt.Errorf("error: email format incorrect")
	}
	return true, nil
}

func ValidatePhone(phone string) (bool, error) {
	phone_regex_patterns := []string{
		`^\+84[3-9]\d{8}$`, // +84 + mobile (9 digits total after +84)
		`^\+84[1-8]\d{9}$`, // +84 + landline (10 digits total after +84)
		`^0[1-9]\d{8,9}$`,  // Domestic format: 0 + 9-10 digits
		`^84[3-9]\d{8}$`,   // 84 without + (mobile)
		`^84[1-8]\d{9}$`,   // 84 without + (landline)
	}

	for _, pattern := range phone_regex_patterns {
		if matched, _ := regexp.MatchString(pattern, phone); matched {
			return true, nil
		}
	}
	return false, fmt.Errorf("phone format incorrect")
}

func ValidateCCCD(cccd string) bool {
	pattern := `^([0-9]{3})([0-9])([0-9]{2})([0-9]{6})$`

	// Clean input and validate pattern
	cleanCCCD := regexp.MustCompile(`[^\d]`).ReplaceAllString(cccd, "")
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(cleanCCCD)

	if len(matches) != 5 {
		return false
	}

	province, _ := strconv.Atoi(matches[1])
	centuryGender, _ := strconv.Atoi(matches[2])
	birthYear, _ := strconv.Atoi(matches[3])

	// Validate province code (001-096)
	if province < 1 || province > 96 {
		return false
	}

	// Validate century/gender code (0-9)
	if centuryGender < 0 || centuryGender > 9 {
		return false
	}

	// Calculate full birth year and validate
	centuryBase := 1900 + (centuryGender/2)*100
	fullYear := centuryBase + birthYear
	currentYear := time.Now().Year()

	// Check birth year is reasonable and person is at least 14
	return fullYear >= 1900 && fullYear <= currentYear && (currentYear-fullYear) >= 14
}

func GetQueryParamAsInt(c *gin.Context, paramName string, defaultValue int) (int, error) {
	// Get the query parameter value
	paramValue := c.Query(paramName)

	// If parameter is not provided or empty, return default value
	if paramValue == "" {
		return defaultValue, nil
	}

	// Try to convert to integer
	intValue, err := strconv.Atoi(paramValue)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", paramName)
	}

	// Validate that value is greater than 0
	if intValue <= 0 {
		return 0, fmt.Errorf("invalid %s", paramName)
	}

	return intValue, nil
}


