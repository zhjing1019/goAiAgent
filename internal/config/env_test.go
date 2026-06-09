package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppEnvDefault(t *testing.T) {
	t.Setenv("APP_ENV", "")
	if got := AppEnv(); got != "development" {
		t.Fatalf("AppEnv() = %q, want development", got)
	}
}

func TestAppEnvAliases(t *testing.T) {
	cases := map[string]string{
		"dev":  "development",
		"prod": "production",
		"stage": "staging",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			t.Setenv("APP_ENV", in)
			if got := AppEnv(); got != want {
				t.Fatalf("AppEnv() = %q, want %q", got, want)
			}
		})
	}
}

func TestLoadEnvLayering(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APP_ENV", "development")
	os.Unsetenv("LAYER_TEST")

	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(".env", "LAYER_TEST=base\n")
	write(".env.development", "LAYER_TEST=dev\n")

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	LoadEnv()

	if got := os.Getenv("LAYER_TEST"); got != "dev" {
		t.Fatalf("LAYER_TEST = %q, want dev", got)
	}
}

func TestProductionSkipsDotEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	if AppEnv() != "production" {
		t.Fatal("expected production")
	}
	// LoadEnv returns early for production; no panic is success.
	LoadEnv()
}
