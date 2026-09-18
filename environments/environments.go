// Package environments provides a single, standardized way for Hellnet Go
// libraries to read configuration from environment variables and .env files.
//
// It is the shared backend used by hellnet-lib-cache and hellnet-lib-telemetry
// so that env handling is consistent across the ecosystem.
package environments

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// maxParentDirs bounds how far LoadDotEnv walks up from the working directory
// looking for a .env file, so discovery cannot reach shared directories such as
// /tmp, /home or the filesystem root.
const maxParentDirs = 8

// devEnvNames are the HELLNET_ENVIRONMENT values recognised as non-production.
// Anything else — including unknown or misspelled values — is treated as
// production so that a typo cannot enable development behaviour.
var devEnvNames = map[string]bool{
	"":            true,
	"development": true,
	"dev":         true,
	"local":       true,
	"test":        true,
	"testing":     true,
}

// DeploymentEnv returns the value of HELLNET_ENVIRONMENT, or "" if unset.
func DeploymentEnv() string {
	return os.Getenv("HELLNET_ENVIRONMENT")
}

// IsDev reports whether the current deployment is a non-production environment.
// It returns true only when HELLNET_ENVIRONMENT is empty or one of the known
// development names ("Development", "Dev", "Local", "Test", "Testing", matched
// case-insensitively). Every other value is treated as production.
func IsDev() bool {
	return devEnvNames[strings.ToLower(strings.TrimSpace(DeploymentEnv()))]
}

// LoadDotEnv loads environment variables from a .env file for local development.
//
// In any non-development environment it is a no-op. When a customVars entry
// resolves to a non-empty env var, that value must point to a loadable file,
// which is then loaded; a set-but-unusable path is an error rather than being
// skipped. Otherwise it searches, in order: the .env next to the executable,
// then .env in the working directory and up to maxParentDirs of its parent
// directories, loading the first one found.
//
// A candidate is only loaded when it is trusted: a regular file, not writable
// by other users, in a directory not writable by other users. An untrusted file
// could be planted by another local user to inject configuration (credentials,
// endpoints) into the process; discovered ones are skipped, while an explicitly
// configured customVars path is reported as an error.
func LoadDotEnv(customVars ...string) error {
	if !IsDev() {
		return nil
	}

	if loaded, err := loadFromCustomVars(customVars); loaded || err != nil {
		return err
	}
	return loadFirstFound(envCandidates())
}

// loadFromCustomVars loads the .env pointed to by the first of vars that is set
// to a non-empty value, reporting whether a file was loaded.
func loadFromCustomVars(vars []string) (bool, error) {
	for _, v := range vars {
		p := os.Getenv(v)
		if p == "" {
			continue
		}
		p = filepath.Clean(p)
		if _, err := os.Stat(p); err != nil {
			return false, fmt.Errorf("environments: %s=%q: %w", v, p, err)
		}
		if !isTrustedEnvFile(p) {
			return false, fmt.Errorf("environments: %s=%q: %w", v, p, errUntrustedEnvFile)
		}
		return true, loadEnvFile(p)
	}
	return false, nil
}

// envCandidates lists the .env paths to try, in order: next to the executable,
// then in the working directory and up to maxParentDirs of its parents.
func envCandidates() []string {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), ".env"))
	}
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i <= maxParentDirs; i++ {
			candidates = append(candidates, filepath.Join(dir, ".env"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return candidates
}

// loadFirstFound loads the first existing, trusted candidate. Candidates are
// probed with Lstat so that a symlink — including a broken or looping one —
// is rejected by isTrustedEnvFile instead of failing discovery.
func loadFirstFound(candidates []string) error {
	for _, c := range candidates {
		if _, err := os.Lstat(c); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("environments: stat %q: %w", c, err)
		}
		if !isTrustedEnvFile(c) {
			continue
		}
		return loadEnvFile(c)
	}
	return nil
}

// errUntrustedEnvFile is returned when an explicitly configured .env path is
// writable by users other than its owner.
var errUntrustedEnvFile = errors.New("refusing to load .env writable by other users")

// isTrustedEnvFile reports whether path is an existing regular file that only
// its owner can modify, inside a directory that only its owner can modify.
// Symlinks are rejected because their target can be swapped after the check.
func isTrustedEnvFile(path string) bool {
	//nolint:gosec // G703: path is a .env candidate, validated here before use
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() || !ownerOnlyWritable(fi.Mode()) {
		return false
	}
	di, err := os.Stat(filepath.Dir(path))
	return err == nil && di.IsDir() && ownerOnlyWritable(di.Mode())
}

// ownerOnlyWritable reports whether m denies write access to group and others.
func ownerOnlyWritable(m os.FileMode) bool {
	return m.Perm()&0o022 == 0
}

// Get returns the value of the environment variable with the given name.
// If the variable is set, returns its value.
// If not set and a default is provided (via variadic), returns the default.
// If not set and no default provided, panics with a clear message.
//
// Usage:
//   Get("HELLNET_KAFKA_TOPIC_ORDER_REQUESTED")                              // panic if not set
//   Get("HELLNET_KAFKA_TOPIC_ORDER_REQUESTED", "my-default")                // return default if not set
func Get(name string, def ...string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	panic("environments: required environment variable " + name + " is not set")
}

// GetInt returns the value of the environment variable as an integer.
// If the variable is set and can be parsed as an integer, returns its value.
// If not set and a default is provided (via variadic), returns the default.
// If not set and no default provided, panics with a clear message.
// If set but cannot be parsed as an integer, panics.
//
// Usage:
//   GetInt("HELLNET_KAFKA_MAX_RETRIES")                              // panic if not set
//   GetInt("HELLNET_KAFKA_MAX_RETRIES", "3")                        // return default if not set
//   GetInt("HELLNET_KAFKA_MAX_RETRIES", "3", "5")                   // return default if not set (first def used)
func GetInt(name string, def ...string) int {
	s := Get(name, def...)
	n, err := strconv.Atoi(s)
	if err != nil {
		panic("environments: environment variable " + name + " = " + s + " is not a valid integer")
	}
	return n
}

// GetBool returns the value of the environment variable as a boolean.
// If the variable is set and can be parsed as a boolean, returns its value.
// If not set and a default is provided (via variadic), returns the default.
// If not set and no default provided, panics with a clear message.
// If set but cannot be parsed as a boolean, panics.
//
// Usage:
//   GetBool("HELLNET_KAFKA_IDEMPOTENT")                              // panic if not set
//   GetBool("HELLNET_KAFKA_IDEMPOTENT", "true")                     // return default if not set
func GetBool(name string, def ...string) bool {
	s := Get(name, def...)
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		panic("environments: environment variable " + name + " = " + s + " is not a valid boolean")
	}
}

// GetDuration returns the value of the environment variable as a time.Duration.
// If the variable is set and can be parsed as a duration, returns its value.
// If not set and a default is provided (via variadic), returns the default.
// If not set and no default provided, panics with a clear message.
// If set but cannot be parsed as a duration, panics.
//
// Usage:
//   GetDuration("HELLNET_KAFKA_RETRY_DELAY")                              // panic if not set
//   GetDuration("HELLNET_KAFKA_RETRY_DELAY", "200ms")                    // return default if not set
func GetDuration(name string, def ...string) time.Duration {
	s := Get(name, def...)
	d, err := time.ParseDuration(s)
	if err != nil {
		panic("environments: environment variable " + name + " = " + s + " is not a valid duration")
	}
	return d
}

// GetString returns the first non-empty of prefix+suffix, fallbackPrefix+suffix,
// or def.
func GetString(prefix, fallbackPrefix, suffix string, def string) string {
	if v, _, ok := lookup(prefix, fallbackPrefix, suffix); ok {
		return v
	}
	return def
}

// GetRequiredString returns the value of prefix+suffix environment variable,
// or panics if the environment variable is not set.
// Use for required configuration that must be explicitly set.
// The environment variable name is prefix + "HELLNET_" + suffix.
func GetRequiredString(prefix, suffix string) string {
	if v, _, ok := lookup(prefix, "", suffix); ok {
		return v
	}
	name := prefix + "HELLNET_" + suffix
	panic("environments: required environment variable " + name + " is not set")
}

// lookup returns the first non-empty of prefix+suffix or fallbackPrefix+suffix,
// along with the name of the variable it came from.
func lookup(prefix, fallbackPrefix, suffix string) (val, name string, ok bool) {
	name = prefix + "HELLNET_" + suffix
	if v := os.Getenv(name); v != "" {
		return v, name, true
	}
	if fallbackPrefix != "" {
		name = fallbackPrefix + "HELLNET_" + suffix
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