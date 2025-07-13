package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Resource represents an MTA resource with its meta.xml and all file references
type Resource struct {
	MetaXMLPath string          // Path to the meta.xml file
	BaseDir     string          // Base directory of the resource
	Name        string          // Resource name (derived from directory name)
	Meta        Meta            // Parsed meta.xml structure
	Files       []FileReference // All file references from meta.xml
}

// NewResource creates a new Resource from a meta.xml file path
func NewResource(metaXMLPath string) (*Resource, error) {
	// Read the meta.xml file
	data, err := os.ReadFile(metaXMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read meta.xml: %w", err)
	}

	// Parse the XML
	var meta Meta
	err = xml.Unmarshal(data, &meta)
	if err != nil {
		return nil, fmt.Errorf("failed to parse meta.xml: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(metaXMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create resource
	baseDir := filepath.Dir(absPath)
	resourceName := filepath.Base(baseDir)

	resource := &Resource{
		MetaXMLPath: absPath,
		BaseDir:     baseDir,
		Name:        resourceName,
		Meta:        meta,
	}

	// Get all file references
	resource.Files, err = GetAllFiles(meta, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file references: %w", err)
	}

	return resource, nil
}

// GetLuaFiles returns all Lua script files from the resource
func (r *Resource) GetLuaFiles() []FileReference {
	var luaFiles []FileReference
	for _, fileRef := range r.Files {
		if fileRef.ReferenceType == "Script" && strings.ToLower(filepath.Ext(fileRef.FullPath)) == ".lua" {
			luaFiles = append(luaFiles, fileRef)
		}
	}
	return luaFiles
}

// Compile compiles all Lua scripts in the resource
func (r *Resource) Compile(compiler *CLICompiler, inputPath, outputFile string, options CompilationOptions) error {
	fmt.Printf("Compiling resource: %s\n", r.Name)
	fmt.Printf("Base directory: %s\n", r.BaseDir)

	// Get all Lua script files
	luaFiles := r.GetLuaFiles()
	if len(luaFiles) == 0 {
		fmt.Printf("  Warning: No Lua script files found in resource %s\n", r.Name)
		return nil
	}

	fmt.Printf("  Found %d Lua script(s) to compile\n", len(luaFiles))

	// Get absolute paths for calculation
	absInputPath, err := filepath.Abs(inputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute input path: %v", err)
	}

	// Determine base output directory
	baseOutputDir, err := r.getBaseOutputDir(outputFile)
	if err != nil {
		return err
	}

	// Create base output directory if it doesn't exist
	if err := os.MkdirAll(baseOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Compile each file individually while preserving directory structure
	var successCount, errorCount int
	totalStartTime := time.Now()

	for _, fileRef := range luaFiles {
		fmt.Printf("  Processing: %s\n", fileRef.RelativePath)

		outputPath, err := r.calculateOutputPath(absInputPath, outputFile, baseOutputDir, fileRef)
		if err != nil {
			fmt.Printf("    ✗ Failed to calculate output path: %v\n", err)
			errorCount++
			continue
		}

		// Ensure output subdirectory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			fmt.Printf("    ✗ Failed to create output directory: %v\n", err)
			errorCount++
			continue
		}

		// Compile the file
		result, err := compiler.CompileFile(fileRef.FullPath, outputPath, options)
		if err != nil {
			fmt.Printf("    ✗ %s: %v\n", fileRef.RelativePath, err)
			errorCount++
		} else if result.Success {
			// Show relative output path from baseOutputDir
			relativeOutputPath, err := filepath.Rel(baseOutputDir, outputPath)
			if err != nil {
				relativeOutputPath = filepath.Base(outputPath)
			}
			fmt.Printf("    ✓ %s -> %s (%v)\n", fileRef.RelativePath, relativeOutputPath, result.CompileTime)
			successCount++
		} else {
			fmt.Printf("    ✗ %s: %v\n", fileRef.RelativePath, result.Error)
			errorCount++
		}
	}

	totalTime := time.Since(totalStartTime)
	fmt.Printf("  Compilation completed: %d successful, %d errors\n", successCount, errorCount)
	fmt.Printf("  Total time: %v\n", totalTime)

	if errorCount > 0 {
		return fmt.Errorf("compilation completed with %d errors", errorCount)
	}

	return nil
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
	baseName := strings.TrimSuffix(filepath.Base(fileRef.RelativePath), ".lua")
	outputFileName := baseName + ".luac"

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
			outputPath = filepath.Join(baseOutputDir, outputFileName)
		} else {
			outputPath = filepath.Join(baseOutputDir, fullRelativeDir, outputFileName)
		}
	} else {
		// Output to same directory structure as source
		relativeDir := filepath.Dir(fileRef.RelativePath)
		if relativeDir == "." {
			outputPath = filepath.Join(baseOutputDir, outputFileName)
		} else {
			outputPath = filepath.Join(baseOutputDir, relativeDir, outputFileName)
		}
	}

	return outputPath, nil
}
