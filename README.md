# MTA Bundler

> ⚠️ **Disclaimer: This project is under active development with heavy use of AI assistance**  
> This tool is created for learning purposes and may contain bugs or unexpected behavior.  
> Use at your own risk and always test thoroughly before deploying the compiled resources.

A command-line tool for compiling and bundling Multi Theft Auto (MTA) resources. This tool can process a single resource's (meta.xml) files or entire directories containing multiple resources, automatically discovering all meta.xml files recursively. It compiles Lua scripts using [luac_mta](https://wiki.multitheftauto.com/wiki/Lua_compilation_API), and creates optimized resource bundles with configurable obfuscation levels.

## Features

- **Input Support**: Process meta.xml files or directories containing MTA resources
- **Batch Resource Processing**: When given a directory, recursively finds and compiles ALL meta.xml files
- **Obfuscation Levels**: Support for 4 levels of obfuscation (0-3)
- **Resource Processing**: Automatically processes MTA resources based on meta.xml structure
- **Directory Structure Preservation**: Maintains original directory structure in output
- **File Management**: Copies non-script files (maps, configs, assets) to output directory
- **Meta.xml Updates**: Automatically updates script references from `.lua` to `.luac`
- **Cross-Platform**: Works on Windows, Linux, and macOS

## Usage

### Basic Usage

The MTA Bundler supports two input types:

```bash
# Compile a single MTA resource (meta.xml file)
mta-bundler /path/to/resource/meta.xml

# Compile ALL resources in a directory (recursively finds meta.xml files)
mta-bundler /path/to/resources/
```

### Command Line Options

```bash
mta-bundler [OPTIONS] [input_path]

Options:
  -o string    Output directory for compiled files (default: same as source)
  -s           Strip debug information
  -e int       Obfuscation level (0-3) (default: 0)
  -m           Merge all scripts into client.luac and server.luac
  -d           Suppress decompile warning
  -v           Show version information
  -h           Show help information
```

### Examples

```bash
# Compile ALL resources in a directory with maximum obfuscation and strip debug info
mta-bundler -s -e 3 /path/to/resources/

# Compile all resources to a specific output directory
mta-bundler -o compiled/ /path/to/resources/

# Compile a single resource with obfuscation level 2 and suppress warnings
mta-bundler -e 2 -d /path/to/resource/meta.xml

# Merge all scripts in a single resource into client.luac and server.luac
mta-bundler -m /path/to/resource/

# Process entire server resources folder with custom output
mta-bundler -o /path/to/compiled-server/ /path/to/server/mods/deathmatch/resources/
```

## Obfuscation Levels

| Level | Flag | Description | MTA Version Required |
|-------|------|-------------|---------------------|
| 0     | `-e 0` | No obfuscation | All versions |
| 1     | `-e 1` | Basic obfuscation | All versions |
| 2     | `-e 2` | Enhanced obfuscation | MTA 1.5.2-9.07903+ |
| 3     | `-e 3` | Maximum obfuscation | MTA 1.5.6-9.18728+ |

## How It Works

### Input Processing
The tool supports two input types:
- **Single meta.xml file**: Compiles all scripts referenced in the resource
- **Directory**: Recursively finds ALL `meta.xml` files and compiles each resource

### Processing Workflow
1. **Input Analysis**: Determines if input is file or directory
2. **Resource Discovery**: For directories, recursively searches for all `meta.xml` files
3. **Resource Processing**: For each found resource:
   - **Meta.xml Parsing**: Extracts script file references from meta.xml structure
   - **Lua Compilation**: Compiles each Lua script using `luac_mta` with specified options
   - **File Management**: Copies non-script files to maintain resource structure
   - **Meta.xml Updates**: Updates script references from `.lua` to `.luac` extensions
4. **Output Generation**: Creates organized output directory with compiled resources

### Directory Processing (Batch Mode)

When a directory is provided as input, the tool:

1. **Recursive Search**: Walks through all subdirectories to find `meta.xml` files
2. **Resource Identification**: Each `meta.xml` file represents an MTA resource
3. **Batch Compilation**: Processes all found resources sequentially
4. **Progress Reporting**: Shows current progress (`[1/5] Processing: resource-name`)
5. **Error Handling**: Continues processing other resources if one fails
6. **Structure Preservation**: Maintains directory hierarchy in output

This is particularly useful for:
- Compiling entire server resource folders
- Processing multiple resources with a single command
- Batch deployment preparation

### Merge Mode

When using the merge flag (`-m`), the tool changes its compilation behavior:

1. **Script Grouping**: Groups Lua scripts by type (client, server, shared)
2. **Shared Script Handling**: Merges shared scripts with both client and server groups
3. **Consolidated Compilation**: Compiles all client scripts into a single `client.luac` file and all server scripts into a single `server.luac` file
4. **Meta.xml Updates**: Updates the meta.xml file to reference the merged compiled files instead of individual scripts

This mode is useful for creating simplified resource bundles with just two main script files.

## Project Structure

```
mta-bundler/
├── main.go         # CLI interface and main logic
├── compiler.go     # Lua compilation engine
├── resource.go     # MTA resource processing
├── meta.go         # Meta.xml parsing and structures
├── explorer.go     # Directory traversal and file discovery
├── go.mod          # Go module dependencies
└── README.md       # This file
```

## Configuration

### Binary Detection

The tool automatically detects the `luac_mta` binary in the following locations:

**Windows:**
- `luac_mta.exe` (in PATH)
- `./luac_mta.exe`
- `./bin/luac_mta.exe`
- `C:\bin\luac_mta.exe`

**Linux/macOS:**
- `luac_mta` (in PATH)
- `./luac_mta`
- `./bin/luac_mta`
- `/usr/local/bin/luac_mta`
- `/usr/bin/luac_mta`

### Meta.xml Support

The tool supports all standard MTA meta.xml file references:

- `<script>` - Lua script files
- `<file>` - Client-side files
- `<map>` - Map files
- `<config>` - Configuration files
- `<html>` - HTML files

## Error Handling

- **File Validation**: Checks for file existence and valid extensions
- **Binary Detection**: Provides clear error messages if `luac_mta` is not found
- **Compilation Errors**: Reports detailed compilation failures with context
- **Directory Creation**: Automatically creates output directories as needed

## Dependencies

This tool uses only Go standard library packages with no external dependencies.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source and available under the [MIT License](LICENSE).

## Acknowledgments

- Multi Theft Auto team for the MTA platform and `luac_mta` compiler
- The Go community for excellent tooling and libraries

---

## Practice Project Note

This project serves as a practice exercise for learning:
- **Claude Code**: Exploring AI-assisted development workflows and code generation
- **Go Programming**: Building command-line tools, file processing, and XML parsing in Go

The goal is to create a practical tool while experimenting with modern development practices and AI-assisted coding techniques.