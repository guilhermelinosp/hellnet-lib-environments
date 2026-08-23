package environments

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// lookup returns the first non-empty value of prefix+suffix or
// fallbackPrefix+suffix, along with the name of the variable it came from.
func lookup(prefix, fallbackPrefix, suffix string) (val, name string, ok bool) {
	name = prefix + suffix
	if v := os.Getenv(name); v != "" {
		return v, name, true
	}
	if fallbackPrefix != "" {
		name = fallbackPrefix + suffix
		if v := os.Getenv(name); v != "" {
			return v, name, true
		}
	}
	return "", "", false
}

// parsedEnv resolves an environment variable with the standard precedence and
// converts it with parse. Unset variables yield (def, nil); a parse failure
// yields def and an error naming the variable it came from.
func parsedEnv[T any](prefix, fallbackPrefix, suffix string, def T, parse func(string) (T, error)) (T, error) {
	s, name, ok := lookup(prefix, fallbackPrefix, suffix)
	if !ok {
		return def, nil
	}
	v, err := parse(s)
	if err != nil {
		return def, fmt.Errorf("environments: %s: %w", name, err)
	}
	return v, nil
}

// parseInt parses an integer, describing the offending value on failure.
func parseInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", s, err)
	}
	return n, nil
}

// parseBool accepts the common textual spellings of a boolean
// (true/false, 1/0, yes/no, on/off, case-insensitive).
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", s)
	}
}

// parseClockDuration parses the .NET "HH:MM:SS" and "HH:MM:SS.FFF" formats.
func parseClockDuration(s string) (time.Duration, bool) {
	hms, frac, hasFrac := strings.Cut(s, ".")
	parts := strings.Split(hms, ":")
	if len(parts) != 3 {
		return 0, false
	}
	var total time.Duration
	for i, unit := range [...]time.Duration{time.Hour, time.Minute, time.Second} {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return 0, false
		}
		total += time.Duration(n) * unit
	}
	if hasFrac {
		f, err := strconv.ParseFloat("0."+frac, 64)
		if err != nil {
			return 0, false
		}
		total += time.Duration(f * float64(time.Second))
	}
	return total, true
}

// loadEnvFile loads a .env file, wrapping any failure with its path.
func loadEnvFile(path string) error {
	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("environments: load %q: %w", path, err)
	}
	return nil
}
