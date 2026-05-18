from __future__ import annotations

import json
import re
import shutil
import sys
from pathlib import Path

SEMVER_RE = re.compile(r"^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$")


def fail(message: str) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(1)


def load_version(target_dir: Path) -> str:
    version_file = target_dir / "VERSION"
    if not version_file.exists():
        fail(f"missing VERSION in dependency target: {target_dir}")
    version = version_file.read_text(encoding="utf-8").strip()
    if not SEMVER_RE.match(version):
        fail(f"invalid VERSION in dependency target {target_dir}: {version}")
    return version


def rewrite_file_dependencies(module_dir: Path, package_data: dict) -> None:
    for section in ("dependencies", "optionalDependencies", "peerDependencies", "devDependencies"):
        deps = package_data.get(section)
        if not isinstance(deps, dict):
            continue

        for package_name, package_version in list(deps.items()):
            if not isinstance(package_version, str) or not package_version.startswith("file:"):
                continue

            relative_target = package_version.removeprefix("file:")
            target_dir = (module_dir / relative_target).resolve()
            if not target_dir.exists():
                fail(f"dependency target does not exist for {package_name}: {target_dir}")

            target_version = load_version(target_dir)
            deps[package_name] = f"^{target_version}"


def main() -> None:
    if len(sys.argv) != 3:
        fail("usage: prepare-ts-package-release.py <module/runtime> <output-dir>")

    module_rel = Path(sys.argv[1])
    output_dir = Path(sys.argv[2]).resolve()
    module_dir = (Path(__file__).resolve().parents[1] / module_rel).resolve()

    package_json = module_dir / "package.json"
    if not package_json.exists():
        fail(f"package.json not found in module: {module_rel}")

    if output_dir.exists():
        shutil.rmtree(output_dir)

    shutil.copytree(module_dir, output_dir)

    out_package_json = output_dir / "package.json"
    package_data = json.loads(out_package_json.read_text(encoding="utf-8"))

    package_data["private"] = False
    publish_config = package_data.get("publishConfig", {})
    publish_config.setdefault("access", "public")
    package_data["publishConfig"] = publish_config

    if "files" not in package_data:
        files = [candidate for candidate in ("src", "README.md", "LICENSE") if (output_dir / candidate).exists()]
        if files:
            package_data["files"] = files

    rewrite_file_dependencies(module_dir, package_data)

    out_package_json.write_text(json.dumps(package_data, indent=2) + "\n", encoding="utf-8")

    package_lock = output_dir / "package-lock.json"
    if package_lock.exists():
        package_lock.unlink()


if __name__ == "__main__":
    main()
