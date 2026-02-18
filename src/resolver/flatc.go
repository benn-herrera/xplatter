package resolver

import (
	"fmt"
	"os"
	"os/exec"
)

// ResolveFlatc finds the flatc binary using the resolution order:
// 1. Explicit flag path (if non-empty)
// 2. XPLATTER_FLATC_PATH environment variable
// 3. "flatc" in PATH
func ResolveFlatc(flagPath string) (string, error) {
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err != nil {
			return "", fmt.Errorf("flatc not found at specified path: %s", flagPath)
		}
		return flagPath, nil
	}

	if envPath := os.Getenv("XPLATTER_FLATC_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err != nil {
			return "", fmt.Errorf("flatc not found at XPLATTER_FLATC_PATH: %s", envPath)
		}
		return envPath, nil
	}

	path, err := exec.LookPath("flatc")
	if err != nil {
		return "", fmt.Errorf("flatc not found in PATH; set --flatc flag or XPLATTER_FLATC_PATH environment variable")
	}
	return path, nil
}
