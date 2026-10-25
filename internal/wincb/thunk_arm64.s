//go:build windows && arm64

#include "textflag.h"

// cbCommon calls the callback at index R9 of cbTargets, which is the entry
// syscall.NewCallback made, and keeps F8-F15 across it: native code
// expects a C function to (they are callee-saved), runtime·callbackasm1
// does not, and the Go code of a callback may use them.
//
// The arguments are those of the entry: R0-R7 stay as they are, and the
// first eight words of the arguments on the stack are copied to the
// bottom of the new frame, where the callee expects them. R9 and R10-R11
// are not arguments nor callee-saved.
TEXT ·cbCommon(SB), NOSPLIT|NOFRAME, $0
	SUB   $176, RSP
	MOVD  LR, 128(RSP)
	FSTPD (F8, F9), 64(RSP)
	FSTPD (F10, F11), 80(RSP)
	FSTPD (F12, F13), 96(RSP)
	FSTPD (F14, F15), 112(RSP)
	LDP   176(RSP), (R10, R11)
	STP   (R10, R11), 0(RSP)
	LDP   192(RSP), (R10, R11)
	STP   (R10, R11), 16(RSP)
	LDP   208(RSP), (R10, R11)
	STP   (R10, R11), 32(RSP)
	LDP   224(RSP), (R10, R11)
	STP   (R10, R11), 48(RSP)
	MOVD  $·cbTargets(SB), R10
	ADD   R9<<3, R10, R10
	MOVD  (R10), R10
	BL    (R10)
	FLDPD 64(RSP), (F8, F9)
	FLDPD 80(RSP), (F10, F11)
	FLDPD 96(RSP), (F12, F13)
	FLDPD 112(RSP), (F14, F15)
	MOVD  128(RSP), LR
	ADD   $176, RSP
	RET

TEXT ·cbTableAddr(SB), NOSPLIT, $0-8
	MOVD $·cbTable(SB), R0
	MOVD R0, ret+0(FP)
	RET
