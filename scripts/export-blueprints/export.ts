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
    ids: { type: 'string' },
  },
});

if (!values.source) {
  console.error('Usage: npm run export -- --source <path-to-exported-json> --out <output-dir>');
  console.error('');
  console.error('The --source should point to pay-server exported blueprint JSON:');
  console.error('  e.g. ../../../mint/pay-server/frontend/workbench/shared/blueprints/dist/blueprints');
  process.exit(1);
}

const sourceDir = resolve(values.source);
const outDir = resolve(values.out!);

if (!existsSync(sourceDir)) {
  console.error(`Source directory not found: ${sourceDir}`);
  process.exit(1);
}

mkdirSync(outDir, { recursive: true });

const requestedIds = values.ids
  ? new Set(values.ids.split(',').map(id => id.trim()).filter(Boolean))
  : null;

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

function buildAPIRequestRefMap(blueprint: Record<string, unknown>): Map<string, string> {
  const refs = new Map<string, string>();
  for (const chapter of (blueprint.chapters as Array<Record<string, unknown>> || [])) {
    const chapterKey = chapter.key as string;
    for (const node of (chapter.nodes as Array<Record<string, unknown>> || [])) {
      const nodeKey = node.key as string;
      const request = node.request as Record<string, unknown> | undefined;
      const requestKey = request?.key as string | undefined;
      if (chapterKey && nodeKey && requestKey) {
        refs.set(`${chapterKey}.${nodeKey}.${requestKey}`, `${chapterKey}.${nodeKey}`);
      }
    }
  }
  return refs;
}

function rewriteNodeReferences(value: unknown, apiRequestRefs: Map<string, string>): unknown {
  if (typeof value === 'string') {
    return value.replace(/\$\{node\.([^:}]+):([^}]+)\}/g, (match, ref, field) => {
      const rewritten = apiRequestRefs.get(ref);
      return rewritten ? `\${node.${rewritten}:${field}}` : match;
    });
  }
  if (Array.isArray(value)) {
    return value.map(item => rewriteNodeReferences(item, apiRequestRefs));
  }
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>).map(([key, entry]) => [
        rewriteNodeReferences(key, apiRequestRefs) as string,
        rewriteNodeReferences(entry, apiRequestRefs),
      ]),
    );
  }
  return value;
}

function sortJSONValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(sortJSONValue);
  }
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([key, entry]) => [key, sortJSONValue(entry)]),
    );
  }
  return value;
}

function reviewPromptFor(type: string): string {
  switch (type) {
    case 'apiRequest':
      return 'Confirm the implementation calls the intended Stripe API, reuses values needed by later steps, and handles API errors without exposing secrets.';
    case 'asyncHandler':
      return 'Run the relevant Stripe CLI trigger or complete the upstream flow, then confirm the handler receives and verifies the expected event.';
    case 'uiComponent':
      return 'Open the app and confirm the user-facing flow works as described.';
    case 'testHelper':
      return 'Run the helper flow and confirm it advances test state without adding helper-only parameters to application code.';
    case 'dashboard':
      return 'Open Dashboard and confirm the required configuration is present before continuing.';
    case 'setUpWebhooks':
      return 'Run the Stripe CLI listener and confirm webhook forwarding reaches the local endpoint.';
    default:
      return 'Confirm this step is complete and report the observable result before continuing.';
  }
}

function descriptionFor(type: string, description: string): string {
  if (type === 'testHelper' && description) {
    return `Run this test helper to ${description.charAt(0).toLowerCase()}${description.slice(1)}`;
  }
  if (type === 'setUpWebhooks' && description) {
    return `Run webhook setup and confirm ${description.charAt(0).toLowerCase()}${description.slice(1)}`;
  }
  return description;
}

function humanizeKey(key: unknown): string {
  if (typeof key !== 'string' || key.trim() === '') {
    return 'Complete API request';
  }
  const words = key
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[-_]+/g, ' ')
    .trim()
    .split(/\s+/);
  return words
    .map((word, index) => {
      const lower = word.toLowerCase();
      if (index > 0 && ['a', 'an', 'and', 'for', 'in', 'of', 'to', 'with'].includes(lower)) {
        return lower;
      }
      return lower.charAt(0).toUpperCase() + lower.slice(1);
    })
    .join(' ');
}

function stringifyLikeGo(value: unknown): string {
  return JSON.stringify(value, null, 2)
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e')
    .replace(/&/g, '\\u0026');
}

function transformTestHelperRequest(request: Record<string, unknown>): Record<string, unknown> {
  const transformed: Record<string, unknown> = {
    key: request.key,
    path: request.path,
    method: request.method,
  };
  if (request.params) {
    transformed.params = sortJSONValue(request.params);
  }
  if (request.hidden_params) {
    transformed.hidden_params = sortJSONValue(request.hidden_params);
  }
  return transformed;
}

/**
 * Transform a node from the TypeScript definition to CLI-friendly JSON.
 */
function transformNode(node: Record<string, unknown>, apiRequestRefs: Map<string, string>): Record<string, unknown> | null {
  const type = node.type as string;

  // Skip node types we can't handle in CLI
  if (['settingsUpdate', 'contactStripe', 'enableCustomFeature', 'installApp', 'reverseApiRequest'].includes(type)) {
    return null;
  }

  const result: Record<string, unknown> = {
    type,
    key: node.key,
    title: resolveMessage(node.title),
  };
  if (result.title === '' || String(result.title).trim().toLowerCase() === 'api request') {
    result.title = humanizeKey(node.key);
  }

  if (node.description) {
    result.description = descriptionFor(type, resolveMessage(node.description));
  }
  result.review_prompt = reviewPromptFor(type);

  // API Request nodes
  if (type === 'apiRequest' && node.request) {
    const req = node.request as Record<string, unknown>;
    const transformed: Record<string, unknown> = {
      path: rewriteNodeReferences(req.path, apiRequestRefs),
      method: req.method,
    };

    // Get params from configuredDetails or direct params
    if (req.configuredDetails && Array.isArray(req.configuredDetails) && req.configuredDetails.length > 0) {
      const first = req.configuredDetails[0] as Record<string, unknown>;
      if (first.params) {
        transformed.params = sortJSONValue(rewriteNodeReferences(first.params, apiRequestRefs));
      }
    } else if (req.params) {
      transformed.params = sortJSONValue(rewriteNodeReferences(req.params, apiRequestRefs));
    }

    result.request = transformed;
  }

  if (type === 'testHelper' && Array.isArray(node.requests)) {
    result.requests = (node.requests as Array<Record<string, unknown>>).map(request =>
      rewriteNodeReferences(transformTestHelperRequest(request), apiRequestRefs),
    );
  }

  // Async handler nodes
  if (type === 'asyncHandler' && node.events) {
    const events = (node.events as Array<Record<string, unknown>>)
      .map(e => e.eventType as string)
      .filter(Boolean);
    if (events.length === 0) {
      return null;
    }
    if (events.length > 0) {
      result.review_command = `stripe trigger ${events[0]}`;
    }
    result.events = events;
  }

  return result;
}

/**
 * Transform an upstream chapter from the TypeScript definition into a CLI step.
 */
function transformStep(chapter: Record<string, unknown>, apiRequestRefs: Map<string, string>): Record<string, unknown> | null {
  const nodes = (chapter.nodes as Array<Record<string, unknown>> || [])
    .map(node => transformNode(node, apiRequestRefs))
    .filter((n): n is Record<string, unknown> => n !== null);

  if (nodes.length === 0) return null;

  const result: Record<string, unknown> = {
    key: chapter.key,
    title: resolveMessage(chapter.title),
  };

  if (chapter.description) {
    result.description = resolveMessage(chapter.description);
  }
  if (chapter.required) {
    result.required = true;
  }
  result.nodes = nodes;

  return result;
}

/**
 * Transform a full blueprint definition.
 */
function transformBlueprint(id: string, blueprint: Record<string, unknown>): Record<string, unknown> | null {
  const apiRequestRefs = buildAPIRequestRefMap(blueprint);
  const steps = (blueprint.chapters as Array<Record<string, unknown>> || [])
    .map(step => transformStep(step, apiRequestRefs))
    .filter((c): c is Record<string, unknown> => c !== null);

  if (steps.length === 0) return null;

  let description = '';
  if (blueprint.listViewDescription) {
    description = resolveMessage(blueprint.listViewDescription);
  } else if (blueprint.detailViewDescription) {
    description = resolveMessage(blueprint.detailViewDescription);
  }
  let products: unknown;
  if (blueprint.metadata && typeof blueprint.metadata === 'object') {
    const meta = blueprint.metadata as Record<string, unknown>;
    if (meta.products) {
      products = meta.products;
    }
  }

  const title = resolveMessage(blueprint.title);
  if (!description && !products) {
    description = `Implement ${title} with Stripe.`;
  }

  const result: Record<string, unknown> = {
    id,
    title,
  };
  if (description) {
    result.description = description;
  }
  result.type = blueprint.blueprintType || 'learning';
  if (products) {
    result.products = products;
  }
  result.settings = null;
  result.steps = steps;

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

// For manual/CI usage: check if source has pre-exported JSON files
const jsonFiles = readdirSync(sourceDir).filter(f => {
  if (!f.endsWith('.json')) return false;
  if (!requestedIds) return true;
  return requestedIds.has(basename(f, '.json'));
});
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
        writeFileSync(outPath, stringifyLikeGo(transformed) + '\n');
        console.log(`  ✓ ${id}`);
      }
    } catch (err) {
      console.error(`  ✗ ${file}: ${(err as Error).message}`);
    }
  }
} else {
  console.error('No matching JSON source files found.');
  if (requestedIds) {
    console.error(`Requested IDs: ${[...requestedIds].join(', ')}`);
  }
  process.exit(1);
}

console.log('');
console.log('Done.');
