package main

import "fmt"

// printFileCopyResults logs the results of file copy operations
func printFileCopyResults(result FileCopyBatchResult) {
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
