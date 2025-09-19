package bundler

import (
	"fmt"
	"time"

	"github.com/davidbozo/mta-bundler/internal/compiler"
	"github.com/davidbozo/mta-bundler/internal/resource"
)

func logResourceStart(name, baseDir string) {
	fmt.Printf("Compiling resource: %s\n", name)
	fmt.Printf("Base directory: %s\n", baseDir)
}

func logLuaFilesFound(count int, resourceName string) {
	if count == 0 {
		fmt.Printf("  Warning: No Lua script files found in resource %s\n", resourceName)
	} else {
		fmt.Printf("  Found %d Lua script(s) to compile\n", count)
	}
}

func logMergedFilesFound(clientCount, serverCount, sharedCount int) {
	fmt.Printf("  Found %d client script(s), %d server script(s), %d shared script(s)\n",
		clientCount, serverCount, sharedCount)
}

func logFileProcessing(filename string) {
	fmt.Printf("  Processing: %s\n", filename)
}

func logFileSuccess(relativePath, relativeOutputPath string, result compiler.CompilationResult) {
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

	fmt.Printf("    ✓ %s -> %s (%v)%s\n", relativePath, relativeOutputPath, result.CompileTime, sizeInfo)
}

func logFileError(filename string, err error) {
	fmt.Printf("    ✗ %s: %v\n", filename, err)
}

func logPathCalculationError(filename string, err error) {
	fmt.Printf("    ✗ Failed to calculate output path: %v\n", err)
}

func logDirectoryCreationError(err error) {
	fmt.Printf("    ✗ Failed to create output directory: %v\n", err)
}

func logIndividualCompilationSummary(successCount, errorCount int, totalTime time.Duration, totalInputSize, totalOutputSize int64) {
	fmt.Printf("  Compilation completed: %d successful, %d errors\n", successCount, errorCount)
	if totalInputSize > 0 && totalOutputSize > 0 && successCount > 0 {
		reduction := (1.0 - float64(totalOutputSize)/float64(totalInputSize)) * 100
		fmt.Printf("  Resource size summary: %s → %s (%.0f%% reduction)\n",
			compiler.FormatSize(totalInputSize), compiler.FormatSize(totalOutputSize), reduction)
	}
	fmt.Printf("  Total time: %v\n", totalTime)
}

func logMergedCompilationStart(fileType string) {
	fmt.Printf("  Compiling %s files to %s.luac...\n", fileType, fileType)
}

func logMergedCompilationSuccess(fileType string, result compiler.CompilationResult) {
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
	fmt.Printf("    ✓ %s compilation successful: %s.luac (%v)%s\n",
		capitalize(fileType), fileType, result.CompileTime, sizeInfo)
}

func logMergedCompilationError(fileType string, err error) {
	fmt.Printf("    ✗ %s compilation failed: %v\n", capitalize(fileType), err)
}

func logMergedDirectoryCreationError(fileType string, err error) {
	fmt.Printf("    ✗ Failed to create %s output directory: %v\n", fileType, err)
}

func logMergedCompilationSummary(successCount, errorCount int, totalTime time.Duration) {
	fmt.Printf("  Merge compilation completed: %d successful, %d errors\n", successCount, errorCount)
	fmt.Printf("  Total time: %v\n", totalTime)
}

func logFileCopyResults(result resource.FileCopyBatchResult) {
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

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}