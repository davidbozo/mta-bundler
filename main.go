package main

import (
	"fmt"
)

// Example usage function
func main() {
	// Example 1: Compile individual files
	compiler, err := NewCLICompiler("")
	if err != nil {
		fmt.Printf("Error creating compiler: %v\n", err)
		return
	}

	files := []string{
		"script1.lua",
		"scripts/script2.lua",
		"modules/utils.lua",
	}

	// Individual compilation
	fmt.Println("Compiling individual files...")
	batchResult, err := compiler.Compile(files, DefaultOptions())
	if err != nil {
		fmt.Printf("Compilation completed with errors: %v\n", err)
	}

	// Print detailed results
	fmt.Printf("Compilation Summary:\n")
	fmt.Printf("  Total Time: %v\n", batchResult.TotalTime)
	fmt.Printf("  Success: %d files\n", batchResult.SuccessCount)
	fmt.Printf("  Errors: %d files\n", batchResult.ErrorCount)
	fmt.Printf("\nDetailed Results:\n")

	for _, result := range batchResult.Results {
		status := "✓"
		if !result.Success {
			status = "✗"
		}
		fmt.Printf("  %s %s -> %s (%v)\n", status, result.InputFile, result.OutputFile, result.CompileTime)
		if result.Error != nil {
			fmt.Printf("    Error: %v\n", result.Error)
		}
	}

	// Example 2: Compile single file
	fmt.Println("\nCompiling single file...")
	singleResult, err := compiler.CompileFile("example.lua", "example.luac", DefaultOptions())
	if err != nil {
		fmt.Printf("Single file compilation failed: %v\n", err)
	} else {
		fmt.Printf("Successfully compiled %s -> %s in %v\n",
			singleResult.InputFile, singleResult.OutputFile, singleResult.CompileTime)
	}

	// Example 3: Merge compilation
	fmt.Println("\nCompiling merged file...")
	mergedBatchResult, err := compiler.Compile(files, MergedOptions("merged.luac"))
	if err != nil {
		fmt.Printf("Merged compilation failed: %v\n", err)
	} else if len(mergedBatchResult.Results) > 0 {
		result := mergedBatchResult.Results[0]
		fmt.Printf("Successfully created merged file: %s in %v\n", result.OutputFile, result.CompileTime)
	}
}
