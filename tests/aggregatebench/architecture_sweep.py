#!/usr/bin/env python3
import argparse
import concurrent.futures
import csv
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import time


VARIANTS = ['dev', 'markers', '16', '32', '64', '128', '256', '1024']
SELECTIONS = ['16', '32', '64', '65', '128', '256', '512', '1024', '1025', 'shapes']


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--root', type=Path, default=Path.cwd())
    parser.add_argument('--compiler-manifest', type=Path, required=True)
    parser.add_argument('--compilers', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--jobs', type=int, default=2)
    parser.add_argument('--targets', nargs='+', choices=['cortex-m-qemu', 'riscv-qemu'],
                        default=['cortex-m-qemu', 'riscv-qemu'])
    args = parser.parse_args()
    for name in ['root', 'compiler_manifest', 'compilers', 'output', 'artifacts']:
        setattr(args, name, getattr(args, name).resolve())
    if args.output.exists():
        parser.error('choose a new output directory')
    source = json.loads(args.compiler_manifest.read_text())
    compilers = {}
    for variant in VARIANTS:
        path = args.compilers / f'tinygo-{variant}'
        if digest(path) != source['compilers'][variant]['sha256']:
            parser.error(f'compiler hash mismatch: {variant}')
        compilers[variant] = {**source['compilers'][variant], 'path': str(path)}
    args.output.mkdir(parents=True)
    args.artifacts.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, TINYGOROOT=str(args.root))

    def capture(command):
        return subprocess.check_output(command, env=env, cwd=args.root, text=True).strip()

    def run(command, logfile, timeout):
        started = time.monotonic()
        with (args.output / logfile).open('w') as out:
            try:
                status = subprocess.run(command, env=env, cwd=args.root, stdout=out,
                                        stderr=subprocess.STDOUT, timeout=timeout).returncode
            except subprocess.TimeoutExpired:
                out.write('\nTIMEOUT\n')
                status = 124
        return {'command': list(map(str, command)), 'status': status,
                'seconds': time.monotonic() - started, 'log': logfile}

    metadata = {
        'started': time.strftime('%Y-%m-%dT%H:%M:%S%z'),
        'host': platform.platform(), 'workload_commit': capture(['git', 'rev-parse', 'HEAD']),
        'compiler_manifest': str(args.compiler_manifest),
        'compiler_manifest_sha256': digest(args.compiler_manifest),
        'source_commits': source['source_commits'], 'compilers': compilers,
        'workload_sha256': {p.name: digest(p) for p in sorted((args.root / 'tests/aggregatebench').glob('*.go'))},
        'script_sha256': digest(__file__),
        'benchtime': '100x', 'purpose': 'emulated correctness and allocation coverage, not hardware timing',
        'stack_sizes': {'cortex-m-qemu': '16kb', 'riscv-qemu': '64kb'},
        'versions': {name: capture(command) for name, command in {
            'qemu_arm': ['qemu-system-arm', '--version'],
            'qemu_riscv': ['qemu-system-riscv32', '--version'],
            'llvm': ['llvm-config', '--version'], 'lld': ['ld.lld', '--version'],
            'go': ['go', 'version'],
        }.items()}, 'builds': [], 'runs': [],
    }

    def save():
        (args.output / 'manifest.json').write_text(json.dumps(metadata, indent=2) + '\n')

    def build(spec):
        target, variant, opt, selection = spec
        key = f'{target}-opt{opt}-{variant}-{selection}'
        binary = args.artifacts / (key + '.elf')
        command = [compilers[variant]['path'], 'test', '-c', f'-target={target}', f'-opt={opt}',
                   f'-stack-size={metadata["stack_sizes"][target]}', '-bench=.*', '-benchmem',
                   '-benchtime=100x', '-v', '-o', str(binary)]
        if selection != 'all':
            command.append(f'-tags=aggregatebench_{selection},aggregatebench_flat')
        command.append('./tests/aggregatebench')
        result = {**run(command, key + '-build.log', 300), 'key': key, 'target': target,
                  'variant': variant, 'opt': opt, 'selection': selection, 'binary': str(binary)}
        if result['status'] == 0:
            result.update(sha256=digest(binary), binary_bytes=binary.stat().st_size)
            result['size'] = run(['llvm-size', '-A', str(binary)], key + '-size.log', 30)
        return result

    specs = [(t, v, o, s) for t in args.targets for v in VARIANTS for o in ['2', 'z']
             for s in (SELECTIONS if t == 'cortex-m-qemu' else ['all'])]
    save()
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as pool:
        for future in concurrent.futures.as_completed([pool.submit(build, s) for s in specs]):
            result = future.result()
            metadata['builds'].append(result)
            save()
            print(f'build {len(metadata["builds"])}/{len(specs)} {result["key"]}: {result["status"]}', flush=True)

    rows = []
    for build in sorted(metadata['builds'], key=lambda b: b['key']):
        if build['status']:
            continue
        if build['target'] == 'cortex-m-qemu':
            command = ['qemu-system-arm', '-machine', 'lm3s6965evb', '-semihosting',
                       '-nographic', '-kernel', build['binary']]
        else:
            command = ['qemu-system-riscv32', '-machine', 'virt,aclint=on', '-smp', '4',
                       '-nographic', '-bios', 'none', '-device', 'virtio-rng-device', '-kernel', build['binary']]
        result = {**run(command, build['key'] + '-run.log', 60), 'key': build['key']}
        log = (args.output / result['log']).read_text(errors='replace')
        parsed = []
        for line in log.splitlines():
            match = re.match(r'^(Benchmark\S+)\s+(\d+)\s+(.*)$', line)
            if not match:
                continue
            name, iterations, metrics = match.groups()
            name = re.sub(r'^BenchmarkBytes(\d+)(Update|Read|Serial|Live)$', r'BenchmarkBytes/\1/\2', name)
            metrics = {unit: float(value) for value, unit in re.findall(r'([\d.eE+-]+)\s+(\S+)', metrics)}
            if int(iterations) == 0 or not {'B/op', 'allocs/op'} <= metrics.keys():
                continue
            parsed.append({**{k: build[k] for k in ['target', 'variant', 'opt', 'selection']},
                           'benchmark': name, 'iterations': int(iterations),
                           'emulated_ns_per_op': metrics.get('ns/op'),
                           'bytes_per_op': metrics['B/op'], 'allocs_per_op': metrics['allocs/op']})
        expected = 41 if build['selection'] == 'all' else 5 if build['selection'] == 'shapes' else 4
        result.update(parsed_benchmarks=len(parsed), expected_benchmarks=expected,
                      passed=result['status'] == 0 and bool(re.search(r'^PASS\s*$', log, re.M))
                      and len(parsed) == expected and all(r['iterations'] == 100 for r in parsed))
        if result['passed']:
            rows.extend(parsed)
        metadata['runs'].append(result)
        save()
        print(f'run {build["key"]}: {result["status"]}, passed={result["passed"]}, cases={len(parsed)}', flush=True)
    if rows:
        with (args.output / 'measurements.csv').open('w', newline='') as out:
            writer = csv.DictWriter(out, fieldnames=list(rows[0]), lineterminator='\n')
            writer.writeheader()
            writer.writerows(rows)
    metadata['finished'] = time.strftime('%Y-%m-%dT%H:%M:%S%z')
    save()
    failures = sum(b['status'] != 0 for b in metadata['builds']) + sum(not r['passed'] for r in metadata['runs'])
    print(f'finished: {len(rows)} measurements, {failures} failed configurations', flush=True)
    return bool(failures)


if __name__ == '__main__':
    raise SystemExit(main())
