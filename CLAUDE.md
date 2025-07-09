# MTA (Multi Theft Auto) Resources Guide

## What is MTA?

Multi Theft Auto (MTA) is a multiplayer modification for Grand Theft Auto: San Andreas that allows players to create custom game modes, scripts, and modifications. MTA extends the original game with a comprehensive scripting system and resource management framework.

## What are Resources?

Resources are a key part of MTA. A resource is essentially a folder or zip file that contains a collection of files - including script files, plus a meta file that describes how the resource should be loaded. A resource can be seen as being partly equivalent to a program running in an operating system - it can be started and stopped, and multiple resources can run at once.

### Key Terminology

- **Resource**: A zip file or folder containing a meta.xml file and a number of resource items, placed in the `mods/deathmatch/resources` folder
- **Resource Item**: A file contained within a resource, including maps, scripts, images, and other assets
- **Meta File**: The core descriptor file (`meta.xml`) that defines how the resource should be loaded and what files it contains

### Resource Characteristics

- Each resource runs in its own virtual machine (VM)
- Variables are not shared between resources
- Resources communicate through exported functions using the `call` scripting function
- Each resource can only be loaded once, the server will ensure this
- Resources can depend on other resources using the `include` tag

## The Meta.xml File

The meta.xml file presents MTA with a set of metadata, such as the resource's name, the scripts to include, and which files to precache for sending to clients among other things. This XML-based file is the core of every resource.

### Essential Meta.xml Tags

#### Basic Information
```xml
<info author="AuthorName" 
      version="1.0.0" 
      name="ResourceName" 
      description="Brief description" 
      type="gamemode|script|map|misc" />
```

#### Scripts
```xml
<script src="filename.lua" type="server|client|shared" cache="true|false" />
```
- `server`: Runs on server only
- `client`: Runs on client only  
- `shared`: Runs on both server and client separately

#### Files (Client-side Resources)
```xml
<file src="image.png" download="true|false" />
```
Used for images, models (.dff), textures (.txd), collision files (.col), etc.

#### Maps
```xml
<map src="mapfile.map" dimension="99" />
```

#### Dependencies
```xml
<include resource="other-resource-name" />
```

#### Exported Functions
```xml
<export function="functionName" type="server|client|shared" http="true|false" />
```

#### Configuration Files
```xml
<config src="config.xml" type="server|client" />
```

### Complete Meta.xml Example

```xml
<meta>
    <info author="Developer" 
          version="1.0.0" 
          name="Example Resource" 
          description="An example resource" 
          type="gamemode" />
    
    <script src="server.lua" type="server" />
    <script src="client.lua" type="client" />
    <script src="shared.lua" type="shared" />
    
    <file src="logo.png" />
    <file src="model.dff" />
    <file src="texture.txd" />
    
    <map src="mymap.map" />
    
    <include resource="scoreboard" />
    <include resource="killmessages" />
    
    <export function="getPlayerCount" type="server" />
    <export function="showMessage" type="client" />
    
    <config src="settings.xml" type="server" />
</meta>
```

## Directory Structure

### File Storage Location

Resources are stored in:
- **Client installation**: `server/mods/deathmatch/resources/`
- **Dedicated server**: `mods/deathmatch/resources/`

### Storage Format

Resources can be stored as:
- **Directory**: A folder containing all resource files
- **Zip file**: A compressed archive of the resource
- **Both**: Directory takes precedence over zip file (useful for development)

### Directory Layout Guidelines

The directory layout is to assist organising resources in the server directory only. Internally MTA:SA still sees all the resources in a big flat list.

Important considerations:
- Do not use the [...] paths in script or config files
- You can move resources around without having to worry about [...] paths
- A resource will not load if it exists twice anywhere in the directory hierarchy

### Example Directory Structure

```
mods/deathmatch/resources/
├── [gamemodes]/
│   ├── race/
│   ├── deathmatch/
│   └── roleplay/
├── [maps]/
│   ├── race-maps/
│   ├── dm-maps/
│   └── rp-maps/
├── [scripts]/
│   ├── admin/
│   ├── anticheat/
│   └── utilities/
└── [misc]/
    ├── gui/
    ├── sounds/
    └── models/
```

This guide provides the essential knowledge needed to understand and work with MTA resources, from basic concepts to advanced configuration options.