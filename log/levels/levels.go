package levels

import (
	"fmt"
	"strings"
)

type Level uint8

const (
	Debug Level = iota
	Info
	Warn
	Error
)

const (
	StrDebug = "DEBUG"
	StrInfo  = "INFO"
	StrWarn  = "WARN"
	StrError = "ERROR"
)

func FromString(s string) (Level, error) {
	switch strings.ToUpper(s) {
	case StrDebug:
		return Debug, nil

	case StrInfo:
		return Info, nil

	case StrWarn:
		return Warn, nil

	case StrError:
		return Error, nil

	default:
		return 0, fmt.Errorf("unrecognised level: %q", s)
	}
}

func (l Level) String() string {
	switch l {
	case Debug:
		return StrDebug

	case Info:
		return StrInfo

	case Warn:
		return StrWarn

	case Error:
		return StrError

	default:
		return fmt.Sprintf("UNK(%v)", uint8(l))
	}
}
