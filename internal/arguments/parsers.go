package arguments

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"slices"
	"strings"
)

func ParseNonEmptyStr(a *Argument[string]) (string, error) {
	raw := a.RawValue()
	val := strings.TrimSpace(strings.ToLower(raw))
	if val == "" {
		return "", fmt.Errorf("invalid %s: undefined or empty", a.Name())
	}
	return raw, nil
}

func ParsePositiveInt(a *Argument[int]) (int, error) {
	raw := a.RawValue()
	if raw < 1 {
		return 0, fmt.Errorf("invalid %s (%d): value cannot be negative", a.Name(), raw)
	}
	return raw, nil
}

func ParseUnsignedInt(a *Argument[int]) (int, error) {
	raw := a.RawValue()
	if raw < 0 {
		return 0, fmt.Errorf("invalid %s (%d): value cannot be negative", a.Name(), raw)
	}
	return raw, nil
}

func ParseIntRange(a *Argument[int], min int, max int) func(a *Argument[int]) (int, error) {
	return func(b *Argument[int]) (int, error) {
		raw := a.RawValue()
		if raw < min || raw > max {
			return 0, fmt.Errorf("invalid %s (%d): value must be between %d-%d", a.Name(), raw, min, max)
		}
		return raw, nil
	}
}

func ParsePort(a *Argument[int]) (int, error) {
	raw := a.RawValue()
	if raw < 1024 || raw > 65535 {
		return 0, fmt.Errorf("invalid %s port (%d): value must be in range 1025-65534", a.Name(), raw)
	}
	return raw, nil
}

func ParsePassword(a *Argument[string]) (string, error) {
	raw := a.RawValue()
	val := strings.TrimSpace(strings.ToLower(raw))
	if val != "" && len(val) > 16 {
		return "", fmt.Errorf("invalid %s (%s): value cannot exceed 16 characters", a.Name(), raw)
	}
	return strings.TrimSpace(raw), nil
}

func ParseURL(a *Argument[string]) (string, error) {
	raw := a.RawValue()
	val := strings.TrimSpace(raw)
	if val != "" {
		parsedURL, err := url.Parse(val)
		if err != nil || !strings.HasPrefix(parsedURL.Scheme, "http") {
			return "", fmt.Errorf("invalid URL '%s'", raw)
		}
	}
	return val, nil
}

func ParseMail(a *Argument[string]) (string, error) {
	raw := a.RawValue()
	val := strings.TrimSpace(strings.ToLower(raw))
	_, err := mail.ParseAddress(val)
	if val != "" && err != nil {
		return "", fmt.Errorf("invalid Email: %s", raw)
	}
	return raw, nil
}

func ParseIP(a *Argument[string]) (string, error) {
	raw := a.RawValue()
	val := strings.TrimSpace(raw)
	if val == "" {
		return "", fmt.Errorf("IP address is empty")
	}

	parsedIP := net.ParseIP(val)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address: '%s'", raw)
	}
	return val, nil
}

func ParseExistingDir(a *Argument[string]) (string, error) {
	raw := a.RawValue()
	val := strings.TrimSpace(raw)
	if val != "" {
		info, err := os.Stat(val)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("directory does not exist: '%s'", raw)
			}
			return "", fmt.Errorf("error checking directory: '%s': %v", raw, err)
		}

		if !info.IsDir() {
			return "", fmt.Errorf("path is not a directory: '%s'", raw)
		}
	}
	return val, nil
}

func ParseGameMode(a *Argument[string]) (string, error) {
	validModes := map[string]string{
		"survival":  "KFmod.KFGameType",
		"objective": "KFStoryGame.KFstoryGameInfo",
		"toymaster": "KFCharPuppets.TOYGameInfo",
	}

	raw := a.RawValue()
	val, ok := validModes[strings.ToLower(raw)]
	if val == "" {
		val = validModes["survival"]
		ok = true
	}
	if ok {
		return val, nil
	}
	return raw, nil // Custom
}

func ParseGameDifficulty(raw string) func(a *Argument[int]) (int, error) {
	difficulties := map[string]int{
		"easy":     1,
		"normal":   2,
		"hard":     4,
		"suicidal": 5,
		"hell":     7,
	}

	var (
		val int
		err error
		ok  bool
	)

	val, ok = difficulties[strings.ToLower(raw)]
	if !ok {
		val = 0.0
		err = fmt.Errorf("invalid Game Difficulty: %s", raw)
	}
	return func(a *Argument[int]) (int, error) {
		return val, err
	}
}
func ParseGameLength(raw string) func(a *Argument[int]) (int, error) {
	lengths := map[string]int{
		"short":  0,
		"medium": 1,
		"long":   2,
	}

	var (
		val int
		err error
		ok  bool
	)

	val, ok = lengths[strings.ToLower(raw)]
	if !ok {
		val = 0
		err = fmt.Errorf("invalid Game Length: %s", raw)
	}
	return func(a *Argument[int]) (int, error) {
		return val, err
	}
}

func ParseFriendlyFireRate(a *Argument[float64]) (float64, error) {
	raw := a.RawValue()
	if raw < 0.0 || raw > 1.0 {
		return 0, fmt.Errorf("invalid Friendly Fire Rate (%f): value must be between 0.0 and 1.0", raw)
	}
	return raw, nil
}

func ParseSpecimenType(a *Argument[string]) (string, error) {
	specimenTypes := map[string]string{
		"default":   "ET_None",
		"summer":    "ET_SummerSideshow",
		"halloween": "ET_HillbillyHorror",
		"christmas": "ET_TwistedChristmas",
	}

	raw := a.RawValue()
	val, ok := specimenTypes[strings.ToLower(raw)]
	if val == "" {
		val = specimenTypes["default"]
		ok = true
	}
	if !ok {
		return "", fmt.Errorf("invalid Specimen Type: %s", raw)
	}
	return val, nil
}

func ParseLogLevel(a *Argument[string]) (string, error) {
	levels := []string{"info", "debug", "warn", "error"}

	raw := a.RawValue()
	val := strings.TrimSpace(strings.ToLower(raw))
	if !slices.Contains(levels, val) {
		return "", fmt.Errorf("invalid Log Level: %s", raw)
	}
	return val, nil
}

func ParseLogFileFormat(a *Argument[string]) (string, error) {
	raw := a.RawValue()
	val := strings.TrimSpace(strings.ToLower(raw))
	if val != "text" && val != "json" {
		return "", fmt.Errorf("invalid Log File Format: %s", raw)
	}
	return val, nil
}
