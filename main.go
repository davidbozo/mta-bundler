package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mta-bundler",
	Short: "A CLI tool for compiling and bundling Lua files",
	Long:  `MTA Bundler is a command-line tool for compiling individual Lua files or merging multiple Lua files into a single compiled output.`,
}

var compileCmd = &cobra.Command{
	Use:   "compile [files...]",
	Short: "Compile individual Lua files",
	Long:  `Compile one or more Lua files individually, maintaining separate output files for each input.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		compiler, err := NewCLICompiler("")
		if err != nil {
			fmt.Printf("Error creating compiler: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Compiling individual files...")
		batchResult, err := compiler.Compile(args, DefaultOptions())
		if err != nil {
			fmt.Printf("Compilation completed with errors: %v\n", err)
		}

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
	},
}

var mergeCmd = &cobra.Command{
	Use:   "merge [files...] -o output",
	Short: "Merge and compile multiple Lua files into a single output",
	Long:  `Merge multiple Lua files and compile them into a single output file.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			fmt.Println("Error: output file is required for merge command")
			os.Exit(1)
		}

		compiler, err := NewCLICompiler("")
		if err != nil {
			fmt.Printf("Error creating compiler: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Compiling merged file...")
		mergedBatchResult, err := compiler.Compile(args, MergedOptions(output))
		if err != nil {
			fmt.Printf("Merged compilation failed: %v\n", err)
			os.Exit(1)
		}

		if len(mergedBatchResult.Results) > 0 {
			result := mergedBatchResult.Results[0]
			fmt.Printf("Successfully created merged file: %s in %v\n", result.OutputFile, result.CompileTime)
		}
	},
}

var singleCmd = &cobra.Command{
	Use:   "single <input> <output>",
	Short: "Compile a single Lua file",
	Long:  `Compile a single Lua file to a specified output file.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		output := args[1]

		compiler, err := NewCLICompiler("")
		if err != nil {
			fmt.Printf("Error creating compiler: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Compiling %s to %s...\n", input, output)
		result, err := compiler.CompileFile(input, output, DefaultOptions())
		if err != nil {
			fmt.Printf("Single file compilation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully compiled %s -> %s in %v\n",
			result.InputFile, result.OutputFile, result.CompileTime)
	},
}

func init() {
	mergeCmd.Flags().StringP("output", "o", "", "Output file for merged compilation (required)")
	mergeCmd.MarkFlagRequired("output")

	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(mergeCmd)
	rootCmd.AddCommand(singleCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
