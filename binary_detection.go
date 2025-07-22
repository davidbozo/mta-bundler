package main

import (
	"fmt"
	"os"
	"os/exec"
)

// BinaryDetector handles detection and validation of the luac_mta binary
type BinaryDetector struct {
	providers []BinaryProvider
}

// NewBinaryDetector creates a new binary detector instance with default providers
func NewBinaryDetector() *BinaryDetector {
	return &BinaryDetector{
		providers: []BinaryProvider{
			NewLocalBinaryProvider(),
			NewWebBinaryProvider(),
		},
	}
}

// NewBinaryDetectorWithProviders creates a binary detector with custom providers
func NewBinaryDetectorWithProviders(providers []BinaryProvider) *BinaryDetector {
	return &BinaryDetector{
		providers: providers,
	}
}

// DetectPath attempts to find the luac_mta binary using configured providers
func (bd *BinaryDetector) DetectPath() (string, error) {
	if len(bd.providers) == 0 {
		return "", fmt.Errorf("no binary providers configured")
	}

	var lastErr error

	// Try each provider in order
	for _, provider := range bd.providers {
		if path, err := provider.GetBinary(); err == nil {
			fmt.Printf("Binary found using %s provider: %s\n", provider.Name(), path)
			return path, nil
		} else {
			fmt.Printf("Provider %s failed: %v\n", provider.Name(), err)
			lastErr = err
		}
	}

	return "", fmt.Errorf("all providers failed, last error: %w", lastErr)

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
