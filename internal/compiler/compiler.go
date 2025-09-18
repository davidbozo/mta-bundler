package compiler

import (
	"fmt"
	"os"
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

// CalculateFileSize returns the size of a file in bytes
func CalculateFileSize(filePath string) (int64, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}
	return fileInfo.Size(), nil
}

// CalculateTotalSize returns the total size of multiple files in bytes
func CalculateTotalSize(filePaths []string) (int64, error) {
	var totalSize int64
	for _, filePath := range filePaths {
		size, err := CalculateFileSize(filePath)
		if err != nil {
			return 0, err
		}
		totalSize += size
	}
	return totalSize, nil
}

// UpdateSizeMetrics calculates and updates size-related metrics in the compilation result
func UpdateSizeMetrics(result *CompilationResult) {
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

// FormatSize formats a size in bytes to a human-readable string
func FormatSize(bytes int64) string {
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
