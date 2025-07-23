package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

			// Format size information
			sizeInfo := ""
			if result.InputSize > 0 && result.OutputSize > 0 {
				reduction := (1.0 - result.CompressionRatio) * 100
				if reduction > 0 {
					sizeInfo = fmt.Sprintf(" [%s → %s, %.0f%% reduction]",
						formatSize(result.InputSize), formatSize(result.OutputSize), reduction)
				} else {
					sizeInfo = fmt.Sprintf(" [%s → %s]",
						formatSize(result.InputSize), formatSize(result.OutputSize))
				}
			}

			fmt.Printf("    ✓ %s -> %s (%v)%s\n", fileRef.RelativePath, relativeOutputPath, result.CompileTime, sizeInfo)
			successCount++
		} else {
			fmt.Printf("    ✗ %s: %v\n", fileRef.RelativePath, result.Error)
			errorCount++
		}
	}

	totalTime := time.Since(totalStartTime)

	// Calculate resource-level size summary
	var totalInputSize, totalOutputSize int64
	for _, fileRef := range luaFiles {
		if info, err := os.Stat(fileRef.FullPath); err == nil {
			totalInputSize += info.Size()
		}
	}

	// Sum up output sizes from successful compilations
	for _, fileRef := range luaFiles {
		outputPath, err := r.calculateOutputPath(absInputPath, outputFile, baseOutputDir, fileRef)
		if err == nil {
			if info, err := os.Stat(outputPath); err == nil {
				totalOutputSize += info.Size()
			}
		}
	}

	fmt.Printf("  Compilation completed: %d successful, %d errors\n", successCount, errorCount)
	if totalInputSize > 0 && totalOutputSize > 0 && successCount > 0 {
		reduction := (1.0 - float64(totalOutputSize)/float64(totalInputSize)) * 100
		fmt.Printf("  Resource size summary: %s \u2192 %s (%.0f%% reduction)\n",
			formatSize(totalInputSize), formatSize(totalOutputSize), reduction)
	}
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
				// Format size information for merged client files
				sizeInfo := ""
				if result.InputSize > 0 && result.OutputSize > 0 {
					reduction := (1.0 - result.CompressionRatio) * 100
					if reduction > 0 {
						sizeInfo = fmt.Sprintf(" [%s → %s, %.0f%% reduction]",
							formatSize(result.InputSize), formatSize(result.OutputSize), reduction)
					} else {
						sizeInfo = fmt.Sprintf(" [%s → %s, %.0f%% reduction]",
							formatSize(result.InputSize), formatSize(result.OutputSize), reduction)
					}
				}
				fmt.Printf("    ✓ Client compilation successful: client.luac (%v)%s\n", result.CompileTime, sizeInfo)
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
				// Format size information for merged server files
				sizeInfo := ""
				if result.InputSize > 0 && result.OutputSize > 0 {
					reduction := (1.0 - result.CompressionRatio) * 100
					if reduction > 0 {
						sizeInfo = fmt.Sprintf(" [%s → %s, %.0f%% reduction]",
							formatSize(result.InputSize), formatSize(result.OutputSize), reduction)
					} else {
						sizeInfo = fmt.Sprintf(" [%s → %s]",
							formatSize(result.InputSize), formatSize(result.OutputSize))
					}
				}
				fmt.Printf("    ✓ Server compilation successful: server.luac (%v)%s\n", result.CompileTime, sizeInfo)
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

	// Calculate total input size
	if inputSize, err := calculateTotalSize(filePaths); err == nil {
		result.InputSize = inputSize
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

	// Calculate output file size and update metrics
	if outputSize, err := calculateFileSize(outputPath); err == nil {
		result.OutputSize = outputSize
		updateSizeMetrics(result)
	}

	return result, nil
}
