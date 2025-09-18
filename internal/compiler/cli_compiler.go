package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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
	if inputSize, err := CalculateFileSize(filePath); err == nil {
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
	if outputSize, err := CalculateFileSize(outputPath); err == nil {
		result.OutputSize = outputSize
		UpdateSizeMetrics(result)
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
	if inputSize, err := CalculateTotalSize(filePaths); err == nil {
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
		if outputSize, err := CalculateFileSize(outputPath); err == nil {
			result.OutputSize = outputSize
			UpdateSizeMetrics(&result)
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
		if inputSize, err := CalculateFileSize(inputPath); err == nil {
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
			if outputSize, err := CalculateFileSize(outputPath); err == nil {
				result.OutputSize = outputSize
				UpdateSizeMetrics(&result)
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