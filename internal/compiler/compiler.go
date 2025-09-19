package compiler

import (
	"fmt"
	"os"
	"time"
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

// LuaCompiler interface defines the contract for Lua compilation
type LuaCompiler interface {
	// Compile compiles multiple Lua files into a single merged output file
	Compile(filePaths []string, outputPath string, options CompilationOptions) (*CompilationResult, error)
	// CompileFile compiles a single Lua file to its individual output
	CompileFile(filePath string, outputPath string, options CompilationOptions) (*CompilationResult, error)
	// ValidateFiles checks if all provided files exist and are valid
	ValidateFiles(filePaths []string) error
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

// DefaultOptions returns sensible default compilation options
func DefaultOptions() CompilationOptions {
	return CompilationOptions{
		ObfuscationLevel:         ObfuscationMaximum,
		StripDebug:               true,
		SuppressDecompileWarning: true,
	}
}
