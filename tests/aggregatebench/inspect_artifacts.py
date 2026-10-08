#!/usr/bin/env python3
import argparse
import csv
import json
from pathlib import Path
import re
import subprocess


def wasm_code_bytes(path):
    data = path.read_bytes()
    offset = 8
    while offset < len(data):
        section = data[offset]
        offset += 1
        size, shift = 0, 0
        while True:
            byte = data[offset]
            offset += 1
            size |= (byte & 127) << shift
            if byte < 128:
                break
            shift += 7
        if section == 10:
            return size
        offset += size
    raise ValueError("no wasm code section")


def wasm_frames(path):
    wat = subprocess.check_output(["wasm-tools", "print", str(path)], text=True)
    parts = re.split(r'^  \(func (\$(?:"[^"\n]*"|[^\s()]+))', wat, flags=re.M)
    frames = []
    for index in range(1, len(parts), 2):
        name, body = parts[index:index + 2]
        if "/tests/aggregatebench." not in name:
            continue
        matches = re.findall(r"global.get \$__stack_pointer\s+(?:local.tee \S+\s+)?i32.const (-?\d+)\s+i32.(sub|add)", body)
        sizes = [int(value) if op == "sub" else -int(value) for value, op in matches]
        frames.append((name, max([0] + sizes), "wasm_constant_stack_decrement",
                       "; ".join(f"{op} {value}" for value, op in matches)))
    return frames


def arm64_frames(path, objdump):
    asm = subprocess.check_output([str(objdump), "-d", "--no-show-raw-insn", str(path)], text=True)
    parts = re.split(r"^[0-9a-f]+ <(.*)>:\n", asm, flags=re.M)
    frames = []
    for index in range(1, len(parts), 2):
        name, body = parts[index:index + 2]
        if "/tests/aggregatebench." not in name:
            continue
        size, evidence = 0, []
        for line in body.splitlines():
            match = re.match(r"[0-9a-f]+:\s+(.*)", line)
            if not match:
                continue
            instruction = match[1]
            opcode = instruction.split()[0]
            if opcode in ["b", "bl", "blr", "br", "ret", "cbz", "cbnz", "tbz", "tbnz"] or opcode.startswith("b."):
                break
            subtract = re.match(r"sub\s+sp, sp, #(0x[0-9a-f]+|\d+)(?:, lsl #(\d+))?", instruction)
            preindex = re.search(r"\[sp, #-(0x[0-9a-f]+|\d+)\]!", instruction)
            if subtract:
                size += int(subtract[1], 0) << int(subtract[2] or 0)
                evidence.append(instruction)
            elif preindex:
                size += int(preindex[1], 0)
                evidence.append(instruction)
            elif re.match(r"add\s+sp,", instruction) or re.search(r"\[sp\],", instruction):
                break
        frames.append((name, size, "arm64_entry_stack_reservation", "; ".join(evidence)))
    return frames


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--results", type=Path, required=True)
    parser.add_argument("--llvm-bin", type=Path, required=True)
    args = parser.parse_args()
    manifest = json.loads((args.results / "manifest.json").read_text())
    frames, sizes = [], []
    for build in manifest["builds"]:
        if build["status"]:
            continue
        binary = Path(build["binary"])
        common = {k: build[k] for k in ["target", "opt", "variant"]}
        if build["target"] == "host":
            output = subprocess.check_output([str(args.llvm_bin / "llvm-size"), "-m", str(binary)], text=True)
            code = int(re.search(r"Section __text: (\d+)", output)[1])
            metric = "Mach-O __text section"
            observations = arm64_frames(binary, args.llvm_bin / "llvm-objdump")
        else:
            code = wasm_code_bytes(binary)
            metric = "wasm code section payload"
            observations = wasm_frames(binary)
        sizes.append({**common, "binary_bytes": binary.stat().st_size, "code_bytes": code, "code_metric": metric})
        for name, size, metric, evidence in observations:
            frames.append({**common, "function": name, "observed_bytes": size, "metric": metric, "evidence": evidence})
    for name, rows in [("sizes.csv", sizes), ("frames.csv", frames)]:
        with (args.results / name).open("w") as out:
            writer = csv.DictWriter(out, fieldnames=list(rows[0]))
            writer.writeheader()
            writer.writerows(rows)
    print(f"Recorded {len(sizes)} binary sizes and {len(frames)} function observations")


if __name__ == "__main__":
    main()
