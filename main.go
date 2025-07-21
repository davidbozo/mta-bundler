package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	outputFile     = flag.String("o", "", "output directory for compiled files (default is same directory as source files)")
	stripDebug     = flag.Bool("s", false, "strip debug information")
	obfuscateLevel = flag.Int("e", 0, "obfuscation level (0-3)")
	suppressWarn   = flag.Bool("d", false, "suppress decompile warning")
	showVersion    = flag.Bool("v", false, "show version information")
	mergeMode      = flag.Bool("m", false, "merge all scripts into client.luac and server.luac")

	// Build-time variables set by GoReleaser
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	flag.Usage = func() {
		binaryName := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "MTA Lua Compiler - Compile and obfuscate Lua scripts for Multi Theft Auto\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [input_path]\n\n", binaryName)
		fmt.Fprintf(os.Stderr, "MTA Lua Compiler is a tool for compiling and obfuscating Lua scripts\n")
		fmt.Fprintf(os.Stderr, "for Multi Theft Auto servers. It can process individual files, directories,\n")
		fmt.Fprintf(os.Stderr, "or meta.xml files to compile all referenced scripts.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s script.lua                    # Compile single file to same directory\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s -o output/ script.lua         # Compile to specific output directory\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s -e3 script.lua                # Compile with obfuscation level 3\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s -s -e2 script.lua             # Strip debug info and obfuscate level 2\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s /path/to/resource/            # Process directory (looks for meta.xml)\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s -o compiled/ /path/to/meta.xml # Process meta.xml with custom output directory\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s -m /path/to/resource/         # Merge all scripts into client.luac and server.luac\n", binaryName)
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	
	if err := runCompiler(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCompiler() error {
	if *showVersion {
		fmt.Printf("mta-bundler version %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Build Date: %s\n", date)
		fmt.Println("MTA Lua Compiler for Multi Theft Auto")
		return nil
	}

	// Handle obfuscation level flags
	obfuscationLevel := *obfuscateLevel

	// Validate obfuscation level
	if obfuscationLevel < 0 || obfuscationLevel > 3 {
		return fmt.Errorf("invalid obfuscation level: %d (must be 0-3)", obfuscationLevel)
	}

	args := flag.Args()
	if len(args) == 0 {
		return fmt.Errorf("no input files given")
	}

	inputPath := args[0]

	// Print parsed arguments for demonstration
	fmt.Printf("Input path: %s\n", inputPath)
	fmt.Printf("Output file: %s\n", *outputFile)
	fmt.Printf("Strip debug: %t\n", *stripDebug)
	fmt.Printf("Obfuscate level: %d\n", obfuscationLevel)
	fmt.Printf("Suppress warnings: %t\n", *suppressWarn)
	fmt.Printf("Merge mode: %t\n", *mergeMode)

	// Implement actual compilation logic
	return compileResources(inputPath, obfuscationLevel)
}

// compileResources handles the compilation of MTA resources using the compiler.go implementation
func compileResources(inputPath string, obfuscationLevel int) error {
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
			return compileSingleLuaFile(compiler, inputPath, inputPath, obfuscationLevel)
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

		// Create compilation options
		options := CompilationOptions{
			ObfuscationLevel:         ObfuscationLevel(obfuscationLevel),
			StripDebug:               *stripDebug,
			SuppressDecompileWarning: *suppressWarn,
		}

		err = resource.Compile(compiler, inputPath, *outputFile, options, *mergeMode)
		if err != nil {
			fmt.Printf("Error compiling resource %s: %v\n", resource.Name, err)
			continue
		}

		fmt.Printf("Successfully compiled resource: %s\n", resource.Name)
	}

	return nil
}

// compileSingleLuaFile compiles a single Lua file using the compiler.go implementation
func compileSingleLuaFile(compiler *CLICompiler, luaPath string, basePath string, obfuscationLevel int) error {
	fmt.Printf("Compiling single Lua file: %s\n", luaPath)

	absPath, err := filepath.Abs(luaPath)
	if err != nil {
		return fmt.Errorf("cannot get absolute path: %v", err)
	}

	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("cannot get absolute base path: %v", err)
	}

	// Generate output filename preserving original name
	baseName := strings.TrimSuffix(filepath.Base(absPath), ".lua")
	outputFileName := baseName + ".luac"

	var outputPath string
	if *outputFile != "" {
		// Use specified output directory and preserve relative structure from basePath
		var baseOutputDir string
		if filepath.IsAbs(*outputFile) {
			baseOutputDir = *outputFile
		} else {
			// If outputFile is relative, resolve it from current working directory
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %v", err)
			}
			baseOutputDir = filepath.Join(cwd, *outputFile)
		}

		// Calculate relative path from basePath to maintain directory structure
		relativeFromBase, err := filepath.Rel(absBasePath, filepath.Dir(absPath))
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %v", err)
		}

		// Build output path preserving structure
		if relativeFromBase == "." || relativeFromBase == "" {
			outputPath = filepath.Join(baseOutputDir, outputFileName)
		} else {
			outputPath = filepath.Join(baseOutputDir, relativeFromBase, outputFileName)
		}

		// Create output directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}
	} else {
		// Output to same directory as source file
		outputPath = filepath.Join(filepath.Dir(absPath), outputFileName)
	}

	// Create compilation options from CLI flags
	options := CompilationOptions{
		ObfuscationLevel:         ObfuscationLevel(obfuscationLevel),
		StripDebug:               *stripDebug,
		SuppressDecompileWarning: *suppressWarn,
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
