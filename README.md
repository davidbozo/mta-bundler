# MTA Bundler

A command-line tool for compiling and bundling Lua scripts for Multi Theft Auto (MTA) servers. This tool processes MTA resources, compiles Lua scripts using [luac_mta](https://wiki.multitheftauto.com/wiki/Lua_compilation_API), and creates optimized bundles with configurable obfuscation levels.

## Features

- **Multiple Input Support**: Process single Lua files, directories, or meta.xml files
- **Obfuscation Levels**: Support for 4 levels of obfuscation (0-3)
- **Resource Processing**: Automatically processes MTA resources based on meta.xml structure
- **Directory Structure Preservation**: Maintains original directory structure in output
- **File Management**: Copies non-script files (maps, configs, assets) to output directory
- **Meta.xml Updates**: Automatically updates script references from `.lua` to `.luac`
- **Cross-Platform**: Works on Windows, Linux, and macOS

## Usage

### Basic Usage

```bash
# Compile a single Lua file
mta-bundler script.lua

# Compile a directory (searches for meta.xml files)
mta-bundler /path/to/resources/

# Compile a specific meta.xml file
mta-bundler /path/to/resource/meta.xml
```

### Command Line Options

```bash
mta-bundler [OPTIONS] [input_path]

Options:
  -o, --output string      Output directory for compiled files (default: same as source)
  -s, --strip             Strip debug information
  -e, --obfuscate int     Obfuscation level (0-3) (default: 0)
  -2, --obfuscate2        Obfuscation level 2 (equivalent to -e2)
  -3, --obfuscate3        Obfuscation level 3 (equivalent to -e3)
  -d, --suppress          Suppress decompile warning
  -v, --version           Show version information
  -h, --help              Show help information
```

### Examples

```bash
# Compile with maximum obfuscation and strip debug info
mta-bundler -s -e3 /path/to/resource/

# Compile to a specific output directory
mta-bundler -o compiled/ /path/to/resource/

# Compile with obfuscation level 2 and suppress warnings
mta-bundler -e2 -d script.lua

# Using shorthand flags
mta-bundler -3 -s /path/to/resource/
```

## Obfuscation Levels

| Level | Flag | Description | MTA Version Required |
|-------|------|-------------|---------------------|
| 0     | None | No obfuscation | All versions |
| 1     | `-e` | Basic obfuscation | All versions |
| 2     | `-e2` | Enhanced obfuscation | MTA 1.5.2-9.07903+ |
| 3     | `-e3` | Maximum obfuscation | MTA 1.5.6-9.18728+ |

## How It Works

1. **Input Processing**: The tool accepts single files, directories, or meta.xml files
2. **Resource Discovery**: For directories, it searches for all meta.xml files recursively
3. **Meta.xml Parsing**: Extracts file references from meta.xml structure
4. **Lua Compilation**: Compiles each Lua script using `luac_mta` with specified options
5. **File Management**: Copies non-script files to maintain resource structure
6. **Meta.xml Updates**: Updates script references from `.lua` to `.luac` extensions
7. **Output Generation**: Creates organized output directory with compiled resources

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

- [spf13/cobra](https://github.com/spf13/cobra) - Command-line interface framework

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