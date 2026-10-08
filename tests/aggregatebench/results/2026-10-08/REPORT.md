# Aggregate threshold sweep results

See the [integrated report](../../REPORT.md) for the combined findings across architectures.

The [architecture extension](../2026-10-08-architectures/REPORT.md) adds
Cortex-M and RV32 execution plus x86-64 code inspection.

Lower aggregate thresholds reduce code size and substantially speed up some
large-value workloads, but introduce heap allocations and slow down repeated
value updates. This happens on native ARM64 as well as wasm. The sweep does not
support choosing 64 as a general default yet. Investigate the indirect update
allocations first, then repeat the sweep. The lifetime-marker change remains a
separate improvement.

All 48 configurations passed correctness checks. All 144 timed runs completed,
producing 5,904 measurements: 41 cases, three repetitions, two optimization levels,
three execution targets, and eight compiler variants. These are controlled
synthetic workloads, not a repeat of the Bubble Tea application measurements.

## Allocation changes

The following number of previously allocation-free cases gained nonzero B/op
relative to the threshold-1024 sweep variant. The counts were identical across
all three execution targets and both optimization levels.

| Threshold | Cases gaining allocations, out of 41 |
| --- | ---: |
| 16 | 8 |
| 32 | 7 |
| 64 | 6 |
| 128 | 3 |
| 256 | 2 |
| 1024 | 0 |

These are array Update cases above the chosen threshold, plus Record at 16, 32,
and 64. Words64, Scene, and RecordInterface do not show the same new allocation.
This is a workload-dependent effect, not a claim that all indirect returns allocate.

At threshold 64, Bytes/65/Update allocates 65 B/op and Record allocates 292 B/op
on wasm, each with one allocation per operation. Their threshold-1024 versions
allocate zero. ARM64 reports 80 B/op and 672 B/op respectively. Its Boehm GC
reports Mallocs as zero, so native allocs/op cannot establish allocation counts.
B/op is the usable native metric here; do not equate allocator accounting across GCs.

Thresholds 128 and 256 avoid the medium-sized record regression, but still add
allocations to larger array updates. The 1025-byte cases already allocate at 1024.

## Runtime tradeoffs

Medians in nanoseconds per operation for Record across the complete sweep:

| Threshold | ARM64 opt 2 | ARM64 opt z | Node opt 2 | Node opt z | Wasmtime opt 2 | Wasmtime opt z |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 16 | 285.60 | 288.80 | 155.10 | 172.30 | 149.90 | 156.70 |
| 32 | 289.10 | 288.90 | 154.70 | 172.50 | 150.40 | 154.40 |
| 64 | 283.90 | 283.70 | 155.30 | 171.00 | 150.00 | 152.50 |
| 128 | 45.10 | 44.85 | 53.50 | 54.31 | 72.47 | 72.18 |
| 256 | 45.09 | 44.64 | 53.62 | 53.86 | 72.47 | 72.32 |
| 1024 | 45.38 | 43.95 | 53.48 | 54.00 | 72.60 | 72.16 |

The tradeoff is visible in contrasting cases. Below, each ratio is time at 64
**divided by** time at 1024. Lower is faster; values above 1 indicate a slowdown.
Both sides include lifetime markers and the same ABI-wrapper code.

| Execution target | Opt | Record ratio | Bytes/65/Update ratio | Bytes/512/Serial ratio | RecordInterface ratio |
| --- | --- | ---: | ---: | ---: | ---: |
| Native ARM64 | 2 | 6.256 | 1.470 | 0.037 | 0.431 |
| Native ARM64 | z | 6.455 | 1.513 | 0.019 | 0.437 |
| js/wasm Node | 2 | 2.904 | 1.780 | 0.066 | 0.327 |
| js/wasm Node | z | 3.167 | 1.776 | 0.062 | 0.319 |
| wasip1 Wasmtime | 2 | 2.066 | 1.126 | 0.041 | 0.221 |
| wasip1 Wasmtime | z | 2.113 | 1.125 | 0.041 | 0.224 |

For example, at opt 2, Record is 6.26 times slower on ARM64, 2.90 times slower
in Node, and 2.07 times slower in Wasmtime. The 512-byte serial workload is
26.93, 15.07, and 24.40 times faster respectively. An average of these cases would
hide the regressions; their relative importance depends on the application.

## Code and stack size

Code section sizes for the whole test executable, including its testing harness,
are below. These are within-target comparisons, not application size predictions.
Native size is Mach-O `__text`; wasm size is the code-section payload.

| Threshold | ARM64 opt 2 KiB | ARM64 opt z KiB | Node opt 2 KiB | Node opt z KiB | Wasmtime opt 2 KiB | Wasmtime opt z KiB |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 16 | 339.3 | 206.4 | 741.9 | 524.2 | 758.0 | 546.6 |
| 32 | 345.7 | 212.7 | 747.2 | 529.0 | 763.4 | 551.4 |
| 64 | 364.3 | 231.2 | 762.0 | 543.5 | 778.1 | 565.9 |
| 128 | 432.6 | 293.7 | 818.5 | 599.3 | 834.4 | 621.8 |
| 256 | 527.1 | 379.9 | 878.2 | 653.5 | 894.0 | 676.0 |
| 1024 | 1127.1 | 838.0 | 1375.7 | 1139.2 | 1391.4 | 1161.6 |

Lifetime markers alone reduce SequentialBlocks from 4096 to 512 bytes on both
wasm targets, at both optimization levels. ARM64 entry stack reservation drops
from 4144 to 560 bytes. This result does not require lowering the aggregate threshold.

For serialBytes[512] at opt 2, lowering the threshold from 1024 to 64 with markers
reduces wasm's observed reservation from 2048 to 512 bytes and ARM64's entry
reservation from 8176 to 576 bytes.

The wasm inspector recognizes constant stack-pointer adjustments. The native
inspector measures immediate entry reservations before the first branch or call,
not a complete stack bound. Outlined prologues can be missed, especially at opt z.
A zero observation does not establish zero stack usage. Neither the largest frame
nor a sum of frame sizes establishes peak call-chain stack usage.

## Method

- Machine: Apple M5 Max, macOS 27.0.1, native ARM64.
- Toolchain: Go 1.27.1, LLVM and LLD 22.1.8, wasm-opt 133, wasm-tools 1.261.0.
- Engines: Node 26.10.0 and Wasmtime 49.0.2.
- Targets: native Darwin ARM64 with Boehm/threads, js/wasm and wasip1 with precise GC/asyncify.
- Stack setting: 64 KiB for every build. This keeps the stack-promotion limit at 1024 bytes.
- Variants: unmodified dev f6d269f7; lifetime markers only 1901a9c4; and the six
  thresholds on the combined threshold/ABI-wrapper and lifetime changes.
- The combined source comes from e361fafc plus 1901a9c4. Each sweep compiler
  changes maxDirectAggregateValues() on all architectures; exported ABI limits
  are preserved. This experiment is broader than the proposed wasm-only patch.
- Each benchmark targets 100 ms, with three separate process runs in randomized
  configuration order per round. Compilation finished before measurements began.
- Checksums are logged after stopping the benchmark timer so optimization cannot
  discard unused results. js/wasm flags are baked into the test executable.
  Its correctness and benchmark passes both execute tests and benchmarks, with
  correctness tests outside the benchmark timers.

## Interpretation limits

These are short exploratory samples, not confidence intervals. Use the per-case
ranges in summary.csv, repeat representative applications, and use longer runs
before choosing a default. Compare timings within one execution target, not
between native and wasm as if their runtimes and GCs were equivalent.

One control discrepancy warrants special care. For Node opt z,
Bytes/128/Update measured 73.32 ns/op with markers only and 118.70 ns/op with
the threshold-1024 sweep compiler, despite identical wasm code-section bytes.
Three focused one-second runs gave medians of 73.49 and 73.44 ns/op respectively.
The discrepancy disappeared in the longer isolated check, suggesting sensitivity
to the short suite's execution conditions. Treat that particular comparison as
unresolved in the main matrix, not a threshold effect. The raw observations are
retained; [focused commands and logs](control128.json) record the follow-up.

The large boundary cases also interact with the 1000-parameter signature budget
and the stack-promotion size limit. Array updates, sparse reads, nested records,
and interface calls intentionally stress different behaviors. No single workload
establishes the best threshold for all programs.

This sweep executes ARM64 and wasm only. Cortex-M and RISC-V were not timed;
the earlier cross-compilation checks do not substitute for hardware measurements.

An earlier attempt was discarded because unread global sinks let LLVM remove
loops and standalone js/wasm ignored runtime benchmark flags. None of that timing
data is used here. Its logs remain in /private/tmp/tinygo-aggregate-sweep-invalid-harness.

## Data and reproduction

- [Summary with medians and ranges](summary.csv)
- [All individual measurements](measurements.csv)
- [Code and file sizes](sizes.csv)
- [Stack observations with instruction evidence](frames.csv)
- [Commands, statuses, versions, commits, and hashes](manifest.json)
- [Exact workload source snapshot](workload-source.tar.gz)
- [Sweep and inspection instructions](../../README.md)

Raw build, correctness, and benchmark logs and the six compiler patches are in
this directory. Copies of the compilers and test executables are preserved under
`tests/aggregatebench/artifacts/`. See the stable layout in the workload README.
Historical manifests retain their original temporary paths.
