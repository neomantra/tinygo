# Aggregate architecture sweep

See the [integrated report](../../REPORT.md) for the combined findings across architectures.

The allocation tradeoff also appears on Cortex-M3 and RV32. Lower thresholds
reduce code size and stack reservations, but introduce allocations in update
loops. The lifetime-marker improvement applies independently on Cortex-M3,
RV32, and x86-64. These results do not establish a new default threshold.

This is local testing work on `nm-aggregate-workloads`, not intended for a PR.
It extends the [native ARM64 and Wasm sweep](../2026-10-08/REPORT.md).

## Coverage

Eight compiler variants were tested at opt 2 and z: upstream dev, lifetime
markers only, and thresholds 16, 32, 64, 128, 256, and 1024 with markers and the
ABI-wrapper changes. Compiler binary hashes match the original sweep.

| Architecture | Attempted builds | Build success | Complete execution | Coverage |
| --- | ---: | ---: | ---: | --- |
| Cortex-M3 | 160 | 144 | 123 | Ten workload selections per compiler/optimization |
| RV32 | 16 | 16 | 16 | All 41 workloads per compiler/optimization |
| x86-64 Darwin | 16 | 16 | Not run | Code sections and partial entry stack reservations |

The embedded runs produced 1,164 accepted allocation measurements, each with
100 iterations. All 41 workload checksums agreed across successful configurations
and both embedded architectures. No value assertion failed. The 37 unsuccessful
Cortex-M configurations remain explicit failures, not passing results.

Cortex-M had 16 flash overflows and 21 out-of-memory exits. Its emulated board
has 256 KiB flash and 64 KiB RAM. All 1025-byte selections ran out of memory;
some 1024-byte selections also exhausted RAM. Larger direct aggregate workloads
exceeded flash even when compiled separately. These failures include the test
harness and do not establish minimum memory requirements for standalone workloads.
See [every failed configuration](failures.csv) and its raw log.

## Allocations

The RV32 counts exactly match the earlier ARM64 and Wasm sweep. Cortex-M counts
use only workloads that completed at both the candidate and threshold 1024.
The missing large cases prevent a full 41-case comparison on Cortex-M.

| Threshold | RV32 added allocation cases, both opts, out of 41 | Cortex-M opt 2, out of 25 comparable cases | Cortex-M opt z, out of 29 comparable cases |
| --- | ---: | ---: | ---: |
| 16 | 8 | 5 | 6 |
| 32 | 7 | 4 | 5 |
| 64 | 6 | 3 | 4 |
| 128 | 3 | 0 | 1 |
| 256 | 2 | 0 | 0 |
| 1024 | 0 | 0 | 0 |

On both Cortex-M and RV32, threshold 64 changes Bytes/65/Update from zero to
65 B/op and one allocation per operation. Record changes from zero to 292 B/op
and one allocation per operation. Words64, Scene, and RecordInterface do not
show that additional allocation. The effect depends on the workload.

The smaller Cortex-M counts do not show that it avoids the regression. Its
larger baseline workloads are unavailable because of the memory limits.

## Code size

These are whole-harness code sections in bytes. Cortex-M rows contain only the
named selection, so their absolute sizes cannot be compared to full-suite rows.
ELF `.text` can include embedded literals as well as instructions. x86-64 uses
Mach-O `__text`. Both threshold variants include lifetime markers.

| Target and selection | Opt | Threshold 1024 | Threshold 64 |
| --- | --- | ---: | ---: |
| Cortex-M, 65-byte cases | 2 | 210,296 | 195,688 |
| Cortex-M, 65-byte cases | z | 129,096 | 117,080 |
| Cortex-M, shapes | 2 | 218,312 | 208,040 |
| Cortex-M, shapes | z | 137,720 | 128,296 |
| RV32, full suite | 2 | 1,330,128 | 397,328 |
| RV32, full suite | z | 1,138,400 | 276,224 |
| x86-64, full suite | 2 | 1,643,181 | 460,989 |
| x86-64, full suite | z | 1,372,574 | 304,057 |

Lower thresholds also allow several Cortex-M configurations to fit flash when
the threshold-1024 version cannot link. That is useful, but must be weighed
against allocations and RAM consumption.

## Stack observations

Observed immediate entry reservations at opt 2, in bytes:

| Workload and comparison | Cortex-M3 | RV32 | x86-64 |
| --- | ---: | ---: | ---: |
| SequentialBlocks, dev to lifetime markers | 4,128 to 536 | 4,128 to 544 | 4,136 to 552 |
| Bytes/512/Serial, threshold 1024 to 64 with markers | Baseline exceeds flash; 64 uses 536 | 6,112 to 544 | 8,200 to 552 |

The inspector stops at the first branch or call and counts recognized immediate
stack adjustments and pushes. These observations are not complete frame or
call-chain bounds. Outlined prologues, register-sized adjustments, x86 return
addresses, and red-zone storage are not fully covered. Zero does not mean zero
stack use. Instruction evidence and unresolved adjustments are recorded per row.

## Method and limits

- Host: Apple M5 Max, macOS 27.0.1. LLVM and LLD 22.1.8, Go 1.27.1, QEMU 11.1.2.
- Cortex-M uses `cortex-m-qemu`, conservative GC, task scheduler, and a requested
  16 KiB stack. Flat benchmark entry points avoid nested benchmark goroutines.
- RV32 uses `riscv-qemu`, conservative GC, the four-core scheduler, and a requested
  64 KiB stack. It uses the nested full-suite benchmark entry points.
- Both stack settings retain the 1024-byte allocation-promotion limit. These are
  controlled settings, not the Cortex-M target's default stack configuration.
- Correctness and fixed-count benchmarks execute together with baked-in flags.
  Bare-metal matching requires `-bench='.*'`. Each accepted benchmark completed
  100 iterations. Incomplete binaries contribute no accepted measurements.
- Cortex-M reports no elapsed time for these runs. Those timing cells stay empty.
  RV32 QEMU timings are retained as emulated observations, not hardware performance.
- x86-64 was cross-compiled only. This machine has no x86-64 execution runner.
  The x86 results make no allocation or runtime-performance claim.
- Shared byte benchmark helpers now initialize inputs before resetting the timer.
  Cortex-M has flat entry points; RV32 retains nested entry points. These results
  are not a timing continuation of the earlier ARM64/Wasm run. Within each new
  target, every compiler variant uses the same workload sources and harness.
- Sources and all successful executable hashes were verified against manifests.
  Workload source snapshots accompany both result directories.

The update-loop mechanism was separately confirmed in optimized Wasm IR: a
lower threshold makes the result buffer pointer loop-carried through a PHI node.
The existing escape analysis treats that instruction conservatively, preventing
stack promotion. The threshold override exposes the limitation for smaller
values. The default 1025-byte case also exceeds the independent stack-promotion
size limit, so its allocation alone does not isolate PHI handling as the cause.
No compiler fix for this limitation is included in this testing branch.

## Data and local layout

- [Embedded commands, versions, hashes, and statuses](manifest.json)
- [Embedded allocation measurements](measurements.csv)
- [Embedded code sections](sizes.csv)
- [Embedded stack observations](frames.csv)
- [Matching 100-iteration checksums](checksums.csv)
- [Failures](failures.csv)
- [x86-64 manifest](../2026-10-08-amd64/manifest.json)
- [x86-64 code sections](../2026-10-08-amd64/sizes.csv)
- [x86-64 stack observations](../2026-10-08-amd64/frames.csv)
- [Reproduction instructions and stable directories](../../README.md)

Workloads, scripts, logs, reports, and manifests are committed on the testing
branch. Generated executables and compiler binaries are under the ignored
`tests/aggregatebench/artifacts/` directory. Stable compiler worktrees are under
`tests/aggregatebench/worktrees/`; the combined compiler commit is retained by
local branch `nm-aggregate-sweep-compiler`.
