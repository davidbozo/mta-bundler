package resource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetaXMLRegexReplacement(t *testing.T) {
	// Read the test meta.xml file
	content, err := os.ReadFile("resource_test.xml")
	if err != nil {
		t.Fatalf("Error reading test file: %v", err)
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

	// Test cases to verify the replacement worked correctly
	testCases := []struct {
		original string
		expected string
	}{
		{`src="server.lua"`, `src="server.luac"`},
		{`src="client.lua"`, `src="client.luac"`},
		{`src="shared.lua"`, `src="shared.luac"`},
		{`src="utils/helper.lua"`, `src="utils/helper.luac"`},
		{`src="modules/core.lua"`, `src="modules/core.luac"`},
	}

	for _, tc := range testCases {
		if !strings.Contains(modifiedContent, tc.expected) {
			t.Errorf("Expected to find %q in modified content", tc.expected)
		}
		if strings.Contains(modifiedContent, tc.original) {
			t.Errorf("Original %q should have been replaced", tc.original)
		}
	}

	// Verify that non-lua files are not affected
	nonLuaFiles := []string{
		`src="logo.png"`,
		`src="model.dff"`,
		`src="texture.txd"`,
		`src="mymap.map"`,
		`src="settings.xml"`,
	}

	for _, nonLua := range nonLuaFiles {
		if !strings.Contains(modifiedContent, nonLua) {
			t.Errorf("Non-lua file reference %q should remain unchanged", nonLua)
		}
	}
}

func TestCopyAndModifyMetaFileFunction(t *testing.T) {
	// Create a temporary test resource
	testResource := Resource{}

	// Test the copyAndModifyMetaFile function directly
	tempOutput := "test_output_meta.xml"
	defer os.Remove(tempOutput) // Clean up after test

	err := testResource.CopyAndModifyMetaFile("resource_test.xml", tempOutput)
	if err != nil {
		t.Fatalf("copyAndModifyMetaFile failed: %v", err)
	}

	// Read the output file
	content, err := os.ReadFile(tempOutput)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	modifiedContent := string(content)

	// Verify that .lua files were converted to .luac
	if strings.Contains(modifiedContent, `src="server.lua"`) {
		t.Error("Found unconverted .lua reference")
	}
	if !strings.Contains(modifiedContent, `src="server.luac"`) {
		t.Error("Expected .luac reference not found")
	}
}

// Test for copyAndModifyMergedMetaFile method
func TestCopyAndModifyMergedMetaFile(t *testing.T) {
	// Create a temporary test resource
	testResource := Resource{}

	tests := []struct {
		name           string
		inputXML       string
		hasClientFiles bool
		hasServerFiles bool
		expectedTags   []string
		preservedTags  []string
	}{
		{
			name: "Both client and server files",
			inputXML: `<?xml version="1.0" encoding="UTF-8"?>
<meta>
    <info author="Test" version="1.0" name="TestResource" type="gamemode" />
    <script src="client.lua" type="client" />
    <script src="server.lua" type="server" />
    <script src="shared.lua" type="shared" />
    <file src="logo.png" />
    <include resource="scoreboard" />
    <!-- Custom comment -->
    <export function="test" type="server" />
</meta>`,
			hasClientFiles: true,
			hasServerFiles: true,
			expectedTags: []string{
				`<script src="client.luac" type="client" cache="true" />`,
				`<script src="server.luac" type="server" cache="true" />`,
			},
			preservedTags: []string{
				`<info author="Test" version="1.0" name="TestResource" type="gamemode" />`,
				`<file src="logo.png" />`,
				`<include resource="scoreboard" />`,
				`<export function="test" type="server" />`,
				`<!-- Custom comment -->`,
			},
		},
		{
			name: "Client files only",
			inputXML: `<?xml version="1.0" encoding="UTF-8"?>
<meta>
    <info author="Test" version="1.0" name="TestResource" type="gamemode" />
    <script src="client.lua" type="client" />
    <file src="logo.png" />
</meta>`,
			hasClientFiles: true,
			hasServerFiles: false,
			expectedTags: []string{
				`<script src="client.luac" type="client" cache="true" />`,
			},
			preservedTags: []string{
				`<info author="Test" version="1.0" name="TestResource" type="gamemode" />`,
				`<file src="logo.png" />`,
			},
		},
		{
			name: "Server files only",
			inputXML: `<?xml version="1.0" encoding="UTF-8"?>
<meta>
    <info author="Test" version="1.0" name="TestResource" type="gamemode" />
    <script src="server.lua" type="server" />
    <file src="logo.png" />
</meta>`,
			hasClientFiles: false,
			hasServerFiles: true,
			expectedTags: []string{
				`<script src="server.luac" type="server" cache="true" />`,
			},
			preservedTags: []string{
				`<info author="Test" version="1.0" name="TestResource" type="gamemode" />`,
				`<file src="logo.png" />`,
			},
		},
		{
			name: "No files",
			inputXML: `<?xml version="1.0" encoding="UTF-8"?>
<meta>
    <info author="Test" version="1.0" name="TestResource" type="gamemode" />
    <script src="client.lua" type="client" />
    <script src="server.lua" type="server" />
    <file src="logo.png" />
</meta>`,
			hasClientFiles: false,
			hasServerFiles: false,
			expectedTags:   []string{},
			preservedTags: []string{
				`<info author="Test" version="1.0" name="TestResource" type="gamemode" />`,
				`<file src="logo.png" />`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary input file
			tempInputFile := filepath.Join(os.TempDir(), "test_input_"+tt.name+".xml")
			defer os.Remove(tempInputFile)

			err := os.WriteFile(tempInputFile, []byte(tt.inputXML), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp input file: %v", err)
			}

			// Create temporary output file
			tempOutputFile := filepath.Join(os.TempDir(), "test_output_"+tt.name+".xml")
			defer os.Remove(tempOutputFile)

			// Test the copyAndModifyMergedMetaFile function
			err = testResource.CopyAndModifyMergedMetaFile(tempInputFile, tempOutputFile, tt.hasClientFiles, tt.hasServerFiles)
			if err != nil {
				t.Fatalf("copyAndModifyMergedMetaFile failed: %v", err)
			}

			// Read the output file
			content, err := os.ReadFile(tempOutputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			modifiedContent := string(content)

			// Verify expected tags are present
			for _, expectedTag := range tt.expectedTags {
				if !strings.Contains(modifiedContent, expectedTag) {
					t.Errorf("Expected tag %q not found in output", expectedTag)
				}
			}

			// Verify preserved tags are still present
			for _, preservedTag := range tt.preservedTags {
				if !strings.Contains(modifiedContent, preservedTag) {
					t.Errorf("Preserved tag %q not found in output", preservedTag)
				}
			}

			// Verify original script tags are removed
			originalScriptTags := []string{
				`src="client.lua"`,
				`src="server.lua"`,
				`src="shared.lua"`,
			}
			for _, originalTag := range originalScriptTags {
				if strings.Contains(modifiedContent, originalTag) {
					t.Errorf("Original script tag %q should have been removed", originalTag)
				}
			}
		})
	}
}
