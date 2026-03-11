package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDotenvAutoLoad(t *testing.T) {
	// Save and restore global state
	oldDotenv := dotenv
	oldEnvFile := envFile
	defer func() {
		dotenv = oldDotenv
		envFile = oldEnvFile
	}()

	// Create a temp directory
	tmpDir := t.TempDir()
	prevDir, _ := os.Getwd()
	defer os.Chdir(prevDir)
	os.Chdir(tmpDir)

	// Create a .env file with secure permissions
	envContent := "STRIPE_SECRET_KEY=sk_test_123\nSTRIPE_DEVICE_NAME=test_device\n"
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte(envContent), 0600)
	require.NoError(t, err)

	// Clear any existing env vars
	os.Unsetenv("STRIPE_SECRET_KEY")
	os.Unsetenv("STRIPE_DEVICE_NAME")

	// Test auto-loading
	dotenv = false
	envFile = ""
	loadDotenvFromFlags()

	// Verify env vars were loaded
	require.Equal(t, "sk_test_123", os.Getenv("STRIPE_SECRET_KEY"))
	require.Equal(t, "test_device", os.Getenv("STRIPE_DEVICE_NAME"))
}

func TestLoadDotenvExplicitFlag(t *testing.T) {
	// Save and restore global state
	oldDotenv := dotenv
	oldEnvFile := envFile
	defer func() {
		dotenv = oldDotenv
		envFile = oldEnvFile
	}()

	tmpDir := t.TempDir()
	prevDir, _ := os.Getwd()
	defer os.Chdir(prevDir)
	os.Chdir(tmpDir)

	envContent := "STRIPE_SECRET_KEY=sk_test_explicit\n"
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte(envContent), 0600)
	require.NoError(t, err)

	os.Unsetenv("STRIPE_SECRET_KEY")

	// Test explicit --dotenv flag
	dotenv = true
	envFile = ""
	loadDotenvFromFlags()

	require.Equal(t, "sk_test_explicit", os.Getenv("STRIPE_SECRET_KEY"))
}

func TestLoadDotenvCustomFile(t *testing.T) {
	// Save and restore global state
	oldDotenv := dotenv
	oldEnvFile := envFile
	defer func() {
		dotenv = oldDotenv
		envFile = oldEnvFile
	}()

	tmpDir := t.TempDir()

	envContent := "STRIPE_SECRET_KEY=sk_test_custom\n"
	customPath := filepath.Join(tmpDir, "custom.env")
	err := os.WriteFile(customPath, []byte(envContent), 0600)
	require.NoError(t, err)

	os.Unsetenv("STRIPE_SECRET_KEY")

	// Test custom --env-file
	dotenv = false
	envFile = customPath
	loadDotenvFromFlags()

	require.Equal(t, "sk_test_custom", os.Getenv("STRIPE_SECRET_KEY"))
}

func TestLoadDotenvInsecurePermissions(t *testing.T) {
	// Save and restore global state
	oldDotenv := dotenv
	oldEnvFile := envFile
	defer func() {
		dotenv = oldDotenv
		envFile = oldEnvFile
	}()

	tmpDir := t.TempDir()
	prevDir, _ := os.Getwd()
	defer os.Chdir(prevDir)
	os.Chdir(tmpDir)

	// Create a world-readable .env file
	envContent := "STRIPE_SECRET_KEY=sk_test_insecure\n"
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte(envContent), 0644)
	require.NoError(t, err)

	os.Unsetenv("STRIPE_SECRET_KEY")

	// Auto-load should skip insecure file
	dotenv = false
	envFile = ""
	loadDotenvFromFlags()

	// Env var should NOT be set
	require.Equal(t, "", os.Getenv("STRIPE_SECRET_KEY"))
}

func TestLoadDotenvInsecurePermissionsExplicit(t *testing.T) {
	// Save and restore global state
	oldDotenv := dotenv
	oldEnvFile := envFile
	defer func() {
		dotenv = oldDotenv
		envFile = oldEnvFile
	}()

	tmpDir := t.TempDir()

	// Create a world-readable file
	envContent := "STRIPE_SECRET_KEY=sk_test_insecure\n"
	envPath := filepath.Join(tmpDir, "insecure.env")
	err := os.WriteFile(envPath, []byte(envContent), 0644)
	require.NoError(t, err)

	// Explicit request should panic
	dotenv = false
	envFile = envPath

	require.Panics(t, func() {
		loadDotenvFromFlags()
	}, "Should panic on insecure permissions when explicitly requested")
}

func TestLoadDotenvMissingFile(t *testing.T) {
	// Save and restore global state
	oldDotenv := dotenv
	oldEnvFile := envFile
	defer func() {
		dotenv = oldDotenv
		envFile = oldEnvFile
	}()

	tmpDir := t.TempDir()
	prevDir, _ := os.Getwd()
	defer os.Chdir(prevDir)
	os.Chdir(tmpDir)

	// Auto-load with missing file should not panic
	dotenv = false
	envFile = ""
	require.NotPanics(t, func() {
		loadDotenvFromFlags()
	})
}

func TestLoadDotenvMissingFileExplicit(t *testing.T) {
	// Save and restore global state
	oldDotenv := dotenv
	oldEnvFile := envFile
	defer func() {
		dotenv = oldDotenv
		envFile = oldEnvFile
	}()

	dotenv = false
	envFile = "/nonexistent/file.env"

	require.Panics(t, func() {
		loadDotenvFromFlags()
	}, "Should panic when explicitly requested file is missing")
}

func TestLoadDotenvNoOverride(t *testing.T) {
	// Save and restore global state
	oldDotenv := dotenv
	oldEnvFile := envFile
	defer func() {
		dotenv = oldDotenv
		envFile = oldEnvFile
	}()

	tmpDir := t.TempDir()

	envContent := "STRIPE_SECRET_KEY=sk_test_from_file\n"
	envPath := filepath.Join(tmpDir, "test.env")
	err := os.WriteFile(envPath, []byte(envContent), 0600)
	require.NoError(t, err)

	// Set env var before loading
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_existing")

	dotenv = false
	envFile = envPath
	loadDotenvFromFlags()

	// Should NOT override existing env var
	require.Equal(t, "sk_test_existing", os.Getenv("STRIPE_SECRET_KEY"))
}
