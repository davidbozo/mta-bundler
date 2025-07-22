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
		fmt.Fprintf(os.Stderr, "Usage: %s [options] input_path\n\n", binaryName)
		fmt.Fprintf(os.Stderr, "MTA Lua Compiler accepts only two input types:\n")
		fmt.Fprintf(os.Stderr, "  • Single meta.xml file - Compiles all referenced scripts in the resource\n")
		fmt.Fprintf(os.Stderr, "  • Directory - Recursively finds and compiles ALL meta.xml files found\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s /path/to/resource/meta.xml    # Compile single MTA resource\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s /path/to/resources/           # Compile ALL resources in directory\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s -o compiled/ /path/to/resources/ # Compile all resources to output dir\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s -e3 -s /path/to/resources/    # Max obfuscation + strip debug for all resources\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s -m /path/to/resource/meta.xml # Merge mode: create client.luac and server.luac\n", binaryName)
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
		return fmt.Errorf("no input path provided")
	}
	
	if len(args) > 1 {
		return fmt.Errorf("only one input path is allowed, got %d arguments", len(args))
	}

	inputPath := args[0]

	// Validate input path before proceeding
	if err := validateInputPath(inputPath); err != nil {
		return err
	}

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

// validateInputPath validates that the input path is either a meta.xml file or a directory
func validateInputPath(inputPath string) error {
	// Check if input path exists and get file info
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("cannot access input path '%s': %v", inputPath, err)
	}

	if fileInfo.IsDir() {
		// Directory is valid
		return nil
	} else {
		// If it's a file, check if it's meta.xml
		if strings.ToLower(filepath.Base(inputPath)) == "meta.xml" {
			return nil
		} else {
			return fmt.Errorf("input must be either a meta.xml file or a directory, got: %s", filepath.Base(inputPath))
		}
	}
}

// compileResources handles the compilation of MTA resources using the compiler.go implementation
func compileResources(inputPath string, obfuscationLevel int) error {
	fmt.Printf("Starting compilation for: %s\n", inputPath)

	// Detect luac_mta binary path
	detector := NewBinaryDetector()
	binaryPath, err := detector.DetectAndValidate()
	if err != nil {
		return fmt.Errorf("failed to detect luac_mta binary: %v", err)
	}

	// Initialize the CLI compiler with detected binary path
	compiler, err := NewCLICompiler(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to initialize compiler: %v", err)
	}

	// Get file info (validation already done in validateInputPath)
	fileInfo, _ := os.Stat(inputPath)
	var metaPaths []string

	if fileInfo.IsDir() {
		// If it's a directory, find all meta.xml files
		fmt.Println("Searching for meta.xml files in directory...")
		metaPaths, err := FindMTAResourceMetas(inputPath)
		if err != nil {
			return fmt.Errorf("error finding meta.xml files: %v", err)
		}

		if len(metaPaths) == 0 {
			return fmt.Errorf("no meta.xml files found in directory: %s", inputPath)
		}
	} else {
		// Single meta.xml file (already validated)
		absPath, err := filepath.Abs(inputPath)
		if err != nil {
			return fmt.Errorf("cannot get absolute path: %v", err)
		}
		metaPaths = []string{absPath}
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

