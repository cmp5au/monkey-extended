#include "textflag.h"

TEXT ·ExecMem(SB),NOSPLIT,$0-16
    MOVD	mem+0(FP), R0
    MOVD	sp+8(FP), R1
    JMP		(R0)
    RET
