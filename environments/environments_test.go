package environments

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetString(t *testing.T) {
	t.Setenv("MY_PREFIX_KEY", "primary")
	t.Setenv("MY_FALLBACK_KEY", "fallback")

	if got := GetString("MY_PREFIX_", "MY_FALLBACK_", "KEY", "def"); got != "primary" {
		t.Fatalf("expected primary, got %q", got)
	}

	os.Unsetenv("MY_PREFIX_KEY")
	if got := GetString("MY_PREFIX_", "MY_FALLBACK_", "KEY", "def"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}

	os.Unsetenv("MY_FALLBACK_KEY")
	if got := GetString("MY_PREFIX_", "MY_FALLBACK_", "KEY", "def"); got != "def" {
		t.Fatalf("expected def, got %q", got)
	}

	// No fallback prefix.
	if got := GetString("MISSING_", "", "KEY", "onlydef"); got != "onlydef" {
		t.Fatalf("expected onlydef, got %q", got)
	}
}

func TestGetInt(t *testing.T) {
	t.Setenv("P_INT_KEY", "42")
	if got := GetInt("P_", "F_", "INT_KEY", 7); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	os.Unsetenv("P_INT_KEY")
	t.Setenv("F_INT_KEY", "99")
	if got := GetInt("P_", "F_", "INT_KEY", 7); got != 99 {
		t.Fatalf("expected 99, got %d", got)
	}

	os.Unsetenv("F_INT_KEY")
	if got := GetInt("P_", "F_", "INT_KEY", 7); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}

	// Non-integer value falls back to default.
	t.Setenv("P_INT_KEY", "notanint")
	if got := GetInt("P_", "F_", "INT_KEY", 7); got != 7 {
		t.Fatalf("expected 7 on parse error, got %d", got)
	}
}

func TestGetBool(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"on", true},
		{"TRUE", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"maybe", false}, // default when unparseable
	}
	for _, c := range cases {
		t.Setenv("B_KEY", c.val)
		if got := GetBool("B_", "", "KEY", false); got != c.want {
			t.Fatalf("value %q: expected %v, got %v", c.val, c.want, got)
		}
	}

	os.Unsetenv("B_KEY")
	if got := GetBool("B_", "", "KEY", true); got != true {
		t.Fatalf("expected default true, got %v", got)
	}
}

func TestParseDuration(t *testing.T) {
	// Go duration format.
	d, err := ParseDuration("1h30m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 90*time.Minute {
		t.Fatalf("expected 90m, got %v", d)
	}

	// .NET HH:MM:SS.
	d, err = ParseDuration("01:30:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 90*time.Minute {
		t.Fatalf("expected 90m, got %v", d)
	}

	// .NET HH:MM:SS.FFF.
	d, err = ParseDuration("00:00:05.500")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 5*time.Second+500*time.Millisecond {
		t.Fatalf("expected 5.5s, got %v", d)
	}

	// Invalid.
	if _, err := ParseDuration("garbage"); err == nil {
		t.Fatalf("expected error for invalid duration")
	}
}

func TestGetDuration(t *testing.T) {
	t.Setenv("DUR_KEY", "1h30m")
	if got := GetDuration("DUR_", "", "KEY", time.Second); got != 90*time.Minute {
		t.Fatalf("expected 90m, got %v", got)
	}

	os.Unsetenv("DUR_KEY")
	t.Setenv("DURF_KEY", "00:00:05.500")
	if got := GetDuration("DUR_", "DURF_", "KEY", time.Second); got != 5500*time.Millisecond {
		t.Fatalf("expected 5.5s, got %v", got)
	}

	os.Unsetenv("DURF_KEY")
	if got := GetDuration("DUR_", "DURF_", "KEY", time.Second); got != time.Second {
		t.Fatalf("expected default 1s, got %v", got)
	}
}

func TestDeploymentEnvAndIsDev(t *testing.T) {
	os.Unsetenv("HELLNET_ENVIRONMENT")
	if DeploymentEnv() != "" {
		t.Fatalf("expected empty DeploymentEnv")
	}
	if !IsDev() {
		t.Fatalf("expected IsDev true when HELLNET_ENVIRONMENT unset")
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Production")
	if DeploymentEnv() != "Production" {
		t.Fatalf("expected Production")
	}
	if IsDev() {
		t.Fatalf("expected IsDev false when Production")
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Staging")
	if IsDev() {
		t.Fatalf("expected IsDev false when Staging")
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	if !IsDev() {
		t.Fatalf("expected IsDev true when Development")
	}
}

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("FOO=bar\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("FOO"); got != "bar" {
		t.Fatalf("expected FOO=bar, got %q", got)
	}
}

func TestLoadDotEnvNoopInProd(t *testing.T) {
	t.Setenv("HELLNET_ENVIRONMENT", "Production")
	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv prod: %v", err)
	}
}

func TestLoadDotEnvCustomVar(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "custom.env")
	if err := os.WriteFile(envPath, []byte("CUSTOM_VAR_KEY=from-custom\n"), 0o600); err != nil {
		t.Fatalf("write custom env: %v", err)
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Setenv("CUSTOM_ENV_PATH", envPath)
	t.Chdir(dir)

	if err := LoadDotEnv("CUSTOM_ENV_PATH"); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("CUSTOM_VAR_KEY"); got != "from-custom" {
		t.Fatalf("expected from-custom, got %q", got)
	}
}

func TestLoadDotEnvCustomVarSkipsEmptyAndMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FALLTHROUGH_KEY=from-cwd\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Setenv("EMPTY_ENV_PATH", "")
	t.Setenv("MISSING_ENV_PATH", filepath.Join(dir, "does-not-exist.env"))
	t.Chdir(dir)

	if err := LoadDotEnv("UNSET_ENV_PATH", "EMPTY_ENV_PATH", "MISSING_ENV_PATH"); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("FALLTHROUGH_KEY"); got != "from-cwd" {
		t.Fatalf("expected from-cwd, got %q", got)
	}
}

func TestLoadDotEnvCustomVarLoadError(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	// A directory passes the os.Stat check but cannot be parsed as a .env file.
	t.Setenv("BROKEN_ENV_PATH", dir)
	t.Chdir(dir)

	if err := LoadDotEnv("BROKEN_ENV_PATH"); err == nil {
		t.Fatalf("expected error when custom path is not a readable .env file")
	}
}

func TestLoadDotEnvFromParentDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("PARENT_DIR_KEY=from-parent\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Chdir(sub)

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("PARENT_DIR_KEY"); got != "from-parent" {
		t.Fatalf("expected from-parent, got %q", got)
	}
}

func TestLoadDotEnvNoFileFound(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Chdir(dir)

	// No .env exists in the temp dir; parents are system dirs without one.
	if err := LoadDotEnv(); err != nil {
		t.Fatalf("expected nil when no .env is found, got %v", err)
	}
}

func TestGetDurationInvalidValueFallsBackToDefault(t *testing.T) {
	t.Setenv("DUR_KEY", "not-a-duration")
	if got := GetDuration("DUR_", "", "KEY", 2*time.Second); got != 2*time.Second {
		t.Fatalf("expected default 2s, got %v", got)
	}
}
