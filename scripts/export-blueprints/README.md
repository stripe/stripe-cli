# Blueprint Exporter

Converts exported Workbench Blueprint JSON into CLI-friendly JSON for co-op mode.

## Pipeline

```
pay-server/blueprintDefinitions/*.tsx
    ↓  (pay-server: pay js:run export)
pay-server/dist/blueprints/*.json
    ↓  (this script)
pkg/coop/blueprints/*.json  (checked into stripe-cli and embedded via //go:embed)
```

## Usage

### From pay-server source

Point `BLUEPRINT_SOURCE` at pay-server's `src/blueprintDefinitions` directory.
The Make target runs pay-server's exporter first, then transforms the exported
JSON into the CLI schema:

```bash
BLUEPRINT_SOURCE=/path/to/pay-server/frontend/workbench/shared/blueprints/src/blueprintDefinitions make sync-blueprints
```

By default, the Make target syncs Workbench learning blueprints that represent
merchant integration guides. Testing blueprints, examples, health alert helpers,
and partner-certification/dashboard-only flows are left out of the coop catalog.
To try exporting every pay-server blueprint, run with `BLUEPRINT_IDS=all`.

### From exported JSON

If pay-server has already produced `dist/blueprints/*.json`, point
`BLUEPRINT_SOURCE` there:

```bash
BLUEPRINT_SOURCE=/path/to/pay-server/frontend/workbench/shared/blueprints/dist/blueprints make sync-blueprints
```

## What gets stripped

- `MessageDescriptor` objects → resolved to `defaultMessage` string
- React JSX components (`display`, `messageDescriptorFormatters`) → removed
- Dashboard-only nodes (`settingsUpdate`, `contactStripe`, etc.) → removed
- Environment conditions (`hiddenIfEnvMatchesOne`, etc.) → removed

## What gets kept

- Blueprint structure: id, title, description, type, steps
- Node types: apiRequest, asyncHandler, uiComponent, testHelper, dashboard
- API request details: path, method, params (from first configuredDetails)
- Interpolation strings: `${node.chapter.node:field}` preserved as-is
- Event types for asyncHandler nodes
- Product metadata
