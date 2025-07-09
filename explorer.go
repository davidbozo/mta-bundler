package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindMTAResourceMetas recursively searches for meta.xml files in MTA resources
// and returns a slice of their full paths
func FindMTAResourceMetas(rootDir string) ([]string, error) {
	var metaPaths []string

	// Check if the root directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", rootDir)
	}

	// Walk through the directory tree
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log the error but continue walking
			fmt.Printf("Warning: cannot access %s: %v\n", path, err)
			return nil
		}

		// Check if it's a meta.xml file
		if !info.IsDir() && strings.ToLower(info.Name()) == "meta.xml" {
			absPath, err := filepath.Abs(path)
			if err != nil {
				fmt.Printf("Warning: cannot get absolute path for %s: %v\n", path, err)
				metaPaths = append(metaPaths, path)
			} else {
				metaPaths = append(metaPaths, absPath)
			}
		}

		return nil
	})

	if err != nil {
		return metaPaths, fmt.Errorf("error walking directory tree: %v", err)
	}

	return metaPaths, nil
}
