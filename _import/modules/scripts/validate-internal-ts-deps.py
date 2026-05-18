from __future__ import annotations

import json
import sys
from pathlib import Path


DEPENDENCY_SECTIONS = ("dependencies", "optionalDependencies", "peerDependencies", "devDependencies")
INTERNAL_SCOPE = "@devpablocristo/modules-"


def fail(message: str) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(1)


def load_json(path: Path) -> dict:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        fail(f"invalid JSON in {path}: {exc}")


def repo_files(repo_root: Path, pattern: str) -> list[Path]:
    return sorted(path for path in repo_root.glob(pattern) if "node_modules" not in path.parts)


def discover_ts_packages(repo_root: Path) -> dict[str, Path]:
    packages: dict[str, Path] = {}

    for package_json in repo_files(repo_root, "**/ts/package.json"):
        package_dir = package_json.parent
        data = load_json(package_json)
        name = data.get("name")
        if not isinstance(name, str):
            fail(f"missing package name: {package_json}")

        if name.startswith(INTERNAL_SCOPE):
            packages[name] = package_dir.resolve()

    if not packages:
        fail("no internal TS packages discovered")

    return packages


def validate_manifest_dependency(
    package_dir: Path,
    source: str,
    section: str,
    dependency_name: str,
    dependency_value: object,
    internal_packages: dict[str, Path],
) -> int:
    if dependency_name not in internal_packages:
        return 0

    if not isinstance(dependency_value, str) or not dependency_value.startswith("file:"):
        fail(
            f"{source}: internal dependency {section}.{dependency_name} "
            f"must use file:, got {dependency_value!r}"
        )

    target = (package_dir / dependency_value.removeprefix("file:")).resolve()
    expected = internal_packages[dependency_name]
    if target != expected:
        fail(
            f"{source}: internal dependency {section}.{dependency_name} "
            f"points to {target}, expected {expected}"
        )

    return 1


def validate_package_json(package_json: Path, internal_packages: dict[str, Path]) -> int:
    package_dir = package_json.parent
    data = load_json(package_json)
    count = 0

    for section in DEPENDENCY_SECTIONS:
        dependencies = data.get(section)
        if not isinstance(dependencies, dict):
            continue

        for dependency_name, dependency_value in dependencies.items():
            count += validate_manifest_dependency(
                package_dir,
                str(package_json),
                section,
                dependency_name,
                dependency_value,
                internal_packages,
            )

    return count


def validate_package_lock(package_lock: Path, internal_packages: dict[str, Path]) -> int:
    data = load_json(package_lock)
    package_dir = package_lock.parent
    packages = data.get("packages")
    if not isinstance(packages, dict):
        fail(f"missing packages object in {package_lock}")

    count = 0
    for package_path, package_data in packages.items():
        if not isinstance(package_data, dict):
            continue

        if package_path == "":
            lock_package_dir = package_dir
            source = str(package_lock)
        else:
            lock_package_dir = (package_dir / package_path).resolve()
            source = f"{package_lock}:{package_path}"

        for section in DEPENDENCY_SECTIONS:
            dependencies = package_data.get(section)
            if not isinstance(dependencies, dict):
                continue

            for dependency_name, dependency_value in dependencies.items():
                count += validate_manifest_dependency(
                    lock_package_dir,
                    source,
                    section,
                    dependency_name,
                    dependency_value,
                    internal_packages,
                )

    for package_path, package_data in packages.items():
        if not package_path.startswith("node_modules/"):
            continue
        if not isinstance(package_data, dict):
            continue

        dependency_name = package_path.removeprefix("node_modules/")
        if dependency_name not in internal_packages:
            continue

        resolved = package_data.get("resolved")
        if isinstance(resolved, str) and resolved.startswith("https://registry.npmjs.org/"):
            fail(f"{package_lock}: internal dependency {dependency_name} resolves from registry: {resolved}")

        if isinstance(resolved, str):
            target = (package_dir / resolved).resolve()
            expected = internal_packages[dependency_name]
            if target != expected:
                fail(f"{package_lock}: internal dependency {dependency_name} resolves to {target}, expected {expected}")

    return count


def validate_tsconfig(tsconfig: Path) -> None:
    data = load_json(tsconfig)
    compiler_options = data.get("compilerOptions")
    if not isinstance(compiler_options, dict):
        return

    paths = compiler_options.get("paths")
    if not isinstance(paths, dict):
        return

    for alias in paths:
        if alias.startswith(INTERNAL_SCOPE):
            fail(f"{tsconfig}: internal modules must resolve through file: package deps, not compilerOptions.paths")


def main() -> None:
    repo_root = Path(__file__).resolve().parents[1]
    internal_packages = discover_ts_packages(repo_root)
    internal_dependency_count = 0

    for package_json in repo_files(repo_root, "**/ts/package.json"):
        internal_dependency_count += validate_package_json(package_json, internal_packages)

    package_locks = [repo_root / "package-lock.json", *repo_files(repo_root, "**/ts/package-lock.json")]
    for package_lock in package_locks:
        if not package_lock.exists():
            continue
        internal_dependency_count += validate_package_lock(package_lock, internal_packages)

    for tsconfig in repo_files(repo_root, "**/ts/tsconfig.json"):
        validate_tsconfig(tsconfig)

    print(f"validated {internal_dependency_count} local internal TS dependency references")


if __name__ == "__main__":
    main()
