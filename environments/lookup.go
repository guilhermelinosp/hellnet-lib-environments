package environments

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// lookupEnv returns the first non-empty value of prefix+suffix or
// fallbackPrefix+suffix, reporting whether one was found. An empty
// fallbackPrefix disables the fallback lookup.
func lookupEnv(prefix, fallbackPrefix, suffix string) (string, bool) {
	for _, p := range [...]string{prefix, fallbackPrefix} {
		if p == "" {
			continue
		}
		if v := os.Getenv(p + suffix); v != "" {
			return v, true
		}
	}
	return "", false
}

// parsedEnv resolves an environment variable with the standard precedence and
// converts it with parse. It returns def when no value is set or when parsing
// fails.
func parsedEnv[T any](prefix, fallbackPrefix, suffix string, def T, parse func(string) (T, error)) T {
	s, ok := lookupEnv(prefix, fallbackPrefix, suffix)
	if !ok {
		return def
	}
	v, err := parse(s)
	if err != nil {
		return def
	}
	return v
}

// firstExistingFile returns the first path that exists on disk.
func firstExistingFile(paths ...string) (string, bool) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		p = filepath.Clean(p)
		//nolint:gosec // G703: paths come from env vars for .env file discovery
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

// parseBool accepts the common textual spellings of a boolean.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, strconv.ErrSyntax
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
