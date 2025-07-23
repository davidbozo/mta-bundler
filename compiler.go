package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CompilationMode defines how files should be compiled
type CompilationMode int

const (
	// ModeIndividual compiles each file to its own output, preserving directory structure
	ModeIndividual CompilationMode = iota
	// ModeMerged compiles all files into a single output file
	ModeMerged
)

// ObfuscationLevel defines the level of code obfuscation
type ObfuscationLevel int

const (
	// ObfuscationNone - No obfuscation
	ObfuscationNone ObfuscationLevel = iota
	// ObfuscationBasic - Basic obfuscation (luac_mta -e)
	ObfuscationBasic
	// ObfuscationEnhanced - Enhanced obfuscation (luac_mta -e2, available from MTA 1.5.2-9.07903)
	ObfuscationEnhanced
	// ObfuscationMaximum - Maximum obfuscation (luac_mta -e3, available from MTA 1.5.6-9.18728)
	ObfuscationMaximum
)

// CompilationOptions holds configuration for the compilation process
type CompilationOptions struct {
	// ObfuscationLevel defines the level of code obfuscation
	ObfuscationLevel ObfuscationLevel
	// StripDebug removes debug information
	StripDebug bool
	// SuppressDecompileWarning suppresses decompile warnings
	SuppressDecompileWarning bool
	// Mode determines how files are compiled
	Mode CompilationMode
	// OutputPath is the output file path (for merged mode) or output directory (for individual mode)
	OutputPath string
	// BinaryPath is the path to luac_mta executable (optional, will auto-detect)
	BinaryPath string
}

// CompilationResult holds the result of a single file compilation operation
type CompilationResult struct {
	InputFile        string
	OutputFile       string
	Success          bool
	Error            error
	CompileTime      time.Duration
	InputSize        int64   // Size before compilation in bytes
	OutputSize       int64   // Size after compilation in bytes
	CompressionRatio float64 // Compression ratio (0-1, where 0.2 = 20% of original size)
}

// BatchCompilationResult holds the results of multiple file compilations
type BatchCompilationResult struct {
	Results         []CompilationResult
	TotalTime       time.Duration
	SuccessCount    int
	ErrorCount      int
	TotalInputSize  int64   // Total size before compilation
	TotalOutputSize int64   // Total size after compilation
	TotalRatio      float64 // Overall compression ratio
}

// LuaCompiler interface defines the contract for Lua compilation
type LuaCompiler interface {
	// Compile compiles the given Lua files according to the provided options
	Compile(filePaths []string, options CompilationOptions) (*BatchCompilationResult, error)
	// CompileFile compiles a single Lua file
	CompileFile(filePath string, outputPath string, options CompilationOptions) (*CompilationResult, error)
	// ValidateFiles checks if all provided files exist and are valid
	ValidateFiles(filePaths []string) error
	// GetBinaryPath returns the path to the luac_mta binary
	GetBinaryPath() (string, error)
}

// CLICompiler implements LuaCompiler using the luac_mta CLI binary
type CLICompiler struct {
	binaryPath string
}

// NewCLICompiler creates a new CLI-based Lua compiler
func NewCLICompiler(binaryPath string) (*CLICompiler, error) {
	if binaryPath == "" {
		return nil, fmt.Errorf("binaryPath cannot be empty")
	}

	compiler := &CLICompiler{
		binaryPath: binaryPath,
	}

	return compiler, nil
}


// GetBinaryPath returns the path to the luac_mta binary
func (c *CLICompiler) GetBinaryPath() (string, error) {
	return c.binaryPath, nil
}

// ValidateFiles checks if all provided files exist and are Lua files
func (c *CLICompiler) ValidateFiles(filePaths []string) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("no files provided")
	}

	var errors []string
	for _, path := range filePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("file not found: %s", path))
			continue
		}

		if !strings.HasSuffix(strings.ToLower(path), ".lua") {
			errors = append(errors, fmt.Sprintf("not a Lua file: %s", path))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Compile compiles the given Lua files according to the provided options
func (c *CLICompiler) Compile(filePaths []string, options CompilationOptions) (*BatchCompilationResult, error) {
	startTime := time.Now()

	result := &BatchCompilationResult{
		Results: make([]CompilationResult, 0),
	}

	// Validate input files
	if err := c.ValidateFiles(filePaths); err != nil {
		return result, err
	}

	switch options.Mode {
	case ModeMerged:
		return c.compileMerged(filePaths, options, result, startTime)
	case ModeIndividual:
		return c.compileIndividual(filePaths, options, result, startTime)
	default:
		return result, fmt.Errorf("unsupported compilation mode: %d", options.Mode)
	}
}

// CompileFile compiles a single Lua file
func (c *CLICompiler) CompileFile(filePath string, outputPath string, options CompilationOptions) (*CompilationResult, error) {
	startTime := time.Now()

	result := &CompilationResult{
		InputFile:  filePath,
		OutputFile: outputPath,
	}

	// Calculate input file size
	if inputSize, err := calculateFileSize(filePath); err == nil {
		result.InputSize = inputSize
	}

	// Validate input file
	if err := c.ValidateFiles([]string{filePath}); err != nil {
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

	// Build command arguments
	args := c.buildArgs(options, outputPath)
	args = append(args, filePath)

	// Execute compilation
	cmd := exec.Command(c.binaryPath, args...)
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

// compileMerged compiles all files into a single output file
func (c *CLICompiler) compileMerged(filePaths []string, options CompilationOptions, batchResult *BatchCompilationResult, startTime time.Time) (*BatchCompilationResult, error) {
	outputPath := options.OutputPath
	if outputPath == "" {
		outputPath = "compiled.luac"
	}

	// Create a single compilation result for the merged operation
	result := CompilationResult{
		InputFile:  strings.Join(filePaths, ", "),
		OutputFile: outputPath,
	}

	// Calculate total input size
	if inputSize, err := calculateTotalSize(filePaths); err == nil {
		result.InputSize = inputSize
	}

	// Build command arguments
	args := c.buildArgs(options, outputPath)
	args = append(args, filePaths...)

	// Execute compilation
	cmd := exec.Command(c.binaryPath, args...)
	output, err := cmd.CombinedOutput()

	result.CompileTime = time.Since(startTime)
	batchResult.TotalTime = result.CompileTime

	if err != nil {
		result.Error = fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
		batchResult.ErrorCount = 1
	} else {
		result.Success = true
		batchResult.SuccessCount = 1
		
		// Calculate output file size and update metrics
		if outputSize, err := calculateFileSize(outputPath); err == nil {
			result.OutputSize = outputSize
			updateSizeMetrics(&result)
		}
	}

	batchResult.Results = append(batchResult.Results, result)
	
	// Update batch size metrics
	updateBatchSizeMetrics(batchResult)

	if err != nil {
		return batchResult, result.Error
	}

	return batchResult, nil
}

// compileIndividual compiles each file to its own output, preserving directory structure
func (c *CLICompiler) compileIndividual(filePaths []string, options CompilationOptions, batchResult *BatchCompilationResult, startTime time.Time) (*BatchCompilationResult, error) {
	outputDir := options.OutputPath
	if outputDir == "" {
		outputDir = "compiled"
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return batchResult, fmt.Errorf("failed to create output directory: %w", err)
	}

	var hasErrors bool

	for _, inputPath := range filePaths {
		fileStartTime := time.Now()

		// Calculate output path, preserving directory structure
		relPath, err := filepath.Rel(".", inputPath)
		if err != nil {
			relPath = filepath.Base(inputPath)
		}

		// Change extension to .luac
		outputPath := filepath.Join(outputDir, strings.TrimSuffix(relPath, ".lua")+".luac")

		result := CompilationResult{
			InputFile:  inputPath,
			OutputFile: outputPath,
		}

		// Calculate input file size
		if inputSize, err := calculateFileSize(inputPath); err == nil {
			result.InputSize = inputSize
		}

		// Ensure output subdirectory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			result.Error = fmt.Errorf("failed to create output subdirectory: %w", err)
			result.CompileTime = time.Since(fileStartTime)
			batchResult.Results = append(batchResult.Results, result)
			batchResult.ErrorCount++
			hasErrors = true
			continue
		}

		// Build command arguments for this file
		args := c.buildArgs(options, outputPath)
		args = append(args, inputPath)

		// Execute compilation
		cmd := exec.Command(c.binaryPath, args...)
		output, err := cmd.CombinedOutput()

		result.CompileTime = time.Since(fileStartTime)

		if err != nil {
			result.Error = fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
			batchResult.ErrorCount++
			hasErrors = true
		} else {
			result.Success = true
			batchResult.SuccessCount++
			
			// Calculate output file size and update metrics
			if outputSize, err := calculateFileSize(outputPath); err == nil {
				result.OutputSize = outputSize
				updateSizeMetrics(&result)
			}
		}

		batchResult.Results = append(batchResult.Results, result)
	}

	batchResult.TotalTime = time.Since(startTime)
	
	// Update batch size metrics
	updateBatchSizeMetrics(batchResult)

	if hasErrors {
		return batchResult, fmt.Errorf("compilation completed with %d errors out of %d files", batchResult.ErrorCount, len(filePaths))
	}

	return batchResult, nil
}

// buildArgs builds the command line arguments for luac_mta
func (c *CLICompiler) buildArgs(options CompilationOptions, outputPath string) []string {
	var args []string

	// Output file
	args = append(args, "-o", outputPath)

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

	return args
}

// calculateFileSize returns the size of a file in bytes
func calculateFileSize(filePath string) (int64, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}
	return fileInfo.Size(), nil
}

// calculateTotalSize returns the total size of multiple files in bytes
func calculateTotalSize(filePaths []string) (int64, error) {
	var totalSize int64
	for _, filePath := range filePaths {
		size, err := calculateFileSize(filePath)
		if err != nil {
			return 0, err
		}
		totalSize += size
	}
	return totalSize, nil
}

// updateSizeMetrics calculates and updates size-related metrics in the compilation result
func updateSizeMetrics(result *CompilationResult) {
	if result.InputSize > 0 && result.OutputSize > 0 {
		result.CompressionRatio = float64(result.OutputSize) / float64(result.InputSize)
	}
}

// updateBatchSizeMetrics calculates and updates total size metrics for batch results
func updateBatchSizeMetrics(batchResult *BatchCompilationResult) {
	batchResult.TotalInputSize = 0
	batchResult.TotalOutputSize = 0
	
	for _, result := range batchResult.Results {
		if result.Success {
			batchResult.TotalInputSize += result.InputSize
			batchResult.TotalOutputSize += result.OutputSize
		}
	}
	
	if batchResult.TotalInputSize > 0 {
		batchResult.TotalRatio = float64(batchResult.TotalOutputSize) / float64(batchResult.TotalInputSize)
	}
}

// formatSize formats a size in bytes to a human-readable string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Example usage and helper functions

// DefaultOptions returns sensible default compilation options
func DefaultOptions() CompilationOptions {
	return CompilationOptions{
		ObfuscationLevel:         ObfuscationMaximum,
		StripDebug:               true,
		SuppressDecompileWarning: true,
		Mode:                     ModeIndividual,
		OutputPath:               "luac.out",
	}
}

// MergedOptions returns options configured for merged compilation
func MergedOptions(outputFile string) CompilationOptions {
	opts := DefaultOptions()
	opts.Mode = ModeMerged
	opts.OutputPath = outputFile
	return opts
}
