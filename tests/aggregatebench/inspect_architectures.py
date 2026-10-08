#!/usr/bin/env python3
import argparse
import csv
import json
from pathlib import Path
import re
import subprocess


def entry_reservation(body, target):
    total, evidence, unresolved = 0, [], False
    for line in body.splitlines():
        match = re.match(r'\s*[0-9a-f]+:\s+(.*)', line)
        if not match:
            continue
        instruction = match[1].strip()
        opcode = instruction.split()[0].removesuffix('.w').removesuffix('.n')
        amount = None
        if target == 'cortex-m-qemu':
            if opcode in ['b', 'bl', 'blx', 'bx', 'cbz', 'cbnz', 'pop'] or re.fullmatch(r'b(eq|ne|lt|le|gt|ge|hi|ls|hs|lo|mi|pl|vs|vc)', opcode):
                break
            push = re.match(r'(v?push)(?:\.w)?\s+\{([^}]+)\}', instruction)
            subtract = re.match(r'sub(?:\.w)?\s+sp, (?:sp, )?#(0x[0-9a-f]+|\d+)', instruction)
            if push:
                registers = 0
                for item in push[2].split(','):
                    bounds = re.fullmatch(r'\s*[rd](\d+)-[rd](\d+)\s*', item)
                    registers += int(bounds[2]) - int(bounds[1]) + 1 if bounds else 1
                amount = registers * (8 if push[1] == 'vpush' else 4)
            elif subtract:
                amount = int(subtract[1], 0)
            elif re.match(r'(sub|add|mov|bic)(?:\.w)?\s+sp[,\s]', instruction):
                unresolved = True
        elif target == 'riscv-qemu':
            if opcode in ['j', 'jr', 'jal', 'jalr', 'call', 'tail', 'ret'] or opcode.startswith(('beq', 'bne', 'blt', 'bge', 'ble', 'bgt')):
                break
            subtract = re.match(r'addi\s+sp, sp, -(0x[0-9a-f]+|\d+)', instruction)
            if subtract:
                amount = int(subtract[1], 0)
            elif re.match(r'(sub|add|addi|mv|andi)\s+sp,', instruction):
                unresolved = True
        else:
            if opcode.startswith(('j', 'call', 'ret', 'pop')):
                break
            subtract = re.match(r'subq\s+\$(0x[0-9a-f]+|\d+), %rsp', instruction)
            if opcode == 'pushq':
                amount = 8
            elif subtract:
                amount = int(subtract[1], 0)
            elif re.search(r', %rsp$', instruction):
                unresolved = True
        if amount is not None:
            total += amount
            evidence.append(instruction)
    return total, '; '.join(evidence), unresolved


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--results', type=Path, required=True)
    parser.add_argument('--llvm-bin', type=Path, required=True)
    args = parser.parse_args()
    manifest = json.loads((args.results / 'manifest.json').read_text())
    frames, sizes = [], []
    for build in manifest['builds']:
        if build['status']:
            continue
        common = {k: build[k] for k in ['target', 'opt', 'variant', 'selection']}
        size_text = (args.results / build['size']['log']).read_text()
        section = '__text' if build['target'] == 'darwin-amd64' else '.text'
        match = re.search(r'^' + re.escape(section) + r'\s+(\d+)\s+', size_text, re.M)
        if not match:
            raise ValueError(f'no {section} section for {build["key"]}')
        sizes.append({**common, 'binary_bytes': build['binary_bytes'],
                      'code_bytes': int(match[1]), 'code_metric': section})
        asm = subprocess.check_output([str(args.llvm_bin / 'llvm-objdump'), '-d',
                                       '--no-show-raw-insn', build['binary']], text=True)
        parts = re.split(r'^[0-9a-f]+ <(.*)>:\n', asm, flags=re.M)
        for index in range(1, len(parts), 2):
            name, body = parts[index:index + 2]
            if '/tests/aggregatebench.' not in name:
                continue
            size, evidence, unresolved = entry_reservation(body, build['target'])
            frames.append({**common, 'function': name, 'observed_bytes': size,
                           'metric': 'immediate entry reservation before first branch or call',
                           'unresolved_adjustment': unresolved, 'evidence': evidence})
    for filename, rows in [('sizes.csv', sizes), ('frames.csv', frames)]:
        if rows:
            with (args.results / filename).open('w', newline='') as out:
                writer = csv.DictWriter(out, fieldnames=list(rows[0]), lineterminator='\n')
                writer.writeheader()
                writer.writerows(rows)
    print(f'{len(sizes)} code sizes, {len(frames)} partial entry reservations')


if __name__ == '__main__':
    main()
