#include "textflag.h"

TEXT ·execMemInner(SB),NOSPLIT,$0-16
    MOVD	mem+0(FP), R0
    MOVD	sp+8(FP), R1
    MOVD	R30, R19
    JMP		(R0)
    RET
