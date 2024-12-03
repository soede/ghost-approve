package utils

import (
	"errors"
	"fmt"
	"ghost-approve/pkg/botErrors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type EmailExtractResult struct {
	ValidEmails     []string
	InvalidElements []string
	AllValid        bool
}

func DefineLink(s string) ([]string, bool) {
	var wrongLinks []string
	var ok = true
	s = strings.Trim(strings.ReplaceAll(s, " ", ""), ",")
	var ans = strings.Split(s, ",")

	for _, el := range ans {
		if strings.HasPrefix(el, "http") != true {
			ok = false
			wrongLinks = append(wrongLinks, el)
		}
	}

	if !ok {
		return wrongLinks, ok
	}
	return ans, ok
}

// DefineTime принимает строку с продолжительностью или точной датой и возвращает количество часов
func DefineTime(input string) (int, error) {
	input = strings.TrimSpace(input)
	input = strings.Trim(input, "\u00A0")
	if strings.Contains(input, "/") {
		return handleExactDate(input)
	}

	units := strings.Fields(input)
	var totalHours int

	for _, unit := range units {
		if len(unit) < 2 {
			return 0, fmt.Errorf("некорректный формат: %s", unit)
		}

		number := strings.TrimRightFunc(unit, func(r rune) bool {
			return r < '0' || r > '9'
		})
		suffix := unit[len(number):]

		value, err := strconv.Atoi(number)
		if err != nil {
			return 0, fmt.Errorf("некорректное значение: %s", unit)
		}

		switch suffix {
		case "ч":
			totalHours += value
		case "д": // Дни
			totalHours += value * 24
		case "н":
			totalHours += value * 24 * 7
		case "м":
			totalHours += value * 24 * 30
		default:
			return 0, fmt.Errorf("некорректная единица времени: %s", suffix)
		}
	}

	if totalHours < 1 {
		return 0, botErrors.ErrHoursBelowMinimum
	}
	if totalHours > 24*60 {
		return 0, botErrors.ErrHoursExceedLimit
	}

	return totalHours, nil
}

func handleExactDate(input string) (int, error) {
	layout := "2006/1/2 15:04"
	targetTime, err := time.Parse(layout, input)
	if err != nil {
		return 0, fmt.Errorf("некорректный формат даты: %s", input)
	}

	now := time.Now()
	if targetTime.Before(now) {
		return 0, botErrors.ErrDateInPast
	}

	duration := targetTime.Sub(now)
	totalHours := int(duration.Hours())

	if totalHours > 24*60 {
		return 0, botErrors.ErrHoursExceedLimit
	}

	return totalHours, nil
}

// isValidEmail проверяет, является ли строка валидным email
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// extractEmails извлекает и проверяет email из строки
func ExtractEmails(inputStr string) EmailExtractResult {
	var validEmails []string
	var invalidEmails []string
	var allValid bool

	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailPattern)

	inputStr = strings.ReplaceAll(inputStr, string('\u00A0'), " ")
	inputStr = strings.TrimRight(inputStr, " ")

	rawEmails := strings.FieldsFunc(inputStr, func(r rune) bool {
		return r == ' ' || r == ','
	})
	for _, email := range rawEmails {
		cleanedEmail := strings.Trim(email, "@[]")
		if re.MatchString(cleanedEmail) {
			validEmails = append(validEmails, cleanedEmail)
		} else {
			invalidEmails = append(invalidEmails, email)
		}
	}
	allValid = len(invalidEmails) == 0
	return EmailExtractResult{
		validEmails, invalidEmails, allValid,
	}
}

func CreateUserLink(ids []string) string {
	length := len(ids)
	if length == 0 {
		return ""
	}

	newIds := make([]string, length)

	for i, email := range ids {
		newIds[i] = "@[" + email + "]"
	}

	if length == 1 {
		return newIds[0]
	}

	return strings.Join(newIds, ", ")

}

func ParseFileInfo(input string) (int, int, error) {
	parts := strings.Split(input, "_")

	if len(parts) < 2 || len(parts) > 3 {
		return 0, 0, errors.New("Invalid format")
	}

	fileID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	version := 0
	if len(parts) == 3 {
		version, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, err
		}
	}

	return fileID, version, nil
}
