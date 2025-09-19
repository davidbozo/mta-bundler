package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/davidbozo/mta-bundler/internal/compiler"
	"github.com/davidbozo/mta-bundler/internal/resource"
)

// ResourceBundler orchestrates the compilation of MTA resources
type ResourceBundler struct {
	compiler compiler.CLICompiler
	options  compiler.CompilationOptions
}

// NewResourceBundler creates a new ResourceBundler instance
func NewResourceBundler(comp compiler.CLICompiler, options compiler.CompilationOptions) ResourceBundler {
	return ResourceBundler{
		compiler: comp,
		options:  options,
	}
}

// CompileResource compiles an MTA resource using either individual or merged mode
func (rb ResourceBundler) CompileResource(r *resource.Resource, inputPath, outputFile string, mergeMode bool) error {
	fmt.Printf("Compiling resource: %s\n", r.Name)
	fmt.Printf("Base directory: %s\n", r.BaseDir)

	if mergeMode {
		return rb.compileMerged(r, inputPath, outputFile)
	} else {
		return rb.compileIndividual(r, inputPath, outputFile)
	}
}

// compileIndividual compiles each file individually (original behavior)
func (rb ResourceBundler) compileIndividual(r *resource.Resource, inputPath, outputFile string) error {
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
	baseOutputDir, err := r.GetBaseOutputDir(outputFile)
	if err != nil {
		return err
	}

	// Create base output directory if it doesn't exist
	if err := os.MkdirAll(baseOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Copy meta.xml file to output directory
	if err := r.CopyMetaFile(baseOutputDir, absInputPath, outputFile); err != nil {
		return fmt.Errorf("failed to copy meta.xml: %v", err)
	}

	// Copy all non-script file references to output directory
	copyResult, err := r.CopyFileReferences(baseOutputDir, absInputPath, outputFile)
	if err != nil {
		return fmt.Errorf("failed to copy file references: %v", err)
	}

	// Log file copy results
	rb.printFileCopyResults(copyResult)

	// Compile each file individually while preserving directory structure
	var successCount, errorCount int
	totalStartTime := time.Now()

	for _, fileRef := range luaFiles {
		fmt.Printf("  Processing: %s\n", fileRef.RelativePath)

		outputPath, err := r.CalculateOutputPath(absInputPath, outputFile, baseOutputDir, fileRef)
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
		result, err := rb.compiler.CompileFile(fileRef.FullPath, outputPath, rb.options)
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
				reduction := (1.0 - result.CompressionRatio()) * 100
				if reduction > 0 {
					sizeInfo = fmt.Sprintf(" [%s → %s, %.0f%% reduction]",
						compiler.FormatSize(result.InputSize), compiler.FormatSize(result.OutputSize), reduction)
				} else {
					sizeInfo = fmt.Sprintf(" [%s → %s]",
						compiler.FormatSize(result.InputSize), compiler.FormatSize(result.OutputSize))
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
		outputPath, err := r.CalculateOutputPath(absInputPath, outputFile, baseOutputDir, fileRef)
		if err == nil {
			if info, err := os.Stat(outputPath); err == nil {
				totalOutputSize += info.Size()
			}
		}
	}

	fmt.Printf("  Compilation completed: %d successful, %d errors\n", successCount, errorCount)
	if totalInputSize > 0 && totalOutputSize > 0 && successCount > 0 {
		reduction := (1.0 - float64(totalOutputSize)/float64(totalInputSize)) * 100
		fmt.Printf("  Resource size summary: %s → %s (%.0f%% reduction)\n",
			compiler.FormatSize(totalInputSize), compiler.FormatSize(totalOutputSize), reduction)
	}
	fmt.Printf("  Total time: %v\n", totalTime)

	if errorCount > 0 {
		return fmt.Errorf("compilation completed with %d errors", errorCount)
	}

	return nil
}

// compileMerged compiles scripts into client.luac and server.luac files
func (rb ResourceBundler) compileMerged(r *resource.Resource, inputPath, outputFile string) error {
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
	baseOutputDir, err := r.GetBaseOutputDir(outputFile)
	if err != nil {
		return err
	}

	// Create base output directory if it doesn't exist
	if err := os.MkdirAll(baseOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Copy meta.xml file to output directory (will be updated for merged files)
	if err := r.CopyMergedMetaFile(baseOutputDir, absInputPath, outputFile, len(allClientFiles) > 0, len(allServerFiles) > 0); err != nil {
		return fmt.Errorf("failed to copy meta.xml: %v", err)
	}

	// Copy all non-script file references to output directory
	copyResult, err := r.CopyFileReferences(baseOutputDir, absInputPath, outputFile)
	if err != nil {
		return fmt.Errorf("failed to copy file references: %v", err)
	}

	rb.printFileCopyResults(copyResult)

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
			result, err := rb.compiler.Compile(clientPaths, clientOutputPath, rb.options)
			if err != nil {
				fmt.Printf("    ✗ Client compilation failed: %v\n", err)
				errorCount++
			} else if result.Success {
				// Format size information for merged client files
				sizeInfo := ""
				if result.InputSize > 0 && result.OutputSize > 0 {
					reduction := (1.0 - result.CompressionRatio()) * 100
					if reduction > 0 {
						sizeInfo = fmt.Sprintf(" [%s → %s, %.0f%% reduction]",
							compiler.FormatSize(result.InputSize), compiler.FormatSize(result.OutputSize), reduction)
					} else {
						sizeInfo = fmt.Sprintf(" [%s → %s, %.0f%% reduction]",
							compiler.FormatSize(result.InputSize), compiler.FormatSize(result.OutputSize), reduction)
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
			result, err := rb.compiler.Compile(serverPaths, serverOutputPath, rb.options)
			if err != nil {
				fmt.Printf("    ✗ Server compilation failed: %v\n", err)
				errorCount++
			} else if result.Success {
				// Format size information for merged server files
				sizeInfo := ""
				if result.InputSize > 0 && result.OutputSize > 0 {
					reduction := (1.0 - result.CompressionRatio()) * 100
					if reduction > 0 {
						sizeInfo = fmt.Sprintf(" [%s → %s, %.0f%% reduction]",
							compiler.FormatSize(result.InputSize), compiler.FormatSize(result.OutputSize), reduction)
					} else {
						sizeInfo = fmt.Sprintf(" [%s → %s]",
							compiler.FormatSize(result.InputSize), compiler.FormatSize(result.OutputSize))
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

// printFileCopyResults prints the results of file copy operations
func (rb ResourceBundler) printFileCopyResults(result resource.FileCopyBatchResult) {
	if result.TotalFiles == 0 {
		return
	}

	fmt.Printf("  Copying %d non-script file(s)\n", result.TotalFiles)
	for _, copyResult := range result.Results {
		if copyResult.Success {
			fmt.Printf("    ✓ Copied %s\n", copyResult.RelativePath)
		} else {
			fmt.Printf("    ✗ Failed to copy %s: %v\n", copyResult.RelativePath, copyResult.Error)
		}
	}
}
