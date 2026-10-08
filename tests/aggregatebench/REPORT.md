# Aggregate passing and allocation lifetime results across architectures

Measurements from 8 October 2026. Local testing branch `nm-aggregate-workloads`.

## Findings

Lifetime markers and the aggregate-passing threshold solve different problems.
Lifetime markers let LLVM reuse storage after a promoted allocation is no longer
needed. Lowering the threshold makes more values use indirect argument and result
buffers, reducing the cost of passing large aggregates directly.

The results support keeping the lifetime-marker change separate and moving it
forward. In the sequential-allocation workload, markers reduce observed stack
reservations from about 4 KiB to about 0.5 KiB on every inspected architecture.
That improvement does not require lowering the aggregate threshold.

Lower thresholds offer substantial code-size and runtime benefits for some
workloads, but introduce heap allocations and slow down repeated value updates.
The allocation pattern reproduces on ARM64, Wasm, Cortex-M3, and RV32. Threshold
64 is therefore not established as a general default. Thresholds 128 and 256
avoid the Record regression but still add allocations for larger array updates.

These are controlled synthetic workloads. They extend the investigation motivated
by Bubble Tea, but do not replace application measurements or establish a minimum
safe application stack size.

## What was compared

The direct aggregate threshold counts scalar leaves, not bytes. For example,
64 `uint64` elements occupy 512 bytes but count as 64 leaves. A 65-byte array
crosses threshold 64; the Record workload has 73 leaves and occupies 292 bytes
on the tested 32-bit targets.

Eight compiler variants were tested at both `-opt=2` and `-opt=z`:

| Variant | Purpose |
| --- | --- |
| Upstream dev `f6d269f7` | Unmodified baseline, threshold 1024 |
| Lifetime markers `1901a9c4` | Isolate promoted-allocation lifetime markers |
| Thresholds 16, 32, 64, 128, 256, 1024 | Compare thresholds with markers and the same ABI-wrapper changes |

The threshold compilers combine `e361fafc` and `1901a9c4`, retained as
`115ef5e1` on local branch `nm-aggregate-sweep-compiler`. Each candidate overrides
the threshold on **all architectures**. This experiment is broader than the
proposed Wasm-only threshold change; exported ABI limits remain unchanged.
Threshold comparisons use the experimental 1024 variant as their baseline,
so both sides include the same marker and wrapper changes.

The 41 workloads cover nine byte-array sizes from 16 to 1025 with Read, Update,
Serial, and Live patterns, plus Words64, Scene, Record, RecordInterface, and
SequentialBlocks. Correctness tests check results and preservation of caller
values. No-inline boundaries retain aggregate calls, and checksums are consumed
after timing stops so LLVM cannot discard the loops.

## Coverage and strength of evidence

| Target | Builds attempted | Builds succeeded | Configurations completing correctness and benchmarks | Evidence |
| --- | ---: | ---: | ---: | --- |
| Native Darwin ARM64 | 16 | 16 | 16 | Repeated timings, allocations, code, stack observations |
| Wasm under Node | 16 | 16 | 16 | Repeated timings, allocations, code, stack observations |
| Wasm under Wasmtime | 16 | 16 | 16 | Repeated timings, allocations, code, stack observations |
| Cortex-M3 under QEMU | 160 | 144 | 123 | Partial workload coverage, allocations, code, stack observations |
| RV32 under QEMU | 16 | 16 | 16 | Full workload coverage, allocations, code, stack observations |
| Darwin x86-64 | 16 | 16 | Not executed | Cross-compiled code and stack observations only |

Cortex-M uses ten separate workload selections per compiler/optimization to fit
its emulated board. Its build count is not directly comparable to the full-suite
build counts for other targets. Across the campaigns, 240 builds were attempted,
224 linked, and 187 configurations completed execution. The 16 x86-64 binaries
were not executed; 21 linked Cortex-M binaries failed during execution.

The ARM64/Wasm campaign produced **5,904 timed measurements** across 144 runs,
with three repetitions per configuration. The embedded campaign produced
**1,164 accepted allocation measurements**, each over 100 iterations. All 41
workload checksums agreed wherever the embedded configurations completed.
No value assertion failed; resource failures remain failures.

QEMU does not establish hardware performance. Cortex-M reported no elapsed time;
RV32 emulated timings are retained in the raw data only. There is no x86-64
runtime or allocation result.

## Lifetime markers reduce stack reservations independently

For SequentialBlocks, which uses eight sequential 512-byte temporaries, the
comparison below changes only lifetime markers. Observations are bytes at opt 2.

| Target | Upstream dev | Lifetime markers |
| --- | ---: | ---: |
| ARM64 | 4,144 | 560 |
| Wasm Node | 4,096 | 512 |
| Wasm Wasmtime | 4,096 | 512 |
| Cortex-M3 | 4,128 | 536 |
| RV32 | 4,128 | 544 |
| x86-64 | 4,136 | 552 |

Both Wasm targets also show 4096 to 512 at opt z. These results demonstrate
storage reuse in this workload, not a universal percentage improvement for
applications or a complete stack bound.

Lowering the aggregate threshold provides a separate reduction. For
Bytes/512/Serial at opt 2, with markers enabled on both sides:

| Target | Threshold 1024 | Threshold 64 |
| --- | ---: | ---: |
| ARM64 | 8,176 | 576 |
| Wasm Node | 2,048 | 512 |
| Wasm Wasmtime | 2,048 | 512 |
| Cortex-M3 | Baseline exceeds flash | 536 |
| RV32 | 6,112 | 544 |
| x86-64 | 8,200 | 552 |

Wasm values are recognized constant stack-pointer decrements. Machine-code values
are recognized immediate entry reservations and pushes before the first branch
or call. Outlined prologues, register-sized adjustments, x86 return addresses,
and red-zone storage are not fully covered. A zero observation does not prove
zero stack use. Neither the largest observed reservation nor their sum gives
peak call-chain usage. Raw frame tables retain instruction evidence.

## Lower thresholds add allocations to update loops

The number of previously allocation-free cases gaining nonzero B/op, relative
to experimental threshold 1024, is:

| Threshold | ARM64, both Wasm engines, and RV32, each opt, out of 41 | Cortex-M opt 2, out of 25 comparable cases | Cortex-M opt z, out of 29 comparable cases |
| --- | ---: | ---: | ---: |
| 16 | 8 | 5 | 6 |
| 32 | 7 | 4 | 5 |
| 64 | 6 | 3 | 4 |
| 128 | 3 | 0 | 1 |
| 256 | 2 | 0 | 0 |
| 1024 | 0 | 0 | 0 |

The complete targets agree on which cases gain allocations. They are byte-array
Update workloads above the chosen threshold, plus Record at thresholds 16, 32,
and 64. Words64, Scene, and RecordInterface do not show the same new allocation.
Indirect passing does not necessarily allocate for every workload.

Cortex-M's smaller counts reflect missing large baseline cases. They do not
show that Cortex-M avoids the regression. Its comparisons include only workloads
that completed at both thresholds.

At threshold 64, compared with zero B/op at threshold 1024:

| Target | Bytes/65/Update | Record | Allocation counter |
| --- | ---: | ---: | --- |
| ARM64 with Boehm GC | 80 B/op | 672 B/op | Mallocs is reported as zero; use B/op |
| Wasm Node and Wasmtime | 65 B/op | 292 B/op | One allocation per operation |
| Cortex-M3 and RV32 | 65 B/op | 292 B/op | One allocation per operation |

Allocator accounting differs between GCs, so native and Wasm byte counts should
not be interpreted as equivalent object-layout measurements. Repeated allocations
create allocation and GC work; they do not by themselves establish a memory leak.

### Why the update pattern allocates

The minimal pattern is a value updated and carried into the next iteration:

```go
for i := 0; i < n; i++ {
    current = edit(current, i)
    sum += read(current, i)
}
```

Optimized Wasm IR for a 65-byte array confirms the mechanism. At threshold 1024,
the loop carries an aggregate SSA value without `runtime.alloc`. At threshold
64, the caller creates a result buffer for each call, and the buffer pointer
becomes the next iteration's input through a PHI node.

TinyGo initially creates indirect storage with `runtime.alloc`, then attempts
stack promotion. The existing escape analysis does not handle PHI nodes and
conservatively treats this use as escaping. Thus a buffer can remain on the heap
even when the Go value never leaves the function. The lower configuration exposes
an existing analysis limitation for smaller aggregates; it does not introduce
a new allocation mechanism. The 1025-byte cases already allocate at the default
threshold, but also exceed the independent stack-promotion size limit, so those
cases alone do not isolate PHI handling as the cause.

Lifetime markers cannot fix a buffer that never reaches the stack. A separate
optimization must preserve value semantics across iterations. Simply allowing
PHIs and replacing fresh buffers with one stack slot could make the next call's
input and output alias. Safe promotion or buffer reuse needs a focused compiler
change and regression tests; no such fix is included here.

## Runtime benefits and regressions coexist

The following ratios are median time at threshold 64 divided by median time at
1024. Both variants include markers and the same ABI-wrapper changes. Below 1
is faster; above 1 is slower.

| Execution target | Opt | Record | Bytes/65/Update | Bytes/512/Serial | RecordInterface |
| --- | --- | ---: | ---: | ---: | ---: |
| ARM64 | 2 | 6.256 | 1.470 | 0.037 | 0.431 |
| ARM64 | z | 6.455 | 1.513 | 0.019 | 0.437 |
| Wasm Node | 2 | 2.904 | 1.780 | 0.066 | 0.327 |
| Wasm Node | z | 3.167 | 1.776 | 0.062 | 0.319 |
| Wasm Wasmtime | 2 | 2.066 | 1.126 | 0.041 | 0.221 |
| Wasm Wasmtime | z | 2.113 | 1.125 | 0.041 | 0.224 |

At opt 2, the 512-byte serial workload is 26.93 times faster on ARM64, 15.07 times
faster in Node, and 24.40 times faster in Wasmtime. Record is respectively 6.26,
2.90, and 2.07 times slower. An average across the suite would hide that tradeoff.

Record also illustrates why the next threshold above a workload's size matters.
At opt 2, median ns/op across the threshold sweep is:

| Threshold | ARM64 | Wasm Node | Wasm Wasmtime |
| --- | ---: | ---: | ---: |
| 16 | 285.60 | 155.10 | 149.90 |
| 32 | 289.10 | 154.70 | 150.40 |
| 64 | 283.90 | 155.30 | 150.00 |
| 128 | 45.10 | 53.50 | 72.47 |
| 256 | 45.09 | 53.62 | 72.32 |
| 1024 | 45.38 | 53.48 | 72.60 |

These are three short, approximately 100 ms samples per configuration, not
confidence intervals. Compare changes within a target; differences between
engines, runtimes, and GCs prevent a clean architecture-speed ranking.

One control discrepancy remains marked in the original matrix. Node opt z
Bytes/128/Update measured 73.32 ns/op for markers only and 118.70 for experimental
1024 despite identical Wasm code sections. Three isolated one-second follow-ups
gave 73.49 and 73.44. This suggests sensitivity to the short suite's execution
conditions; it is not evidence of a threshold effect. The original observations
and [follow-up logs](results/2026-10-08/control128.json) are preserved.

## Code size decreases across architectures

The table compares code sections for experimental thresholds 1024 and 64, both
with markers. KiB means 1024 bytes; percentages use the unrounded byte counts.

| Target and suite | Opt 2 KiB, 1024 to 64 | Reduction | Opt z KiB, 1024 to 64 | Reduction |
| --- | ---: | ---: | ---: | ---: |
| ARM64, full | 1,127.1 to 364.3 | 67.7% | 838.0 to 231.2 | 72.4% |
| Wasm Node, full | 1,375.7 to 762.0 | 44.6% | 1,139.2 to 543.5 | 52.3% |
| Wasm Wasmtime, full | 1,391.4 to 778.1 | 44.1% | 1,161.6 to 565.9 | 51.3% |
| Cortex-M3, 65-byte cases | 205.4 to 191.1 | 6.9% | 126.1 to 114.3 | 9.3% |
| Cortex-M3, shapes | 213.2 to 203.2 | 4.7% | 134.5 to 125.3 | 6.8% |
| RV32, full | 1,299.0 to 388.0 | 70.1% | 1,111.7 to 269.8 | 75.7% |
| x86-64, full | 1,604.7 to 450.2 | 71.9% | 1,340.4 to 296.9 | 77.8% |

These sizes include the testing harness. Full-suite rows contain all 41 cases;
Cortex-M rows contain only the named selection. Cortex-M reductions cannot be
ranked against full-suite reductions as an architecture effect. Native code uses
Mach-O `__text`; Wasm uses code-section payload size; ELF uses `.text`, which can
also contain literals. These are not total file sizes or predicted application
savings. Per-threshold code and file sizes remain in the source tables.

## Cortex-M capacity limits are part of the result

The emulated Cortex-M3 board provides 256 KiB flash and 64 KiB RAM. The suite
uses separate array sizes and one shapes selection, with flat benchmark entry
points to avoid nested benchmark goroutines exhausting memory.

Of 160 configurations, 16 exceeded flash during linking and 21 exited with
out-of-memory errors. Every 1025-byte selection ran out of RAM, as did some
1024-byte selections. Some large direct-aggregate configurations exceeded flash
even when split. Lower thresholds let several of those configurations link,
but fitting flash does not guarantee that the test harness and workload fit RAM.

Failures include harness overhead and the chosen stack setting, so they do not
establish standalone workload memory requirements. Incomplete binaries contribute
no accepted allocation measurements. The [failure list](results/2026-10-08-architectures/failures.csv)
links each configuration to its raw log.

## Method and reproducibility

The host was an Apple M5 Max running macOS 27.0.1. The toolchain used Go 1.27.1,
LLVM and LLD 22.1.8, wasm-opt 133, wasm-tools 1.261.0, Node 26.10.0, Wasmtime
49.0.2, and QEMU 11.1.2.

| Target | GC and scheduler | Requested stack | Execution method |
| --- | --- | --- | --- |
| ARM64 Darwin | Boehm, threads | 64 KiB | Three randomized benchmark rounds |
| Wasm Node and Wasmtime | Precise, asyncify | 64 KiB | Three randomized benchmark rounds |
| Cortex-M3 | Conservative, tasks | 16 KiB | Correctness plus 100-iteration benchmarks |
| RV32 | Conservative, four-core scheduler | 64 KiB | Correctness plus 100-iteration benchmarks |
| x86-64 Darwin | Boehm, threads | 64 KiB | Compilation and inspection only |

All requested stack sizes retain the 1024-byte stack-promotion limit. Cortex-M's
setting differs from its default; testing smaller embedded stacks could change
promotion behavior independently of the aggregate threshold. Large boundary
cases also interact with call-signature limits, including Wasm's scalar-parameter
budget.

Compilation finished before the original timed campaign. Node test flags were
baked into the binary, so its runs also execute correctness tests outside the
benchmark timers. Embedded tests use baked-in `-bench='.*'` flags and fixed counts
because the bare-metal matcher and clocks differ from hosted environments.

The embedded extension refactored byte loops into shared helpers, initializing
inputs before resetting the timer. Cortex-M uses flat entry points; RV32 and
x86-64 builds use the nested suite. Each target's variants use identical workload
sources. Absolute sizes and timings across campaigns should not be treated as
controlled architecture comparisons.

An initial invalid harness let LLVM remove loops whose only outputs were unread
globals. Its timings are excluded. Accepted campaigns log observable checksums,
retain source snapshots, and record commands, source commits, binary hashes, and
exit statuses. The embedded source and executable hashes, and copies of the
original executables, were verified against manifests.

## What to do next

1. Keep the lifetime-marker change independent of threshold tuning. The measured
   reuse benefit spans all inspected architectures and does not depend on the
   proposed lower threshold.
2. Investigate safe promotion or reuse of loop-carried indirect result buffers.
   Cover aliasing and retained earlier values before broadening escape analysis.
3. Repeat the threshold sweep after that change, retaining upstream and marker-only
   controls. Include longer timing runs and representative application workloads.
4. Measure Cortex-M and RISC-V on hardware and execute x86-64 before choosing
   platform defaults. Include realistic embedded stack settings and capacity limits.

Architecture-specific tuning may still be useful after an allocation fix. Register
pressure, copy cost, code size, and stack use differ between backends. The current
evidence identifies the tradeoff; it does not show that any one threshold is optimal.

## Evidence and stable local files

| Campaign | Report | Runtime and allocation data | Code and stack data | Provenance |
| --- | --- | --- | --- | --- |
| ARM64 and Wasm | [Detailed report](results/2026-10-08/REPORT.md) | [Medians and ranges](results/2026-10-08/summary.csv), [samples](results/2026-10-08/measurements.csv) | [Sizes](results/2026-10-08/sizes.csv), [frames](results/2026-10-08/frames.csv) | [Manifest](results/2026-10-08/manifest.json) |
| Cortex-M and RV32 | [Detailed report](results/2026-10-08-architectures/REPORT.md) | [Accepted measurements](results/2026-10-08-architectures/measurements.csv), [checksums](results/2026-10-08-architectures/checksums.csv) | [Sizes](results/2026-10-08-architectures/sizes.csv), [frames](results/2026-10-08-architectures/frames.csv) | [Manifest](results/2026-10-08-architectures/manifest.json) |
| x86-64 | [Scope](results/2026-10-08-amd64/REPORT.md) | Not executed | [Sizes](results/2026-10-08-amd64/sizes.csv), [frames](results/2026-10-08-amd64/frames.csv) | [Manifest](results/2026-10-08-amd64/manifest.json) |

This is local experiment work, not intended for a PR. Workloads, scripts, logs,
and reports are retained on `nm-aggregate-workloads`. Generated binaries and
compiler worktrees live in ignored `artifacts/` and `worktrees/` directories beside
this report. Historical manifests keep the paths actually used for their runs.
The [README](README.md) provides reproduction commands and the stable local layout.
