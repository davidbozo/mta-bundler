package main

import (
	"os"
	"regexp"
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
	// Match both single and double quoted src attributes ending with .lua
	luaToLuacRegex := regexp.MustCompile(`(src\s*=\s*"[^"]*?)\.lua(")|(src\s*=\s*'[^']*?)\.lua(')`)
	
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
	testResource := &Resource{}
	
	// Test the copyAndModifyMetaFile function directly
	tempOutput := "test_output_meta.xml"
	defer os.Remove(tempOutput) // Clean up after test
	
	err := testResource.copyAndModifyMetaFile("resource_test.xml", tempOutput)
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