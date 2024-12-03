package utils

import (
	"fmt"
	"time"
)

const (
	Hour       = 1
	FourHours  = Hour * 4
	EightHours = Hour * 8
	Day        = Hour * 24
	ThreeDays  = Day * 3
	Week       = Day * 7
	Month      = Day * 30
)

func pluralizeHours(hours int) string {
	if hours%10 == 1 && hours%100 != 11 {
		return "час"
	} else if hours%10 >= 2 && hours%10 <= 4 && (hours%100 < 10 || hours%100 >= 20) {
		return "часа"
	} else {
		return "часов"
	}
}

func pluralizeDays(days int) string {
	if days%10 == 1 && days%100 != 11 {
		return "день"
	} else if days%10 >= 2 && days%10 <= 4 && (days%100 < 10 || days%100 >= 20) {
		return "дня"
	} else {
		return "дней"
	}
}

func pluralizeMinutes(minutes int) string {
	if minutes%10 == 1 && minutes%100 != 11 {
		return "минута"
	} else if minutes%10 >= 2 && minutes%10 <= 4 && (minutes%100 < 10 || minutes%100 >= 20) {
		return "минуты"
	} else {
		return "минут"
	}
}

func FormatHours(hours int) string {
	if hours < 24 {
		return fmt.Sprintf("%d %s", hours, pluralizeHours(hours))
	}
	days := hours / 24
	remainingHours := hours % 24
	if days == 1 && remainingHours == 0 {
		return "1 день"
	}

	if remainingHours == 0 {
		return fmt.Sprintf("%d %s", days, pluralizeDays(days))
	}
	return fmt.Sprintf("%d %s и %d %s", days, pluralizeDays(days), remainingHours, pluralizeHours(remainingHours))
}

func FormatMinutes(totalMinutes int) string {
	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60

	var answer string
	if days != 0 {
		answer += fmt.Sprintf("%d %s ", days, pluralizeDays(days))
	}
	if hours != 0 {
		answer += fmt.Sprintf("%d %s ", hours, pluralizeHours(hours))
	}
	if minutes != 0 && days == 0 {
		answer += fmt.Sprintf("%d %s ", minutes, pluralizeMinutes(minutes))
	}

	return answer
}

var Months = map[time.Month]string{
	time.January:   "янв",
	time.February:  "фев",
	time.March:     "мар",
	time.April:     "апр",
	time.May:       "мая",
	time.June:      "июн",
	time.July:      "июл",
	time.August:    "авг",
	time.September: "сен",
	time.October:   "окт",
	time.November:  "ноя",
	time.December:  "дек",
}

// приводит createdAt к формату 03 дек 01:31
func FormatCreatedAt(ca *time.Time) string {
	months := []string{
		"янв", "фев", "мар", "апр", "май", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек",
	}
	return fmt.Sprintf("%02d %s %02d:%02d",
		ca.Day(), months[ca.Month()-1], ca.Hour(), ca.Minute())
}
