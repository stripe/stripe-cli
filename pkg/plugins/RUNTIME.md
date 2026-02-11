# Plugin Runtime Support

This document describes the Node.js runtime installation feature for Stripe CLI plugins.

## Overview

Plugins can declare runtime dependencies in their `plugins.toml` manifest. When a plugin requires a specific Node.js version, the Stripe CLI will automatically download and install the required runtime during plugin installation.

## Manifest Configuration

In `plugins.toml`, specify the required Node.js version using the `Runtime` field:

```toml
[[Plugin.Release]]
  Arch = "amd64"
  OS = "darwin"
  Version = "1.0.0"
  Sum = "abc123..."
  Runtime = {node = "20"}  # Requires Node.js 20.x LTS
```

### Supported Node.js Versions

Only LTS (Long-Term Support) versions of Node.js are supported:
- Node.js 20.x (Iron) - v20.18.1
- Node.js 22.x (upcoming LTS)
- Node.js 24.x (future LTS)

## Runtime Installation

When a user installs a plugin with runtime requirements:

```bash
$ stripe plugin install generate
downloading Node.js v20.18.1 runtime...
installing 'generate' v1.0.0...
✔ installation complete.
```

The runtime is:
1. Downloaded from https://nodejs.org/dist
2. Verified using SHA256 checksums
3. Extracted to `~/.config/stripe/runtimes/node/<version>/`
4. Reused for other plugins requiring the same version

## Runtime Execution

When a plugin is executed, the CLI automatically detects if it requires a runtime:

```bash
$ stripe generate create-component
```

Behind the scenes:
1. CLI reads the plugin's `Release` entry from `plugins.toml`
2. Checks if the `Runtime` field is present
3. If Node.js runtime is required:
   - Locates the installed Node.js binary at `~/.config/stripe/runtimes/node/<version>/bin/node`
   - Executes: `node /path/to/plugin.js [args...]`
4. If no runtime required:
   - Executes plugin binary directly: `/path/to/plugin [args...]`

The plugin continues to use the HashiCorp go-plugin framework for RPC/gRPC communication with the CLI, regardless of whether it's a native binary or JavaScript.

## Directory Structure

```
~/.config/stripe/
├── plugins/
│   └── generate/
│       └── 1.0.0/
│           └── stripe-cli-generate
└── runtimes/
    └── node/
        └── 20.18.1/
            └── bin/
                └── node
```

## Platform Support

The runtime installer supports:
- **macOS**: darwin/amd64, darwin/arm64 (Apple Silicon)
- **Linux**: linux/amd64, linux/arm64
- **Windows**: windows/amd64 (coming soon)

## Implementation Details

### Hardcoded Configurations

Runtime versions and checksums are hardcoded in `runtime.go`:
- Provides offline verification without network calls
- Ensures security through checksum validation
- Simplifies deployment (no external config files needed)

### Checksum Verification

All downloaded runtimes are verified against hardcoded SHA256 checksums from the official Node.js releases at https://nodejs.org/dist/vX.Y.Z/SHASUMS256.txt

### Deduplication

Multiple plugins can share the same runtime installation. The CLI checks if a runtime is already installed before downloading.

## Updating Checksums

When adding or updating Node.js versions:

1. Download the SHASUMS256.txt from https://nodejs.org/dist/vX.Y.Z/
2. Find checksums for:
   - `node-vX.Y.Z-darwin-x64.tar.gz`
   - `node-vX.Y.Z-darwin-arm64.tar.gz`
   - `node-vX.Y.Z-linux-x64.tar.gz`
   - `node-vX.Y.Z-linux-arm64.tar.gz`
   - `node-vX.Y.Z-win-x64.zip`
3. Update the `nodeRuntimeConfigs` map in `runtime.go`

## JavaScript Plugin Structure

For a JavaScript plugin to work with the Stripe CLI, it must:

1. **Entry Point**: The downloaded "binary" is actually a `.js` file (e.g., `stripe-cli-generate.js`)

2. **Shebang (Optional)**: Include a Node.js shebang for Unix-like systems:
   ```javascript
   #!/usr/bin/env node
   ```

3. **go-plugin Protocol**: Implement the HashiCorp go-plugin protocol using a Node.js library like [`@hashicorp/go-plugin`](https://github.com/hashicorp/node-go-plugin) to:
   - Handle the plugin handshake
   - Implement RPC/gRPC interface
   - Communicate with the CLI

4. **Example Structure**:
   ```javascript
   #!/usr/bin/env node
   const plugin = require('@hashicorp/go-plugin');

   // Implement your plugin logic
   class MyPlugin {
     async runCommand(args) {
       // Your plugin implementation
       return { success: true };
     }
   }

   // Set up plugin server
   plugin.server({
     myPlugin: new MyPlugin()
   });
   ```

The CLI handles the runtime invocation transparently - plugin developers don't need to worry about how Node.js is launched.

## Testing

Tests are located in `runtime_test.go` and cover:
- Runtime path generation
- Installation detection
- Download URL construction
- Checksum verification
- Release requirement extraction
- Runtime-based execution path

Run tests with:
```bash
go test ./pkg/plugins/... -v
```
