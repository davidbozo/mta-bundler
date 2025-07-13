package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
		if fileRef.ReferenceType == ReferenceTypeScript && strings.ToLower(filepath.Ext(fileRef.FullPath)) == ".lua" {
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

	// Copy meta.xml file to output directory
	if err := r.copyMetaFile(baseOutputDir, absInputPath, outputFile); err != nil {
		return fmt.Errorf("failed to copy meta.xml: %v", err)
	}

	// Copy all non-script file references to output directory
	if err := r.copyFileReferences(baseOutputDir, absInputPath, outputFile); err != nil {
		return fmt.Errorf("failed to copy file references: %v", err)
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

// copyMetaFile copies the meta.xml file to the output directory and updates lua file references to luac
func (r *Resource) copyMetaFile(baseOutputDir, absInputPath, outputFile string) error {
	// Calculate the output path for meta.xml
	var outputPath string
	
	if outputFile != "" {
		// When output directory is specified, calculate relative path from inputPath
		relativeFromInput, err := filepath.Rel(absInputPath, r.BaseDir)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %v", err)
		}

		if relativeFromInput == "" || relativeFromInput == "." {
			outputPath = filepath.Join(baseOutputDir, "meta.xml")
		} else {
			outputPath = filepath.Join(baseOutputDir, relativeFromInput, "meta.xml")
		}
	} else {
		// Output to same directory structure as source
		outputPath = filepath.Join(baseOutputDir, "meta.xml")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory for meta.xml: %v", err)
	}

	// Copy and modify the meta.xml file
	if err := r.copyAndModifyMetaFile(r.MetaXMLPath, outputPath); err != nil {
		return fmt.Errorf("failed to copy and modify meta.xml: %v", err)
	}

	fmt.Printf("  ✓ Copied and updated meta.xml\n")
	return nil
}

// copyAndModifyMetaFile copies the meta.xml file and updates .lua file extensions to .luac using regex
func (r *Resource) copyAndModifyMetaFile(src, dst string) error {
	// Read the source meta.xml file
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source meta.xml: %v", err)
	}

	// Convert to string for regex processing
	metaContent := string(content)

	// Use regex to replace .lua with .luac in src attributes
	// Match both single and double quoted src attributes ending with .lua
	luaToLuacRegex := regexp.MustCompile(`(src\s*=\s*"[^"]*?)\.lua(")|(src\s*=\s*'[^']*?)\.lua(')`)
	
	// Replace .lua with .luac while preserving the quotes
	modifiedContent := luaToLuacRegex.ReplaceAllStringFunc(metaContent, func(match string) string {
		if strings.Contains(match, `"`) {
			return strings.Replace(match, ".lua\"", ".luac\"", 1)
		} else {
			return strings.Replace(match, ".lua'", ".luac'", 1)
		}
	})

	// Write the modified content to the destination file
	err = os.WriteFile(dst, []byte(modifiedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write modified meta.xml: %v", err)
	}

	return nil
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
