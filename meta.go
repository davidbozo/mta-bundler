package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

// Meta represents the root meta.xml structure with only file-related fields
type Meta struct {
	XMLName xml.Name `xml:"meta"`
	Scripts []Script `xml:"script"`
	Maps    []Map    `xml:"map"`
	Files   []File   `xml:"file"`
	Configs []Config `xml:"config"`
	HTMLs   []HTML   `xml:"html"`
}

// Script represents a script file reference
type Script struct {
	Src      string `xml:"src,attr"`      // The file name of the source code
	Type     string `xml:"type,attr"`     // "client", "server" or "shared"
	Cache    string `xml:"cache,attr"`    // "true" or "false" (default: "true")
	Validate string `xml:"validate,attr"` // "true" or "false" (default: "true")
}

// Map represents a map file reference
type Map struct {
	Src       string `xml:"src,attr"`       // .map file name (can be path too)
	Dimension string `xml:"dimension,attr"` // Dimension in which the map will be loaded (optional)
}

// File represents a client-side file reference
type File struct {
	Src      string `xml:"src,attr"`      // Client-side file name (can be path too)
	Download string `xml:"download,attr"` // "true" or "false" (default: "true")
}

// Config represents a config file reference
type Config struct {
	Src  string `xml:"src,attr"`  // The file name of the config file
	Type string `xml:"type,attr"` // "client" or "server"
}

// HTML represents an HTML file reference
type HTML struct {
	Src     string `xml:"src,attr"`     // The filename for the HTTP file (can be a path)
	Default string `xml:"default,attr"` // "true" or "false" - shown by default when visiting /resourceName/
	Raw     string `xml:"raw,attr"`     // "true" or "false" - treated as binary data
}

// FileReference represents a file reference with its full path and reference type
type FileReference struct {
	FullPath      string // Absolute file path
	ReferenceType string // How the file was referenced (Script, Map, Config, File, HTML)
	RelativePath  string // Original relative path from meta.xml
}

// Resource represents an MTA resource with its meta.xml and all file references
type Resource struct {
	MetaXMLPath string          // Path to the meta.xml file
	BaseDir     string          // Base directory of the resource
	Name        string          // Resource name (derived from directory name)
	Meta        Meta            // Parsed meta.xml structure
	Files       []FileReference // All file references from meta.xml
}

// NewResource creates a new Resource from a meta.xml file path
func NewResource(metaXMLPath string) (*Resource, error) {
	// Read the meta.xml file
	data, err := os.ReadFile(metaXMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read meta.xml: %w", err)
	}

	// Parse the XML
	var meta Meta
	err = xml.Unmarshal(data, &meta)
	if err != nil {
		return nil, fmt.Errorf("failed to parse meta.xml: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(metaXMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create resource
	baseDir := filepath.Dir(absPath)
	resourceName := filepath.Base(baseDir)

	resource := &Resource{
		MetaXMLPath: absPath,
		BaseDir:     baseDir,
		Name:        resourceName,
		Meta:        meta,
	}

	// Get all file references
	resource.Files, err = GetAllFiles(meta, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file references: %w", err)
	}

	return resource, nil
}

// GetAllFiles extracts all file references from Meta structure and returns their full paths
func GetAllFiles(meta Meta, metaXMLPath string) ([]FileReference, error) {
	var files []FileReference

	// Get the directory containing the meta.xml file
	baseDir := filepath.Dir(metaXMLPath)

	// Process Script files
	for _, script := range meta.Scripts {
		fullPath := filepath.Join(baseDir, script.Src)
		files = append(files, FileReference{
			FullPath:      fullPath,
			ReferenceType: "Script",
			RelativePath:  script.Src,
		})
	}

	// Process Map files
	for _, mapFile := range meta.Maps {
		fullPath := filepath.Join(baseDir, mapFile.Src)
		files = append(files, FileReference{
			FullPath:      fullPath,
			ReferenceType: "Map",
			RelativePath:  mapFile.Src,
		})
	}

	// Process Config files
	for _, config := range meta.Configs {
		fullPath := filepath.Join(baseDir, config.Src)
		files = append(files, FileReference{
			FullPath:      fullPath,
			ReferenceType: "Config",
			RelativePath:  config.Src,
		})
	}

	// Process File entries
	for _, file := range meta.Files {
		fullPath := filepath.Join(baseDir, file.Src)
		files = append(files, FileReference{
			FullPath:      fullPath,
			ReferenceType: "File",
			RelativePath:  file.Src,
		})
	}

	// Process HTML files
	for _, html := range meta.HTMLs {
		fullPath := filepath.Join(baseDir, html.Src)
		files = append(files, FileReference{
			FullPath:      fullPath,
			ReferenceType: "HTML",
			RelativePath:  html.Src,
		})
	}

	return files, nil
}
