package resource

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
	baseName := r.generateOutputFilename(fileRef.RelativePath)

	if outputFile != "" {
		return r.calculateOutputPathWithCustomDir(absInputPath, baseOutputDir, fileRef, baseName)
	}
	return r.calculateOutputPathSameStructure(baseOutputDir, fileRef, baseName), nil
}

// copyFileReferences copies all non-script file references to the output directory
func (r *Resource) copyFileReferences(baseOutputDir, absInputPath, outputFile string) (FileCopyBatchResult, error) {
	nonScriptFiles := r.getNonScriptFiles()
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
		copyResult := r.processSingleFile(fileRef, absInputPath, outputFile, baseOutputDir)
		result.Results = append(result.Results, copyResult)
		if copyResult.Success {
			result.SuccessCount++
			result.TotalSize += copyResult.Size
		} else {
			result.ErrorCount++
		}
	}

	return result, nil
}

// calculateFileOutputPath calculates the output path for a non-script file reference
func (r *Resource) calculateFileOutputPath(absInputPath, outputFile, baseOutputDir string, fileRef FileReference) (string, error) {
	if outputFile != "" {
		return r.calculateFileOutputPathWithCustomDir(absInputPath, baseOutputDir, fileRef)
	}
	return filepath.Join(baseOutputDir, fileRef.RelativePath), nil
}

// getNonScriptFiles returns all non-script file references
func (r *Resource) getNonScriptFiles() []FileReference {
	var nonScriptFiles []FileReference
	for _, fileRef := range r.Files {
		if fileRef.ReferenceType != ReferenceTypeScript {
			nonScriptFiles = append(nonScriptFiles, fileRef)
		}
	}
	return nonScriptFiles
}

// processSingleFile handles the copying of a single file and returns the result
func (r *Resource) processSingleFile(fileRef FileReference, absInputPath, outputFile, baseOutputDir string) FileCopyResult {
	copyResult := FileCopyResult{
		RelativePath: fileRef.RelativePath,
		Success:      false,
		Error:        nil,
		Size:         0,
	}

	outputPath, err := r.calculateFileOutputPath(absInputPath, outputFile, baseOutputDir, fileRef)
	if err != nil {
		copyResult.Error = fmt.Errorf("failed to calculate output path: %v", err)
		return copyResult
	}
	copyResult.OutputPath = outputPath

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		copyResult.Error = fmt.Errorf("failed to create output directory: %v", err)
		return copyResult
	}

	if err := copyFile(fileRef.FullPath, outputPath); err != nil {
		copyResult.Error = fmt.Errorf("failed to copy file: %v", err)
		return copyResult
	}

	if fileInfo, err := os.Stat(outputPath); err == nil {
		copyResult.Size = fileInfo.Size()
	}

	copyResult.Success = true
	return copyResult
}

// generateOutputFilename generates the output filename, converting .lua to .luac
func (r *Resource) generateOutputFilename(relativePath string) string {
	baseName := filepath.Base(relativePath)
	if filepath.Ext(baseName) == ".lua" {
		return baseName[:len(baseName)-4] + ".luac"
	}
	return baseName
}

// calculateOutputPathWithCustomDir calculates output path when a custom directory is specified
func (r *Resource) calculateOutputPathWithCustomDir(absInputPath, baseOutputDir string, fileRef FileReference, baseName string) (string, error) {
	relativeFromInput, err := filepath.Rel(absInputPath, r.BaseDir)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %v", err)
	}

	fullRelativeDir := buildFullRelativeDir(relativeFromInput, filepath.Dir(fileRef.RelativePath))

	if fullRelativeDir == "." || fullRelativeDir == "" {
		return filepath.Join(baseOutputDir, baseName), nil
	}
	return filepath.Join(baseOutputDir, fullRelativeDir, baseName), nil
}

// calculateOutputPathSameStructure calculates output path using the same directory structure as source
func (r *Resource) calculateOutputPathSameStructure(baseOutputDir string, fileRef FileReference, baseName string) string {
	relativeDir := filepath.Dir(fileRef.RelativePath)
	if relativeDir == "." {
		return filepath.Join(baseOutputDir, baseName)
	}
	return filepath.Join(baseOutputDir, relativeDir, baseName)
}

// calculateFileOutputPathWithCustomDir calculates file output path when a custom directory is specified
func (r *Resource) calculateFileOutputPathWithCustomDir(absInputPath, baseOutputDir string, fileRef FileReference) (string, error) {
	relativeFromInput, err := filepath.Rel(absInputPath, r.BaseDir)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %v", err)
	}

	var fullRelativePath string
	if relativeFromInput != "" && relativeFromInput != "." {
		fullRelativePath = filepath.Join(relativeFromInput, fileRef.RelativePath)
	} else {
		fullRelativePath = fileRef.RelativePath
	}

	return filepath.Join(baseOutputDir, fullRelativePath), nil
}

// buildFullRelativeDir builds the full relative directory path
func buildFullRelativeDir(relativeFromInput, relativeDir string) string {
	if relativeFromInput != "" && relativeFromInput != "." {
		if relativeDir == "." {
			return relativeFromInput
		}
		return filepath.Join(relativeFromInput, relativeDir)
	}
	return relativeDir
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
