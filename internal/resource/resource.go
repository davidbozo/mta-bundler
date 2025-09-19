package resource

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// GetLuaFiles returns all Lua script files from the resource
func (r *Resource) GetLuaFiles() []FileReference {
	var luaFiles []FileReference
	for _, fileRef := range r.Files {
		if fileRef.ReferenceType == ReferenceTypeScript && strings.ToLower(filepath.Ext(fileRef.FullPath)) == ".lua" {
			luaFiles = append(luaFiles, fileRef)
		}
	}
	return luaFiles
}

// GetLuaFilesByType returns Lua script files grouped by type (client, server, shared)
func (r *Resource) GetLuaFilesByType() (client, server, shared []FileReference) {
	for _, script := range r.Meta.Scripts {
		if strings.ToLower(filepath.Ext(script.Src)) == ".lua" {
			fileRef := FileReference{
				FullPath:      filepath.Join(r.BaseDir, script.Src),
				ReferenceType: ReferenceTypeScript,
				RelativePath:  script.Src,
			}

			switch strings.ToLower(script.Type) {
			case "client":
				client = append(client, fileRef)
			case "server":
				server = append(server, fileRef)
			case "shared":
				shared = append(shared, fileRef)
			default:
				// Default to server if no type specified
				server = append(server, fileRef)
			}
		}
	}
	return client, server, shared
}
