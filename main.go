package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	outputFile     string
	stripDebug     bool
	obfuscateLevel int
	suppressWarn   bool
	showVersion    bool
)

var rootCmd = &cobra.Command{
	Use:   "mta-bundler [input_path]",
	Short: "MTA Lua Compiler - Compile and obfuscate Lua scripts for Multi Theft Auto",
	Long: `MTA Lua Compiler is a tool for compiling and obfuscating Lua scripts 
for Multi Theft Auto servers. It can process individual files, directories, 
or meta.xml files to compile all referenced scripts.

Examples:
  mta-bundler script.lua                    # Compile single file
  mta-bundler -o compiled.lua script.lua    # Compile with custom output
  mta-bundler -e3 script.lua                # Compile with obfuscation level 3
  mta-bundler -s -e2 script.lua             # Strip debug info and obfuscate level 2
  mta-bundler /path/to/resource/            # Process directory (looks for meta.xml)
  mta-bundler /path/to/meta.xml             # Process meta.xml file directly`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCompiler,
}

func init() {
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output to file 'name' (default is 'luac.out' or original filename with .luac extension)")
	rootCmd.Flags().BoolVarP(&stripDebug, "strip", "s", false, "strip debug information")
	rootCmd.Flags().IntVarP(&obfuscateLevel, "obfuscate", "e", 0, "obfuscation level (0-3)")
	rootCmd.Flags().BoolVarP(&suppressWarn, "suppress", "d", false, "suppress decompile warning")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version information")
	rootCmd.MarkFlagRequired("output")

	// Add support for -e2 and -e3 flags
	rootCmd.Flags().BoolP("obfuscate2", "2", false, "obfuscation level 2 (equivalent to -e2)")
	rootCmd.Flags().BoolP("obfuscate3", "3", false, "obfuscation level 3 (equivalent to -e3)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCompiler(cmd *cobra.Command, args []string) error {
	if showVersion {
		fmt.Println("mta-bundler version 1.0.0")
		fmt.Println("MTA Lua Compiler for Multi Theft Auto")
		return nil
	}

	// Handle obfuscation level flags
	if obfuscate2, _ := cmd.Flags().GetBool("obfuscate2"); obfuscate2 {
		obfuscateLevel = 2
	}
	if obfuscate3, _ := cmd.Flags().GetBool("obfuscate3"); obfuscate3 {
		obfuscateLevel = 3
	}

	// Validate obfuscation level
	if obfuscateLevel < 0 || obfuscateLevel > 3 {
		return fmt.Errorf("invalid obfuscation level: %d (must be 0-3)", obfuscateLevel)
	}

	if len(args) == 0 {
		return fmt.Errorf("no input files given")
	}

	inputPath := args[0]

	// Print parsed arguments for demonstration
	fmt.Printf("Input path: %s\n", inputPath)
	fmt.Printf("Output file: %s\n", outputFile)
	fmt.Printf("Strip debug: %t\n", stripDebug)
	fmt.Printf("Obfuscate level: %d\n", obfuscateLevel)
	fmt.Printf("Suppress warnings: %t\n", suppressWarn)

	// TODO: Implement actual compilation logic here
	fmt.Println("Compilation logic would go here...")

	return nil
}
