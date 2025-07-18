package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// getBaseOutputDir determines the base output directory
func (r *Resource) getBaseOutputDir(outputFile string) (string, error) {
	if outputFile != "" {
		if filepath.IsAbs(outputFile) {
			return outputFile, nil
		} else {
			// If outputFile is relative, resolve it from current working directory
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get current working directory: %v", err)
			}
			return filepath.Join(cwd, outputFile), nil
		}
	} else {
		return r.BaseDir, nil
	}
}

// calculateOutputPath calculates the output path for a file reference
func (r *Resource) calculateOutputPath(absInputPath, outputFile, baseOutputDir string, fileRef FileReference) (string, error) {
	// Generate output filename
	baseName := filepath.Base(fileRef.RelativePath)
	if filepath.Ext(baseName) == ".lua" {
		baseName = baseName[:len(baseName)-4] + ".luac"
	}

	var outputPath string
	if outputFile != "" {
		// When output directory is specified, calculate relative path from inputPath
		relativeFromInput, err := filepath.Rel(absInputPath, r.BaseDir)
		if err != nil {
			return "", fmt.Errorf("failed to calculate relative path: %v", err)
		}

		// Build path: baseOutputDir + relativeFromInput + fileRef.RelativePath (but with .luac)
		relativeDir := filepath.Dir(fileRef.RelativePath)
		var fullRelativeDir string
		if relativeFromInput != "" && relativeFromInput != "." {
			if relativeDir == "." {
				fullRelativeDir = relativeFromInput
			} else {
				fullRelativeDir = filepath.Join(relativeFromInput, relativeDir)
			}
		} else {
			fullRelativeDir = relativeDir
		}

		if fullRelativeDir == "." || fullRelativeDir == "" {
			outputPath = filepath.Join(baseOutputDir, baseName)
		} else {
			outputPath = filepath.Join(baseOutputDir, fullRelativeDir, baseName)
		}
	} else {
		// Output to same directory structure as source
		relativeDir := filepath.Dir(fileRef.RelativePath)
		if relativeDir == "." {
			outputPath = filepath.Join(baseOutputDir, baseName)
		} else {
			outputPath = filepath.Join(baseOutputDir, relativeDir, baseName)
		}
	}

	return outputPath, nil
}

// copyFileReferences copies all non-script file references to the output directory
func (r *Resource) copyFileReferences(baseOutputDir, absInputPath, outputFile string) error {
	// Get all non-script file references
	var nonScriptFiles []FileReference
	for _, fileRef := range r.Files {
		if fileRef.ReferenceType != ReferenceTypeScript {
			nonScriptFiles = append(nonScriptFiles, fileRef)
		}
	}

	if len(nonScriptFiles) == 0 {
		return nil
	}

	fmt.Printf("  Copying %d non-script file(s)\n", len(nonScriptFiles))

	for _, fileRef := range nonScriptFiles {
		outputPath, err := r.calculateFileOutputPath(absInputPath, outputFile, baseOutputDir, fileRef)
		if err != nil {
			fmt.Printf("    ✗ Failed to calculate output path for %s: %v\n", fileRef.RelativePath, err)
			continue
		}

		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			fmt.Printf("    ✗ Failed to create output directory for %s: %v\n", fileRef.RelativePath, err)
			continue
		}

		// Copy the file
		if err := copyFile(fileRef.FullPath, outputPath); err != nil {
			fmt.Printf("    ✗ Failed to copy %s: %v\n", fileRef.RelativePath, err)
			continue
		}

		fmt.Printf("    ✓ Copied %s\n", fileRef.RelativePath)
	}

	return nil
}

// calculateFileOutputPath calculates the output path for a non-script file reference
func (r *Resource) calculateFileOutputPath(absInputPath, outputFile, baseOutputDir string, fileRef FileReference) (string, error) {
	var outputPath string

	if outputFile != "" {
		// When output directory is specified, calculate relative path from inputPath
		relativeFromInput, err := filepath.Rel(absInputPath, r.BaseDir)
		if err != nil {
			return "", fmt.Errorf("failed to calculate relative path: %v", err)
		}

		// Build path: baseOutputDir + relativeFromInput + fileRef.RelativePath
		var fullRelativePath string
		if relativeFromInput != "" && relativeFromInput != "." {
			fullRelativePath = filepath.Join(relativeFromInput, fileRef.RelativePath)
		} else {
			fullRelativePath = fileRef.RelativePath
		}

		outputPath = filepath.Join(baseOutputDir, fullRelativePath)
	} else {
		// Output to same directory structure as source
		outputPath = filepath.Join(baseOutputDir, fileRef.RelativePath)
	}

	return outputPath, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
