package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// BinaryDetector handles detection and validation of the luac_mta binary
type BinaryDetector struct{}

// NewBinaryDetector creates a new binary detector instance
func NewBinaryDetector() *BinaryDetector {
	return &BinaryDetector{}
}

// DetectPath attempts to find the luac_mta binary
func (bd *BinaryDetector) DetectPath() (string, error) {
	var candidates []string

	// Platform-specific binary names
	if runtime.GOOS == "windows" {
		candidates = []string{
			"luac_mta.exe",
			"./luac_mta.exe",
			"./bin/luac_mta.exe",
			"C:\\bin\\luac_mta.exe",
		}
	} else {
		candidates = []string{
			"luac_mta",
			"./luac_mta",
			"./bin/luac_mta",
			"/usr/local/bin/luac_mta",
			"/usr/bin/luac_mta",
		}
	}

	// Check PATH first
	if path, err := exec.LookPath("luac_mta"); err == nil {
		return path, nil
	}

	// Check candidate locations
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("luac_mta binary not found in PATH or common locations")
}

// ValidatePath checks if the binary exists and is executable
func (bd *BinaryDetector) ValidatePath(binaryPath string) error {
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", binaryPath)
	}

	// Test if binary is executable by running with no arguments
	cmd := exec.Command(binaryPath)
	if err := cmd.Run(); err != nil {
		// luac_mta returns non-zero when no files are provided, which is expected
		if _, ok := err.(*exec.ExitError); ok {
			// Check if it's the expected "no input files" error
			return nil
		}
		return fmt.Errorf("binary is not executable: %w", err)
	}

	return nil
}

// DetectAndValidate performs both detection and validation in one step
func (bd *BinaryDetector) DetectAndValidate() (string, error) {
	path, err := bd.DetectPath()
	if err != nil {
		return "", err
	}

	if err := bd.ValidatePath(path); err != nil {
		return "", err
	}

	return path, nil
}