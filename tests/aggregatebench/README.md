# Aggregate passing workloads

This is local testing work on `nm-aggregate-workloads`, not intended for a PR.
Keep generated compilers and executables in the ignored `artifacts/` directory
and compiler checkouts in the ignored `worktrees/` directory beside this file.
Commit workload sources, sweep scripts, manifests, logs, and reports.

Start with the [integrated report](REPORT.md) for findings across all architectures.

The [architecture extension report](results/2026-10-08-architectures/REPORT.md)
adds Cortex-M and RISC-V execution plus x86-64 code inspection.

The [2026-10-08 sweep report](results/2026-10-08/REPORT.md) includes runtime,
allocation, code-size, and stack observations for native ARM64 and wasm.

These deterministic workloads measure TinyGo's internal aggregate calling
convention and promoted allocation lifetimes. They use only the standard
library and run with ordinary Go for a correctness reference.

| Workload | What it exercises |
| --- | --- |
| Bytes/16 through Bytes/1025 | Scalar leaf counts around candidate limits, including 64/65 and 1024/1025 |
| Bytes/*/Read | By-value input with sparse, runtime-selected reads |
| Bytes/*/Update | Repeated by-value updates and reads of one evolving value |
| Bytes/*/Serial | Four independent edits and reads with sequential temporary lifetimes |
| Bytes/*/Live | The same edits and reads with four results retained before consumption |
| Words64 | 64 wide integer leaves, compared with 64-byte and 512-byte arrays |
| Scene | Nested transforms with 64 float32 leaves and one integer field |
| Record | Twelve string/slice records and an ID, totaling 73 scalar leaves |
| RecordInterface | Aggregate arguments and results through two interface implementations |
| SequentialBlocks | Eight promoted 512-byte allocations, independent of the aggregate threshold |

Call boundaries use `//go:noinline` to keep aggregate passing observable. Seeds
vary per iteration and results are logged as a checksum after timing
stops. A write to an unread package global is not sufficient under TinyGo's
whole-program optimization. These are controlled
workloads, not substitutes for application measurements such as Bubble Tea.
The tests check results and caller value preservation, including byte wraparound,
wide integer values, nested fields, and interface dispatch.

Run from the repository root with a TinyGo binary built from the version under
test. For example, set `TINYGO` to an absolute path to that binary.

```sh
go test ./tests/aggregatebench
"$TINYGO" test -target=wasm -opt=2 ./tests/aggregatebench
"$TINYGO" test -target=wasm -opt=2 -run='^$' -bench=. -benchmem -benchtime=1s ./tests/aggregatebench
```

Omit `-target` for the host, or select `wasip1`, `cortex-m-qemu`, or `riscv-qemu`.
The target's emulator must be installed. First check correctness on every target.
Use `-benchtime=1x` for a benchmark smoke test; timings from that run are not
performance measurements. Use repeated longer runs for comparisons.

The full test binary can exceed a microcontroller's flash capacity. Compile one
array size at a time using exactly one of `aggregatebench_16`, `aggregatebench_32`,
`aggregatebench_64`, `aggregatebench_65`, `aggregatebench_128`, `aggregatebench_256`,
`aggregatebench_512`, `aggregatebench_1024`, or `aggregatebench_1025`. Use
`aggregatebench_shapes` for the word, scene, record, interface, and allocation
workloads. Without these tags, all workloads are included.

```sh
"$TINYGO" test -target=cortex-m-qemu -tags=aggregatebench_65 -opt=2 -bench='.*' -benchmem ./tests/aggregatebench
```

Use the same selection tag on all compared builds. Run every selection to cover
the full suite. Record any selection that still exceeds a target's capacity.
Array-only builds explicitly skip the shape and allocation tests and benchmarks.

The largest cases deliberately put pressure on the stack. If a target needs
`-stack-size`, use the same value for every compiler variant and record it.
Keep the larger cases when comparing variants; do not silently omit failures.
The stack-promotion limit is also controlled by the stack setting. It is 256
bytes below a 16 KiB stack and 1024 bytes otherwise; see
[`Config.MaxStackAlloc`](../../compileopts/config.go). The 1025-byte cases
therefore exercise heap allocation as well as aggregate passing. Check B/op
alongside allocs/op because allocation counter support depends on the GC.

For binary and frame inspection, compile the test executable without running it.
The resulting binary contains the whole suite and its testing harness.

```sh
"$TINYGO" test -c -target=wasm -opt=2 -o /tmp/aggregatebench.wasm ./tests/aggregatebench
wasm-tools print /tmp/aggregatebench.wasm > /tmp/aggregatebench.wat
```

Inspect workload functions separately from the harness. A benchmark filter
selects what runs; it does not remove the other workloads from the binary.
Summed frame sizes are not peak stack usage. The Live workload requests longer
lifetimes in source, but the optimizer may reschedule operations. Check generated
code before drawing conclusions about stack overlap.

For a threshold sweep, compare 16, 32, 64, 128, 256, and 1024 at both `-opt=2`
and `-opt=z`. Keep lifetime markers enabled for the main sweep and retain an
unmodified dev baseline. Compare marker-disabled variants separately if measuring
their interaction with the threshold. The current threshold branch changes only
wasm; testing native thresholds requires an explicit experimental compiler change.

Record the compiler commit, actual threshold, marker setting, target and CPU,
Go/LLVM/linker versions, optimization level, stack setting, and execution engine.
For wasm, also record wasm-opt and engine versions. Collect ns/op, B/op,
allocs/op, binary size, and workload frame sizes. Compare timings within the same
architecture and execution environment. QEMU timings are useful for smoke tests,
not evidence of performance on physical microcontrollers.

## Running the sweep

`sweep.py` builds the compiler variants, compiles the full suite, checks
correctness, then runs three randomized rounds of benchmarks. Compilation
finishes before timed runs begin. It sets a 64 KiB stack for every configuration.
It currently runs the host, js/wasm under Node, and wasip1 under Wasmtime.

Prepare three source trees: unmodified dev, the lifetime-marker branch with these
workloads, and a temporary checkout combining the threshold and lifetime patches.
The script temporarily changes `maxDirectAggregateValues()` in the combined
checkout to return each candidate limit on all architectures, then restores it.
The exported ABI limit stays unchanged. The working directory and TINYGOROOT
for workload builds point to the workload checkout.

```sh
python3 tests/aggregatebench/sweep.py \
  --root "$PWD" \
  --dev-source /tmp/tinygo-dev \
  --sweep-source /tmp/tinygo-combined \
  --output tests/aggregatebench/results/my-run \
  --artifacts /tmp/aggregatebench-binaries
```

Use LLVM and LLD 22 on PATH, including `llvm-config`, and Go with CGO enabled.
The manifest records versions, source commits, patches, source and binary hashes,
commands, and exit statuses. Raw logs, individual measurements, and median/range
summaries remain in the output directory. Each sample targets 100 ms by default;
use `--benchtime` and `--repetitions` for longer runs. js/wasm test flags are baked
into the binary because that runtime does not read Node's command-line arguments.
Consequently its correctness pass also executes benchmarks, and its measured
runs also execute correctness tests outside the benchmark timers.

After benchmarking, collect binary sizes and stack reservations separately:

```sh
python3 tests/aggregatebench/inspect_artifacts.py \
  --results tests/aggregatebench/results/my-run \
  --llvm-bin /path/to/llvm/bin
```

Wasm observations recognize constant decrements of `__stack_pointer`. Native
ARM64 observations sum immediate entry stack reservations before the first branch
or call; these are not complete stack bounds. A zero observation does not prove
zero stack usage. The native frame inspector supports ARM64 Mach-O only.

## Embedded architecture sweep

`architecture_sweep.py` reuses compilers whose hashes match a prior sweep manifest.
It runs all eight variants at opt 2 and z on Cortex-M3 and RV32 QEMU targets.
Cortex-M uses ten selections with flat benchmark entry points to fit 64 KiB RAM.
The benchmark loops are shared with the nested host suite. Initialization is
outside each measured loop. RISC-V runs the complete suite in one binary.

```sh
python3 tests/aggregatebench/architecture_sweep.py \
  --compiler-manifest tests/aggregatebench/results/2026-10-08/manifest.json \
  --compilers tests/aggregatebench/artifacts \
  --output tests/aggregatebench/results/my-architecture-run \
  --artifacts tests/aggregatebench/artifacts/my-architecture-run
```

Use LLVM/LLD 22 on PATH and install `qemu-system-arm` and `qemu-system-riscv32`.
Tests and 100-iteration benchmarks run together with baked-in arguments.
Bare-metal matching requires `-bench='.*'`, not `-bench=.`. Cortex-M uses a
16 KiB stack and RISC-V uses 64 KiB, both retaining the 1024-byte stack-promotion
limit. Cortex-M's clock can report no elapsed time, so missing timings are kept
empty. QEMU timings are not hardware performance measurements. Allocation results
are retained only when correctness passes and every selected benchmark completes
100 iterations. Build failures, timeouts, and incomplete runs remain in the manifest.

Pass `--targets darwin-amd64` to the architecture runner for an x86-64 build-only
sweep. It records execution as skipped. Use a separate output directory.

Inspect code sections and partial entry stack reservations with:

```sh
python3 tests/aggregatebench/inspect_architectures.py \
  --results tests/aggregatebench/results/my-architecture-run \
  --llvm-bin /opt/homebrew/opt/llvm@22/bin
```

These stack observations stop at the first branch or call and recognize immediate
adjustments and pushes. They do not include complete call chains, x86 return
addresses, red-zone use, or every outlined/register-sized adjustment. Inspect
`evidence` and `unresolved_adjustment` before interpreting a row. Zero does not
establish zero stack use.

## Stable local layout

- `worktrees/dev` pins upstream dev at `f6d269f7`.
- `worktrees/combined` uses local branch `nm-aggregate-sweep-compiler` at `115ef5e1`.
- `artifacts/tinygo-*` contains the eight hash-verified compiler binaries.
- `artifacts/2026-10-08` contains the original ARM64 and Wasm executables.
- `artifacts/2026-10-08-architectures` contains Cortex-M and RISC-V executables.
- `artifacts/2026-10-08-amd64` contains x86-64 executables.

The artifact and worktree directories are ignored by Git. Source commits and
results remain committed; binaries can be regenerated. Historical manifests retain
the paths actually used when those measurements were taken. To inspect relocated
original executables, pass `--artifacts tests/aggregatebench/artifacts/2026-10-08`
to `inspect_artifacts.py`.

For a new full host/Wasm sweep, use `--dev-source tests/aggregatebench/worktrees/dev`
and `--sweep-source tests/aggregatebench/worktrees/combined` with `sweep.py`.
Choose fresh output and artifact directories. This branch contains lifetime markers;
it remains the marker-enabled workload source. Do not run two compiler rebuild
sweeps against the same combined checkout at once.
