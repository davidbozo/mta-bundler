package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Implement actual compilation logic
	return compileResources(inputPath)

}

// compileResources handles the compilation of MTA resources using the compiler.go implementation
func compileResources(inputPath string) error {
	fmt.Printf("Starting compilation for: %s\n", inputPath)

	// Initialize the CLI compiler
	compiler, err := NewCLICompiler("")
	if err != nil {
		return fmt.Errorf("failed to initialize compiler: %v", err)
	}

	// Check if input is a file or directory
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("cannot access input path: %v", err)
	}

	var metaPaths []string

	if fileInfo.IsDir() {
		// If it's a directory, find all meta.xml files
		fmt.Println("Searching for meta.xml files in directory...")
		metaPaths, err = FindMTAResourceMetas(inputPath)
		if err != nil {
			return fmt.Errorf("error finding meta.xml files: %v", err)
		}

		if len(metaPaths) == 0 {
			return fmt.Errorf("no meta.xml files found in directory: %s", inputPath)
		}
	} else {
		// If it's a file, check if it's meta.xml or a single Lua file
		if strings.ToLower(filepath.Base(inputPath)) == "meta.xml" {
			// Single meta.xml file
			absPath, err := filepath.Abs(inputPath)
			if err != nil {
				return fmt.Errorf("cannot get absolute path: %v", err)
			}
			metaPaths = []string{absPath}
		} else if strings.ToLower(filepath.Ext(inputPath)) == ".lua" {
			// Single Lua file - compile directly
			return compileSingleLuaFile(compiler, inputPath)
		} else {
			return fmt.Errorf("unsupported file type: %s (expected .lua or meta.xml)", inputPath)
		}
	}

	fmt.Printf("Found %d meta.xml file(s) to process\n", len(metaPaths))

	// Process each meta.xml file
	for i, metaPath := range metaPaths {
		fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(metaPaths), metaPath)
		
		resource, err := NewResource(metaPath)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", metaPath, err)
			continue
		}

		err = compileResource(compiler, resource)
		if err != nil {
			fmt.Printf("Error compiling resource %s: %v\n", resource.Name, err)
			continue
		}

		fmt.Printf("Successfully compiled resource: %s\n", resource.Name)
	}

	return nil
}

// compileResource compiles all Lua scripts in a single MTA resource using the compiler.go implementation
func compileResource(compiler *CLICompiler, resource *Resource) error {
	fmt.Printf("Compiling resource: %s\n", resource.Name)
	fmt.Printf("Base directory: %s\n", resource.BaseDir)

	// Collect all Lua script files
	var luaFiles []string
	for _, fileRef := range resource.Files {
		if fileRef.ReferenceType == "Script" && strings.ToLower(filepath.Ext(fileRef.FullPath)) == ".lua" {
			luaFiles = append(luaFiles, fileRef.FullPath)
		}
	}

	if len(luaFiles) == 0 {
		fmt.Printf("  Warning: No Lua script files found in resource %s\n", resource.Name)
		return nil
	}

	fmt.Printf("  Found %d Lua script(s) to compile\n", len(luaFiles))

	// Create compilation options from CLI flags
	options := CompilationOptions{
		ObfuscationLevel:         ObfuscationLevel(obfuscateLevel),
		StripDebug:               stripDebug,
		SuppressDecompileWarning: suppressWarn,
		Mode:                     ModeIndividual,
		OutputPath:               resource.BaseDir,
	}

	// If outputFile is specified, use merged mode for single output
	if outputFile != "" {
		options.Mode = ModeMerged
		if filepath.IsAbs(outputFile) {
			options.OutputPath = outputFile
		} else {
			options.OutputPath = filepath.Join(resource.BaseDir, outputFile)
		}
	}

	// Compile using the CLI compiler
	result, err := compiler.Compile(luaFiles, options)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	// Display results
	fmt.Printf("  Compilation completed: %d successful, %d errors\n", result.SuccessCount, result.ErrorCount)
	fmt.Printf("  Total time: %v\n", result.TotalTime)

	// Display detailed results
	for _, fileResult := range result.Results {
		if fileResult.Success {
			fmt.Printf("    ✓ %s -> %s (%v)\n", filepath.Base(fileResult.InputFile), filepath.Base(fileResult.OutputFile), fileResult.CompileTime)
		} else {
			fmt.Printf("    ✗ %s: %v\n", filepath.Base(fileResult.InputFile), fileResult.Error)
		}
	}

	return nil
}

// compileSingleLuaFile compiles a single Lua file using the compiler.go implementation
func compileSingleLuaFile(compiler *CLICompiler, luaPath string) error {
	fmt.Printf("Compiling single Lua file: %s\n", luaPath)
	
	absPath, err := filepath.Abs(luaPath)
	if err != nil {
		return fmt.Errorf("cannot get absolute path: %v", err)
	}

	// Generate output filename
	var outputPath string
	if outputFile != "" {
		// Use specified output file
		if filepath.IsAbs(outputFile) {
			outputPath = outputFile
		} else {
			outputPath = filepath.Join(filepath.Dir(absPath), outputFile)
		}
	} else {
		// Generate default output filename
		baseName := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
		outputPath = filepath.Join(filepath.Dir(absPath), baseName+".luac")
	}

	// Create compilation options from CLI flags
	options := CompilationOptions{
		ObfuscationLevel:         ObfuscationLevel(obfuscateLevel),
		StripDebug:               stripDebug,
		SuppressDecompileWarning: suppressWarn,
	}

	// Compile the single file
	result, err := compiler.CompileFile(absPath, outputPath, options)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	// Display result
	if result.Success {
		fmt.Printf("✓ Compilation successful: %s -> %s (%v)\n", 
			filepath.Base(result.InputFile), 
			filepath.Base(result.OutputFile), 
			result.CompileTime)
	} else {
		fmt.Printf("✗ Compilation failed: %v\n", result.Error)
	}

	return nil
}
