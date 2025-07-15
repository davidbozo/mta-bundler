package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// luaToLuacRegex is the compiled regex pattern for replacing .lua with .luac in src attributes
var luaToLuacRegex = regexp.MustCompile(`(src\s*=\s*"[^"]*?)\.lua(")|(src\s*=\s*'[^']*?)\.lua(')`)

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

// GetLuaFilesByType returns Lua script files grouped by type (client, server, shared)
func (r *Resource) GetLuaFilesByType() (client, server, shared []FileReference) {
	for _, script := range r.Meta.Scripts {
		if strings.ToLower(filepath.Ext(script.Src)) == ".lua" {
			fileRef := FileReference{
				FullPath:      filepath.Join(r.BaseDir, script.Src),
				ReferenceType: ReferenceTypeScript,
				RelativePath:  script.Src,
			}
			
			switch strings.ToLower(script.Type) {
			case "client":
				client = append(client, fileRef)
			case "server":
				server = append(server, fileRef)
			case "shared":
				shared = append(shared, fileRef)
			default:
				// Default to server if no type specified
				server = append(server, fileRef)
			}
		}
	}
	return client, server, shared
}

// Compile compiles all Lua scripts in the resource
func (r *Resource) Compile(compiler *CLICompiler, inputPath, outputFile string, options CompilationOptions, mergeMode bool) error {
	fmt.Printf("Compiling resource: %s\n", r.Name)
	fmt.Printf("Base directory: %s\n", r.BaseDir)

	if mergeMode {
		return r.compileMerged(compiler, inputPath, outputFile, options)
	} else {
		return r.compileIndividual(compiler, inputPath, outputFile, options)
	}
}

// compileIndividual compiles each file individually (original behavior)
func (r *Resource) compileIndividual(compiler *CLICompiler, inputPath, outputFile string, options CompilationOptions) error {
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

// compileMerged compiles scripts into client.luac and server.luac files
func (r *Resource) compileMerged(compiler *CLICompiler, inputPath, outputFile string, options CompilationOptions) error {
	// Get scripts grouped by type
	clientFiles, serverFiles, sharedFiles := r.GetLuaFilesByType()

	// Combine shared files with both client and server
	allClientFiles := append(clientFiles, sharedFiles...)
	allServerFiles := append(serverFiles, sharedFiles...)

	if len(allClientFiles) == 0 && len(allServerFiles) == 0 {
		fmt.Printf("  Warning: No Lua script files found in resource %s\n", r.Name)
		return nil
	}

	fmt.Printf("  Found %d client script(s), %d server script(s), %d shared script(s)\n", 
		len(clientFiles), len(serverFiles), len(sharedFiles))

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

	// Copy meta.xml file to output directory (will be updated for merged files)
	if err := r.copyMergedMetaFile(baseOutputDir, absInputPath, outputFile, len(allClientFiles) > 0, len(allServerFiles) > 0); err != nil {
		return fmt.Errorf("failed to copy meta.xml: %v", err)
	}

	// Copy all non-script file references to output directory
	if err := r.copyFileReferences(baseOutputDir, absInputPath, outputFile); err != nil {
		return fmt.Errorf("failed to copy file references: %v", err)
	}

	var successCount, errorCount int
	totalStartTime := time.Now()

	// Compile client files if any
	if len(allClientFiles) > 0 {
		clientOutputPath := filepath.Join(baseOutputDir, "client.luac")
		if outputFile != "" {
			relativeFromInput, err := filepath.Rel(absInputPath, r.BaseDir)
			if err == nil && relativeFromInput != "" && relativeFromInput != "." {
				clientOutputPath = filepath.Join(baseOutputDir, relativeFromInput, "client.luac")
			}
		}

		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(clientOutputPath), 0755); err != nil {
			fmt.Printf("    ✗ Failed to create client output directory: %v\n", err)
			errorCount++
		} else {
			// Get file paths for compilation
			var clientPaths []string
			for _, fileRef := range allClientFiles {
				clientPaths = append(clientPaths, fileRef.FullPath)
			}

			fmt.Printf("  Compiling client files to client.luac...\n")
			result, err := r.compileMergedFiles(compiler, clientPaths, clientOutputPath, options)
			if err != nil {
				fmt.Printf("    ✗ Client compilation failed: %v\n", err)
				errorCount++
			} else if result.Success {
				fmt.Printf("    ✓ Client compilation successful: client.luac (%v)\n", result.CompileTime)
				successCount++
			} else {
				fmt.Printf("    ✗ Client compilation failed: %v\n", result.Error)
				errorCount++
			}
		}
	}

	// Compile server files if any
	if len(allServerFiles) > 0 {
		serverOutputPath := filepath.Join(baseOutputDir, "server.luac")
		if outputFile != "" {
			relativeFromInput, err := filepath.Rel(absInputPath, r.BaseDir)
			if err == nil && relativeFromInput != "" && relativeFromInput != "." {
				serverOutputPath = filepath.Join(baseOutputDir, relativeFromInput, "server.luac")
			}
		}

		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(serverOutputPath), 0755); err != nil {
			fmt.Printf("    ✗ Failed to create server output directory: %v\n", err)
			errorCount++
		} else {
			// Get file paths for compilation
			var serverPaths []string
			for _, fileRef := range allServerFiles {
				serverPaths = append(serverPaths, fileRef.FullPath)
			}

			fmt.Printf("  Compiling server files to server.luac...\n")
			result, err := r.compileMergedFiles(compiler, serverPaths, serverOutputPath, options)
			if err != nil {
				fmt.Printf("    ✗ Server compilation failed: %v\n", err)
				errorCount++
			} else if result.Success {
				fmt.Printf("    ✓ Server compilation successful: server.luac (%v)\n", result.CompileTime)
				successCount++
			} else {
				fmt.Printf("    ✗ Server compilation failed: %v\n", result.Error)
				errorCount++
			}
		}
	}

	totalTime := time.Since(totalStartTime)
	fmt.Printf("  Merge compilation completed: %d successful, %d errors\n", successCount, errorCount)
	fmt.Printf("  Total time: %v\n", totalTime)

	if errorCount > 0 {
		return fmt.Errorf("compilation completed with %d errors", errorCount)
	}

	return nil
}

// compileMergedFiles compiles multiple Lua files into a single output file
func (r *Resource) compileMergedFiles(compiler *CLICompiler, filePaths []string, outputPath string, options CompilationOptions) (*CompilationResult, error) {
	startTime := time.Now()

	result := &CompilationResult{
		InputFile:  strings.Join(filePaths, ", "),
		OutputFile: outputPath,
	}

	// Validate input files
	if err := compiler.ValidateFiles(filePaths); err != nil {
		result.Error = err
		result.CompileTime = time.Since(startTime)
		return result, err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		result.Error = fmt.Errorf("failed to create output directory: %w", err)
		result.CompileTime = time.Since(startTime)
		return result, result.Error
	}

	// Build command arguments for merged compilation
	args := []string{"-o", outputPath}
	
	// Strip debug information
	if options.StripDebug {
		args = append(args, "-s")
	}

	// Obfuscation level
	switch options.ObfuscationLevel {
	case ObfuscationBasic:
		args = append(args, "-e")
	case ObfuscationEnhanced:
		args = append(args, "-e2")
	case ObfuscationMaximum:
		args = append(args, "-e3")
	case ObfuscationNone:
		// No obfuscation flag needed
	}

	// Suppress decompile warning
	if options.SuppressDecompileWarning {
		args = append(args, "-d")
	}

	// Add all input files
	args = append(args, filePaths...)

	// Execute compilation
	binaryPath, err := compiler.GetBinaryPath()
	if err != nil {
		result.Error = fmt.Errorf("failed to get binary path: %w", err)
		result.CompileTime = time.Since(startTime)
		return result, result.Error
	}

	cmd := exec.Command(binaryPath, args...)
	output, err := cmd.CombinedOutput()

	result.CompileTime = time.Since(startTime)

	if err != nil {
		result.Error = fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
		return result, result.Error
	}

	result.Success = true
	return result, nil
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

// copyMergedMetaFile copies the meta.xml file to the output directory and updates it for merged compilation
func (r *Resource) copyMergedMetaFile(baseOutputDir, absInputPath, outputFile string, hasClientFiles, hasServerFiles bool) error {
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

	// Copy and modify the meta.xml file for merged compilation
	if err := r.copyAndModifyMergedMetaFile(r.MetaXMLPath, outputPath, hasClientFiles, hasServerFiles); err != nil {
		return fmt.Errorf("failed to copy and modify meta.xml: %v", err)
	}

	fmt.Printf("  ✓ Copied and updated meta.xml for merged compilation\n")
	return nil
}

// copyAndModifyMergedMetaFile copies the meta.xml file and updates it for merged compilation
func (r *Resource) copyAndModifyMergedMetaFile(src, dst string, hasClientFiles, hasServerFiles bool) error {
	// Read the source meta.xml file
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source meta.xml: %v", err)
	}

	// Parse the XML
	var meta Meta
	err = xml.Unmarshal(content, &meta)
	if err != nil {
		return fmt.Errorf("failed to parse meta.xml: %v", err)
	}

	// Clear existing scripts and add merged ones
	meta.Scripts = nil
	
	if hasClientFiles {
		meta.Scripts = append(meta.Scripts, Script{
			Src:  "client.luac",
			Type: "client",
			Cache: "true",
		})
	}
	
	if hasServerFiles {
		meta.Scripts = append(meta.Scripts, Script{
			Src:  "server.luac",
			Type: "server",
			Cache: "true",
		})
	}

	// Marshal the modified XML
	modifiedContent, err := xml.MarshalIndent(meta, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal modified meta.xml: %v", err)
	}

	// Add XML header
	finalContent := []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(modifiedContent))

	// Write the modified content to the destination file
	err = os.WriteFile(dst, finalContent, 0644)
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
