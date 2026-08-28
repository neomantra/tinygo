target datalayout = "e-m:e-p:32:32-i64:64-v128:64:128-a:0:32-n32-S64"
target triple = "armv7m-none-eabi"

@runtime.zeroSizedAlloc = internal global i8 0, align 1

declare nonnull ptr @runtime.alloc(i32, ptr)

; Test allocating a single int (i32) that should be allocated on the stack.
define void @testInt() {
  %alloc = call align 4 ptr @runtime.alloc(i32 4, ptr null)
  store i32 5, ptr %alloc
  ret void
}

; Test allocating an array of 3 i16 values that should be allocated on the
; stack.
define i16 @testArray() {
  %alloc = call align 2 ptr @runtime.alloc(i32 6, ptr null)
  %alloc.1 = getelementptr i16, ptr %alloc, i32 1
  store i16 5, ptr %alloc.1
  %alloc.2 = getelementptr i16, ptr %alloc, i32 2
  %val = load i16, ptr %alloc.2
  ret i16 %val
}

; Test allocating objects with an unknown alignment.
define void @testUnknownAlign() {
  %alloc32 = call ptr @runtime.alloc(i32 32, ptr null)
  store i8 5, ptr %alloc32
  %alloc24 = call ptr @runtime.alloc(i32 24, ptr null)
  store i16 5, ptr %alloc24
  %alloc12 = call ptr @runtime.alloc(i32 12, ptr null)
  store i16 5, ptr %alloc12
  %alloc6 = call ptr @runtime.alloc(i32 6, ptr null)
  store i16 5, ptr %alloc6
  %alloc3 = call ptr @runtime.alloc(i32 3, ptr null)
  store i16 5, ptr %alloc3
  ret void
}

; Call a function that will let the pointer escape, so the heap-to-stack
; transform shouldn't be applied.
define void @testEscapingCall() {
  %alloc = call align 4 ptr @runtime.alloc(i32 4, ptr null)
  %val = call ptr @escapeIntPtr(ptr %alloc)
  ret void
}

define void @testEscapingCall2() {
  %alloc = call align 4 ptr @runtime.alloc(i32 4, ptr null)
  %val = call ptr @escapeIntPtrSometimes(ptr %alloc, ptr %alloc)
  ret void
}

; Call a function that doesn't let the pointer escape.
define void @testNonEscapingCall() {
  %alloc = call align 4 ptr @runtime.alloc(i32 4, ptr null)
  %val = call ptr @noescapeIntPtr(ptr %alloc)
  ret void
}

; Return the allocated value, which lets it escape.
define ptr @testEscapingReturn() {
  %alloc = call align 4 ptr @runtime.alloc(i32 4, ptr null)
  ret ptr %alloc
}

; Do a non-escaping allocation in a loop.
define void @testNonEscapingLoop() {
entry:
  br label %loop
loop:
  %alloc = call align 4 ptr @runtime.alloc(i32 4, ptr null)
  %ptr = call ptr @noescapeIntPtr(ptr %alloc)
  %result = icmp eq ptr null, %ptr
  br i1 %result, label %loop, label %end
end:
  ret void
}

; Test a zero-sized allocation.
define void @testZeroSizedAlloc() {
  %alloc = call align 1 ptr @runtime.alloc(i32 0, ptr null)
  %ptr = call ptr @noescapeIntPtr(ptr %alloc)
  ret void
}

; Two sequential allocations with disjoint lifetimes in one block: the
; promoted allocas get lifetime markers so their stack slots can overlap.
define void @testSequentialLifetimes() {
  %alloc1 = call align 4 ptr @runtime.alloc(i32 12, ptr null)
  %v1 = call ptr @noescapeIntPtr(ptr %alloc1)
  %alloc2 = call align 4 ptr @runtime.alloc(i32 12, ptr null)
  %v2 = call ptr @noescapeIntPtr(ptr %alloc2)
  ret void
}

; The allocation is used in another block, so the conservative lifetime
; analysis does not emit markers for it.
define void @testCrossBlockUse(i1 %c) {
entry:
  %alloc = call align 4 ptr @runtime.alloc(i32 4, ptr null)
  store i32 5, ptr %alloc
  br i1 %c, label %then, label %done
then:
  %v = call ptr @noescapeIntPtr(ptr %alloc)
  br label %done
done:
  ret void
}

; A defined function may return its pointer argument; the returned alias's
; uses extend the allocation's lifetime.
define ptr @returnsArg(ptr %p) {
  ret ptr %p
}

define void @testReturnedPointer() {
  %alloc = call align 4 ptr @runtime.alloc(i32 8, ptr null)
  %alias = call ptr @returnsArg(ptr %alloc)
  store i32 7, ptr %alias
  ret void
}

; A defined function may also return its pointer argument inside an
; aggregate (like a slice). The lifetime analysis cannot cheaply follow
; that, so it must not emit markers at all.
define { ptr, i32 } @returnsArgInAggregate(ptr %p) {
  %agg = insertvalue { ptr, i32 } undef, ptr %p, 0
  %agg2 = insertvalue { ptr, i32 } %agg, i32 3, 1
  ret { ptr, i32 } %agg2
}

define void @testReturnedAggregate() {
  %alloc = call align 4 ptr @runtime.alloc(i32 8, ptr null)
  %agg = call { ptr, i32 } @returnsArgInAggregate(ptr %alloc)
  %ptr = extractvalue { ptr, i32 } %agg, 0
  store i32 7, ptr %ptr
  ret void
}

; The same call can receive both the original pointer and a derived alias;
; returnability must be checked per source value, not once per call.
define ptr @returnsSecondArg(ptr %unused, ptr %p) {
  ret ptr %p
}

define void @testReturnedDerivedArgument() {
  %alloc = call align 4 ptr @runtime.alloc(i32 8, ptr null)
  %derived = getelementptr i8, ptr %alloc, i32 0
  %alias = call ptr @returnsSecondArg(ptr %alloc, ptr %derived)
  store i32 9, ptr %alias
  ret void
}

declare ptr @escapeIntPtr(ptr)

declare ptr @noescapeIntPtr(ptr nocapture)

declare ptr @escapeIntPtrSometimes(ptr nocapture, ptr)
