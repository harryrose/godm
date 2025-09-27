package size

import (
	"fmt"
	"strconv"
	"strings"
)

type Size int64

func (s Size) Bytes() int64 {
	return int64(s)
}

func (s Size) Kilobytes() float64 {
	return float64(s) / 1024.0
}

func (s Size) Megabytes() float64 {
	return float64(s) / (1024.0 * 1024.0)
}

func (s Size) Gigabytes() float64 {
	return float64(s) / (1024.0 * 1024.0 * 1024.0)
}

func (s Size) String() string {
	suffix := "B"
	if s < 1024 {
		return fmt.Sprintf("%d%s", s, suffix)
	}

	suffix = "KB"
	f := float64(s) / 1024.0

	if f >= 1024 {
		suffix = "MB"
		f = f / 1024.0
	}
	if f >= 1024 {
		suffix = "GB"
		f = f / 1024.0
	}
	return fmt.Sprintf("%s%s", strconv.FormatFloat(f, 'f', 2, 64), suffix)
}

func FromString(str string) (Size, error) {
	str = strings.TrimSpace(str)
	lastDigit := 0
	for i, r := range str {
		if r >= '0' && r <= '9' || r == '.' {
			lastDigit = i
			continue
		}
		if r == ' ' {
			continue
		}
		break
	}

	numberPortion := str[:lastDigit+1]
	number, err := strconv.ParseFloat(numberPortion, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing number %q: %w", numberPortion, err)
	}

	str = strings.ToUpper(strings.TrimSpace(str[lastDigit+1:]))
	if len(str) == 0 || str[0] == 'B' {
		// bytes
		return Size(number), nil
	}
	if len(str) > 1 && str[1] != 'B' {
		return 0, fmt.Errorf("invalid size suffix %q", str)
	}
	if len(str) > 2 {
		return 0, fmt.Errorf("invalid size suffix %q", str)
	}

	switch str[0] {
	case 'K':
		return Size(number * 1024), nil
	case 'M':
		return Size(number * 1024 * 1024), nil
	case 'G':
		return Size(number * 1024 * 1024 * 1024), nil
	case 'T':
		return Size(number * 1024 * 1024 * 1024 * 1024), nil
	default:
		return 0, fmt.Errorf("unknown size suffix %q", str)
	}
}
