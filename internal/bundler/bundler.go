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
	logResourceStart(r.Name, r.BaseDir)

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
		logLuaFilesFound(0, r.Name)
		return nil
	}

	logLuaFilesFound(len(luaFiles), r.Name)

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
	logFileCopyResults(copyResult)

	// Compile each file individually while preserving directory structure
	var successCount, errorCount int
	totalStartTime := time.Now()

	for _, fileRef := range luaFiles {
		logFileProcessing(fileRef.RelativePath)

		outputPath, err := r.CalculateOutputPath(absInputPath, outputFile, baseOutputDir, fileRef)
		if err != nil {
			logPathCalculationError(fileRef.RelativePath, err)
			errorCount++
			continue
		}

		// Ensure output subdirectory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			logDirectoryCreationError(err)
			errorCount++
			continue
		}

		// Compile the file
		result, err := rb.compiler.CompileFile(fileRef.FullPath, outputPath, rb.options)
		if err != nil {
			logFileError(fileRef.RelativePath, err)
			errorCount++
		} else if result.Success {
			// Show relative output path from baseOutputDir
			relativeOutputPath, err := filepath.Rel(baseOutputDir, outputPath)
			if err != nil {
				relativeOutputPath = filepath.Base(outputPath)
			}

			logFileSuccess(fileRef.RelativePath, relativeOutputPath, result)
			successCount++
		} else {
			logFileError(fileRef.RelativePath, result.Error)
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

	logIndividualCompilationSummary(successCount, errorCount, totalTime, totalInputSize, totalOutputSize)

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
		logLuaFilesFound(0, r.Name)
		return nil
	}

	logMergedFilesFound(len(clientFiles), len(serverFiles), len(sharedFiles))

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

	logFileCopyResults(copyResult)

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
			logMergedDirectoryCreationError("client", err)
			errorCount++
		} else {
			// Get file paths for compilation
			var clientPaths []string
			for _, fileRef := range allClientFiles {
				clientPaths = append(clientPaths, fileRef.FullPath)
			}

			logMergedCompilationStart("client")
			result, err := rb.compiler.Compile(clientPaths, clientOutputPath, rb.options)
			if err != nil {
				logMergedCompilationError("client", err)
				errorCount++
			} else if result.Success {
				logMergedCompilationSuccess("client", result)
				successCount++
			} else {
				logMergedCompilationError("client", result.Error)
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
			logMergedDirectoryCreationError("server", err)
			errorCount++
		} else {
			// Get file paths for compilation
			var serverPaths []string
			for _, fileRef := range allServerFiles {
				serverPaths = append(serverPaths, fileRef.FullPath)
			}

			logMergedCompilationStart("server")
			result, err := rb.compiler.Compile(serverPaths, serverOutputPath, rb.options)
			if err != nil {
				logMergedCompilationError("server", err)
				errorCount++
			} else if result.Success {
				logMergedCompilationSuccess("server", result)
				successCount++
			} else {
				logMergedCompilationError("server", result.Error)
				errorCount++
			}
		}
	}

	totalTime := time.Since(totalStartTime)
	logMergedCompilationSummary(successCount, errorCount, totalTime)

	if errorCount > 0 {
		return fmt.Errorf("compilation completed with %d errors", errorCount)
	}

	return nil
}

