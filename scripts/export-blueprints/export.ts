/**
 * Blueprint Exporter for Stripe CLI Co-op Mode
 *
 * This script imports Blueprint definitions from pay-server's TypeScript source
 * and exports them as CLI-friendly JSON files. It:
 *
 * 1. Resolves MessageDescriptor objects to their defaultMessage strings
 * 2. Strips React/JSX components (display, messageDescriptorFormatters)
 * 3. Retains the structural data as CLI schema fields: steps, nodes, requests, events
 * 4. Writes one JSON file per blueprint to the output directory
 *
 * Usage:
 *   cd scripts/export-blueprints
 *   npm install
 *   npm run export -- --source <path-to-blueprintDefinitions> --out <output-dir>
 *
 * Example:
 *   npm run export -- \
 *     --source ../../pay-server/frontend/workbench/shared/blueprints/src/blueprintDefinitions \
 *     --out ../../api/blueprints
 */

import { writeFileSync, mkdirSync, existsSync, readdirSync, readFileSync } from 'fs';
import { join, resolve, basename } from 'path';
import { parseArgs } from 'util';

const { values } = parseArgs({
  options: {
    source: { type: 'string' },
    out: { type: 'string', default: '../../api/blueprints' },
  },
});

if (!values.source) {
  console.error('Usage: npm run export -- --source <path-to-blueprintDefinitions> --out <output-dir>');
  console.error('');
  console.error('The --source should point to the blueprintDefinitions directory in pay-server:');
  console.error('  e.g. ../../../mint/pay-server/frontend/workbench/shared/blueprints/src/blueprintDefinitions');
  process.exit(1);
}

const sourceDir = resolve(values.source);
const outDir = resolve(values.out!);

if (!existsSync(sourceDir)) {
  console.error(`Source directory not found: ${sourceDir}`);
  process.exit(1);
}

mkdirSync(outDir, { recursive: true });

/**
 * Resolve a MessageDescriptor-like object to its plain string.
 * MessageDescriptors have shape: { id: string, defaultMessage: string, description?: string }
 */
function resolveMessage(value: unknown): string {
  if (typeof value === 'string') return value;
  if (value && typeof value === 'object' && 'defaultMessage' in value) {
    return (value as { defaultMessage: string }).defaultMessage;
  }
  return '';
}

/**
 * Transform a node from the TypeScript definition to CLI-friendly JSON.
 */
function transformNode(node: Record<string, unknown>): Record<string, unknown> | null {
  const type = node.type as string;

  // Skip node types we can't handle in CLI
  if (['settingsUpdate', 'contactStripe', 'enableCustomFeature', 'installApp'].includes(type)) {
    return null;
  }

  const result: Record<string, unknown> = {
    type,
    key: node.key,
    title: resolveMessage(node.title),
  };

  if (node.description) {
    result.description = resolveMessage(node.description);
  }

  // API Request nodes
  if (type === 'apiRequest' && node.request) {
    const req = node.request as Record<string, unknown>;
    const transformed: Record<string, unknown> = {
      path: req.path,
      method: req.method,
    };

    // Get params from configuredDetails or direct params
    if (req.configuredDetails && Array.isArray(req.configuredDetails) && req.configuredDetails.length > 0) {
      const first = req.configuredDetails[0] as Record<string, unknown>;
      if (first.params) {
        transformed.params = first.params;
      }
    } else if (req.params) {
      transformed.params = req.params;
    }

    result.request = transformed;
  }

  // Async handler nodes
  if (type === 'asyncHandler' && node.events) {
    const events = (node.events as Array<Record<string, unknown>>)
      .map(e => e.eventType as string)
      .filter(Boolean);
    result.events = events;
  }

  return result;
}

/**
 * Transform an upstream chapter from the TypeScript definition into a CLI step.
 */
function transformStep(chapter: Record<string, unknown>): Record<string, unknown> | null {
  const nodes = (chapter.nodes as Array<Record<string, unknown>> || [])
    .map(transformNode)
    .filter((n): n is Record<string, unknown> => n !== null);

  if (nodes.length === 0) return null;

  const result: Record<string, unknown> = {
    key: chapter.key,
    title: resolveMessage(chapter.title),
    nodes,
  };

  if (chapter.description) {
    result.description = resolveMessage(chapter.description);
  }
  if (chapter.required) {
    result.required = true;
  }

  return result;
}

/**
 * Transform a full blueprint definition.
 */
function transformBlueprint(id: string, blueprint: Record<string, unknown>): Record<string, unknown> | null {
  const steps = (blueprint.chapters as Array<Record<string, unknown>> || [])
    .map(transformStep)
    .filter((c): c is Record<string, unknown> => c !== null);

  if (steps.length === 0) return null;

  const result: Record<string, unknown> = {
    id,
    title: resolveMessage(blueprint.title),
    type: blueprint.blueprintType || 'learning',
    steps,
  };

  if (blueprint.listViewDescription) {
    result.description = resolveMessage(blueprint.listViewDescription);
  }

  if (blueprint.metadata && typeof blueprint.metadata === 'object') {
    const meta = blueprint.metadata as Record<string, unknown>;
    if (meta.products) {
      result.products = meta.products;
    }
  }

  return result;
}

function validateCLIBlueprint(id: string, blueprint: Record<string, unknown>): void {
  if ('chapters' in blueprint) {
    throw new Error(`${id}: exported blueprint must use "steps", not "chapters"`);
  }
  if ('prompt' in blueprint) {
    throw new Error(`${id}: top-level "prompt" is not supported by the CLI schema`);
  }
  const steps = blueprint.steps as Array<Record<string, unknown>> | undefined;
  if (!Array.isArray(steps) || steps.length === 0) {
    throw new Error(`${id}: exported blueprint must include non-empty "steps"`);
  }
  for (const step of steps) {
    if ('review_granularity' in step) {
      throw new Error(`${id}: step "review_granularity" is not supported by the CLI schema`);
    }
    for (const node of (step.nodes as Array<Record<string, unknown>> || [])) {
      const request = node.request as Record<string, unknown> | undefined;
      if (request && 'key' in request) {
        throw new Error(`${id}: request.key is redundant and not part of the CLI schema`);
      }
    }
  }
}

// Main execution
console.log(`Exporting blueprints from: ${sourceDir}`);
console.log(`Output directory: ${outDir}`);
console.log('');

// This script is designed to be run with the actual TypeScript source
// For now, it provides the framework - actual import requires the pay-server
// module resolution context.
console.log('NOTE: This script requires running within the pay-server module context');
console.log('to resolve TypeScript imports. For standalone use, place pre-exported');
console.log('blueprint objects in a JSON format and use this as a transformer.');
console.log('');
console.log('To use with pay-server:');
console.log('  1. Add this script to pay-server devDependencies context');
console.log('  2. Import rawBlueprintsList from the definitions index');
console.log('  3. Run transformBlueprint on each entry');
console.log('');

// For manual/CI usage: check if source has pre-exported JSON files
const jsonFiles = readdirSync(sourceDir).filter(f => f.endsWith('.json'));
if (jsonFiles.length > 0) {
  console.log(`Found ${jsonFiles.length} JSON files in source, transforming...`);
  for (const file of jsonFiles) {
    try {
      const raw = JSON.parse(readFileSync(join(sourceDir, file), 'utf-8'));
      const id = basename(file, '.json');
      const transformed = transformBlueprint(id, raw);
      if (transformed) {
        validateCLIBlueprint(id, transformed);
        const outPath = join(outDir, `${id}.json`);
        writeFileSync(outPath, JSON.stringify(transformed, null, 2) + '\n');
        console.log(`  ✓ ${id}`);
      }
    } catch (err) {
      console.error(`  ✗ ${file}: ${(err as Error).message}`);
    }
  }
} else {
  console.log('No JSON source files found. To export from TypeScript:');
  console.log('  Create a wrapper that imports rawBlueprintsList and calls transformBlueprint.');
}

console.log('');
console.log('Done.');
