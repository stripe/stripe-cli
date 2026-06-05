# Blueprint Exporter

Converts Workbench Blueprint TypeScript definitions into CLI-friendly JSON for co-op mode.

## Pipeline

```
pay-server/blueprintDefinitions/*.tsx
    ↓  (this script)
api/blueprints/*.json  (checked into stripe-cli)
    ↓  (go generate)
pkg/coop/blueprints/*.json  (embedded via //go:embed)
```

## Usage

### From pre-exported JSON (recommended for CI)

If someone has already exported the raw blueprint objects as JSON:

```bash
cd scripts/export-blueprints
npm install
npm run export -- --source ./raw-exports --out ../../api/blueprints
```

### From pay-server source (requires module context)

The TypeScript blueprints use `MessageDescriptor` objects and React components
that require pay-server's module resolution. To export from source:

1. Copy this script into the pay-server tree (or symlink)
2. Add an entry point that imports `rawBlueprintsList` from the definitions index
3. Call `transformBlueprint` on each entry and write to the output directory

### After exporting

Run the Go generator to copy and validate the blueprints into the embed directory:

```bash
make generate-blueprints
```

## What gets stripped

- `MessageDescriptor` objects → resolved to `defaultMessage` string
- React JSX components (`display`, `messageDescriptorFormatters`) → removed
- Dashboard-only nodes (`settingsUpdate`, `contactStripe`, etc.) → removed
- Environment conditions (`hiddenIfEnvMatchesOne`, etc.) → removed

## What gets kept

- Blueprint structure: id, title, description, type, chapters
- Node types: apiRequest, asyncHandler, uiComponent, testHelper, dashboard
- API request details: path, method, params (from first configuredDetails)
- Interpolation strings: `${node.chapter.node:field}` preserved as-is
- Event types for asyncHandler nodes
- Product metadata
