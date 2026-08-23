// Package environments provides a single, standardized way for Hellnet Go
// libraries to read configuration from environment variables and .env files.
//
// It is the shared backend used by hellnet-lib-cache and hellnet-lib-telemetry
// so that env handling is consistent across the ecosystem.
package environments

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// DeploymentEnv returns the value of HELLNET_ENVIRONMENT, or "" if unset.
func DeploymentEnv() string {
	return os.Getenv("HELLNET_ENVIRONMENT")
}

// IsDev reports whether the current deployment is a non-production environment.
// It returns true when HELLNET_ENVIRONMENT is empty, "Development" or any value
// other than "Production"/"Staging".
func IsDev() bool {
	e := DeploymentEnv()
	return e == "" || (e != "Production" && e != "Staging")
}

// LoadDotEnv loads environment variables from a .env file for local development.
//
// In Production/Staging it is a no-op. When a customVars entry resolves to a
// non-empty env var pointing to an existing file, that file is loaded.
// Otherwise it searches, in order: the .env next to the executable, then .env
// in the working directory and each of its parent directories, loading the
// first one found.
func LoadDotEnv(customVars ...string) error {
	if !IsDev() {
		return nil
	}

	customPaths := make([]string, 0, len(customVars))
	for _, v := range customVars {
		customPaths = append(customPaths, os.Getenv(v))
	}
	if p, ok := firstExistingFile(customPaths...); ok {
		return godotenv.Load(p)
	}

	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), ".env"))
	}
	if wd, err := os.Getwd(); err == nil {
		for dir := wd; ; {
			candidates = append(candidates, filepath.Join(dir, ".env"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if p, ok := firstExistingFile(candidates...); ok {
		return godotenv.Load(p)
	}
	return nil
}

// GetString returns the first non-empty of prefix+suffix, fallbackPrefix+suffix,
// or def.
func GetString(prefix, fallbackPrefix, suffix string, def string) string {
	if v, ok := lookupEnv(prefix, fallbackPrefix, suffix); ok {
		return v
	}
	return def
}

// GetInt is like GetString but parses an integer with the same precedence.
func GetInt(prefix, fallbackPrefix string, suffix string, def int) int {
	return parsedEnv(prefix, fallbackPrefix, suffix, def, strconv.Atoi)
}

// GetBool is like GetString but parses a boolean with the same precedence.
func GetBool(prefix, fallbackPrefix string, suffix string, def bool) bool {
	return parsedEnv(prefix, fallbackPrefix, suffix, def, parseBool)
}

// GetDuration is like GetString but parses a time.Duration with the same
// precedence. It accepts both Go duration strings and .NET "HH:MM:SS[.FFF]".
func GetDuration(prefix, fallbackPrefix string, suffix string, def time.Duration) time.Duration {
	return parsedEnv(prefix, fallbackPrefix, suffix, def, ParseDuration)
}

// ParseDuration parses a duration string trying Go's time.ParseDuration first
// and then the .NET "HH:MM:SS" or "HH:MM:SS.FFF" format.
func ParseDuration(s string) (time.Duration, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	if d, ok := parseClockDuration(s); ok {
		return d, nil
	}
	return 0, fmt.Errorf("invalid duration: %q", s)
}
