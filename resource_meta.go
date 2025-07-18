package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// luaToLuacRegex is the compiled regex pattern for replacing .lua with .luac in src attributes
var luaToLuacRegex = regexp.MustCompile(`(src\s*=\s*"[^"]*?)\.lua(")|(src\s*=\s*'[^']*?)\.lua(')`)

// copyMetaFile copies the meta.xml file to the output directory and updates lua file references to luac
func (r *Resource) copyMetaFile(baseOutputDir, absInputPath, outputFile string) error {
	// Calculate the output path for meta.xml
	var outputPath string

	if outputFile != "" {
		// When output directory is specified, calculate relative path from inputPath
		relativeFromInput, err := filepath.Rel(absInputPath, r.BaseDir)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %v", err)
		}

		if relativeFromInput == "" || relativeFromInput == "." {
			outputPath = filepath.Join(baseOutputDir, "meta.xml")
		} else {
			outputPath = filepath.Join(baseOutputDir, relativeFromInput, "meta.xml")
		}
	} else {
		// Output to same directory structure as source
		outputPath = filepath.Join(baseOutputDir, "meta.xml")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory for meta.xml: %v", err)
	}

	// Copy and modify the meta.xml file
	if err := r.copyAndModifyMetaFile(r.MetaXMLPath, outputPath); err != nil {
		return fmt.Errorf("failed to copy and modify meta.xml: %v", err)
	}

	fmt.Printf("  ✓ Copied and updated meta.xml\n")
	return nil
}

// copyAndModifyMetaFile copies the meta.xml file and updates .lua file extensions to .luac using regex
func (r *Resource) copyAndModifyMetaFile(src, dst string) error {
	// Read the source meta.xml file
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source meta.xml: %v", err)
	}

	// Convert to string for regex processing
	metaContent := string(content)

	// Use regex to replace .lua with .luac in src attributes
	// Replace .lua with .luac while preserving the quotes
	modifiedContent := luaToLuacRegex.ReplaceAllStringFunc(metaContent, func(match string) string {
		if strings.Contains(match, `"`) {
			return strings.Replace(match, ".lua\"", ".luac\"", 1)
		} else {
			return strings.Replace(match, ".lua'", ".luac'", 1)
		}
	})

	// Write the modified content to the destination file
	err = os.WriteFile(dst, []byte(modifiedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write modified meta.xml: %v", err)
	}

	return nil
}

// copyMergedMetaFile copies the meta.xml file to the output directory and updates it for merged compilation
func (r *Resource) copyMergedMetaFile(baseOutputDir, absInputPath, outputFile string, hasClientFiles, hasServerFiles bool) error {
	// Calculate the output path for meta.xml
	var outputPath string

	if outputFile != "" {
		// When output directory is specified, calculate relative path from inputPath
		relativeFromInput, err := filepath.Rel(absInputPath, r.BaseDir)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %v", err)
		}

		if relativeFromInput == "" || relativeFromInput == "." {
			outputPath = filepath.Join(baseOutputDir, "meta.xml")
		} else {
			outputPath = filepath.Join(baseOutputDir, relativeFromInput, "meta.xml")
		}
	} else {
		// Output to same directory structure as source
		outputPath = filepath.Join(baseOutputDir, "meta.xml")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory for meta.xml: %v", err)
	}

	// Copy and modify the meta.xml file for merged compilation
	if err := r.copyAndModifyMergedMetaFile(r.MetaXMLPath, outputPath, hasClientFiles, hasServerFiles); err != nil {
		return fmt.Errorf("failed to copy and modify meta.xml: %v", err)
	}

	fmt.Printf("  ✓ Copied and updated meta.xml for merged compilation\n")
	return nil
}

// copyAndModifyMergedMetaFile copies the meta.xml file and updates it for merged compilation
func (r *Resource) copyAndModifyMergedMetaFile(src, dst string, hasClientFiles, hasServerFiles bool) error {
	// Read the source meta.xml file
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source meta.xml: %v", err)
	}

	// Convert to string for regex processing
	metaContent := string(content)

	// Remove all existing <script> tags using regex
	// This regex matches <script...> tags (both self-closing and with closing tags)
	scriptRegex := regexp.MustCompile(`(?s)<script[^>]*(?:/>|>.*?</script>)`)
	modifiedContent := scriptRegex.ReplaceAllString(metaContent, "")

	// Build replacement script tags
	var scriptTags []string

	if hasClientFiles {
		scriptTags = append(scriptTags, `    <script src="client.luac" type="client" cache="true" />`)
	}

	if hasServerFiles {
		scriptTags = append(scriptTags, `    <script src="server.luac" type="server" cache="true" />`)
	}

	// Find the position to insert the new script tags
	// Look for the closing </meta> tag and insert before it
	metaEndRegex := regexp.MustCompile(`(\s*</meta>)`)
	if metaEndRegex.MatchString(modifiedContent) {
		// Insert the new script tags before the closing </meta> tag
		replacement := ""
		if len(scriptTags) > 0 {
			replacement = strings.Join(scriptTags, "\n") + "\n$1"
		} else {
			replacement = "$1"
		}
		modifiedContent = metaEndRegex.ReplaceAllString(modifiedContent, replacement)
	} else {
		// Fallback: if no closing </meta> tag found, look for <meta> self-closing tag
		metaSelfClosingRegex := regexp.MustCompile(`(<meta[^>]*)/>\s*$`)
		if metaSelfClosingRegex.MatchString(modifiedContent) {
			// Convert self-closing <meta/> to <meta>...</meta> format
			replacement := "$1>\n"
			if len(scriptTags) > 0 {
				replacement += strings.Join(scriptTags, "\n") + "\n"
			}
			replacement += "</meta>"
			modifiedContent = metaSelfClosingRegex.ReplaceAllString(modifiedContent, replacement)
		} else {
			// Last resort: append before the end of the file
			if len(scriptTags) > 0 {
				modifiedContent = strings.TrimSpace(modifiedContent) + "\n" + strings.Join(scriptTags, "\n") + "\n"
			}
		}
	}

	// Write the modified content to the destination file
	err = os.WriteFile(dst, []byte(modifiedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write modified meta.xml: %v", err)
	}

	return nil
}
