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

// compileResources handles the compilation of MTA resources
func compileResources(inputPath string) error {
	fmt.Printf("Starting compilation for: %s\n", inputPath)

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
			return compileSingleLuaFile(inputPath)
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

		err = compileResource(resource)
		if err != nil {
			fmt.Printf("Error compiling resource %s: %v\n", resource.Name, err)
			continue
		}

		fmt.Printf("Successfully compiled resource: %s\n", resource.Name)
	}

	return nil
}

// compileResource compiles all Lua scripts in a single MTA resource
func compileResource(resource *Resource) error {
	fmt.Printf("Compiling resource: %s\n", resource.Name)
	fmt.Printf("Base directory: %s\n", resource.BaseDir)

	luaFiles := 0
	
	// Process all file references and compile Lua scripts
	for _, fileRef := range resource.Files {
		if fileRef.ReferenceType == "Script" && strings.ToLower(filepath.Ext(fileRef.FullPath)) == ".lua" {
			luaFiles++
			fmt.Printf("  Processing Lua script: %s\n", fileRef.RelativePath)
			
			err := compileLuaFile(fileRef.FullPath, fileRef.RelativePath, resource.BaseDir)
			if err != nil {
				return fmt.Errorf("failed to compile %s: %v", fileRef.RelativePath, err)
			}
		}
	}

	if luaFiles == 0 {
		fmt.Printf("  Warning: No Lua script files found in resource %s\n", resource.Name)
	} else {
		fmt.Printf("  Compiled %d Lua script(s)\n", luaFiles)
	}

	return nil
}

// compileSingleLuaFile compiles a single Lua file
func compileSingleLuaFile(luaPath string) error {
	fmt.Printf("Compiling single Lua file: %s\n", luaPath)
	
	absPath, err := filepath.Abs(luaPath)
	if err != nil {
		return fmt.Errorf("cannot get absolute path: %v", err)
	}

	baseDir := filepath.Dir(absPath)
	relativePath := filepath.Base(absPath)
	
	return compileLuaFile(absPath, relativePath, baseDir)
}

// compileLuaFile handles the actual compilation of a Lua file
func compileLuaFile(fullPath, relativePath, baseDir string) error {
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("Lua file does not exist: %s", fullPath)
	}

	// Generate output filename
	var outputPath string
	if outputFile != "" {
		// Use specified output file
		if filepath.IsAbs(outputFile) {
			outputPath = outputFile
		} else {
			outputPath = filepath.Join(baseDir, outputFile)
		}
	} else {
		// Generate default output filename
		baseName := strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
		outputPath = filepath.Join(baseDir, baseName+".luac")
	}

	fmt.Printf("    Input:  %s\n", relativePath)
	fmt.Printf("    Output: %s\n", filepath.Base(outputPath))

	// Display compilation settings
	if stripDebug {
		fmt.Printf("    Strip debug: enabled\n")
	}
	if obfuscateLevel > 0 {
		fmt.Printf("    Obfuscation level: %d\n", obfuscateLevel)
	}
	if suppressWarn {
		fmt.Printf("    Suppress warnings: enabled\n")
	}

	// TODO: Implement actual Lua compilation here
	// This would involve:
	// 1. Reading the Lua source file
	// 2. Compiling it to bytecode
	// 3. Applying obfuscation if requested
	// 4. Stripping debug info if requested
	// 5. Writing the compiled output

	fmt.Printf("    Status: Compilation simulation complete\n")

	return nil
}
