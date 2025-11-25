#!/usr/bin/env python3
"""
Validate that the unified OpenAPI specs match the separate v1/v2 specs.

This script verifies:
1. All v1 GA paths in spec3.unified.sdk.json are present in spec3.sdk.json
2. All v2 GA paths in spec3.unified.sdk.json are present in spec3.v2.sdk.json
3. All v2 preview paths in spec3.unified.sdk.beta.json are present in spec3.v2.sdk.preview.json

Note: v1 preview paths are not validated as they don't exist in the old spec structure.
"""

import json
import sys
from pathlib import Path
from typing import Dict, Set, List, Tuple


def load_spec(file_path: Path) -> Dict:
    """Load an OpenAPI spec file."""
    with open(file_path, 'r') as f:
        return json.load(f)


def get_paths(spec: Dict) -> Set[str]:
    """Extract all paths from an OpenAPI spec."""
    return set(spec.get('paths', {}).keys())


def separate_paths_by_namespace(paths: Set[str]) -> Tuple[Set[str], Set[str]]:
    """
    Separate paths into v1 and v2 namespaces.

    Returns:
        (v1_paths, v2_paths): Tuple of sets containing v1 and v2 paths
    """
    v1_paths = {p for p in paths if p.startswith('/v1/')}
    v2_paths = {p for p in paths if p.startswith('/v2/')}
    return v1_paths, v2_paths


def validate_paths(
    expected: Set[str],
    actual: Set[str],
    spec_name: str,
    namespace: str
) -> List[str]:
    """
    Validate that all expected paths are present in actual paths.

    Returns:
        List of error messages (empty if validation passes)
    """
    errors = []

    missing = expected - actual
    if missing:
        errors.append(f"\n{spec_name}: Missing {len(missing)} {namespace} paths:")
        for path in sorted(missing):
            errors.append(f"  - {path}")

    extra = actual - expected
    if extra:
        errors.append(f"\n{spec_name}: Found {len(extra)} unexpected {namespace} paths:")
        for path in sorted(extra):
            errors.append(f"  + {path}")

    return errors


def main():
    # Define spec file paths
    spec_dir = Path(__file__).parent.parent / 'api' / 'openapi-spec'

    v1_ga_spec_path = spec_dir / 'spec3.sdk.json'
    v2_ga_spec_path = spec_dir / 'spec3.v2.sdk.json'
    v2_preview_spec_path = spec_dir / 'spec3.v2.sdk.preview.json'
    unified_ga_spec_path = spec_dir / 'spec3.cli.json'
    unified_preview_spec_path = spec_dir / 'spec3.cli.preview.json'

    # Verify all files exist
    for path in [v1_ga_spec_path, v2_ga_spec_path, v2_preview_spec_path,
                 unified_ga_spec_path, unified_preview_spec_path]:
        if not path.exists():
            print(f"Error: Spec file not found: {path}")
            sys.exit(1)

    print("Loading OpenAPI specs...")
    print(f"  - {v1_ga_spec_path.name}")
    print(f"  - {v2_ga_spec_path.name}")
    print(f"  - {v2_preview_spec_path.name}")
    print(f"  - {unified_ga_spec_path.name}")
    print(f"  - {unified_preview_spec_path.name}")
    print()

    # Load all specs
    v1_ga_spec = load_spec(v1_ga_spec_path)
    v2_ga_spec = load_spec(v2_ga_spec_path)
    v2_preview_spec = load_spec(v2_preview_spec_path)
    unified_ga_spec = load_spec(unified_ga_spec_path)
    unified_preview_spec = load_spec(unified_preview_spec_path)

    # Extract paths
    v1_ga_paths = get_paths(v1_ga_spec)
    v2_ga_paths = get_paths(v2_ga_spec)
    v2_preview_paths = get_paths(v2_preview_spec)
    unified_ga_paths = get_paths(unified_ga_spec)
    unified_preview_paths = get_paths(unified_preview_spec)

    print(f"Path counts:")
    print(f"  v1 GA (spec3.sdk.json): {len(v1_ga_paths)}")
    print(f"  v2 GA (spec3.v2.sdk.json): {len(v2_ga_paths)}")
    print(f"  v2 Preview (spec3.v2.sdk.preview.json): {len(v2_preview_paths)}")
    print(f"  Unified GA (spec3.unified.sdk.json): {len(unified_ga_paths)}")
    print(f"  Unified Preview (spec3.unified.sdk.beta.json): {len(unified_preview_paths)}")
    print()

    # Separate unified specs by namespace
    unified_ga_v1, unified_ga_v2 = separate_paths_by_namespace(unified_ga_paths)
    unified_preview_v1, unified_preview_v2 = separate_paths_by_namespace(unified_preview_paths)

    print(f"Unified GA namespace breakdown:")
    print(f"  v1 paths: {len(unified_ga_v1)}")
    print(f"  v2 paths: {len(unified_ga_v2)}")
    print()

    print(f"Unified Preview namespace breakdown:")
    print(f"  v1 paths: {len(unified_preview_v1)}")
    print(f"  v2 paths: {len(unified_preview_v2)}")
    print()

    # Collect all errors
    all_errors = []

    # Validation 1: v1 GA paths in unified should match spec3.sdk.json
    print("Validating v1 GA paths...")
    errors = validate_paths(unified_ga_v1, v1_ga_paths, "spec3.sdk.json", "v1")
    all_errors.extend(errors)
    if not errors:
        print(f"  ✓ All {len(unified_ga_v1)} v1 GA paths match")

    # Validation 2: v2 GA paths in unified should match spec3.v2.sdk.json
    print("Validating v2 GA paths...")
    errors = validate_paths(unified_ga_v2, v2_ga_paths, "spec3.v2.sdk.json", "v2")
    all_errors.extend(errors)
    if not errors:
        print(f"  ✓ All {len(unified_ga_v2)} v2 GA paths match")

    # Validation 3: v2 preview paths in unified should match spec3.v2.sdk.preview.json
    print("Validating v2 Preview paths...")
    errors = validate_paths(unified_preview_v2, v2_preview_paths, "spec3.v2.sdk.preview.json", "v2")
    all_errors.extend(errors)
    if not errors:
        print(f"  ✓ All {len(unified_preview_v2)} v2 Preview paths match")

    # Note about v1 preview paths
    if unified_preview_v1:
        print(f"\nNote: Found {len(unified_preview_v1)} v1 Preview paths in unified spec.")
        print("  These cannot be validated as v1 preview paths don't exist in the old spec structure.")

    # Print summary
    print("\n" + "=" * 80)
    if all_errors:
        print("VALIDATION FAILED")
        print("=" * 80)
        for error in all_errors:
            print(error)
        sys.exit(1)
    else:
        print("VALIDATION PASSED")
        print("=" * 80)
        print("\nAll validations passed! The unified specs correctly match the separate specs.")
        sys.exit(0)


if __name__ == '__main__':
    main()
