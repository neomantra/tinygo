#!/usr/bin/env python3
import argparse
import concurrent.futures
import csv
import hashlib
import json
import os
from pathlib import Path
import platform
import random
import re
import statistics
import subprocess
import time


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--dev-source", type=Path, required=True)
    parser.add_argument("--sweep-source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--artifacts", type=Path, required=True)
    parser.add_argument("--benchtime", default="100ms")
    parser.add_argument("--repetitions", type=int, default=3)
    parser.add_argument("--jobs", type=int, default=2)
    args = parser.parse_args()
    for key in ["root", "dev_source", "sweep_source", "output", "artifacts"]:
        setattr(args, key, getattr(args, key).resolve())
    if (args.output / "manifest.json").exists():
        parser.error("output already contains a sweep; choose a new output directory")
    args.output.mkdir(parents=True, exist_ok=True)
    args.artifacts.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, TINYGOROOT=str(args.root))

    def announce(message):
        print(time.strftime("%H:%M:%S"), message, flush=True)

    def run(command, logfile, cwd=None, timeout=600):
        logfile = args.output / logfile
        started = time.monotonic()
        with logfile.open("w") as out:
            try:
                proc = subprocess.run(command, cwd=cwd or args.root, env=env,
                                      stdout=out, stderr=subprocess.STDOUT, timeout=timeout)
                status = proc.returncode
            except subprocess.TimeoutExpired:
                out.write("\nTIMEOUT\n")
                status = 124
        return {"command": list(map(str, command)), "status": status,
                "seconds": time.monotonic() - started, "log": logfile.name}

    def capture(command):
        return subprocess.check_output(command, env=env, text=True).strip()

    metadata = {
        "started": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
        "host": platform.platform(),
        "cpu": capture(["sysctl", "-n", "machdep.cpu.brand_string"]) if platform.system() == "Darwin" else platform.processor(),
        "benchtime": args.benchtime, "repetitions": args.repetitions,
        "stack_size": "64kb", "threshold_scope": "all architectures in experimental compilers",
        "source_commits": {label: capture(["git", "-C", str(root), "rev-parse", "HEAD"])
                           for label, root in [("dev", args.dev_source), ("markers", args.root),
                                               ("sweep", args.sweep_source)]},
        "versions": {name: capture(command) for name, command in {
            "go": ["go", "version"], "llvm": ["llvm-config", "--version"],
            "lld": ["ld.lld", "--version"], "node": ["node", "--version"],
            "wasmtime": ["wasmtime", "--version"], "wasm_opt": ["wasm-opt", "--version"],
            "wasm_tools": ["wasm-tools", "--version"],
        }.items()},
        "workload_sha256": {p.name: digest(p) for p in sorted((args.root / "tests/aggregatebench").glob("*.go"))},
        "compilers": {}, "builds": [], "runs": [],
    }

    def save():
        (args.output / "manifest.json").write_text(json.dumps(metadata, indent=2) + "\n")

    variants = ["dev", "markers", "16", "32", "64", "128", "256", "1024"]
    function_file = args.sweep_source / "compiler/func.go"
    original = function_file.read_text()
    pattern = r"func \(c \*compilerContext\) maxDirectAggregateValues\(\) uint64 \{.*?\n\}"
    if len(re.findall(pattern, original, re.S)) != 1:
        raise RuntimeError("sweep source must contain the aggregate threshold method")
    try:
        for variant in variants:
            source = args.dev_source if variant == "dev" else args.root if variant == "markers" else args.sweep_source
            if variant.isdigit():
                modified = re.sub(pattern, "func (c *compilerContext) maxDirectAggregateValues() uint64 {\n"
                                  f"\treturn {variant}\n}}", original, flags=re.S)
                function_file.write_text(modified)
                (args.output / f"compiler-{variant}.patch").write_text(capture(
                    ["git", "-C", str(source), "diff", "--", "compiler/func.go"]) + "\n")
            binary = args.artifacts / f"tinygo-{variant}"
            announce(f"build compiler {variant}")
            result = run(["go", "build", "-tags=llvm22", "-o", str(binary), "."],
                         f"compiler-{variant}.log", cwd=source)
            if result["status"]:
                raise RuntimeError(f"compiler {variant} failed: {result}")
            metadata["compilers"][variant] = {**result, "path": str(binary), "sha256": digest(binary),
                                             "threshold": int(variant) if variant.isdigit() else 1024,
                                             "markers": variant != "dev"}
            save()
    finally:
        function_file.write_text(original)

    def compile_workload(spec):
        variant, target, opt = spec
        key = f"{target}-opt{opt}-{variant}"
        binary = args.artifacts / (key + (".wasm" if target != "host" else ".bin"))
        command = [metadata["compilers"][variant]["path"], "test", "-c", f"-opt={opt}",
                   "-stack-size=64kb", "-o", str(binary)]
        if target != "host":
            command.append(f"-target={target}")
        if target == "wasm":
            command.extend(["-bench=.", "-benchmem", f"-benchtime={args.benchtime}", "-v"])
        command.append("./tests/aggregatebench")
        result = {**run(command, key + "-build.log"), "key": key, "variant": variant,
                  "target": target, "opt": opt, "binary": str(binary)}
        if not result["status"]:
            result.update(binary_bytes=binary.stat().st_size, sha256=digest(binary))
        announce(f"compiled {key}: {result['status']}")
        return result

    specs = [(v, t, o) for v in variants for t in ["host", "wasm", "wasip1"] for o in ["2", "z"]]
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as pool:
        for result in pool.map(compile_workload, specs):
            metadata["builds"].append(result)
            save()

    def runner(build):
        if build["target"] == "wasm":
            return ["node", str(args.root / "targets/wasm_exec_node.js"), build["binary"]]
        if build["target"] == "wasip1":
            return ["wasmtime", "run", build["binary"]]
        return [build["binary"]]

    builds = [b for b in metadata["builds"] if not b["status"]]
    eligible = []
    for build in builds:
        result = run(runner(build) + ["-test.v"], build["key"] + "-correctness.log", timeout=120)
        result.update(key=build["key"], phase="correctness")
        metadata["runs"].append(result)
        if not result["status"] and "PASS" in (args.output / result["log"]).read_text():
            eligible.append(build)
        announce(f"correctness {build['key']}: {result['status']}")
        save()

    for repeat in range(args.repetitions):
        order = eligible.copy()
        random.Random(5778 + repeat).shuffle(order)
        for build in order:
            announce(f"measure {build['key']} repetition {repeat + 1}")
            result = run(runner(build) + ["-test.run=^$", "-test.bench=.", "-test.benchmem",
                         f"-test.benchtime={args.benchtime}"],
                         f"{build['key']}-run{repeat + 1}.log", timeout=180)
            result.update(key=build["key"], phase="benchmark", repeat=repeat + 1)
            metadata["runs"].append(result)
            save()

    rows = []
    by_key = {b["key"]: b for b in builds}
    for result in metadata["runs"]:
        if result["phase"] != "benchmark" or result["status"]:
            continue
        build = by_key[result["key"]]
        count = 0
        for line in (args.output / result["log"]).read_text().splitlines():
            match = re.match(r"^(Benchmark\S+)\s+(\d+)\s+(.*)$", line)
            if not match:
                continue
            name, iterations, metrics = match.groups()
            metrics = dict((unit, float(value)) for value, unit in re.findall(r"([\d.eE+-]+)\s+(\S+)", metrics))
            if not {"ns/op", "B/op", "allocs/op"} <= metrics.keys() or metrics["ns/op"] <= 0:
                announce(f"INVALID benchmark: {result['log']}: {line}")
                continue
            rows.append({"target": build["target"], "opt": build["opt"], "variant": build["variant"],
                         "benchmark": name, "repeat": result["repeat"], "iterations": int(iterations),
                         "ns_per_op": metrics["ns/op"], "bytes_per_op": int(metrics["B/op"]),
                         "allocs_per_op": int(metrics["allocs/op"])})
            count += 1
        result["parsed_benchmarks"] = count
        if count != 41:
            announce(f"WARNING {result['log']}: parsed {count} of 41 benchmarks")
    if rows:
        with (args.output / "measurements.csv").open("w") as out:
            writer = csv.DictWriter(out, fieldnames=list(rows[0]))
            writer.writeheader()
            writer.writerows(rows)
        groups = {}
        for row in rows:
            key = tuple(row[k] for k in ["target", "opt", "variant", "benchmark"])
            groups.setdefault(key, []).append(row)
        summaries = []
        for key, samples in sorted(groups.items()):
            summaries.append({**dict(zip(["target", "opt", "variant", "benchmark"], key)),
                              "samples": len(samples),
                              "median_ns": statistics.median(s["ns_per_op"] for s in samples),
                              "min_ns": min(s["ns_per_op"] for s in samples),
                              "max_ns": max(s["ns_per_op"] for s in samples),
                              "median_bytes": statistics.median(s["bytes_per_op"] for s in samples),
                              "median_allocs": statistics.median(s["allocs_per_op"] for s in samples)})
        with (args.output / "summary.csv").open("w") as out:
            writer = csv.DictWriter(out, fieldnames=list(summaries[0]))
            writer.writeheader()
            writer.writerows(summaries)
    metadata["finished"] = time.strftime("%Y-%m-%dT%H:%M:%S%z")
    save()
    announce(f"finished: {len(rows)} measurements")


if __name__ == "__main__":
    main()
