package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileCopyResult represents the result of copying a single non-Lua file (images, models, textures, etc.)
// from an MTA resource. Lua script files are handled separately through compilation processes.
type FileCopyResult struct {
	RelativePath string // Original relative path from meta.xml
	OutputPath   string // Full output path where file was copied
	Success      bool   // Whether the copy operation succeeded
	Error        error  // Error if copy failed
	Size         int64  // Size of the copied file in bytes
}

// FileCopyBatchResult represents the result of copying multiple non-Lua files (images, models, textures, etc.)
// from an MTA resource. Lua script files are handled separately through compilation processes.
type FileCopyBatchResult struct {
	Results      []FileCopyResult // Individual copy results for files
	TotalFiles   int              // Total number of files processed
	SuccessCount int              // Number of successful copies
	ErrorCount   int              // Number of failed copies
	TotalSize    int64            // Total size of all successfully copied files
}

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
func (r *Resource) copyFileReferences(baseOutputDir, absInputPath, outputFile string) (FileCopyBatchResult, error) {
	// Get all non-script file references
	var nonScriptFiles []FileReference
	for _, fileRef := range r.Files {
		if fileRef.ReferenceType != ReferenceTypeScript {
			nonScriptFiles = append(nonScriptFiles, fileRef)
		}
	}

	result := FileCopyBatchResult{
		Results:      make([]FileCopyResult, 0, len(nonScriptFiles)),
		TotalFiles:   len(nonScriptFiles),
		SuccessCount: 0,
		ErrorCount:   0,
		TotalSize:    0,
	}

	if len(nonScriptFiles) == 0 {
		return result, nil
	}

	for _, fileRef := range nonScriptFiles {
		copyResult := FileCopyResult{
			RelativePath: fileRef.RelativePath,
			Success:      false,
			Error:        nil,
			Size:         0,
		}

		outputPath, err := r.calculateFileOutputPath(absInputPath, outputFile, baseOutputDir, fileRef)
		if err != nil {
			copyResult.Error = fmt.Errorf("failed to calculate output path: %v", err)
			result.Results = append(result.Results, copyResult)
			result.ErrorCount++
			continue
		}
		copyResult.OutputPath = outputPath

		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			copyResult.Error = fmt.Errorf("failed to create output directory: %v", err)
			result.Results = append(result.Results, copyResult)
			result.ErrorCount++
			continue
		}

		// Copy the file
		if err := copyFile(fileRef.FullPath, outputPath); err != nil {
			copyResult.Error = fmt.Errorf("failed to copy file: %v", err)
			result.Results = append(result.Results, copyResult)
			result.ErrorCount++
			continue
		}

		// Get file size
		if fileInfo, err := os.Stat(outputPath); err == nil {
			copyResult.Size = fileInfo.Size()
			result.TotalSize += copyResult.Size
		}

		copyResult.Success = true
		result.Results = append(result.Results, copyResult)
		result.SuccessCount++
	}

	return result, nil
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
