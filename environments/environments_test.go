package environments

import (
	"os"
	"path/filepath"
	"strings"
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

	// Empty primary prefix reads the bare suffix variable.
	t.Setenv("BARE_KEY", "bare")
	if got := GetString("", "MY_FALLBACK_", "BARE_KEY", "def"); got != "bare" {
		t.Fatalf("expected bare, got %q", got)
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

func TestGetIntE(t *testing.T) {
	t.Setenv("P_INT_KEY", "42")
	n, err := GetIntE("P_", "F_", "INT_KEY", 7)
	if err != nil || n != 42 {
		t.Fatalf("expected (42, nil), got (%d, %v)", n, err)
	}

	os.Unsetenv("P_INT_KEY")
	n, err = GetIntE("P_", "F_", "INT_KEY", 7)
	if err != nil || n != 7 {
		t.Fatalf("expected (7, nil) when unset, got (%d, %v)", n, err)
	}

	t.Setenv("F_INT_KEY", "notanint")
	n, err = GetIntE("P_", "F_", "INT_KEY", 7)
	if err == nil {
		t.Fatalf("expected error for invalid integer")
	}
	if n != 7 {
		t.Fatalf("expected default 7 on error, got %d", n)
	}
	if !strings.Contains(err.Error(), "F_INT_KEY") {
		t.Fatalf("error should name the variable, got %v", err)
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

func TestGetBoolE(t *testing.T) {
	t.Setenv("B_KEY", "yes")
	b, err := GetBoolE("B_", "", "KEY", false)
	if err != nil || !b {
		t.Fatalf("expected (true, nil), got (%v, %v)", b, err)
	}

	t.Setenv("B_KEY", "maybe")
	b, err = GetBoolE("B_", "", "KEY", true)
	if err == nil {
		t.Fatalf("expected error for invalid boolean")
	}
	if !b {
		t.Fatalf("expected default true on error, got %v", b)
	}
	if !strings.Contains(err.Error(), "B_KEY") {
		t.Fatalf("error should name the variable, got %v", err)
	}

	os.Unsetenv("B_KEY")
	b, err = GetBoolE("B_", "", "KEY", true)
	if err != nil || !b {
		t.Fatalf("expected (true, nil) when unset, got (%v, %v)", b, err)
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
	for _, s := range []string{"garbage", "01:30", "aa:bb:cc", "00:00:05.xx"} {
		if _, err := ParseDuration(s); err == nil {
			t.Fatalf("expected error for invalid duration %q", s)
		}
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

func TestGetDurationE(t *testing.T) {
	t.Setenv("DUR_KEY", "1h30m")
	d, err := GetDurationE("DUR_", "", "KEY", time.Second)
	if err != nil || d != 90*time.Minute {
		t.Fatalf("expected (90m, nil), got (%v, %v)", d, err)
	}

	t.Setenv("DUR_KEY", "garbage")
	d, err = GetDurationE("DUR_", "", "KEY", time.Second)
	if err == nil {
		t.Fatalf("expected error for invalid duration")
	}
	if d != time.Second {
		t.Fatalf("expected default 1s on error, got %v", d)
	}
	if !strings.Contains(err.Error(), "DUR_KEY") {
		t.Fatalf("error should name the variable, got %v", err)
	}

	os.Unsetenv("DUR_KEY")
	d, err = GetDurationE("DUR_", "", "KEY", time.Second)
	if err != nil || d != time.Second {
		t.Fatalf("expected (1s, nil) when unset, got (%v, %v)", d, err)
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

	// Known names are matched case-insensitively and ignoring surrounding space.
	for _, v := range []string{"development", " Local ", "TEST", "dev"} {
		t.Setenv("HELLNET_ENVIRONMENT", v)
		if !IsDev() {
			t.Fatalf("expected IsDev true for %q", v)
		}
	}

	// Unknown values must not enable development behaviour.
	for _, v := range []string{"production", "PRODUCTION", "prod", " Staging", "Homolog", "typo"} {
		t.Setenv("HELLNET_ENVIRONMENT", v)
		if IsDev() {
			t.Fatalf("expected IsDev false for %q", v)
		}
	}
}

func TestLoadDotEnvSkipsUntrustedFiles(t *testing.T) {
	t.Setenv("HELLNET_ENVIRONMENT", "Development")

	// World-writable .env is ignored.
	dir := t.TempDir()
	writable := filepath.Join(dir, ".env")
	if err := os.WriteFile(writable, []byte("UNTRUSTED=yes\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.Chmod(writable, 0o666); err != nil {
		t.Fatalf("chmod .env: %v", err)
	}
	t.Setenv("CUSTOM_ENV_PATH", writable)
	if err := LoadDotEnv("CUSTOM_ENV_PATH"); err == nil {
		t.Fatalf("expected error for world-writable .env")
	}
	if got := os.Getenv("UNTRUSTED"); got != "" {
		t.Fatalf("expected world-writable .env not to be loaded, got UNTRUSTED=%q", got)
	}

	// .env inside a world-writable directory is ignored.
	shared := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(shared, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(shared, 0o777); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	inShared := filepath.Join(shared, ".env")
	if err := os.WriteFile(inShared, []byte("PLANTED=yes\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	t.Setenv("CUSTOM_ENV_PATH", inShared)
	if err := LoadDotEnv("CUSTOM_ENV_PATH"); err == nil {
		t.Fatalf("expected error for .env in world-writable dir")
	}
	if got := os.Getenv("PLANTED"); got != "" {
		t.Fatalf("expected .env in world-writable dir not to be loaded, got PLANTED=%q", got)
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

func TestLoadDotEnvCustomVarMissingFile(t *testing.T) {
	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Setenv("MY_ENV_FILE", filepath.Join(t.TempDir(), "does-not-exist.env"))

	err := LoadDotEnv("MY_ENV_FILE")
	if err == nil {
		t.Fatalf("expected error when custom var points to a missing file")
	}
	if !strings.Contains(err.Error(), "MY_ENV_FILE") {
		t.Fatalf("error should name the variable, got %v", err)
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

func TestLoadDotEnvCustomVarSkipsUnsetAndEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FALLTHROUGH_KEY=from-cwd\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Setenv("EMPTY_ENV_PATH", "")
	t.Chdir(dir)

	if err := LoadDotEnv("UNSET_ENV_PATH", "EMPTY_ENV_PATH"); err != nil {
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
