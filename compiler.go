package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	InputFile   string
	OutputFile  string
	Success     bool
	Error       error
	CompileTime time.Duration
}

// BatchCompilationResult holds the results of multiple file compilations
type BatchCompilationResult struct {
	Results      []CompilationResult
	TotalTime    time.Duration
	SuccessCount int
	ErrorCount   int
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
	compiler := &CLICompiler{binaryPath: binaryPath}

	// If no binary path provided, try to auto-detect
	if binaryPath == "" {
		detectedPath, err := compiler.detectBinaryPath()
		if err != nil {
			return nil, fmt.Errorf("failed to detect luac_mta binary: %w", err)
		}
		compiler.binaryPath = detectedPath
	}

	// Validate that the binary exists and is executable
	if err := compiler.validateBinary(); err != nil {
		return nil, fmt.Errorf("binary validation failed: %w", err)
	}

	return compiler, nil
}

// detectBinaryPath attempts to find the luac_mta binary
func (c *CLICompiler) detectBinaryPath() (string, error) {
	var candidates []string

	// Platform-specific binary names
	if runtime.GOOS == "windows" {
		candidates = []string{
			"luac_mta.exe",
			"./luac_mta.exe",
			"./bin/luac_mta.exe",
			"C:\\bin\\luac_mta.exe",
		}
	} else {
		candidates = []string{
			"luac_mta",
			"./luac_mta",
			"./bin/luac_mta",
			"/usr/local/bin/luac_mta",
			"/usr/bin/luac_mta",
		}
	}

	// Check PATH first
	if path, err := exec.LookPath("luac_mta"); err == nil {
		return path, nil
	}

	// Check candidate locations
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("luac_mta binary not found in PATH or common locations")
}

// validateBinary checks if the binary exists and is executable
func (c *CLICompiler) validateBinary() error {
	if _, err := os.Stat(c.binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", c.binaryPath)
	}

	// Test if binary is executable by running with no arguments
	cmd := exec.Command(c.binaryPath)
	if err := cmd.Run(); err != nil {
		// luac_mta returns non-zero when no files are provided, which is expected
		if _, ok := err.(*exec.ExitError); ok {
			// Check if it's the expected "no input files" error
			return nil
		}
		return fmt.Errorf("binary is not executable: %w", err)
	}

	return nil
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
	}

	batchResult.Results = append(batchResult.Results, result)

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
		}

		batchResult.Results = append(batchResult.Results, result)
	}

	batchResult.TotalTime = time.Since(startTime)

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

// Example usage and helper functions

// DefaultOptions returns sensible default compilation options
func DefaultOptions() CompilationOptions {
	return CompilationOptions{
		ObfuscationLevel:         ObfuscationMaximum,
		StripDebug:               true,
		SuppressDecompileWarning: true,
		Mode:                     ModeIndividual,
		OutputPath:               "compiled",
	}
}

// MergedOptions returns options configured for merged compilation
func MergedOptions(outputFile string) CompilationOptions {
	opts := DefaultOptions()
	opts.Mode = ModeMerged
	opts.OutputPath = outputFile
	return opts
}
