#!/usr/bin/env python3
"""Rename Go module paths and npm package names from core/modules to platform.

Used in two contexts:
1. Fase A5 (in platform/): rename declarations and internal imports.
2. Fase A8 (in pymes/): rename consumer-side imports to point to platform.

Renaming rules:
- Go: changes `module github.com/devpablocristo/{core,modules}/X` to platform path.
- Go: changes `"github.com/devpablocristo/{core,modules}/X"` in imports.
- Go: changes `\\tgithub.com/devpablocristo/{core,modules}/X` in go.mod requires.
- npm: changes `"@devpablocristo/core-X"` and `"@devpablocristo/modules-X"` to platform-*.
- npm: changes `from '@devpablocristo/...'` and `from "@devpablocristo/..."` in TS.

Anchors prevent false positives:
- ONLY .go, go.mod, go.work for Go renames.
- ONLY .ts, .tsx, .js, .jsx, .json for npm renames.
- Specific path segments (longest first) to avoid prefix collisions.
"""
from __future__ import annotations
import argparse
import re
import sys
from pathlib import Path

# Mapping table: longest paths FIRST to avoid prefix collisions
# (e.g. "core/ai/contracts/go" must precede "core/ai/go").
GO_MODULE_MAP: list[tuple[str, str]] = [
    # --- Multi-segment / contracts ---
    ("github.com/devpablocristo/core/ai/contracts/go", "github.com/devpablocristo/platform/contracts/ai/go"),
    ("github.com/devpablocristo/core/calendar/sync/google/go", "github.com/devpablocristo/platform/sdks/google-calendar/go"),
    ("github.com/devpablocristo/core/calendar/ics/go", "github.com/devpablocristo/platform/calendar/ics/go"),
    ("github.com/devpablocristo/core/databases/postgres/go", "github.com/devpablocristo/platform/databases/postgres/go"),
    ("github.com/devpablocristo/core/databases/dynamodb/go", "github.com/devpablocristo/platform/databases/dynamodb/go"),
    ("github.com/devpablocristo/core/providers/aws/lambda/go", "github.com/devpablocristo/platform/sdks/aws/lambda/go"),
    ("github.com/devpablocristo/core/providers/aws/s3/go", "github.com/devpablocristo/platform/sdks/aws/s3/go"),
    ("github.com/devpablocristo/core/providers/aws/sqs/go", "github.com/devpablocristo/platform/sdks/aws/sqs/go"),
    ("github.com/devpablocristo/core/http/gin/go", "github.com/devpablocristo/platform/http/gin/go"),

    # --- Kernels ---
    ("github.com/devpablocristo/core/saas/go", "github.com/devpablocristo/platform/kernels/saas/go"),
    ("github.com/devpablocristo/core/governance/go", "github.com/devpablocristo/platform/kernels/governance/go"),
    ("github.com/devpablocristo/core/activity/go", "github.com/devpablocristo/platform/kernels/activity/go"),
    ("github.com/devpablocristo/core/artifact/go", "github.com/devpablocristo/platform/kernels/artifact/go"),
    ("github.com/devpablocristo/core/ai/go", "github.com/devpablocristo/platform/kernels/ai/go"),

    # --- Rename of capability: scheduling primitive -> jobs ---
    ("github.com/devpablocristo/core/scheduling/go", "github.com/devpablocristo/platform/jobs/go"),

    # --- L0 capabilities (single segment) ---
    ("github.com/devpablocristo/core/errors/go", "github.com/devpablocristo/platform/errors/go"),
    ("github.com/devpablocristo/core/validate/go", "github.com/devpablocristo/platform/validate/go"),
    ("github.com/devpablocristo/core/config/go", "github.com/devpablocristo/platform/config/go"),
    ("github.com/devpablocristo/core/observability/go", "github.com/devpablocristo/platform/observability/go"),
    ("github.com/devpablocristo/core/http/go", "github.com/devpablocristo/platform/http/go"),
    ("github.com/devpablocristo/core/security/go", "github.com/devpablocristo/platform/security/go"),
    ("github.com/devpablocristo/core/concurrency/go", "github.com/devpablocristo/platform/concurrency/go"),
    ("github.com/devpablocristo/core/authn/go", "github.com/devpablocristo/platform/authn/go"),
    ("github.com/devpablocristo/core/authz/go", "github.com/devpablocristo/platform/authz/go"),
    ("github.com/devpablocristo/core/eventing/go", "github.com/devpablocristo/platform/eventing/go"),
    ("github.com/devpablocristo/core/webhook/go", "github.com/devpablocristo/platform/webhook/go"),
    ("github.com/devpablocristo/core/notifications/go", "github.com/devpablocristo/platform/notifications/go"),
    ("github.com/devpablocristo/core/ingestion/go", "github.com/devpablocristo/platform/ingestion/go"),

    # --- modules/ ---
    ("github.com/devpablocristo/modules/scheduling/go", "github.com/devpablocristo/platform/features/scheduling/go"),
    ("github.com/devpablocristo/modules/crud/paths/go", "github.com/devpablocristo/platform/features/crud/paths/go"),
    ("github.com/devpablocristo/modules/crud/archive/go", "github.com/devpablocristo/platform/features/crud/archive/go"),
]

# Sort by length descending to ensure longest-prefix matches first
GO_MODULE_MAP.sort(key=lambda pair: -len(pair[0]))

NPM_PACKAGE_MAP: list[tuple[str, str]] = [
    # core/
    ("@devpablocristo/core-authn", "@devpablocristo/platform-authn"),
    ("@devpablocristo/core-browser", "@devpablocristo/platform-browser"),
    ("@devpablocristo/core-http", "@devpablocristo/platform-http"),
    ("@devpablocristo/core-fsm", "@devpablocristo/platform-fsm"),
    ("@devpablocristo/core-ai-contracts", "@devpablocristo/platform-ai-contracts"),

    # modules/
    ("@devpablocristo/modules-scheduling", "@devpablocristo/platform-scheduling"),
    ("@devpablocristo/modules-search", "@devpablocristo/platform-search"),
    ("@devpablocristo/modules-shell-sidebar", "@devpablocristo/platform-shell-sidebar"),
    ("@devpablocristo/modules-admin-insights", "@devpablocristo/platform-admin-insights"),
    ("@devpablocristo/modules-calendar-board", "@devpablocristo/platform-calendar-board"),
    ("@devpablocristo/modules-kanban-board", "@devpablocristo/platform-kanban-board"),
    ("@devpablocristo/modules-crud-ui", "@devpablocristo/platform-crud-ui"),

    # modules/ui/* (drop the 'ui-' prefix for verticals; keep for primitives)
    ("@devpablocristo/modules-ui-conversation-inbox", "@devpablocristo/platform-conversation-inbox"),
    ("@devpablocristo/modules-ui-notification-feed", "@devpablocristo/platform-notification-feed"),

    # UI primitives (keep 'ui-' prefix)
    ("@devpablocristo/modules-ui-modal", "@devpablocristo/platform-ui-modal"),
    ("@devpablocristo/modules-ui-data-display", "@devpablocristo/platform-ui-data-display"),
    ("@devpablocristo/modules-ui-section-hub", "@devpablocristo/platform-ui-section-hub"),
    ("@devpablocristo/modules-ui-page-shell", "@devpablocristo/platform-ui-page-shell"),
]
NPM_PACKAGE_MAP.sort(key=lambda pair: -len(pair[0]))

PYTHON_PACKAGE_MAP: list[tuple[str, str]] = [
    ("devpablocristo-core-ai", "devpablocristo-platform-ai-runtime"),
    ("devpablocristo-httpserver", "devpablocristo-platform-http"),
]


GO_EXTS = {".go"}
GO_MOD_FILES = {"go.mod", "go.work"}
NPM_EXTS = {".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}
NPM_MANIFEST = "package.json"
PYTHON_MANIFEST = "pyproject.toml"

SKIP_DIRS = {".git", "node_modules", "target", "__pycache__", ".venv", "venv", "dist", "build", "bin", ".pytest_cache", ".next", "coverage"}


def iter_files(root: Path):
    """Walk the tree skipping known generated/cache dirs."""
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        parts = set(path.parts)
        if parts & SKIP_DIRS:
            continue
        yield path


def rewrite_go(path: Path, dry_run: bool) -> int:
    """Rewrite Go imports/module decls. Returns count of replacements."""
    is_go_mod = path.name in GO_MOD_FILES
    is_go_src = path.suffix in GO_EXTS
    if not (is_go_mod or is_go_src):
        return 0

    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return 0
    original = text
    count = 0

    for old, new in GO_MODULE_MAP:
        # Quoted import in .go files: "github.com/devpablocristo/core/x"
        if is_go_src:
            pattern = f'"{old}'
            replacement = f'"{new}'
            occurrences = text.count(pattern)
            if occurrences:
                text = text.replace(pattern, replacement)
                count += occurrences

        # go.mod / go.work patterns
        if is_go_mod:
            # `module <old>` declaration
            module_decl_re = re.compile(rf"^module {re.escape(old)}(\s|$)", re.MULTILINE)
            new_text, n = module_decl_re.subn(f"module {new}\\1", text)
            if n:
                text = new_text
                count += n

            # `\tgithub.com/...` in require/replace directives (tab-indented)
            tab_pattern = f"\t{old}"
            tab_replacement = f"\t{new}"
            n2 = text.count(tab_pattern)
            if n2:
                text = text.replace(tab_pattern, tab_replacement)
                count += n2

            # `  github.com/...` in require/replace (sometimes spaces)
            space_pattern_re = re.compile(rf"(\s){re.escape(old)}(\s|$)")
            new_text, n3 = space_pattern_re.subn(f"\\1{new}\\2", text)
            if n3:
                text = new_text
                count += n3

    if count and not dry_run:
        path.write_text(text, encoding="utf-8")
    return count


def rewrite_npm(path: Path, dry_run: bool) -> int:
    """Rewrite npm package names. Returns count of replacements."""
    is_package_json = path.name == NPM_MANIFEST
    is_ts = path.suffix in NPM_EXTS
    if not (is_package_json or is_ts):
        return 0

    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return 0
    count = 0

    for old, new in NPM_PACKAGE_MAP:
        for delim_open, delim_close in [('"', '"'), ("'", "'")]:
            pattern = f"{delim_open}{old}"
            replacement = f"{delim_open}{new}"
            occurrences = text.count(pattern)
            if occurrences:
                text = text.replace(pattern, replacement)
                count += occurrences

    if count and not dry_run:
        path.write_text(text, encoding="utf-8")
    return count


def rewrite_python(path: Path, dry_run: bool) -> int:
    """Rewrite Python package names in pyproject.toml."""
    if path.name != PYTHON_MANIFEST:
        return 0
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return 0
    count = 0
    for old, new in PYTHON_PACKAGE_MAP:
        for delim in ['"', "'"]:
            pattern = f'name = {delim}{old}{delim}'
            replacement = f'name = {delim}{new}{delim}'
            occurrences = text.count(pattern)
            if occurrences:
                text = text.replace(pattern, replacement)
                count += occurrences
    if count and not dry_run:
        path.write_text(text, encoding="utf-8")
    return count


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("root", help="Directory tree to rewrite")
    parser.add_argument("--dry-run", action="store_true", help="Only report; do not write")
    parser.add_argument("--skip-go", action="store_true")
    parser.add_argument("--skip-npm", action="store_true")
    parser.add_argument("--skip-python", action="store_true")
    args = parser.parse_args()

    root = Path(args.root).resolve()
    if not root.is_dir():
        print(f"Not a directory: {root}", file=sys.stderr)
        return 2

    totals = {"go_files": 0, "go_repls": 0, "npm_files": 0, "npm_repls": 0, "py_files": 0, "py_repls": 0}
    for path in iter_files(root):
        if not args.skip_go:
            n = rewrite_go(path, args.dry_run)
            if n:
                totals["go_files"] += 1
                totals["go_repls"] += n
        if not args.skip_npm:
            n = rewrite_npm(path, args.dry_run)
            if n:
                totals["npm_files"] += 1
                totals["npm_repls"] += n
        if not args.skip_python:
            n = rewrite_python(path, args.dry_run)
            if n:
                totals["py_files"] += 1
                totals["py_repls"] += n

    print(f"Go: {totals['go_repls']} replacements across {totals['go_files']} files")
    print(f"npm: {totals['npm_repls']} replacements across {totals['npm_files']} files")
    print(f"Python: {totals['py_repls']} replacements across {totals['py_files']} files")
    if args.dry_run:
        print("(dry-run, no writes)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
