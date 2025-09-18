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

// Compile compiles multiple Lua files into a single merged output file
func (c *CLICompiler) Compile(filePaths []string, outputPath string, options CompilationOptions) (*CompilationResult, error) {
	startTime := time.Now()

	result := &CompilationResult{
		InputFile:  strings.Join(filePaths, ", "),
		OutputFile: outputPath,
	}

	// Validate input files
	if err := c.ValidateFiles(filePaths); err != nil {
		result.Error = err
		result.CompileTime = time.Since(startTime)
		return result, err
	}

	// Calculate total input size
	if inputSize, err := CalculateTotalSize(filePaths); err == nil {
		result.InputSize = inputSize
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		result.Error = fmt.Errorf("failed to create output directory: %w", err)
		result.CompileTime = time.Since(startTime)
		return result, result.Error
	}

	// Build command arguments
	args := c.buildArgs(options, outputPath)
	args = append(args, filePaths...)

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
