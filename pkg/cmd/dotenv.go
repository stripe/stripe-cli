package cmd

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

// loadDotenvFromFlags is called by cobra.OnInitialize
func loadDotenvFromFlags() {
	// Decide which file to use
	path := ""
	explicitlyRequested := false

	switch {
	case envFile != "":
		path = envFile
		explicitlyRequested = true
	case dotenv:
		path = ".env"
		explicitlyRequested = true
	default:
		// Auto-load .env from current directory if it exists
		path = ".env"
	}

	// Check if file exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If explicitly requested via flag, this is an error
			if explicitlyRequested {
				panic(fmt.Errorf("failed to load %s: file not found", path))
			}
			return // missing file is fine when auto-loading
		}
		if explicitlyRequested {
			panic(fmt.Errorf("failed to stat %s: %w", path, err))
		}
		return
	}

	// Security check: ensure file is not world-readable (especially important for auto-loading)
	mode := fileInfo.Mode()
	if mode&0004 != 0 { // Check if world-readable bit is set
		log.WithFields(log.Fields{
			"prefix": "cmd.loadDotenvFromFlags",
			"path":   path,
			"mode":   fmt.Sprintf("%#o", mode.Perm()),
		}).Warn("Skipping .env file: file permissions are too permissive (world-readable). Run 'chmod 600 .env' to fix this.")

		// Only fail if explicitly requested
		if explicitlyRequested {
			panic(fmt.Errorf(".env file has insecure permissions (world-readable): %s. Run 'chmod 600 %s' to fix this", path, path))
		}
		return
	}

	env, err := godotenv.Read(path)
	if err != nil {
		// Cobra will print this and exit
		panic(fmt.Errorf("failed to load %s: %w", path, err))
	}

	log.WithFields(log.Fields{
		"prefix": "cmd.loadDotenvFromFlags",
		"path":   path,
	}).Debug("Loaded environment variables from .env file")

	// Print message when explicitly using --dotenv flag
	if dotenv {
		fmt.Printf("Loaded environment variables from %s\n", path)
	}

	// allowlist — adjust later if needed
	allowlist := []string{
		"STRIPE_SECRET_KEY",
		"STRIPE_DEVICE_NAME",
	}

	for _, k := range allowlist {
		if v, ok := env[k]; ok {
			// Don't override existing environment
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
	}
}
