#!/usr/bin/env python3
"""Check independent modules using temporary copies and a local release proxy."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[1]
PREFIX = "github.com/time-origin/warpin-go-common/"
VERSION = "v0.1.0"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--module", action="append", help="Check only this module (repeatable)")
    parser.add_argument("command", nargs=argparse.REMAINDER, help="Go command, default: test ./...")
    args = parser.parse_args()
    workspace = json.loads(subprocess.check_output(["go", "work", "edit", "-json"], cwd=ROOT))
    modules = sorted((ROOT / item["DiskPath"]).resolve() for item in workspace["Use"])
    if set(modules) != {p.parent.resolve() for p in ROOT.glob("warpin-*/go.mod")}:
        parser.error("go.work must contain every module")
    names = {p.name for p in modules}
    if args.module and not set(args.module) <= names:
        parser.error("unknown module: " + ", ".join(set(args.module) - names))
    command = args.command or ["test", "./..."]
    if command[0] not in ("test", "vet", "build"):
        parser.error("command must be test, vet or build")
    original_cache = Path(subprocess.check_output(["go", "env", "GOMODCACHE"], cwd=ROOT).decode().strip())
    with tempfile.TemporaryDirectory(prefix="warpin-module-check-") as temporary:
        temporary = Path(temporary)
        proxy = temporary / "proxy"
        for module in modules:
            module_path = PREFIX + module.name
            release = proxy / module_path / "@v"
            release.mkdir(parents=True)
            (release / "list").write_text(VERSION + "\n")
            (release / (VERSION + ".info")).write_text(json.dumps({"Version": VERSION, "Time": "2026-10-01T00:00:00Z"}))
            shutil.copyfile(module / "go.mod", release / (VERSION + ".mod"))
            with zipfile.ZipFile(release / (VERSION + ".zip"), "w", zipfile.ZIP_DEFLATED) as archive:
                for source in sorted(module.rglob("*")):
                    if source.is_file() and source.name != ".DS_Store":
                        archive.writestr(module_path + "@" + VERSION + "/" + source.relative_to(module).as_posix(), source.read_bytes())
        env = os.environ.copy()
        env["GOWORK"] = "off"
        env["GOMODCACHE"] = str(temporary / "cache")
        env["GOPROXY"] = ",".join((proxy.as_uri(), (original_cache / "cache/download").as_uri(), env.get("GOPROXY", "https://proxy.golang.org,direct")))
        env["GONOSUMDB"] = ",".join(filter(None, (env.get("GONOSUMDB"), PREFIX + "*")))
        for module in modules:
            if args.module and module.name not in args.module:
                continue
            copy = temporary / "sources" / module.name
            shutil.copytree(module, copy)
            print("Checking " + module.name + ": go " + " ".join(command), flush=True)
            # Resolve local unpublished release checksums only in temporary copies.
            subprocess.run(["go", "mod", "tidy"], cwd=copy, env=env, check=True)
            if (copy / "go.mod").read_bytes() != (module / "go.mod").read_bytes():
                raise RuntimeError(module.name + ": go.mod is not tidy independently")
            subprocess.run(["go", command[0], "-mod=readonly", *command[1:]], cwd=copy, env=env, check=True)


if __name__ == "__main__":
    main()
