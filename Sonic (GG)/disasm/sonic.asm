
; --- reset_boot  $0000 — reset: DI / IM 1 / poll VDP V-counter (IN A,($7E)) until line $B0 / JP init $0296. ---
0000  F3          DI
0001  ED 56       IM 1
0003  DB 7E       IN A,($7E)
0005  FE B0       CP $B0
0007  20 FA       JR NZ,$0003
0009  C3 96 02    JP $0296
000C  .byte 00 00                                           ; ..
000E  00          NOP
000F  00          NOP
0010  00          NOP
0011  00          NOP
0012  00          NOP
0013  00          NOP
0014  00          NOP
0015  00          NOP
0016  00          NOP
0017  00          NOP

; ==== rst18  $0018  (10 callers) — RST $18 gateway -> bankcall_4012 (a 1-byte call into bank 3 at $4012). ====
0018  C3 E2 02    JP $02E2
001B  00          NOP
001C  00          NOP
001D  00          NOP
001E  00          NOP
001F  00          NOP

; ==== rst20  $0020  (7 callers) — RST $20 gateway -> bankcall_4006 (a 1-byte call into bank 3 at $4006). ====
0020  C3 F8 02    JP $02F8
0023  00          NOP
0024  00          NOP
0025  00          NOP
0026  00          NOP
0027  00          NOP

; ==== rst28  $0028  (13 callers) — RST $28 gateway -> bankcall_4015 (a 1-byte call into bank 3 at $4015). ====
0028  C3 09 03    JP $0309
002B  00          NOP
002C  00          NOP
002D  00          NOP
002E  00          NOP
002F  00          NOP
0030  00          NOP
0031  00          NOP
0032  00          NOP
0033  00          NOP
0034  00          NOP
0035  00          NOP
0036  00          NOP
0037  00          NOP

; ==== int_vec  $0038  (91 callers) — maskable-interrupt vector (IM 1 / RST $38): JP frame_int $0073. ====
0038  C3 73 00    JP $0073
003B  .byte 44 65 76 65 6C 6F 70 65 64                      ; Developed
0044  20 42       JR NZ,$0088
0046  79          LD A,C
0047  20 28       JR NZ,$0071
0049  43          LD B,E
004A  29          ADD HL,HL
004B  20 31       JR NZ,$007E
004D  39          ADD HL,SP
004E  39          ADD HL,SP
004F  31 20 41    LD SP,$4120
0052  6E          LD L,(HL)
0053  63          LD H,E
0054  69          LD L,C
0055  65          LD H,L
0056  6E          LD L,(HL)
0057  74          LD (HL),H
0058  20 2D       JR NZ,$0087
005A  20 53       JR NZ,$00AF
005C  A5          AND L
005D  48          LD C,B
005E  61          LD H,C
005F  79          LD A,C
0060  61          LD H,C
0061  73          LD (HL),E
0062  68          LD L,B
0063  69          LD L,C
0064  2E 00       LD L,$00

; --- nmi_pause  $0066 — NMI vector = the Start/Pause button (pause handler). ---
0066  F3          DI
0067  F5          PUSH AF
0068  FD 7E 07    LD A,(IY+7)
006B  EE 08       XOR $08
006D  FD 77 07    LD (IY+7),A
0070  F1          POP AF
0071  FB          EI
0072  C9          RET

; --- frame_int  $0073 — per-frame vblank handler: IN A,($BF) acks the VDP int; gated on (IY+6) bit7; sets the VDP line counter (reg10/$8A) for the mid-frame HUD line-interrupt split; preserves IX/IY + the bank shadows ($D22F/$D230). ---
0073  F3          DI
0074  F5          PUSH AF
0075  E5          PUSH HL
0076  D5          PUSH DE
0077  C5          PUSH BC
0078  DB BF       IN A,($BF)
007A  FD CB 06 7E BIT 7,(IY+6)
007E  28 2C       JR Z,$00AC
0080  3A 41 D2    LD A,($D241)
0083  A7          AND A
0084  C2 B4 01    JP NZ,$01B4
0087  3A DC D2    LD A,($D2DC)
008A  A7          AND A
008B  28 1F       JR Z,$00AC
008D  FE FF       CP $FF
008F  28 1B       JR Z,$00AC
0091  32 42 D2    LD ($D242),A
0094  3E 0A       LD A,$0A
0096  D3 BF       OUT ($BF),A
0098  3E 8A       LD A,$8A
009A  D3 BF       OUT ($BF),A
009C  3A 19 D2    LD A,($D219)
009F  F6 10       OR $10
00A1  D3 BF       OUT ($BF),A
00A3  3E 80       LD A,$80
00A5  D3 BF       OUT ($BF),A
00A7  3E 03       LD A,$03
00A9  32 41 D2    LD ($D241),A
00AC  DD E5       PUSH IX
00AE  FD E5       PUSH IY
00B0  2A 2F D2    LD HL,($D22F)
00B3  E5          PUSH HL
00B4  FD CB 00 46 BIT 0,(IY+0)
00B8  C4 80 01    CALL NZ,$0180
00BB  FD CB 00 46 BIT 0,(IY+0)
00BF  CC F0 00    CALL Z,$00F0
00C2  FB          EI
00C3  3E 03       LD A,$03
00C5  32 FE FF    LD ($FFFE),A
00C8  32 2F D2    LD ($D22F),A
00CB  CD 00 40    CALL $4000
00CE  CD 02 06    CALL $0602
00D1  FD CB 03 66 BIT 4,(IY+3)
00D5  CC EB 00    CALL Z,$00EB
00D8  CD 88 06    CALL $0688
00DB  E1          POP HL
00DC  22 FE FF    LD ($FFFE),HL
00DF  22 2F D2    LD ($D22F),HL
00E2  FD E1       POP IY
00E4  DD E1       POP IX
00E6  C1          POP BC
00E7  D1          POP DE
00E8  E1          POP HL
00E9  F1          POP AF
00EA  C9          RET

; ==== sub_00EB (1 caller) ====
00EB  FD CB 03 AE RES 5,(IY+3)
00EF  C9          RET

; ==== sub_00F0 (1 caller) ====
00F0  3A 1A D2    LD A,($D21A)
00F3  E6 BF       AND $BF
00F5  D3 BF       OUT ($BF),A
00F7  3E 81       LD A,$81
00F9  D3 BF       OUT ($BF),A
00FB  3A 4B D2    LD A,($D24B)
00FE  ED 44       NEG
0100  D3 BF       OUT ($BF),A
0102  3E 88       LD A,$88
0104  D3 BF       OUT ($BF),A
0106  3A 4C D2    LD A,($D24C)
0109  D3 BF       OUT ($BF),A
010B  3E 89       LD A,$89
010D  D3 BF       OUT ($BF),A
010F  FD CB 00 6E BIT 5,(IY+0)
0113  C4 3E 08    CALL NZ,$083E
0116  CD 72 01    CALL $0172
0119  3E 01       LD A,$01
011B  32 FE FF    LD ($FFFE),A
011E  32 2F D2    LD ($D22F),A
0121  3E 02       LD A,$02
0123  32 FF FF    LD ($FFFF),A
0126  32 30 D2    LD ($D230),A
0129  FD CB 00 4E BIT 1,(IY+0)
012D  C4 3F 03    CALL NZ,$033F

; --- gp_vdp_update  $0130 — GAMEPLAY per-frame VDP update (NOT the title loader): page banks 8/9; CALL tile_stream $31BC (stream new level tiles); CALL scroll_draw $3282 (draw the newly-revealed map column); re-assert VDP reg 1 from shadow $D21A; SET (IY+0).0. ---
0130  3E 08       LD A,$08
0132  32 FE FF    LD ($FFFE),A
0135  32 2F D2    LD ($D22F),A
0138  3E 09       LD A,$09
013A  32 FF FF    LD ($FFFF),A
013D  32 30 D2    LD ($D230),A
0140  FD CB 07 7E BIT 7,(IY+7)
0144  C4 BC 31    CALL NZ,$31BC
0147  3E 01       LD A,$01
0149  32 FE FF    LD ($FFFE),A
014C  32 2F D2    LD ($D22F),A
014F  3E 02       LD A,$02
0151  32 FF FF    LD ($FFFF),A
0154  32 30 D2    LD ($D230),A
0157  3A AC D2    LD A,($D2AC)
015A  E6 80       AND $80
015C  CC 82 32    CALL Z,$3282
015F  3E FF       LD A,$FF
0161  32 AC D2    LD ($D2AC),A
0164  3A 1A D2    LD A,($D21A)
0167  D3 BF       OUT ($BF),A
0169  3E 81       LD A,$81
016B  D3 BF       OUT ($BF),A
016D  FD CB 00 C6 SET 0,(IY+0)
0171  C9          RET

; ==== palette_dispatch  $0172  (1 caller) — BIT (IY+6).7: clear -> load_palette $0586 ; set -> palette_special $0185. ====
0172  FD CB 06 7E BIT 7,(IY+6)
0176  20 04       JR NZ,$017C
0178  CD 86 05    CALL $0586
017B  C9          RET
017C  CD 85 01    CALL $0185
017F  C9          RET

; ==== sub_0180 (1 caller) ====
0180  FD CB 06 7E BIT 7,(IY+6)
0184  C8          RET Z

; ==== palette_special  $0185  (1 caller) — if $D2DC is 0/$FF -> load_palette; else load a full 32-colour CRAM palette from a home-bank table ($0216, or $0256 if (IY+7).4) via cram_load32. ====
0185  3A DC D2    LD A,($D2DC)
0188  A7          AND A
0189  28 ED       JR Z,$0178
018B  FE FF       CP $FF
018D  20 E9       JR NZ,$0178
018F  21 16 02    LD HL,$0216
0192  FD CB 07 66 BIT 4,(IY+7)
0196  28 03       JR Z,$019B
0198  21 56 02    LD HL,$0256
019B  CD 9F 01    CALL $019F
019E  C9          RET

; ==== cram_load32  $019F  (1 caller) — write the whole 32-colour CRAM (64 bytes) from (HL): CRAM addr 0 + $C0 cmd, then 32x (2 bytes -> data port $BE). ====
019F  06 20       LD B,$20
01A1  3E 00       LD A,$00
01A3  D3 BF       OUT ($BF),A
01A5  3E C0       LD A,$C0
01A7  D3 BF       OUT ($BF),A
01A9  7E          LD A,(HL)
01AA  D3 BE       OUT ($BE),A
01AC  23          INC HL
01AD  7E          LD A,(HL)
01AE  D3 BE       OUT ($BE),A
01B0  23          INC HL
01B1  10 F6       DJNZ $01A9
01B3  C9          RET
01B4  FE 01       CP $01
01B6  28 1F       JR Z,$01D7
01B8  FE 02       CP $02
01BA  28 14       JR Z,$01D0
01BC  3D          DEC A
01BD  32 41 D2    LD ($D241),A
01C0  DB 7E       IN A,($7E)
01C2  4F          LD C,A
01C3  3A 42 D2    LD A,($D242)
01C6  91          SUB C
01C7  D3 BF       OUT ($BF),A
01C9  3E 8A       LD A,$8A
01CB  D3 BF       OUT ($BF),A
01CD  C3 10 02    JP $0210
01D0  3D          DEC A
01D1  32 41 D2    LD ($D241),A
01D4  C3 10 02    JP $0210
01D7  3D          DEC A
01D8  32 41 D2    LD ($D241),A
01DB  3E 00       LD A,$00
01DD  D3 BF       OUT ($BF),A
01DF  3E C0       LD A,$C0
01E1  D3 BF       OUT ($BF),A
01E3  06 10       LD B,$10
01E5  21 16 02    LD HL,$0216
01E8  FD CB 07 66 BIT 4,(IY+7)
01EC  28 03       JR Z,$01F1
01EE  21 56 02    LD HL,$0256
01F1  7E          LD A,(HL)
01F2  D3 BE       OUT ($BE),A
01F4  23          INC HL
01F5  00          NOP
01F6  7E          LD A,(HL)
01F7  D3 BE       OUT ($BE),A
01F9  23          INC HL
01FA  00          NOP
01FB  7E          LD A,(HL)
01FC  D3 BE       OUT ($BE),A
01FE  23          INC HL
01FF  7E          LD A,(HL)
0200  23          INC HL
0201  D3 BE       OUT ($BE),A
0203  10 EC       DJNZ $01F1
0205  3A 19 D2    LD A,($D219)
0208  E6 EF       AND $EF
020A  D3 BF       OUT ($BF),A
020C  3E 80       LD A,$80
020E  D3 BF       OUT ($BF),A
0210  C1          POP BC
0211  D1          POP DE
0212  E1          POP HL
0213  F1          POP AF
0214  FB          EI
0215  C9          RET

; --- palette_data_A  $0216 — hardcoded 32-colour (64-byte) CRAM palette in the home bank (logo/title?). (data) ---
0216  .byte 20 04 40 07 70 07 A0 09 74 0B 10 0E E3 0B 50 0F ;  .@.p...t.....P.
0226  .byte 40 08 10 09 FB 0A B6 09 70 03 FB 07 20 05 FA 0F ; @.......p... ...
0236  .byte 40 04 00 0B 77 0F FB 0B B7 0B BB 0F 00 00 FF 0F ; @...w...........
0246  .byte 70 07 97 0B BB 0F 90 0A FB 0F DC 0F B7 0B 77 0B ; p.............w.

; --- palette_data_B  $0256 — hardcoded 32-colour CRAM palette (alternate). (data) ---
0256  .byte 20 04 40 07 70 07 A0 09 74 0B 10 0E E3 0B 50 0F ;  .@.p...t.....P.
0266  .byte 40 08 10 09 FB 0A B6 09 70 03 FB 07 20 05 FA 0F ; @.......p... ...
0276  .byte 00 07 00 0B 77 0F FB 0B B7 0B BB 0F 00 00 FF 0F ; ....w...........
0286  .byte 54 07 EA 0F FF 07 BA 0C 70 07 BB 0F B7 07 00 00 ; T.......p.......

; --- init  $0296 — cold start: program the mapper (control $FFFC=$80; slots $FFFD/E/F <- banks 0/1/2); clear work RAM via overlapping LDIR and set SP=$DFEF; write VDP regs 0-10 from vdp_reg_table with a shadow at $D219; hide all sprites; CALL bankcall_4006; LD IY,$D200; JP main_entry. ---
0296  3E 80       LD A,$80
0298  32 FC FF    LD ($FFFC),A
029B  3E 00       LD A,$00
029D  32 FD FF    LD ($FFFD),A
02A0  3E 01       LD A,$01
02A2  32 FE FF    LD ($FFFE),A
02A5  3E 02       LD A,$02
02A7  32 FF FF    LD ($FFFF),A
02AA  21 00 C0    LD HL,$C000
02AD  11 01 C0    LD DE,$C001
02B0  01 EF 1F    LD BC,$1FEF
02B3  75          LD (HL),L
02B4  ED B0       LDIR
02B6  F9          LD SP,HL
02B7  21 1C 03    LD HL,$031C
02BA  11 19 D2    LD DE,$D219
02BD  06 0B       LD B,$0B
02BF  0E 8B       LD C,$8B
02C1  7E          LD A,(HL)
02C2  12          LD (DE),A
02C3  23          INC HL
02C4  13          INC DE
02C5  D3 BF       OUT ($BF),A
02C7  79          LD A,C
02C8  90          SUB B
02C9  D3 BF       OUT ($BF),A
02CB  10 F4       DJNZ $02C1
02CD  21 00 3F    LD HL,$3F00
02D0  01 40 00    LD BC,$0040
02D3  3E E0       LD A,$E0
02D5  CD F0 05    CALL $05F0
02D8  CD F8 02    CALL $02F8
02DB  FD 21 00 D2 LD IY,$D200
02DF  C3 56 13    JP $1356

; --- bankcall_4012  $02E2 — banked call: DI; slot1 <- bank3 ($FFFE=3); save A->$D2D3; CALL $4012; restore slot1 from shadow $D22F; EI; RET. (RST $18 target.) ---
02E2  F3          DI
02E3  F5          PUSH AF
02E4  3E 03       LD A,$03
02E6  32 FE FF    LD ($FFFE),A
02E9  F1          POP AF
02EA  32 D3 D2    LD ($D2D3),A
02ED  CD 12 40    CALL $4012
02F0  3A 2F D2    LD A,($D22F)
02F3  32 FE FF    LD ($FFFE),A
02F6  FB          EI
02F7  C9          RET

; ==== bankcall_4006  $02F8  (1 caller) — banked call: DI; slot1 <- bank3; CALL $4006; restore slot1 from shadow $D22F; EI; RET. (RST $20 target.) ====
02F8  F3          DI
02F9  3E 03       LD A,$03
02FB  32 FE FF    LD ($FFFE),A
02FE  CD 06 40    CALL $4006
0301  3A 2F D2    LD A,($D22F)
0304  32 FE FF    LD ($FFFE),A
0307  FB          EI
0308  C9          RET

; --- bankcall_4015  $0309 — banked call: DI; slot1 <- bank3; save A; CALL $4015; restore slot1 from shadow $D22F; EI; RET. (RST $28 target.) ---
0309  F3          DI
030A  F5          PUSH AF
030B  3E 03       LD A,$03
030D  32 FE FF    LD ($FFFE),A
0310  F1          POP AF
0311  CD 15 40    CALL $4015
0314  3A 2F D2    LD A,($D22F)
0317  32 FE FF    LD ($FFFE),A
031A  FB          EI
031B  C9          RET

; --- vdp_reg_table  $031C — 11-byte VDP register table (regs 0..10) = 26 A2 FF FF FF FF FF 00 00 00 FF. R0=Mode4+hide-left-col, R1=display OFF+frame-int+8x16 sprites, R2=name table $3800, R5=SAT $3F00, R6=sprite patterns $2000, R7=backdrop col 0, R8/9=scroll 0, R10=line counter. (data) ---
031C  .byte 26 A2 FF FF FF FF FF 00 00 00 FF                ; &..........

; ==== sub_0327 (35 callers) ====
0327  FD CB 00 46 BIT 0,(IY+0)
032B  28 FA       JR Z,$0327
032D  C9          RET
032E  .byte FD CB 00 D6 22 26 D2 ED 53 28 D2 ED 43 2A D2 C9 ; ...."&..S(..C*..
033E  .byte C9                                              ; .

; ==== sub_033F (1 caller) ====
033F  3E 00       LD A,$00
0341  D3 BF       OUT ($BF),A
0343  3E 3F       LD A,$3F
0345  F6 40       OR $40
0347  D3 BF       OUT ($BF),A
0349  FD 46 0A    LD B,(IY+10)
034C  21 01 D0    LD HL,$D001
034F  11 03 00    LD DE,$0003
0352  78          LD A,B
0353  A7          AND A
0354  28 06       JR Z,$035C
0356  7E          LD A,(HL)
0357  D3 BE       OUT ($BE),A
0359  19          ADD HL,DE
035A  10 FA       DJNZ $0356
035C  3A B5 D2    LD A,($D2B5)
035F  47          LD B,A
0360  FD 7E 0A    LD A,(IY+10)
0363  4F          LD C,A
0364  B8          CP B
0365  30 09       JR NC,$0370
0367  78          LD A,B
0368  91          SUB C
0369  47          LD B,A
036A  3E E0       LD A,$E0
036C  D3 BE       OUT ($BE),A
036E  10 FA       DJNZ $036A
0370  79          LD A,C
0371  A7          AND A
0372  C8          RET Z
0373  21 00 D0    LD HL,$D000
0376  FD 46 0A    LD B,(IY+10)
0379  3E 80       LD A,$80
037B  D3 BF       OUT ($BF),A
037D  3E 3F       LD A,$3F
037F  F6 40       OR $40
0381  D3 BF       OUT ($BF),A
0383  7E          LD A,(HL)
0384  D3 BE       OUT ($BE),A
0386  2C          INC L
0387  2C          INC L
0388  7E          LD A,(HL)
0389  D3 BE       OUT ($BE),A
038B  2C          INC L
038C  10 F5       DJNZ $0383
038E  FD 7E 0A    LD A,(IY+10)
0391  32 B5 D2    LD ($D2B5),A
0394  FD 70 0A    LD (IY+10),B
0397  C9          RET
0398  .byte F3 7B D3 BF 7A F6 40 D3 BF FB 7E D3 BE 23 0B 78 ; .{..z.@...~..#.x
03A8  .byte B1 C2 A2 03 C9 F3 F5 7B D3 BF 7A F6 40 D3 BF F1 ; .......{..z.@...
03B8  .byte ED 5B 2F D2 D5 32 FE FF 32 2F D2 3C 32 FF FF 32 ; .[/..2..2/.<2..2
03C8  .byte 30 D2 FB 7E 2F 5F 7E BB 28 0C D3 BE 5F 23 0B 78 ; 0..~/_~.(..._#.x
03D8  .byte B1 C2 CE 03 18 18 57 23 0B 78 B1 28 11 7A 5E D3 ; ......W#.x.(.z^.
03E8  .byte BE 1D 00 00 C2 E7 03 23 0B 78 B1 C2 CB 03 F3 D1 ; .......#.x......
03F8  .byte ED 53 2F D2 7B 32 FE FF 7A 32 FF FF FB C9       ; .S/.{2..z2....

; ==== decompress  $0406  (21 callers) — THE graphics decompressor. Source = (bank A in reg A, address HL); NORMALISE so HL can span banks: while HL>=$4000 { HL-=$4000; A++ }, then map bank A into slot1 + bank A+1 into slot2 and HL+=$4000 (a 32 KB source window). Output streams to VRAM (OUT $BE) in 4-BYTE UNITS (= one 4bpp tile row). HEADER at the source: +0 2 bytes(skip), +2 word1, +4 word2, +6 word3=COUNT (units); +8 = CONTROL BITMAP; (source+word1)=MATCH-INFO stream; (source+word2)=LITERAL stream (also the back-ref base). For unit i=0..count-1: if control bit i (byte ctrl[i>>3], mask bitmask_tab[i&7]) is CLEAR -> LITERAL: emit the next 4 bytes from the literal stream (advance). If SET -> MATCH: read a byte b from match-info; if b>=$F0 then hi=b-$F0 and read another byte b2, off=((hi<<8)|b2)*4 else off=b*4; emit the 4 bytes at literal_base+off. So it deduplicates repeated 4-byte tile rows. ====
0406  F3          DI
0407  F5          PUSH AF
0408  7C          LD A,H
0409  FE 40       CP $40
040B  38 08       JR C,$0415
040D  D6 40       SUB $40
040F  67          LD H,A
0410  F1          POP AF
0411  3C          INC A
0412  C3 07 04    JP $0407
0415  7B          LD A,E
0416  D3 BF       OUT ($BF),A
0418  7A          LD A,D
0419  F6 40       OR $40
041B  D3 BF       OUT ($BF),A
041D  F1          POP AF
041E  11 00 40    LD DE,$4000
0421  19          ADD HL,DE
0422  ED 5B 2F D2 LD DE,($D22F)
0426  D5          PUSH DE
0427  32 FE FF    LD ($FFFE),A
042A  32 2F D2    LD ($D22F),A
042D  3C          INC A
042E  32 FF FF    LD ($FFFF),A
0431  32 30 D2    LD ($D230),A
0434  FD CB 09 4E BIT 1,(IY+9)
0438  20 01       JR NZ,$043B
043A  FB          EI
043B  22 13 D2    LD ($D213),HL
043E  23          INC HL
043F  23          INC HL
0440  5E          LD E,(HL)
0441  23          INC HL
0442  56          LD D,(HL)
0443  23          INC HL
0444  D5          PUSH DE
0445  5E          LD E,(HL)
0446  23          INC HL
0447  56          LD D,(HL)
0448  D5          PUSH DE
0449  23          INC HL
044A  4E          LD C,(HL)
044B  23          INC HL
044C  46          LD B,(HL)
044D  23          INC HL
044E  ED 43 11 D2 LD ($D211),BC
0452  22 15 D2    LD ($D215),HL
0455  D9          EXX
0456  ED 4B 13 D2 LD BC,($D213)
045A  59          LD E,C
045B  50          LD D,B
045C  E1          POP HL
045D  09          ADD HL,BC
045E  22 0F D2    LD ($D20F),HL
0461  4D          LD C,L
0462  44          LD B,H
0463  E1          POP HL
0464  19          ADD HL,DE
0465  EB          EX DE,HL
0466  D9          EXX
0467  2A 11 D2    LD HL,($D211)
046A  AF          XOR A
046B  ED 42       SBC HL,BC
046D  E5          PUSH HL
046E  57          LD D,A
046F  7D          LD A,L
0470  E6 07       AND $07
0472  5F          LD E,A
0473  21 FA 04    LD HL,$04FA
0476  19          ADD HL,DE
0477  7E          LD A,(HL)
0478  D1          POP DE
0479  CB 3A       SRL D
047B  CB 1B       RR E
047D  CB 3A       SRL D
047F  CB 1B       RR E
0481  CB 3A       SRL D
0483  CB 1B       RR E
0485  2A 15 D2    LD HL,($D215)
0488  19          ADD HL,DE
0489  5F          LD E,A
048A  7E          LD A,(HL)
048B  A3          AND E
048C  20 21       JR NZ,$04AF
048E  D9          EXX
048F  0A          LD A,(BC)
0490  D3 BE       OUT ($BE),A
0492  03          INC BC
0493  E5          PUSH HL
0494  E1          POP HL
0495  0A          LD A,(BC)
0496  D3 BE       OUT ($BE),A
0498  03          INC BC
0499  E5          PUSH HL
049A  E1          POP HL
049B  0A          LD A,(BC)
049C  D3 BE       OUT ($BE),A
049E  03          INC BC
049F  E5          PUSH HL
04A0  E1          POP HL
04A1  0A          LD A,(BC)
04A2  D3 BE       OUT ($BE),A
04A4  03          INC BC
04A5  D9          EXX
04A6  0B          DEC BC
04A7  78          LD A,B
04A8  B1          OR C
04A9  C2 67 04    JP NZ,$0467
04AC  C3 E4 04    JP $04E4
04AF  D9          EXX
04B0  1A          LD A,(DE)
04B1  13          INC DE
04B2  D9          EXX
04B3  26 00       LD H,$00
04B5  FE F0       CP $F0
04B7  38 07       JR C,$04C0
04B9  D6 F0       SUB $F0
04BB  67          LD H,A
04BC  D9          EXX
04BD  1A          LD A,(DE)
04BE  13          INC DE
04BF  D9          EXX
04C0  6F          LD L,A
04C1  29          ADD HL,HL
04C2  29          ADD HL,HL
04C3  ED 5B 0F D2 LD DE,($D20F)
04C7  19          ADD HL,DE
04C8  7E          LD A,(HL)
04C9  D3 BE       OUT ($BE),A
04CB  23          INC HL
04CC  E5          PUSH HL
04CD  E1          POP HL
04CE  7E          LD A,(HL)
04CF  D3 BE       OUT ($BE),A
04D1  23          INC HL
04D2  E5          PUSH HL
04D3  E1          POP HL
04D4  7E          LD A,(HL)
04D5  D3 BE       OUT ($BE),A
04D7  23          INC HL
04D8  E5          PUSH HL
04D9  E1          POP HL
04DA  7E          LD A,(HL)
04DB  D3 BE       OUT ($BE),A
04DD  23          INC HL
04DE  0B          DEC BC
04DF  78          LD A,B
04E0  B1          OR C
04E1  C2 67 04    JP NZ,$0467
04E4  FD CB 09 4E BIT 1,(IY+9)
04E8  20 01       JR NZ,$04EB
04EA  F3          DI
04EB  D1          POP DE
04EC  ED 53 2F D2 LD ($D22F),DE
04F0  ED 53 FE FF LD ($FFFE),DE
04F4  FB          EI
04F5  FD CB 09 8E RES 1,(IY+9)
04F9  C9          RET

; --- bitmask_tab  $04FA — 8 single-bit masks: 01 02 04 08 10 20 40 80 (control-bitmap bit select, indexed i&7). (data) ---
04FA  .byte 01 02 04 08 10 20 40 80                         ; ..... @.

; ==== sub_0502 (10 callers) ====
0502  F3          DI
0503  F5          PUSH AF
0504  7B          LD A,E
0505  D3 BF       OUT ($BF),A
0507  7A          LD A,D
0508  F6 40       OR $40
050A  D3 BF       OUT ($BF),A
050C  F1          POP AF
050D  ED 5B 2F D2 LD DE,($D22F)
0511  D5          PUSH DE
0512  32 FE FF    LD ($FFFE),A
0515  32 2F D2    LD ($D22F),A
0518  3C          INC A
0519  32 FF FF    LD ($FFFF),A
051C  32 30 D2    LD ($D230),A
051F  FB          EI
0520  7E          LD A,(HL)
0521  2F          CPL
0522  5F          LD E,A
0523  7E          LD A,(HL)
0524  BB          CP E
0525  28 15       JR Z,$053C
0527  FE FF       CP $FF
0529  28 3E       JR Z,$0569
052B  D3 BE       OUT ($BE),A
052D  5F          LD E,A
052E  3A 0F D2    LD A,($D20F)
0531  D3 BE       OUT ($BE),A
0533  23          INC HL
0534  0B          DEC BC
0535  78          LD A,B
0536  B1          OR C
0537  C2 23 05    JP NZ,$0523
053A  18 21       JR $055D
053C  57          LD D,A
053D  23          INC HL
053E  0B          DEC BC
053F  78          LD A,B
0540  B1          OR C
0541  28 1A       JR Z,$055D
0543  7A          LD A,D
0544  5E          LD E,(HL)
0545  FE FF       CP $FF
0547  28 2F       JR Z,$0578
0549  D3 BE       OUT ($BE),A
054B  F5          PUSH AF
054C  3A 0F D2    LD A,($D20F)
054F  D3 BE       OUT ($BE),A
0551  F1          POP AF
0552  1D          DEC E
0553  C2 49 05    JP NZ,$0549
0556  23          INC HL
0557  0B          DEC BC
0558  78          LD A,B
0559  B1          OR C
055A  C2 20 05    JP NZ,$0520
055D  F3          DI
055E  D1          POP DE
055F  ED 53 2F D2 LD ($D22F),DE
0563  ED 53 FE FF LD ($FFFE),DE
0567  FB          EI
0568  C9          RET
0569  5F          LD E,A
056A  DB BE       IN A,($BE)
056C  00          NOP
056D  23          INC HL
056E  0B          DEC BC
056F  DB BE       IN A,($BE)
0571  78          LD A,B
0572  B1          OR C
0573  C2 23 05    JP NZ,$0523
0576  18 E5       JR $055D
0578  DB BE       IN A,($BE)
057A  F5          PUSH AF
057B  F1          POP AF
057C  DB BE       IN A,($BE)
057E  00          NOP
057F  1D          DEC E
0580  C2 78 05    JP NZ,$0578
0583  C3 56 05    JP $0556

; ==== load_palette  $0586  (1 caller) — page bank 8 into slot 1; CRAM addr 0 + $C0 cmd; load BG palette [$D22C] (16 colours) then sprite palette [$D22D] (15) via palette_by_index; restore slot 1 <- bank 1. (1 caller: $0178.) ====
0586  3E 08       LD A,$08
0588  32 FE FF    LD ($FFFE),A
058B  32 2F D2    LD ($D22F),A
058E  3E 00       LD A,$00
0590  D3 BF       OUT ($BF),A
0592  3E C0       LD A,$C0
0594  D3 BF       OUT ($BF),A
0596  3A 2C D2    LD A,($D22C)
0599  ED 5B A5 D2 LD DE,($D2A5)
059D  06 10       LD B,$10
059F  CD C4 05    CALL $05C4
05A2  2A A7 D2    LD HL,($D2A7)
05A5  06 01       LD B,$01
05A7  CD E5 05    CALL $05E5
05AA  3A 2D D2    LD A,($D22D)
05AD  11 02 00    LD DE,$0002
05B0  06 0F       LD B,$0F
05B2  CD C4 05    CALL $05C4
05B5  21 00 00    LD HL,$0000
05B8  22 A5 D2    LD ($D2A5),HL
05BB  3E 01       LD A,$01
05BD  32 FE FF    LD ($FFFE),A
05C0  32 2F D2    LD ($D22F),A
05C3  C9          RET

; ==== palette_by_index  $05C4  (2 callers) — A = palette index -> bank-8 pointer table at $7400: ptr = *($7400+A*2); data = ptr + $7400; then write B colours (2 bytes each) to CRAM ($BE). A=$FF/$FE select the RAM working palettes $D3BD/$D3DF. ====
05C4  21 BD D3    LD HL,$D3BD
05C7  FE FF       CP $FF
05C9  28 17       JR Z,$05E2
05CB  21 DF D3    LD HL,$D3DF
05CE  FE FE       CP $FE
05D0  28 10       JR Z,$05E2
05D2  C5          PUSH BC
05D3  87          ADD A,A
05D4  6F          LD L,A
05D5  26 00       LD H,$00
05D7  01 00 74    LD BC,$7400
05DA  09          ADD HL,BC
05DB  7E          LD A,(HL)
05DC  23          INC HL
05DD  66          LD H,(HL)
05DE  6F          LD L,A
05DF  09          ADD HL,BC
05E0  19          ADD HL,DE
05E1  C1          POP BC
05E2  22 A7 D2    LD ($D2A7),HL

; ==== sub_05E5 (1 caller) ====
05E5  7E          LD A,(HL)
05E6  D3 BE       OUT ($BE),A
05E8  23          INC HL
05E9  7E          LD A,(HL)
05EA  D3 BE       OUT ($BE),A
05EC  23          INC HL
05ED  10 F6       DJNZ $05E5
05EF  C9          RET

; ==== vdp_fill  $05F0  (1 caller) — VDP fill primitive: fill(HL=VRAM address, BC=count, A=byte). Writes HL low then HL high|$40 (the write-VRAM command) to control port $BF, then OUTs the byte BC times to data port $BE. (VRAM cmd bits: $00 read, $40 write VRAM, $80 register, $C0 write CRAM.) ====
05F0  5F          LD E,A
05F1  7D          LD A,L
05F2  D3 BF       OUT ($BF),A
05F4  7C          LD A,H
05F5  F6 40       OR $40
05F7  D3 BF       OUT ($BF),A
05F9  7B          LD A,E
05FA  D3 BE       OUT ($BE),A
05FC  0B          DEC BC
05FD  78          LD A,B
05FE  B1          OR C
05FF  20 F8       JR NZ,$05F9
0601  C9          RET

; ==== sub_0602 (1 caller) ====
0602  DB 00       IN A,($00)
0604  E6 80       AND $80
0606  57          LD D,A
0607  DB DC       IN A,($DC)
0609  E6 7F       AND $7F
060B  F6 40       OR $40
060D  B2          OR D
060E  FD 77 03    LD (IY+3),A
0611  C9          RET

; ==== sub_0612 (24 callers) ====
0612  4E          LD C,(HL)
0613  23          INC HL
0614  7E          LD A,(HL)
0615  23          INC HL
0616  0F          RRCA
0617  0F          RRCA
0618  5F          LD E,A
0619  E6 3F       AND $3F
061B  57          LD D,A
061C  7B          LD A,E
061D  E6 C0       AND $C0
061F  5F          LD E,A
0620  06 00       LD B,$00
0622  EB          EX DE,HL
0623  CB 21       SLA C
0625  09          ADD HL,BC
0626  01 00 38    LD BC,$3800
0629  09          ADD HL,BC
062A  F3          DI
062B  7D          LD A,L
062C  D3 BF       OUT ($BF),A
062E  7C          LD A,H
062F  F6 40       OR $40
0631  D3 BF       OUT ($BF),A
0633  FB          EI
0634  1A          LD A,(DE)
0635  FE FF       CP $FF
0637  C8          RET Z
0638  D3 BE       OUT ($BE),A
063A  F5          PUSH AF
063B  F1          POP AF
063C  3A 0F D2    LD A,($D20F)
063F  D3 BE       OUT ($BE),A
0641  13          INC DE
0642  10 F0       DJNZ $0634
0644  C9          RET

; ==== sub_0645 (6 callers) ====
0645  21 00 D0    LD HL,$D000
0648  5D          LD E,L
0649  54          LD D,H
064A  01 BD 00    LD BC,$00BD
064D  3E E0       LD A,$E0
064F  12          LD (DE),A
0650  13          INC DE
0651  12          LD (DE),A
0652  13          INC DE
0653  13          INC DE
0654  ED B0       LDIR
0656  FD 36 0A 40 LD (IY+10),$40
065A  AF          XOR A
065B  32 B5 D2    LD ($D2B5),A
065E  C9          RET

; ==== sub_065F (3 callers) ====
065F  AF          XOR A
0660  06 07       LD B,$07
0662  EB          EX DE,HL
0663  6F          LD L,A
0664  67          LD H,A
0665  CB 11       RL C
0667  D2 6B 06    JP NC,$066B
066A  19          ADD HL,DE
066B  29          ADD HL,HL
066C  10 F7       DJNZ $0665
066E  B1          OR C
066F  C8          RET Z
0670  19          ADD HL,DE
0671  C9          RET

; ==== sub_0672 (2 callers) ====
0672  AF          XOR A
0673  06 10       LD B,$10
0675  CB 15       RL L
0677  CB 14       RL H
0679  17          RLA
067A  B9          CP C
067B  DA 7F 06    JP C,$067F
067E  91          SUB C
067F  3F          CCF
0680  CB 13       RL E
0682  CB 12       RL D
0684  10 EF       DJNZ $0675
0686  EB          EX DE,HL
0687  C9          RET

; ==== sub_0688 (6 callers) ====
0688  E5          PUSH HL
0689  D5          PUSH DE
068A  2A D8 D2    LD HL,($D2D8)
068D  5D          LD E,L
068E  54          LD D,H
068F  19          ADD HL,DE
0690  19          ADD HL,DE
0691  7D          LD A,L
0692  84          ADD A,H
0693  67          LD H,A
0694  85          ADD A,L
0695  6F          LD L,A
0696  11 54 00    LD DE,$0054
0699  19          ADD HL,DE
069A  22 D8 D2    LD ($D2D8),HL
069D  7C          LD A,H
069E  D1          POP DE
069F  E1          POP HL
06A0  C9          RET

; ==== sub_06A1 (2 callers) ====
06A1  ED 4B 4B D2 LD BC,($D24B)
06A5  2A 54 D2    LD HL,($D254)
06A8  ED 5B 69 D2 LD DE,($D269)
06AC  A7          AND A
06AD  ED 52       SBC HL,DE
06AF  38 0A       JR C,$06BB
06B1  7D          LD A,L
06B2  81          ADD A,C
06B3  4F          LD C,A
06B4  FD CB 00 B6 RES 6,(IY+0)
06B8  C3 C2 06    JP $06C2
06BB  7D          LD A,L
06BC  81          ADD A,C
06BD  4F          LD C,A
06BE  FD CB 00 F6 SET 6,(IY+0)
06C2  2A 57 D2    LD HL,($D257)
06C5  ED 5B 6B D2 LD DE,($D26B)
06C9  A7          AND A
06CA  ED 52       SBC HL,DE
06CC  38 10       JR C,$06DE
06CE  7D          LD A,L
06CF  80          ADD A,B
06D0  FE E0       CP $E0
06D2  38 02       JR C,$06D6
06D4  C6 20       ADD A,$20
06D6  47          LD B,A
06D7  FD CB 00 BE RES 7,(IY+0)
06DB  C3 EB 06    JP $06EB
06DE  7D          LD A,L
06DF  80          ADD A,B
06E0  FE E0       CP $E0
06E2  38 02       JR C,$06E6
06E4  D6 20       SUB $20
06E6  47          LD B,A
06E7  FD CB 00 FE SET 7,(IY+0)
06EB  ED 43 4B D2 LD ($D24B),BC
06EF  2A 54 D2    LD HL,($D254)
06F2  CB 25       SLA L
06F4  CB 14       RL H
06F6  CB 25       SLA L
06F8  CB 14       RL H
06FA  CB 25       SLA L
06FC  CB 14       RL H
06FE  4C          LD C,H
06FF  2A 57 D2    LD HL,($D257)
0702  CB 25       SLA L
0704  CB 14       RL H
0706  CB 25       SLA L
0708  CB 14       RL H
070A  CB 25       SLA L
070C  CB 14       RL H
070E  44          LD B,H
070F  ED 43 51 D2 LD ($D251),BC
0713  2A 54 D2    LD HL,($D254)
0716  22 69 D2    LD ($D269),HL
0719  2A 57 D2    LD HL,($D257)
071C  22 6B D2    LD ($D26B),HL
071F  C9          RET

; ==== sub_0720 (3 callers) ====
0720  FD CB 00 6E BIT 5,(IY+0)
0724  C8          RET Z
0725  F3          DI
0726  3E 04       LD A,$04
0728  32 FE FF    LD ($FFFE),A
072B  32 2F D2    LD ($D22F),A
072E  3E 05       LD A,$05
0730  32 FF FF    LD ($FFFF),A
0733  32 30 D2    LD ($D230),A
0736  FB          EI
0737  3A D5 D2    LD A,($D2D5)
073A  87          ADD A,A
073B  4F          LD C,A
073C  06 00       LD B,$00
073E  21 3D 34    LD HL,$343D
0741  09          ADD HL,BC
0742  7E          LD A,(HL)
0743  23          INC HL
0744  66          LD H,(HL)
0745  6F          LD L,A
0746  22 11 D2    LD ($D211),HL
0749  FD CB 02 46 BIT 0,(IY+2)
074D  CA D5 07    JP Z,$07D5
0750  FD CB 00 76 BIT 6,(IY+0)
0754  20 07       JR NZ,$075D
0756  06 00       LD B,$00
0758  0E 08       LD C,$08
075A  C3 6E 07    JP $076E
075D  3A 4B D2    LD A,($D24B)
0760  E6 1F       AND $1F
0762  C6 08       ADD A,$08
0764  0F          RRCA
0765  0F          RRCA
0766  0F          RRCA
0767  0F          RRCA
0768  0F          RRCA
0769  E6 01       AND $01
076B  06 00       LD B,$00
076D  4F          LD C,A
076E  CD 38 09    CALL $0938
0771  3A 4B D2    LD A,($D24B)
0774  FD CB 00 76 BIT 6,(IY+0)
0778  28 02       JR Z,$077C
077A  C6 08       ADD A,$08
077C  E6 1F       AND $1F
077E  CB 3F       SRL A
0780  CB 3F       SRL A
0782  CB 3F       SRL A
0784  4F          LD C,A
0785  06 00       LD B,$00
0787  ED 43 0F D2 LD ($D20F),BC
078B  D9          EXX
078C  11 80 D1    LD DE,$D180
078F  D9          EXX
0790  ED 5B 32 D2 LD DE,($D232)
0794  06 07       LD B,$07
0796  7E          LD A,(HL)
0797  D9          EXX
0798  4F          LD C,A
0799  06 00       LD B,$00
079B  2A 11 D2    LD HL,($D211)
079E  09          ADD HL,BC
079F  07          RLCA
07A0  07          RLCA
07A1  07          RLCA
07A2  07          RLCA
07A3  4F          LD C,A
07A4  E6 0F       AND $0F
07A6  47          LD B,A
07A7  79          LD A,C
07A8  A8          XOR B
07A9  4F          LD C,A
07AA  7E          LD A,(HL)
07AB  0F          RRCA
07AC  0F          RRCA
07AD  0F          RRCA
07AE  E6 10       AND $10
07B0  2A 0F D2    LD HL,($D20F)
07B3  09          ADD HL,BC
07B4  ED 4B 49 D2 LD BC,($D249)
07B8  09          ADD HL,BC
07B9  01 04 00    LD BC,$0004
07BC  ED A0       LDI
07BE  12          LD (DE),A
07BF  1C          INC E
07C0  09          ADD HL,BC
07C1  ED A0       LDI
07C3  12          LD (DE),A
07C4  1C          INC E
07C5  0C          INC C
07C6  09          ADD HL,BC
07C7  ED A0       LDI
07C9  12          LD (DE),A
07CA  1C          INC E
07CB  0C          INC C
07CC  09          ADD HL,BC
07CD  ED A0       LDI
07CF  12          LD (DE),A
07D0  1C          INC E
07D1  D9          EXX
07D2  19          ADD HL,DE
07D3  10 C1       DJNZ $0796
07D5  FD CB 02 4E BIT 1,(IY+2)
07D9  CA 3D 08    JP Z,$083D
07DC  FD CB 00 7E BIT 7,(IY+0)
07E0  20 07       JR NZ,$07E9
07E2  06 06       LD B,$06
07E4  0E 00       LD C,$00
07E6  C3 EC 07    JP $07EC
07E9  06 00       LD B,$00
07EB  48          LD C,B
07EC  CD 38 09    CALL $0938
07EF  3A 4C D2    LD A,($D24C)
07F2  E6 1F       AND $1F
07F4  CB 3F       SRL A
07F6  E6 FC       AND $FC
07F8  4F          LD C,A
07F9  06 00       LD B,$00
07FB  ED 43 0F D2 LD ($D20F),BC
07FF  D9          EXX
0800  11 00 D1    LD DE,$D100
0803  D9          EXX
0804  06 09       LD B,$09
0806  7E          LD A,(HL)
0807  D9          EXX
0808  4F          LD C,A
0809  06 00       LD B,$00
080B  2A 11 D2    LD HL,($D211)
080E  09          ADD HL,BC
080F  07          RLCA
0810  07          RLCA
0811  07          RLCA
0812  07          RLCA
0813  4F          LD C,A
0814  E6 0F       AND $0F
0816  47          LD B,A
0817  79          LD A,C
0818  A8          XOR B
0819  4F          LD C,A
081A  7E          LD A,(HL)
081B  0F          RRCA
081C  0F          RRCA
081D  0F          RRCA
081E  E6 10       AND $10
0820  2A 0F D2    LD HL,($D20F)
0823  09          ADD HL,BC
0824  ED 4B 49 D2 LD BC,($D249)
0828  09          ADD HL,BC
0829  ED A0       LDI
082B  12          LD (DE),A
082C  1C          INC E
082D  ED A0       LDI
082F  12          LD (DE),A
0830  1C          INC E
0831  ED A0       LDI
0833  12          LD (DE),A
0834  1C          INC E
0835  ED A0       LDI
0837  12          LD (DE),A
0838  1C          INC E
0839  D9          EXX
083A  23          INC HL
083B  10 C9       DJNZ $0806
083D  C9          RET

; ==== sub_083E (1 caller) ====
083E  FD CB 02 46 BIT 0,(IY+2)
0842  CA AC 08    JP Z,$08AC
0845  D9          EXX
0846  E5          PUSH HL
0847  D5          PUSH DE
0848  C5          PUSH BC
0849  3A 4C D2    LD A,($D24C)
084C  E6 F8       AND $F8
084E  06 00       LD B,$00
0850  87          ADD A,A
0851  CB 10       RL B
0853  87          ADD A,A
0854  CB 10       RL B
0856  87          ADD A,A
0857  CB 10       RL B
0859  4F          LD C,A
085A  3A 4B D2    LD A,($D24B)
085D  FD CB 00 76 BIT 6,(IY+0)
0861  28 02       JR Z,$0865
0863  C6 08       ADD A,$08
0865  E6 F8       AND $F8
0867  CB 3F       SRL A
0869  CB 3F       SRL A
086B  81          ADD A,C
086C  4F          LD C,A
086D  21 00 38    LD HL,$3800
0870  09          ADD HL,BC
0871  CB F4       SET 6,H
0873  01 40 00    LD BC,$0040
0876  16 7F       LD D,$7F
0878  1E 07       LD E,$07
087A  D9          EXX
087B  21 80 D1    LD HL,$D180
087E  3A 4C D2    LD A,($D24C)
0881  E6 1F       AND $1F
0883  CB 3F       SRL A
0885  CB 3F       SRL A
0887  CB 3F       SRL A
0889  4F          LD C,A
088A  06 00       LD B,$00
088C  09          ADD HL,BC
088D  09          ADD HL,BC
088E  06 32       LD B,$32
0890  0E BE       LD C,$BE
0892  D9          EXX
0893  7D          LD A,L
0894  D3 BF       OUT ($BF),A
0896  7C          LD A,H
0897  D3 BF       OUT ($BF),A
0899  09          ADD HL,BC
089A  7C          LD A,H
089B  BA          CP D
089C  D2 33 09    JP NC,$0933
089F  D9          EXX
08A0  ED A3       OUTI
08A2  ED A3       OUTI
08A4  C2 92 08    JP NZ,$0892
08A7  D9          EXX
08A8  C1          POP BC
08A9  D1          POP DE
08AA  E1          POP HL
08AB  D9          EXX
08AC  FD CB 02 4E BIT 1,(IY+2)
08B0  CA 32 09    JP Z,$0932
08B3  3A 4C D2    LD A,($D24C)
08B6  06 00       LD B,$00
08B8  CB 3F       SRL A
08BA  CB 3F       SRL A
08BC  CB 3F       SRL A
08BE  FD CB 00 7E BIT 7,(IY+0)
08C2  20 02       JR NZ,$08C6
08C4  C6 18       ADD A,$18
08C6  FE 1C       CP $1C
08C8  38 02       JR C,$08CC
08CA  D6 1C       SUB $1C
08CC  87          ADD A,A
08CD  87          ADD A,A
08CE  87          ADD A,A
08CF  87          ADD A,A
08D0  CB 10       RL B
08D2  87          ADD A,A
08D3  CB 10       RL B
08D5  87          ADD A,A
08D6  CB 10       RL B
08D8  4F          LD C,A
08D9  3A 4B D2    LD A,($D24B)
08DC  C6 08       ADD A,$08
08DE  E6 F8       AND $F8
08E0  CB 3F       SRL A
08E2  CB 3F       SRL A
08E4  81          ADD A,C
08E5  4F          LD C,A
08E6  21 00 38    LD HL,$3800
08E9  09          ADD HL,BC
08EA  CB F4       SET 6,H
08EC  EB          EX DE,HL
08ED  21 00 D1    LD HL,$D100
08F0  3A 4B D2    LD A,($D24B)
08F3  E6 1F       AND $1F
08F5  C6 08       ADD A,$08
08F7  CB 3F       SRL A
08F9  CB 3F       SRL A
08FB  CB 3F       SRL A
08FD  4F          LD C,A
08FE  06 00       LD B,$00
0900  09          ADD HL,BC
0901  09          ADD HL,BC
0902  7B          LD A,E
0903  E6 C0       AND $C0
0905  32 0F D2    LD ($D20F),A
0908  7B          LD A,E
0909  D3 BF       OUT ($BF),A
090B  E6 3F       AND $3F
090D  5F          LD E,A
090E  7A          LD A,D
090F  D3 BF       OUT ($BF),A
0911  06 3E       LD B,$3E
0913  0E BE       LD C,$BE
0915  CB 73       BIT 6,E
0917  20 0A       JR NZ,$0923
0919  1C          INC E
091A  1C          INC E
091B  ED A3       OUTI
091D  ED A3       OUTI
091F  C2 15 09    JP NZ,$0915
0922  C9          RET
0923  3A 0F D2    LD A,($D20F)
0926  D3 BF       OUT ($BF),A
0928  7A          LD A,D
0929  D3 BF       OUT ($BF),A
092B  ED A3       OUTI
092D  ED A3       OUTI
092F  C2 2B 09    JP NZ,$092B
0932  C9          RET
0933  93          SUB E
0934  67          LD H,A
0935  C3 9F 08    JP $089F

; ==== sub_0938 (3 callers) ====
0938  3A 32 D2    LD A,($D232)
093B  07          RLCA
093C  38 0C       JR C,$094A
093E  07          RLCA
093F  38 1F       JR C,$0960
0941  07          RLCA
0942  38 36       JR C,$097A
0944  07          RLCA
0945  38 51       JR C,$0998
0947  C3 BA 09    JP $09BA
094A  3A 52 D2    LD A,($D252)
094D  80          ADD A,B
094E  1E 00       LD E,$00
0950  CB 3F       SRL A
0952  CB 1B       RR E
0954  57          LD D,A
0955  3A 51 D2    LD A,($D251)
0958  81          ADD A,C
0959  83          ADD A,E
095A  5F          LD E,A
095B  21 00 C0    LD HL,$C000
095E  19          ADD HL,DE
095F  C9          RET
0960  3A 52 D2    LD A,($D252)
0963  80          ADD A,B
0964  1E 00       LD E,$00
0966  CB 3F       SRL A
0968  CB 1B       RR E
096A  CB 3F       SRL A
096C  CB 1B       RR E
096E  57          LD D,A
096F  3A 51 D2    LD A,($D251)
0972  81          ADD A,C
0973  83          ADD A,E
0974  5F          LD E,A
0975  21 00 C0    LD HL,$C000
0978  19          ADD HL,DE
0979  C9          RET
097A  3A 52 D2    LD A,($D252)
097D  80          ADD A,B
097E  1E 00       LD E,$00
0980  CB 3F       SRL A
0982  CB 1B       RR E
0984  CB 3F       SRL A
0986  CB 1B       RR E
0988  CB 3F       SRL A
098A  CB 1B       RR E
098C  57          LD D,A
098D  3A 51 D2    LD A,($D251)
0990  81          ADD A,C
0991  83          ADD A,E
0992  5F          LD E,A
0993  21 00 C0    LD HL,$C000
0996  19          ADD HL,DE
0997  C9          RET
0998  3A 52 D2    LD A,($D252)
099B  80          ADD A,B
099C  1E 00       LD E,$00
099E  CB 3F       SRL A
09A0  CB 1B       RR E
09A2  CB 3F       SRL A
09A4  CB 1B       RR E
09A6  CB 3F       SRL A
09A8  CB 1B       RR E
09AA  CB 3F       SRL A
09AC  CB 1B       RR E
09AE  57          LD D,A
09AF  3A 51 D2    LD A,($D251)
09B2  81          ADD A,C
09B3  83          ADD A,E
09B4  5F          LD E,A
09B5  21 00 C0    LD HL,$C000
09B8  19          ADD HL,DE
09B9  C9          RET
09BA  3A 52 D2    LD A,($D252)
09BD  80          ADD A,B
09BE  57          LD D,A
09BF  3A 51 D2    LD A,($D251)
09C2  81          ADD A,C
09C3  5F          LD E,A
09C4  21 00 C0    LD HL,$C000
09C7  19          ADD HL,DE
09C8  C9          RET

; ==== sub_09C9 (1 caller) ====
09C9  F3          DI
09CA  3E 04       LD A,$04
09CC  32 FE FF    LD ($FFFE),A
09CF  32 2F D2    LD ($D22F),A
09D2  3E 05       LD A,$05
09D4  32 FF FF    LD ($FFFF),A
09D7  32 30 D2    LD ($D230),A
09DA  01 00 00    LD BC,$0000
09DD  CD 38 09    CALL $0938
09E0  11 00 38    LD DE,$3800
09E3  06 06       LD B,$06
09E5  C5          PUSH BC
09E6  E5          PUSH HL
09E7  D5          PUSH DE
09E8  06 08       LD B,$08
09EA  C5          PUSH BC
09EB  E5          PUSH HL
09EC  D5          PUSH DE
09ED  7E          LD A,(HL)
09EE  D9          EXX
09EF  5F          LD E,A
09F0  3A D5 D2    LD A,($D2D5)
09F3  87          ADD A,A
09F4  4F          LD C,A
09F5  06 00       LD B,$00
09F7  21 3D 34    LD HL,$343D
09FA  09          ADD HL,BC
09FB  7E          LD A,(HL)
09FC  23          INC HL
09FD  66          LD H,(HL)
09FE  6F          LD L,A
09FF  16 00       LD D,$00
0A01  19          ADD HL,DE
0A02  7E          LD A,(HL)
0A03  0F          RRCA
0A04  0F          RRCA
0A05  0F          RRCA
0A06  E6 10       AND $10
0A08  4F          LD C,A
0A09  D9          EXX
0A0A  6E          LD L,(HL)
0A0B  26 00       LD H,$00
0A0D  29          ADD HL,HL
0A0E  29          ADD HL,HL
0A0F  29          ADD HL,HL
0A10  29          ADD HL,HL
0A11  ED 4B 49 D2 LD BC,($D249)
0A15  09          ADD HL,BC
0A16  EB          EX DE,HL
0A17  06 04       LD B,$04
0A19  7D          LD A,L
0A1A  D3 BF       OUT ($BF),A
0A1C  7C          LD A,H
0A1D  F6 40       OR $40
0A1F  D3 BF       OUT ($BF),A
0A21  1A          LD A,(DE)
0A22  D3 BE       OUT ($BE),A
0A24  13          INC DE
0A25  D9          EXX
0A26  79          LD A,C
0A27  D9          EXX
0A28  D3 BE       OUT ($BE),A
0A2A  00          NOP
0A2B  00          NOP
0A2C  1A          LD A,(DE)
0A2D  D3 BE       OUT ($BE),A
0A2F  13          INC DE
0A30  D9          EXX
0A31  79          LD A,C
0A32  D9          EXX
0A33  D3 BE       OUT ($BE),A
0A35  00          NOP
0A36  00          NOP
0A37  1A          LD A,(DE)
0A38  D3 BE       OUT ($BE),A
0A3A  13          INC DE
0A3B  D9          EXX
0A3C  79          LD A,C
0A3D  D9          EXX
0A3E  D3 BE       OUT ($BE),A
0A40  00          NOP
0A41  00          NOP
0A42  1A          LD A,(DE)
0A43  D3 BE       OUT ($BE),A
0A45  13          INC DE
0A46  D9          EXX
0A47  79          LD A,C
0A48  D9          EXX
0A49  D3 BE       OUT ($BE),A
0A4B  78          LD A,B
0A4C  01 40 00    LD BC,$0040
0A4F  09          ADD HL,BC
0A50  47          LD B,A
0A51  10 C6       DJNZ $0A19
0A53  D1          POP DE
0A54  E1          POP HL
0A55  23          INC HL
0A56  01 08 00    LD BC,$0008
0A59  EB          EX DE,HL
0A5A  09          ADD HL,BC
0A5B  EB          EX DE,HL
0A5C  C1          POP BC
0A5D  10 8B       DJNZ $09EA
0A5F  D1          POP DE
0A60  E1          POP HL
0A61  ED 4B 32 D2 LD BC,($D232)
0A65  09          ADD HL,BC
0A66  EB          EX DE,HL
0A67  01 00 01    LD BC,$0100
0A6A  09          ADD HL,BC
0A6B  EB          EX DE,HL
0A6C  C1          POP BC
0A6D  05          DEC B
0A6E  C2 E5 09    JP NZ,$09E5
0A71  FB          EI
0A72  C9          RET

; ==== sub_0A73 (1 caller) ====
0A73  11 00 C0    LD DE,$C000
0A76  7E          LD A,(HL)
0A77  2F          CPL
0A78  FD 77 01    LD (IY+1),A
0A7B  7E          LD A,(HL)
0A7C  FD BE 01    CP (IY+1)
0A7F  28 0D       JR Z,$0A8E
0A81  12          LD (DE),A
0A82  FD 77 01    LD (IY+1),A
0A85  23          INC HL
0A86  13          INC DE
0A87  0B          DEC BC
0A88  78          LD A,B
0A89  B1          OR C
0A8A  C2 7B 0A    JP NZ,$0A7B
0A8D  C9          RET
0A8E  0B          DEC BC
0A8F  78          LD A,B
0A90  B1          OR C
0A91  C8          RET Z
0A92  7E          LD A,(HL)
0A93  23          INC HL
0A94  C5          PUSH BC
0A95  46          LD B,(HL)
0A96  12          LD (DE),A
0A97  13          INC DE
0A98  10 FC       DJNZ $0A96
0A9A  C1          POP BC
0A9B  23          INC HL
0A9C  0B          DEC BC
0A9D  78          LD A,B
0A9E  B1          OR C
0A9F  C2 76 0A    JP NZ,$0A76
0AA2  C9          RET

; ==== palette_fade  $0AA3  (7 callers) — cross-fade the CRAM palette toward a target index over 16 steps, interpolating the RAM working palette at $D3BD and applying it each vblank (CALL $0327). ====
0AA3  21 16 16    LD HL,$1616
0AA6  CD B7 0A    CALL $0AB7
0AA9  C9          RET
0AAA  .byte C9                                              ; .

; ==== sub_0AAB (3 callers) ====
0AAB  E5          PUSH HL
0AAC  21 16 16    LD HL,$1616
0AAF  22 2C D2    LD ($D22C),HL
0AB2  E1          POP HL
0AB3  CD B7 0A    CALL $0AB7
0AB6  C9          RET

; ==== sub_0AB7 (5 callers) ====
0AB7  22 15 D2    LD ($D215),HL
0ABA  3E 08       LD A,$08
0ABC  32 FE FF    LD ($FFFE),A
0ABF  32 2F D2    LD ($D22F),A
0AC2  11 BD D3    LD DE,$D3BD
0AC5  3A 2C D2    LD A,($D22C)
0AC8  CD 1E 0B    CALL $0B1E
0ACB  3A 2D D2    LD A,($D22D)
0ACE  CD 1E 0B    CALL $0B1E
0AD1  21 FF FE    LD HL,$FEFF
0AD4  22 2C D2    LD ($D22C),HL
0AD7  FD 4E 0A    LD C,(IY+10)
0ADA  3A 1A D2    LD A,($D21A)
0ADD  F6 40       OR $40
0ADF  32 1A D2    LD ($D21A),A
0AE2  FD CB 00 86 RES 0,(IY+0)
0AE6  CD 27 03    CALL $0327
0AE9  FD 71 0A    LD (IY+10),C
0AEC  06 03       LD B,$03
0AEE  CD 31 0B    CALL $0B31
0AF1  10 FB       DJNZ $0AEE
0AF3  06 10       LD B,$10
0AF5  C5          PUSH BC
0AF6  11 BD D3    LD DE,$D3BD
0AF9  3A 15 D2    LD A,($D215)
0AFC  CD 3F 0B    CALL $0B3F
0AFF  3A 16 D2    LD A,($D216)
0B02  CD 3F 0B    CALL $0B3F
0B05  06 04       LD B,$04
0B07  CD 31 0B    CALL $0B31
0B0A  10 FB       DJNZ $0B07
0B0C  C1          POP BC
0B0D  10 E6       DJNZ $0AF5
0B0F  2A 15 D2    LD HL,($D215)
0B12  22 2C D2    LD ($D22C),HL
0B15  3E 01       LD A,$01
0B17  32 FE FF    LD ($FFFE),A
0B1A  32 2F D2    LD ($D22F),A
0B1D  C9          RET

; ==== sub_0B1E (2 callers) ====
0B1E  87          ADD A,A
0B1F  6F          LD L,A
0B20  26 00       LD H,$00
0B22  01 00 74    LD BC,$7400
0B25  09          ADD HL,BC
0B26  7E          LD A,(HL)
0B27  23          INC HL
0B28  66          LD H,(HL)
0B29  6F          LD L,A
0B2A  09          ADD HL,BC
0B2B  01 20 00    LD BC,$0020
0B2E  ED B0       LDIR
0B30  C9          RET

; ==== sub_0B31 (2 callers) ====
0B31  FD 7E 0A    LD A,(IY+10)
0B34  FD CB 00 86 RES 0,(IY+0)
0B38  CD 27 03    CALL $0327
0B3B  FD 77 0A    LD (IY+10),A
0B3E  C9          RET

; ==== sub_0B3F (2 callers) ====
0B3F  87          ADD A,A
0B40  6F          LD L,A
0B41  26 00       LD H,$00
0B43  01 00 74    LD BC,$7400
0B46  09          ADD HL,BC
0B47  7E          LD A,(HL)
0B48  23          INC HL
0B49  66          LD H,(HL)
0B4A  6F          LD L,A
0B4B  09          ADD HL,BC
0B4C  06 10       LD B,$10
0B4E  C5          PUSH BC
0B4F  7E          LD A,(HL)
0B50  E6 0F       AND $0F
0B52  4F          LD C,A
0B53  1A          LD A,(DE)
0B54  E6 0F       AND $0F
0B56  B9          CP C
0B57  28 06       JR Z,$0B5F
0B59  38 03       JR C,$0B5E
0B5B  3D          DEC A
0B5C  18 01       JR $0B5F
0B5E  3C          INC A
0B5F  47          LD B,A
0B60  7E          LD A,(HL)
0B61  E6 F0       AND $F0
0B63  4F          LD C,A
0B64  1A          LD A,(DE)
0B65  E6 F0       AND $F0
0B67  B9          CP C
0B68  28 08       JR Z,$0B72
0B6A  38 04       JR C,$0B70
0B6C  D6 10       SUB $10
0B6E  18 02       JR $0B72
0B70  C6 10       ADD A,$10
0B72  B0          OR B
0B73  12          LD (DE),A
0B74  13          INC DE
0B75  23          INC HL
0B76  7E          LD A,(HL)
0B77  E6 0F       AND $0F
0B79  4F          LD C,A
0B7A  1A          LD A,(DE)
0B7B  E6 0F       AND $0F
0B7D  B9          CP C
0B7E  28 06       JR Z,$0B86
0B80  38 03       JR C,$0B85
0B82  3D          DEC A
0B83  18 01       JR $0B86
0B85  3C          INC A
0B86  12          LD (DE),A
0B87  13          INC DE
0B88  23          INC HL
0B89  C1          POP BC
0B8A  10 C2       DJNZ $0B4E
0B8C  C9          RET

; ==== sub_0B8D (3 callers) ====
0B8D  3A 38 D2    LD A,($D238)
0B90  4F          LD C,A
0B91  CB 3F       SRL A
0B93  CB 3F       SRL A
0B95  CB 3F       SRL A
0B97  5F          LD E,A
0B98  16 00       LD D,$00
0B9A  19          ADD HL,DE
0B9B  79          LD A,C
0B9C  0E 01       LD C,$01
0B9E  E6 07       AND $07
0BA0  C8          RET Z
0BA1  47          LD B,A
0BA2  79          LD A,C
0BA3  07          RLCA
0BA4  10 FD       DJNZ $0BA3
0BA6  4F          LD C,A
0BA7  C9          RET
0BA8  .byte F3 3E 05 32 FE FF 3A 24 D2 E6 0F 87 87 87 5F 16 ; .>.2..:$......_.
0BB8  .byte 00 19 EB 01 80 2B 09 7D D3 BF 7C F6 40 D3 BF 06 ; .....+.}..|.@...
0BC8  .byte 04 1A D3 BE 00 00 13 1A D3 BE 13 10 F4 3A 2F D2 ; .............:/.
0BD8  .byte 32 FE FF FB C9                                  ; 2....

; ==== sub_0BDD (1 caller) ====
0BDD  AF          XOR A
0BDE  32 4B D2    LD ($D24B),A
0BE1  32 4C D2    LD ($D24C),A
0BE4  32 00 D3    LD ($D300),A
0BE7  FD CB 05 4E BIT 1,(IY+5)
0BEB  28 0E       JR Z,$0BFB
0BED  0E 00       LD C,$00
0BEF  3A 05 D3    LD A,($D305)
0BF2  0F          RRCA
0BF3  30 02       JR NC,$0BF7
0BF5  0E 06       LD C,$06
0BF7  79          LD A,C
0BF8  32 38 D2    LD ($D238),A
0BFB  3E FF       LD A,$FF
0BFD  32 17 D2    LD ($D217),A
0C00  0E 01       LD C,$01
0C02  3A 38 D2    LD A,($D238)
0C05  FE 12       CP $12
0C07  D0          RET NC
0C08  FE 09       CP $09
0C0A  38 02       JR C,$0C0E
0C0C  0E 02       LD C,$02
0C0E  3A 17 D2    LD A,($D217)
0C11  B9          CP C
0C12  CA D9 0C    JP Z,$0CD9
0C15  79          LD A,C
0C16  32 17 D2    LD ($D217),A
0C19  3D          DEC A
0C1A  20 5E       JR NZ,$0C7A
0C1C  3A 1A D2    LD A,($D21A)
0C1F  E6 BF       AND $BF
0C21  32 1A D2    LD ($D21A),A
0C24  FD CB 00 86 RES 0,(IY+0)
0C28  CD 27 03    CALL $0327
0C2B  21 00 00    LD HL,$0000
0C2E  11 00 00    LD DE,$0000
0C31  3E 0C       LD A,$0C
0C33  CD 06 04    CALL $0406
0C36  21 D0 4A    LD HL,$4AD0
0C39  11 00 20    LD DE,$2000
0C3C  3E 09       LD A,$09
0C3E  CD 06 04    CALL $0406
0C41  21 54 B3    LD HL,$B354
0C44  11 00 30    LD DE,$3000
0C47  3E 09       LD A,$09
0C49  CD 06 04    CALL $0406
0C4C  21 62 69    LD HL,$6962
0C4F  01 9B 01    LD BC,$019B
0C52  11 00 38    LD DE,$3800
0C55  3E 10       LD A,$10
0C57  32 0F D2    LD ($D20F),A
0C5A  3E 05       LD A,$05
0C5C  CD 02 05    CALL $0502
0C5F  21 FD 6A    LD HL,$6AFD
0C62  01 70 01    LD BC,$0170
0C65  11 00 38    LD DE,$3800
0C68  3E 00       LD A,$00
0C6A  32 0F D2    LD ($D20F),A
0C6D  3E 05       LD A,$05
0C6F  CD 02 05    CALL $0502
0C72  21 0A 0B    LD HL,$0B0A
0C75  CD AB 0A    CALL $0AAB
0C78  18 5C       JR $0CD6
0C7A  3A 1A D2    LD A,($D21A)
0C7D  E6 BF       AND $BF
0C7F  32 1A D2    LD ($D21A),A
0C82  FD CB 00 86 RES 0,(IY+0)
0C86  CD 27 03    CALL $0327
0C89  21 1A 17    LD HL,$171A
0C8C  11 00 00    LD DE,$0000
0C8F  3E 0C       LD A,$0C
0C91  CD 06 04    CALL $0406
0C94  21 A7 51    LD HL,$51A7
0C97  11 00 20    LD DE,$2000
0C9A  3E 09       LD A,$09
0C9C  CD 06 04    CALL $0406
0C9F  21 54 B3    LD HL,$B354
0CA2  11 00 30    LD DE,$3000
0CA5  3E 09       LD A,$09
0CA7  CD 06 04    CALL $0406
0CAA  21 6D 6C    LD HL,$6C6D
0CAD  01 56 01    LD BC,$0156
0CB0  11 00 38    LD DE,$3800
0CB3  3E 10       LD A,$10
0CB5  32 0F D2    LD ($D20F),A
0CB8  3E 05       LD A,$05
0CBA  CD 02 05    CALL $0502
0CBD  21 C3 6D    LD HL,$6DC3
0CC0  01 98 01    LD BC,$0198
0CC3  11 00 38    LD DE,$3800
0CC6  3E 00       LD A,$00
0CC8  32 0F D2    LD ($D20F),A
0CCB  3E 05       LD A,$05
0CCD  CD 02 05    CALL $0502
0CD0  21 0C 0D    LD HL,$0D0C
0CD3  CD AB 0A    CALL $0AAB
0CD6  3E 07       LD A,$07
0CD8  DF          RST $18
0CD9  CD 23 0E    CALL $0E23
0CDC  3A 38 D2    LD A,($D238)
0CDF  87          ADD A,A
0CE0  4F          LD C,A
0CE1  06 00       LD B,$00
0CE3  21 63 11    LD HL,$1163
0CE6  09          ADD HL,BC
0CE7  7E          LD A,(HL)
0CE8  23          INC HL
0CE9  66          LD H,(HL)
0CEA  6F          LD L,A
0CEB  3E 10       LD A,$10
0CED  32 0F D2    LD ($D20F),A
0CF0  CD 12 06    CALL $0612
0CF3  3A 38 D2    LD A,($D238)
0CF6  4F          LD C,A
0CF7  87          ADD A,A
0CF8  81          ADD A,C
0CF9  5F          LD E,A
0CFA  16 00       LD D,$00
0CFC  21 A8 0E    LD HL,$0EA8
0CFF  19          ADD HL,DE
0D00  5E          LD E,(HL)
0D01  23          INC HL
0D02  56          LD D,(HL)
0D03  23          INC HL
0D04  ED 53 11 D2 LD ($D211),DE
0D08  7E          LD A,(HL)
0D09  A7          AND A
0D0A  28 0E       JR Z,$0D1A
0D0C  3D          DEC A
0D0D  87          ADD A,A
0D0E  5F          LD E,A
0D0F  16 00       LD D,$00
0D11  21 5B 11    LD HL,$115B
0D14  19          ADD HL,DE
0D15  7E          LD A,(HL)
0D16  23          INC HL
0D17  66          LD H,(HL)
0D18  6F          LD L,A
0D19  E9          JP (HL)
0D1A  3E 01       LD A,$01
0D1C  32 0F D2    LD ($D20F),A
0D1F  01 2C 01    LD BC,$012C
0D22  C5          PUSH BC
0D23  CD 23 0E    CALL $0E23
0D26  3A 0F D2    LD A,($D20F)
0D29  3D          DEC A
0D2A  32 0F D2    LD ($D20F),A
0D2D  20 22       JR NZ,$0D51
0D2F  2A 11 D2    LD HL,($D211)
0D32  5E          LD E,(HL)
0D33  23          INC HL
0D34  56          LD D,(HL)
0D35  23          INC HL
0D36  4E          LD C,(HL)
0D37  23          INC HL
0D38  46          LD B,(HL)
0D39  23          INC HL
0D3A  ED 43 15 D2 LD ($D215),BC
0D3E  7E          LD A,(HL)
0D3F  23          INC HL
0D40  A7          AND A
0D41  20 04       JR NZ,$0D47
0D43  EB          EX DE,HL
0D44  C3 32 0D    JP $0D32
0D47  32 0F D2    LD ($D20F),A
0D4A  22 11 D2    LD ($D211),HL
0D4D  ED 53 13 D2 LD ($D213),DE
0D51  2A 15 D2    LD HL,($D215)
0D54  E5          PUSH HL
0D55  5C          LD E,H
0D56  26 00       LD H,$00
0D58  54          LD D,H
0D59  ED 4B 13 D2 LD BC,($D213)
0D5D  CD 07 2F    CALL $2F07
0D60  E1          POP HL
0D61  22 15 D2    LD ($D215),HL
0D64  C1          POP BC
0D65  0B          DEC BC
0D66  78          LD A,B
0D67  B1          OR C
0D68  C8          RET Z
0D69  FD 7E 03    LD A,(IY+3)
0D6C  E6 B0       AND $B0
0D6E  FE B0       CP $B0
0D70  CA 22 0D    JP Z,$0D22
0D73  C0          RET NZ
0D74  37          SCF
0D75  C9          RET
0D76  .byte 21 00 00 22 0F D2 21 DC 00 11 32 00 06 00 CD 23 ; !.."..!...2....#
0D86  .byte 0E FD 7E 03 FE FF C2 1A 0D C5 01 0F 0E CD 7A 0E ; ..~...........z.
0D96  .byte C1 2B 10 EA 21 00 00 22 0F D2 21 F6 FF 11 50 00 ; .+..!.."..!...P.
0DA6  .byte 06 72 CD 23 0E FD 7E 03 FE FF C2 1A 0D C5 01 17 ; .r.#..~.........
0DB6  .byte 0E CD 7A 0E C1 23 10 EA C3 1A 0D 21 00 00 22 0F ; ..z..#.....!..".
0DC6  .byte D2 21 90 00 11 C0 00 06 80 CD 23 0E FD 7E 03 FE ; .!........#..~..
0DD6  .byte FF C2 1A 0D C5 01 1F 0E CD 7A 0E C1 1B 10 EA C3 ; .........z......
0DE6  .byte 1A 0D 21 00 00 22 0F D2 21 88 00 11 00 00 06 30 ; ..!.."..!......0
0DF6  .byte CD 23 0E FD 7E 03 FE FF C2 1A 0D C5 01 1F 0E CD ; .#..~...........
0E06  .byte 7A 0E C1 13 10 EA C3 1A 0D 83 10 04 01 95 10 04 ; z...............
0E16  .byte 00 A7 10 04 01 B9 10 04 00 DD 10 04 00          ; .............

; ==== sub_0E23 (2 callers) ====
0E23  E5          PUSH HL
0E24  D5          PUSH DE
0E25  C5          PUSH BC
0E26  2A 0F D2    LD HL,($D20F)
0E29  E5          PUSH HL
0E2A  FD CB 00 86 RES 0,(IY+0)
0E2E  CD 27 03    CALL $0327
0E31  FD 36 0A 00 LD (IY+10),$00
0E35  3A 40 D2    LD A,($D240)
0E38  6F          LD L,A
0E39  26 00       LD H,$00
0E3B  0E 0A       LD C,$0A
0E3D  CD 72 06    CALL $0672
0E40  7D          LD A,L
0E41  87          ADD A,A
0E42  C6 80       ADD A,$80
0E44  32 BF D2    LD ($D2BF),A
0E47  0E 0A       LD C,$0A
0E49  CD 5F 06    CALL $065F
0E4C  EB          EX DE,HL
0E4D  3A 40 D2    LD A,($D240)
0E50  6F          LD L,A
0E51  26 00       LD H,$00
0E53  A7          AND A
0E54  ED 52       SBC HL,DE
0E56  7D          LD A,L
0E57  87          ADD A,A
0E58  C6 80       ADD A,$80
0E5A  32 C0 D2    LD ($D2C0),A
0E5D  3E FF       LD A,$FF
0E5F  32 C1 D2    LD ($D2C1),A
0E62  0E 48       LD C,$48
0E64  06 97       LD B,$97
0E66  21 00 D0    LD HL,$D000
0E69  11 BF D2    LD DE,$D2BF
0E6C  CD A8 2F    CALL $2FA8
0E6F  22 36 D2    LD ($D236),HL
0E72  E1          POP HL
0E73  22 0F D2    LD ($D20F),HL
0E76  C1          POP BC
0E77  D1          POP DE
0E78  E1          POP HL
0E79  C9          RET
0E7A  .byte E5 D5 69 60 3A 10 D2 87 87 5F 16 00 19 4E 23 46 ; ..i`:...._...N#F
0E8A  .byte 23 3A 0F D2 BE 38 09 23 7E 32 10 D2 AF 32 0F D2 ; #:...8.#~2...2..
0E9A  .byte D1 E1 E5 D5 CD 07 2F 2A 0F D2 34 D1 E1 C9 DE 0E ; ....../*..4.....
0EAA  .byte 00 ED 0E 00 38 0F 01 FC 0E 00 0B 0F 00 D8 0F 02 ; ....8...........
0EBA  .byte 1A 0F 00 29 0F 00 E2 0F 03 65 0F 00 74 0F 00 EC ; ...).....e..t...
0ECA  .byte 0F 00 83 0F 00 92 0F 00 F6 0F 00 A1 0F 00 B0 0F ; ................
0EDA  .byte 00 B0 0F 00 17 10 60 68 1E 05 10 60 68 1E DE 0E ; ......`h...`h...
0EEA  .byte 00 00 00 29 10 60 60 1E 05 10 60 60 1E ED 0E 00 ; ...).``...``....
0EFA  .byte 00 00 3B 10 70 60 1E 05 10 70 60 1E FC 0E 00 00 ; ..;.p`...p`.....
0F0A  .byte 00 4D 10 90 50 1E 05 10 90 50 1E 0B 0F 00 00 00 ; .M..P....P......
0F1A  .byte 5F 10 80 48 1E 05 10 80 48 1E 1A 0F 00 00 00 71 ; _..H....H......q
0F2A  .byte 10 80 30 1E 05 10 80 30 1E 29 0F 00 00 00 DD 10 ; ..0....0.)......
0F3A  .byte 68 50 08 DD 10 68 50 08 DD 10 68 4E 08 DD 10 68 ; hP...hP...hN...h
0F4A  .byte 4E 08 DD 10 68 4D 08 DD 10 68 4D 08 DD 10 68 4E ; N...hM...hM...hN
0F5A  .byte 08 DD 10 68 4E 08 38 0F 00 00 00 EF 10 58 68 1E ; ...hN.8......Xh.
0F6A  .byte 05 10 58 68 1E 65 0F 00 00 00 01 11 68 78 1E 05 ; ..Xh.e......hx..
0F7A  .byte 10 68 78 1E 74 0F 00 00 00 13 11 70 58 1E 05 10 ; .hx.t......pX...
0F8A  .byte 70 58 1E 83 0F 00 00 00 25 11 78 48 1E 05 10 78 ; pX......%.xH...x
0F9A  .byte 48 1E 92 0F 00 00 00 37 11 68 28 1E 05 10 68 28 ; H......7.h(...h(
0FAA  .byte 1E A1 0F 00 00 00 49 11 80 28 1E 49 11 80 26 08 ; ......I..(.I..&.
0FBA  .byte 49 11 80 26 08 49 11 80 25 08 49 11 80 25 08 49 ; I..&.I..%.I..%.I
0FCA  .byte 11 80 26 08 49 11 80 26 08 B0 0F 00 00 00 DD 10 ; ..&.I..&........
0FDA  .byte 90 40 08 D8 0F 00 00 00 DD 10 88 30 08 E2 0F 00 ; .@.........0....
0FEA  .byte 00 00 DD 10 70 60 08 EC 0F 00 00 00 83 10 68 40 ; ....p`........h@
0FFA  .byte 08 95 10 68 40 08 F6 0F 00 00 00 FF FF FF FF FF ; ...h@...........
100A  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF 00 02 FF ; ................
101A  .byte FF FF FF FE 22 24 26 28 FF FF FF FF FF FF FF 04 ; ...."$&(........
102A  .byte 06 08 FF FF FF FF FF FF FF FF FF FF FF FF FF FF ; ................
103A  .byte FF 40 42 44 46 48 FF FF FF FF FF FF FF FF FF FF ; .@BDFH..........
104A  .byte FF FF FF 4A 4C FF FF FF FF 6A 6C FF FF FF FF FF ; ...JL....jl.....
105A  .byte FF FF FF FF FF 60 62 64 66 FF FF FF FF FF FF FF ; .....`bdf.......
106A  .byte FF FF FF FF FF FF FF FE FE 0E FF FF FF 2A 2C 2E ; .............*,.
107A  .byte FF FF FF FF FF FF FF FF FF 10 12 14 16 FF FF 30 ; ...............0
108A  .byte 32 34 36 FF FF FF FF FF FF FF FF 10 12 14 18 FF ; 246.............
109A  .byte FF 30 32 34 38 FF FF FF FF FF FF FF FF 50 54 56 ; .0248........PTV
10AA  .byte 58 FF FF 70 74 76 78 FF FF FF FF FF FF FF FF 52 ; X..ptvx........R
10BA  .byte 54 56 58 FF FF 72 74 76 78 FF FF FF FF FF FF FF ; TVX..rtvx.......
10CA  .byte FF 50 54 56 58 FF FF 70 74 76 78 FF FF FF FF FF ; .PTVX..ptvx.....
10DA  .byte FF FF FF 5A 5C 5E FF FF FF 7A 7C 7E FF FF FF FF ; ...Z\^...z|~....
10EA  .byte FF FF FF FF FF 00 02 FF FF FF FF 20 22 04 FF FF ; ........... "...
10FA  .byte FF FF FF FF FF FF FF 0A 0C 0E FF FF FF 2A 2C 2E ; .............*,.
110A  .byte FF FF FF FF FF FF FF FF FF 68 6A 6C FF FF FF FE ; .........hjl....
111A  .byte FE 6E FF FF FF FF FF FF FF FF FF 06 08 4A 4C FF ; .n...........JL.
112A  .byte FF FE FE 4E 3E FF FF FE 40 42 44 FF FF 60 62 64 ; ...N>...@BD..`bd
113A  .byte 66 FF FF FF FF FF FF FF FF FF FF FF FF FF FF 46 ; f..............F
114A  .byte 48 26 28 FF FF 1A 1C 3A 3C FF FF FF FF FF FF FF ; H&(....:<.......
115A  .byte FF 76 0D C1 0D E8 0D 76 0D 87 11 87 11 87 11 96 ; .v.....v........
116A  .byte 11 96 11 96 11 A5 11 A5 11 A5 11 B4 11 B4 11 B4 ; ................
117A  .byte 11 C3 11 C3 11 C3 11 D2 11 D2 11 D2 11 0C 13 46 ; ...............F
118A  .byte 62 44 44 51 EB 47 40 43 43 EB EB FF 0C 13 35 62 ; bDDQ.G@CC.....5b
119A  .byte 40 37 46 44 EB EB EB EB EB EB FF 0C 13 41 81 51 ; @7FD.........A.Q
11AA  .byte 46 43 44 EB EB EB EB EB EB FF 0C 13 6F 1E 1F DE ; FCD.........o...
11BA  .byte 9F 5E 7F AF 4F EB EB EB FF 0C 13 AE 2E 9F 1E 8F ; .^..O...........
11CA  .byte EB 1F 9F 1E 5E 7F EB FF 0C 13 AE 6E DE EB 1F 1E ; ....^......n....
11DA  .byte AE 3E EB EB EB EB FF                            ; .>.....

; ==== sub_11E1 (1 caller) ====
11E1  3A 1A D2    LD A,($D21A)
11E4  E6 BF       AND $BF
11E6  32 1A D2    LD ($D21A),A
11E9  FD CB 00 86 RES 0,(IY+0)
11ED  CD 27 03    CALL $0327
11F0  F3          DI
11F1  21 31 2A    LD HL,$2A31
11F4  11 00 00    LD DE,$0000
11F7  3E 09       LD A,$09
11F9  CD 06 04    CALL $0406
11FC  21 5B 6F    LD HL,$6F5B
11FF  01 2B 00    LD BC,$002B
1202  11 00 38    LD DE,$3800
1205  3E 00       LD A,$00
1207  32 0F D2    LD ($D20F),A
120A  3E 05       LD A,$05
120C  CD 02 05    CALL $0502
120F  AF          XOR A
1210  32 4B D2    LD ($D24B),A
1213  32 4C D2    LD ($D24C),A
1216  21 0F 10    LD HL,$100F
1219  22 2C D2    LD ($D22C),HL
121C  FB          EI
121D  06 78       LD B,$78
121F  3A 1A D2    LD A,($D21A)
1222  F6 40       OR $40
1224  32 1A D2    LD ($D21A),A
1227  FD CB 00 86 RES 0,(IY+0)
122B  CD 27 03    CALL $0327
122E  10 EF       DJNZ $121F
1230  3A 7E D2    LD A,($D27E)
1233  A7          AND A
1234  20 19       JR NZ,$124F
1236  01 B4 00    LD BC,$00B4
1239  C5          PUSH BC
123A  FD CB 00 86 RES 0,(IY+0)
123E  CD 27 03    CALL $0327
1241  C1          POP BC
1242  0B          DEC BC
1243  78          LD A,B
1244  B1          OR C
1245  C8          RET Z
1246  FD CB 03 6E BIT 5,(IY+3)
124A  C2 39 12    JP NZ,$1239
124D  A7          AND A
124E  C9          RET
124F  F5          PUSH AF
1250  06 10       LD B,$10
1252  FD CB 00 86 RES 0,(IY+0)
1256  CD 27 03    CALL $0327
1259  21 4C D2    LD HL,$D24C
125C  34          INC (HL)
125D  10 F3       DJNZ $1252
125F  F1          POP AF
1260  21 C7 12    LD HL,$12C7
1263  0E 10       LD C,$10
1265  CD 0F 46    CALL $460F
1268  21 CF 12    LD HL,$12CF
126B  CD 12 06    CALL $0612
126E  21 DC 12    LD HL,$12DC
1271  CD 12 06    CALL $0612
1274  3E 09       LD A,$09
1276  32 17 D2    LD ($D217),A
1279  06 3C       LD B,$3C
127B  C5          PUSH BC
127C  FD CB 00 86 RES 0,(IY+0)
1280  CD 27 03    CALL $0327
1283  FD 36 0A 00 LD (IY+10),$00
1287  21 17 D2    LD HL,$D217
128A  11 BF D2    LD DE,$D2BF
128D  06 01       LD B,$01
128F  CD B2 49    CALL $49B2
1292  EB          EX DE,HL
1293  21 00 D0    LD HL,$D000
1296  0E 74       LD C,$74
1298  06 67       LD B,$67
129A  CD A8 2F    CALL $2FA8
129D  22 36 D2    LD ($D236),HL
12A0  C1          POP BC
12A1  FD CB 03 7E BIT 7,(IY+3)
12A5  28 0E       JR Z,$12B5
12A7  10 D2       DJNZ $127B
12A9  3E 1A       LD A,$1A
12AB  EF          RST $28
12AC  21 17 D2    LD HL,$D217
12AF  7E          LD A,(HL)
12B0  A7          AND A
12B1  C8          RET Z
12B2  35          DEC (HL)
12B3  18 C4       JR $1279
12B5  21 12 D3    LD HL,$D312
12B8  CD 8D 0B    CALL $0B8D
12BB  79          LD A,C
12BC  2F          CPL
12BD  4F          LD C,A
12BE  7E          LD A,(HL)
12BF  A1          AND C
12C0  77          LD (HL),A
12C1  21 7E D2    LD HL,$D27E
12C4  35          DEC (HL)
12C5  37          SCF
12C6  C9          RET
12C7  .byte 12 80 81 FF 13 90 91 FF 0B 0C 67 68 69 6A 6B 6C ; ..........ghijkl
12D7  .byte 6D 6E 00 8F FF 0B 0D 77 78 79 7A 7B 7C 7D 7E 00 ; mn.....wxyz{|}~.
12E7  .byte 9F FF                                           ; ..

; ==== sub_12E9 (1 caller) ====
12E9  2A B6 D2    LD HL,($D2B6)
12EC  11 06 13    LD DE,$1306
12EF  19          ADD HL,DE
12F0  7E          LD A,(HL)
12F1  A7          AND A
12F2  37          SCF
12F3  C8          RET Z
12F4  FD 77 03    LD (IY+3),A
12F7  3A 24 D2    LD A,($D224)
12FA  E6 1F       AND $1F
12FC  C0          RET NZ
12FD  2A B6 D2    LD HL,($D2B6)
1300  23          INC HL
1301  22 B6 D2    LD ($D2B6),HL
1304  A7          AND A
1305  C9          RET
1306  .byte F7 F7 F7 F7 DF F7 FF FF D7 F7 F7 F7 FF DF F7 F7 ; ................
1316  .byte DF F7 F7 F7 F7 F7 F7 F7 DF F7 F7 F7 F7 F7 F7 F7 ; ................
1326  .byte F7 F7 DF F7 F5 F5 F5 F5 F5 00 F7 F7 DF F7 DF FF ; ................
1336  .byte FF FF FF F7 F7 DF F7 DF F7 DF F7 F7 F7 F7 F7 FF ; ................
1346  .byte FF F7 DF F7 F7 F7 F7 F7 F7 F7 F7 F7 F7 F7 F7 00 ; ................

; --- main_entry  $1356 — post-init entry: SET (IY+0).0; EI; set the running bank shadows (slot1=bank1/$D22F, slot2=bank2/$D230); RES (IY+2).0/1; CALL subsystem inits $0645/$1CD7/$0AA3; set top-level game mode $D240=3. ---
1356  FD CB 00 C6 SET 0,(IY+0)
135A  FB          EI
135B  3E 01       LD A,$01
135D  32 FE FF    LD ($FFFE),A
1360  32 2F D2    LD ($D22F),A
1363  3E 02       LD A,$02
1365  32 FF FF    LD ($FFFF),A
1368  32 30 D2    LD ($D230),A
136B  FD CB 02 86 RES 0,(IY+2)
136F  FD CB 02 8E RES 1,(IY+2)
1373  CD 45 06    CALL $0645
1376  CD D7 1C    CALL $1CD7
1379  CD A3 0A    CALL $0AA3
137C  3E 03       LD A,$03
137E  32 40 D2    LD ($D240),A
1381  3E 05       LD A,$05
1383  32 F8 D2    LD ($D2F8),A
1386  3E 1C       LD A,$1C
1388  32 39 D2    LD ($D239),A
138B  AF          XOR A
138C  32 38 D2    LD ($D238),A
138F  32 24 D2    LD ($D224),A
1392  FD 77 0D    LD (IY+13),A
1395  21 79 D2    LD HL,$D279
1398  06 08       LD B,$08
139A  CD 0F 14    CALL $140F
139D  21 00 D2    LD HL,$D200
13A0  06 0E       LD B,$0E
13A2  CD 0F 14    CALL $140F
13A5  21 BB D2    LD HL,$D2BB
13A8  06 04       LD B,$04
13AA  CD 0F 14    CALL $140F
13AD  21 06 D3    LD HL,$D306
13B0  06 18       LD B,$18
13B2  CD 0F 14    CALL $140F
13B5  CD 45 06    CALL $0645
13B8  CD DA 42    CALL $42DA
13BB  FD CB 05 8E RES 1,(IY+5)
13BF  38 04       JR C,$13C5
13C1  FD CB 05 CE SET 1,(IY+5)
13C5  3A 38 D2    LD A,($D238)
13C8  FE 13       CP $13
13CA  D2 5B 13    JP NC,$135B
13CD  FD CB 02 86 RES 0,(IY+2)
13D1  FD CB 02 8E RES 1,(IY+2)
13D5  CD 45 06    CALL $0645
13D8  CD DD 0B    CALL $0BDD
13DB  FD CB 05 4E BIT 1,(IY+5)
13DF  28 03       JR Z,$13E4
13E1  DA 5B 13    JP C,$135B
13E4  CD A3 0A    CALL $0AA3
13E7  CD 45 06    CALL $0645
13EA  FD CB 05 46 BIT 0,(IY+5)
13EE  20 06       JR NZ,$13F6
13F0  FD CB 06 66 BIT 4,(IY+6)
13F4  20 0C       JR NZ,$1402
13F6  06 3C       LD B,$3C
13F8  FD CB 00 86 RES 0,(IY+0)
13FC  CD 27 03    CALL $0327
13FF  10 F7       DJNZ $13F8
1401  E7          RST $20
1402  CD 14 14    CALL $1414
1405  A7          AND A
1406  CA 5B 13    JP Z,$135B
1409  3D          DEC A
140A  28 B9       JR Z,$13C5
140C  C3 E4 13    JP $13E4

; ==== sub_140F (5 callers) ====
140F  77          LD (HL),A
1410  23          INC HL
1411  10 FC       DJNZ $140F
1413  C9          RET

; ==== sub_1414 (2 callers) ====
1414  3E 05       LD A,$05
1416  32 FE FF    LD ($FFFE),A
1419  32 2F D2    LD ($D22F),A
141C  3A 38 D2    LD A,($D238)
141F  FD CB 06 66 BIT 4,(IY+6)
1423  28 03       JR Z,$1428
1425  3A D4 D2    LD A,($D2D4)
1428  87          ADD A,A
1429  6F          LD L,A
142A  26 00       LD H,$00
142C  11 00 56    LD DE,$5600
142F  19          ADD HL,DE
1430  7E          LD A,(HL)
1431  23          INC HL
1432  66          LD H,(HL)
1433  6F          LD L,A
1434  B4          OR H
1435  CA AE 1F    JP Z,$1FAE
1438  19          ADD HL,DE
1439  CD 5D 18    CALL $185D
143C  FD CB 02 C6 SET 0,(IY+2)
1440  FD CB 02 CE SET 1,(IY+2)
1444  FD CB 00 CE SET 1,(IY+0)
1448  FD CB 06 DE SET 3,(IY+6)
144C  FD CB 00 BE RES 7,(IY+0)
1450  FD CB 06 B6 RES 6,(IY+6)
1454  FD CB 07 9E RES 3,(IY+7)
1458  FD CB 08 86 RES 0,(IY+8)
145C  FD CB 09 86 RES 0,(IY+9)
1460  FD CB 00 B6 RES 6,(IY+0)
1464  FD CB 05 5E BIT 3,(IY+5)
1468  C4 68 16    CALL NZ,$1668
146B  FD CB 00 EE SET 5,(IY+0)
146F  CD 20 07    CALL $0720
1472  06 10       LD B,$10
1474  C5          PUSH BC
1475  FD CB 00 86 RES 0,(IY+0)
1479  CD 27 03    CALL $0327
147C  FD 36 03 FF LD (IY+3),$FF
1480  2A 24 D2    LD HL,($D224)
1483  23          INC HL
1484  22 24 D2    LD ($D224),HL
1487  FD CB 05 56 BIT 2,(IY+5)
148B  C4 1E 16    CALL NZ,$161E
148E  21 50 00    LD HL,$0050
1491  22 59 D2    LD ($D259),HL
1494  21 58 00    LD HL,$0058
1497  22 5B D2    LD ($D25B),HL
149A  21 40 00    LD HL,$0040
149D  22 5D D2    LD ($D25D),HL
14A0  21 48 00    LD HL,$0048
14A3  22 5F D2    LD ($D25F),HL
14A6  CD F1 1A    CALL $1AF1
14A9  3E 01       LD A,$01
14AB  32 FE FF    LD ($FFFE),A
14AE  32 2F D2    LD ($D22F),A
14B1  3E 02       LD A,$02
14B3  32 FF FF    LD ($FFFF),A
14B6  32 30 D2    LD ($D230),A
14B9  CD 1E 28    CALL $281E
14BC  CD A1 06    CALL $06A1
14BF  CD 20 07    CALL $0720
14C2  C1          POP BC
14C3  10 AF       DJNZ $1474
14C5  FD CB 05 4E BIT 1,(IY+5)
14C9  28 12       JR Z,$14DD
14CB  21 00 00    LD HL,$0000
14CE  3A 05 D3    LD A,($D305)
14D1  0F          RRCA
14D2  30 03       JR NC,$14D7
14D4  21 2A 00    LD HL,$002A
14D7  22 B6 D2    LD ($D2B6),HL
14DA  FD 74 0A    LD (IY+10),H
14DD  FD CB 00 86 RES 0,(IY+0)
14E1  CD 27 03    CALL $0327
14E4  FD CB 05 56 BIT 2,(IY+5)
14E8  C4 1E 16    CALL NZ,$161E
14EB  FD CB 06 5E BIT 3,(IY+6)
14EF  C4 D7 33    CALL NZ,$33D7
14F2  3A 24 D2    LD A,($D224)
14F5  E6 01       AND $01
14F7  20 02       JR NZ,$14FB
14F9  18 0E       JR $1509
14FB  3A B1 D2    LD A,($D2B1)
14FE  A7          AND A
14FF  C4 96 16    CALL NZ,$1696
1502  FD CB 07 4E BIT 1,(IY+7)
1506  C4 E4 16    CALL NZ,$16E4
1509  FD CB 06 4E BIT 1,(IY+6)
150D  C4 AC 15    CALL NZ,$15AC
1510  FD CB 05 4E BIT 1,(IY+5)
1514  28 0D       JR Z,$1523
1516  FD CB 03 7E BIT 7,(IY+3)
151A  CA 46 18    JP Z,$1846
151D  CD E9 12    CALL $12E9
1520  DA 46 18    JP C,$1846
1523  2A 24 D2    LD HL,($D224)
1526  23          INC HL
1527  22 24 D2    LD ($D224),HL
152A  FD CB 05 5E BIT 3,(IY+5)
152E  C4 72 16    CALL NZ,$1672
1531  FD CB 05 66 BIT 4,(IY+5)
1535  C4 82 16    CALL NZ,$1682
1538  FD CB 05 7E BIT 7,(IY+5)
153C  C4 8F 16    CALL NZ,$168F
153F  CD 1E 1B    CALL $1B1E
1542  FD CB 05 56 BIT 2,(IY+5)
1546  C4 F1 1A    CALL NZ,$1AF1
1549  AF          XOR A
154A  32 FD D2    LD ($D2FD),A
154D  32 DF D2    LD ($D2DF),A
1550  FD 36 0A 15 LD (IY+10),$15
1554  21 3F D0    LD HL,$D03F
1557  22 36 D2    LD ($D236),HL
155A  21 01 D0    LD HL,$D001
155D  06 07       LD B,$07
155F  11 03 00    LD DE,$0003
1562  3E E0       LD A,$E0
1564  77          LD (HL),A
1565  19          ADD HL,DE
1566  77          LD (HL),A
1567  19          ADD HL,DE
1568  77          LD (HL),A
1569  19          ADD HL,DE
156A  10 F8       DJNZ $1564
156C  3E 01       LD A,$01
156E  32 FE FF    LD ($FFFE),A
1571  32 2F D2    LD ($D22F),A
1574  3E 02       LD A,$02
1576  32 FF FF    LD ($FFFF),A
1579  32 30 D2    LD ($D230),A
157C  CD 1E 28    CALL $281E
157F  CD A1 06    CALL $06A1
1582  CD 20 07    CALL $0720
1585  21 1A D2    LD HL,$D21A
1588  CB F6       SET 6,(HL)
158A  FD CB 03 7E BIT 7,(IY+3)
158E  CC D2 15    CALL Z,$15D2
1591  3A 24 D2    LD A,($D224)
1594  E6 01       AND $01
1596  20 0A       JR NZ,$15A2
1598  3A 83 D2    LD A,($D283)
159B  A7          AND A
159C  C4 3D 17    CALL NZ,$173D
159F  C3 DD 14    JP $14DD
15A2  3A 81 D2    LD A,($D281)
15A5  A7          AND A
15A6  C2 EB 17    JP NZ,$17EB
15A9  C3 DD 14    JP $14DD

; ==== sub_15AC (1 caller) ====
15AC  FD 36 03 F7 LD (IY+3),$F7
15B0  2A 6D D2    LD HL,($D26D)
15B3  11 12 01    LD DE,$0112
15B6  19          ADD HL,DE
15B7  EB          EX DE,HL
15B8  2A FF D3    LD HL,($D3FF)
15BB  AF          XOR A
15BC  ED 52       SBC HL,DE
15BE  D8          RET C
15BF  FD 36 03 FF LD (IY+3),$FF
15C3  6F          LD L,A
15C4  67          LD H,A
15C5  22 04 D4    LD ($D404),HL
15C8  32 06 D4    LD ($D406),A
15CB  22 07 D4    LD ($D407),HL
15CE  32 09 D4    LD ($D409),A
15D1  C9          RET

; ==== sub_15D2 (1 caller) ====
15D2  FD CB 05 4E BIT 1,(IY+5)
15D6  C0          RET NZ
15D7  E7          RST $20
15D8  CD FF 15    CALL $15FF
15DB  FD CB 03 7E BIT 7,(IY+3)
15DF  28 F7       JR Z,$15D8
15E1  CD FF 15    CALL $15FF
15E4  FD CB 03 7E BIT 7,(IY+3)
15E8  20 F7       JR NZ,$15E1
15EA  CD FF 15    CALL $15FF
15ED  FD CB 03 7E BIT 7,(IY+3)
15F1  28 F7       JR Z,$15EA
15F3  3E 03       LD A,$03
15F5  32 FE FF    LD ($FFFE),A
15F8  32 2F D2    LD ($D22F),A
15FB  CD 09 40    CALL $4009
15FE  C9          RET

; ==== sub_15FF (3 callers) ====
15FF  FD 7E 0A    LD A,(IY+10)
1602  FD CB 00 86 RES 0,(IY+0)
1606  CD 27 03    CALL $0327
1609  FD 77 0A    LD (IY+10),A
160C  FD CB 05 56 BIT 2,(IY+5)
1610  C4 1E 16    CALL NZ,$161E
1613  CD 1E 1B    CALL $1B1E
1616  FD CB 05 56 BIT 2,(IY+5)
161A  C4 F1 1A    CALL NZ,$1AF1
161D  C9          RET

; ==== sub_161E (3 callers) ====
161E  ED 5B 8E D2 LD DE,($D28E)
1622  2A 90 D2    LD HL,($D290)
1625  A7          AND A
1626  ED 52       SBC HL,DE
1628  28 15       JR Z,$163F
162A  3E 0B       LD A,$0B
162C  32 FE FF    LD ($FFFE),A
162F  32 2F D2    LD ($D22F),A
1632  21 80 1F    LD HL,$1F80
1635  EB          EX DE,HL
1636  CD 57 32    CALL $3257
1639  2A 8E D2    LD HL,($D28E)
163C  22 90 D2    LD ($D290),HL
163F  3A D5 D2    LD A,($D2D5)
1642  A7          AND A
1643  C0          RET NZ
1644  3A 93 D2    LD A,($D293)
1647  FE 05       CP $05
1649  C0          RET NZ
164A  3E 0B       LD A,$0B
164C  32 FE FF    LD ($FFFE),A
164F  32 2F D2    LD ($D22F),A
1652  01 00 00    LD BC,$0000
1655  3A 92 D2    LD A,($D292)
1658  0F          RRCA
1659  30 02       JR NC,$165D
165B  0E 80       LD C,$80
165D  21 3D 7A    LD HL,$7A3D
1660  09          ADD HL,BC
1661  11 80 01    LD DE,$0180
1664  CD 57 32    CALL $3257
1667  C9          RET

; ==== sub_1668 (1 caller) ====
1668  2A 54 D2    LD HL,($D254)
166B  22 6D D2    LD ($D26D),HL
166E  22 6F D2    LD ($D26F),HL
1671  C9          RET

; ==== sub_1672 (1 caller) ====
1672  3A 24 D2    LD A,($D224)
1675  0F          RRCA
1676  D0          RET NC
1677  2A 6D D2    LD HL,($D26D)
167A  23          INC HL
167B  22 6D D2    LD ($D26D),HL
167E  22 6F D2    LD ($D26F),HL
1681  C9          RET

; ==== sub_1682 (1 caller) ====
1682  3A 24 D2    LD A,($D224)
1685  0F          RRCA
1686  D0          RET NC
1687  2A 73 D2    LD HL,($D273)
168A  2B          DEC HL
168B  22 73 D2    LD ($D273),HL
168E  C9          RET

; ==== sub_168F (1 caller) ====
168F  2A 57 D2    LD HL,($D257)
1692  22 73 D2    LD ($D273),HL
1695  C9          RET

; ==== sub_1696 (1 caller) ====
1696  3D          DEC A
1697  32 B1 D2    LD ($D2B1),A
169A  5F          LD E,A
169B  F3          DI
169C  3E 08       LD A,$08
169E  32 FE FF    LD ($FFFE),A
16A1  32 2F D2    LD ($D22F),A
16A4  1E 00       LD E,$00
16A6  43          LD B,E
16A7  3A B2 D2    LD A,($D2B2)
16AA  87          ADD A,A
16AB  4F          LD C,A
16AC  57          LD D,A
16AD  3A 2C D2    LD A,($D22C)
16B0  D2 B8 16    JP NC,$16B8
16B3  1E 20       LD E,$20
16B5  3A 2D D2    LD A,($D22D)
16B8  D5          PUSH DE
16B9  87          ADD A,A
16BA  6F          LD L,A
16BB  26 00       LD H,$00
16BD  11 00 74    LD DE,$7400
16C0  19          ADD HL,DE
16C1  7E          LD A,(HL)
16C2  23          INC HL
16C3  66          LD H,(HL)
16C4  6F          LD L,A
16C5  19          ADD HL,DE
16C6  09          ADD HL,BC
16C7  D1          POP DE
16C8  7A          LD A,D
16C9  83          ADD A,E
16CA  D3 BF       OUT ($BF),A
16CC  3E C0       LD A,$C0
16CE  D3 BF       OUT ($BF),A
16D0  3A B1 D2    LD A,($D2B1)
16D3  0F          RRCA
16D4  30 03       JR NC,$16D9
16D6  21 B3 D2    LD HL,$D2B3
16D9  7E          LD A,(HL)
16DA  D3 BE       OUT ($BE),A
16DC  23          INC HL
16DD  E5          PUSH HL
16DE  E1          POP HL
16DF  7E          LD A,(HL)
16E0  D3 BE       OUT ($BE),A
16E2  FB          EI
16E3  C9          RET

; ==== sub_16E4 (1 caller) ====
16E4  ED 5B EA D2 LD DE,($D2EA)
16E8  21 AA 00    LD HL,$00AA
16EB  AF          XOR A
16EC  ED 52       SBC HL,DE
16EE  30 08       JR NC,$16F8
16F0  01 34 17    LD BC,$1734
16F3  5F          LD E,A
16F4  57          LD D,A
16F5  C3 1B 17    JP $171B
16F8  01 3A 17    LD BC,$173A
16FB  21 82 00    LD HL,$0082
16FE  ED 52       SBC HL,DE
1700  28 14       JR Z,$1716
1702  01 37 17    LD BC,$1737
1705  21 64 00    LD HL,$0064
1708  ED 52       SBC HL,DE
170A  28 0F       JR Z,$171B
170C  01 34 17    LD BC,$1734
170F  7B          LD A,E
1710  B2          OR D
1711  28 08       JR Z,$171B
1713  C3 2E 17    JP $172E
1716  C5          PUSH BC
1717  3E 13       LD A,$13
1719  EF          RST $28
171A  C1          POP BC
171B  21 9F D2    LD HL,$D29F
171E  0A          LD A,(BC)
171F  77          LD (HL),A
1720  23          INC HL
1721  77          LD (HL),A
1722  23          INC HL
1723  03          INC BC
1724  36 00       LD (HL),$00
1726  23          INC HL
1727  0A          LD A,(BC)
1728  77          LD (HL),A
1729  03          INC BC
172A  0A          LD A,(BC)
172B  32 2C D2    LD ($D22C),A
172E  13          INC DE
172F  ED 53 EA D2 LD ($D2EA),DE
1733  C9          RET
1734  .byte 02 04 08 02 04 13 02 04 14                      ; .........

; ==== sub_173D (1 caller) ====
173D  3D          DEC A
173E  32 83 D2    LD ($D283),A
1741  28 15       JR Z,$1758
1743  FE 88       CP $88
1745  C0          RET NZ
1746  3A 82 D2    LD A,($D282)
1749  87          ADD A,A
174A  5F          LD E,A
174B  16 00       LD D,$00
174D  21 C7 17    LD HL,$17C7
1750  19          ADD HL,DE
1751  7E          LD A,(HL)
1752  23          INC HL
1753  66          LD H,(HL)
1754  6F          LD L,A
1755  B4          OR H
1756  C8          RET Z
1757  E9          JP (HL)
1758  CD A3 0A    CALL $0AA3
175B  E1          POP HL
175C  FD CB 00 AE RES 5,(IY+0)
1760  FD CB 0D 56 BIT 2,(IY+13)
1764  20 5A       JR NZ,$17C0
1766  FD CB 06 66 BIT 4,(IY+6)
176A  20 58       JR NZ,$17C4
176C  E7          RST $20
176D  FD CB 06 7E BIT 7,(IY+6)
1771  C4 32 18    CALL NZ,$1832
1774  3E 01       LD A,$01
1776  32 FE FF    LD ($FFFE),A
1779  32 2F D2    LD ($D22F),A
177C  3E 02       LD A,$02
177E  32 FF FF    LD ($FFFF),A
1781  32 30 D2    LD ($D230),A
1784  CD 45 06    CALL $0645
1787  CD 9D 44    CALL $449D
178A  3A 38 D2    LD A,($D238)
178D  FE 1A       CP $1A
178F  30 28       JR NC,$17B9
1791  FD CB 07 46 BIT 0,(IY+7)
1795  28 1B       JR Z,$17B2
1797  21 17 17    LD HL,$1717
179A  CD B7 0A    CALL $0AB7
179D  3A 38 D2    LD A,($D238)
17A0  F5          PUSH AF
17A1  3A 39 D2    LD A,($D239)
17A4  32 38 D2    LD ($D238),A
17A7  3C          INC A
17A8  32 39 D2    LD ($D239),A
17AB  CD 14 14    CALL $1414
17AE  F1          POP AF
17AF  32 38 D2    LD ($D238),A
17B2  21 38 D2    LD HL,$D238
17B5  34          INC (HL)
17B6  3E 01       LD A,$01
17B8  C9          RET
17B9  FD CB 07 86 RES 0,(IY+7)
17BD  3E FF       LD A,$FF
17BF  C9          RET
17C0  21 38 D2    LD HL,$D238
17C3  34          INC (HL)
17C4  3E FF       LD A,$FF
17C6  C9          RET
17C7  .byte 00 00 D1 17 D5 17 DD 17 E3 17 3E 0E EF C9 21 40 ; ..........>...!@
17D7  .byte D2 34 3E 09 EF C9 3E 10 CD 7E 33 C9 3E 07 EF FD ; .4>...>..~3.>...
17E7  .byte CB 07 C6 C9                                     ; ....
17EB  3D          DEC A
17EC  32 81 D2    LD ($D281),A
17EF  C2 A9 15    JP NZ,$15A9
17F2  FD CB 05 4E BIT 1,(IY+5)
17F6  20 4E       JR NZ,$1846
17F8  FD CB 0C 66 BIT 4,(IY+12)
17FC  28 04       JR Z,$1802
17FE  FD CB 06 E6 SET 4,(IY+6)
1802  FD CB 06 7E BIT 7,(IY+6)
1806  C4 32 18    CALL NZ,$1832
1809  3E FF       LD A,$FF
180B  32 D3 D2    LD ($D2D3),A
180E  21 00 D3    LD HL,$D300
1811  34          INC (HL)
1812  3A 40 D2    LD A,($D240)
1815  A7          AND A
1816  3E 02       LD A,$02
1818  C0          RET NZ
1819  CD A3 0A    CALL $0AA3
181C  CD 45 06    CALL $0645
181F  E7          RST $20
1820  FD CB 00 AE RES 5,(IY+0)
1824  CD E1 11    CALL $11E1
1827  3E 00       LD A,$00
1829  D0          RET NC
182A  3E 03       LD A,$03
182C  32 40 D2    LD ($D240),A
182F  3E 01       LD A,$01
1831  C9          RET

; ==== sub_1832 (2 callers) ====
1832  3A 41 D2    LD A,($D241)
1835  A7          AND A
1836  20 FA       JR NZ,$1832
1838  F3          DI
1839  FD CB 06 BE RES 7,(IY+6)
183D  AF          XOR A
183E  32 42 D2    LD ($D242),A
1841  32 DC D2    LD ($D2DC),A
1844  FB          EI
1845  C9          RET
1846  21 05 D3    LD HL,$D305
1849  34          INC (HL)
184A  3E 03       LD A,$03
184C  32 FE FF    LD ($FFFE),A
184F  32 2F D2    LD ($D22F),A
1852  21 28 00    LD HL,$0028
1855  CD 0C 40    CALL $400C
1858  CD A3 0A    CALL $0AA3
185B  AF          XOR A
185C  C9          RET

; ==== sub_185D (1 caller) ====
185D  3A 1A D2    LD A,($D21A)
1860  E6 BF       AND $BF
1862  32 1A D2    LD ($D21A),A
1865  FD CB 00 86 RES 0,(IY+0)
1869  CD 27 03    CALL $0327
186C  11 55 D3    LD DE,$D355
186F  01 28 00    LD BC,$0028
1872  ED B0       LDIR
1874  21 55 D3    LD HL,$D355
1877  E5          PUSH HL
1878  FD 7E 05    LD A,(IY+5)
187B  FD 77 0B    LD (IY+11),A
187E  FD 7E 06    LD A,(IY+6)
1881  FD 77 0C    LD (IY+12),A
1884  3E FF       LD A,$FF
1886  32 AB D2    LD ($D2AB),A
1889  AF          XOR A
188A  6F          LD L,A
188B  67          LD H,A
188C  32 4B D2    LD ($D24B),A
188F  32 4C D2    LD ($D24C),A
1892  22 75 D2    LD ($D275),HL
1895  22 77 D2    LD ($D277),HL
1898  22 B8 D2    LD ($D2B8),HL
189B  32 41 D2    LD ($D241),A
189E  32 42 D2    LD ($D242),A
18A1  22 EA D2    LD ($D2EA),HL
18A4  21 81 D2    LD HL,$D281
18A7  06 1E       LD B,$1E
18A9  CD 0F 14    CALL $140F
18AC  21 12 D3    LD HL,$D312
18AF  CD 8D 0B    CALL $0B8D
18B2  EB          EX DE,HL
18B3  21 00 08    LD HL,$0800
18B6  3A 38 D2    LD A,($D238)
18B9  FE 09       CP $09
18BB  38 12       JR C,$18CF
18BD  FE 0B       CP $0B
18BF  28 06       JR Z,$18C7
18C1  30 0C       JR NC,$18CF
18C3  1A          LD A,(DE)
18C4  A1          AND C
18C5  28 08       JR Z,$18CF
18C7  3E FF       LD A,$FF
18C9  32 DC D2    LD ($D2DC),A
18CC  21 20 00    LD HL,$0020
18CF  22 DD D2    LD ($D2DD),HL
18D2  21 FE FF    LD HL,$FFFE
18D5  22 9A D2    LD ($D29A),HL
18D8  21 4B 1B    LD HL,$1B4B
18DB  FD CB 06 66 BIT 4,(IY+6)
18DF  28 09       JR Z,$18EA
18E1  FD CB 05 46 BIT 0,(IY+5)
18E5  28 20       JR Z,$1907
18E7  21 4E 1B    LD HL,$1B4E
18EA  AF          XOR A
18EB  32 A9 D2    LD ($D2A9),A
18EE  3A 38 D2    LD A,($D238)
18F1  D6 1C       SUB $1C
18F3  38 0A       JR C,$18FF
18F5  4F          LD C,A
18F6  87          ADD A,A
18F7  81          ADD A,C
18F8  5F          LD E,A
18F9  16 00       LD D,$00
18FB  21 51 1B    LD HL,$1B51
18FE  19          ADD HL,DE
18FF  11 CF D2    LD DE,$D2CF
1902  01 03 00    LD BC,$0003
1905  ED B0       LDIR
1907  21 54 B3    LD HL,$B354
190A  11 00 30    LD DE,$3000
190D  3E 09       LD A,$09
190F  CD 06 04    CALL $0406
1912  E1          POP HL
1913  7E          LD A,(HL)
1914  32 D5 D2    LD ($D2D5),A
1917  23          INC HL
1918  5E          LD E,(HL)
1919  23          INC HL
191A  56          LD D,(HL)
191B  23          INC HL
191C  ED 53 32 D2 LD ($D232),DE
1920  5E          LD E,(HL)
1921  23          INC HL
1922  56          LD D,(HL)
1923  23          INC HL
1924  ED 53 34 D2 LD ($D234),DE
1928  11 6D D2    LD DE,$D26D
192B  01 08 00    LD BC,$0008
192E  ED B0       LDIR
1930  E5          PUSH HL
1931  E5          PUSH HL
1932  21 12 D3    LD HL,$D312
1935  CD 8D 0B    CALL $0B8D
1938  7E          LD A,(HL)
1939  EB          EX DE,HL
193A  E1          POP HL
193B  A1          AND C
193C  28 1B       JR Z,$1959
193E  2F          CPL
193F  4F          LD C,A
1940  1A          LD A,(DE)
1941  A1          AND C
1942  12          LD (DE),A
1943  21 4E 1B    LD HL,$1B4E
1946  11 CF D2    LD DE,$D2CF
1949  01 03 00    LD BC,$0003
194C  ED B0       LDIR
194E  3A 38 D2    LD A,($D238)
1951  87          ADD A,A
1952  5F          LD E,A
1953  16 00       LD D,$00
1955  21 2F D3    LD HL,$D32F
1958  19          ADD HL,DE
1959  22 17 D2    LD ($D217),HL
195C  7E          LD A,(HL)
195D  D6 03       SUB $03
195F  30 01       JR NC,$1962
1961  AF          XOR A
1962  32 51 D2    LD ($D251),A
1965  1E 00       LD E,$00
1967  0F          RRCA
1968  CB 1B       RR E
196A  0F          RRCA
196B  CB 1B       RR E
196D  0F          RRCA
196E  CB 1B       RR E
1970  E6 1F       AND $1F
1972  57          LD D,A
1973  ED 53 54 D2 LD ($D254),DE
1977  ED 53 69 D2 LD ($D269),DE
197B  23          INC HL
197C  7E          LD A,(HL)
197D  D6 03       SUB $03
197F  30 01       JR NC,$1982
1981  AF          XOR A
1982  32 52 D2    LD ($D252),A
1985  1E 00       LD E,$00
1987  0F          RRCA
1988  CB 1B       RR E
198A  0F          RRCA
198B  CB 1B       RR E
198D  0F          RRCA
198E  CB 1B       RR E
1990  E6 1F       AND $1F
1992  57          LD D,A
1993  ED 53 57 D2 LD ($D257),DE
1997  ED 53 6B D2 LD ($D26B),DE
199B  E1          POP HL
199C  23          INC HL
199D  23          INC HL
199E  5E          LD E,(HL)
199F  23          INC HL
19A0  56          LD D,(HL)
19A1  23          INC HL
19A2  4E          LD C,(HL)
19A3  23          INC HL
19A4  46          LD B,(HL)
19A5  23          INC HL
19A6  E5          PUSH HL
19A7  EB          EX DE,HL
19A8  7C          LD A,H
19A9  F3          DI
19AA  FE 40       CP $40
19AC  38 15       JR C,$19C3
19AE  D6 40       SUB $40
19B0  67          LD H,A
19B1  3E 06       LD A,$06
19B3  32 FE FF    LD ($FFFE),A
19B6  32 2F D2    LD ($D22F),A
19B9  3E 07       LD A,$07
19BB  32 FF FF    LD ($FFFF),A
19BE  32 30 D2    LD ($D230),A
19C1  18 10       JR $19D3
19C3  3E 05       LD A,$05
19C5  32 FE FF    LD ($FFFE),A
19C8  32 2F D2    LD ($D22F),A
19CB  3E 06       LD A,$06
19CD  32 FF FF    LD ($FFFF),A
19D0  32 30 D2    LD ($D230),A
19D3  FB          EI
19D4  11 00 40    LD DE,$4000
19D7  19          ADD HL,DE
19D8  CD 73 0A    CALL $0A73
19DB  E1          POP HL
19DC  5E          LD E,(HL)
19DD  23          INC HL
19DE  56          LD D,(HL)
19DF  23          INC HL
19E0  EB          EX DE,HL
19E1  01 00 40    LD BC,$4000
19E4  09          ADD HL,BC
19E5  22 49 D2    LD ($D249),HL
19E8  EB          EX DE,HL
19E9  5E          LD E,(HL)
19EA  23          INC HL
19EB  56          LD D,(HL)
19EC  23          INC HL
19ED  E5          PUSH HL
19EE  EB          EX DE,HL
19EF  11 00 00    LD DE,$0000
19F2  3E 0C       LD A,$0C
19F4  CD 06 04    CALL $0406
19F7  E1          POP HL
19F8  7E          LD A,(HL)
19F9  23          INC HL
19FA  5E          LD E,(HL)
19FB  23          INC HL
19FC  56          LD D,(HL)
19FD  23          INC HL
19FE  E5          PUSH HL
19FF  EB          EX DE,HL
1A00  11 00 20    LD DE,$2000
1A03  CD 06 04    CALL $0406
1A06  E1          POP HL
1A07  7E          LD A,(HL)
1A08  32 2D D2    LD ($D22D),A
1A0B  23          INC HL
1A0C  E5          PUSH HL
1A0D  FD CB 00 86 RES 0,(IY+0)
1A11  CD 27 03    CALL $0327
1A14  CD C9 09    CALL $09C9
1A17  E1          POP HL
1A18  11 9F D2    LD DE,$D29F
1A1B  7E          LD A,(HL)
1A1C  12          LD (DE),A
1A1D  13          INC DE
1A1E  12          LD (DE),A
1A1F  13          INC DE
1A20  23          INC HL
1A21  AF          XOR A
1A22  12          LD (DE),A
1A23  13          INC DE
1A24  7E          LD A,(HL)
1A25  12          LD (DE),A
1A26  23          INC HL
1A27  7E          LD A,(HL)
1A28  32 2C D2    LD ($D22C),A
1A2B  23          INC HL
1A2C  5E          LD E,(HL)
1A2D  23          INC HL
1A2E  56          LD D,(HL)
1A2F  23          INC HL
1A30  E5          PUSH HL
1A31  21 00 56    LD HL,$5600
1A34  19          ADD HL,DE
1A35  3E 05       LD A,$05
1A37  32 FE FF    LD ($FFFE),A
1A3A  32 2F D2    LD ($D22F),A
1A3D  CD 80 1A    CALL $1A80
1A40  E1          POP HL
1A41  4E          LD C,(HL)
1A42  FD 7E 05    LD A,(IY+5)
1A45  E6 02       AND $02
1A47  B1          OR C
1A48  FD 77 05    LD (IY+5),A
1A4B  23          INC HL
1A4C  7E          LD A,(HL)
1A4D  FD 77 06    LD (IY+6),A
1A50  23          INC HL
1A51  7E          LD A,(HL)
1A52  FD 77 07    LD (IY+7),A
1A55  23          INC HL
1A56  7E          LD A,(HL)
1A57  FD 77 08    LD (IY+8),A
1A5A  23          INC HL
1A5B  3A D3 D2    LD A,($D2D3)
1A5E  BE          CP (HL)
1A5F  28 09       JR Z,$1A6A
1A61  7E          LD A,(HL)
1A62  A7          AND A
1A63  FA 6A 1A    JP M,$1A6A
1A66  32 F7 D2    LD ($D2F7),A
1A69  DF          RST $18
1A6A  06 20       LD B,$20
1A6C  21 7D D3    LD HL,$D37D
1A6F  AF          XOR A
1A70  77          LD (HL),A
1A71  23          INC HL
1A72  77          LD (HL),A
1A73  23          INC HL
1A74  10 FA       DJNZ $1A70
1A76  FD CB 0C 6E BIT 5,(IY+12)
1A7A  C8          RET Z
1A7B  FD CB 06 EE SET 5,(IY+6)
1A7F  C9          RET

; ==== sub_1A80 (1 caller) ====
1A80  E5          PUSH HL
1A81  DD 21 FD D3 LD IX,$D3FD
1A85  11 1A 00    LD DE,$001A
1A88  0E 00       LD C,$00
1A8A  2A 17 D2    LD HL,($D217)
1A8D  3E 00       LD A,$00
1A8F  CD B3 1A    CALL $1AB3
1A92  E1          POP HL
1A93  7E          LD A,(HL)
1A94  23          INC HL
1A95  32 F3 D2    LD ($D2F3),A
1A98  3D          DEC A
1A99  47          LD B,A
1A9A  7E          LD A,(HL)
1A9B  23          INC HL
1A9C  CD B3 1A    CALL $1AB3
1A9F  10 F9       DJNZ $1A9A
1AA1  3A F3 D2    LD A,($D2F3)
1AA4  47          LD B,A
1AA5  3E 20       LD A,$20
1AA7  90          SUB B
1AA8  C8          RET Z
1AA9  47          LD B,A
1AAA  DD 36 00 FF LD (IX+0),$FF
1AAE  DD 19       ADD IX,DE
1AB0  10 F8       DJNZ $1AAA
1AB2  C9          RET

; ==== sub_1AB3 (2 callers) ====
1AB3  DD 77 00    LD (IX+0),A
1AB6  7E          LD A,(HL)
1AB7  D9          EXX
1AB8  6F          LD L,A
1AB9  26 00       LD H,$00
1ABB  DD 74 01    LD (IX+1),H
1ABE  29          ADD HL,HL
1ABF  29          ADD HL,HL
1AC0  29          ADD HL,HL
1AC1  29          ADD HL,HL
1AC2  29          ADD HL,HL
1AC3  DD 75 02    LD (IX+2),L
1AC6  DD 74 03    LD (IX+3),H
1AC9  D9          EXX
1ACA  23          INC HL
1ACB  7E          LD A,(HL)
1ACC  D9          EXX
1ACD  6F          LD L,A
1ACE  26 00       LD H,$00
1AD0  DD 74 04    LD (IX+4),H
1AD3  29          ADD HL,HL
1AD4  29          ADD HL,HL
1AD5  29          ADD HL,HL
1AD6  29          ADD HL,HL
1AD7  29          ADD HL,HL
1AD8  DD 75 05    LD (IX+5),L
1ADB  DD 74 06    LD (IX+6),H
1ADE  DD E5       PUSH IX
1AE0  E1          POP HL
1AE1  11 07 00    LD DE,$0007
1AE4  19          ADD HL,DE
1AE5  06 13       LD B,$13
1AE7  AF          XOR A
1AE8  77          LD (HL),A
1AE9  23          INC HL
1AEA  10 FC       DJNZ $1AE8
1AEC  D9          EXX
1AED  23          INC HL
1AEE  DD 19       ADD IX,DE
1AF0  C9          RET

; ==== sub_1AF1 (3 callers) ====
1AF1  3A 92 D2    LD A,($D292)
1AF4  5F          LD E,A
1AF5  16 00       LD D,$00
1AF7  21 45 1B    LD HL,$1B45
1AFA  19          ADD HL,DE
1AFB  7E          LD A,(HL)
1AFC  6A          LD L,D
1AFD  CB 3F       SRL A
1AFF  CB 1D       RR L
1B01  67          LD H,A
1B02  11 3D 77    LD DE,$773D
1B05  19          ADD HL,DE
1B06  22 8E D2    LD ($D28E),HL
1B09  21 93 D2    LD HL,$D293
1B0C  7E          LD A,(HL)
1B0D  3C          INC A
1B0E  77          LD (HL),A
1B0F  FE 0A       CP $0A
1B11  D8          RET C
1B12  36 00       LD (HL),$00
1B14  2B          DEC HL
1B15  7E          LD A,(HL)
1B16  3C          INC A
1B17  FE 06       CP $06
1B19  38 01       JR C,$1B1C
1B1B  AF          XOR A
1B1C  77          LD (HL),A
1B1D  C9          RET

; ==== sub_1B1E (2 callers) ====
1B1E  3A A1 D2    LD A,($D2A1)
1B21  6F          LD L,A
1B22  26 00       LD H,$00
1B24  29          ADD HL,HL
1B25  29          ADD HL,HL
1B26  29          ADD HL,HL
1B27  29          ADD HL,HL
1B28  29          ADD HL,HL
1B29  22 A5 D2    LD ($D2A5),HL
1B2C  21 9F D2    LD HL,$D29F
1B2F  35          DEC (HL)
1B30  C0          RET NZ
1B31  2A A1 D2    LD HL,($D2A1)
1B34  7D          LD A,L
1B35  3C          INC A
1B36  BC          CP H
1B37  38 01       JR C,$1B3A
1B39  AF          XOR A
1B3A  6F          LD L,A
1B3B  22 A1 D2    LD ($D2A1),HL
1B3E  3A A0 D2    LD A,($D2A0)
1B41  32 9F D2    LD ($D29F),A
1B44  C9          RET
1B45  .byte 05 04 03 02 01 00 00 00 00 01 30 00 02 00 00 02 ; ..........0.....
1B55  .byte 00 00 02 00 00 02 00 00 01 00 00 01 00 00 01 00 ; ................
1B65  .byte 00 01 00 00 01 00 00 01 00 01 02 00 01 02 FF 02 ; ................
1B75  .byte 03 01 01 03 FE 02 04 01 01 04 FD 03 05 02 01 06 ; ................
1B85  .byte FB 03 06 03 00 07 FA 03 06 05 FF 08 F9 03 07 06 ; ................
1B95  .byte FE 09 F7 03 07 08 FD 0A F6 02 07 09 FB 0B F4 01 ; ................
1BA5  .byte 06 0B FA 0B F3 00 06 0D F8 0B F2 FF 05 0E F6 0B ; ................
1BB5  .byte F1 FD 03 10 F4 0B F0 FB 02 12 F2 0A F0 F9 00 13 ; ................
1BC5  .byte F0 09 F0 F7 FE 14 EE 08 F0 F4 FC 15 EC 07 F0 F2 ; ................
1BD5  .byte F9 15 EA 05 F1 EF F6 16 E9 02 F2 ED F4 15 E7 00 ; ................
1BE5  .byte F4 EB F1 15 E6 FD F5 E8 EE 14 E5 FA F8 E6 EB 13 ; ................
1BF5  .byte E5 F7 FA E4 E8 11 E5 F4 FD E3 E5 0F E5 F1 00 E1 ; ................
1C05  .byte E3 0D E6 ED 03 E0 E0 0A E7 EA 07 E0 DE 07 E9 E6 ; ................
1C15  .byte 0B DF DD 04 EB E3 0E DF DB 00 EE E0 12 E0 DA FC ; ................
1C25  .byte F1 DD 16 E1 DA F8 F4 DB 1A E3 DA F4 F8 D8 1E E5 ; ................
1C35  .byte DA EF FC D7 22 E8 DB EB 00 D5 25 EB DC E6 05 D4 ; ....".....%.....
1C45  .byte 28 EE DE E2 09 D4 2B F2 E1 DE 0E D4 2D F6 E4 D9 ; (.....+.....-...
1C55  .byte 13 D5 2F FB E8 D6 18 D6 31 00 EC D2 1D D8 32 05 ; ../.....1.....2.
1C65  .byte F0 CF 22 DA 32 0B F5 CD 27 DD 32 10 FA CB 2B E0 ; ..".2...'.2...+.
1C75  .byte 31 16 00 C9 2F E5 2F 1B 06 C8 33 E9 2D 21 0C C8 ; 1..././...3.-!..
1C85  .byte 36 EE 2B 26 12 C8 39 F4 27 2B 18 CA 3B FA 23 30 ; 6.+&..9.'+..;.#0
1C95  .byte 1E CB 3D 00 1E 35 24 CE 3E 06 19 39 2A D1 3E 0D ; ..=..5$.>..9*.>.
1CA5  .byte 14 3C 30 D5 3D 14 0D 3F 35 D9 3C 1B 07 41 3A DF ; .<0.=..?5.<..A:.
1CB5  .byte 3A 21 00 43 3E E4 37 28 F9 44 42 EB 33 2E F2 44 ; :!.C>.7(.DB.3..D
1CC5  .byte 45 F1 2F 34 EA 43 47 F9 2A 3A E3 41 49 00 24 3F ; E./4.CG.*:.AI.$?
1CD5  .byte DC 3F                                           ; .?

; ==== sega_logo  $1CD7  (1 caller) — load the SEGA-logo boot screen (verified by rendering it): RST $20 (bank-3 setup); display OFF; decompress the BG tiles (A=$0C,HL=$FA74 -> normalised bank 15 $7A74 = file $3FA74; header match+$88/lit+$461/count=1024 -> 128 tiles) to VRAM $0000; decompress sprite tiles (A=$04,HL=$F600 -> bank 7 = file $1F600) to VRAM $2000; BG/sprite palette index $12 (the blue logo palette, bank-8 table -> file $23B10); clear the name table $3800 to tile $70; display ON; lay out the logo via the name table ($1E7C/$1E8A/$1F03/$1EC1). The actual game title screen is a LATER screen loaded the same way; loaded compressed tilemaps are a LEVEL feature (decompressed to RAM, drawn by scroll_draw). ====
1CD7  E7          RST $20
1CD8  3A 1A D2    LD A,($D21A)
1CDB  E6 BF       AND $BF
1CDD  32 1A D2    LD ($D21A),A
1CE0  FD CB 00 86 RES 0,(IY+0)
1CE4  CD 27 03    CALL $0327
1CE7  AF          XOR A
1CE8  32 4B D2    LD ($D24B),A
1CEB  32 4C D2    LD ($D24C),A
1CEE  21 74 FA    LD HL,$FA74
1CF1  11 00 00    LD DE,$0000
1CF4  3E 0C       LD A,$0C
1CF6  CD 06 04    CALL $0406
1CF9  21 00 F6    LD HL,$F600
1CFC  11 00 20    LD DE,$2000
1CFF  3E 04       LD A,$04
1D01  CD 06 04    CALL $0406
1D04  21 12 12    LD HL,$1212
1D07  22 2C D2    LD ($D22C),HL
1D0A  FD CB 00 CE SET 1,(IY+0)
1D0E  3E 00       LD A,$00
1D10  D3 BF       OUT ($BF),A
1D12  3E 38       LD A,$38
1D14  F6 40       OR $40
1D16  D3 BF       OUT ($BF),A
1D18  01 80 03    LD BC,$0380
1D1B  3E 70       LD A,$70
1D1D  D3 BE       OUT ($BE),A
1D1F  F5          PUSH AF
1D20  F1          POP AF
1D21  AF          XOR A
1D22  D3 BE       OUT ($BE),A
1D24  0B          DEC BC
1D25  78          LD A,B
1D26  B1          OR C
1D27  20 F2       JR NZ,$1D1B
1D29  3A 1A D2    LD A,($D21A)
1D2C  F6 40       OR $40
1D2E  32 1A D2    LD ($D21A),A
1D31  FD CB 00 86 RES 0,(IY+0)
1D35  CD 27 03    CALL $0327
1D38  21 2C 00    LD HL,$002C
1D3B  22 13 D2    LD ($D213),HL
1D3E  CD 7C 1E    CALL $1E7C
1D41  CD 8A 1E    CALL $1E8A
1D44  CD 03 1F    CALL $1F03
1D47  FD CB 03 7E BIT 7,(IY+3)
1D4B  CA 4A 1E    JP Z,$1E4A
1D4E  2E 2C       LD L,$2C
1D50  01 27 1F    LD BC,$1F27
1D53  11 04 00    LD DE,$0004
1D56  CD C1 1E    CALL $1EC1
1D59  7D          LD A,L
1D5A  FE 6C       CP $6C
1D5C  DA 3E 1D    JP C,$1D3E
1D5F  21 39 1F    LD HL,$1F39
1D62  22 17 D2    LD ($D217),HL
1D65  06 08       LD B,$08
1D67  CD EE 1E    CALL $1EEE
1D6A  CD 7C 1E    CALL $1E7C
1D6D  CD 8A 1E    CALL $1E8A
1D70  CD 03 1F    CALL $1F03
1D73  FD CB 03 7E BIT 7,(IY+3)
1D77  CA 4A 1E    JP Z,$1E4A
1D7A  2E 6C       LD L,$6C
1D7C  01 27 1F    LD BC,$1F27
1D7F  11 04 00    LD DE,$0004
1D82  CD C1 1E    CALL $1EC1
1D85  7D          LD A,L
1D86  FE AC       CP $AC
1D88  DA 6A 1D    JP C,$1D6A
1D8B  AF          XOR A
1D8C  32 0F D2    LD ($D20F),A
1D8F  21 AA 1F    LD HL,$1FAA
1D92  CD 12 06    CALL $0612
1D95  CD 70 1E    CALL $1E70
1D98  21 39 1F    LD HL,$1F39
1D9B  22 17 D2    LD ($D217),HL
1D9E  06 08       LD B,$08
1DA0  CD EE 1E    CALL $1EEE
1DA3  21 4B 1F    LD HL,$1F4B
1DA6  22 17 D2    LD ($D217),HL
1DA9  06 0A       LD B,$0A
1DAB  CD EE 1E    CALL $1EEE
1DAE  21 5D 1F    LD HL,$1F5D
1DB1  22 17 D2    LD ($D217),HL
1DB4  06 08       LD B,$08
1DB6  CD EE 1E    CALL $1EEE
1DB9  CD 7C 1E    CALL $1E7C
1DBC  FD CB 03 7E BIT 7,(IY+3)
1DC0  CA 4A 1E    JP Z,$1E4A
1DC3  2E 6C       LD L,$6C
1DC5  01 6F 1F    LD BC,$1F6F
1DC8  11 FC FF    LD DE,$FFFC
1DCB  CD C1 1E    CALL $1EC1
1DCE  7D          LD A,L
1DCF  FE 6C       CP $6C
1DD1  D2 B9 1D    JP NC,$1DB9
1DD4  21 5D 1F    LD HL,$1F5D
1DD7  22 17 D2    LD ($D217),HL
1DDA  06 08       LD B,$08
1DDC  CD EE 1E    CALL $1EEE
1DDF  CD 7C 1E    CALL $1E7C
1DE2  FD CB 03 7E BIT 7,(IY+3)
1DE6  CA 4A 1E    JP Z,$1E4A
1DE9  2E 2C       LD L,$2C
1DEB  01 6F 1F    LD BC,$1F6F
1DEE  11 FC FF    LD DE,$FFFC
1DF1  CD C1 1E    CALL $1EC1
1DF4  7D          LD A,L
1DF5  FE 2C       CP $2C
1DF7  C2 DF 1D    JP NZ,$1DDF
1DFA  21 5D 1F    LD HL,$1F5D
1DFD  22 17 D2    LD ($D217),HL
1E00  06 08       LD B,$08
1E02  CD EE 1E    CALL $1EEE
1E05  21 4B 1F    LD HL,$1F4B
1E08  22 17 D2    LD ($D217),HL
1E0B  06 0A       LD B,$0A
1E0D  CD EE 1E    CALL $1EEE
1E10  21 81 1F    LD HL,$1F81
1E13  22 17 D2    LD ($D217),HL
1E16  06 14       LD B,$14
1E18  CD EE 1E    CALL $1EEE
1E1B  3E 08       LD A,$08
1E1D  32 FE FF    LD ($FFFE),A
1E20  32 2F D2    LD ($D22F),A
1E23  3E 09       LD A,$09
1E25  32 FF FF    LD ($FFFF),A
1E28  32 30 D2    LD ($D230),A
1E2B  CD 00 7C    CALL $7C00
1E2E  3E 01       LD A,$01
1E30  32 FE FF    LD ($FFFE),A
1E33  32 2F D2    LD ($D22F),A
1E36  3E 02       LD A,$02
1E38  32 FF FF    LD ($FFFF),A
1E3B  32 30 D2    LD ($D230),A
1E3E  21 81 1F    LD HL,$1F81
1E41  22 17 D2    LD ($D217),HL
1E44  06 50       LD B,$50
1E46  CD EE 1E    CALL $1EEE
1E49  C9          RET
1E4A  CD 8A 1E    CALL $1E8A
1E4D  3A 13 D2    LD A,($D213)
1E50  C6 08       ADD A,$08
1E52  32 13 D2    LD ($D213),A
1E55  C6 18       ADD A,$18
1E57  FE C8       CP $C8
1E59  38 EF       JR C,$1E4A
1E5B  AF          XOR A
1E5C  32 0F D2    LD ($D20F),A
1E5F  21 AA 1F    LD HL,$1FAA
1E62  CD 12 06    CALL $0612
1E65  CD 70 1E    CALL $1E70
1E68  21 2C 00    LD HL,$002C
1E6B  22 13 D2    LD ($D213),HL
1E6E  18 CE       JR $1E3E

; ==== sub_1E70 (2 callers) ====
1E70  DB 00       IN A,($00)
1E72  07          RLCA
1E73  07          RLCA
1E74  D0          RET NC
1E75  21 A5 1F    LD HL,$1FA5
1E78  CD 12 06    CALL $0612
1E7B  C9          RET

; ==== nt_buf_init  $1E7C  (5 callers) — point the name-table build buffer at RAM $D000 ($D236 = $D000). ====
1E7C  FD CB 00 86 RES 0,(IY+0)
1E80  CD 27 03    CALL $0327
1E83  21 00 D0    LD HL,$D000
1E86  22 36 D2    LD ($D236),HL
1E89  C9          RET

; ==== sub_1E8A (3 callers) ====
1E8A  3A 13 D2    LD A,($D213)
1E8D  C6 18       ADD A,$18
1E8F  4F          LD C,A
1E90  FE 48       CP $48
1E92  D8          RET C
1E93  FE C8       CP $C8
1E95  D0          RET NC
1E96  21 52 3A    LD HL,$3A52
1E99  79          LD A,C
1E9A  CB 3F       SRL A
1E9C  CB 3F       SRL A
1E9E  CB 3F       SRL A
1EA0  D6 09       SUB $09
1EA2  4F          LD C,A
1EA3  06 00       LD B,$00
1EA5  09          ADD HL,BC
1EA6  09          ADD HL,BC
1EA7  06 06       LD B,$06
1EA9  11 40 00    LD DE,$0040
1EAC  7D          LD A,L
1EAD  D3 BF       OUT ($BF),A
1EAF  7C          LD A,H
1EB0  F6 40       OR $40
1EB2  D3 BF       OUT ($BF),A
1EB4  79          LD A,C
1EB5  D3 BE       OUT ($BE),A
1EB7  C6 10       ADD A,$10
1EB9  4F          LD C,A
1EBA  19          ADD HL,DE
1EBB  AF          XOR A
1EBC  D3 BE       OUT ($BE),A
1EBE  10 EC       DJNZ $1EAC
1EC0  C9          RET

; ==== nt_draw_rows  $1EC1  (4 callers) — build the screen's name table procedurally: per row, index the offset table $1F14 by ($D213-L)>>2 to pick a layout run, then CALL nt_writer $2F07 to place its tiles into the $D000 buffer; advance the dest $D213. (So screens like the SEGA logo have NO stored tilemap — it is composed in code from layout tables $1F14/$1F27/$1F39 then DMA'd to VRAM.) ====
1EC1  D5          PUSH DE
1EC2  C5          PUSH BC
1EC3  3A 13 D2    LD A,($D213)
1EC6  95          SUB L
1EC7  CB 3F       SRL A
1EC9  CB 3F       SRL A
1ECB  5F          LD E,A
1ECC  16 00       LD D,$00
1ECE  21 14 1F    LD HL,$1F14
1ED1  19          ADD HL,DE
1ED2  6E          LD L,(HL)
1ED3  62          LD H,D
1ED4  CB 7D       BIT 7,L
1ED6  28 01       JR Z,$1ED9
1ED8  25          DEC H
1ED9  01 46 00    LD BC,$0046
1EDC  09          ADD HL,BC
1EDD  EB          EX DE,HL
1EDE  C1          POP BC
1EDF  2A 13 D2    LD HL,($D213)
1EE2  CD 07 2F    CALL $2F07
1EE5  D1          POP DE
1EE6  2A 13 D2    LD HL,($D213)
1EE9  19          ADD HL,DE
1EEA  22 13 D2    LD ($D213),HL
1EED  C9          RET

; ==== sub_1EEE (9 callers) ====
1EEE  C5          PUSH BC
1EEF  CD 7C 1E    CALL $1E7C
1EF2  2A 13 D2    LD HL,($D213)
1EF5  11 46 00    LD DE,$0046
1EF8  ED 4B 17 D2 LD BC,($D217)
1EFC  CD 07 2F    CALL $2F07
1EFF  C1          POP BC
1F00  10 EC       DJNZ $1EEE
1F02  C9          RET

; ==== sub_1F03 (2 callers) ====
1F03  11 47 00    LD DE,$0047
1F06  2A 13 D2    LD HL,($D213)
1F09  01 18 00    LD BC,$0018
1F0C  09          ADD HL,BC
1F0D  01 93 1F    LD BC,$1F93
1F10  CD 07 2F    CALL $2F07
1F13  C9          RET
1F14  .byte 00 FD FB F8 F6 F4 F2 F1 F0 F0 F0 F1 F2 F4 F6 F8 ; ................
1F24  .byte FB FD 00 00 02 04 06 FF FF 20 22 24 26 FF FF 40 ; ......... "$&..@
1F34  .byte 42 44 46 FF FF 00 02 04 06 FF FF FE 18 1A 1C FF ; BDF.............
1F44  .byte FF FE 38 3A 3C FF FF 08 0A 0C 0E FF FF 28 2A 2C ; ..8:<........(*,
1F54  .byte 2E FF FF 48 4A 4C 4E FF FF 10 12 14 16 FF FF 58 ; ...HJLN........X
1F64  .byte 5A 5C FF FF FF 78 7A 7C FF FF FF 10 12 14 16 FF ; Z\...xz|........
1F74  .byte FF 30 32 34 36 FF FF 50 52 54 56 FF FF 60 62 64 ; .0246..PRTV..`bd
1F84  .byte 66 FF FF 68 6A 6C 6E FF FF 70 72 74 76 FF FF 1E ; f..hjln..prtv...
1F94  .byte 1E FF FF FF FF 1E 1E FF FF FF FF 1E 1E FF FF FF ; ................
1FA4  .byte FF 18 09 60 61 FF 19 0E 62 FF                   ; ...`a...b.
1FAE  3A 1A D2    LD A,($D21A)
1FB1  E6 BF       AND $BF
1FB3  32 1A D2    LD ($D21A),A
1FB6  FD CB 00 86 RES 0,(IY+0)
1FBA  CD 27 03    CALL $0327
1FBD  AF          XOR A
1FBE  32 4B D2    LD ($D24B),A
1FC1  32 4C D2    LD ($D24C),A
1FC4  21 0A 0B    LD HL,$0B0A
1FC7  22 2C D2    LD ($D22C),HL
1FCA  21 00 00    LD HL,$0000
1FCD  11 00 00    LD DE,$0000
1FD0  3E 0C       LD A,$0C
1FD2  CD 06 04    CALL $0406
1FD5  3E 05       LD A,$05
1FD7  32 FE FF    LD ($FFFE),A
1FDA  32 2F D2    LD ($D22F),A
1FDD  21 86 6F    LD HL,$6F86
1FE0  01 E4 01    LD BC,$01E4
1FE3  11 00 38    LD DE,$3800
1FE6  AF          XOR A
1FE7  32 0F D2    LD ($D20F),A
1FEA  3E 05       LD A,$05
1FEC  CD 02 05    CALL $0502
1FEF  3A 1A D2    LD A,($D21A)
1FF2  F6 40       OR $40
1FF4  32 1A D2    LD ($D21A),A
1FF7  FD CB 00 86 RES 0,(IY+0)
1FFB  CD 27 03    CALL $0327
1FFE  3E 01       LD A,$01
2000  32 FE FF    LD ($FFFE),A
2003  32 2F D2    LD ($D22F),A
2006  3A 79 D2    LD A,($D279)
2009  FE 06       CP $06
200B  DA B0 20    JP C,$20B0
200E  06 3C       LD B,$3C
2010  C5          PUSH BC
2011  FD CB 00 86 RES 0,(IY+0)
2015  CD 27 03    CALL $0327
2018  21 00 D0    LD HL,$D000
201B  0E 78       LD C,$78
201D  06 60       LD B,$60
201F  11 55 22    LD DE,$2255
2022  CD A8 2F    CALL $2FA8
2025  22 36 D2    LD ($D236),HL
2028  C1          POP BC
2029  10 E5       DJNZ $2010
202B  3E 13       LD A,$13
202D  DF          RST $18
202E  21 69 1B    LD HL,$1B69
2031  06 3D       LD B,$3D
2033  C5          PUSH BC
2034  FD 4E 0A    LD C,(IY+10)
2037  FD CB 00 86 RES 0,(IY+0)
203B  CD 27 03    CALL $0327
203E  FD 71 0A    LD (IY+10),C
2041  FD CB 00 86 RES 0,(IY+0)
2045  CD 27 03    CALL $0327
2048  11 00 D0    LD DE,$D000
204B  ED 53 36 D2 LD ($D236),DE
204F  06 03       LD B,$03
2051  C5          PUSH BC
2052  E5          PUSH HL
2053  3E 78       LD A,$78
2055  86          ADD A,(HL)
2056  4F          LD C,A
2057  23          INC HL
2058  3E 60       LD A,$60
205A  86          ADD A,(HL)
205B  47          LD B,A
205C  23          INC HL
205D  C5          PUSH BC
205E  11 55 22    LD DE,$2255
2061  2A 36 D2    LD HL,($D236)
2064  CD A8 2F    CALL $2FA8
2067  22 36 D2    LD ($D236),HL
206A  C1          POP BC
206B  E1          POP HL
206C  7E          LD A,(HL)
206D  ED 44       NEG
206F  C6 78       ADD A,$78
2071  4F          LD C,A
2072  23          INC HL
2073  7E          LD A,(HL)
2074  ED 44       NEG
2076  C6 60       ADD A,$60
2078  47          LD B,A
2079  23          INC HL
207A  E5          PUSH HL
207B  11 55 22    LD DE,$2255
207E  2A 36 D2    LD HL,($D236)
2081  CD A8 2F    CALL $2FA8
2084  22 36 D2    LD ($D236),HL
2087  E1          POP HL
2088  C1          POP BC
2089  10 C6       DJNZ $2051
208B  C1          POP BC
208C  10 A5       DJNZ $2033
208E  21 17 17    LD HL,$1717
2091  CD B7 0A    CALL $0AB7
2094  FD 36 0A 00 LD (IY+10),$00
2098  21 6A 71    LD HL,$716A
209B  01 C0 01    LD BC,$01C0
209E  11 00 38    LD DE,$3800
20A1  AF          XOR A
20A2  32 0F D2    LD ($D20F),A
20A5  3E 05       LD A,$05
20A7  CD 02 05    CALL $0502
20AA  21 0A 0B    LD HL,$0B0A
20AD  CD B7 0A    CALL $0AB7
20B0  01 F0 00    LD BC,$00F0
20B3  CD 75 21    CALL $2175
20B6  E7          RST $20
20B7  CD 9D 44    CALL $449D
20BA  01 F0 00    LD BC,$00F0
20BD  CD 75 21    CALL $2175
20C0  CD A3 0A    CALL $0AA3
20C3  01 78 00    LD BC,$0078
20C6  CD 75 21    CALL $2175
20C9  21 1A 17    LD HL,$171A
20CC  11 00 00    LD DE,$0000
20CF  3E 0C       LD A,$0C
20D1  CD 06 04    CALL $0406
20D4  21 6F 43    LD HL,$436F
20D7  11 00 20    LD DE,$2000
20DA  3E 09       LD A,$09
20DC  CD 06 04    CALL $0406
20DF  21 2A 73    LD HL,$732A
20E2  01 06 01    LD BC,$0106
20E5  11 00 38    LD DE,$3800
20E8  AF          XOR A
20E9  32 0F D2    LD ($D20F),A
20EC  3E 05       LD A,$05
20EE  CD 02 05    CALL $0502
20F1  AF          XOR A
20F2  21 23 D3    LD HL,$D323
20F5  36 58       LD (HL),$58
20F7  23          INC HL
20F8  36 22       LD (HL),$22
20FA  23          INC HL
20FB  77          LD (HL),A
20FC  23          INC HL
20FD  36 67       LD (HL),$67
20FF  23          INC HL
2100  36 22       LD (HL),$22
2102  23          INC HL
2103  77          LD (HL),A
2104  23          INC HL
2105  36 79       LD (HL),$79
2107  23          INC HL
2108  36 22       LD (HL),$22
210A  23          INC HL
210B  77          LD (HL),A
210C  23          INC HL
210D  36 82       LD (HL),$82
210F  23          INC HL
2110  36 22       LD (HL),$22
2112  23          INC HL
2113  77          LD (HL),A
2114  01 01 00    LD BC,$0001
2117  CD 48 21    CALL $2148
211A  21 15 15    LD HL,$1515
211D  CD AB 0A    CALL $0AAB
2120  3E 0E       LD A,$0E
2122  DF          RST $18
2123  AF          XOR A
2124  32 0F D2    LD ($D20F),A
2127  21 15 23    LD HL,$2315
212A  CD C5 21    CALL $21C5
212D  01 2C 01    LD BC,$012C
2130  FD 7E 0A    LD A,(IY+10)
2133  FD CB 00 86 RES 0,(IY+0)
2137  CD 27 03    CALL $0327
213A  FD 77 0A    LD (IY+10),A
213D  0B          DEC BC
213E  78          LD A,B
213F  B1          OR C
2140  20 EE       JR NZ,$2130
2142  CD A3 0A    CALL $0AA3
2145  C3 5B 13    JP $135B

; ==== sub_2148 (4 callers) ====
2148  F5          PUSH AF
2149  E5          PUSH HL
214A  D5          PUSH DE
214B  C5          PUSH BC
214C  C5          PUSH BC
214D  FD CB 00 86 RES 0,(IY+0)
2151  CD 27 03    CALL $0327
2154  FD 36 0A 00 LD (IY+10),$00
2158  21 00 D0    LD HL,$D000
215B  22 36 D2    LD ($D236),HL
215E  21 23 D3    LD HL,$D323
2161  06 04       LD B,$04
2163  C5          PUSH BC
2164  CD 8A 21    CALL $218A
2167  C1          POP BC
2168  10 F9       DJNZ $2163
216A  C1          POP BC
216B  0B          DEC BC
216C  78          LD A,B
216D  B1          OR C
216E  20 DC       JR NZ,$214C
2170  C1          POP BC
2171  D1          POP DE
2172  E1          POP HL
2173  F1          POP AF
2174  C9          RET

; ==== sub_2175 (3 callers) ====
2175  C5          PUSH BC
2176  FD 7E 0A    LD A,(IY+10)
2179  FD CB 00 86 RES 0,(IY+0)
217D  CD 27 03    CALL $0327
2180  FD 77 0A    LD (IY+10),A
2183  C1          POP BC
2184  0B          DEC BC
2185  78          LD A,B
2186  B1          OR C
2187  20 EC       JR NZ,$2175
2189  C9          RET

; ==== sub_218A (1 caller) ====
218A  5E          LD E,(HL)
218B  23          INC HL
218C  56          LD D,(HL)
218D  23          INC HL
218E  34          INC (HL)
218F  1A          LD A,(DE)
2190  BE          CP (HL)
2191  30 1B       JR NC,$21AE
2193  36 00       LD (HL),$00
2195  13          INC DE
2196  13          INC DE
2197  13          INC DE
2198  2B          DEC HL
2199  72          LD (HL),D
219A  2B          DEC HL
219B  73          LD (HL),E
219C  23          INC HL
219D  23          INC HL
219E  1A          LD A,(DE)
219F  FE FF       CP $FF
21A1  20 0B       JR NZ,$21AE
21A3  13          INC DE
21A4  1A          LD A,(DE)
21A5  47          LD B,A
21A6  13          INC DE
21A7  1A          LD A,(DE)
21A8  2B          DEC HL
21A9  77          LD (HL),A
21AA  2B          DEC HL
21AB  70          LD (HL),B
21AC  18 DC       JR $218A
21AE  23          INC HL
21AF  13          INC DE
21B0  E5          PUSH HL
21B1  EB          EX DE,HL
21B2  5E          LD E,(HL)
21B3  23          INC HL
21B4  56          LD D,(HL)
21B5  EB          EX DE,HL
21B6  7E          LD A,(HL)
21B7  23          INC HL
21B8  5E          LD E,(HL)
21B9  23          INC HL
21BA  4D          LD C,L
21BB  44          LD B,H
21BC  6F          LD L,A
21BD  26 00       LD H,$00
21BF  54          LD D,H
21C0  CD 07 2F    CALL $2F07
21C3  E1          POP HL
21C4  C9          RET

; ==== sub_21C5 (1 caller) ====
21C5  11 BF D2    LD DE,$D2BF
21C8  ED A0       LDI
21CA  ED A0       LDI
21CC  13          INC DE
21CD  3E FF       LD A,$FF
21CF  12          LD (DE),A
21D0  7E          LD A,(HL)
21D1  23          INC HL
21D2  FE FF       CP $FF
21D4  C8          RET Z
21D5  FE FE       CP $FE
21D7  28 EC       JR Z,$21C5
21D9  FE FC       CP $FC
21DB  28 24       JR Z,$2201
21DD  FE FD       CP $FD
21DF  20 09       JR NZ,$21EA
21E1  4E          LD C,(HL)
21E2  23          INC HL
21E3  46          LD B,(HL)
21E4  23          INC HL
21E5  CD 48 21    CALL $2148
21E8  18 E6       JR $21D0
21EA  E5          PUSH HL
21EB  32 C1 D2    LD ($D2C1),A
21EE  01 09 00    LD BC,$0009
21F1  CD 48 21    CALL $2148
21F4  21 BF D2    LD HL,$D2BF
21F7  CD 12 06    CALL $0612
21FA  21 BF D2    LD HL,$D2BF
21FD  34          INC (HL)
21FE  E1          POP HL
21FF  18 CF       JR $21D0
2201  46          LD B,(HL)
2202  23          INC HL
2203  E5          PUSH HL
2204  C5          PUSH BC
2205  01 0C 00    LD BC,$000C
2208  CD 48 21    CALL $2148
220B  11 9E 3A    LD DE,$3A9E
220E  21 DE 3A    LD HL,$3ADE
2211  06 09       LD B,$09
2213  C5          PUSH BC
2214  E5          PUSH HL
2215  D5          PUSH DE
2216  06 14       LD B,$14
2218  F3          DI
2219  7D          LD A,L
221A  D3 BF       OUT ($BF),A
221C  7C          LD A,H
221D  D3 BF       OUT ($BF),A
221F  DD E5       PUSH IX
2221  DD E1       POP IX
2223  DB BE       IN A,($BE)
2225  4F          LD C,A
2226  DD E5       PUSH IX
2228  DD E1       POP IX
222A  7B          LD A,E
222B  D3 BF       OUT ($BF),A
222D  7A          LD A,D
222E  F6 40       OR $40
2230  D3 BF       OUT ($BF),A
2232  DD E5       PUSH IX
2234  DD E1       POP IX
2236  79          LD A,C
2237  D3 BE       OUT ($BE),A
2239  DD E5       PUSH IX
223B  DD E1       POP IX
223D  FB          EI
223E  23          INC HL
223F  13          INC DE
2240  10 D6       DJNZ $2218
2242  D1          POP DE
2243  E1          POP HL
2244  01 40 00    LD BC,$0040
2247  09          ADD HL,BC
2248  EB          EX DE,HL
2249  09          ADD HL,BC
224A  EB          EX DE,HL
224B  C1          POP BC
224C  10 C5       DJNZ $2213
224E  C1          POP BC
224F  10 B3       DJNZ $2204
2251  E1          POP HL
2252  C3 D0 21    JP $21D0
2255  .byte 5C 5E FF E9 12 23 6F AF 22 96 12 23 86 AF 22 FF ; \^...#o."..#..".
2265  .byte 58 22 48 CA 22 54 B8 22 1E C1 22 44 CA 22 36 C1 ; X"H."T.".."D."6.
2275  .byte 22 FF 67 22 23 D3 22 23 DC 22 FF 79 22 E4 03 23 ; ".g"#."#.".y"..#
2285  .byte 19 F4 22 19 E5 22 19 F4 22 19 E5 22 FA 03 23 85 ; ..".."..".."..#.
2295  .byte F4 22 E8 03 23 19 E5 22 19 F4 22 19 E5 22 19 F4 ; ."..#.."..".."..
22A5  .byte 22 19 E5 22 19 F4 22 FF 82 22 48 48 50 FF FF FF ; ".."..".."HHP...
22B5  .byte FF FF FF 48 58 4A FF FF FF FF FF FF 48 58 4C FF ; ...HXJ......HXL.
22C5  .byte FF FF FF FF FF 48 58 4E FF FF FF FF FF FF 48 78 ; .....HXN......Hx
22D5  .byte 6A 6C 6E FF FF FF FF 48 78 70 72 74 FF FF FF FF ; jln....Hxprt....
22E5  .byte 50 50 0A 0C FF FF FF FF 2A 2C FF FF FF FF FF 50 ; PP......*,.....P
22F5  .byte 50 0E 10 FF FF FF FF 2E 30 FF FF FF FF FF 50 60 ; P.......0.....P`
2305  .byte 12 14 FF FF FF FF 32 34 FF FF FF FF FF 48 48 FF ; ......24.....HH.
2315  .byte 11 04 AE 9E 7F 5E 2E FE 12 05 AF 4F 3E FE 10 06 ; .....^.....O>...
2325  .byte 4F 3E 2F 4E 3E 4F 9E 4E FD 3C 00 FE 0F 0C 4E 1E ; O>/N>O.N.<....N.
2335  .byte 7E 3E FE 10 0D 4E 3E 1E 9F FE 11 0E BF 3E 9F AE ; ~>...N>......>..
2345  .byte 5E 9E 7F FD B4 00 FC 09 FE 11 0E AE AF 1E 3F 3F ; ^.............??
2355  .byte FD F0 00 FC 09 FE 0F 0D 4E 1E 7E 3E FE 10 0E 8F ; ........N.~>....
2365  .byte 9F 9E 4E 9F 1E 7E 3E 9F FD 78 00 FC 02 FE 0F 0F ; ..N..~>..x......
2375  .byte AE 4F 5E 7F 9E 1F BE FE 10 10 E9 8F 5E 7F 1F 9E ; .O^.........^...
2385  .byte AF E9 FE 12 11 4F 1E DE 1E AE 4F 5E FD F0 00 FC ; .....O....O^....
2395  .byte 09 FE 0F 0D 4E 9F 1E 8F 4F 5E 2E FE 11 0E 2F 3E ; ....N...O^..../>
23A5  .byte AE 5E 4E 7F 3E 9F FD 78 00 FC 02 FE 10 0E 1E DE ; .^N.>..x........
23B5  .byte 1E 7F 9E FE 11 0F 6E 9E AE 4F 5E 9F 9E FD 3C 00 ; ......n..O^...<.
23C5  .byte FE 10 11 AF 1E CF 3E 3F BE 7F 5E FE 11 12 DE BE ; ......>?..^.....
23D5  .byte 7F 9E BE 3E FD F0 00 FC 09 FE 0F 0D AE 9E BE 7F ; ...>............
23E5  .byte 2F FE 10 0E 8F 9F 9E 2F BE 2E 3E 9F FD 78 00 FC ; /....../..>..x..
23F5  .byte 02 FE 10 0F 7E 1E AE 1E AF 9E FE 11 10 7F 1E CF ; ....~...........
2405  .byte 1E 7E BE 9F 1E FD F0 00 FC 09 FE 0F 0C 9F 3E E8 ; .~............>.
2415  .byte FE 0F 0D 1E 9F 9F 1E 7F 4E 5E 7F 4E FE 12 0E 1E ; ........N^.N....
2425  .byte 7F 2F FE 0F 0F 9E 9F 5E 4E 5E 7F 1E 6F FE 13 10 ; ./.....^N^..o...
2435  .byte 7E BE AE 5E 2E FD 78 00 FC 01 FE 10 11 DE BE DF ; ~..^..x.........
2445  .byte 9E FE 11 12 6E 9E AE 4F 5E 9F 9E FD F0 00 FC 09 ; ....n..O^.......
2455  .byte FE 10 0D AE 8F 3E 2E 5E 1E 6F FE 12 0E AF 4F 1E ; .....>.^.o....O.
2465  .byte 7F 6E AE FD 78 00 FC 02 FE 10 0F DE 8E AE 4F 5E ; .n..x.........O^
2475  .byte 8E EB DE FD 3C 00 FE 10 11 6F BE 7F 1E 9F 5E 1E ; ....<....o....^.
2485  .byte 7F FE 17 12 AE 4E FD B4 00 FC 09 FE 0F 0C 8F 9F ; .....N..........
2495  .byte 3E AE 3E 7F AF 3E 2F FE 13 0E 1F DE FE 12 10 AE ; >.>..>/.........
24A5  .byte 3E 4E 1E FD 96 00 FE 16 13 3E 7F 2F FF D0 4A E1 ; >N.......>./..J.
24B5  .byte 5D B1 5E DD 5E AF 5F D7 5F 83 61 F8 61 F9 65 47 ; ].^.^._._.a.a.eG
24C5  .byte 67 3E 69 ED 69 55 6A 26 6B D9 6B CA 6D 65 6E 61 ; g>i.iUj&k.k.mena
24D5  .byte 6F 65 70 4A 9B BD 9B 45 9C 63 9C C3 9D 2B 9F EE ; oepJ...E.c...+..
24E5  .byte 9F B1 A0 73 A1 05 A3 C1 A3 74 A4 1A A5 FA 96 D0 ; ...s.....t......
24F5  .byte 9A B6 A7 D8 76 D3 75 6B 73 25 7D 31 7E C8 7E FC ; ....v.uks%}1~.~.
2505  .byte 7E AA 96 2D 82 6B 80 FB 82 D6 83 A7 94 8D A9 30 ; ~..-.k.........0
2515  .byte AA E7 AA 32 AD FB AD 4E AE BA B0 32 B1 5D B2 72 ; ...2...N...2.].r
2525  .byte B3 47 B4 E8 B4 4C 88 10 89 08 8B 28 8C 59 8D 6A ; .G...L.....(.Y.j
2535  .byte 8E CF 8E 71 8F 72 8F C5 90 F2 BC F2 BC AB 84 71 ; ...q.r.........q
2545  .byte 92 0E B6 E3 7A 68 98 00 00 81 86 00 00 29 7B 10 ; ....zh.......){.
2555  .byte 60 61 60 03 BE 53 BF D1 7B 95 BB 00 01 00 02 00 ; `a`..S..{.......
2565  .byte 01 00 02 20 00 20 01 20 00 E0 00 20 00 20 01 20 ; ... . . ... . . 
2575  .byte 00 E0 00 20 00 20 01 20 00 E0 00 20 00 20 01 20 ; ... . . ... . . 
2585  .byte 00 E0 00 20 00 20 01 20 00 E0 00 20 00 20 01 20 ; ... . . ... . . 
2595  .byte 00 E0 00 20 00 20 01 60 00 E0 00 10 00 10 01 20 ; ... . .`....... 
25A5  .byte 00 E0 00 A0 00 A0 01 40 00 00 01 40 00 40 01 40 ; .......@...@.@.@
25B5  .byte 00 00 01 20 00 20 01 20 00 E0 00 20 00 20 01 30 ; ... . . ... . .0
25C5  .byte 00 F0 00 00 01 00 02 00 01 C0 01 40 00 40 01 40 ; ...........@.@.@
25D5  .byte 00 00 01 A0 00 A0 01 20 00 E0 00 10 00 10 01 10 ; ....... ........
25E5  .byte 00 D0 00 10 00 10 01 10 00 D0 00 C0 00 C0 01 80 ; ................
25F5  .byte 00 40 01 20 00 20 01 20 00 E0 00 08 00 40 01 10 ; .@. . . .....@..
2605  .byte 00 D0 00 40 00 08 01 10 00 D0 00 10 00 10 01 20 ; ...@........... 
2615  .byte 00 E0 00 20 00 20 01 30 00 CC 00 20 00 20 01 30 ; ... . .0... . .0
2625  .byte 00 CC 00 20 00 20 01 30 00 CC 00 20 00 20 01 20 ; ... . .0... . . 
2635  .byte 00 DA 00 30 00 30 01 30 00 F0 00 00 01 80 01 00 ; ...0.0.0........
2645  .byte 01 C0 01 10 00 10 01 10 00 D0 00 20 00 20 01 30 ; ........... . .0
2655  .byte 00 C8 00 20 00 20 01 20 00 E0 00 20 00 20 01 20 ; ... . . ... . . 
2665  .byte 00 E0 00 20 00 20 01 80 00 40 01 10 00 10 01 80 ; ... . ...@......
2675  .byte 00 F0 00 20 00 20 01 10 00 D0 00 20 00 20 01 10 ; ... . ..... . ..
2685  .byte 00 D0 00 20 00 20 01 20 00 E0 00 10 00 10 01 60 ; ... . . .......`
2695  .byte 00 00 01 10 00 10 01 00 01 C0 01 10 00 10 01 00 ; ................
26A5  .byte 01 C0 01 10 00 10 01 10 00 D0 00 20 00 20 01 20 ; ........... . . 
26B5  .byte 00 E0 00 10 00 10 01 10 00 D0 00 40 00 40 01 C0 ; ...........@.@..
26C5  .byte 00 80 01 10 00 10 01 10 00 D0 00 80 00 80 01 40 ; ...............@
26D5  .byte 00 C0 01 20 00 20 01 20 00 E0 00 00 08 00 08 30 ; ... . . .......0
26E5  .byte 00 F0 00 10 00 10 01 20 00 E0 00 20 00 20 01 20 ; ....... ... . . 
26F5  .byte 00 E0 00 00 00 00 01 00 00 C0 00 00 02 00 03 00 ; ................
2705  .byte 02 C0 02 10 00 10 01 10 00 D0 00 40 00 40 01 40 ; ...........@.@.@
2715  .byte 00 00 01 10 00 10 01 10 00 D0 00 40 00 40 01 20 ; ...........@.@. 
2725  .byte 00 E0 00 80 00 80 01 50 00 D0 00 10 00 10 01 10 ; .......P........
2735  .byte 00 D0 00 10 00 10 01 80 00 20 01 10 00 10 01 10 ; ......... ......
2745  .byte 00 D0 00 60 00 60 01 60 00 20 01 10 00 10 01 10 ; ...`.`.`. ......
2755  .byte 00 D0 00 20 00 20 01 20 00 E0 00 00 20 00 21 20 ; ... . . .... .! 
2765  .byte 00 E0 00 08 00 08 01 08 00 C8 00 20 00 20 01 20 ; ........... . . 
2775  .byte 00 E0 00 20 00 20 01 20 00 E0 00 20 00 20 01 20 ; ... . . ... . . 
2785  .byte 00 E0 00 28 00 28 01 28 00 E8 00 60 00 60 01 20 ; ...(.(.(...`.`. 
2795  .byte 00 E0 00 00 01 00 02 00 01 C0 01 10 00 10 01 10 ; ................
27A5  .byte 00 D0 00 10 00 10 01 00 01 C0 01 10 00 10 01 10 ; ................
27B5  .byte 00 D0 00 10 00 10 01 10 00 D0 00 20 00 20 01 20 ; ........... . . 
27C5  .byte 00 E0 00 20 00 20 01 20 00 E0 00 38 00 28 01 30 ; ... . . ...8.(.0
27D5  .byte 00 F0 00 20 00 20 01 20 00 E0 00 10 00 10 01 10 ; ... . . ........
27E5  .byte 00 D0 00 20 00 20 01 20 00 E0 00 20 00 20 01 20 ; ... . . ... . . 
27F5  .byte 00 E0 00 00 01 E0 01 C0 00 80 01 00 01 00 02 00 ; ................
2805  .byte 01 C0 01 00 08 00 09 00 08 C0 08 20 00 20 01 20 ; ........... . . 
2815  .byte 00 E0 00 A6 A8 FF A0 A2 FF                      ; .........

; ==== sub_281E (2 callers) ====
281E  FD CB 07 BE RES 7,(IY+7)
2822  21 1B 28    LD HL,$281B
2825  11 BF D2    LD DE,$D2BF
2828  01 04 00    LD BC,$0004
282B  ED B0       LDIR
282D  0E 30       LD C,$30
282F  06 97       LD B,$97
2831  2A 36 D2    LD HL,($D236)
2834  11 BF D2    LD DE,$D2BF
2837  CD A8 2F    CALL $2FA8
283A  22 36 D2    LD ($D236),HL
283D  3A 40 D2    LD A,($D240)
2840  FE 09       CP $09
2842  38 02       JR C,$2846
2844  3E 09       LD A,$09
2846  87          ADD A,A
2847  C6 80       ADD A,$80
2849  32 BF D2    LD ($D2BF),A
284C  3E FF       LD A,$FF
284E  32 C0 D2    LD ($D2C0),A
2851  0E 42       LD C,$42
2853  06 97       LD B,$97
2855  2A 36 D2    LD HL,($D236)
2858  11 BF D2    LD DE,$D2BF
285B  CD A8 2F    CALL $2FA8
285E  22 36 D2    LD ($D236),HL
2861  FD CB 05 56 BIT 2,(IY+5)
2865  C4 D3 28    CALL NZ,$28D3
2868  FD CB 07 6E BIT 5,(IY+7)
286C  C4 0C 29    CALL NZ,$290C
286F  11 50 00    LD DE,$0050
2872  3A 15 D4    LD A,($D415)
2875  E6 02       AND $02
2877  28 03       JR Z,$287C
2879  11 90 00    LD DE,$0090
287C  21 61 D2    LD HL,$D261
287F  7E          LD A,(HL)
2880  23          INC HL
2881  B6          OR (HL)
2882  CC 0E 2B    CALL Z,$2B0E
2885  23          INC HL
2886  11 58 00    LD DE,$0058
2889  3A 15 D4    LD A,($D415)
288C  E6 02       AND $02
288E  28 03       JR Z,$2893
2890  11 98 00    LD DE,$0098
2893  7E          LD A,(HL)
2894  23          INC HL
2895  B6          OR (HL)
2896  CC 0E 2B    CALL Z,$2B0E
2899  23          INC HL
289A  11 40 00    LD DE,$0040
289D  7E          LD A,(HL)
289E  23          INC HL
289F  B6          OR (HL)
28A0  CC 0E 2B    CALL Z,$2B0E
28A3  23          INC HL
28A4  11 60 00    LD DE,$0060
28A7  FD CB 05 76 BIT 6,(IY+5)
28AB  28 03       JR Z,$28B0
28AD  11 80 00    LD DE,$0080
28B0  7E          LD A,(HL)
28B1  23          INC HL
28B2  B6          OR (HL)
28B3  CC 0E 2B    CALL Z,$2B0E
28B6  FD CB 05 46 BIT 0,(IY+5)
28BA  CC 53 29    CALL Z,$2953
28BD  21 00 00    LD HL,$0000
28C0  22 61 D2    LD ($D261),HL
28C3  22 63 D2    LD ($D263),HL
28C6  22 65 D2    LD ($D265),HL
28C9  22 67 D2    LD ($D267),HL
28CC  CD D8 2B    CALL $2BD8
28CF  CD 8D 2C    CALL $2C8D
28D2  C9          RET

; ==== sub_28D3 (1 caller) ====
28D3  3A A9 D2    LD A,($D2A9)
28D6  4F          LD C,A
28D7  0F          RRCA
28D8  0F          RRCA
28D9  0F          RRCA
28DA  0F          RRCA
28DB  E6 0F       AND $0F
28DD  87          ADD A,A
28DE  C6 80       ADD A,$80
28E0  32 BF D2    LD ($D2BF),A
28E3  79          LD A,C
28E4  E6 0F       AND $0F
28E6  87          ADD A,A
28E7  C6 80       ADD A,$80
28E9  32 C0 D2    LD ($D2C0),A
28EC  3E FF       LD A,$FF
28EE  32 C1 D2    LD ($D2C1),A
28F1  0E 2E       LD C,$2E
28F3  06 18       LD B,$18
28F5  2A 36 D2    LD HL,($D236)
28F8  11 18 28    LD DE,$2818
28FB  CD A8 2F    CALL $2FA8
28FE  0E 42       LD C,$42
2900  06 18       LD B,$18
2902  11 BF D2    LD DE,$D2BF
2905  CD A8 2F    CALL $2FA8
2908  22 36 D2    LD ($D236),HL
290B  C9          RET

; ==== sub_290C (1 caller) ====
290C  21 BF D2    LD HL,$D2BF
290F  3A CF D2    LD A,($D2CF)
2912  E6 0F       AND $0F
2914  87          ADD A,A
2915  C6 80       ADD A,$80
2917  77          LD (HL),A
2918  23          INC HL
2919  36 B0       LD (HL),$B0
291B  23          INC HL
291C  3A D0 D2    LD A,($D2D0)
291F  4F          LD C,A
2920  CB 3F       SRL A
2922  CB 3F       SRL A
2924  CB 3F       SRL A
2926  CB 3F       SRL A
2928  87          ADD A,A
2929  C6 80       ADD A,$80
292B  77          LD (HL),A
292C  23          INC HL
292D  79          LD A,C
292E  E6 0F       AND $0F
2930  87          ADD A,A
2931  C6 80       ADD A,$80
2933  77          LD (HL),A
2934  23          INC HL
2935  36 FF       LD (HL),$FF
2937  0E 32       LD C,$32
2939  06 28       LD B,$28
293B  3A 38 D2    LD A,($D238)
293E  FE 1C       CP $1C
2940  38 04       JR C,$2946
2942  0E 70       LD C,$70
2944  06 28       LD B,$28
2946  2A 36 D2    LD HL,($D236)
2949  11 BF D2    LD DE,$D2BF
294C  CD A8 2F    CALL $2FA8
294F  22 36 D2    LD ($D236),HL
2952  C9          RET

; ==== sub_2953 (1 caller) ====
2953  FD CB 07 76 BIT 6,(IY+7)
2957  C0          RET NZ
2958  2A 75 D2    LD HL,($D275)
295B  7D          LD A,L
295C  B4          OR H
295D  C4 34 2B    CALL NZ,$2B34
2960  2A 77 D2    LD HL,($D277)
2963  7D          LD A,L
2964  B4          OR H
2965  C4 16 2B    CALL NZ,$2B16
2968  2A 61 D2    LD HL,($D261)
296B  ED 5B 59 D2 LD DE,($D259)
296F  A7          AND A
2970  ED 52       SBC HL,DE
2972  C4 52 2B    CALL NZ,$2B52
2975  ED 53 59 D2 LD ($D259),DE
2979  2A 63 D2    LD HL,($D263)
297C  ED 5B 5B D2 LD DE,($D25B)
2980  A7          AND A
2981  ED 52       SBC HL,DE
2983  C4 52 2B    CALL NZ,$2B52
2986  ED 53 5B D2 LD ($D25B),DE
298A  2A 65 D2    LD HL,($D265)
298D  ED 5B 5D D2 LD DE,($D25D)
2991  A7          AND A
2992  ED 52       SBC HL,DE
2994  C4 52 2B    CALL NZ,$2B52
2997  ED 53 5D D2 LD ($D25D),DE
299B  2A 67 D2    LD HL,($D267)
299E  ED 5B 5F D2 LD DE,($D25F)
29A2  A7          AND A
29A3  ED 52       SBC HL,DE
29A5  C4 52 2B    CALL NZ,$2B52
29A8  ED 53 5F D2 LD ($D25F),DE
29AC  ED 4B 59 D2 LD BC,($D259)
29B0  ED 5B FF D3 LD DE,($D3FF)
29B4  2A 54 D2    LD HL,($D254)
29B7  09          ADD HL,BC
29B8  A7          AND A
29B9  ED 52       SBC HL,DE
29BB  38 2A       JR C,$29E7
29BD  7C          LD A,H
29BE  A7          AND A
29BF  20 05       JR NZ,$29C6
29C1  7D          LD A,L
29C2  FE 09       CP $09
29C4  38 03       JR C,$29C9
29C6  21 08 00    LD HL,$0008
29C9  FD CB 05 5E BIT 3,(IY+5)
29CD  20 51       JR NZ,$2A20
29CF  FD CB 05 6E BIT 5,(IY+5)
29D3  28 03       JR Z,$29D8
29D5  21 01 00    LD HL,$0001
29D8  EB          EX DE,HL
29D9  2A 54 D2    LD HL,($D254)
29DC  A7          AND A
29DD  ED 52       SBC HL,DE
29DF  38 3F       JR C,$2A20
29E1  22 54 D2    LD ($D254),HL
29E4  C3 20 2A    JP $2A20
29E7  ED 4B 5B D2 LD BC,($D25B)
29EB  2A 54 D2    LD HL,($D254)
29EE  09          ADD HL,BC
29EF  A7          AND A
29F0  ED 52       SBC HL,DE
29F2  30 2C       JR NC,$2A20
29F4  7D          LD A,L
29F5  2F          CPL
29F6  6F          LD L,A
29F7  7C          LD A,H
29F8  2F          CPL
29F9  67          LD H,A
29FA  23          INC HL
29FB  7C          LD A,H
29FC  A7          AND A
29FD  20 05       JR NZ,$2A04
29FF  7D          LD A,L
2A00  FE 09       CP $09
2A02  38 03       JR C,$2A07
2A04  21 08 00    LD HL,$0008
2A07  FD CB 05 5E BIT 3,(IY+5)
2A0B  20 13       JR NZ,$2A20
2A0D  FD CB 05 6E BIT 5,(IY+5)
2A11  28 03       JR Z,$2A16
2A13  21 01 00    LD HL,$0001
2A16  ED 5B 54 D2 LD DE,($D254)
2A1A  19          ADD HL,DE
2A1B  38 03       JR C,$2A20
2A1D  22 54 D2    LD ($D254),HL
2A20  2A 54 D2    LD HL,($D254)
2A23  ED 5B 6D D2 LD DE,($D26D)
2A27  A7          AND A
2A28  ED 52       SBC HL,DE
2A2A  30 06       JR NC,$2A32
2A2C  ED 53 54 D2 LD ($D254),DE
2A30  18 10       JR $2A42
2A32  2A 54 D2    LD HL,($D254)
2A35  ED 5B 6F D2 LD DE,($D26F)
2A39  A7          AND A
2A3A  ED 52       SBC HL,DE
2A3C  38 04       JR C,$2A42
2A3E  ED 53 54 D2 LD ($D254),DE
2A42  FD CB 05 76 BIT 6,(IY+5)
2A46  C4 58 2B    CALL NZ,$2B58
2A49  ED 4B 5D D2 LD BC,($D25D)
2A4D  ED 5B 02 D4 LD DE,($D402)
2A51  2A 57 D2    LD HL,($D257)
2A54  FD CB 05 76 BIT 6,(IY+5)
2A58  C4 C3 2B    CALL NZ,$2BC3
2A5B  FD CB 05 7E BIT 7,(IY+5)
2A5F  C4 CB 2B    CALL NZ,$2BCB
2A62  09          ADD HL,BC
2A63  FD CB 05 7E BIT 7,(IY+5)
2A67  CC CD 2B    CALL Z,$2BCD
2A6A  A7          AND A
2A6B  ED 52       SBC HL,DE
2A6D  38 37       JR C,$2AA6
2A6F  0E 09       LD C,$09
2A71  7C          LD A,H
2A72  A7          AND A
2A73  20 0B       JR NZ,$2A80
2A75  FD CB 05 76 BIT 6,(IY+5)
2A79  C4 13 2B    CALL NZ,$2B13
2A7C  7D          LD A,L
2A7D  B9          CP C
2A7E  38 04       JR C,$2A84
2A80  0D          DEC C
2A81  69          LD L,C
2A82  26 00       LD H,$00
2A84  FD CB 05 7E BIT 7,(IY+5)
2A88  28 0D       JR Z,$2A97
2A8A  CB 3C       SRL H
2A8C  CB 1D       RR L
2A8E  FD CB 08 4E BIT 1,(IY+8)
2A92  20 03       JR NZ,$2A97
2A94  21 00 00    LD HL,$0000
2A97  EB          EX DE,HL
2A98  2A 57 D2    LD HL,($D257)
2A9B  A7          AND A
2A9C  ED 52       SBC HL,DE
2A9E  38 4D       JR C,$2AED
2AA0  22 57 D2    LD ($D257),HL
2AA3  C3 ED 2A    JP $2AED
2AA6  ED 4B 5F D2 LD BC,($D25F)
2AAA  2A 57 D2    LD HL,($D257)
2AAD  09          ADD HL,BC
2AAE  FD CB 05 76 BIT 6,(IY+5)
2AB2  C4 C7 2B    CALL NZ,$2BC7
2AB5  FD CB 05 7E BIT 7,(IY+5)
2AB9  CC CD 2B    CALL Z,$2BCD
2ABC  A7          AND A
2ABD  ED 52       SBC HL,DE
2ABF  30 2C       JR NC,$2AED
2AC1  7D          LD A,L
2AC2  2F          CPL
2AC3  6F          LD L,A
2AC4  7C          LD A,H
2AC5  2F          CPL
2AC6  67          LD H,A
2AC7  23          INC HL
2AC8  0E 09       LD C,$09
2ACA  7C          LD A,H
2ACB  A7          AND A
2ACC  20 0B       JR NZ,$2AD9
2ACE  FD CB 05 76 BIT 6,(IY+5)
2AD2  C4 13 2B    CALL NZ,$2B13
2AD5  7D          LD A,L
2AD6  B9          CP C
2AD7  38 04       JR C,$2ADD
2AD9  0D          DEC C
2ADA  69          LD L,C
2ADB  26 00       LD H,$00
2ADD  FD CB 05 66 BIT 4,(IY+5)
2AE1  20 0A       JR NZ,$2AED
2AE3  ED 5B 57 D2 LD DE,($D257)
2AE7  19          ADD HL,DE
2AE8  38 03       JR C,$2AED
2AEA  22 57 D2    LD ($D257),HL
2AED  2A 57 D2    LD HL,($D257)
2AF0  ED 5B 71 D2 LD DE,($D271)
2AF4  A7          AND A
2AF5  ED 52       SBC HL,DE
2AF7  30 04       JR NC,$2AFD
2AF9  ED 53 57 D2 LD ($D257),DE
2AFD  2A 57 D2    LD HL,($D257)
2B00  ED 5B 73 D2 LD DE,($D273)
2B04  A7          AND A
2B05  ED 52       SBC HL,DE
2B07  38 04       JR C,$2B0D
2B09  ED 53 57 D2 LD ($D257),DE
2B0D  C9          RET

; ==== sub_2B0E (4 callers) ====
2B0E  72          LD (HL),D
2B0F  2B          DEC HL
2B10  73          LD (HL),E
2B11  23          INC HL
2B12  C9          RET

; ==== sub_2B13 (2 callers) ====
2B13  0E 08       LD C,$08
2B15  C9          RET

; ==== sub_2B16 (1 caller) ====
2B16  ED 5B 71 D2 LD DE,($D271)
2B1A  A7          AND A
2B1B  ED 52       SBC HL,DE
2B1D  C8          RET Z
2B1E  38 0A       JR C,$2B2A
2B20  13          INC DE
2B21  ED 53 71 D2 LD ($D271),DE
2B25  ED 53 73 D2 LD ($D273),DE
2B29  C9          RET
2B2A  1B          DEC DE
2B2B  ED 53 71 D2 LD ($D271),DE
2B2F  ED 53 73 D2 LD ($D273),DE
2B33  C9          RET

; ==== sub_2B34 (1 caller) ====
2B34  ED 5B 6D D2 LD DE,($D26D)
2B38  A7          AND A
2B39  ED 52       SBC HL,DE
2B3B  C8          RET Z
2B3C  38 0A       JR C,$2B48
2B3E  13          INC DE
2B3F  ED 53 6D D2 LD ($D26D),DE
2B43  ED 53 6F D2 LD ($D26F),DE
2B47  C9          RET
2B48  1B          DEC DE
2B49  ED 53 6D D2 LD ($D26D),DE
2B4D  ED 53 6F D2 LD ($D26F),DE
2B51  C9          RET

; ==== sub_2B52 (4 callers) ====
2B52  38 02       JR C,$2B56
2B54  13          INC DE
2B55  C9          RET
2B56  1B          DEC DE
2B57  C9          RET

; ==== sub_2B58 (1 caller) ====
2B58  2A 98 D2    LD HL,($D298)
2B5B  ED 5B 9A D2 LD DE,($D29A)
2B5F  19          ADD HL,DE
2B60  01 00 02    LD BC,$0200
2B63  7C          LD A,H
2B64  A7          AND A
2B65  F2 6D 2B    JP P,$2B6D
2B68  ED 44       NEG
2B6A  01 00 FE    LD BC,$FE00
2B6D  FE 02       CP $02
2B6F  38 02       JR C,$2B73
2B71  69          LD L,C
2B72  60          LD H,B
2B73  22 98 D2    LD ($D298),HL
2B76  4D          LD C,L
2B77  44          LD B,H
2B78  2A 56 D2    LD HL,($D256)
2B7B  3A 58 D2    LD A,($D258)
2B7E  09          ADD HL,BC
2B7F  1E 00       LD E,$00
2B81  CB 78       BIT 7,B
2B83  28 02       JR Z,$2B87
2B85  1E FF       LD E,$FF
2B87  8B          ADC A,E
2B88  22 56 D2    LD ($D256),HL
2B8B  32 58 D2    LD ($D258),A
2B8E  2A 9C D2    LD HL,($D29C)
2B91  3A 9E D2    LD A,($D29E)
2B94  09          ADD HL,BC
2B95  8B          ADC A,E
2B96  22 9C D2    LD ($D29C),HL
2B99  32 9E D2    LD ($D29E),A
2B9C  2A 9D D2    LD HL,($D29D)
2B9F  CB 7C       BIT 7,H
2BA1  28 0F       JR Z,$2BB2
2BA3  01 E0 FF    LD BC,$FFE0
2BA6  A7          AND A
2BA7  ED 42       SBC HL,BC
2BA9  30 07       JR NC,$2BB2
2BAB  21 02 00    LD HL,$0002
2BAE  22 9A D2    LD ($D29A),HL
2BB1  C9          RET
2BB2  2A 9D D2    LD HL,($D29D)
2BB5  01 20 00    LD BC,$0020
2BB8  A7          AND A
2BB9  ED 42       SBC HL,BC
2BBB  D8          RET C
2BBC  21 FE FF    LD HL,$FFFE
2BBF  22 9A D2    LD ($D29A),HL
2BC2  C9          RET

; ==== sub_2BC3 (1 caller) ====
2BC3  01 20 00    LD BC,$0020
2BC6  C9          RET

; ==== sub_2BC7 (1 caller) ====
2BC7  01 3F 00    LD BC,$003F
2BCA  C9          RET

; ==== sub_2BCB (1 caller) ====
2BCB  C9          RET
2BCC  .byte C9                                              ; .

; ==== sub_2BCD (2 callers) ====
2BCD  FD CB 05 76 BIT 6,(IY+5)
2BD1  C0          RET NZ
2BD2  ED 4B B8 D2 LD BC,($D2B8)
2BD6  09          ADD HL,BC
2BD7  C9          RET

; ==== sub_2BD8 (1 caller) ====
2BD8  3A 24 D2    LD A,($D224)
2BDB  E6 07       AND $07
2BDD  4F          LD C,A
2BDE  21 68 00    LD HL,$0068
2BE1  CD 5F 06    CALL $065F
2BE4  11 FD D3    LD DE,$D3FD
2BE7  19          ADD HL,DE
2BE8  EB          EX DE,HL
2BE9  3A 24 D2    LD A,($D224)
2BEC  E6 07       AND $07
2BEE  87          ADD A,A
2BEF  87          ADD A,A
2BF0  87          ADD A,A
2BF1  4F          LD C,A
2BF2  06 00       LD B,$00
2BF4  21 7D D3    LD HL,$D37D
2BF7  09          ADD HL,BC
2BF8  48          LD C,B
2BF9  06 04       LD B,$04
2BFB  1A          LD A,(DE)
2BFC  FE 57       CP $57
2BFE  D2 7D 2C    JP NC,$2C7D
2C01  D5          PUSH DE
2C02  DD E1       POP IX
2C04  D9          EXX
2C05  87          ADD A,A
2C06  6F          LD L,A
2C07  26 00       LD H,$00
2C09  29          ADD HL,HL
2C0A  29          ADD HL,HL
2C0B  11 60 25    LD DE,$2560
2C0E  19          ADD HL,DE
2C0F  4E          LD C,(HL)
2C10  23          INC HL
2C11  46          LD B,(HL)
2C12  23          INC HL
2C13  11 0F D2    LD DE,$D20F
2C16  ED A0       LDI
2C18  ED A0       LDI
2C1A  ED A0       LDI
2C1C  ED A0       LDI
2C1E  ED A0       LDI
2C20  ED A0       LDI
2C22  2A 54 D2    LD HL,($D254)
2C25  AF          XOR A
2C26  ED 42       SBC HL,BC
2C28  30 03       JR NC,$2C2D
2C2A  6F          LD L,A
2C2B  67          LD H,A
2C2C  AF          XOR A
2C2D  DD 5E 02    LD E,(IX+2)
2C30  DD 56 03    LD D,(IX+3)
2C33  ED 52       SBC HL,DE
2C35  D2 7C 2C    JP NC,$2C7C
2C38  2A 0F D2    LD HL,($D20F)
2C3B  ED 4B 54 D2 LD BC,($D254)
2C3F  09          ADD HL,BC
2C40  AF          XOR A
2C41  ED 52       SBC HL,DE
2C43  DA 7C 2C    JP C,$2C7C
2C46  2A 57 D2    LD HL,($D257)
2C49  ED 4B 11 D2 LD BC,($D211)
2C4D  ED 42       SBC HL,BC
2C4F  30 03       JR NC,$2C54
2C51  6F          LD L,A
2C52  67          LD H,A
2C53  AF          XOR A
2C54  DD 5E 05    LD E,(IX+5)
2C57  DD 56 06    LD D,(IX+6)
2C5A  ED 52       SBC HL,DE
2C5C  D2 7C 2C    JP NC,$2C7C
2C5F  2A 13 D2    LD HL,($D213)
2C62  ED 4B 57 D2 LD BC,($D257)
2C66  09          ADD HL,BC
2C67  AF          XOR A
2C68  ED 52       SBC HL,DE
2C6A  DA 7C 2C    JP C,$2C7C
2C6D  D9          EXX
2C6E  73          LD (HL),E
2C6F  23          INC HL
2C70  72          LD (HL),D
2C71  23          INC HL
2C72  E5          PUSH HL
2C73  21 1A 00    LD HL,$001A
2C76  19          ADD HL,DE
2C77  EB          EX DE,HL
2C78  E1          POP HL
2C79  10 80       DJNZ $2BFB
2C7B  C9          RET
2C7C  D9          EXX
2C7D  71          LD (HL),C
2C7E  23          INC HL
2C7F  71          LD (HL),C
2C80  23          INC HL
2C81  E5          PUSH HL
2C82  21 1A 00    LD HL,$001A
2C85  19          ADD HL,DE
2C86  EB          EX DE,HL
2C87  E1          POP HL
2C88  05          DEC B
2C89  C2 FB 2B    JP NZ,$2BFB
2C8C  C9          RET

; ==== sub_2C8D (1 caller) ====
2C8D  21 7F D3    LD HL,$D37F
2C90  06 1F       LD B,$1F
2C92  5E          LD E,(HL)
2C93  23          INC HL
2C94  56          LD D,(HL)
2C95  23          INC HL
2C96  7B          LD A,E
2C97  B2          OR D
2C98  C4 BA 2C    CALL NZ,$2CBA
2C9B  10 F5       DJNZ $2C92
2C9D  FD 7E 0A    LD A,(IY+10)
2CA0  2A 36 D2    LD HL,($D236)
2CA3  F5          PUSH AF
2CA4  E5          PUSH HL
2CA5  21 24 D0    LD HL,$D024
2CA8  22 36 D2    LD ($D236),HL
2CAB  11 FD D3    LD DE,$D3FD
2CAE  CD BA 2C    CALL $2CBA
2CB1  E1          POP HL
2CB2  F1          POP AF
2CB3  22 36 D2    LD ($D236),HL
2CB6  FD 77 0A    LD (IY+10),A
2CB9  C9          RET

; ==== sub_2CBA (2 callers) ====
2CBA  1A          LD A,(DE)
2CBB  FE FF       CP $FF
2CBD  C8          RET Z
2CBE  C5          PUSH BC
2CBF  E5          PUSH HL
2CC0  D5          PUSH DE
2CC1  DD E1       POP IX
2CC3  87          ADD A,A
2CC4  5F          LD E,A
2CC5  16 00       LD D,$00
2CC7  21 B2 24    LD HL,$24B2
2CCA  19          ADD HL,DE
2CCB  7E          LD A,(HL)
2CCC  23          INC HL
2CCD  66          LD H,(HL)
2CCE  6F          LD L,A
2CCF  11 D4 2C    LD DE,$2CD4
2CD2  D5          PUSH DE
2CD3  E9          JP (HL)
2CD4  .byte DD 5E 07 DD 56 08 DD 4E 09 DD 6E 01 DD 66 02 DD ; .^..V..N..n..f..
2CE4  .byte 7E 03 19 89 DD 75 01 DD 74 02 DD 77 03 DD 5E 0A ; ~....u..t..w..^.
2CF4  .byte DD 56 0B DD 4E 0C DD 6E 04 DD 66 05 DD 7E 06 19 ; .V..N..n..f..~..
2D04  .byte 89 DD 75 04 DD 74 05 DD 77 06 DD CB 18 6E C2 DE ; ..u..t..w....n..
2D14  .byte 2E 06 00 50 DD 5E 0E CB 3B DD CB 08 7E 20 09 DD ; ...P.^..;...~ ..
2D24  .byte 4E 0D 21 0A 3B C3 31 2D 0E 00 21 0A 3A ED 43 11 ; N.!.;.1-..!.:.C.
2D34  .byte D2 DD CB 18 B6 D5 E5 CD D5 30 5E 16 00 3A D5 D2 ; .........0^..:..
2D44  .byte 87 4F 42 21 3D 34 09 7E 23 66 6F 19 7E E6 3F 32 ; .OB!=4.~#fo.~.?2
2D54  .byte 15 D2 E1 D1 E6 3F CA EB 2D 3A 15 D2 87 4F 06 00 ; .....?..-:...O..
2D64  .byte 50 09 7E 23 66 6F DD 7E 05 83 E6 1F 5F 19 7E FE ; P.~#fo.~...._.~.
2D74  .byte 80 CA EB 2D 5F A7 F2 7F 2D 16 FF DD 6E 02 DD 66 ; ...-_...-...n..f
2D84  .byte 03 ED 4B 11 D2 09 DD CB 09 7E 20 0D A7 FA A7 2D ; ..K......~ ....-
2D94  .byte 7D E6 1F BB 30 0D C3 EB 2D A7 FA A7 2D 7D E6 1F ; }...0...-...-}..
2DA4  .byte BB 30 44 DD CB 18 F6 7D E6 E0 6F 19 A7 ED 42 DD ; .0D....}..o...B.
2DB4  .byte 75 02 DD 74 03 3A 15 D2 DD 77 19 5F 16 00 21 A8 ; u..t.:...w._..!.
2DC4  .byte 39 19 4E DD 72 07 DD 72 08 DD 72 09 7A 42 CB 79 ; 9.N.r..r..r.zB.y
2DD4  .byte 28 02 3D 05 DD 6E 0A DD 66 0B 09 DD 8E 0C DD 75 ; (.=..n..f......u
2DE4  .byte 0A DD 74 0B DD 77 0C 06 00 50 DD CB 0B 7E 20 0E ; ..t..w...P...~ .
2DF4  .byte DD 4E 0D CB 39 DD 5E 0E 21 7A 3E C3 0C 2E DD 4E ; .N..9.^.!z>....N
2E04  .byte 0D CB 39 1E 00 21 DA 3B ED 53 11 D2 DD CB 18 BE ; ..9..!.;.S......
2E14  .byte C5 E5 CD D5 30 5E 16 00 3A D5 D2 87 4F 42 21 3D ; ....0^..:...OB!=
2E24  .byte 34 09 7E 23 66 6F 19 7E E6 3F 32 15 D2 E1 C1 E6 ; 4.~#fo.~.?2.....
2E34  .byte 3F CA DE 2E 3A 15 D2 87 5F 16 00 42 19 7E 23 66 ; ?...:..._..B.~#f
2E44  .byte 6F DD 7E 02 81 E6 1F 4F 09 7E FE 80 CA DE 2E 4F ; o.~....O.~.....O
2E54  .byte A7 F2 5A 2E 06 FF DD 6E 05 DD 66 06 ED 5B 11 D2 ; ..Z....n..f..[..
2E64  .byte 19 DD CB 0C 7E 20 1D A7 FA 9E 2E 7D E6 1F D9 2A ; ....~ .....}...*
2E74  .byte 15 D2 26 00 11 DA 39 19 86 D9 B9 38 5D DD CB 18 ; ..&...9....8]...
2E84  .byte FE C3 9E 2E A7 FA 9E 2E 7D E6 1F D9 2A 15 D2 26 ; ........}...*..&
2E94  .byte 00 11 DA 39 19 86 D9 B9 30 40 7D E6 E0 6F 09 A7 ; ...9....0@}..o..
2EA4  .byte ED 52 DD 75 05 DD 74 06 3A 15 D2 DD 77 19 5F 16 ; .R.u..t.:...w._.
2EB4  .byte 00 21 78 39 19 4E DD 72 0A DD 72 0B DD 72 0C 7A ; .!x9.N.r..r..r.z
2EC4  .byte 42 CB 79 28 02 3D 05 DD 6E 07 DD 66 08 09 DD 8E ; B.y(.=..n..f....
2ED4  .byte 09 DD 75 07 DD 74 08 DD 77 09 DD 6E 05 DD 66 06 ; ..u..t..w..n..f.
2EE4  .byte ED 4B 57 D2 A7 ED 42 EB DD 6E 02 DD 66 03 ED 4B ; .KW...B..n..f..K
2EF4  .byte 54 D2 A7 ED 42 DD 4E 0F DD 46 10 79 B0 C4 07 2F ; T...B.N..F.y.../
2F04  .byte E1 C1 C9                                        ; ...

; ==== nt_writer  $2F07  (7 callers) — write a run of name-table entries into the $D000 build buffer from a layout source (BC): tile bytes with $FE = skip/transparent and $FF = end-of-run markers; bounds-checked against the buffer. ====
2F07  7C          LD A,H
2F08  A7          AND A
2F09  C0          RET NZ
2F0A  7D          LD A,L
2F0B  FE D0       CP $D0
2F0D  D0          RET NC
2F0E  7A          LD A,D
2F0F  FE FF       CP $FF
2F11  20 06       JR NZ,$2F19
2F13  7B          LD A,E
2F14  FE E8       CP $E8
2F16  D8          RET C
2F17  18 06       JR $2F1F
2F19  A7          AND A
2F1A  C0          RET NZ
2F1B  7B          LD A,E
2F1C  FE A8       CP $A8
2F1E  D0          RET NC
2F1F  63          LD H,E
2F20  D9          EXX
2F21  2A 36 D2    LD HL,($D236)
2F24  D9          EXX
2F25  16 03       LD D,$03
2F27  0A          LD A,(BC)
2F28  3C          INC A
2F29  28 2C       JR Z,$2F57
2F2B  E5          PUSH HL
2F2C  1E 06       LD E,$06
2F2E  0A          LD A,(BC)
2F2F  FE FE       CP $FE
2F31  30 12       JR NC,$2F45
2F33  7D          LD A,L
2F34  D9          EXX
2F35  77          LD (HL),A
2F36  2C          INC L
2F37  D9          EXX
2F38  7C          LD A,H
2F39  D9          EXX
2F3A  77          LD (HL),A
2F3B  2C          INC L
2F3C  D9          EXX
2F3D  0A          LD A,(BC)
2F3E  D9          EXX
2F3F  77          LD (HL),A
2F40  2C          INC L
2F41  FD 34 0A    INC (IY+10)
2F44  D9          EXX
2F45  3E 08       LD A,$08
2F47  85          ADD A,L
2F48  6F          LD L,A
2F49  03          INC BC
2F4A  1D          DEC E
2F4B  C2 2E 2F    JP NZ,$2F2E
2F4E  E1          POP HL
2F4F  3E 10       LD A,$10
2F51  84          ADD A,H
2F52  67          LD H,A
2F53  15          DEC D
2F54  C2 27 2F    JP NZ,$2F27
2F57  D9          EXX
2F58  22 36 D2    LD ($D236),HL
2F5B  D9          EXX
2F5C  C9          RET

; ==== sub_2F5D (1 caller) ====
2F5D  2A 11 D2    LD HL,($D211)
2F60  ED 4B 15 D2 LD BC,($D215)
2F64  09          ADD HL,BC
2F65  ED 4B 57 D2 LD BC,($D257)
2F69  A7          AND A
2F6A  ED 42       SBC HL,BC
2F6C  EB          EX DE,HL
2F6D  2A 0F D2    LD HL,($D20F)
2F70  ED 4B 13 D2 LD BC,($D213)
2F74  09          ADD HL,BC
2F75  ED 4B 54 D2 LD BC,($D254)
2F79  A7          AND A
2F7A  ED 42       SBC HL,BC
2F7C  4F          LD C,A
2F7D  7C          LD A,H
2F7E  A7          AND A
2F7F  C0          RET NZ
2F80  7A          LD A,D
2F81  FE FF       CP $FF
2F83  20 07       JR NZ,$2F8C
2F85  7B          LD A,E
2F86  FE F0       CP $F0
2F88  D8          RET C
2F89  C3 92 2F    JP $2F92
2F8C  A7          AND A
2F8D  C0          RET NZ
2F8E  7B          LD A,E
2F8F  FE C0       CP $C0
2F91  D0          RET NC
2F92  61          LD H,C
2F93  ED 4B 36 D2 LD BC,($D236)
2F97  7D          LD A,L
2F98  02          LD (BC),A
2F99  0C          INC C
2F9A  7B          LD A,E
2F9B  02          LD (BC),A
2F9C  0C          INC C
2F9D  7C          LD A,H
2F9E  02          LD (BC),A
2F9F  0C          INC C
2FA0  ED 43 36 D2 LD ($D236),BC
2FA4  FD 34 0A    INC (IY+10)
2FA7  C9          RET

; ==== sub_2FA8 (21 callers) ====
2FA8  1A          LD A,(DE)
2FA9  FE FF       CP $FF
2FAB  C8          RET Z
2FAC  FE FE       CP $FE
2FAE  28 09       JR Z,$2FB9
2FB0  71          LD (HL),C
2FB1  2C          INC L
2FB2  70          LD (HL),B
2FB3  2C          INC L
2FB4  77          LD (HL),A
2FB5  2C          INC L
2FB6  FD 34 0A    INC (IY+10)
2FB9  13          INC DE
2FBA  79          LD A,C
2FBB  C6 08       ADD A,$08
2FBD  4F          LD C,A
2FBE  C3 A8 2F    JP $2FA8

; ==== sub_2FC1 (1 caller) ====
2FC1  FD CB 05 46 BIT 0,(IY+5)
2FC5  C0          RET NZ
2FC6  FD CB 08 46 BIT 0,(IY+8)
2FCA  C2 9A 30    JP NZ,$309A
2FCD  3A 15 D4    LD A,($D415)
2FD0  0F          RRCA
2FD1  DA 9A 30    JP C,$309A
2FD4  E6 02       AND $02
2FD6  C2 9A 30    JP NZ,$309A

; ==== sub_2FD9 (1 caller) ====
2FD9  FD CB 09 46 BIT 0,(IY+9)
2FDD  C0          RET NZ
2FDE  FD CB 06 76 BIT 6,(IY+6)
2FE2  C0          RET NZ
2FE3  FD CB 08 46 BIT 0,(IY+8)
2FE7  C0          RET NZ
2FE8  FD CB 06 6E BIT 5,(IY+6)
2FEC  20 6C       JR NZ,$305A
2FEE  3A A9 D2    LD A,($D2A9)
2FF1  A7          AND A
2FF2  20 2C       JR NZ,$3020

; ==== sub_2FF4 (1 caller) ====
2FF4  FD CB 05 C6 SET 0,(IY+5)
2FF8  21 15 D4    LD HL,$D415
2FFB  CB FE       SET 7,(HL)
2FFD  21 FC FF    LD HL,$FFFC
3000  AF          XOR A
3001  32 07 D4    LD ($D407),A
3004  22 08 D4    LD ($D408),HL
3007  3E 60       LD A,$60
3009  32 81 D2    LD ($D281),A
300C  FD CB 06 B6 RES 6,(IY+6)
3010  FD CB 06 AE RES 5,(IY+6)
3014  FD CB 06 B6 RES 6,(IY+6)
3018  FD CB 08 86 RES 0,(IY+8)
301C  3E 0A       LD A,$0A
301E  DF          RST $18
301F  C9          RET
3020  AF          XOR A
3021  32 A9 D2    LD ($D2A9),A
3024  CD AF 7C    CALL $7CAF
3027  38 31       JR C,$305A
3029  DD E5       PUSH IX
302B  E5          PUSH HL
302C  DD E1       POP IX
302E  DD 36 00 55 LD (IX+0),$55
3032  DD 36 11 06 LD (IX+17),$06
3036  DD 36 12 00 LD (IX+18),$00
303A  2A FF D3    LD HL,($D3FF)
303D  DD 75 02    LD (IX+2),L
3040  DD 74 03    LD (IX+3),H
3043  2A 02 D4    LD HL,($D402)
3046  DD 75 05    LD (IX+5),L
3049  DD 74 06    LD (IX+6),H
304C  DD 36 0A 00 LD (IX+10),$00
3050  DD 36 0B FC LD (IX+11),$FC
3054  DD 36 0C FF LD (IX+12),$FF
3058  DD E1       POP IX
305A  21 15 D4    LD HL,$D415
305D  11 FC FF    LD DE,$FFFC
3060  AF          XOR A
3061  CB 66       BIT 4,(HL)
3063  28 03       JR Z,$3068
3065  11 FE FF    LD DE,$FFFE
3068  32 07 D4    LD ($D407),A
306B  ED 53 08 D4 LD ($D408),DE
306F  CB 4E       BIT 1,(HL)
3071  28 0A       JR Z,$307D
3073  7E          LD A,(HL)
3074  F6 12       OR $12
3076  77          LD (HL),A
3077  AF          XOR A
3078  11 02 00    LD DE,$0002
307B  18 06       JR $3083
307D  CB 8E       RES 1,(HL)
307F  AF          XOR A
3080  11 FE FF    LD DE,$FFFE
3083  32 04 D4    LD ($D404),A
3086  ED 53 05 D4 LD ($D405),DE
308A  FD CB 06 AE RES 5,(IY+6)
308E  FD CB 06 F6 SET 6,(IY+6)
3092  FD 36 03 FF LD (IY+3),$FF
3096  3E 11       LD A,$11
3098  EF          RST $28
3099  C9          RET
309A  DD 36 00 0A LD (IX+0),$0A
309E  3A 0F D2    LD A,($D20F)
30A1  5F          LD E,A
30A2  16 00       LD D,$00
30A4  DD 6E 02    LD L,(IX+2)
30A7  DD 66 03    LD H,(IX+3)
30AA  19          ADD HL,DE
30AB  DD 75 02    LD (IX+2),L
30AE  DD 74 03    LD (IX+3),H
30B1  3A 10 D2    LD A,($D210)
30B4  5F          LD E,A
30B5  DD 6E 05    LD L,(IX+5)
30B8  DD 66 06    LD H,(IX+6)
30BB  19          ADD HL,DE
30BC  DD 75 05    LD (IX+5),L
30BF  DD 74 06    LD (IX+6),H
30C2  AF          XOR A
30C3  DD 77 0F    LD (IX+15),A
30C6  DD 77 10    LD (IX+16),A
30C9  3E 01       LD A,$01
30CB  EF          RST $28
30CC  11 00 01    LD DE,$0100
30CF  0E 00       LD C,$00
30D1  CD AA 33    CALL $33AA
30D4  C9          RET
30D5  .byte 3A 32 D2 FE 80 28 0F FE 40 28 37 FE 20 28 5C FE ; :2...(..@(7. (\.
30E5  .byte 10 28 7E C3 8F 31 DD 6E 05 DD 66 06 19 7D 87 CB ; .(~..1.n..f..}..
30F5  .byte 14 87 CB 14 E6 80 6F EB DD 6E 02 DD 66 03 09 7D ; ......o..n..f..}
3105  .byte 87 CB 14 87 CB 14 87 CB 14 6C 26 00 19 11 00 C0 ; .........l&.....
3115  .byte 19 C9 DD 6E 05 DD 66 06 19 7D 87 CB 14 E6 C0 6F ; ...n..f..}.....o
3125  .byte EB DD 6E 02 DD 66 03 09 7D 87 CB 14 87 CB 14 87 ; ..n..f..}.......
3135  .byte CB 14 6C 26 00 19 11 00 C0 19 C9 DD 6E 05 DD 66 ; ..l&........n..f
3145  .byte 06 19 7D E6 E0 6F EB DD 6E 02 DD 66 03 09 7D 87 ; ..}..o..n..f..}.
3155  .byte CB 14 87 CB 14 87 CB 14 6C 26 00 19 11 00 C0 19 ; ........l&......
3165  .byte C9 DD 6E 05 DD 66 06 19 7D CB 3C 1F E6 F0 6F EB ; ..n..f..}.<...o.
3175  .byte DD 6E 02 DD 66 03 09 7D 87 CB 14 87 CB 14 87 CB ; .n..f..}........
3185  .byte 14 6C 26 00 19 11 00 C0 19 C9 DD 6E 05 DD 66 06 ; .l&........n..f.
3195  .byte 19 7D 07 CB 14 07 CB 14 07 CB 14 EB DD 6E 02 DD ; .}...........n..
31A5  .byte 66 03 09 7D 07 CB 14 07 CB 14 07 CB 14 6C 26 00 ; f..}.........l&.
31B5  .byte 5C 19 11 00 C0 19 C9                            ; \......

; ==== tile_stream  $31BC  (1 caller) — GAMEPLAY tile streamer: copy level tiles (bank 8) to VRAM, expanding 3 stored bitplanes to 4bpp (OUTI x3 + a zero plane per row) = level tiles stored 3bpp/8-colour. $D289/$D28B = source cursor/end, $D28D = count. ====
31BC  ED 5B 89 D2 LD DE,($D289)
31C0  2A 8B D2    LD HL,($D28B)
31C3  A7          AND A
31C4  ED 52       SBC HL,DE
31C6  C8          RET Z
31C7  21 80 36    LD HL,$3680
31CA  EB          EX DE,HL
31CB  FD CB 06 46 BIT 0,(IY+6)
31CF  C2 0C 32    JP NZ,$320C
31D2  7B          LD A,E
31D3  D3 BF       OUT ($BF),A
31D5  7A          LD A,D
31D6  F6 40       OR $40
31D8  D3 BF       OUT ($BF),A
31DA  3A 8D D2    LD A,($D28D)
31DD  5F          LD E,A
31DE  AF          XOR A
31DF  0E BE       LD C,$BE
31E1  ED A3       OUTI
31E3  ED A3       OUTI
31E5  ED A3       OUTI
31E7  D3 BE       OUT ($BE),A
31E9  ED A3       OUTI
31EB  ED A3       OUTI
31ED  ED A3       OUTI
31EF  D3 BE       OUT ($BE),A
31F1  ED A3       OUTI
31F3  ED A3       OUTI
31F5  ED A3       OUTI
31F7  D3 BE       OUT ($BE),A
31F9  ED A3       OUTI
31FB  ED A3       OUTI
31FD  ED A3       OUTI
31FF  D3 BE       OUT ($BE),A
3201  1D          DEC E
3202  C2 E1 31    JP NZ,$31E1
3205  2A 89 D2    LD HL,($D289)
3208  22 8B D2    LD ($D28B),HL
320B  C9          RET
320C  01 1D 01    LD BC,$011D
320F  09          ADD HL,BC
3210  7B          LD A,E
3211  D3 BF       OUT ($BF),A
3213  7A          LD A,D
3214  F6 40       OR $40
3216  D3 BF       OUT ($BF),A
3218  D9          EXX
3219  C5          PUSH BC
321A  06 10       LD B,$10
321C  D9          EXX
321D  11 FA FF    LD DE,$FFFA
3220  0E BE       LD C,$BE
3222  AF          XOR A
3223  ED A3       OUTI
3225  ED A3       OUTI
3227  ED A3       OUTI
3229  D3 BE       OUT ($BE),A
322B  19          ADD HL,DE
322C  ED A3       OUTI
322E  ED A3       OUTI
3230  ED A3       OUTI
3232  D3 BE       OUT ($BE),A
3234  19          ADD HL,DE
3235  ED A3       OUTI
3237  ED A3       OUTI
3239  ED A3       OUTI
323B  D3 BE       OUT ($BE),A
323D  19          ADD HL,DE
323E  ED A3       OUTI
3240  ED A3       OUTI
3242  ED A3       OUTI
3244  D3 BE       OUT ($BE),A
3246  19          ADD HL,DE
3247  D9          EXX
3248  05          DEC B
3249  D9          EXX
324A  C2 23 32    JP NZ,$3223
324D  D9          EXX
324E  C1          POP BC
324F  D9          EXX
3250  2A 89 D2    LD HL,($D289)
3253  22 8B D2    LD ($D28B),HL
3256  C9          RET

; ==== sub_3257 (2 callers) ====
3257  F3          DI
3258  7B          LD A,E
3259  D3 BF       OUT ($BF),A
325B  7A          LD A,D
325C  F6 40       OR $40
325E  D3 BF       OUT ($BF),A
3260  06 20       LD B,$20
3262  7E          LD A,(HL)
3263  D3 BE       OUT ($BE),A
3265  00          NOP
3266  00          NOP
3267  00          NOP
3268  00          NOP
3269  23          INC HL
326A  7E          LD A,(HL)
326B  D3 BE       OUT ($BE),A
326D  00          NOP
326E  00          NOP
326F  00          NOP
3270  00          NOP
3271  23          INC HL
3272  7E          LD A,(HL)
3273  D3 BE       OUT ($BE),A
3275  00          NOP
3276  00          NOP
3277  00          NOP
3278  00          NOP
3279  23          INC HL
327A  7E          LD A,(HL)
327B  D3 BE       OUT ($BE),A
327D  23          INC HL
327E  10 E2       DJNZ $3262
3280  FB          EI
3281  C9          RET

; ==== scroll_draw  $3282  (1 caller) — GAMEPLAY scroll update: when the camera moves >=8 px, draw the newly-revealed name-table column/row at $3800 from the decompressed level map in RAM (source $D2AF); scroll pos $D2AB/$D2AD vs previous $D254/$D257. ====
3282  2A AB D2    LD HL,($D2AB)
3285  7D          LD A,L
3286  E6 F8       AND $F8
3288  6F          LD L,A
3289  ED 5B 54 D2 LD DE,($D254)
328D  7B          LD A,E
328E  E6 F8       AND $F8
3290  5F          LD E,A
3291  AF          XOR A
3292  ED 52       SBC HL,DE
3294  D8          RET C
3295  B4          OR H
3296  C0          RET NZ
3297  7D          LD A,L
3298  FE 08       CP $08
329A  D8          RET C
329B  57          LD D,A
329C  3A 4B D2    LD A,($D24B)
329F  E6 F8       AND $F8
32A1  5F          LD E,A
32A2  19          ADD HL,DE
32A3  CB 3C       SRL H
32A5  CB 1D       RR L
32A7  CB 3C       SRL H
32A9  CB 1D       RR L
32AB  CB 3C       SRL H
32AD  CB 1D       RR L
32AF  7D          LD A,L
32B0  E6 1F       AND $1F
32B2  87          ADD A,A
32B3  4F          LD C,A
32B4  2A AD D2    LD HL,($D2AD)
32B7  7D          LD A,L
32B8  E6 F8       AND $F8
32BA  6F          LD L,A
32BB  ED 5B 57 D2 LD DE,($D257)
32BF  7B          LD A,E
32C0  E6 F8       AND $F8
32C2  5F          LD E,A
32C3  AF          XOR A
32C4  ED 52       SBC HL,DE
32C6  D8          RET C
32C7  B4          OR H
32C8  C0          RET NZ
32C9  7D          LD A,L
32CA  FE C0       CP $C0
32CC  D0          RET NC
32CD  16 00       LD D,$00
32CF  3A 4C D2    LD A,($D24C)
32D2  E6 F8       AND $F8
32D4  5F          LD E,A
32D5  19          ADD HL,DE
32D6  CB 3C       SRL H
32D8  CB 1D       RR L
32DA  CB 3C       SRL H
32DC  CB 1D       RR L
32DE  CB 3C       SRL H
32E0  CB 1D       RR L
32E2  7D          LD A,L
32E3  FE 1C       CP $1C
32E5  38 02       JR C,$32E9
32E7  D6 1C       SUB $1C
32E9  6F          LD L,A
32EA  26 00       LD H,$00
32EC  44          LD B,H
32ED  0F          RRCA
32EE  0F          RRCA
32EF  67          LD H,A
32F0  E6 C0       AND $C0
32F2  6F          LD L,A
32F3  7C          LD A,H
32F4  AD          XOR L
32F5  67          LD H,A
32F6  09          ADD HL,BC
32F7  01 00 38    LD BC,$3800
32FA  09          ADD HL,BC
32FB  ED 5B AF D2 LD DE,($D2AF)
32FF  06 02       LD B,$02
3301  7D          LD A,L
3302  D3 BF       OUT ($BF),A
3304  7C          LD A,H
3305  F6 40       OR $40
3307  D3 BF       OUT ($BF),A
3309  1A          LD A,(DE)
330A  D3 BE       OUT ($BE),A
330C  13          INC DE
330D  00          NOP
330E  00          NOP
330F  1A          LD A,(DE)
3310  D3 BE       OUT ($BE),A
3312  13          INC DE
3313  00          NOP
3314  00          NOP
3315  1A          LD A,(DE)
3316  D3 BE       OUT ($BE),A
3318  13          INC DE
3319  00          NOP
331A  00          NOP
331B  1A          LD A,(DE)
331C  D3 BE       OUT ($BE),A
331E  13          INC DE
331F  78          LD A,B
3320  01 40 00    LD BC,$0040
3323  09          ADD HL,BC
3324  47          LD B,A
3325  10 DA       DJNZ $3301
3327  C9          RET

; ==== sub_3328 (4 callers) ====
3328  FD CB 05 46 BIT 0,(IY+5)
332C  37          SCF
332D  C0          RET NZ
332E  DD 6E 02    LD L,(IX+2)
3331  DD 66 03    LD H,(IX+3)
3334  DD 4E 0D    LD C,(IX+13)
3337  06 00       LD B,$00
3339  09          ADD HL,BC
333A  ED 5B FF D3 LD DE,($D3FF)
333E  AF          XOR A
333F  ED 52       SBC HL,DE
3341  D8          RET C
3342  DD 6E 02    LD L,(IX+2)
3345  DD 66 03    LD H,(IX+3)
3348  3A 15 D2    LD A,($D215)
334B  4F          LD C,A
334C  09          ADD HL,BC
334D  EB          EX DE,HL
334E  3A 0A D4    LD A,($D40A)
3351  4F          LD C,A
3352  09          ADD HL,BC
3353  AF          XOR A
3354  ED 52       SBC HL,DE
3356  D8          RET C
3357  DD 6E 05    LD L,(IX+5)
335A  DD 66 06    LD H,(IX+6)
335D  DD 4E 0E    LD C,(IX+14)
3360  09          ADD HL,BC
3361  ED 5B 02 D4 LD DE,($D402)
3365  AF          XOR A
3366  ED 52       SBC HL,DE
3368  D8          RET C
3369  DD 6E 05    LD L,(IX+5)
336C  DD 66 06    LD H,(IX+6)
336F  3A 16 D2    LD A,($D216)
3372  4F          LD C,A
3373  09          ADD HL,BC
3374  EB          EX DE,HL
3375  3A 0B D4    LD A,($D40B)
3378  4F          LD C,A
3379  09          ADD HL,BC
337A  AF          XOR A
337B  ED 52       SBC HL,DE
337D  C9          RET
337E  .byte 4F 3A A9 D2 81 4F E6 0F FE 0A 38 04 79 C6 06 4F ; O:...O....8.y..O
338E  .byte 79 FE A0 38 10 D6 A0 32 A9 D2 3A 40 D2 3C 32 40 ; y..8...2..:@.<2@
339E  .byte D2 3E 09 EF C9 32 A9 D2 3E 02 EF C9             ; .>...2..>...

; ==== sub_33AA (6 callers) ====
33AA  21 BE D2    LD HL,$D2BE
33AD  7B          LD A,E
33AE  86          ADD A,(HL)
33AF  27          DAA
33B0  77          LD (HL),A
33B1  2B          DEC HL
33B2  7A          LD A,D
33B3  8E          ADC A,(HL)
33B4  27          DAA
33B5  77          LD (HL),A
33B6  2B          DEC HL
33B7  79          LD A,C
33B8  8E          ADC A,(HL)
33B9  27          DAA
33BA  77          LD (HL),A
33BB  4F          LD C,A
33BC  2B          DEC HL
33BD  3E 00       LD A,$00
33BF  8E          ADC A,(HL)
33C0  27          DAA
33C1  77          LD (HL),A
33C2  21 F8 D2    LD HL,$D2F8
33C5  79          LD A,C
33C6  BE          CP (HL)
33C7  D8          RET C
33C8  7E          LD A,(HL)
33C9  A7          AND A
33CA  C8          RET Z
33CB  C6 05       ADD A,$05
33CD  27          DAA
33CE  77          LD (HL),A
33CF  21 40 D2    LD HL,$D240
33D2  34          INC (HL)
33D3  3E 09       LD A,$09
33D5  EF          RST $28
33D6  C9          RET

; ==== sub_33D7 (1 caller) ====
33D7  FD CB 05 46 BIT 0,(IY+5)
33DB  C0          RET NZ
33DC  21 D1 D2    LD HL,$D2D1
33DF  FD CB 07 46 BIT 0,(IY+7)
33E3  20 26       JR NZ,$340B
33E5  7E          LD A,(HL)
33E6  3C          INC A
33E7  FE 3C       CP $3C
33E9  38 01       JR C,$33EC
33EB  AF          XOR A
33EC  77          LD (HL),A
33ED  2B          DEC HL
33EE  3F          CCF
33EF  7E          LD A,(HL)
33F0  CE 00       ADC A,$00
33F2  27          DAA
33F3  FE 60       CP $60
33F5  38 01       JR C,$33F8
33F7  AF          XOR A
33F8  77          LD (HL),A
33F9  2B          DEC HL
33FA  3F          CCF
33FB  7E          LD A,(HL)
33FC  CE 00       ADC A,$00
33FE  27          DAA
33FF  FE 10       CP $10
3401  38 06       JR C,$3409
3403  E5          PUSH HL
3404  CD F4 2F    CALL $2FF4
3407  E1          POP HL
3408  AF          XOR A
3409  77          LD (HL),A
340A  C9          RET
340B  7E          LD A,(HL)
340C  3C          INC A
340D  FE 3C       CP $3C
340F  38 01       JR C,$3412
3411  AF          XOR A
3412  77          LD (HL),A
3413  2B          DEC HL
3414  3F          CCF
3415  7E          LD A,(HL)
3416  DE 00       SBC A,$00
3418  27          DAA
3419  FE 60       CP $60
341B  38 02       JR C,$341F
341D  3E 59       LD A,$59
341F  77          LD (HL),A
3420  2B          DEC HL
3421  3F          CCF
3422  7E          LD A,(HL)
3423  DE 00       SBC A,$00
3425  27          DAA
3426  FE 60       CP $60
3428  77          LD (HL),A
3429  D8          RET C
342A  AF          XOR A
342B  77          LD (HL),A
342C  23          INC HL
342D  77          LD (HL),A
342E  23          INC HL
342F  77          LD (HL),A
3430  3E 01       LD A,$01
3432  32 83 D2    LD ($D283),A
3435  FD CB 09 D6 SET 2,(IY+9)
3439  C9          RET
343A  .byte 01 30 00 4D 34 0D 35 A5 35 55 36 FD 36 B8 37 90 ; .0.M4.5.5U6.6.7.
344A  .byte 38 10 39 00 16 10 10 10 00 00 08 09 0A 05 06 07 ; 8.9.............
345A  .byte 03 04 01 02 10 00 00 00 10 10 00 00 00 10 00 00 ; ................
346A  .byte 00 00 00 00 00 00 00 00 10 00 00 00 00 00 00 00 ; ................
347A  .byte 10 10 0C 0D 0E 0F 0B 10 10 10 16 00 10 10 10 00 ; ................
348A  .byte 10 10 10 10 10 10 10 10 16 16 12 10 15 00 00 27 ; ...............'
349A  .byte 16 1E 16 11 10 00 10 10 1E 1E 1E 10 1E 00 00 16 ; ................
34AA  .byte 1E 16 1E 00 27 1E 00 27 27 27 27 27 16 27 27 00 ; ....'..'''''.''.
34BA  .byte 00 00 00 00 00 00 14 00 00 05 0A 00 00 00 00 00 ; ................
34CA  .byte 00 00 00 00 10 00 00 00 00 00 10 00 10 00 00 00 ; ................
34DA  .byte 00 00 00 80 80 90 80 96 90 80 90 80 80 80 A7 A7 ; ................
34EA  .byte A7 A7 A7 A7 A7 A7 A7 A7 00 00 00 00 90 9E 27 27 ; ..............''
34FA  .byte 27 00 90 80 80 80 80 80 90 10 00 10 00 00 00 00 ; '...............
350A  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
351A  .byte 00 00 00 13 10 12 12 13 00 00 00 00 00 00 10 10 ; ................
352A  .byte 00 00 00 12 13 10 13 12 00 00 00 07 2B 00 00 08 ; ............+...
353A  .byte 00 09 06 05 29 10 2A 0A 00 00 00 10 10 2E 00 2D ; ....).*........-
354A  .byte 00 00 00 00 00 80 80 80 00 80 80 80 80 00 00 80 ; ................
355A  .byte 00 00 80 2C 2F 10 00 00 00 80 80 10 16 00 00 00 ; ...,/...........
356A  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
357A  .byte 00 00 00 00 00 12 10 13 00 00 10 00 00 00 00 00 ; ................
358A  .byte 00 00 00 13 16 16 12 13 12 01 02 10 2D 2E 00 00 ; ............-...
359A  .byte 00 00 11 00 00 00 03 04 10 00 00 00 10 00 00 00 ; ................
35AA  .byte 00 00 00 10 10 00 00 00 00 00 00 00 00 00 10 10 ; ................
35BA  .byte 10 10 10 10 10 16 16 16 16 27 16 1E 10 10 00 00 ; .........'......
35CA  .byte 00 00 00 00 10 00 00 10 00 00 00 00 00 00 00 00 ; ................
35DA  .byte 00 00 00 00 00 00 00 27 00 00 10 11 00 01 00 00 ; .......'........
35EA  .byte 10 10 00 04 01 02 03 06 07 05 08 09 0A 10 0E 0F ; ................
35FA  .byte 05 0A 04 01 10 10 17 00 0B 05 14 0A 00 10 27 10 ; ..............'.
360A  .byte 00 00 00 10 1E 00 10 10 00 00 10 10 10 00 00 00 ; ................
361A  .byte 1E 00 27 00 00 00 00 00 00 00 00 00 80 80 80 80 ; ..'.............
362A  .byte 80 A7 80 27 A7 A7 A7 A7 A7 A7 A7 A7 A7 80 80 10 ; ...'............
363A  .byte 10 96 96 16 16 16 16 10 27 00 00 00 00 00 00 1E ; ........'.......
364A  .byte 00 00 00 00 00 00 00 00 00 00 00 00 16 16 16 16 ; ................
365A  .byte 16 16 16 16 16 16 16 16 16 16 16 16 16 16 16 16 ; ................
366A  .byte 16 16 16 00 00 00 00 00 00 80 27 00 00 00 00 00 ; ..........'.....
367A  .byte 00 80 27 00 00 00 00 00 27 A7 16 00 00 1E 27 00 ; ..'.....'.....'.
368A  .byte 1E 00 27 00 27 00 16 27 27 9E 80 1E 1E 1E 16 16 ; ..'.'..''.......
369A  .byte 16 16 16 27 1E 1E 16 16 16 16 16 06 07 00 00 08 ; ...'............
36AA  .byte 09 02 01 12 05 14 15 0A 13 04 03 04 00 04 03 08 ; ................
36BA  .byte 09 06 07 03 01 02 01 0A 06 09 05 00 00 04 00 00 ; ................
36CA  .byte 00 00 00 00 00 00 00 00 00 00 00 27 16 16 27 16 ; ...........'..'.
36DA  .byte 16 16 16 16 00 27 16 16 16 16 00 1E 00 27 1E 00 ; .....'.......'..
36EA  .byte 1E 00 00 01 04 01 04 09 06 00 00 00 00 27 00 00 ; .............'..
36FA  .byte 00 00 00 00 16 16 16 16 16 16 16 16 16 16 16 1E ; ................
370A  .byte 1E 1E 1A 1B 1C 1D 1F 20 21 22 23 24 1B 1C 16 00 ; ....... !"#$....
371A  .byte 16 16 00 16 16 16 16 16 16 16 16 16 16 16 16 16 ; ................
372A  .byte 16 16 27 27 27 04 03 02 01 08 09 0A 05 06 07 0A ; ..'''...........
373A  .byte 05 03 02 15 14 16 16 13 12 10 10 10 10 10 10 10 ; ................
374A  .byte 10 16 27 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ..'.............
375A  .byte 00 00 00 00 00 00 1E 00 1E 1E 1E 00 00 10 80 80 ; ................
376A  .byte 27 27 27 16 16 27 27 27 1E 1E 16 00 00 00 00 00 ; '''..'''........
377A  .byte 00 00 00 00 02 03 90 80 9E 16 16 02 03 1B 1C 16 ; ................
378A  .byte 16 19 18 25 26 00 00 00 27 27 1E 1E 27 1E 00 00 ; ...%&...''..'...
379A  .byte 00 00 1E 27 1E 27 9E 9E 16 16 00 00 1E 16 1E 1E ; ...'.'..........
37AA  .byte 90 90 90 16 16 16 16 00 00 00 00 A7 9E 00 00 10 ; ................
37BA  .byte 16 16 10 10 10 10 10 00 00 16 16 1E 00 00 00 00 ; ................
37CA  .byte 10 10 10 00 90 80 1E 00 00 00 10 10 00 00 00 00 ; ................
37DA  .byte 00 00 00 00 00 03 04 00 00 08 09 0A 16 13 15 02 ; ................
37EA  .byte 01 00 07 06 05 16 14 12 0A 05 10 10 00 00 03 02 ; ................
37FA  .byte 10 00 00 10 00 00 00 00 00 00 00 00 10 10 10 00 ; ................
380A  .byte 00 10 00 10 00 00 00 10 10 10 10 16 16 04 03 03 ; ................
381A  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 10 ; ................
382A  .byte 10 16 00 10 00 00 00 00 00 00 00 00 00 00 16 00 ; ................
383A  .byte 00 00 00 00 00 00 00 10 00 00 00 00 00 00 00 1E ; ................
384A  .byte 00 00 00 1E 1E 10 00 00 10 10 1E 1E 16 16 1E 1E ; ................
385A  .byte 1E 1E 1E 00 10 1E 1E 10 10 1E 00 02 0A 16 00 00 ; ................
386A  .byte 00 00 00 00 10 1E 16 1E 00 10 10 10 10 10 1E 00 ; ................
387A  .byte 10 00 00 10 10 10 10 1E 90 00 00 00 00 00 00 00 ; ................
388A  .byte 00 00 00 9E 1E 10 00 27 27 27 00 00 00 00 00 00 ; .......'''......
389A  .byte 00 00 00 00 00 00 00 00 00 00 00 1E 00 00 00 00 ; ................
38AA  .byte 00 00 00 00 00 00 00 00 00 00 27 00 00 00 00 00 ; ..........'.....
38BA  .byte 27 27 16 00 00 00 00 00 00 1E 27 00 00 00 00 00 ; ''........'.....
38CA  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
38DA  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
38EA  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
38FA  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
390A  .byte 00 00 00 00 00 00 00 27 27 16 1E 1E 16 27 27 1E ; .......''....''.
391A  .byte 1E 00 00 16 27 27 16 1E 1E 16 16 16 16 01 02 04 ; ....''..........
392A  .byte 03 1D 1C 1A 1B 01 02 04 03 1D 1C 1A 1B 00 00 00 ; ................
393A  .byte 00 00 00 00 1E 9E 9E 80 1E 27 A7 A7 80 80 16 16 ; .........'......
394A  .byte 80 1E 1E 27 27 27 16 1E 16 16 16 16 16 16 27 00 ; ...'''........'.
395A  .byte 1E 00 00 00 00 00 00 00 16 16 16 16 16 16 16 16 ; ................
396A  .byte A7 A7 9E 9E 16 00 9E A7 80 9E A7 80 1D 27 00 1C ; .............'..
397A  .byte 1C E4 E4 12 12 12 EE EE EE 00 00 00 00 00 00 00 ; ................
398A  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
399A  .byte 00 00 00 00 00 00 00 12 EE 00 00 00 00 00 00 00 ; ................
39AA  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
39BA  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
39CA  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
39DA  .byte 00 08 08 08 08 06 06 06 06 06 06 03 03 03 03 03 ; ................
39EA  .byte 03 08 03 03 03 03 03 03 00 00 00 00 00 00 00 00 ; ................
39FA  .byte 00 00 00 00 00 00 00 03 03 04 04 03 03 03 03 00 ; ................
3A0A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; j:j:j:j:j:j:j:j:
3A1A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; j:j:j:j:j:j:j:j:
3A2A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 8A 3A 6A 3A ; j:j:j:j:j:j:.:j:
3A3A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A AA 3A 6A 3A ; j:j:j:j:j:j:.:j:
3A4A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A CA 3A ; j:j:j:j:j:j:j:.:
3A5A  .byte EA 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; .:j:j:j:j:j:j:j:
3A6A  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3A7A  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3A8A  .byte 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C ; ................
3A9A  .byte 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C ; ................
3AAA  .byte 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C ; ................
3ABA  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3ACA  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3ADA  .byte 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C 1C ; ................
3AEA  .byte 80 80 80 80 80 80 80 80 1C 1C 1C 1C 1C 1C 1C 1C ; ................
3AFA  .byte 1C 1C 1C 1C 1C 1C 1C 1C 80 80 80 80 80 80 80 80 ; ................
3B0A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; j:j:j:j:j:j:j:j:
3B1A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; j:j:j:j:j:j:j:j:
3B2A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3B 6A 3A ; j:j:j:j:j:j:j;j:
3B3A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 7A 3B 6A 3A ; j:j:j:j:j:j:z;j:
3B4A  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 9A 3B ; j:j:j:j:j:j:j:.;
3B5A  .byte BA 3B 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; .;j:j:j:j:j:j:j:
3B6A  .byte 04 04 04 04 04 04 04 04 04 04 04 04 04 04 04 04 ; ................
3B7A  .byte 04 04 04 04 04 04 04 04 04 04 04 04 04 04 04 04 ; ................
3B8A  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3B9A  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3BAA  .byte 04 04 04 04 04 04 04 04 04 04 04 04 04 04 04 04 ; ................
3BBA  .byte 80 80 80 80 80 80 80 80 04 04 04 04 04 04 04 04 ; ................
3BCA  .byte 04 04 04 04 04 04 04 04 80 80 80 80 80 80 80 80 ; ................
3BDA  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; j:j:j:j:j:j:j:j:
3BEA  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; j:j:j:j:j:j:j:j:
3BFA  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 3A 3C 6A 3A ; j:j:j:j:j:j::<j:
3C0A  .byte 5A 3C 7A 3C 9A 3C BA 3C DA 3C FA 3C 1A 3D 3A 3D ; Z<z<.<.<.<.<.=:=
3C1A  .byte 5A 3D 7A 3D 9A 3D BA 3D DA 3D FA 3D 1A 3E 3A 3E ; Z=z=.=.=.=.=.>:>
3C2A  .byte 5A 3E 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; Z>j:j:j:j:j:j:j:
3C3A  .byte 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F ; ................
3C4A  .byte 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F ; ................
3C5A  .byte 18 18 17 17 16 16 15 15 14 14 13 13 12 12 11 11 ; ................
3C6A  .byte 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 ; ................
3C7A  .byte 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 ; ................
3C8A  .byte 11 11 12 12 13 13 14 14 15 15 16 16 17 17 18 18 ; ................
3C9A  .byte 0F 0E 0D 0C 0B 0A 09 08 07 06 05 04 03 02 01 00 ; ................
3CAA  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3CBA  .byte 2F 2E 2D 2C 2B 2A 29 28 27 26 25 24 23 22 21 20 ; /.-,+*)('&%$#"! 
3CCA  .byte 1F 1E 1D 1C 1B 1A 19 18 17 16 15 14 13 12 11 10 ; ................
3CDA  .byte 10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E 1F ; ................
3CEA  .byte 20 21 22 23 24 25 26 27 28 29 2A 2B 2C 2D 2E 2F ;  !"#$%&'()*+,-./
3CFA  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3D0A  .byte 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F ; ................
3D1A  .byte 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F ; ................
3D2A  .byte 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F ; ................
3D3A  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3D4A  .byte 00 00 01 01 02 02 03 03 04 04 05 05 06 06 07 07 ; ................
3D5A  .byte 08 08 09 09 0A 0A 0B 0B 0C 0C 0D 0D 0E 0E 0F 0F ; ................
3D6A  .byte 10 10 11 11 12 12 13 13 14 14 15 15 16 16 17 17 ; ................
3D7A  .byte 18 18 19 19 1A 1A 1B 1B 1C 1C 1D 1D 1E 1E 1F 1F ; ................
3D8A  .byte 20 20 21 21 22 22 23 23 24 24 25 25 26 26 27 27 ;   !!""##$$%%&&''
3D9A  .byte 27 27 26 26 25 25 24 24 23 23 22 22 21 21 20 20 ; ''&&%%$$##""!!  
3DAA  .byte 1F 1F 1E 1E 1D 1D 1C 1C 1B 1B 1A 1A 19 19 18 18 ; ................
3DBA  .byte 17 17 16 16 15 15 14 14 13 13 12 12 11 11 10 10 ; ................
3DCA  .byte 0F 0F 0E 0E 0D 0D 0C 0C 0B 0B 0A 0A 09 09 08 08 ; ................
3DDA  .byte 07 07 06 06 05 05 04 04 03 03 02 02 01 01 00 00 ; ................
3DEA  .byte 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 80 ; ................
3DFA  .byte 08 08 09 09 0A 0A 0B 0B 0C 0C 0D 0D 0E 0E 0F 0F ; ................
3E0A  .byte 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 ; ................
3E1A  .byte 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 ; ................
3E2A  .byte 0F 0F 0E 0E 0D 0D 0C 0C 0B 0B 0A 0A 09 09 08 08 ; ................
3E3A  .byte 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F ; ................
3E4A  .byte 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F 1F ; ................
3E5A  .byte 17 17 17 17 17 17 17 17 17 17 17 17 17 17 17 17 ; ................
3E6A  .byte 17 17 17 17 17 17 17 17 17 17 17 17 17 17 17 17 ; ................
3E7A  .byte 6A 3A DA 3E FA 3E 1A 3F 3A 3F 5A 3F 7A 3F 9A 3F ; j:.>.>.?:?Z?z?.?
3E8A  .byte BA 3F DA 3F FA 3F 1A 40 3A 40 5A 40 7A 40 9A 40 ; .?.?.?.@:@Z@z@.@
3E9A  .byte BA 40 DA 40 FA 40 1A 41 3A 41 5A 41 7A 41 9A 41 ; .@.@.@.A:AZAzA.A
3EAA  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A ; j:j:j:j:j:j:j:j:
3EBA  .byte 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A 6A 3A BA 41 ; j:j:j:j:j:j:j:.A
3ECA  .byte DA 41 FA 41 1A 42 3A 42 5A 42 7A 42 9A 42 BA 42 ; .A.A.B:BZBzB.B.B
3EDA  .byte 10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E 1F ; ................
3EEA  .byte 20 21 22 23 24 25 26 27 28 29 2A 2B 2C 2D 2E 2F ;  !"#$%&'()*+,-./
3EFA  .byte F0 F1 F2 F3 F4 F5 F6 F7 F8 F9 FA FB FC FD FE FF ; ................
3F0A  .byte 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F ; ................
3F1A  .byte 0F 0E 0D 0C 0B 0A 09 08 07 06 05 04 03 02 01 00 ; ................
3F2A  .byte FF FE FD FC FB FA F9 F8 F7 F6 F5 F4 F3 F2 F1 F0 ; ................
3F3A  .byte 2F 2E 2D 2C 2B 2A 29 28 27 26 25 24 23 22 21 20 ; /.-,+*)('&%$#"! 
3F4A  .byte 1F 1E 1D 1C 1B 1A 19 18 17 16 15 14 13 12 11 10 ; ................
3F5A  .byte F8 F8 F9 F9 FA FA FB FB FC FC FD FD FE FE FF FF ; ................
3F6A  .byte 00 00 01 01 02 02 03 03 04 04 05 05 06 06 07 07 ; ................
3F7A  .byte 08 08 09 09 0A 0A 0B 0B 0C 0C 0D 0D 0E 0E 0F 0F ; ................
3F8A  .byte 10 10 11 11 12 12 13 13 14 14 15 15 16 16 17 17 ; ................
3F9A  .byte 18 18 19 19 1A 1A 1B 1B 1C 1C 1D 1D 1E 1E 1F 1F ; ................
3FAA  .byte 20 20 21 21 22 22 23 23 24 24 25 25 26 26 27 27 ;   !!""##$$%%&&''
3FBA  .byte 27 27 26 26 25 25 24 24 23 23 22 22 21 21 20 20 ; ''&&%%$$##""!!  
3FCA  .byte 1F 1F 1E 1E 1D 1D 1C 1C 1B 1B 1A 1A 19 19 18 18 ; ................
3FDA  .byte 17 17 16 16 15 15 14 14 13 13 12 12 11 11 10 10 ; ................
3FEA  .byte 0F 0F 0E 0E 0D 0D 0C 0C 0B 0B 0A 0A 09 09 08 08 ; ................
3FFA  .byte 07 07 06 06 05 05                               ; ......

; ==== sub_4000 (1 caller) ====
4000  04          INC B
4001  04          INC B
4002  03          INC BC
4003  03          INC BC
4004  02          LD (BC),A
4005  02          LD (BC),A

; ==== sub_4006 (1 caller) ====
4006  01 01 00    LD BC,$0001

; ==== sub_4009 (1 caller) ====
4009  00          NOP
400A  FF          RST $38
400B  FF          RST $38

; ==== sub_400C (1 caller) ====
400C  FE FE       CP $FE
400E  FD          NOP*
400F  FD FC FC FB CALL M,$FBFC
4013  FB          EI
4014  FA FA F9    JP M,$F9FA
4017  F9          LD SP,HL
4018  F8          RET M
4019  F8          RET M
401A  10 10       DJNZ $402C
401C  10 10       DJNZ $402E
401E  10 10       DJNZ $4030
4020  10 11       DJNZ $4033
4022  11 11 11    LD DE,$1111
4025  11 12 12    LD DE,$1212
4028  12          LD (DE),A
4029  12          LD (DE),A
402A  12          LD (DE),A
402B  12          LD (DE),A
402C  12          LD (DE),A
402D  12          LD (DE),A
402E  12          LD (DE),A
402F  11 11 11    LD DE,$1111
4032  11 11 10    LD DE,$1011
4035  10 10       DJNZ $4047
4037  10 10       DJNZ $4049
4039  10 10       DJNZ $404B
403B  10 10       DJNZ $404D
403D  10 10       DJNZ $404F
403F  10 10       DJNZ $4051
4041  11 11 11    LD DE,$1111
4044  11 11 12    LD DE,$1211
4047  12          LD (DE),A
4048  12          LD (DE),A
4049  12          LD (DE),A
404A  13          INC DE
404B  13          INC DE
404C  13          INC DE
404D  14          INC D
404E  14          INC D
404F  15          DEC D
4050  15          DEC D
4051  15          DEC D
4052  16 16       LD D,$16
4054  16 17       LD D,$17
4056  17          RLA
4057  17          RLA
4058  17          RLA
4059  17          RLA
405A  17          RLA
405B  17          RLA
405C  17          RLA
405D  17          RLA
405E  17          RLA
405F  16 16       LD D,$16
4061  16 15       LD D,$15
4063  15          DEC D
4064  15          DEC D
4065  14          INC D
4066  14          INC D
4067  13          INC DE
4068  13          INC DE
4069  13          INC DE
406A  12          LD (DE),A
406B  12          LD (DE),A
406C  12          LD (DE),A
406D  12          LD (DE),A
406E  11 11 11    LD DE,$1111
4071  11 11 10    LD DE,$1011
4074  10 10       DJNZ $4086
4076  10 10       DJNZ $4088
4078  10 10       DJNZ $408A
407A  08          EX AF,AF'
407B  08          EX AF,AF'
407C  08          EX AF,AF'
407D  08          EX AF,AF'
407E  08          EX AF,AF'
407F  08          EX AF,AF'
4080  08          EX AF,AF'
4081  09          ADD HL,BC
4082  09          ADD HL,BC
4083  09          ADD HL,BC
4084  09          ADD HL,BC
4085  09          ADD HL,BC
4086  0A          LD A,(BC)
4087  0A          LD A,(BC)
4088  0A          LD A,(BC)
4089  0A          LD A,(BC)
408A  0B          DEC BC
408B  0B          DEC BC
408C  0B          DEC BC
408D  0C          INC C
408E  0C          INC C
408F  0D          DEC C
4090  0D          DEC C
4091  0D          DEC C
4092  0E 0E       LD C,$0E
4094  0E 0F       LD C,$0F
4096  0F          RRCA
4097  0F          RRCA
4098  0F          RRCA
4099  0F          RRCA
409A  0F          RRCA
409B  0F          RRCA
409C  0F          RRCA
409D  0F          RRCA
409E  0F          RRCA
409F  0E 0E       LD C,$0E
40A1  0E 0D       LD C,$0D
40A3  0D          DEC C
40A4  0D          DEC C
40A5  0C          INC C
40A6  0C          INC C
40A7  0B          DEC BC
40A8  0B          DEC BC
40A9  0B          DEC BC
40AA  0A          LD A,(BC)
40AB  0A          LD A,(BC)
40AC  0A          LD A,(BC)
40AD  0A          LD A,(BC)
40AE  09          ADD HL,BC
40AF  09          ADD HL,BC
40B0  09          ADD HL,BC
40B1  09          ADD HL,BC
40B2  09          ADD HL,BC
40B3  08          EX AF,AF'
40B4  08          EX AF,AF'
40B5  08          EX AF,AF'
40B6  08          EX AF,AF'
40B7  08          EX AF,AF'
40B8  08          EX AF,AF'
40B9  08          EX AF,AF'
40BA  10 10       DJNZ $40CC
40BC  10 10       DJNZ $40CE
40BE  10 10       DJNZ $40D0
40C0  10 10       DJNZ $40D2
40C2  10 10       DJNZ $40D4
40C4  10 10       DJNZ $40D6
40C6  10 10       DJNZ $40D8
40C8  10 10       DJNZ $40DA
40CA  10 10       DJNZ $40DC
40CC  10 10       DJNZ $40DE
40CE  10 10       DJNZ $40E0
40D0  10 10       DJNZ $40E2
40D2  10 10       DJNZ $40E4
40D4  10 10       DJNZ $40E6
40D6  10 10       DJNZ $40E8
40D8  10 10       DJNZ $40EA
40DA  10 11       DJNZ $40ED
40DC  12          LD (DE),A
40DD  13          INC DE
40DE  14          INC D
40DF  15          DEC D
40E0  16 17       LD D,$17
40E2  18 19       JR $40FD
40E4  19          ADD HL,DE
40E5  1A          LD A,(DE)
40E6  1A          LD A,(DE)
40E7  1A          LD A,(DE)
40E8  1B          DEC DE
40E9  1B          DEC DE
40EA  1B          DEC DE
40EB  1B          DEC DE
40EC  1B          DEC DE
40ED  1A          LD A,(DE)
40EE  1A          LD A,(DE)
40EF  1A          LD A,(DE)
40F0  19          ADD HL,DE
40F1  19          ADD HL,DE
40F2  18 17       JR $410B
40F4  .byte 16 14 11 10 10 10 10 10 10                      ; .........
40FD  10 10       DJNZ $410F
40FF  10 10       DJNZ $4111
4101  10 10       DJNZ $4113
4103  10 10       DJNZ $4115
4105  10 10       DJNZ $4117
4107  10 10       DJNZ $4119
4109  10 11       DJNZ $411C
410B  11 12 12    LD DE,$1212
410E  13          INC DE
410F  13          INC DE
4110  14          INC D
4111  14          INC D
4112  15          DEC D
4113  15          DEC D
4114  16 16       LD D,$16
4116  17          RLA
4117  17          RLA
4118  18 18       JR $4132
411A  .byte 18 18                                           ; ..
411C  17          RLA
411D  17          RLA
411E  16 16       LD D,$16
4120  15          DEC D
4121  15          DEC D
4122  14          INC D
4123  14          INC D
4124  13          INC DE
4125  13          INC DE
4126  12          LD (DE),A
4127  12          LD (DE),A
4128  11 11 10    LD DE,$1011
412B  10 10       DJNZ $413D
412D  10 10       DJNZ $413F
412F  10 10       DJNZ $4141
4131  10 10       DJNZ $4143
4133  10 10       DJNZ $4145
4135  10 10       DJNZ $4147
4137  10 10       DJNZ $4149
4139  10 08       DJNZ $4143
413B  08          EX AF,AF'
413C  09          ADD HL,BC
413D  09          ADD HL,BC
413E  0A          LD A,(BC)
413F  0A          LD A,(BC)
4140  0B          DEC BC
4141  0B          DEC BC
4142  0C          INC C
4143  0C          INC C
4144  0D          DEC C
4145  0D          DEC C
4146  0E 0E       LD C,$0E
4148  0F          RRCA
4149  0F          RRCA
414A  10 10       DJNZ $415C
414C  10 10       DJNZ $415E
414E  10 10       DJNZ $4160
4150  10 10       DJNZ $4162
4152  10 10       DJNZ $4164
4154  10 10       DJNZ $4166
4156  10 10       DJNZ $4168
4158  10 10       DJNZ $416A
415A  10 10       DJNZ $416C
415C  10 10       DJNZ $416E
415E  10 10       DJNZ $4170
4160  10 10       DJNZ $4172
4162  10 10       DJNZ $4174
4164  10 10       DJNZ $4176
4166  10 10       DJNZ $4178
4168  10 10       DJNZ $417A
416A  0F          RRCA
416B  0F          RRCA
416C  0E 0E       LD C,$0E
416E  0D          DEC C
416F  0D          DEC C
4170  0C          INC C
4171  0C          INC C
4172  0B          DEC BC
4173  0B          DEC BC
4174  0A          LD A,(BC)
4175  0A          LD A,(BC)
4176  09          ADD HL,BC
4177  09          ADD HL,BC
4178  08          EX AF,AF'
4179  08          EX AF,AF'
417A  FF          RST $38
417B  FF          RST $38
417C  FF          RST $38
417D  FF          RST $38
417E  FF          RST $38
417F  FF          RST $38
4180  FF          RST $38
4181  FF          RST $38
4182  FF          RST $38
4183  FF          RST $38
4184  FF          RST $38
4185  FF          RST $38
4186  FF          RST $38
4187  FF          RST $38
4188  FF          RST $38
4189  FF          RST $38
418A  FF          RST $38
418B  FF          RST $38
418C  FF          RST $38
418D  FF          RST $38
418E  FF          RST $38
418F  FF          RST $38
4190  FF          RST $38
4191  FF          RST $38
4192  FF          RST $38
4193  FF          RST $38
4194  FF          RST $38
4195  FF          RST $38
4196  FF          RST $38
4197  FF          RST $38
4198  FF          RST $38
4199  FF          RST $38
419A  08          EX AF,AF'
419B  08          EX AF,AF'
419C  08          EX AF,AF'
419D  08          EX AF,AF'
419E  09          ADD HL,BC
419F  09          ADD HL,BC
41A0  09          ADD HL,BC
41A1  09          ADD HL,BC
41A2  0A          LD A,(BC)
41A3  0A          LD A,(BC)
41A4  0A          LD A,(BC)
41A5  0A          LD A,(BC)
41A6  0B          DEC BC
41A7  0B          DEC BC
41A8  0B          DEC BC
41A9  0B          DEC BC
41AA  0B          DEC BC
41AB  0B          DEC BC
41AC  0B          DEC BC
41AD  0B          DEC BC
41AE  0A          LD A,(BC)
41AF  0A          LD A,(BC)
41B0  0A          LD A,(BC)
41B1  0A          LD A,(BC)
41B2  09          ADD HL,BC
41B3  09          ADD HL,BC
41B4  09          ADD HL,BC
41B5  09          ADD HL,BC
41B6  08          EX AF,AF'
41B7  08          EX AF,AF'
41B8  08          EX AF,AF'
41B9  08          EX AF,AF'
41BA  10 10       DJNZ $41CC
41BC  10 10       DJNZ $41CE
41BE  10 10       DJNZ $41D0
41C0  10 10       DJNZ $41D2
41C2  10 10       DJNZ $41D4
41C4  10 10       DJNZ $41D6
41C6  10 10       DJNZ $41D8
41C8  10 10       DJNZ $41DA
41CA  10 10       DJNZ $41DC
41CC  10 10       DJNZ $41DE
41CE  10 10       DJNZ $41E0
41D0  10 10       DJNZ $41E2
41D2  10 10       DJNZ $41E4
41D4  10 10       DJNZ $41E6
41D6  10 10       DJNZ $41E8
41D8  10 10       DJNZ $41EA
41DA  08          EX AF,AF'
41DB  08          EX AF,AF'
41DC  08          EX AF,AF'
41DD  08          EX AF,AF'
41DE  08          EX AF,AF'
41DF  08          EX AF,AF'
41E0  08          EX AF,AF'
41E1  08          EX AF,AF'
41E2  08          EX AF,AF'
41E3  08          EX AF,AF'
41E4  08          EX AF,AF'
41E5  08          EX AF,AF'
41E6  08          EX AF,AF'
41E7  08          EX AF,AF'
41E8  08          EX AF,AF'
41E9  08          EX AF,AF'
41EA  08          EX AF,AF'
41EB  08          EX AF,AF'
41EC  08          EX AF,AF'
41ED  08          EX AF,AF'
41EE  08          EX AF,AF'
41EF  08          EX AF,AF'
41F0  08          EX AF,AF'
41F1  08          EX AF,AF'
41F2  08          EX AF,AF'
41F3  08          EX AF,AF'
41F4  08          EX AF,AF'
41F5  08          EX AF,AF'
41F6  08          EX AF,AF'
41F7  08          EX AF,AF'
41F8  08          EX AF,AF'
41F9  08          EX AF,AF'
41FA  08          EX AF,AF'
41FB  08          EX AF,AF'
41FC  08          EX AF,AF'
41FD  08          EX AF,AF'
41FE  09          ADD HL,BC
41FF  09          ADD HL,BC
4200  09          ADD HL,BC
4201  09          ADD HL,BC
4202  0A          LD A,(BC)
4203  0A          LD A,(BC)
4204  0A          LD A,(BC)
4205  0A          LD A,(BC)
4206  0B          DEC BC
4207  0B          DEC BC
4208  0B          DEC BC
4209  0B          DEC BC
420A  0C          INC C
420B  0C          INC C
420C  0C          INC C
420D  0C          INC C
420E  0D          DEC C
420F  0D          DEC C
4210  0D          DEC C
4211  0D          DEC C
4212  0E 0E       LD C,$0E
4214  0E 0E       LD C,$0E
4216  0F          RRCA
4217  0F          RRCA
4218  0F          RRCA
4219  0F          RRCA
421A  0F          RRCA
421B  0F          RRCA
421C  0F          RRCA
421D  0F          RRCA
421E  0E 0E       LD C,$0E
4220  0E 0E       LD C,$0E
4222  0D          DEC C
4223  0D          DEC C
4224  0D          DEC C
4225  0D          DEC C
4226  0C          INC C
4227  0C          INC C
4228  0C          INC C
4229  0C          INC C
422A  0B          DEC BC
422B  0B          DEC BC
422C  0B          DEC BC
422D  0B          DEC BC
422E  0A          LD A,(BC)
422F  0A          LD A,(BC)
4230  0A          LD A,(BC)
4231  0A          LD A,(BC)
4232  09          ADD HL,BC
4233  09          ADD HL,BC
4234  09          ADD HL,BC
4235  09          ADD HL,BC
4236  08          EX AF,AF'
4237  08          EX AF,AF'
4238  08          EX AF,AF'
4239  08          EX AF,AF'
423A  07          RLCA
423B  07          RLCA
423C  06 06       LD B,$06
423E  05          DEC B
423F  05          DEC B
4240  04          INC B
4241  04          INC B
4242  03          INC BC
4243  03          INC BC
4244  02          LD (BC),A
4245  02          LD (BC),A
4246  01 01 00    LD BC,$0001
4249  00          NOP
424A  00          NOP
424B  00          NOP
424C  01 01 02    LD BC,$0201
424F  02          LD (BC),A
4250  03          INC BC
4251  03          INC BC
4252  04          INC B
4253  04          INC B
4254  05          DEC B
4255  05          DEC B
4256  06 06       LD B,$06
4258  07          RLCA
4259  07          RLCA
425A  08          EX AF,AF'
425B  08          EX AF,AF'
425C  08          EX AF,AF'
425D  08          EX AF,AF'
425E  09          ADD HL,BC
425F  09          ADD HL,BC
4260  09          ADD HL,BC
4261  09          ADD HL,BC
4262  0A          LD A,(BC)
4263  0A          LD A,(BC)
4264  0A          LD A,(BC)
4265  0A          LD A,(BC)
4266  0B          DEC BC
4267  0B          DEC BC
4268  0C          INC C
4269  0C          INC C
426A  0C          INC C
426B  0C          INC C
426C  0B          DEC BC
426D  0B          DEC BC
426E  0A          LD A,(BC)
426F  0A          LD A,(BC)
4270  0A          LD A,(BC)
4271  0A          LD A,(BC)
4272  09          ADD HL,BC
4273  09          ADD HL,BC
4274  09          ADD HL,BC
4275  09          ADD HL,BC
4276  08          EX AF,AF'
4277  08          EX AF,AF'
4278  08          EX AF,AF'
4279  08          EX AF,AF'
427A  80          ADD A,B
427B  80          ADD A,B
427C  80          ADD A,B
427D  80          ADD A,B
427E  80          ADD A,B
427F  80          ADD A,B
4280  80          ADD A,B
4281  80          ADD A,B
4282  80          ADD A,B
4283  80          ADD A,B
4284  80          ADD A,B
4285  80          ADD A,B
4286  80          ADD A,B
4287  80          ADD A,B
4288  80          ADD A,B
4289  80          ADD A,B
428A  10 10       DJNZ $429C
428C  10 10       DJNZ $429E
428E  10 10       DJNZ $42A0
4290  10 10       DJNZ $42A2
4292  10 10       DJNZ $42A4
4294  10 10       DJNZ $42A6
4296  10 10       DJNZ $42A8
4298  10 10       DJNZ $42AA
429A  10 10       DJNZ $42AC
429C  10 10       DJNZ $42AE
429E  10 10       DJNZ $42B0
42A0  10 10       DJNZ $42B2
42A2  10 10       DJNZ $42B4
42A4  10 10       DJNZ $42B6
42A6  10 10       DJNZ $42B8
42A8  10 10       DJNZ $42BA
42AA  80          ADD A,B
42AB  80          ADD A,B
42AC  80          ADD A,B
42AD  80          ADD A,B
42AE  80          ADD A,B
42AF  80          ADD A,B
42B0  80          ADD A,B
42B1  80          ADD A,B
42B2  80          ADD A,B
42B3  80          ADD A,B
42B4  80          ADD A,B
42B5  80          ADD A,B
42B6  80          ADD A,B
42B7  80          ADD A,B
42B8  80          ADD A,B
42B9  80          ADD A,B
42BA  16 16       LD D,$16
42BC  16 16       LD D,$16
42BE  16 16       LD D,$16
42C0  16 16       LD D,$16
42C2  16 16       LD D,$16
42C4  16 16       LD D,$16
42C6  16 16       LD D,$16
42C8  16 16       LD D,$16
42CA  16 16       LD D,$16
42CC  16 16       LD D,$16
42CE  16 16       LD D,$16
42D0  16 16       LD D,$16
42D2  16 16       LD D,$16
42D4  16 16       LD D,$16
42D6  16 16       LD D,$16
42D8  16 16       LD D,$16

; ==== sub_42DA (1 caller) ====
42DA  3A 1A D2    LD A,($D21A)
42DD  E6 BF       AND $BF
42DF  32 1A D2    LD ($D21A),A
42E2  FD CB 00 86 RES 0,(IY+0)
42E6  CD 27 03    CALL $0327
42E9  21 00 15    LD HL,$1500
42EC  11 00 00    LD DE,$0000
42EF  3E 09       LD A,$09
42F1  CD 06 04    CALL $0406
42F4  21 6F 43    LD HL,$436F
42F7  11 00 20    LD DE,$2000
42FA  3E 09       LD A,$09
42FC  CD 06 04    CALL $0406
42FF  21 38 67    LD HL,$6738
4302  11 00 38    LD DE,$3800
4305  01 2D 01    LD BC,$012D
4308  3E 00       LD A,$00
430A  32 0F D2    LD ($D20F),A
430D  3E 05       LD A,$05
430F  CD 02 05    CALL $0502
4312  DB 00       IN A,($00)
4314  07          RLCA
4315  07          RLCA
4316  30 0A       JR NC,$4322
4318  AF          XOR A
4319  32 0F D2    LD ($D20F),A
431C  21 4C 44    LD HL,$444C
431F  CD 12 06    CALL $0612
4322  21 51 44    LD HL,$4451
4325  CD 12 06    CALL $0612
4328  AF          XOR A
4329  32 4B D2    LD ($D24B),A
432C  32 4C D2    LD ($D24C),A
432F  21 09 09    LD HL,$0909
4332  22 2C D2    LD ($D22C),HL
4335  FD CB 00 CE SET 1,(IY+0)
4339  3E 06       LD A,$06
433B  DF          RST $18
433C  AF          XOR A
433D  32 17 D2    LD ($D217),A
4340  3E 01       LD A,$01
4342  32 10 D2    LD ($D210),A
4345  21 DD 43    LD HL,$43DD
4348  22 11 D2    LD ($D211),HL
434B  3A 1A D2    LD A,($D21A)
434E  F6 40       OR $40
4350  32 1A D2    LD ($D21A),A
4353  FD CB 00 86 RES 0,(IY+0)
4357  CD 27 03    CALL $0327
435A  3A 17 D2    LD A,($D217)
435D  3C          INC A
435E  FE 64       CP $64
4360  38 01       JR C,$4363
4362  AF          XOR A
4363  32 17 D2    LD ($D217),A
4366  21 B3 43    LD HL,$43B3
4369  FE 40       CP $40
436B  38 03       JR C,$4370
436D  21 C8 43    LD HL,$43C8
4370  AF          XOR A
4371  32 0F D2    LD ($D20F),A
4374  CD 12 06    CALL $0612
4377  3A 10 D2    LD A,($D210)
437A  3D          DEC A
437B  32 10 D2    LD ($D210),A
437E  20 16       JR NZ,$4396
4380  2A 11 D2    LD HL,($D211)
4383  5E          LD E,(HL)
4384  23          INC HL
4385  56          LD D,(HL)
4386  23          INC HL
4387  7E          LD A,(HL)
4388  23          INC HL
4389  A7          AND A
438A  28 25       JR Z,$43B1
438C  32 10 D2    LD ($D210),A
438F  22 11 D2    LD ($D211),HL
4392  ED 53 13 D2 LD ($D213),DE
4396  21 00 D0    LD HL,$D000
4399  22 36 D2    LD ($D236),HL
439C  21 88 00    LD HL,$0088
439F  11 20 00    LD DE,$0020
43A2  ED 4B 13 D2 LD BC,($D213)
43A6  CD 07 2F    CALL $2F07
43A9  FD CB 03 7E BIT 7,(IY+3)
43AD  C2 4B 43    JP NZ,$434B
43B0  37          SCF
43B1  E7          RST $20
43B2  C9          RET
43B3  .byte 07 12 E3 E4 E5 E6 E6 01 E6 E7 E8 E4 E7 01 E9 EB ; ................
43C3  .byte E7 E7 EA EC FF 07 12 01 01 01 01 01 01 01 01 01 ; ................
43D3  .byte 01 01 01 01 01 01 01 01 01 FF 28 44 08 3A 44 08 ; ..........(D.:D.
43E3  .byte 28 44 08 3A 44 08 28 44 08 3A 44 08 28 44 08 3A ; (D.:D.(D.:D.(D.:
43F3  .byte 44 08 28 44 08 3A 44 08 28 44 08 3A 44 08 28 44 ; D.(D.:D.(D.:D.(D
4403  .byte 08 3A 44 08 28 44 08 3A 44 08 28 44 08 3A 44 08 ; .:D.(D.:D.(D.:D.
4413  .byte 28 44 08 3A 44 08 28 44 08 3A 44 08 28 44 FF 28 ; (D.:D.(D.:D.(D.(
4423  .byte 44 FF 1F 44 00 00 02 04 FF FF FF 20 22 24 FF FF ; D..D....... "$..
4433  .byte FF 40 42 44 FF FF FF 06 08 FF FF FF FF 26 28 FF ; .@BD.........&(.
4443  .byte FF FF FF 46 48 FF FF FF FF 16 0B 9E 9F FF 0F 14 ; ...FH...........
4453  .byte F1 FF 01 00 00 00 00 00 01 00 00 05 00 00 10 00 ; ................
4463  .byte 00 30 00 00 50 00 01 00 00 03 00 00 04 00 00 05 ; .0..P...........
4473  .byte 00 00 08 00 00 10 00 00 20 00 00 30 00 00 05 00 ; ........ ..0....
4483  .byte 03 00 02 30 02 00 01 30 01 00 00 30 00 25 00 24 ; ...0...0...0.%.$
4493  .byte 00 23 00 22 00 21 00 20 00 00                   ; .#.".!. ..

; ==== sub_449D (2 callers) ====
449D  3A 38 D2    LD A,($D238)
44A0  FE 13       CP $13
44A2  CA 63 46    JP Z,$4663
44A5  3A 1A D2    LD A,($D21A)
44A8  E6 BF       AND $BF
44AA  32 1A D2    LD ($D21A),A
44AD  FD CB 00 86 RES 0,(IY+0)
44B1  CD 27 03    CALL $0327
44B4  21 54 B3    LD HL,$B354
44B7  11 00 30    LD DE,$3000
44BA  3E 09       LD A,$09
44BC  CD 06 04    CALL $0406
44BF  21 31 2A    LD HL,$2A31
44C2  11 00 00    LD DE,$0000
44C5  3E 09       LD A,$09
44C7  CD 06 04    CALL $0406
44CA  21 65 68    LD HL,$6865
44CD  01 84 00    LD BC,$0084
44D0  11 00 38    LD DE,$3800
44D3  3A 38 D2    LD A,($D238)
44D6  FE 1C       CP $1C
44D8  38 09       JR C,$44E3
44DA  21 E9 68    LD HL,$68E9
44DD  01 79 00    LD BC,$0079
44E0  11 00 38    LD DE,$3800
44E3  AF          XOR A
44E4  32 0F D2    LD ($D20F),A
44E7  3E 05       LD A,$05
44E9  CD 02 05    CALL $0502
44EC  21 45 46    LD HL,$4645
44EF  0E 13       LD C,$13
44F1  3A 79 D2    LD A,($D279)
44F4  A7          AND A
44F5  C4 0F 46    CALL NZ,$460F
44F8  3A 38 D2    LD A,($D238)
44FB  FE 1C       CP $1C
44FD  30 37       JR NC,$4536
44FF  3E 17       LD A,$17
4501  32 BF D2    LD ($D2BF),A
4504  3E 05       LD A,$05
4506  32 C0 D2    LD ($D2C0),A
4509  3A 38 D2    LD A,($D238)
450C  5F          LD E,A
450D  16 00       LD D,$00
450F  21 AC 4A    LD HL,$4AAC
4512  19          ADD HL,DE
4513  5E          LD E,(HL)
4514  21 94 4A    LD HL,$4A94
4517  19          ADD HL,DE
4518  06 04       LD B,$04
451A  C5          PUSH BC
451B  E5          PUSH HL
451C  11 C0 D2    LD DE,$D2C0
451F  1A          LD A,(DE)
4520  3C          INC A
4521  12          LD (DE),A
4522  13          INC DE
4523  ED A0       LDI
4525  ED A0       LDI
4527  3E FF       LD A,$FF
4529  12          LD (DE),A
452A  21 BF D2    LD HL,$D2BF
452D  CD 12 06    CALL $0612
4530  E1          POP HL
4531  C1          POP BC
4532  23          INC HL
4533  23          INC HL
4534  10 E4       DJNZ $451A
4536  AF          XOR A
4537  32 4B D2    LD ($D24B),A
453A  32 4C D2    LD ($D24C),A
453D  21 0A 0B    LD HL,$0B0A
4540  22 2C D2    LD ($D22C),HL
4543  3A 38 D2    LD A,($D238)
4546  FE 1C       CP $1C
4548  38 12       JR C,$455C
454A  21 7B D2    LD HL,$D27B
454D  34          INC (HL)
454E  FD CB 09 56 BIT 2,(IY+9)
4552  20 08       JR NZ,$455C
4554  21 7C D2    LD HL,$D27C
4557  34          INC (HL)
4558  21 7F D2    LD HL,$D27F
455B  34          INC (HL)
455C  FD CB 09 56 BIT 2,(IY+9)
4560  C4 4D 46    CALL NZ,$464D
4563  FD CB 09 5E BIT 3,(IY+9)
4567  C4 5A 46    CALL NZ,$465A
456A  21 81 44    LD HL,$4481
456D  11 57 44    LD DE,$4457
4570  06 0E       LD B,$0E
4572  3A CF D2    LD A,($D2CF)
4575  BE          CP (HL)
4576  20 0A       JR NZ,$4582
4578  23          INC HL
4579  3A D0 D2    LD A,($D2D0)
457C  BE          CP (HL)
457D  30 0F       JR NC,$458E
457F  23          INC HL
4580  18 04       JR $4586
4582  30 0A       JR NC,$458E
4584  23          INC HL
4585  23          INC HL
4586  13          INC DE
4587  13          INC DE
4588  13          INC DE
4589  10 E7       DJNZ $4572
458B  11 57 44    LD DE,$4457
458E  21 13 D2    LD HL,$D213
4591  36 00       LD (HL),$00
4593  23          INC HL
4594  EB          EX DE,HL
4595  3A 38 D2    LD A,($D238)
4598  FE 1C       CP $1C
459A  38 03       JR C,$459F
459C  21 9B 48    LD HL,$489B
459F  ED A0       LDI
45A1  ED A0       LDI
45A3  ED A0       LDI
45A5  FD CB 00 CE SET 1,(IY+0)
45A9  06 78       LD B,$78
45AB  C5          PUSH BC
45AC  3A 1A D2    LD A,($D21A)
45AF  F6 40       OR $40
45B1  32 1A D2    LD ($D21A),A
45B4  FD CB 00 86 RES 0,(IY+0)
45B8  CD 27 03    CALL $0327
45BB  CD 9F 48    CALL $489F
45BE  C1          POP BC
45BF  10 EA       DJNZ $45AB
45C1  FD CB 00 86 RES 0,(IY+0)
45C5  CD 27 03    CALL $0327
45C8  CD 9F 48    CALL $489F
45CB  CD 3B 48    CALL $483B
45CE  3A 38 D2    LD A,($D238)
45D1  FE 1C       CP $1C
45D3  DC 66 48    CALL C,$4866
45D6  3A 17 D2    LD A,($D217)
45D9  3C          INC A
45DA  32 17 D2    LD ($D217),A
45DD  E6 03       AND $03
45DF  20 03       JR NZ,$45E4
45E1  3E 02       LD A,$02
45E3  EF          RST $28
45E4  2A 13 D2    LD HL,($D213)
45E7  ED 5B 15 D2 LD DE,($D215)
45EB  3A A9 D2    LD A,($D2A9)
45EE  B4          OR H
45EF  B5          OR L
45F0  B2          OR D
45F1  B3          OR E
45F2  C2 C1 45    JP NZ,$45C1
45F5  06 B4       LD B,$B4
45F7  C5          PUSH BC
45F8  FD CB 00 86 RES 0,(IY+0)
45FC  CD 27 03    CALL $0327
45FF  CD 9F 48    CALL $489F
4602  C1          POP BC
4603  FD 7E 03    LD A,(IY+3)
4606  E6 B0       AND $B0
4608  FE B0       CP $B0
460A  20 02       JR NZ,$460E
460C  10 E9       DJNZ $45F7
460E  C9          RET

; ==== sub_460F (2 callers) ====
460F  47          LD B,A
4610  C5          PUSH BC
4611  11 BF D2    LD DE,$D2BF
4614  47          LD B,A
4615  79          LD A,C
4616  90          SUB B
4617  12          LD (DE),A
4618  13          INC DE
4619  01 04 00    LD BC,$0004
461C  ED B0       LDIR
461E  12          LD (DE),A
461F  13          INC DE
4620  01 04 00    LD BC,$0004
4623  ED B0       LDIR
4625  C1          POP BC
4626  AF          XOR A
4627  32 0F D2    LD ($D20F),A
462A  C5          PUSH BC
462B  21 BF D2    LD HL,$D2BF
462E  CD 12 06    CALL $0612
4631  21 C4 D2    LD HL,$D2C4
4634  CD 12 06    CALL $0612
4637  21 BF D2    LD HL,$D2BF
463A  34          INC (HL)
463B  34          INC (HL)
463C  21 C4 D2    LD HL,$D2C4
463F  34          INC (HL)
4640  34          INC (HL)
4641  C1          POP BC
4642  10 E6       DJNZ $462A
4644  C9          RET
4645  .byte 13 AD AE FF 14 BD BE FF                         ; ........

; ==== sub_464D (1 caller) ====
464D  AF          XOR A
464E  32 A9 D2    LD ($D2A9),A
4651  FD CB 09 9E RES 3,(IY+9)
4655  FD CB 09 96 RES 2,(IY+9)
4659  C9          RET

; ==== sub_465A (1 caller) ====
465A  21 7E D2    LD HL,$D27E
465D  34          INC (HL)
465E  FD CB 09 9E RES 3,(IY+9)
4662  C9          RET
4663  3E FF       LD A,$FF
4665  32 F8 D2    LD ($D2F8),A
4668  0E 00       LD C,$00
466A  3A 79 D2    LD A,($D279)
466D  FE 06       CP $06
466F  38 02       JR C,$4673
4671  0E 05       LD C,$05
4673  3A 7A D2    LD A,($D27A)
4676  FE 12       CP $12
4678  38 05       JR C,$467F
467A  79          LD A,C
467B  C6 05       ADD A,$05
467D  27          DAA
467E  4F          LD C,A
467F  3A 7B D2    LD A,($D27B)
4682  FE 08       CP $08
4684  38 05       JR C,$468B
4686  79          LD A,C
4687  C6 05       ADD A,$05
4689  27          DAA
468A  4F          LD C,A
468B  3A 7C D2    LD A,($D27C)
468E  FE 08       CP $08
4690  38 05       JR C,$4697
4692  79          LD A,C
4693  C6 05       ADD A,$05
4695  27          DAA
4696  4F          LD C,A
4697  3A 7D D2    LD A,($D27D)
469A  A7          AND A
469B  20 05       JR NZ,$46A2
469D  79          LD A,C
469E  C6 0A       ADD A,$0A
46A0  27          DAA
46A1  4F          LD C,A
46A2  79          LD A,C
46A3  FE 30       CP $30
46A5  20 08       JR NZ,$46AF
46A7  79          LD A,C
46A8  C6 0A       ADD A,$0A
46AA  27          DAA
46AB  C6 0A       ADD A,$0A
46AD  27          DAA
46AE  4F          LD C,A
46AF  21 FA D2    LD HL,$D2FA
46B2  71          LD (HL),C
46B3  23          INC HL
46B4  36 00       LD (HL),$00
46B6  23          INC HL
46B7  36 00       LD (HL),$00
46B9  21 F0 49    LD HL,$49F0
46BC  CD 12 06    CALL $0612
46BF  21 02 4A    LD HL,$4A02
46C2  CD 12 06    CALL $0612
46C5  21 14 4A    LD HL,$4A14
46C8  CD 12 06    CALL $0612
46CB  21 26 4A    LD HL,$4A26
46CE  CD 12 06    CALL $0612
46D1  21 33 4A    LD HL,$4A33
46D4  CD 12 06    CALL $0612
46D7  21 40 4A    LD HL,$4A40
46DA  CD 12 06    CALL $0612
46DD  21 4D 4A    LD HL,$4A4D
46E0  CD 12 06    CALL $0612
46E3  21 5E 4A    LD HL,$4A5E
46E6  CD 12 06    CALL $0612
46E9  AF          XOR A
46EA  32 17 D2    LD ($D217),A
46ED  01 B4 00    LD BC,$00B4
46F0  CD 94 47    CALL $4794
46F3  01 3C 00    LD BC,$003C
46F6  CD 94 47    CALL $4794
46F9  3A 79 D2    LD A,($D279)
46FC  A7          AND A
46FD  28 12       JR Z,$4711
46FF  3D          DEC A
4700  32 79 D2    LD ($D279),A
4703  11 00 00    LD DE,$0000
4706  0E 02       LD C,$02
4708  CD AA 33    CALL $33AA
470B  3E 02       LD A,$02
470D  EF          RST $28
470E  C3 F3 46    JP $46F3
4711  01 B4 00    LD BC,$00B4
4714  CD 94 47    CALL $4794
4717  3E 01       LD A,$01
4719  32 17 D2    LD ($D217),A
471C  21 6E 4A    LD HL,$4A6E
471F  CD 12 06    CALL $0612
4722  01 B4 00    LD BC,$00B4
4725  CD 94 47    CALL $4794
4728  01 1E 00    LD BC,$001E
472B  CD 94 47    CALL $4794
472E  3A 40 D2    LD A,($D240)
4731  A7          AND A
4732  28 12       JR Z,$4746
4734  3D          DEC A
4735  32 40 D2    LD ($D240),A
4738  11 00 50    LD DE,$5000
473B  0E 00       LD C,$00
473D  CD AA 33    CALL $33AA
4740  3E 02       LD A,$02
4742  EF          RST $28
4743  C3 28 47    JP $4728
4746  01 B4 00    LD BC,$00B4
4749  CD 94 47    CALL $4794
474C  3E 02       LD A,$02
474E  32 17 D2    LD ($D217),A
4751  21 7E 4A    LD HL,$4A7E
4754  CD 12 06    CALL $0612
4757  21 5A 4A    LD HL,$4A5A
475A  CD 12 06    CALL $0612
475D  01 B4 00    LD BC,$00B4
4760  CD 94 47    CALL $4794
4763  01 1E 00    LD BC,$001E
4766  CD 94 47    CALL $4794
4769  3A FA D2    LD A,($D2FA)
476C  A7          AND A
476D  28 1E       JR Z,$478D
476F  3D          DEC A
4770  4F          LD C,A
4771  E6 0F       AND $0F
4773  FE 0A       CP $0A
4775  38 04       JR C,$477B
4777  79          LD A,C
4778  D6 06       SUB $06
477A  4F          LD C,A
477B  79          LD A,C
477C  32 FA D2    LD ($D2FA),A
477F  11 00 00    LD DE,$0000
4782  0E 01       LD C,$01
4784  CD AA 33    CALL $33AA
4787  3E 02       LD A,$02
4789  EF          RST $28
478A  C3 63 47    JP $4763
478D  01 E0 01    LD BC,$01E0
4790  CD 94 47    CALL $4794
4793  C9          RET

; ==== sub_4794 (9 callers) ====
4794  C5          PUSH BC
4795  FD CB 00 86 RES 0,(IY+0)
4799  CD 27 03    CALL $0327
479C  FD 36 0A 00 LD (IY+10),$00
47A0  21 00 D0    LD HL,$D000
47A3  22 36 D2    LD ($D236),HL
47A6  21 BB D2    LD HL,$D2BB
47A9  11 BF D2    LD DE,$D2BF
47AC  06 04       LD B,$04
47AE  CD B2 49    CALL $49B2
47B1  EB          EX DE,HL
47B2  2A 36 D2    LD HL,($D236)
47B5  0E 80       LD C,$80
47B7  06 70       LD B,$70
47B9  CD A8 2F    CALL $2FA8
47BC  22 36 D2    LD ($D236),HL
47BF  3A 17 D2    LD A,($D217)
47C2  A7          AND A
47C3  20 34       JR NZ,$47F9
47C5  21 79 D2    LD HL,$D279
47C8  11 BF D2    LD DE,$D2BF
47CB  06 01       LD B,$01
47CD  CD B2 49    CALL $49B2
47D0  EB          EX DE,HL
47D1  2A 36 D2    LD HL,($D236)
47D4  0E 80       LD C,$80
47D6  06 50       LD B,$50
47D8  CD A8 2F    CALL $2FA8
47DB  22 36 D2    LD ($D236),HL
47DE  21 8E 4A    LD HL,$4A8E
47E1  11 BF D2    LD DE,$D2BF
47E4  06 03       LD B,$03
47E6  CD B2 49    CALL $49B2
47E9  EB          EX DE,HL
47EA  2A 36 D2    LD HL,($D236)
47ED  0E 90       LD C,$90
47EF  06 50       LD B,$50
47F1  CD A8 2F    CALL $2FA8
47F4  22 36 D2    LD ($D236),HL
47F7  18 3A       JR $4833
47F9  3D          DEC A
47FA  20 1E       JR NZ,$481A
47FC  CD 69 49    CALL $4969
47FF  21 91 4A    LD HL,$4A91
4802  11 BF D2    LD DE,$D2BF
4805  06 03       LD B,$03
4807  CD B2 49    CALL $49B2
480A  EB          EX DE,HL
480B  2A 36 D2    LD HL,($D236)
480E  0E 90       LD C,$90
4810  06 50       LD B,$50
4812  CD A8 2F    CALL $2FA8
4815  22 36 D2    LD ($D236),HL
4818  18 19       JR $4833
481A  21 FA D2    LD HL,$D2FA
481D  11 BF D2    LD DE,$D2BF
4820  06 03       LD B,$03
4822  CD B2 49    CALL $49B2
4825  EB          EX DE,HL
4826  2A 36 D2    LD HL,($D236)
4829  0E 90       LD C,$90
482B  06 50       LD B,$50
482D  CD A8 2F    CALL $2FA8
4830  22 36 D2    LD ($D236),HL
4833  C1          POP BC
4834  0B          DEC BC
4835  78          LD A,B
4836  B1          OR C
4837  C2 94 47    JP NZ,$4794
483A  C9          RET

; ==== sub_483B (1 caller) ====
483B  21 A9 D2    LD HL,$D2A9
483E  7E          LD A,(HL)
483F  A7          AND A
4840  C8          RET Z
4841  3D          DEC A
4842  4F          LD C,A
4843  E6 0F       AND $0F
4845  FE 0A       CP $0A
4847  38 04       JR C,$484D
4849  79          LD A,C
484A  D6 06       SUB $06
484C  4F          LD C,A
484D  71          LD (HL),C
484E  11 00 01    LD DE,$0100
4851  0E 00       LD C,$00
4853  3A 38 D2    LD A,($D238)
4856  FE 1C       CP $1C
4858  38 08       JR C,$4862
485A  3A 7F D2    LD A,($D27F)
485D  57          LD D,A
485E  3A 80 D2    LD A,($D280)
4861  5F          LD E,A
4862  CD AA 33    CALL $33AA
4865  C9          RET

; ==== sub_4866 (1 caller) ====
4866  2A 13 D2    LD HL,($D213)
4869  ED 5B 15 D2 LD DE,($D215)
486D  7C          LD A,H
486E  B5          OR L
486F  B2          OR D
4870  B3          OR E
4871  C8          RET Z
4872  06 03       LD B,$03
4874  21 15 D2    LD HL,$D215
4877  37          SCF
4878  7E          LD A,(HL)
4879  DE 00       SBC A,$00
487B  4F          LD C,A
487C  E6 0F       AND $0F
487E  FE 0A       CP $0A
4880  38 04       JR C,$4886
4882  79          LD A,C
4883  D6 06       SUB $06
4885  4F          LD C,A
4886  79          LD A,C
4887  FE A0       CP $A0
4889  38 02       JR C,$488D
488B  D6 60       SUB $60
488D  77          LD (HL),A
488E  3F          CCF
488F  2B          DEC HL
4890  10 E6       DJNZ $4878
4892  11 00 01    LD DE,$0100
4895  0E 00       LD C,$00
4897  CD AA 33    CALL $33AA
489A  C9          RET
489B  .byte 00 00 00 00                                     ; ....

; ==== sub_489F (3 callers) ====
489F  FD 36 0A 00 LD (IY+10),$00
48A3  21 00 D0    LD HL,$D000
48A6  22 36 D2    LD ($D236),HL
48A9  21 BB D2    LD HL,$D2BB
48AC  11 BF D2    LD DE,$D2BF
48AF  06 04       LD B,$04
48B1  CD B2 49    CALL $49B2
48B4  EB          EX DE,HL
48B5  2A 36 D2    LD HL,($D236)
48B8  0E 80       LD C,$80
48BA  06 58       LD B,$58
48BC  3A 38 D2    LD A,($D238)
48BF  FE 1C       CP $1C
48C1  38 04       JR C,$48C7
48C3  0E 80       LD C,$80
48C5  06 50       LD B,$50
48C7  CD A8 2F    CALL $2FA8
48CA  22 36 D2    LD ($D236),HL
48CD  21 A9 D2    LD HL,$D2A9
48D0  11 BF D2    LD DE,$D2BF
48D3  06 01       LD B,$01
48D5  CD B2 49    CALL $49B2
48D8  EB          EX DE,HL
48D9  2A 36 D2    LD HL,($D236)
48DC  0E 88       LD C,$88
48DE  06 80       LD B,$80
48E0  3A 38 D2    LD A,($D238)
48E3  FE 1C       CP $1C
48E5  38 04       JR C,$48EB
48E7  0E 80       LD C,$80
48E9  06 68       LD B,$68
48EB  CD A8 2F    CALL $2FA8
48EE  22 36 D2    LD ($D236),HL
48F1  3A 38 D2    LD A,($D238)
48F4  FE 1C       CP $1C
48F6  38 0F       JR C,$4907
48F8  21 7F D2    LD HL,$D27F
48FB  11 BF D2    LD DE,$D2BF
48FE  06 02       LD B,$02
4900  CD B2 49    CALL $49B2
4903  06 68       LD B,$68
4905  18 0D       JR $4914
4907  21 55 44    LD HL,$4455
490A  11 BF D2    LD DE,$D2BF
490D  06 02       LD B,$02
490F  CD B2 49    CALL $49B2
4912  06 80       LD B,$80
4914  0E A0       LD C,$A0
4916  EB          EX DE,HL
4917  2A 36 D2    LD HL,($D236)
491A  CD A8 2F    CALL $2FA8
491D  22 36 D2    LD ($D236),HL
4920  CD 69 49    CALL $4969
4923  3A 38 D2    LD A,($D238)
4926  FE 1C       CP $1C
4928  30 25       JR NC,$494F
492A  21 13 D2    LD HL,$D213
492D  11 BF D2    LD DE,$D2BF
4930  06 04       LD B,$04
4932  CD B2 49    CALL $49B2
4935  EB          EX DE,HL
4936  2A 36 D2    LD HL,($D236)
4939  0E 80       LD C,$80
493B  06 70       LD B,$70
493D  3A 38 D2    LD A,($D238)
4940  FE 1C       CP $1C
4942  38 04       JR C,$4948
4944  0E 78       LD C,$78
4946  06 78       LD B,$78
4948  CD A8 2F    CALL $2FA8
494B  22 36 D2    LD ($D236),HL
494E  C9          RET
494F  21 7E D2    LD HL,$D27E
4952  11 BF D2    LD DE,$D2BF
4955  06 01       LD B,$01
4957  CD B2 49    CALL $49B2
495A  EB          EX DE,HL
495B  2A 36 D2    LD HL,($D236)
495E  0E 98       LD C,$98
4960  06 80       LD B,$80
4962  CD A8 2F    CALL $2FA8
4965  22 36 D2    LD ($D236),HL
4968  C9          RET

; ==== sub_4969 (2 callers) ====
4969  3A 40 D2    LD A,($D240)
496C  6F          LD L,A
496D  26 00       LD H,$00
496F  0E 0A       LD C,$0A
4971  CD 72 06    CALL $0672
4974  7D          LD A,L
4975  87          ADD A,A
4976  C6 80       ADD A,$80
4978  32 BF D2    LD ($D2BF),A
497B  0E 0A       LD C,$0A
497D  CD 5F 06    CALL $065F
4980  EB          EX DE,HL
4981  3A 40 D2    LD A,($D240)
4984  6F          LD L,A
4985  26 00       LD H,$00
4987  A7          AND A
4988  ED 52       SBC HL,DE
498A  7D          LD A,L
498B  87          ADD A,A
498C  C6 80       ADD A,$80
498E  32 C0 D2    LD ($D2C0),A
4991  3E FF       LD A,$FF
4993  32 C1 D2    LD ($D2C1),A
4996  0E 50       LD C,$50
4998  06 97       LD B,$97
499A  3A 38 D2    LD A,($D238)
499D  FE 13       CP $13
499F  20 04       JR NZ,$49A5
49A1  0E 80       LD C,$80
49A3  06 50       LD B,$50
49A5  2A 36 D2    LD HL,($D236)
49A8  11 BF D2    LD DE,$D2BF
49AB  CD A8 2F    CALL $2FA8
49AE  22 36 D2    LD ($D236),HL
49B1  C9          RET

; ==== sub_49B2 (12 callers) ====
49B2  7E          LD A,(HL)
49B3  E6 F0       AND $F0
49B5  20 1B       JR NZ,$49D2
49B7  3E FE       LD A,$FE
49B9  12          LD (DE),A
49BA  13          INC DE
49BB  7E          LD A,(HL)
49BC  E6 0F       AND $0F
49BE  20 1E       JR NZ,$49DE
49C0  3E FE       LD A,$FE
49C2  12          LD (DE),A
49C3  23          INC HL
49C4  13          INC DE
49C5  10 EB       DJNZ $49B2
49C7  3E FF       LD A,$FF
49C9  12          LD (DE),A
49CA  1B          DEC DE
49CB  3E 80       LD A,$80
49CD  12          LD (DE),A
49CE  21 BF D2    LD HL,$D2BF
49D1  C9          RET
49D2  7E          LD A,(HL)
49D3  0F          RRCA
49D4  0F          RRCA
49D5  0F          RRCA
49D6  0F          RRCA
49D7  E6 0F       AND $0F
49D9  87          ADD A,A
49DA  C6 80       ADD A,$80
49DC  12          LD (DE),A
49DD  13          INC DE
49DE  7E          LD A,(HL)
49DF  E6 0F       AND $0F
49E1  87          ADD A,A
49E2  C6 80       ADD A,$80
49E4  12          LD (DE),A
49E5  23          INC HL
49E6  13          INC DE
49E7  10 E9       DJNZ $49D2
49E9  3E FF       LD A,$FF
49EB  12          LD (DE),A
49EC  21 BF D2    LD HL,$D2BF
49EF  C9          RET
49F0  .byte 07 07 DA DB DB DB DB DB DB DB DB DB DB DB DB DB ; ................
4A00  .byte DC FF 07 08 EA EB EB EB EB EB EB EB EB EB EB EB ; ................
4A10  .byte EB EB EC FF 07 09 FB FC FC FC FC FC FC FC FC FC ; ................
4A20  .byte FC FC FC FC FD FF 0F 09 DA DB DB DB DB DB DB DB ; ................
4A30  .byte DB DC FF 0F 0A EA EB EB EB EB EB EB EB EB EC FF ; ................
4A40  .byte 0F 0B EA EB EB FA EB EB EB EB EB EC FF 0F 0C FB ; ................
4A50  .byte FC FC FC FC FC FC FC FC FD FF 12 0B EB FF 08 08 ; ................
4A60  .byte 36 47 34 61 70 EB 44 50 44 62 34 43 37 FF 08 08 ; 6G4ap.DPDb4C7...
4A70  .byte 70 52 51 40 36 EB 43 44 45 80 EB EB EB FF 08 08 ; pRQ@6.CDE.......
4A80  .byte 70 60 44 36 40 34 43 EB 35 52 51 81 70 FF 02 00 ; p`D6@4C.5RQ.p...
4A90  .byte 00 00 50 00 83 84 93 94 A3 A4 B3 B4 85 86 95 96 ; ..P.............
4AA0  .byte A5 A6 B5 B6 87 88 97 98 A7 A8 B7 B8 00 08 10 00 ; ................
4AB0  .byte 08 10 00 08 10 00 08 10 00 08 10 00 08 10 00 00 ; ................
4AC0  .byte 08 08 08 08 08 08 08 08 00 00 00 00 00 00 00 00 ; ................
4AD0  .byte FD CB 08 8E DD CB 18 7E C4 9A 50 FD CB 07 FE FD ; .......~..P.....
4AE0  .byte CB 05 46 C2 D5 56 3A 13 D4 A7 C4 4D 52 DD CB 18 ; ..F..V:....MR...
4AF0  .byte AE FD CB 06 76 C4 7E 53 3A 86 D2 A7 C4 2C 59 FD ; ....v.~S:....,Y.
4B00  .byte CB 07 46 C4 74 53 FD CB 08 46 C4 52 52 DD CB 18 ; ..F.tS...F.RR...
4B10  .byte 66 C4 6C 52 3A 85 D2 A7 C4 22 55 3A 84 D2 A7 C2 ; f.lR:...."U:....
4B20  .byte 8B 53 FD CB 08 76 C2 14 54 FD CB 08 7E C4 39 55 ; .S...v..T...~.9U
4B30  .byte DD CB 18 66 CA 57 4B 21 E9 4F 11 0F D2 01 09 00 ; ...f.WK!.O......
4B40  .byte ED B0 21 00 01 22 3A D2 21 C0 FD 22 3C D2 21 10 ; ..!..":.!.."<.!.
4B50  .byte 00 22 3E D2 C3 E1 4B DD 7E 15 A7 20 58 FD CB 07 ; .">...K.~.. X...
4B60  .byte 46 20 26 21 D7 4F 11 0F D2 01 09 00 ED B0 21 00 ; F &!.O........!.
4B70  .byte 03 22 3A D2 21 00 FD 22 3C D2 21 38 00 22 3E D2 ; .":.!.."<.!8.">.
4B80  .byte 2A 0C DC 22 0A DC C3 E1 4B DD CB 18 7E 20 D4 21 ; *.."....K...~ .!
4B90  .byte E0 4F 11 0F D2 01 09 00 ED B0 21 00 0C 22 3A D2 ; .O........!..":.
4BA0  .byte 21 00 FD 22 3C D2 21 38 00 22 3E D2 2A 0C DC 22 ; !.."<.!8.">.*.."
4BB0  .byte 0A DC C3 E1 4B 21 F2 4F 11 0F D2 01 09 00 ED B0 ; ....K!.O........
4BC0  .byte 21 00 06 22 3A D2 21 00 FD 22 3C D2 21 38 00 22 ; !..":.!.."<.!8."
4BD0  .byte 3E D2 2A 0C DC 23 22 0A DC 3A 24 D2 E6 03 CC 49 ; >.*..#"..:$....I
4BE0  .byte 52 FD CB 03 4E CC 35 53 FD CB 03 4E C4 57 53 3E ; R...N.5S...N.WS>
4BF0  .byte 05 32 FF FF 32 30 D2 01 08 00 11 10 00 3A 38 D2 ; .2..20.......:8.
4C00  .byte FE 0F 20 03 11 08 00 CD D5 30 5E 16 00 3A D5 D2 ; .. ......0^..:..
4C10  .byte 87 6F 62 01 00 A2 09 7E 23 66 6F 19 09 7E FE 1D ; .ob....~#fo..~..
4C20  .byte 30 18 87 6F 62 11 E1 5B 19 7E 23 66 6F 11 3A 4C ; 0..ob..[.~#fo.:L
4C30  .byte 3E 02 32 FF FF 32 30 D2 D5 E9 2A 02 D4 11 24 00 ; >.2..20...*...$.
4C40  .byte 19 EB 2A 73 D2 01 C0 00 09 AF ED 52 DC F4 2F 21 ; ..*s.......R../!
4C50  .byte 00 00 FD 7E 03 FE FF 20 12 ED 5B 04 D4 7B B2 20 ; ...~... ..[..{. 
4C60  .byte 0A 3A 15 D4 07 30 04 2A 94 D2 23 22 94 D2 FD CB ; .:...0.*..#"....
4C70  .byte 06 7E C4 5C 53 DD 36 14 05 2A 94 D2 11 68 01 A7 ; .~.\S.6..*...h..
4C80  .byte ED 52 D4 79 53 FD 7E 03 FE FE F5 CC 3A 51 F1 C4 ; .R.yS.~.....:Q..
4C90  .byte 30 52 DD CB 18 46 C2 C7 55 DD 7E 0E FE 20 28 11 ; 0R...F..U.~.. (.
4CA0  .byte DD CB 18 56 C2 B1 4C 2A 02 D4 11 F5 FF 19 22 02 ; ...V..L*......".
4CB0  .byte D4 DD 36 0D 10 DD 36 0E 20 2A 04 D4 DD 46 09 0E ; ..6...6. *...F..
4CC0  .byte 00 59 51 FD CB 03 5E CA 5E 51 FD CB 03 56 CA B9 ; .YQ...^.^Q...V..
4CD0  .byte 51 7C B5 B0 28 5C DD 36 14 01 CB 78 20 30 ED 5B ; Q|..(\.6...x 0.[
4CE0  .byte 13 D2 7B 2F 5F 7A 2F 57 13 0E FF E5 D5 ED 5B 3A ; ..{/_z/W......[:
4CF0  .byte D2 AF ED 52 D1 E1 38 3A ED 5B 0F D2 7B 2F 5F 7A ; ...R..8:.[..{/_z
4D00  .byte 2F 57 13 0E FF 3A 17 D2 DD 77 14 C3 32 4D ED 5B ; /W...:...w..2M.[
4D10  .byte 13 D2 0E 00 E5 D5 7D 2F 6F 7C 2F 67 23 ED 5B 3A ; ......}/o|/g#.[:
4D20  .byte D2 AF ED 52 D1 E1 38 0A ED 5B 0F D2 3A 17 D2 DD ; ...R..8..[..:...
4D30  .byte 77 14 78 A7 FA 4F 4D 19 89 4F F2 59 4D 3A 04 D4 ; w.x..OM..O.YM:..
4D40  .byte DD B6 08 DD B6 09 28 11 0E 00 69 61 C3 59 4D 19 ; ......(...ia.YM.
4D50  .byte 89 4F FA 59 4D 0E 00 69 61 79 22 04 D4 32 06 D4 ; .O.YM..iay"..2..
4D60  .byte 2A 07 D4 DD 46 0C 0E 00 59 51 DD CB 18 7E C4 12 ; *...F...YQ...~..
4D70  .byte 53 DD CB 18 46 C2 A0 56 3A 88 D2 A7 20 12 DD CB ; S...F..V:... ...
4D80  .byte 18 7E 28 30 DD CB 18 5E 20 06 FD CB 03 6E 28 24 ; .~(0...^ ....n($
4D90  .byte FD CB 03 6E 20 25 3A 88 D2 A7 CC 00 53 2A 3C D2 ; ...n %:.....S*<.
4DA0  .byte 06 FF 0E 00 59 51 3A 88 D2 3D 32 88 D2 DD CB 18 ; ....YQ:..=2.....
4DB0  .byte D6 C3 D5 4D DD CB 18 9E C3 BF 4D DD CB 18 DE AF ; ...M......M.....
4DC0  .byte 32 88 D2 CB 7C 20 08 3A 16 D2 BC 28 08 38 06 ED ; 2...| .:...(.8..
4DD0  .byte 5B 3E D2 0E 00 FD CB 06 46 28 12 E5 7B 2F 5F 7A ; [>......F(..{/_z
4DE0  .byte 2F 57 79 2F 21 01 00 19 EB CE 00 4F E1 19 78 89 ; /Wy/!......O..x.
4DF0  .byte 22 07 D4 32 09 D4 E5 7B 2F 6F 7A 2F 67 79 2F 11 ; "..2...{/oz/gy/.
4E00  .byte 01 00 19 CE 00 22 E7 D2 32 E9 D2 E1 DD CB 18 56 ; ....."..2......V
4E10  .byte C4 1D 55 7C A7 F2 1F 4E 7C 2F 67 7D 2F 6F 23 11 ; ..U|...N|/g}/o#.
4E20  .byte 00 01 EB A7 ED 52 30 17 3A 15 D4 E6 85 20 10 DD ; .....R0.:.... ..
4E30  .byte CB 0C 7E 28 06 DD 36 14 13 18 04 DD 36 14 01 01 ; ..~(..6.....6...
4E40  .byte 08 00 11 08 00 CD D5 30 7E E6 7F FE 79 D4 00 50 ; .......0~...y..P
4E50  .byte 3A 86 D2 A7 C4 34 54 FD CB 06 76 C4 3D 54 FD CB ; :....4T...v.=T..
4E60  .byte 08 56 C4 5E 54 3A 11 D4 FE 0A CC 74 54 DD 6E 14 ; .V.^T:.....tT.n.
4E70  .byte 4D 26 00 29 11 5B 5C 19 5E 23 56 ED 53 0E D4 3A ; M&.).[\.^#V.S..:
4E80  .byte E0 D2 91 C4 9E 54 3A 10 D4 26 00 6F 19 7E A7 F2 ; .....T:..&.o.~..
4E90  .byte 9A 4E 23 7E 32 10 D4 C3 89 4E 57 01 00 40 DD CB ; .N#~2....NW..@..
4EA0  .byte 18 4E 28 03 01 00 58 FD CB 06 6E C4 87 54 3A FD ; .N(...X...n..T:.
4EB0  .byte D2 A7 C4 59 50 7A 0F 0F 5F E6 C0 6F 7B AD 67 5D ; ...YPz.._..o{.g]
4EC0  .byte 54 29 19 09 22 89 D2 3E 10 32 8D D2 21 1B 5C FD ; T).."..>.2..!.\.
4ED0  .byte CB 06 46 C4 99 54 3A FD D2 A7 C4 5F 50 22 0C D4 ; ..F..T:...._P"..
4EE0  .byte 0E 0A 3A 05 D4 A7 F2 ED 4E ED 44 0E F6 FE 0A 38 ; ..:.....N.D....8
4EF0  .byte 04 79 32 05 D4 0E 0C 3A 08 D4 A7 F2 02 4F ED 44 ; .y2....:.....O.D
4F00  .byte 0E F4 FE 0C 38 04 79 32 08 D4 FD CB 06 7E C4 A3 ; ....8.y2.....~..
4F10  .byte 54 FD CB 08 46 C4 9F 50 3A E2 D2 A7 C4 B0 54 3A ; T...F..P:.....T:
4F20  .byte 22 D3 A7 C4 63 50 DD CB 18 56 C4 FB 4F FD CB 06 ; "...cP...V..O...
4F30  .byte 4E 20 5A 2A 6D D2 01 30 00 09 EB 2A FF D3 A7 ED ; N Z*m..0...*....
4F40  .byte 52 30 18 ED 53 FF D3 3A 06 D4 A7 F2 8D 4F AF 32 ; R0..S..:.....O.2
4F50  .byte 04 D4 32 05 D4 32 06 D4 C3 8D 4F 2A 6F D2 11 D0 ; ..2..2....O*o...
4F60  .byte 00 19 EB 2A FF D3 0E 10 09 A7 ED 52 38 1F EB 37 ; ...*.......R8..7
4F70  .byte ED 42 22 FF D3 3A 06 D4 A7 FA 8D 4F 2A 05 D4 B4 ; .B"..:.....O*...
4F80  .byte B5 28 0A AF 32 04 D4 32 05 D4 32 06 D4 3A 15 D4 ; .(..2..2..2..:..
4F90  .byte 32 BA D2 3A 11 D4 32 E0 D2 16 01 0E 30 FE 01 28 ; 2..:..2.....0..(
4FA0  .byte 0C 16 04 0E 46 FE 09 28 04 DD 34 13 C9 3A E1 D2 ; ....F..(..4..:..
4FB0  .byte 47 2A 04 D4 CB 7C 28 07 7D 2F 6F 7C 2F 67 23 CB ; G*...|(.}/o|/g#.
4FC0  .byte 3C CB 1D 7D 80 32 E1 D2 7C 8A DD 8E 13 32 10 D4 ; <..}.2..|....2..
4FD0  .byte B9 D8 91 32 10 D4 C9 10 00 30 00 08 00 00 08 02 ; ...2.....0......
4FE0  .byte 10 00 30 00 02 00 00 08 02 04 00 0C 00 02 00 00 ; ..0.............
4FF0  .byte 02 01 10 00 30 00 08 00 00 08 02 DD 36 0E 19 C9 ; ....0.......6...
5000  .byte EB 2A 02 D4 ED 4B 57 D2 A7 ED 42 D8 01 10 00 A7 ; .*...KW...B.....
5010  .byte ED 42 D8 2A FF D3 01 08 00 09 1A 4F 7D 0F 0F 0F ; .B.*.......O}...
5020  .byte 0F E6 01 3C 47 79 A0 C8 7D E6 F0 6F 22 AB D2 22 ; ...<Gy..}..o".."
5030  .byte 1E D3 79 A8 12 2A 02 D4 01 08 00 09 7D E6 E0 C6 ; ..y..*......}...
5040  .byte 08 6F 22 AD D2 22 20 D3 3E 06 32 22 D3 21 53 5C ; .o".." .>.2".!S\
5050  .byte 22 AF D2 3E 01 CD 7E 33 C9 3D 57 01 00 70 C9 21 ; "..>..~3.=W..p.!
5060  .byte 00 00 C9 3D 32 22 D3 2A 1E D3 22 0F D2 2A 20 D3 ; ...=2".*.."..* .
5070  .byte 22 11 D2 21 00 00 22 13 D2 21 FE FF 22 15 D2 FE ; "..!.."..!.."...
5080  .byte 03 38 11 3E B2 CD 5D 2F 21 08 00 22 13 D2 21 02 ; .8.>..]/!.."..!.
5090  .byte 00 22 15 D2 3E 5A CD 5D 2F C9 FD CB 08 CE C9 2A ; ."..>Z.]/......*
50A0  .byte FF D3 ED 4B 05 D4 11 05 00 19 09 22 0F D2 2A 02 ; ...K......."..*.
50B0  .byte D4 ED 4B 08 D4 11 09 00 19 09 22 11 D2 21 F4 D2 ; ..K......."..!..
50C0  .byte 3E 94 CD CE 50 21 F5 D2 3E 96 CD CE 50 C9 E5 F5 ; >...P!..>...P...
50D0  .byte 5E 16 00 21 0A 51 19 5E CB 7B 28 01 15 ED 53 13 ; ^..!.Q.^.{(...S.
50E0  .byte D2 23 16 00 5E 21 00 00 CB 7B 28 01 15 3A 15 D4 ; .#..^!...{(..:..
50F0  .byte E6 05 28 03 21 FC FF 19 22 15 D2 F1 CD 5D 2F E1 ; ..(.!..."....]/.
5100  .byte 7E C6 02 FE 30 38 01 AF 77 C9 10 00 F0 00 0F 04 ; ~...08..w.......
5110  .byte F1 FC 0E 08 F2 F8 0B 0B F5 F5 08 0E F8 F2 04 0F ; ................
5120  .byte FC F1 00 10 00 F0 FC 0F 04 F1 F8 0E 08 F2 F5 0B ; ................
5130  .byte 0B F5 F2 08 0E F8 F1 04 0F FC 2A 04 D4 7C B5 C0 ; ..........*..|..
5140  .byte 3A 15 D4 07 D0 DD 36 14 0C ED 5B B8 D2 CB 7A 20 ; :.....6...[...z 
5150  .byte 07 21 30 00 A7 ED 52 D8 13 ED 53 B8 D2 C9 DD CB ; .!0...R...S.....
5160  .byte 18 8E CB 78 20 28 ED 5B 0F D2 0E 00 DD 36 14 01 ; ...x (.[.....6..
5170  .byte E5 D9 E1 ED 5B 3A D2 AF ED 52 D9 DA 32 4D 47 5F ; ....[:...R..2MG_
5180  .byte 57 4F 2A 3A D2 3A 17 D2 DD 77 14 C3 32 4D DD CB ; WO*:.:...w..2M..
5190  .byte 18 CE DD 36 14 0A E5 7D 2F 6F 7C 2F 67 23 11 00 ; ...6...}/o|/g#..
51A0  .byte 01 A7 ED 52 E1 ED 5B 11 D2 0E 00 D2 32 4D DD CB ; ...R..[.....2M..
51B0  .byte 18 8E DD 36 14 01 C3 32 4D DD CB 18 CE 7D B4 28 ; ...6...2M....}.(
51C0  .byte 04 CB 78 28 3E ED 5B 0F D2 7B 2F 5F 7A 2F 57 13 ; ..x(>.[..{/_z/W.
51D0  .byte 0E FF DD 36 14 01 E5 D9 E1 7D 2F 6F 7C 2F 67 23 ; ...6.....}/o|/g#
51E0  .byte ED 5B 3A D2 AF ED 52 D9 DA 32 4D 5F 57 4F 2A 3A ; .[:...R..2M_WO*:
51F0  .byte D2 7D 2F 6F 7C 2F 67 23 06 FF 3A 17 D2 DD 77 14 ; .}/o|/g#..:...w.
5200  .byte C3 32 4D DD CB 18 8E DD 36 14 0A ED 5B 11 D2 7B ; .2M.....6...[..{
5210  .byte 2F 5F 7A 2F 57 13 0E FF E5 D9 E1 01 00 01 A7 ED ; /_z/W...........
5220  .byte 42 D9 D2 32 4D DD CB 18 CE DD 36 14 01 C3 32 4D ; B..2M.....6...2M
5230  .byte DD CB 18 46 C0 2A B8 D2 7C B5 C8 CB 7C 28 05 23 ; ...F.*..|...|(.#
5240  .byte 22 B8 D2 C9 2B 22 B8 D2 C9 DD 35 15 C9 3D 32 13 ; "...+"....5..=2.
5250  .byte D4 C9 3A 24 D2 E6 03 C0 21 87 D2 35 C0 FD CB 08 ; ..:$....!..5....
5260  .byte 86 3A D3 D2 FE 09 C8 3A F7 D2 DF C9 3A D5 D2 FE ; .:.....:....:...
5270  .byte 03 C0 3A 38 D2 FE 0B C8 2A 96 D2 23 22 96 D2 11 ; ..:8....*..#"...
5280  .byte 00 03 A7 ED 52 D8 3E 05 94 30 29 FD CB 06 AE FD ; ....R.>..0).....
5290  .byte CB 06 B6 FD CB 08 86 FD CB 08 DE FD CB 05 C6 3E ; ...............>
52A0  .byte C0 32 81 D2 3E 0A DF CD F5 91 CD F5 91 CD F5 91 ; .2..>...........
52B0  .byte CD F5 91 AF 5F 87 C6 80 32 BF D2 3E FF 32 C0 D2 ; ...._...2..>.2..
52C0  .byte 16 00 21 FA 52 19 3A 24 D2 A6 20 03 3E 1A EF 3A ; ..!.R.:$.. .>..:
52D0  .byte 24 D2 0F D0 2A FF D3 ED 5B 54 D2 A7 ED 52 7D C6 ; $...*...[T...R}.
52E0  .byte 04 4F 2A 02 D4 ED 5B 57 D2 A7 ED 52 7D C6 EC 47 ; .O*...[W...R}..G
52F0  .byte 21 3C D0 11 BF D2 CD A8 2F C9 01 07 0F 1F 3F 7F ; !<....../.....?.
5300  .byte 3E 10 32 88 D2 3E 00 EF C9 AF 32 FE D3 ED 53 FF ; >.2..>....2...S.
5310  .byte D3 C9 D9 2A 02 D4 22 DA D2 D9 DD CB 18 56 C8 DD ; ...*.."......V..
5320  .byte CB 18 96 FD CB 07 46 C0 D9 2A 02 D4 11 F5 FF 19 ; ......F..*......
5330  .byte 22 02 D4 D9 C9 DD CB 18 56 C0 DD CB 18 46 C0 DD ; ".......V....F..
5340  .byte CB 18 7E C8 DD CB 18 C6 2A 04 D4 7D B4 28 03 3E ; ..~.....*..}.(.>
5350  .byte 06 EF FD CB 07 D6 C9 FD CB 07 96 C9 2A DD D2 ED ; ............*...
5360  .byte 5B 02 D4 A7 ED 52 DA 45 58 21 00 00 22 96 D2 DD ; [....R.EX!.."...
5370  .byte CB 18 A6 C9 DD CB 18 D6 C9 DD 36 14 0D C9 FD 36 ; ..........6....6
5380  .byte 03 FF 3A 15 D4 E6 FA 32 15 D4 C9 3D 32 84 D2 28 ; ..:....2...=2..(
5390  .byte 25 FE 14 38 16 AF 6F 67 32 04 D4 22 05 D4 32 07 ; %..8..og2.."..2.
53A0  .byte D4 22 08 D4 DD 36 14 0F C3 50 4E DD CB 18 8E DD ; ."...6...PN.....
53B0  .byte 36 14 0E C3 50 4E 2A D6 D2 46 23 4E 23 7E A7 28 ; 6...PN*..F#N#~.(
53C0  .byte 21 FA CD 53 32 D4 D2 FD CB 06 E6 18 04 FD CB 0D ; !..S2...........
53D0  .byte D6 3E 01 32 83 D2 21 00 00 22 FF D3 22 02 D4 C3 ; .>.2..!..".."...
53E0  .byte 50 4E 78 26 00 06 05 87 CB 14 10 FB 6F 11 08 00 ; PNx&........o...
53F0  .byte 19 22 FF D3 79 26 00 87 CB 14 87 CB 14 87 CB 14 ; ."..y&..........
5400  .byte 87 CB 14 87 CB 14 6F 22 02 D4 AF 32 FE D3 32 01 ; ......o"...2..2.
5410  .byte D4 C3 50 4E AF 6F 67 22 07 D4 32 09 D4 DD 36 14 ; ..PN.og"..2...6.
5420  .byte 16 3A 10 D4 FE 12 DA 50 4E FD CB 08 B6 DD CB 18 ; .:.....PN.......
5430  .byte D6 C3 50 4E 3D 32 86 D2 DD 36 14 11 C9 DD 36 0D ; ..PN=2...6....6.
5440  .byte 14 DD 36 14 10 DD CB 0C 7E C0 DD CB 18 7E C8 FD ; ..6.....~....~..
5450  .byte CB 06 B6 AF 32 04 D4 32 05 D4 32 06 D4 C9 3A 15 ; ....2..2..2...:.
5460  .byte D4 E6 FA 32 15 D4 DD 36 14 14 21 F6 D2 35 C0 FD ; ...2...6..!..5..
5470  .byte CB 08 96 C9 3A 13 D4 A7 C0 DD CB 18 7E C8 3E 03 ; ....:.......~.>.
5480  .byte EF 3E 3C 32 13 D4 C9 3A 24 D2 0F D8 01 00 58 16 ; .><2...:$.....X.
5490  .byte 23 3A 15 D4 E6 05 C8 14 C9 11 0E 00 19 C9 DD 36 ; #:.............6
54A0  .byte 13 00 C9 DD CB 18 66 C8 3A 24 D2 A7 CC F5 91 C9 ; ......f.:$......
54B0  .byte 3D 32 E2 D2 FD 4E 0A 2A 36 D2 C5 E5 21 00 D0 22 ; =2...N.*6...!.."
54C0  .byte 36 D2 ED 5B 57 D2 2A E5 D2 22 11 D2 A7 ED 52 EB ; 6..[W.*.."....R.
54D0  .byte ED 4B 54 D2 2A E3 D2 22 0F D2 A7 ED 42 FE 06 38 ; .KT.*.."....B..8
54E0  .byte 04 FE 0A 38 08 F5 01 0B 55 CD 07 2F F1 21 0C 00 ; ...8....U../.!..
54F0  .byte 22 13 D2 4F 06 00 21 F0 FF 09 22 15 D2 3E 50 CD ; "..O..!..."..>P.
5500  .byte 5D 2F E1 C1 22 36 D2 FD 71 0A C9 00 02 04 06 FF ; ]/.."6..q.......
5510  .byte FF 20 22 24 26 FF FF FF FF FF FF FF FF DD 36 14 ; . "$&.........6.
5520  .byte 09 C9 3D 32 85 D2 C0 3A F7 D2 DF FD 4E 0A FD CB ; ..=2...:....N...
5530  .byte 00 86 CD 27 03 FD 71 0A C9 FD 36 03 FB 2A FF D3 ; ...'..q...6..*..
5540  .byte 11 60 1B A7 ED 52 D0 FD 36 03 FF 2A 04 D4 7D B4 ; .`...R..6..*..}.
5550  .byte C0 DD CB 18 8E E1 DD CB 18 CE DD 36 14 18 21 F9 ; ...........6..!.
5560  .byte D2 FD CB 0D 46 20 3D 36 50 CD AF 7C DA 50 4E DD ; ....F =6P..|.PN.
5570  .byte E5 E5 DD E1 AF DD 36 00 54 DD 77 11 DD 77 18 DD ; ......6.T.w..w..
5580  .byte 77 01 2A FF D3 DD 75 02 DD 74 03 DD 77 04 2A 02 ; w.*...u..t..w.*.
5590  .byte D4 11 0E 00 19 DD 75 05 DD 74 06 DD E1 FD CB 0D ; ......u..t......
55A0  .byte C6 C3 50 4E FD CB 0D 4E 20 0A 35 C2 50 4E FD CB ; ..PN...N .5.PN..
55B0  .byte 0D CE 36 8C DD 36 14 17 7E A7 28 04 35 C3 50 4E ; ..6..6..~.(.5.PN
55C0  .byte DD 36 14 19 C3 50 4E DD 7E 0E FE 15 28 0A 2A 02 ; .6...PN.~...(.*.
55D0  .byte D4 11 0B 00 19 22 02 D4 DD 36 0D 10 DD 36 0E 15 ; ....."...6...6..
55E0  .byte 2A 04 D4 DD 46 09 0E 00 59 51 7C B5 B0 CA 52 56 ; *...F...YQ|...RV
55F0  .byte DD 36 14 09 FD CB 03 56 20 20 FD CB 03 4E 28 1A ; .6.....V  ...N(.
5600  .byte DD CB 18 7E CA 12 56 CB 78 20 35 DD CB 18 86 C3 ; ...~..V.x 5.....
5610  .byte 03 52 11 F0 FF 0E FF C3 32 4D FD CB 03 5E 20 20 ; .R......2M...^  
5620  .byte FD CB 03 4E 28 1A DD CB 18 7E CA 38 56 CB 78 28 ; ...N(....~.8V.x(
5630  .byte 0F DD CB 18 86 C3 03 52 11 10 00 0E 00 C3 32 4D ; .......R......2M
5640  .byte 11 04 00 0E 00 78 A7 FA 32 4D 11 FC FF 0E FF C3 ; .....x..2M......
5650  .byte 32 4D DD CB 18 7E 28 21 DD 36 14 07 DD CB 18 86 ; 2M...~(!.6......
5660  .byte ED 5B B8 D2 CB 7A 28 09 21 D8 FF A7 ED 52 D2 60 ; .[...z(.!....R.`
5670  .byte 4D 1B ED 53 B8 D2 C3 60 4D DD 36 14 09 D5 E5 CB ; M..S...`M.6.....
5680  .byte 78 28 07 7D 2F 6F 7C 2F 67 23 ED 5B 3A D2 AF ED ; x(.}/o|/g#.[:...
5690  .byte 52 E1 D1 DA 32 4D 4F 59 51 DD 36 14 09 C3 32 4D ; R...2MOYQ.6...2M
56A0  .byte DD CB 18 7E 28 21 DD CB 18 5E 20 06 FD CB 03 6E ; ...~(!...^ ....n
56B0  .byte 28 15 FD CB 03 6E 20 16 DD CB 18 86 3A 04 D4 E6 ; (....n .....:...
56C0  .byte F8 32 04 D4 C3 96 4D DD CB 18 9E C3 C3 4D DD CB ; .2....M......M..
56D0  .byte 18 DE C3 C3 4D DD CB 18 EE DD CB 18 CE 3A 81 D2 ; ....M........:..
56E0  .byte FE 60 28 63 2A 57 D2 11 C0 00 19 ED 5B 02 D4 ED ; .`(c*W......[...
56F0  .byte 52 30 16 FD CB 06 56 20 10 3E 01 32 7D D2 21 40 ; R0....V .>.2}.!@
5700  .byte D2 35 FD CB 06 D6 C3 47 57 AF 21 80 00 FD CB 08 ; .5.....GW.!.....
5710  .byte 5E 20 25 ED 5B 07 D4 CB 7A 20 08 21 00 06 A7 ED ; ^ %.[...z .!....
5720  .byte 52 38 1B EB DD 46 0C 7C FE 80 30 04 FE 08 30 05 ; R8...F.|..0...0.
5730  .byte 11 20 00 0E 00 19 78 89 22 07 D4 32 09 D4 AF 6F ; . ....x."..2...o
5740  .byte 67 22 04 D4 32 06 D4 DD 36 14 0B FD CB 08 5E CA ; g"..2...6.....^.
5750  .byte 50 4E DD 36 14 15 C3 50 4E FD CB 06 7E C0 DD CB ; PN.6...PN...~...
5760  .byte 18 A6 C9 FD CB 05 46 CA D9 2F C9 DD 7E 02 C6 08 ; ......F../..~...
5770  .byte E6 1F FE 1A D8 3A 15 D4 0F 38 03 E6 02 C8 DD 6E ; .....:...8.....n
5780  .byte 07 DD 66 08 DD CB 09 7E C0 11 01 03 A7 ED 52 D8 ; ..f....~......R.
5790  .byte DD 6E 08 DD 66 09 29 7D 2F 6F 7C 2F 67 23 DD 36 ; .n..f.)}/o|/g#.6
57A0  .byte 0A 00 DD 75 0B DD 74 0C 3E 05 EF C9 DD 7E 02 C6 ; ...u..t.>....~..
57B0  .byte 08 E6 1F FE 10 D8 DD 36 07 00 DD 36 08 F8 DD 36 ; .......6...6...6
57C0  .byte 09 FF DD CB 18 CE 3E 04 EF C9 DD 7E 02 C6 08 E6 ; ......>....~....
57D0  .byte 1F FE 10 D8 DD CB 18 7E C8 3A BA D2 E6 80 C0 FD ; .......~.:......
57E0  .byte CB 06 B6 DD 36 0A 00 DD 36 0B F4 DD 36 0C FF 3E ; ....6...6...6..>
57F0  .byte 04 EF C9 DD 7E 02 C6 08 E6 1F FE 10 D0 FD CB 06 ; ....~...........
5800  .byte B6 DD 36 07 00 DD 36 08 08 DD 36 09 00 DD CB 18 ; ..6...6...6.....
5810  .byte 8E 3E 04 EF C9 DD CB 18 7E C8 2A FE D3 3A 00 D4 ; .>......~.*..:..
5820  .byte 11 80 FE 19 CE FF 22 FE D3 32 00 D4 C9 DD CB 18 ; ......"..2......
5830  .byte 7E C8 2A FE D3 3A 00 D4 11 00 02 19 CE 00 22 FE ; ~.*..:........".
5840  .byte D3 32 00 D4 C9 DD CB 18 66 20 03 3E 12 EF DD CB ; .2......f .>....
5850  .byte 18 E6 C9 DD 7E 02 C6 08 E6 1F FE 08 D8 FE 18 D0 ; ....~...........
5860  .byte DD CB 18 7E C8 3A BA D2 E6 80 C0 FD CB 06 B6 DD ; ...~.:..........
5870  .byte 36 0A 00 DD 36 0B F4 DD 36 0C FF 3E 04 EF C9 DD ; 6...6...6..>....
5880  .byte CB 0C 7E C0 3E 05 EF C9 FD CB 06 66 C0 3A FF D3 ; ..~.>......f.:..
5890  .byte C6 08 E6 1F FE 08 D8 FE 18 D0 2A FF D3 01 08 00 ; ..........*.....
58A0  .byte 09 7D 87 CB 14 87 CB 14 87 CB 14 5C 2A 02 D4 01 ; .}.........\*...
58B0  .byte 10 00 09 7D 87 CB 14 87 CB 14 87 CB 14 54 21 E0 ; ...}.........T!.
58C0  .byte 58 06 05 7E 23 BB 20 11 7E BA 20 0D 23 22 D6 D2 ; X..~#. .~. .#"..
58D0  .byte 3E 50 32 84 D2 3E 06 EF C9 23 23 23 23 10 E4 C9 ; >P2..>...####...
58E0  .byte 34 3D 34 2F 00 18 3A 19 03 00 0E 3A 00 00 16 1B ; 4=4/..:....:....
58F0  .byte 32 00 00 17 2F 0C 00 00 FF 2A 04 D4 3A 06 D4 11 ; 2.../....*..:...
5900  .byte F8 FF 19 CE FF 22 04 D4 32 06 D4 DD CB 18 66 20 ; ....."..2.....f 
5910  .byte 03 3E 12 EF DD CB 18 E6 C9 AF 21 05 00 32 04 D4 ; .>........!..2..
5920  .byte 22 05 D4 DD CB 18 8E 3E 06 32 86 D2 FD 7E 03 F6 ; "......>.2...~..
5930  .byte 0F FD 77 03 21 04 00 22 08 D4 DD CB 18 86 DD CB ; ..w.!.."........
5940  .byte 18 96 C9 AF 21 06 00 32 04 D4 22 05 D4 DD CB 18 ; ....!..2..".....
5950  .byte 8E 18 D4 AF 21 FB FF 32 04 D4 22 05 D4 DD CB 18 ; ....!..2..".....
5960  .byte CE 18 C4 AF 21 FA FF 32 04 D4 22 05 D4 DD CB 18 ; ....!..2..".....
5970  .byte CE 18 B4 3A E2 D2 FE 08 D0 CD CC 59 11 01 00 2A ; ...:.......Y...*
5980  .byte 07 D4 7D 2F 6F 7C 2F 67 3A 09 D4 2F 19 CE 00 A7 ; ..}/o|/g:../....
5990  .byte F2 99 59 11 C8 FF 19 CE FF 22 07 D4 32 09 D4 01 ; ..Y......"..2...
59A0  .byte 08 00 2A FF D3 09 7D E6 E0 6F 22 E3 D2 01 10 00 ; ..*...}..o".....
59B0  .byte 2A 02 D4 09 7D E6 E0 6F 22 E5 D2 3E 10 32 E2 D2 ; *...}..o"..>.2..
59C0  .byte 11 10 00 0E 00 CD AA 33 3E 07 EF C9 2A 04 D4 3A ; .......3>...*..:
59D0  .byte 06 D4 4F E6 80 47 3A FF D3 C6 08 E6 1F D6 10 E6 ; ..O..G:.........
59E0  .byte 80 B8 28 09 7D 2F 6F 7C 2F 67 79 2F 4F 11 01 00 ; ..(.}/o|/gy/O...
59F0  .byte 79 19 CE 00 5D 54 4F CB 29 CB 1A CB 1B 19 89 22 ; y...]TO.)......"
5A00  .byte 04 D4 32 06 D4 C9 DD 36 0A 00 DD 36 0B F6 DD 36 ; ..2....6...6...6
5A10  .byte 0C FF 3E 04 EF C9 DD 36 0A 00 DD 36 0B F4 DD 36 ; ..>....6...6...6
5A20  .byte 0C FF 3E 04 EF C9 DD 36 0A 00 DD 36 0B F2 DD 36 ; ..>....6...6...6
5A30  .byte 0C FF 3E 04 EF C9 3A B1 D2 A7 C0 CD 78 5A 11 01 ; ..>...:.....xZ..
5A40  .byte 00 2A 04 D4 7D 2F 6F 7C 2F 67 3A 06 D4 2F 19 CE ; .*..}/o|/g:../..
5A50  .byte 00 11 00 FF 0E FF FA 5E 5A 11 00 01 0E 00 19 89 ; .......^Z.......
5A60  .byte 22 04 D4 32 06 D4 21 B1 D2 36 04 23 36 0E 23 36 ; "..2..!..6.#6.#6
5A70  .byte FF 23 36 0F 3E 07 EF C9 3A 06 D4 A7 11 F0 FF F2 ; .#6.>...:.......
5A80  .byte 85 5A 11 20 00 2A FF D3 01 08 00 09 7D E6 E0 6F ; .Z. .*......}..o
5A90  .byte 19 22 FF D3 C9 3A B1 D2 A7 C0 CD 78 5A CD CC 59 ; ."...:.....xZ..Y
5AA0  .byte 11 01 00 2A 07 D4 7D 2F 6F 7C 2F 67 3A 09 D4 2F ; ...*..}/o|/g:../
5AB0  .byte 19 CE 00 A7 F2 BD 5A 11 C8 FF 19 CE FF 22 07 D4 ; ......Z......"..
5AC0  .byte 32 09 D4 C3 66 5A 2A EA D2 11 82 00 A7 ED 52 D8 ; 2...fZ*.......R.
5AD0  .byte FD CB 05 46 CA D9 2F C9 3A 15 D4 07 D0 2A FF D3 ; ...F../.:....*..
5AE0  .byte 01 08 00 09 7D E6 1F FE 10 30 42 2A FF D3 01 08 ; ....}....0B*....
5AF0  .byte 00 09 7D E6 E0 4F 44 2A 02 D4 11 10 00 19 7D E6 ; ..}..OD*......}.
5B00  .byte E0 5F 54 CD 6D 5B D8 01 08 00 11 10 00 CD D5 30 ; ._T.m[.........0
5B10  .byte 0E 00 7E FE 8A 28 02 0E 89 71 C9 3A 15 D4 07 D0 ; ..~..(...q.:....
5B20  .byte 2A FF D3 01 08 00 09 7D E6 1F FE 10 D8 7D E6 E0 ; *......}.....}..
5B30  .byte C6 10 4F 44 2A 02 D4 11 10 00 19 7D E6 E0 5F 54 ; ..OD*......}.._T
5B40  .byte CD 6D 5B D8 01 08 00 11 10 00 CD D5 30 0E 00 7E ; .m[.........0..~
5B50  .byte FE 89 28 C5 0E 8A 71 C9 3A 15 D4 07 D0 2A FF D3 ; ..(...q.:....*..
5B60  .byte 01 08 00 09 7D E6 1F FE 10 D0 C3 EB 5A C5 D5 CD ; ....}.......Z...
5B70  .byte AF 7C D1 C1 D8 DD E5 E5 DD E1 AF DD 36 00 2E DD ; .|..........6...
5B80  .byte 77 01 DD 71 02 DD 70 03 DD 77 04 DD 73 05 DD 72 ; w..q..p..w..s..r
5B90  .byte 06 DD 77 07 DD 77 08 DD 77 09 DD 77 0A DD 77 0B ; ..w..w..w..w..w.
5BA0  .byte DD 77 0C DD 77 18 DD E1 A7 C9 DD CB 18 7E C8 2A ; .w..w........~.*
5BB0  .byte 02 D4 ED 5B 57 D2 A7 ED 52 D0 FD 36 03 FF C9 2A ; ...[W...R..6...*
5BC0  .byte EA D2 11 82 00 A7 ED 52 D8 2A FF D3 11 08 00 19 ; .......R.*......
5BD0  .byte 7D E6 1F FE 06 D8 FE 1A D0 FD CB 05 46 CA D9 2F ; }...........F../
5BE0  .byte C9 59 57 63 57 6B 57 AC 57 CA 57 F3 57 15 58 2D ; .YWcWkW.W.W.W.X-
5BF0  .byte 58 45 58 53 58 7F 58 88 58 F9 58 19 59 43 59 53 ; XEXSX.X.X.X.YCYS
5C00  .byte 59 63 59 73 59 06 5A 16 5A 26 5A 36 5A 95 5A C6 ; YcYsY.Z.Z&Z6Z.Z.
5C10  .byte 5A D8 5A 1B 5B 58 5B AA 5B BF 5B B4 B6 FF FF FF ; Z.Z.[X[.[.[.....
5C20  .byte FF B8 BA FF FF FF FF FF FF B6 B8 FF FF FF FF BA ; ................
5C30  .byte B8 FF FF FF FF FF FF B4 B6 B8 FF FF FF BA BC BE ; ................
5C40  .byte FF FF FF FF FF B8 B6 B4 FF FF FF BE BC BA FF FF ; ................
5C50  .byte FF FF FF 00 00 00 00 00 00 00 00 8F 5C 8F 5C C1 ; ............\.\.
5C60  .byte 5C D3 5C D5 5C D8 5C DB 5C DD 5C E0 5C E3 5C 2B ; \.\.\.\.\.\.\.\+
5C70  .byte 5D 39 5D 3C 5D 3F 5D 83 5D 99 5D A5 5D A8 5D B6 ; ]9]<]?].].].].].
5C80  .byte 5D B9 5D BC 5D BF 5D C2 5D D8 5D DB 5D DE 5D 04 ; ].].].].].].].].
5C90  .byte 04 04 04 04 04 04 04 05 05 05 05 05 05 05 05 06 ; ................
5CA0  .byte 06 06 06 06 06 06 06 07 07 07 07 07 07 07 07 08 ; ................
5CB0  .byte 08 08 08 08 08 08 08 09 09 09 09 09 09 09 09 FF ; ................
5CC0  .byte 00 0B 0B 0B 0B 0C 0C 0C 0C 0D 0D 0D 0D 0E 0E 0E ; ................
5CD0  .byte 0E FF 00 FF 00 00 FF 00 00 FF 00 FF 00 0A FF 00 ; ................
5CE0  .byte 00 FF 00 13 13 13 13 13 13 13 13 13 13 13 13 13 ; ................
5CF0  .byte 13 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 0F 10 ; ................
5D00  .byte 10 10 10 10 10 10 10 10 10 10 10 10 10 11 11 11 ; ................
5D10  .byte 11 11 11 11 11 11 11 11 11 11 11 12 12 12 12 12 ; ................
5D20  .byte 12 12 12 12 12 12 12 12 12 FF 00 19 19 19 19 19 ; ................
5D30  .byte 19 1A 1A 1A 1A 1A 1A FF 00 18 FF 00 16 FF 00 02 ; ................
5D40  .byte 02 02 02 02 02 02 02 02 02 02 02 02 02 02 02 01 ; ................
5D50  .byte 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 ; ................
5D60  .byte 01 02 02 02 02 02 02 02 02 02 02 02 02 02 02 02 ; ................
5D70  .byte 02 03 03 03 03 03 03 03 03 03 03 03 03 03 03 03 ; ................
5D80  .byte 03 FF 22 18 18 18 18 18 18 18 1D 1D 1D 1D 1D 1E ; ..".............
5D90  .byte 1E 1E 1E 1F 1F 1F 1F FF 12 13 13 0F 0F 10 10 11 ; ................
5DA0  .byte 11 12 12 FF 00 14 FF 00 14 14 14 14 14 14 15 15 ; ................
5DB0  .byte 15 15 15 15 FF 00 00 FF 00 17 FF 00 1C FF 00 1B ; ................
5DC0  .byte FF 00 1F 1F 1F 1E 1E 1E 1E 1D 1D 1D 1D 1D 18 18 ; ................
5DD0  .byte 18 18 18 18 18 18 FF 12 1D FF 00 1E FF 00 1F FF ; ................
5DE0  .byte 00 DD 36 0D 14 DD 36 0E 18 CD 89 60 21 03 00 22 ; ..6...6....`!.."
5DF0  .byte 15 D2 CD 28 33 38 12 CD CC 60 38 0D 3E 10 CD 7E ; ...(38...`8.>..~
5E00  .byte 33 AF DD 77 0F DD 77 10 C9 21 00 52 CD A8 0B DD ; 3..w..w..!.R....
5E10  .byte 36 0F 97 DD 36 10 5E 3A 24 D2 E6 07 FE 05 D0 DD ; 6...6.^:$.......
5E20  .byte 36 0F A4 DD 36 10 5E DD 6E 01 DD 66 02 DD 7E 03 ; 6...6.^.n..f..~.
5E30  .byte DD 5E 07 DD 56 08 19 DD 8E 09 6C 67 22 0F D2 DD ; .^..V.....lg"...
5E40  .byte 6E 04 DD 66 05 DD 7E 06 DD CB 18 7E 20 0A DD 5E ; n..f..~....~ ..^
5E50  .byte 0A DD 56 0B 19 DD 8E 0C 6C 67 22 11 D2 21 04 00 ; ..V.....lg"..!..
5E60  .byte 22 13 D2 21 00 00 22 15 D2 3E 5C CD 5D 2F 21 0C ; "..!.."..>\.]/!.
5E70  .byte 00 22 13 D2 3E 5E CD 5D 2F DD CB 18 4E C8 DD 6E ; ."..>^.]/...N..n
5E80  .byte 0A DD 66 0B DD 7E 0C 11 40 00 19 CE 00 DD 75 0A ; ..f..~..@.....u.
5E90  .byte DD 74 0B DD 77 0C C9 54 56 58 FF FF FF AA AC AE ; .t..w..TVX......
5EA0  .byte FF FF FF FF 54 FE 58 FF FF FF AA AC AE FF FF FF ; ....T.X.........
5EB0  .byte FF DD 36 0D 14 DD 36 0E 18 CD 89 60 21 03 00 22 ; ..6...6....`!.."
5EC0  .byte 15 D2 CD 28 33 38 10 CD CC 60 38 0B 3E F0 32 12 ; ...(38...`8.>.2.
5ED0  .byte D4 3E 02 EF C3 01 5E 21 80 52 C3 0C 5E DD 36 0D ; .>....^!.R..^.6.
5EE0  .byte 14 DD 36 0E 18 CD 89 60 21 06 D3 CD 8D 0B 7E A1 ; ..6....`!.....~.
5EF0  .byte 28 07 DD 36 00 FF C3 01 5E 21 03 00 22 15 D2 CD ; (..6....^!.."...
5F00  .byte 28 33 38 2E CD CC 60 38 29 DD CB 18 56 C2 FC 5D ; (38...`8)...V..]
5F10  .byte 21 40 D2 34 21 06 D3 CD 8D 0B 7E B1 77 AF DD 77 ; !@.4!.....~.w..w
5F20  .byte 0F DD 77 10 3E 09 EF 3A 38 D2 FE 1C D0 21 7A D2 ; ..w.>..:8....!z.
5F30  .byte 34 C9 3A 38 D2 FE 04 28 12 FE 09 28 37 FE 0C 28 ; 4.:8...(...(7..(
5F40  .byte 4F FE 11 28 5D 21 00 53 C3 0C 5E 0E 00 11 40 00 ; O..(]!.S..^...@.
5F50  .byte DD 7E 13 FE 3C 38 04 0D 11 C0 FF DD 73 0A DD 72 ; .~..<8......s..r
5F60  .byte 0B DD 71 0C DD 34 13 DD 7E 13 FE 50 38 D7 DD 36 ; ..q..4..~..P8..6
5F70  .byte 13 28 18 D1 DD CB 18 D6 21 18 D3 CD 8D 0B 7E 21 ; .(......!.....~!
5F80  .byte 00 52 A1 CA 0C 5E DD CB 18 96 21 00 53 C3 0C 5E ; .R...^....!.S..^
5F90  .byte DD CB 18 CE DD 36 07 80 DD 36 08 00 DD 36 09 00 ; .....6...6...6..
5FA0  .byte 18 A3 3A 7A D2 FE 11 30 9C DD 36 00 FF 18 96 DD ; ..:z...0..6.....
5FB0  .byte 36 0D 14 DD 36 0E 18 CD 89 60 21 03 00 22 15 D2 ; 6...6....`!.."..
5FC0  .byte CD 28 33 38 0C CD CC 60 38 07 FD CB 06 EE C3 01 ; .(38...`8.......
5FD0  .byte 5E 21 80 53 C3 0C 5E DD 36 0D 14 DD 36 0E 18 CD ; ^!.S..^.6...6...
5FE0  .byte 89 60 21 03 00 22 15 D2 CD 28 33 38 1D CD CC 60 ; .`!.."...(38...`
5FF0  .byte 38 18 FD CB 08 C6 3E F0 32 87 D2 3E 18 32 F4 D2 ; 8.....>.2..>.2..
6000  .byte AF 32 F5 D2 3E 08 DF C3 01 5E 21 00 54 C3 0C 5E ; .2..>....^!.T..^
6010  .byte DD 36 0D 14 DD 36 0E 18 CD 89 60 21 03 00 22 15 ; .6...6....`!..".
6020  .byte D2 CD 28 33 38 35 CD CC 60 38 30 21 12 D3 CD 8D ; ..(385..`80!....
6030  .byte 0B 7E B1 77 3A 38 D2 87 5F 16 00 21 2F D3 19 EB ; .~.w:8.._..!/...
6040  .byte DD 6E 02 DD 66 03 29 29 29 7C 12 13 DD 6E 05 DD ; .n..f.)))|...n..
6050  .byte 66 06 29 29 29 7C 3D 12 C3 01 5E 21 00 55 C3 0C ; f.)))|=...^!.U..
6060  .byte 5E DD 36 0D 14 DD 36 0E 18 CD 89 60 21 03 00 22 ; ^.6...6....`!.."
6070  .byte 15 D2 CD 28 33 38 0C CD CC 60 38 07 FD CB 09 DE ; ...(38...`8.....
6080  .byte C3 01 5E 21 80 55 C3 0C 5E DD CB 18 46 C0 3A D5 ; ..^!.U..^...F.:.
6090  .byte D2 A7 20 13 01 00 00 59 50 CD D5 30 11 16 00 01 ; .. ....YP..0....
60A0  .byte 16 00 7E FE B0 28 06 11 04 00 01 00 00 DD 6E 02 ; ..~..(........n.
60B0  .byte DD 66 03 19 DD 75 02 DD 74 03 DD 6E 05 DD 66 06 ; .f...u..t..n..f.
60C0  .byte 09 DD 75 05 DD 74 06 DD CB 18 C6 C9 21 04 08 22 ; ..u..t......!.."
60D0  .byte 0F D2 3A 15 D4 E6 01 20 51 ED 5B FF D3 DD 4E 02 ; ..:.... Q.[...N.
60E0  .byte DD 46 03 21 F6 FF 09 A7 ED 52 30 62 21 10 00 09 ; .F.!.....R0b!...
60F0  .byte A7 ED 52 38 59 3A 15 D4 E6 04 20 27 DD 6E 05 DD ; ..R8Y:.... '.n..
6100  .byte 66 06 3A 0B D4 4F AF 47 ED 42 22 02 D4 32 88 D2 ; f.:..O.G.B"..2..
6110  .byte 3A E9 D2 2A E7 D2 22 07 D4 32 09 D4 21 15 D4 CB ; :..*.."..2..!...
6120  .byte FE 37 C9 3A 09 D4 A7 FA 2F 61 CD 9A 30 A7 C9 DD ; .7.:..../a..0...
6130  .byte 36 0A 80 DD 36 0B FE DD 36 0C FF 21 00 04 AF 22 ; 6...6...6..!..."
6140  .byte 07 D4 32 09 D4 32 88 D2 DD CB 18 CE 37 C9 2A FF ; ..2..2......7.*.
6150  .byte D3 11 08 00 19 EB DD 6E 02 DD 66 03 01 0A 00 09 ; .......n..f.....
6160  .byte 01 F3 FF A7 ED 52 30 03 01 15 00 DD 6E 02 DD 66 ; .....R0.....n..f
6170  .byte 03 09 22 FF D3 AF 32 FE D3 6F 67 32 04 D4 22 05 ; .."...2..og2..".
6180  .byte D4 37 C9 21 0C D3 CD 8D 0B 7E A1 20 32 DD 36 0D ; .7.!.....~. 2.6.
6190  .byte 0C DD 36 0E 11 CD 89 60 AF DD 77 0F DD 77 10 21 ; ..6....`..w..w.!
61A0  .byte 02 02 22 15 D2 CD 28 33 38 1A 21 0C D3 CD 8D 0B ; .."...(38.!.....
61B0  .byte 7E B1 77 21 79 D2 34 3E FE 32 85 D2 3E 14 DF DD ; ~.w!y.4>.2..>...
61C0  .byte 36 00 FF C9 3A 24 D2 0F 38 08 DD 36 0F F1 DD 36 ; 6...:$..8..6...6
61D0  .byte 10 61 DD 6E 0A DD 66 0B DD 7E 0C 11 20 00 19 CE ; .a.n..f..~.. ...
61E0  .byte 00 DD 75 0A DD 74 0B DD 77 0C 21 80 54 CD A8 0B ; ..u..t..w.!.T...
61F0  .byte C9 5C 5E FF FF FF FF FF DD 36 0D 18 DD 36 0E 30 ; .\^......6...6.0
6200  .byte DD CB 11 46 20 22 FD CB 06 BE FD CB 05 9E 21 B8 ; ...F "........!.
6210  .byte 3A 11 00 20 3E 09 CD 06 04 3E 0E 32 2D D2 3A A9 ; :.. >....>.2-.:.
6220  .byte D2 32 AA D2 DD CB 11 C6 2A 54 D2 22 6D D2 DD 6E ; .2......*T."m..n
6230  .byte 02 DD 66 03 11 90 FF 19 22 6F D2 21 70 00 22 65 ; ..f....."o.!p."e
6240  .byte D2 21 78 00 22 67 D2 DD 4E 13 3A 15 D4 E6 80 DD ; .!x."g..N.:.....
6250  .byte 77 13 28 34 B9 28 31 DD CB 18 7E 28 2B DD 5E 02 ; w.(4.(1...~(+.^.
6260  .byte DD 56 03 2A FF D3 A7 ED 52 CB 7C 28 07 7D 2F 6F ; .V.*....R.|(.}/o
6270  .byte 7C 2F 67 23 11 64 00 A7 ED 52 30 0C DD 36 0A 00 ; |/g#.d...R0..6..
6280  .byte DD 36 0B FE DD 36 0C FF DD 6E 0A DD 66 0B DD 7E ; .6...6...n..f..~
6290  .byte 0C 11 1A 00 19 CE 00 DD 75 0A DD 74 0B DD 77 0C ; ........u..t..w.
62A0  .byte DD CB 11 5E 20 72 DD CB 11 56 28 20 DD CB 18 7E ; ...^ r...V( ...~
62B0  .byte 28 66 3E 09 DF 3E 0C EF DD CB 11 96 DD CB 11 DE ; (f>..>..........
62C0  .byte 3E A0 32 83 D2 FD CB 06 CE C3 18 63 21 0A 0A 22 ; >.2........c!.."
62D0  .byte 15 D2 CD 28 33 38 41 DD CB 0C 7E 20 3B DD CB 11 ; ...(38A...~ ;...
62E0  .byte 4E 20 35 ED 5B 04 D4 CB 7A 28 07 7B 2F 5F 7A 2F ; N 5.[...z(.{/_z/
62F0  .byte 57 13 ED 53 01 D3 21 00 03 A7 ED 52 30 03 11 00 ; W..S..!....R0...
6300  .byte 03 EB 29 DD 75 14 DD 74 15 DD 36 12 00 DD CB 11 ; ..).u..t..6.....
6310  .byte CE FD CB 06 9E 3E 0B EF 11 A8 64 DD CB 11 4E C2 ; .....>....d...N.
6320  .byte E7 63 DD CB 11 56 C2 E7 63 11 C2 64 DD CB 11 5E ; .c...V..c..d...^
6330  .byte CA E7 63 3A 38 D2 FE 0C 38 0B FE 1C 38 13 11 DF ; ..c:8...8...8...
6340  .byte 64 0E 01 18 28 11 F9 64 0E 04 3A AA D2 FE 50 30 ; d...(..d..:...P0
6350  .byte 1C 3A 00 D3 FE 02 20 07 11 13 65 0E 03 18 0E 11 ; .:.... ...e.....
6360  .byte C5 64 0E 02 FE 03 30 05 11 DF 64 0E 01 79 32 82 ; .d....0...d..y2.
6370  .byte D2 D5 ED 5B 01 D3 7A 21 C5 65 FE 04 30 29 A7 28 ; ...[..z!.e..0).(
6380  .byte 65 21 BD 65 3D 20 0E 7B FE 60 38 5A FE A0 38 17 ; e!.e= .{.`8Z..8.
6390  .byte 21 D5 65 18 12 21 C5 65 3D 20 4B 7B FE 80 38 07 ; !.e..!.e= K{..8.
63A0  .byte FE A0 30 42 21 CD 65 5E 23 56 23 4E 23 46 23 E5 ; ..0B!.e^#V#N#F#.
63B0  .byte D5 DD 6E 05 DD 66 06 11 F2 FF 19 ED 5B 57 D2 A7 ; ..n..f......[W..
63C0  .byte ED 52 EB DD 6E 02 DD 66 03 09 ED 4B 54 D2 A7 ED ; .R..n..f...KT...
63D0  .byte 42 C1 C4 07 2F E1 4E 23 5E 23 56 DD CB 11 7E CC ; B.../.N#^#V...~.
63E0  .byte AA 33 DD CB 11 FE D1 DD 6E 12 26 00 19 7E FE FF ; .3......n.&..~..
63F0  .byte 20 08 23 7E DD 77 12 C3 E7 63 6F 26 00 29 5D 54 ;  .#~.w...co&.)]T
6400  .byte 29 29 29 19 11 2D 65 19 DD 75 0F DD 74 10 DD CB ; )))..-e..u..t...
6410  .byte 11 4E 20 04 DD 34 12 C9 DD 7E 14 DD 86 16 DD 77 ; .N ..4...~.....w
6420  .byte 16 DD 7E 15 F5 DD 8E 17 DD 77 17 F1 DD 8E 12 FE ; ..~......w......
6430  .byte 18 38 01 AF DD 77 12 DD 5E 0A DD 56 0B DD 7E 0C ; .8...w..^..V..~.
6440  .byte A7 F2 4A 64 21 00 FC ED 52 D0 EB DD 5E 14 DD 56 ; ..Jd!...R...^..V
6450  .byte 15 4B 42 CB 3A CB 1B CB 3A CB 1B CB 3A CB 1B CB ; .KB.:...:...:...
6460  .byte 3A CB 1B CB 3A CB 1B A7 ED 52 DE 00 DD 75 0A DD ; :...:....R...u..
6470  .byte 74 0B DD 77 0C DD 6E 05 DD 66 06 AF 11 08 00 ED ; t..w..n..f......
6480  .byte 52 38 0F 69 60 11 10 00 AF ED 52 DD 75 14 DD 74 ; R8.i`.....R.u..t
6490  .byte 15 D0 DD 77 0A DD 77 0B DD 77 0C DD CB 11 8E DD ; ...w..w..w......
64A0  .byte CB 11 D6 DD 36 12 00 C9 00 00 00 00 00 00 03 03 ; ....6...........
64B0  .byte 03 03 03 03 02 02 02 02 02 02 04 04 04 04 04 04 ; ................
64C0  .byte FF 00 00 FF 00 00 00 00 00 00 00 03 03 03 03 03 ; ................
64D0  .byte 03 02 02 02 02 02 02 01 01 01 01 01 01 FF 12 00 ; ................
64E0  .byte 00 00 00 00 00 03 03 03 03 03 03 02 02 02 02 02 ; ................
64F0  .byte 02 05 05 05 05 05 05 FF 12 00 00 00 00 00 00 03 ; ................
6500  .byte 03 03 03 03 03 02 02 02 02 02 02 06 06 06 06 06 ; ................
6510  .byte 06 FF 12 00 00 00 00 00 00 03 03 03 03 03 03 02 ; ................
6520  .byte 02 02 02 02 02 07 07 07 07 07 07 FF 12 4E 50 52 ; .............NPR
6530  .byte 54 FF FF 6E 70 72 74 FF FF FE 42 44 FF FF FF 08 ; T..nprt...BD....
6540  .byte 0A 0C 0E FF FF 28 2A 2C 2E FF FF FE 42 44 FF FF ; .....(*,....BD..
6550  .byte FF FE 12 14 FF FF FF FE 32 34 FF FF FF FE 42 44 ; ........24....BD
6560  .byte FF FF FF 16 18 1A 1C FF FF 36 38 3A 3C FF FF FE ; .........68:<...
6570  .byte 42 44 FF FF FF 56 58 5A 5C FF FF 76 78 7A 7C FF ; BD...VXZ\..vxz|.
6580  .byte FF FE 42 44 FF FF FF 00 02 04 06 FF FF 20 22 24 ; ..BD......... "$
6590  .byte 26 FF FF FE 42 44 FF FF FF 4E 4A 4C 54 FF FF 6E ; &...BD...NJLT..n
65A0  .byte 6A 6C 74 FF FF FE 42 44 FF FF FF 4E 46 48 54 FF ; jlt...BD...NFHT.
65B0  .byte FF 6E 66 68 74 FF FF FE 42 44 FF FF FF DD 65 04 ; .nfht...BD....e.
65C0  .byte 00 00 10 00 00 E4 65 00 00 00 00 10 00 EB 65 FE ; ......e.......e.
65D0  .byte FF 01 00 00 00 F2 65 02 00 00 00 01 00 FE 60 FF ; ......e.......`.
65E0  .byte FF FF FF FF FE 60 62 FF FF FF FF FE 60 62 64 FF ; .....`b.....`bd.
65F0  .byte FF FF FE 60 64 FF FF FF FF DD 36 0D 10 DD 36 0E ; ...`d.....6...6.
6600  .byte 1F DD 5E 12 16 00 21 D0 66 19 22 15 D2 7E A7 20 ; ..^...!.f."..~. 
6610  .byte 07 DD 77 12 5F C3 06 66 3D 20 08 0E 00 61 2E 28 ; ..w._..f= ...a.(
6620  .byte C3 7A 66 3D 20 08 0E FF 21 D8 FF C3 7A 66 3D 20 ; .zf= ...!...zf= 
6630  .byte 07 0E 00 69 61 C3 7A 66 DD 7E 11 FE 20 C2 83 66 ; ...ia.zf.~.. ..f
6640  .byte 21 FF FF 22 13 D2 21 FC FF 22 15 D2 CD AF 7C DA ; !.."..!.."....|.
6650  .byte 83 66 11 00 00 4B 42 CD 5C AC 21 01 00 22 13 D2 ; .f...KB.\.!.."..
6660  .byte 21 FC FF 22 15 D2 CD AF 7C 38 18 11 0E 00 01 00 ; !.."....|8......
6670  .byte 00 CD 5C AC 3E 0A EF C3 83 66 DD 75 07 DD 74 08 ; ..\.>....f.u..t.
6680  .byte DD 71 09 DD 6E 11 DD 66 12 11 08 00 19 DD 75 11 ; .q..n..f......u.
6690  .byte DD 74 12 DD 6E 0A DD 66 0B DD 7E 0C 11 20 00 19 ; .t..n..f..~.. ..
66A0  .byte 8A DD 75 0A DD 74 0B DD 77 0C 2A 15 D2 7E 87 5F ; ..u..t..w.*..~._
66B0  .byte 21 EB 66 19 4E 23 46 11 04 67 CD 75 7C 21 04 0A ; !.f.N#F..g.u|!..
66C0  .byte 22 15 D2 CD 28 33 21 04 08 22 0F D2 D4 C1 2F C9 ; "...(3!.."..../.
66D0  .byte 01 01 01 01 01 01 01 01 01 01 03 03 04 02 02 02 ; ................
66E0  .byte 02 02 02 02 02 02 02 03 03 04 00 F5 66 F5 66 F5 ; ............f.f.
66F0  .byte 66 FE 66 01 67 00 0C 01 0C 02 0C 01 0C FF 01 01 ; f.f.g...........
6700  .byte FF 03 01 FF 00 02 04 FF FF FF 20 22 24 FF FF FF ; .......... "$...
6710  .byte FF FF FF FF FF FF 00 02 44 FF FF FF 46 22 4A FF ; ........D...F"J.
6720  .byte FF FF FF FF FF FF FF FF 40 02 44 FF FF FF 26 22 ; ........@.D...&"
6730  .byte 2A FF FF FF FF FF FF FF FF FF 40 02 04 FF FF FF ; *.........@.....
6740  .byte 46 22 4A FF FF FF FF DD CB 18 EE 21 40 00 22 65 ; F"J........!@."e
6750  .byte D2 21 40 00 22 67 D2 DD CB 18 46 20 24 DD 6E 02 ; .!@."g....F $.n.
6760  .byte DD 66 03 DD 75 12 DD 74 13 DD 6E 05 DD 66 06 DD ; .f..u..t..n..f..
6770  .byte 75 14 DD 74 15 DD 36 11 E0 DD CB 18 C6 DD CB 18 ; u..t..6.........
6780  .byte CE DD 36 0D 1A DD 36 0E 10 DD 6E 02 DD 66 03 22 ; ..6...6...n..f."
6790  .byte 0F D2 21 2E 68 DD 5E 11 16 00 19 4D 44 0A A7 F2 ; ..!.h.^....MD...
67A0  .byte A3 67 15 5F DD 6E 12 DD 66 13 19 DD 75 02 DD 74 ; .g._.n..f...u..t
67B0  .byte 03 ED 5B 0F D2 A7 ED 52 22 0F D2 03 16 00 0A A7 ; ..[....R".......
67C0  .byte F2 C4 67 15 5F DD 6E 14 DD 66 15 19 DD 75 05 DD ; ..g._.n..f...u..
67D0  .byte 74 06 3A 09 D4 A7 FA F8 67 21 06 08 22 15 D2 CD ; t.:.....g!.."...
67E0  .byte 28 33 38 14 2A FF D3 ED 5B 0F D2 19 22 FF D3 01 ; (38.*...[..."...
67F0  .byte 10 00 11 00 00 CD F5 7C 21 10 69 3A D5 D2 A7 28 ; .......|!.i:...(
6800  .byte 03 21 22 69 DD 75 0F DD 74 10 DD CB 18 4E 20 10 ; .!"i.u..t....N .
6810  .byte DD 7E 11 3C 3C DD 77 11 FE E0 D8 DD CB 18 CE C9 ; .~.<<.w.........
6820  .byte DD 7E 11 3D 3D DD 77 11 C0 DD CB 18 8E C9 CD 00 ; .~.==.w.........
6830  .byte CD 01 CD 01 CD 02 CD 02 CD 03 CD 04 CD 04 CD 05 ; ................
6840  .byte CD 06 CD 06 CD 07 CD 08 CE 09 CE 09 CE 0A CE 0B ; ................
6850  .byte CE 0C CE 0D CF 0E CF 0F CF 10 D0 11 D0 12 D0 13 ; ................
6860  .byte D1 14 D1 15 D2 16 D3 18 D3 19 D4 1A D5 1B D6 1D ; ................
6870  .byte D6 1E D7 1F D8 20 D9 22 DB 23 DC 24 DD 25 DE 27 ; ..... .".#.$.%.'
6880  .byte E0 28 E1 29 E3 2A E5 2B E6 2C E8 2D EA 2E EC 2F ; .(.).*.+.,.-.../
6890  .byte EE 30 F0 31 F2 31 F5 32 F7 32 F9 33 FB 33 FE 33 ; .0.1.1.2.2.3.3.3
68A0  .byte 00 33 02 33 05 33 07 33 09 32 0B 32 0E 31 10 31 ; .3.3.3.3.2.2.1.1
68B0  .byte 12 30 14 2F 16 2E 18 2D 1A 2C 1C 2B 1D 2A 1F 29 ; .0./...-.,.+.*.)
68C0  .byte 20 28 22 26 23 25 25 24 26 23 27 21 28 20 29 1F ;  ("&#%%$&#'!( ).
68D0  .byte 2A 1D 2B 1C 2C 1B 2C 1A 2D 18 2E 17 2E 16 2F 15 ; *.+.,.,.-...../.
68E0  .byte 2F 13 30 12 30 11 31 10 31 0F 31 0E 31 0D 32 0C ; /.0.0.1.1.1.1.2.
68F0  .byte 32 0B 32 0A 32 09 32 09 33 08 33 07 33 06 33 05 ; 2.2.2.2.3.3.3.3.
6900  .byte 33 05 33 04 33 03 33 03 33 02 33 01 33 01 33 00 ; 3.3.3.3.3.3.3.3.
6910  .byte FE FF FF FF FF FF 18 1A 18 1A FF FF FF FF FF FF ; ................
6920  .byte FF FF FE FF FF FF FF FF 6C 6E 6E 48 FF FF FF FF ; ........lnnH....
6930  .byte FE FF FF FF FF FF 6C 6E 6C 6E FF FF FF FF DD CB ; ......lnln......
6940  .byte 18 EE DD 7E 15 FE AA 28 20 AF DD 77 11 DD 36 15 ; ...~...( ..w..6.
6950  .byte AA DD 77 16 DD 77 17 DD 77 07 DD 77 08 DD 77 09 ; ..w..w..w..w..w.
6960  .byte DD 77 0A DD 77 0B DD 77 0C DD 7E 11 3D 20 35 FD ; .w..w..w..~.= 5.
6970  .byte CB 00 6E 28 2F 3A 38 D2 FE 12 28 28 3A 15 D4 07 ; ..n(/:8...((:...
6980  .byte 38 22 3A E9 D2 ED 5B E7 D2 13 4F 2A 07 D4 7D 2F ; 8":...[...O*..}/
6990  .byte 6F 7C 2F 67 3A 09 D4 A7 FA A4 69 2F 19 89 22 07 ; o|/g:.....i/..".
69A0  .byte D4 32 09 D4 11 C2 69 01 BB 69 CD 75 7C DD 34 11 ; .2....i..i.u|.4.
69B0  .byte DD 7E 11 FE 18 D8 DD 36 00 FF C9 00 08 01 08 02 ; .~.....6........
69C0  .byte 08 FF 74 76 FF FF FF FF FF FF FF FF FF FF FF FF ; ..tv............
69D0  .byte FF FF FF FF 78 7A FF FF FF FF FF FF FF FF FF FF ; ....xz..........
69E0  .byte FF FF FF FF FF FF 7C 7E FF FF FF FF FF DD CB 18 ; ......|~........
69F0  .byte EE DD 36 0D 1A DD 36 0E 10 21 10 69 3A D5 D2 A7 ; ..6...6..!.i:...
6A00  .byte 28 03 21 22 69 DD 75 0F DD 74 10 3A 09 D4 A7 FA ; (.!"i.u..t.:....
6A10  .byte 3C 6A 21 06 08 22 15 D2 CD 28 33 38 1F 11 00 00 ; <j!.."...(38....
6A20  .byte DD 7E 05 E6 1F FE 10 30 02 1E 80 DD 73 0A DD 72 ; .~.....0....s..r
6A30  .byte 0B DD 36 0C 00 01 10 00 CD F5 7C C9 0E 00 69 61 ; ..6.......|...ia
6A40  .byte DD 7E 05 E6 1F 28 04 21 C0 FF 0D DD 75 0A DD 74 ; .~...(.!....u..t
6A50  .byte 0B DD 71 0C C9 DD CB 18 EE DD CB 18 46 20 10 DD ; ..q.........F ..
6A60  .byte 7E 05 DD 77 11 DD 7E 06 DD 77 12 DD CB 18 C6 DD ; ~..w..~..w......
6A70  .byte CB 18 4E 28 1D 2A 57 D2 01 F0 FF 09 DD 5E 05 DD ; ..N(.*W......^..
6A80  .byte 56 06 AF ED 52 30 07 DD 77 0F DD 77 10 C9 DD CB ; V...R0..w..w....
6A90  .byte 18 8E DD 7E 16 DD 86 17 DD 77 17 FE 18 38 17 DD ; ...~.....w...8..
6AA0  .byte 6E 0A DD 66 0B DD 7E 0C 11 40 00 19 8A DD 75 0A ; n..f..~..@....u.
6AB0  .byte DD 74 0B DD 77 0C DD 36 0D 1A DD 36 0E 10 3A 09 ; .t..w..6...6..:.
6AC0  .byte D4 A7 FA E0 6A 21 06 08 22 15 D2 CD 28 33 38 10 ; ....j!.."...(38.
6AD0  .byte DD 36 16 01 01 10 00 DD 5E 0A DD 56 0B CD F5 7C ; .6......^..V...|
6AE0  .byte 21 10 69 3A D5 D2 A7 28 03 21 22 69 DD 75 0F DD ; !.i:...(.!"i.u..
6AF0  .byte 74 10 2A 57 D2 11 A8 00 19 DD 5E 05 DD 56 06 AF ; t.*W......^..V..
6B00  .byte ED 52 D0 DD 77 0A DD 77 0B DD 77 0C DD 77 16 DD ; .R..w..w..w..w..
6B10  .byte 77 17 DD 77 04 DD 7E 11 DD 77 05 DD 7E 12 DD 77 ; w..w..~..w..~..w
6B20  .byte 06 DD CB 18 CE C9 DD CB 18 EE DD 36 0D 02 DD 36 ; ...........6...6
6B30  .byte 0E 02 21 03 03 22 15 D2 CD 28 33 D4 D9 2F DD 6E ; ..!.."...(3../.n
6B40  .byte 0A DD 66 0B DD 7E 0C DD 5E 13 DD 56 14 19 CE 00 ; ..f..~..^..V....
6B50  .byte DD 75 0A DD 74 0B DD 77 0C DD 6E 02 DD 66 03 22 ; .u..t..w..n..f."
6B60  .byte 0F D2 DD 6E 05 DD 66 06 22 11 D2 21 00 00 22 13 ; ...n..f."..!..".
6B70  .byte D2 22 15 D2 DD 75 0F DD 74 10 21 D7 6B 3A 38 D2 ; ."...u..t.!.k:8.
6B80  .byte FE 05 28 07 FE 0B 28 03 21 D5 6B 3A 24 D2 E6 01 ; ..(...(.!.k:$...
6B90  .byte 5F 16 00 19 7E CD 5D 2F DD 4E 02 DD 46 03 69 60 ; _...~.]/.N..F.i`
6BA0  .byte 11 F8 FF 19 ED 5B 54 D2 A7 ED 52 38 23 14 EB ED ; .....[T...R8#...
6BB0  .byte 42 38 1D DD 4E 05 DD 46 06 69 60 11 10 00 19 ED ; B8..N..F.i`.....
6BC0  .byte 5B 57 D2 A7 ED 52 38 08 21 C0 00 19 A7 ED 42 D0 ; [W...R8.!.....B.
6BD0  .byte DD 36 00 FF C9 06 08 34 36 DD CB 18 EE DD CB 18 ; .6.....46.......
6BE0  .byte 46 20 2D DD 5E 02 DD 56 03 DD 73 14 DD 72 15 AF ; F -.^..V..s..r..
6BF0  .byte DD 77 0F DD 77 10 DD 77 12 DD 77 07 DD 77 08 DD ; .w..w..w..w..w..
6C00  .byte 77 09 2A 54 D2 01 00 01 09 ED 52 D0 DD CB 18 C6 ; w.*T......R.....
6C10  .byte DD 36 0D 14 DD 36 0E 20 DD 6E 02 DD 66 03 ED 5B ; .6...6. .n..f..[
6C20  .byte FF D3 A7 ED 52 38 12 11 40 00 ED 52 30 0B DD 7E ; ....R8..@..R0..~
6C30  .byte 12 FE 05 30 04 DD 36 12 05 DD 5E 12 16 00 21 3C ; ...0..6...^...!<
6C40  .byte 6D 19 22 15 D2 7E A7 20 07 DD 77 12 5F C3 3E 6C ; m."..~. ..w._.>l
6C50  .byte 3D 20 32 DD 6E 02 DD 66 03 11 30 00 19 ED 5B 54 ; = 2.n..f..0...[T
6C60  .byte D2 AF ED 52 30 17 DD 77 0F DD 77 10 DD 7E 14 DD ; ...R0..w..w..~..
6C70  .byte 77 02 DD 7E 15 DD 77 03 DD CB 18 86 C9 0E FF 21 ; w..~..w........!
6C80  .byte 00 FE C3 FD 6C 3D 20 07 0E 00 69 61 C3 FD 6C DD ; ....l= ...ia..l.
6C90  .byte 7E 11 FE 20 C2 06 6D CD AF 7C DA 06 6D C5 DD 5E ; ~.. ..m..|..m..^
6CA0  .byte 02 DD 56 03 DD 4E 05 DD 46 06 DD E5 E5 DD E1 AF ; ..V..N..F.......
6CB0  .byte DD 36 00 0D DD 77 01 DD 73 02 DD 72 03 DD 77 04 ; .6...w..s..r..w.
6CC0  .byte 21 20 00 09 DD 75 05 DD 74 06 DD 77 11 DD 77 13 ; ! ...u..t..w..w.
6CD0  .byte DD 77 14 DD 77 15 DD 77 16 DD 77 17 DD 36 07 00 ; .w..w..w..w..6..
6CE0  .byte DD 36 08 FF DD 36 09 FF DD 36 0A 80 DD 36 0B 01 ; .6...6...6...6..
6CF0  .byte DD 77 0C DD E1 C1 3E 0A EF 0E 00 69 61 DD 75 07 ; .w....>....ia.u.
6D00  .byte DD 74 08 DD 71 09 DD 6E 11 DD 66 12 11 08 00 19 ; .t..q..n..f.....
6D10  .byte DD 75 11 DD 74 12 2A 15 D2 7E 87 5F 21 47 6D 19 ; .u..t.*..~._!Gm.
6D20  .byte 4E 23 46 11 5E 6D CD 75 7C 21 00 10 22 15 D2 CD ; N#F.^m.u|!.."...
6D30  .byte 28 33 21 04 10 22 0F D2 D4 C1 2F C9 01 01 01 01 ; (3!.."..../.....
6D40  .byte 00 02 02 03 01 01 00 4F 6D 4F 6D 54 6D 59 6D 00 ; .......OmOmTmYm.
6D50  .byte 02 01 02 FF 02 02 03 02 FF 04 02 05 02 FF FE 0A ; ................
6D60  .byte FF FF FF FF 0C 0E 10 FF FF FF FF FF FF FF FF FF ; ................
6D70  .byte FE FF FF FF FF FF 0C 0E 2C FF FF FF FF FF FF FF ; ........,.......
6D80  .byte FF FF FE 0A FF FF FF FF 12 14 16 FF FF FF 32 34 ; ..............24
6D90  .byte FF FF FF FF FE FF FF FF FF FF 12 14 16 FF FF FF ; ................
6DA0  .byte 32 34 FF FF FF FF FE 0A FF FF FF FF 12 14 16 FF ; 24..............
6DB0  .byte FF FF 30 34 FF FF FF FF FE FF FF FF FF FF 12 14 ; ..04............
6DC0  .byte 16 FF FF FF 30 34 FF FF FF FF DD CB 18 EE 3A 38 ; ....04........:8
6DD0  .byte D2 FE 07 28 0C 21 40 00 22 65 D2 21 40 00 22 67 ; ...(.!@."e.!@."g
6DE0  .byte D2 DD 36 0D 1A DD 36 0E 10 0E 00 3A 09 D4 A7 FA ; ..6...6....:....
6DF0  .byte 0A 6E 21 06 08 22 15 D2 CD 28 33 0E 00 38 0B 01 ; .n!.."...(3..8..
6E00  .byte 10 00 11 00 00 CD F5 7C 0E 01 DD 6E 12 DD 66 13 ; .......|...n..f.
6E10  .byte 23 DD 75 12 DD 74 13 11 A0 00 AF ED 52 38 09 DD ; #.u..t......R8..
6E20  .byte 77 12 DD 77 13 DD 34 14 11 01 00 DD CB 14 46 28 ; w..w..4.......F(
6E30  .byte 03 11 FF FF DD 6E 02 DD 66 03 19 DD 75 02 DD 74 ; .....n..f...u..t
6E40  .byte 03 79 A7 28 07 2A FF D3 19 22 FF D3 21 10 69 3A ; .y.(.*..."..!.i:
6E50  .byte D5 D2 A7 28 09 21 30 69 3D 28 03 21 22 69 DD 75 ; ...(.!0i=(.!"i.u
6E60  .byte 0F DD 74 10 C9 DD CB 18 AE DD 36 0D 0A DD 36 0E ; ..t.......6...6.
6E70  .byte 10 DD 5E 12 16 00 21 EF 6E 19 22 15 D2 7E A7 20 ; ..^...!.n."..~. 
6E80  .byte 07 DD 77 12 5F C3 76 6E 3D 20 08 0E FF 21 00 FF ; ..w._.vn= ...!..
6E90  .byte C3 A2 6E 3D 20 08 0E 00 21 00 01 C3 A2 6E 0E 00 ; ..n= ...!....n..
6EA0  .byte 69 61 DD 75 07 DD 74 08 DD 71 09 DD 6E 11 DD 66 ; ia.u..t..q..n..f
6EB0  .byte 12 11 08 00 19 DD 75 11 DD 74 12 DD 36 0A 00 DD ; ......u..t..6...
6EC0  .byte 36 0B 02 DD 36 0C 00 2A 15 D2 7E 87 5F 16 00 21 ; 6...6..*..~._..!
6ED0  .byte 0A 6F 19 4E 23 46 11 24 6F CD 75 7C 21 03 02 22 ; .o.N#F.$o.u|!.."
6EE0  .byte 15 D2 CD 28 33 21 00 00 22 0F D2 D4 C1 2F C9 01 ; ...(3!.."..../..
6EF0  .byte 01 01 01 01 01 01 01 01 03 03 03 03 02 02 02 02 ; ................
6F00  .byte 02 02 02 02 02 04 04 04 04 00 14 6F 14 6F 19 6F ; ...........o.o.o
6F10  .byte 1E 6F 21 6F 00 08 01 08 FF 02 08 03 08 FF 00 FF ; .o!o............
6F20  .byte FF 02 FF FF 60 62 FF FF FF FF FF FF FF FF FF FF ; ....`b..........
6F30  .byte FF FF FF FF FF FF 64 66 FF FF FF FF FF FF FF FF ; ......df........
6F40  .byte FF FF FF FF FF FF FF FF 68 6A FF FF FF FF FF FF ; ........hj......
6F50  .byte FF FF FF FF FF FF FF FF FF FF 6C 6E FF FF FF FF ; ..........ln....
6F60  .byte FF DD CB 18 EE DD 36 0D 0C DD 36 0E 14 DD 7E 11 ; ......6...6...~.
6F70  .byte FE 02 28 03 A7 20 24 3A 24 D2 E6 01 28 05 01 00 ; ..(.. $:$...(...
6F80  .byte 00 18 03 01 46 70 DD 34 17 DD 7E 17 FE 3C DA 2D ; ....Fp.4..~..<.-
6F90  .byte 70 DD 36 17 00 DD 34 11 C3 2D 70 FE 01 C2 1A 70 ; p.6...4..-p....p
6FA0  .byte DD 34 17 DD 7E 17 FE 64 20 60 CD AF 7C DA 0A 70 ; .4..~..d `..|..p
6FB0  .byte C5 DD 5E 02 DD 56 03 DD 4E 05 DD 46 06 DD E5 E5 ; ..^..V..N..F....
6FC0  .byte DD E1 AF DD 36 00 0D DD 77 01 DD 73 02 DD 72 03 ; ....6...w..s..r.
6FD0  .byte DD 77 04 21 06 00 09 DD 75 05 DD 74 06 DD 77 11 ; .w.!....u..t..w.
6FE0  .byte DD 77 13 DD 77 14 DD 77 15 DD 77 16 DD 77 17 DD ; .w..w..w..w..w..
6FF0  .byte 36 07 00 DD 36 08 FE DD 36 09 FF DD 77 0A DD 77 ; 6...6...6...w..w
7000  .byte 0B DD 77 0C DD E1 C1 3E 0A EF 01 46 70 FE 78 38 ; ..w....>...Fp.x8
7010  .byte 1C DD 36 17 00 DD 34 11 18 13 FE 03 20 0F 01 00 ; ..6...4..... ...
7020  .byte 00 DD 34 17 DD 7E 17 A7 20 03 DD 71 11 DD 71 0F ; ..4..~.. ..q..q.
7030  .byte DD 70 10 21 02 02 22 15 D2 CD 28 33 21 00 00 22 ; .p.!.."...(3!.."
7040  .byte 0F D2 D4 C1 2F C9 1C 1E FF FF FF FF FE 3E FF FF ; ..../........>..
7050  .byte FF FF FF FF FF FF FF FF 40 42 FF FF FF FF FE 62 ; ........@B.....b
7060  .byte FF FF FF FF FF DD CB 18 EE DD 36 0D 20 DD 36 0E ; ..........6. .6.
7070  .byte 1C CD DA 7C DD CB 11 46 20 38 21 18 01 DD 75 05 ; ...|...F 8!...u.
7080  .byte DD 74 06 21 B7 A8 11 00 20 3E 09 CD 06 04 3E 11 ; .t.!.... >....>.
7090  .byte 32 2D D2 3E 0B DF AF 32 ED D2 DD 77 12 DD 36 14 ; 2-.>...2...w..6.
70A0  .byte F0 DD 36 15 72 21 60 07 11 00 01 CD C0 7C DD CB ; ..6.r!`......|..
70B0  .byte 11 C6 DD 7E 13 E6 3F 5F 16 00 21 B0 72 19 7E A7 ; ...~..?_..!.r.~.
70C0  .byte F2 C7 70 0E FF 18 02 0E 00 DD 77 0A DD 71 0B DD ; ..p.......w..q..
70D0  .byte 71 0C DD 5E 12 16 00 DD 6E 14 DD 66 15 19 22 15 ; q..^....n..f..".
70E0  .byte D2 7E A7 20 08 23 7E DD 77 12 C3 D2 70 3D 87 5F ; .~. .#~.w...p=._
70F0  .byte 16 00 21 9A 72 19 7E 23 66 6F E9 2A 6D D2 11 22 ; ..!.r.~#fo.*m.."
7100  .byte 00 19 DD 5E 02 DD 56 03 A7 ED 52 0E FF 21 00 FF ; ...^..V...R..!..
7110  .byte DA 54 72 DD 36 12 00 DD CB 11 4E 20 0F DD 36 14 ; .Tr.6.....N ..6.
7120  .byte F3 DD                                           ; ..
7122  36 15       LD (HL),$15
7124  72          LD (HL),D
7125  DD CB 11 CE SET 1,(IX+17)
7129  C3 54 72    JP $7254
712C  .byte DD 36 14 F6 DD 36 15 72 DD CB 11 8E C3 54 72 2A ; .6...6.r.....Tr*
713C  .byte 6D D2 11 BA 00 19 DD 5E 02 DD 56 03 A7 ED 52 0E ; m......^..V...R.
714C  .byte 00 21 00 01 D2 54 72 DD 36 12 00 DD CB 11 56 20 ; .!...Tr.6.....V 
715C  .byte 0F DD 36 14 F0 DD 36 15 72 DD CB 11 D6 C3 54 72 ; ..6...6.r.....Tr
716C  .byte DD 36 14 F9 DD 36 15 72 DD CB 11 96 C3 54 72 DD ; .6...6.r.....Tr.
717C  .byte 36 0A 60 DD 36 0B 00 DD 36 0C 00 2A 57 D2 11 6C ; 6.`.6...6..*W..l
718C  .byte 00 19 DD 5E 05 DD 56 06 AF ED 52 4F 69 61 D2 54 ; ...^..V...ROia.T
719C  .byte 72 DD 36 12 00 DD 36 14 FF DD 36 15 72 C3 54 72 ; r.6...6...6.r.Tr
71AC  .byte 0E 00 21 00 04 C3 54 72 DD 36 0A 60 DD 36 0B 00 ; ..!...Tr.6.`.6..
71BC  .byte DD 36 0C 00 2A 57 D2 11 6C 00 19 DD 5E 05 DD 56 ; .6..*W..l...^..V
71CC  .byte 06 AF ED 52 4F 69 61 D2 54 72 DD 36 12 00 DD 36 ; ...ROia.Tr.6...6
71DC  .byte 14 0B DD 36 15 73 C3 54 72 0E FF 21 00 FC 18 68 ; ...6.s.Tr..!...h
71EC  .byte 0E 00 69 61 18 62 0E 00 69 61 DD 36 14 FC DD 36 ; ..ia.b..ia.6...6
71FC  .byte 15 72 DD 71 12 DD 71 13 18 4E DD 36 0A 00 DD 36 ; .r.q..q..N.6...6
720C  .byte 0B FF DD 36 0C FF 2A 57 D2 11 18 00 19 DD 5E 05 ; ...6..*W......^.
721C  .byte DD 56 06 AF ED 52 4F 69 61 DA 54 72 DD 6E 02 DD ; .V...ROia.Tr.n..
722C  .byte 66 03 ED 5B 6D D2 AF ED 52 4F 69 61 38 0D DD 36 ; f..[m...ROia8..6
723C  .byte 14 F0 DD 36 15 72 DD 77 12 18 0D DD 36 14 F3 DD ; ...6.r.w....6...
724C  .byte 36 15 72 DD 77 12 18 00                         ; 6.r.w...
7254  DD 75 07    LD (IX+7),L
7257  DD 74 08    LD (IX+8),H
725A  DD 71 09    LD (IX+9),C
725D  2A 15 D2    LD HL,($D215)
7260  5E          LD E,(HL)
7261  16 00       LD D,$00
7263  21 17 73    LD HL,$7317
7266  19          ADD HL,DE
7267  7E          LD A,(HL)
7268  21 47 73    LD HL,$7347
726B  A7          AND A
726C  28 03       JR Z,$7271
726E  21 59 73    LD HL,$7359
7271  5F          LD E,A
7272  DD 7E 18    LD A,(IX+24)
7275  E6 FD       AND $FD
7277  B3          OR E
7278  DD 77 18    LD (IX+24),A
727B  DD 75 0F    LD (IX+15),L
727E  DD 74 10    LD (IX+16),H
7281  21 12 00    LD HL,$0012
7284  22 17 D2    LD ($D217),HL
7287  CD FD 77    CALL $77FD
728A  CD 36 7A    CALL $7A36
728D  DD 34 13    INC (IX+19)
7290  DD 7E 13    LD A,(IX+19)
7293  E6 0F       AND $0F
7295  C0          RET NZ
7296  DD 34 12    INC (IX+18)
7299  C9          RET
729A  .byte FB 70 3B 71 7B 71 AC 71 B4 71 E5 71 EC 71 F2 71 ; .p;q{q.q.q.q.q.q
72AA  .byte 06 72 00 00 EC 71 00 14 28 28 3C 3C 3C 50 50 50 ; .r...q..((<<<PPP
72BA  .byte 50 64 64 64 64 64 64 64 64 64 64 50 50 50 50 3C ; PddddddddddPPPP<
72CA  .byte 3C 3C 28 28 14 00 00 EC D8 D8 C4 C4 C4 B0 B0 B0 ; <<((............
72DA  .byte B0 9C 9C 9C 9C 9C 9C 9C 9C 9C 9C B0 B0 B0 B0 C4 ; ................
72EA  .byte C4 C4 D8 D8 EC 00 01 00 00 02 00 00 03 00 00 05 ; ................
72FA  .byte 00 00 09 00 00 07 07 07 07 04 04 04 04 04 08 00 ; ................
730A  .byte 00 0B 0B 0B 0B 06 06 06 06 06 08 00 00 00 00 02 ; ................
731A  .byte 02 02 00 00 02 02 00 02                         ; ........
7322  00          NOP
7323  00          NOP
7324  00          NOP
7325  01 04 01    LD BC,$0104
7328  00          NOP
7329  01 04 01    LD BC,$0104
732C  01 01 04    LD BC,$0401
732F  01 01 01    LD BC,$0101
7332  04          INC B
7333  01 FF 02    LD BC,$02FF
7336  02          LD (BC),A
7337  01 05 01    LD BC,$0105
733A  02          LD (BC),A
733B  01 05 01    LD BC,$0105
733E  03          INC BC
733F  01 05 01    LD BC,$0105
7342  03          INC BC
7343  01 05 01    LD BC,$0105
7346  FF          RST $38
7347  20 22       JR NZ,$736B
7349  24          INC H
734A  26 28       LD H,$28
734C  FF          RST $38
734D  40          LD B,B
734E  42          LD B,D
734F  44          LD B,H
7350  46          LD B,(HL)
7351  48          LD C,B
7352  FF          RST $38
7353  60          LD H,B
7354  62          LD H,D
7355  64          LD H,H
7356  66          LD H,(HL)
7357  68          LD L,B
7358  FF          RST $38
7359  2A 2C 2E    LD HL,($2E2C)
735C  30 32       JR NC,$7390
735E  FF          RST $38
735F  4A          LD C,D
7360  4C          LD C,H
7361  4E          LD C,(HL)
7362  50          LD D,B
7363  52          LD D,D
7364  FF          RST $38
7365  6A          LD L,D
7366  6C          LD L,H
7367  6E          LD L,(HL)
7368  70          LD (HL),B
7369  72          LD (HL),D
736A  FF          RST $38
736B  DD CB 18 EE SET 5,(IX+24)
736F  DD CB 18 46 BIT 0,(IX+24)
7373  20 14       JR NZ,$7389
7375  DD 6E 05    LD L,(IX+5)
7378  DD 66 06    LD H,(IX+6)
737B  11 10 00    LD DE,$0010
737E  19          ADD HL,DE
737F  DD 75 05    LD (IX+5),L
7382  DD 74 06    LD (IX+6),H
7385  DD CB 18 C6 SET 0,(IX+24)
7389  DD 36 0D 1C LD (IX+13),$1C
738D  DD 36 0E 40 LD (IX+14),$40
7391  21 A3 75    LD HL,$75A3
7394  DD CB 18 4E BIT 1,(IX+24)
7398  28 03       JR Z,$739D
739A  21 BB 75    LD HL,$75BB
739D  3A 24 D2    LD A,($D224)
73A0  0F          RRCA
73A1  30 04       JR NC,$73A7
73A3  11 0C 00    LD DE,$000C
73A6  19          ADD HL,DE
73A7  4E          LD C,(HL)
73A8  23          INC HL
73A9  46          LD B,(HL)
73AA  23          INC HL
73AB  EB          EX DE,HL
73AC  DD 6E 02    LD L,(IX+2)
73AF  DD 66 03    LD H,(IX+3)
73B2  09          ADD HL,BC
73B3  22 AB D2    LD ($D2AB),HL
73B6  EB          EX DE,HL
73B7  4E          LD C,(HL)
73B8  23          INC HL
73B9  46          LD B,(HL)
73BA  23          INC HL
73BB  22 AF D2    LD ($D2AF),HL
73BE  DD 6E 05    LD L,(IX+5)
73C1  DD 66 06    LD H,(IX+6)
73C4  09          ADD HL,BC
73C5  22 AD D2    LD ($D2AD),HL
73C8  21 6D 75    LD HL,$756D
73CB  3A 24 D2    LD A,($D224)
73CE  E6 10       AND $10
73D0  28 03       JR Z,$73D5
73D2  21 91 75    LD HL,$7591
73D5  DD 75 0F    LD (IX+15),L
73D8  DD 74 10    LD (IX+16),H
73DB  2A 54 D2    LD HL,($D254)
73DE  22 6D D2    LD ($D26D),HL
73E1  DD 6E 02    LD L,(IX+2)
73E4  DD 66 03    LD H,(IX+3)
73E7  11 90 FF    LD DE,$FF90
73EA  19          ADD HL,DE
73EB  22 6F D2    LD ($D26F),HL
73EE  21 02 00    LD HL,$0002
73F1  22 15 D2    LD ($D215),HL
73F4  CD 28 33    CALL $3328
73F7  DA 9A 74    JP C,$749A
73FA  3A 09 D4    LD A,($D409)
73FD  A7          AND A
73FE  FA 9A 74    JP M,$749A
7401  DD 5E 05    LD E,(IX+5)
7404  DD 56 06    LD D,(IX+6)
7407  2A 02 D4    LD HL,($D402)
740A  A7          AND A
740B  ED 52       SBC HL,DE
740D  38 26       JR C,$7435
740F  DD 6E 02    LD L,(IX+2)
7412  DD 66 03    LD H,(IX+3)
7415  11 10 00    LD DE,$0010
7418  19          ADD HL,DE
7419  11 F2 FF    LD DE,$FFF2
741C  ED 4B FF D3 LD BC,($D3FF)
7420  A7          AND A
7421  ED 42       SBC HL,BC
7423  30 03       JR NC,$7428
7425  11 1D 00    LD DE,$001D
7428  DD 6E 02    LD L,(IX+2)
742B  DD 66 03    LD H,(IX+3)
742E  19          ADD HL,DE
742F  22 FF D3    LD ($D3FF),HL
7432  C3 91 74    JP $7491
7435  2A FF D3    LD HL,($D3FF)
7438  01 08 00    LD BC,$0008
743B  09          ADD HL,BC
743C  4D          LD C,L
743D  44          LD B,H
743E  DD 5E 02    LD E,(IX+2)
7441  DD 56 03    LD D,(IX+3)
7444  A7          AND A
7445  ED 52       SBC HL,DE
7447  D8          RET C
7448  EB          EX DE,HL
7449  11 20 00    LD DE,$0020
744C  19          ADD HL,DE
744D  A7          AND A
744E  ED 42       SBC HL,BC
7450  D8          RET C
7451  79          LD A,C
7452  E6 1F       AND $1F
7454  4F          LD C,A
7455  06 00       LD B,$00
7457  21 4D 75    LD HL,$754D
745A  09          ADD HL,BC
745B  4E          LD C,(HL)
745C  DD 6E 05    LD L,(IX+5)
745F  DD 66 06    LD H,(IX+6)
7462  11 E0 FF    LD DE,$FFE0
7465  19          ADD HL,DE
7466  09          ADD HL,BC
7467  22 02 D4    LD ($D402),HL
746A  3A E9 D2    LD A,($D2E9)
746D  2A E7 D2    LD HL,($D2E7)
7470  22 07 D4    LD ($D407),HL
7473  32 09 D4    LD ($D409),A
7476  21 15 D4    LD HL,$D415
7479  CB FE       SET 7,(HL)
747B  79          LD A,C
747C  FE 03       CP $03
747E  C0          RET NZ
747F  DD 36 0F 7F LD (IX+15),$7F
7483  DD 36 10 75 LD (IX+16),$75
7487  FD CB 06 4E BIT 1,(IY+6)
748B  20 12       JR NZ,$749F
748D  FD CB 06 CE SET 1,(IY+6)
7491  AF          XOR A
7492  6F          LD L,A
7493  67          LD H,A
7494  22 04 D4    LD ($D404),HL
7497  32 06 D4    LD ($D406),A
749A  FD CB 06 4E BIT 1,(IY+6)
749E  C8          RET Z
749F  FD CB 00 AE RES 5,(IY+0)
74A3  DD 7E 12    LD A,(IX+18)
74A6  FE 08       CP $08
74A8  30 14       JR NC,$74BE
74AA  DD 34 11    INC (IX+17)
74AD  DD 7E 11    LD A,(IX+17)
74B0  FE 14       CP $14
74B2  D8          RET C
74B3  DD 36 11 00 LD (IX+17),$00
74B7  CD 76 7A    CALL $7A76
74BA  DD 34 12    INC (IX+18)
74BD  C9          RET
74BE  DD CB 18 4E BIT 1,(IX+24)
74C2  20 0C       JR NZ,$74D0
74C4  3E A0       LD A,$A0
74C6  32 83 D2    LD ($D283),A
74C9  3E 09       LD A,$09
74CB  DF          RST $18
74CC  DD CB 18 CE SET 1,(IX+24)
74D0  AF          XOR A
74D1  DD 77 0F    LD (IX+15),A
74D4  DD 77 10    LD (IX+16),A
74D7  3A 24 D2    LD A,($D224)
74DA  E6 0F       AND $0F
74DC  C0          RET NZ
74DD  CD 88 06    CALL $0688
74E0  E6 01       AND $01
74E2  C6 23       ADD A,$23
74E4  CD F5 74    CALL $74F5
74E7  DD 34 16    INC (IX+22)
74EA  DD 7E 16    LD A,(IX+22)
74ED  FE 0C       CP $0C
74EF  D8          RET C
74F0  DD 36 00 FF LD (IX+0),$FF
74F4  C9          RET

; ==== sub_74F5 (1 caller) ====
74F5  32 17 D2    LD ($D217),A
74F8  CD AF 7C    CALL $7CAF
74FB  D8          RET C
74FC  DD 5E 02    LD E,(IX+2)
74FF  DD 56 03    LD D,(IX+3)
7502  DD 4E 05    LD C,(IX+5)
7505  DD 46 06    LD B,(IX+6)
7508  DD E5       PUSH IX
750A  E5          PUSH HL
750B  DD E1       POP IX
750D  3A 17 D2    LD A,($D217)
7510  DD 77 00    LD (IX+0),A
7513  AF          XOR A
7514  DD 77 16    LD (IX+22),A
7517  DD 77 17    LD (IX+23),A
751A  DD 77 01    LD (IX+1),A
751D  21 08 00    LD HL,$0008
7520  19          ADD HL,DE
7521  DD 75 02    LD (IX+2),L
7524  DD 74 03    LD (IX+3),H
7527  DD 77 04    LD (IX+4),A
752A  21 1A 00    LD HL,$001A
752D  09          ADD HL,BC
752E  DD 75 05    LD (IX+5),L
7531  DD 74 06    LD (IX+6),H
7534  CD 88 06    CALL $0688
7537  DD 77 0A    LD (IX+10),A
753A  CD 88 06    CALL $0688
753D  E6 01       AND $01
753F  3C          INC A
7540  3C          INC A
7541  ED 44       NEG
7543  DD 77 0B    LD (IX+11),A
7546  DD 36 0C FF LD (IX+12),$FF
754A  DD E1       POP IX
754C  C9          RET
754D  .byte 15 12 11 10 10 0F 0E 0D 03 03 03 03 03 03 03 03 ; ................
755D  .byte 03 03 03 03 03 03 03 03 0D 0E 0F 10 10 11 12 15 ; ................
756D  .byte 00 02 04 06 FF FF 20 22 24 26 FF FF 40 42 44 46 ; ...... "$&..@BDF
757D  .byte FF FF 00 08 0A 06 FF FF 20 22 24 26 FF FF 40 42 ; ........ "$&..@B
758D  .byte 44 46 FF FF 00 68 6A 06 FF FF 20 22 24 26 FF FF ; DF...hj... "$&..
759D  .byte 40 42 44 46 FF FF 00 00 30 00 60 19 62 19 61 19 ; @BDF....0.`.b.a.
75AD  .byte 63 19 10 00 30 00 64 19 66 19 65 19 67 19 00 00 ; c...0.d.f.e.g...
75BD  .byte 20 00 00 00 00 00 49 19 4B 19 10 00 20 00 00 00 ;  .....I.K... ...
75CD  .byte 00 00 4D 19 4F 19 DD CB 18 AE DD 36 0D 0C DD 36 ; ..M.O......6...6
75DD  .byte 0E 10 DD CB 18 7E 28 0C DD 36 0A 00 DD 36 0B FD ; .....~(..6...6..
75ED  .byte DD 36 0C FF 11 12 00 3A D5 D2 FE 03 20 03 11 38 ; .6.....:.... ..8
75FD  .byte 00 DD 6E 0A DD 66 0B DD 7E 0C 19 CE 00 4F FA 18 ; ..n..f..~....O..
760D  .byte 76 7C FE 02 38 05 21 00 02 0E 00 DD 75 0A DD 74 ; v|..8.!.....u..t
761D  .byte 0B DD 71 0C 21 00 FE 3A D5 D2 FE 03 20 03 21 80 ; ..q.!..:.... .!.
762D  .byte FE DD 75 07 DD 74 08 DD 36 09 FF 01 68 76 3A D5 ; ..u..t..6...hv:.
763D  .byte D2 A7 28 0A 01 6D 76 FE 03 20 03 01 72 76 11 77 ; ..(..mv.. ..rv.w
764D  .byte 76 CD 75 7C DD 6E 02 DD 66 03 11 E0 FF 19 ED 5B ; v.u|.n..f......[
765D  .byte 54 D2 A7 ED 52 D0 DD 36 00 FF C9 00 02 01 02 FF ; T...R..6........
766D  .byte 02 04 03 04 FF 04 03 05 03 FF 10 12 FF FF FF FF ; ................
767D  .byte FF FF FF FF FF FF FF FF FF FF FF FF 6E 0E FF FF ; ............n...
768D  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF FF 28 2A ; ..............(*
769D  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF ; ................
76AD  .byte 2C 2E FF FF FF FF FF FF FF FF FF FF FF FF FF FF ; ,...............
76BD  .byte FF FF 30 32 FF FF FF FF FF FF FF FF FF FF FF FF ; ..02............
76CD  .byte FF FF FF FF 50 52 FF FF FF FF FF DD CB 18 AE DD ; ....PR..........
76DD  .byte 36 0D 0C DD 36 0E 20 21 9F 77 3A D5 D2 A7 28 0F ; 6...6. !.w:...(.
76ED  .byte 21 BA 77 3D 28 09 21 D5 77 3D 28 03 21 F0 77 DD ; !.w=(.!.w=(.!.w.
76FD  .byte 75 0F DD 74 10 DD CB 18 7E 28 50 AF DD 77 0A DD ; u..t....~(P..w..
770D  .byte 36 0B 01 DD 77 0C DD 77 07 DD 77 08 DD 77 09 21 ; 6...w..w..w..w.!
771D  .byte 91 77 3A D5 D2 4F A7 28 0F 21 AC 77 3D 28 09 21 ; .w:..O.(.!.w=(.!
772D  .byte C7 77 3D 28 03 21 E2 77 DD 75 0F DD 74 10 DD 34 ; .w=(.!.w.u..t..4
773D  .byte 11 DD 7E 11 FE 08 D8 21 FC FF 79 A7 28 03 21 FE ; ..~....!..y.(.!.
774D  .byte FF DD 36 0A 00 DD 75 0B DD 74 0C DD 6E 0A DD 66 ; ..6...u..t..n..f
775D  .byte 0B DD 7E 0C 11 28 00 19 CE 00 4F FA 75 77 7C FE ; ..~..(....O.uw|.
776D  .byte 02 38 05 21 00 02 0E 00 DD 75 0A DD 74 0B DD 71 ; .8.!.....u..t..q
777D  .byte 0C DD 36 07 80 DD 36 08 FE DD 36 09 FF DD 36 11 ; ..6...6...6...6.
778D  .byte 00 C3 51 76 70 72 FF FF FF FF 54 56 FF FF FF FF ; ..Qvpr....TV....
779D  .byte FF FF 5C 5E FF FF FF FF 58 5A FF FF FF FF FF FE ; ..\^....XZ......
77AD  .byte FF FF FF FF FF 34 36 FF FF FF FF FF FF FE FF FF ; .....46.........
77BD  .byte FF FF FF 38 3A FF FF FF FF FF FE FF FF FF FF FF ; ...8:...........
77CD  .byte 3C 3E FF FF FF FF FF FF FE FF FF FF FF FF 1C 1E ; <>..............
77DD  .byte FF FF FF FF FF FE FF FF FF FF FF 14 16 FF FF FF ; ................
77ED  .byte FF FF FF FE FF FF FF FF FF 18 1A FF FF FF FF FF ; ................

; ==== sub_77FD (2 callers) ====
77FD  3A ED D2    LD A,($D2ED)
7800  FE 08       CP $08
7802  D2 7D 78    JP NC,$787D
7805  3A B1 D2    LD A,($D2B1)
7808  A7          AND A
7809  C2 5D 78    JP NZ,$785D
780C  21 08 0C    LD HL,$0C08
780F  22 15 D2    LD ($D215),HL
7812  CD 28 33    CALL $3328
7815  D8          RET C
7816  FD CB 05 46 BIT 0,(IY+5)
781A  C0          RET NZ
781B  3A 15 D4    LD A,($D415)
781E  0F          RRCA
781F  38 05       JR C,$7826
7821  E6 02       AND $02
7823  CA D9 2F    JP Z,$2FD9
7826  11 01 00    LD DE,$0001
7829  2A 07 D4    LD HL,($D407)
782C  7D          LD A,L
782D  2F          CPL
782E  6F          LD L,A
782F  7C          LD A,H
7830  2F          CPL
7831  67          LD H,A
7832  3A 09 D4    LD A,($D409)
7835  2F          CPL
7836  19          ADD HL,DE
7837  CE 00       ADC A,$00
7839  22 07 D4    LD ($D407),HL
783C  32 09 D4    LD ($D409),A
783F  AF          XOR A
7840  6F          LD L,A
7841  67          LD H,A
7842  22 04 D4    LD ($D404),HL
7845  32 06 D4    LD ($D406),A
7848  21 B1 D2    LD HL,$D2B1
784B  36 18       LD (HL),$18
784D  23          INC HL
784E  36 8F       LD (HL),$8F
7850  23          INC HL
7851  36 FF       LD (HL),$FF
7853  23          INC HL
7854  36 0F       LD (HL),$0F
7856  21 ED D2    LD HL,$D2ED
7859  34          INC (HL)
785A  3E 01       LD A,$01
785C  EF          RST $28
785D  2A 17 D2    LD HL,($D217)
7860  11 5E 79    LD DE,$795E
7863  19          ADD HL,DE
7864  DD CB 18 4E BIT 1,(IX+24)
7868  28 04       JR Z,$786E
786A  11 12 00    LD DE,$0012
786D  19          ADD HL,DE
786E  DD 75 0F    LD (IX+15),L
7871  DD 74 10    LD (IX+16),H
7874  21 EE D2    LD HL,$D2EE
7877  36 18       LD (HL),$18
7879  23          INC HL
787A  36 00       LD (HL),$00
787C  C9          RET
787D  AF          XOR A
787E  DD 77 07    LD (IX+7),A
7881  DD 77 08    LD (IX+8),A
7884  DD 77 09    LD (IX+9),A
7887  DD 77 0A    LD (IX+10),A
788A  DD 77 0B    LD (IX+11),A
788D  DD 77 0C    LD (IX+12),A
7890  11 24 00    LD DE,$0024
7893  2A 17 D2    LD HL,($D217)
7896  DD CB 18 4E BIT 1,(IX+24)
789A  28 03       JR Z,$789F
789C  11 36 00    LD DE,$0036
789F  19          ADD HL,DE
78A0  11 5E 79    LD DE,$795E
78A3  19          ADD HL,DE
78A4  DD 75 0F    LD (IX+15),L
78A7  DD 74 10    LD (IX+16),H
78AA  21 EF D2    LD HL,$D2EF
78AD  7E          LD A,(HL)
78AE  FE 0A       CP $0A
78B0  D2 BE 78    JP NC,$78BE
78B3  2B          DEC HL
78B4  35          DEC (HL)
78B5  C0          RET NZ
78B6  36 18       LD (HL),$18
78B8  23          INC HL
78B9  34          INC (HL)
78BA  CD 76 7A    CALL $7A76
78BD  C9          RET
78BE  3A EF D2    LD A,($D2EF)
78C1  FE 3A       CP $3A
78C3  30 18       JR NC,$78DD
78C5  DD 6E 04    LD L,(IX+4)
78C8  DD 66 05    LD H,(IX+5)
78CB  DD 7E 06    LD A,(IX+6)
78CE  11 20 00    LD DE,$0020
78D1  19          ADD HL,DE
78D2  CE 00       ADC A,$00
78D4  DD 75 04    LD (IX+4),L
78D7  DD 74 05    LD (IX+5),H
78DA  DD 77 06    LD (IX+6),A
78DD  21 EF D2    LD HL,$D2EF
78E0  7E          LD A,(HL)
78E1  FE 5A       CP $5A
78E3  30 02       JR NC,$78E7
78E5  34          INC (HL)
78E6  C9          RET
78E7  20 13       JR NZ,$78FC
78E9  36 5B       LD (HL),$5B
78EB  3A F7 D2    LD A,($D2F7)
78EE  DF          RST $18
78EF  FD 7E 0A    LD A,(IY+10)
78F2  FD CB 00 86 RES 0,(IY+0)
78F6  CD 27 03    CALL $0327
78F9  FD 77 0A    LD (IY+10),A
78FC  DD 36 07 00 LD (IX+7),$00
7900  DD 36 08 03 LD (IX+8),$03
7904  DD 36 09 00 LD (IX+9),$00
7908  DD 36 0A 60 LD (IX+10),$60
790C  DD 36 0B FF LD (IX+11),$FF
7910  DD 36 0C FF LD (IX+12),$FF
7914  DD 36 0F 5E LD (IX+15),$5E
7918  DD 36 10 79 LD (IX+16),$79
791C  DD 6E 02    LD L,(IX+2)
791F  DD 66 03    LD H,(IX+3)
7922  ED 5B 54 D2 LD DE,($D254)
7926  14          INC D
7927  A7          AND A
7928  ED 52       SBC HL,DE
792A  D8          RET C
792B  DD 36 00 FF LD (IX+0),$FF
792F  21 00 20    LD HL,$2000
7932  22 6F D2    LD ($D26F),HL
7935  21 00 00    LD HL,$0000
7938  22 75 D2    LD ($D275),HL
793B  FD CB 00 EE SET 5,(IY+0)
793F  FD CB 02 C6 SET 0,(IY+2)
7943  FD CB 02 8E RES 1,(IY+2)
7947  3A 38 D2    LD A,($D238)
794A  FE 0B       CP $0B
794C  20 04       JR NZ,$7952
794E  FD CB 09 CE SET 1,(IY+9)
7952  21 D0 DA    LD HL,$DAD0
7955  11 00 20    LD DE,$2000
7958  3E 0C       LD A,$0C
795A  CD 06 04    CALL $0406
795D  C9          RET
795E  .byte 2A 2C 2E 30 32 FF 4A 4C 4E 50 52 FF 6A 6C 6E 70 ; *,.02.JLNPR.jlnp
796E  .byte 72 FF 20 10 12 14 28 FF 40 42 44 46 48 FF 60 62 ; r. ...(.@BDFH.`b
797E  .byte 64 66 68 FF 2A 16 18 1A 32 FF 4A 4C 4E 50 52 FF ; dfh.*...2.JLNPR.
798E  .byte 6A 6C 6E 70 72 FF 20 3A 3C 3E 28 FF 40 42 44 46 ; jlnpr. :<>(.@BDF
799E  .byte 48 FF 60 62 64 66 68 FF 2A 34 36 38 32 FF 4A 4C ; H.`bdfh.*4682.JL
79AE  .byte 4E 50 52 FF 6A 6C 6E 70 72 FF 20 10 12 14 28 FF ; NPR.jlnpr. ...(.
79BE  .byte 40 42 44 46 48 FF 60 54 56 66 68 FF 2A 16 18 1A ; @BDFH.`TVfh.*...
79CE  .byte 32 FF 4A 4C 4E 50 52 FF 6A 5A 5C 70 72 FF 20 3A ; 2.JLNPR.jZ\pr. :
79DE  .byte 3C 3E 28 FF 40 42 44 46 48 FF 60 54 56 66 68 FF ; <>(.@BDFH.`TVfh.
79EE  .byte 2A 34 36 38 32 FF 4A 4C 4E 50 52 FF 6A 5A 5C 70 ; *4682.JLNPR.jZ\p
79FE  .byte 72 FF 20 06 08 0A 28 FF 40 42 44 46 48 FF 60 62 ; r. ...(.@BDFH.`b
7A0E  .byte 64 66 68 FF 20 06 08 0A 28 FF 40 42 44 46 48 FF ; dfh. ...(.@BDFH.
7A1E  .byte 60 62 64 66 68 FF 0E 10 12 14 16 FF 40 42 44 46 ; `bdfh.......@BDF
7A2E  .byte 48 FF 60 62 64 66 68 FF                         ; H.`bdfh.

; ==== sub_7A36 (2 callers) ====
7A36  DD 7E 07    LD A,(IX+7)
7A39  DD B6 08    OR (IX+8)
7A3C  C8          RET Z
7A3D  3A 24 D2    LD A,($D224)
7A40  CB 47       BIT 0,A
7A42  C0          RET NZ
7A43  E6 02       AND $02
7A45  DD 6E 02    LD L,(IX+2)
7A48  DD 66 03    LD H,(IX+3)
7A4B  22 0F D2    LD ($D20F),HL
7A4E  DD 6E 05    LD L,(IX+5)
7A51  DD 66 06    LD H,(IX+6)
7A54  22 11 D2    LD ($D211),HL
7A57  21 F8 FF    LD HL,$FFF8
7A5A  11 10 00    LD DE,$0010
7A5D  0E 04       LD C,$04
7A5F  DD CB 09 7E BIT 7,(IX+9)
7A63  28 05       JR Z,$7A6A
7A65  21 28 00    LD HL,$0028
7A68  0E 00       LD C,$00
7A6A  22 13 D2    LD ($D213),HL
7A6D  ED 53 15 D2 LD ($D215),DE
7A71  81          ADD A,C
7A72  CD 5D 2F    CALL $2F5D
7A75  C9          RET

; ==== sub_7A76 (2 callers) ====
7A76  CD AF 7C    CALL $7CAF
7A79  D8          RET C
7A7A  E5          PUSH HL
7A7B  CD 88 06    CALL $0688
7A7E  E6 1F       AND $1F
7A80  6F          LD L,A
7A81  26 00       LD H,$00
7A83  22 0F D2    LD ($D20F),HL
7A86  CD 88 06    CALL $0688
7A89  E6 1F       AND $1F
7A8B  6F          LD L,A
7A8C  26 00       LD H,$00
7A8E  22 11 D2    LD ($D211),HL
7A91  E1          POP HL
7A92  DD 5E 02    LD E,(IX+2)
7A95  DD 56 03    LD D,(IX+3)
7A98  DD 4E 05    LD C,(IX+5)
7A9B  DD 46 06    LD B,(IX+6)
7A9E  DD E5       PUSH IX
7AA0  E5          PUSH HL
7AA1  DD E1       POP IX
7AA3  AF          XOR A
7AA4  DD 36 00 0A LD (IX+0),$0A
7AA8  DD 77 01    LD (IX+1),A
7AAB  2A 0F D2    LD HL,($D20F)
7AAE  19          ADD HL,DE
7AAF  DD 75 02    LD (IX+2),L
7AB2  DD 74 03    LD (IX+3),H
7AB5  DD 77 04    LD (IX+4),A
7AB8  2A 11 D2    LD HL,($D211)
7ABB  09          ADD HL,BC
7ABC  DD 75 05    LD (IX+5),L
7ABF  DD 74 06    LD (IX+6),H
7AC2  DD 77 11    LD (IX+17),A
7AC5  DD 77 16    LD (IX+22),A
7AC8  DD 77 17    LD (IX+23),A
7ACB  DD 77 07    LD (IX+7),A
7ACE  DD 77 08    LD (IX+8),A
7AD1  DD 77 09    LD (IX+9),A
7AD4  DD 77 0A    LD (IX+10),A
7AD7  DD 77 0B    LD (IX+11),A
7ADA  DD 77 0C    LD (IX+12),A
7ADD  DD E1       POP IX
7ADF  3E 01       LD A,$01
7AE1  EF          RST $28
7AE2  C9          RET
7AE3  .byte DD CB 18 EE DD 36 0D 40 DD 36 0E 40 21 00 00 22 ; .....6.@.6.@!.."
7AF3  .byte 15 D2 CD 28 33 D8 FD CB 06 76 C0 3A 15 D4 E6 80 ; ...(3....v.:....
7B03  .byte C8 21 FB FF AF 32 07 D4 22 08 D4 21 03 00 AF 32 ; .!...2.."..!...2
7B13  .byte 04 D4 22 05 D4 21 15 D4 CB 8E FD CB 06 F6 FD 36 ; .."..!.........6
7B23  .byte 03 FF 3E 11 EF C9 DD CB 18 EE DD CB 18 46 20 0C ; ..>..........F .
7B33  .byte DD 36 11 32 DD 36 12 00 DD CB 18 C6 01 00 00 DD ; .6.2.6..........
7B43  .byte 6E 02 DD 66 03 22 AB D2 DD 6E 05 DD 66 06 3A 24 ; n..f."...n..f.:$
7B53  .byte D2 0F 30 05 11 10 00 19 03 22 AD D2 DD 7E 12 87 ; ..0......"...~..
7B63  .byte 87 5F 16 00 21 C1 7B 19 E5 09 7E 87 87 87 5F 16 ; ._..!.{...~..._.
7B73  .byte 00 21 99 7B 19 22 AF D2 E1 23 23 3A 24 D2 0F D8 ; .!.{."...##:$...
7B83  .byte DD 35 11 C0 7E DD 77 11 DD 34 12 DD 7E 12 FE 04 ; .5..~.w..4..~...
7B93  .byte D8 DD 36 12 00 C9 00 00 00 00 00 00 00 00 F0 00 ; ..6.............
7BA3  .byte F1 00 E2 00 F2 00 00 00 00 00 F0 00 F1 00 E2 00 ; ................
7BB3  .byte F2 00 2E 00 2F 00 2E 00 2F 00 2E 00 2F 00 00 01 ; ..../.../.../...
7BC3  .byte 08 00 02 03 78 00 01 04 08 00 02 03 78 00 DD CB ; ....x.......x...
7BD3  .byte 18 EE FD CB 09 C6 3A 24 D2 E6 01 CA FE 7B DD 7E ; ......:$.....{.~
7BE3  .byte 12 4F 87 81 4F 06 00 21 53 7C 09 5E 23 56 23 7E ; .O..O..!S|.^#V#~
7BF3  .byte DD 73 0F DD 72 10 32 FD D2 18 06 DD 77          ; .s..r.2.....w

; ==== sub_7C00 (1 caller) ====
7C00  0F          RRCA
7C01  DD 77 10    LD (IX+16),A
7C04  DD 6E 0A    LD L,(IX+10)
7C07  DD 66 0B    LD H,(IX+11)
7C0A  DD 7E 0C    LD A,(IX+12)
7C0D  11 20 00    LD DE,$0020
7C10  19          ADD HL,DE
7C11  CE 00       ADC A,$00
7C13  DD 75 0A    LD (IX+10),L
7C16  DD 74 0B    LD (IX+11),H
7C19  DD 77 0C    LD (IX+12),A
7C1C  DD 5E 05    LD E,(IX+5)
7C1F  DD 56 06    LD D,(IX+6)
7C22  2A 57 D2    LD HL,($D257)
7C25  24          INC H
7C26  AF          XOR A
7C27  ED 52       SBC HL,DE
7C29  30 09       JR NC,$7C34
7C2B  DD 36 00 FF LD (IX+0),$FF
7C2F  FD CB 09 86 RES 0,(IY+9)
7C33  C9          RET
7C34  DD 77 07    LD (IX+7),A
7C37  DD 77 08    LD (IX+8),A
7C3A  DD 77 09    LD (IX+9),A
7C3D  DD 35 11    DEC (IX+17)
7C40  C0          RET NZ
7C41  DD 36 11 04 LD (IX+17),$04
7C45  DD 34 12    INC (IX+18)
7C48  DD 7E 12    LD A,(IX+18)
7C4B  FE 06       CP $06
7C4D  D8          RET C
7C4E  DD 36 12 00 LD (IX+18),$00
7C52  C9          RET
7C53  .byte 65 7C 01 6D 7C 01 65 7C 02 6D 7C 02 65 7C 03 6D ; e|.m|.e|.m|.e|.m
7C63  .byte 7C 03 B4 B6 FF FF FF FF FF FF B8 BA FF FF FF FF ; |...............
7C73  .byte FF FF                                           ; ..

; ==== sub_7C75 (3 callers) ====
7C75  DD 6E 17    LD L,(IX+23)
7C78  26 00       LD H,$00
7C7A  09          ADD HL,BC
7C7B  7E          LD A,(HL)
7C7C  FE FF       CP $FF
7C7E  20 08       JR NZ,$7C88
7C80  2E 00       LD L,$00
7C82  DD 75 17    LD (IX+23),L
7C85  C3 78 7C    JP $7C78
7C88  23          INC HL
7C89  E5          PUSH HL
7C8A  6F          LD L,A
7C8B  26 00       LD H,$00
7C8D  29          ADD HL,HL
7C8E  4D          LD C,L
7C8F  44          LD B,H
7C90  29          ADD HL,HL
7C91  29          ADD HL,HL
7C92  29          ADD HL,HL
7C93  09          ADD HL,BC
7C94  19          ADD HL,DE
7C95  DD 75 0F    LD (IX+15),L
7C98  DD 74 10    LD (IX+16),H
7C9B  E1          POP HL
7C9C  DD 34 16    INC (IX+22)
7C9F  7E          LD A,(HL)
7CA0  DD BE 16    CP (IX+22)
7CA3  D0          RET NC
7CA4  DD 36 16 00 LD (IX+22),$00
7CA8  DD 34 17    INC (IX+23)
7CAB  DD 34 17    INC (IX+23)
7CAE  C9          RET

; ==== sub_7CAF (3 callers) ====
7CAF  21 17 D4    LD HL,$D417
7CB2  11 1A 00    LD DE,$001A
7CB5  06 1F       LD B,$1F
7CB7  7E          LD A,(HL)
7CB8  FE FF       CP $FF
7CBA  C8          RET Z
7CBB  19          ADD HL,DE
7CBC  10 F9       DJNZ $7CB7
7CBE  37          SCF
7CBF  C9          RET

; ==== sub_7CC0 (1 caller) ====
7CC0  22 75 D2    LD ($D275),HL
7CC3  ED 53 77 D2 LD ($D277),DE
7CC7  2A 54 D2    LD HL,($D254)
7CCA  22 6D D2    LD ($D26D),HL
7CCD  22 6F D2    LD ($D26F),HL
7CD0  2A 57 D2    LD HL,($D257)
7CD3  22 71 D2    LD ($D271),HL
7CD6  22 73 D2    LD ($D273),HL
7CD9  C9          RET
7CDA  .byte 2A 75 D2 ED 5B 54 D2 A7 ED 52 C0 2A 77 D2 ED 5B ; *u..[T...R.*w..[
7CEA  .byte 57 D2 A7 ED 52 C0 FD CB 00 AE C9 DD 6E 04 DD 66 ; W...R.......n..f
7CFA  .byte 05 AF CB 7A 28 01 3D 19 DD 8E 06 6C 67 09 3A 0B ; ...z(.=....lg.:.
7D0A  .byte D4 4F AF 47 ED 42 22 02 D4 3A E9 D2 2A E7 D2 22 ; .O.G.B"..:..*.."
7D1A  .byte 07 D4 32 09 D4 21 15 D4 CB FE C9 DD CB 18 EE DD ; ..2..!..........
7D2A  .byte 36 0D 08 DD 36 0E 0C DD 7E 14 A7 28 0B DD 35 14 ; 6...6...~..(..5.
7D3A  .byte AF DD 77 0F DD 77 10 C9 DD CB 18 46 20 43 DD CB ; ..w..w.....F C..
7D4A  .byte 18 4E 20 24 DD 6E 05 DD 66 06 11 F4 FF 19 DD 75 ; .N $.n..f......u
7D5A  .byte 12 DD 74 13 DD 6E 02 DD 66 03 11 08 00 19 DD 75 ; ..t..n..f......u
7D6A  .byte 02 DD 74 03 DD CB 18 CE DD 36 0A 00 DD 36 0B FC ; ..t......6...6..
7D7A  .byte DD 36 0C FF DD CB 18 C6 3E 12 EF DD 36 11 03 18 ; .6......>...6...
7D8A  .byte 53 DD 6E 0A DD 66 0B DD 7E 0C 11 10 00 19 CE 00 ; S.n..f..~.......
7D9A  .byte EB A7 FA AA 7D 21 00 04 A7 ED 52 30 03 11 00 04 ; ....}!....R0....
7DAA  .byte DD 73 0A DD 72 0B DD 77 0C DD 5E 12 DD 56 13 DD ; .s..r..w..^..V..
7DBA  .byte 6E 05 DD 66 06 AF ED 52 38 1A DD 77 04 DD 73 05 ; n..f...R8..w..s.
7DCA  .byte DD 72 06 DD 77 0A DD 77 0B DD 77 0C DD 36 14 1E ; .r..w..w..w..6..
7DDA  .byte DD CB 18 86 11 10 7E 01 0B 7E CD 75 7C DD 7E 11 ; ......~..~.u|.~.
7DEA  .byte A7 28 0B DD 35 11 DD 36 0F 26 DD 36 10 7E 21 04 ; .(..5..6.&.6.~!.
7DFA  .byte 02 22 15 D2 CD 28 33 21 00 00 22 0F D2 D4 C1 2F ; ."...(3!.."..../
7E0A  .byte C9 00 04 01 04 FF 60 62 FF FF FF FF FF FF FF FF ; ......`b........
7E1A  .byte FF FF FF FF FF FF FF FF 64 66 FF FF FF FF FF FF ; ........df......
7E2A  .byte 68 6A FF FF FF FF FF DD CB 18 EE DD 36 0D 0C DD ; hj..........6...
7E3A  .byte 36 0E 10 DD 36 0F B6 DD 36 10 7E DD 6E 05 DD 66 ; 6...6...6.~.n..f
7E4A  .byte 06 22 03 D3 DD CB 18 46 20 1F DD 7E 05 DD 77 12 ; .".....F ..~..w.
7E5A  .byte DD 7E 06 DD 77 13 DD 36 14 C0 DD CB 18 C6 DD 36 ; .~..w..6.......6
7E6A  .byte 0A 80 AF DD 77 0B DD 77 0C CD 9A 7E 3A 24 D2 E6 ; ....w..w...~:$..
7E7A  .byte 03 C0 DD 34 11 DD 7E 11 DD BE 14 D8 AF DD 77 11 ; ...4..~.......w.
7E8A  .byte DD 77 04 DD 7E 12 DD 77 05 DD 7E 13 DD 77 06 C9 ; .w..~..w..~..w..
7E9A  .byte 3A 09 D4 A7 F8 21 06 08 22 15 D2 CD 28 33 D8 01 ; :....!.."...(3..
7EAA  .byte 10 00 DD 5E 0A DD 56 0B CD F5 7C C9 FE FF FF FF ; ...^..V...|.....
7EBA  .byte FF FF 18 1A FF FF FF FF 28 2E FF FF FF FF DD CB ; ........(.......
7ECA  .byte 18 EE DD 36 0D 1A DD 36 0E 10 DD 36 0F EF DD 36 ; ...6...6...6...6
7EDA  .byte 10 7E 2A 03 D3 11 28 00 A7 ED 52 DD 75 05 DD 74 ; .~*...(...R.u..t
7EEA  .byte 06 CD 9A 7E C9 FE FF FF FF FF FF 6C 6E 6E 48 FF ; ...~.......lnnH.
7EFA  .byte FF FF DD CB 18 EE DD 36 0D 0A DD 36 0E 10 DD CB ; .......6...6....
7F0A  .byte 18 46 20 14 DD 6E 05 DD 66 06 11 E8 FF 19 DD 75 ; .F ..n..f......u
7F1A  .byte 05 DD 74 06 DD CB 18 C6 DD 36 0A 40 AF DD 77 0B ; ..t......6.@..w.
7F2A  .byte DD 77 0C DD 7E 11 FE 14 38 0C DD 36 0A C0 DD 36 ; .w..~...8..6...6
7F3A  .byte 0B FF DD 36 0C FF 3A 09 D4 A7 FA 1B 80 21 06 08 ; ...6..:......!..
7F4A  .byte 22 15 D2 CD 28 33 DA 1B 80 01 10 00 DD 5E 0A DD ; "...(3.......^..
7F5A  .byte 56 0B CD F5 7C 2A 04 D4 7D B4 28 29 01 12 00 CB ; V...|*..}.()....
7F6A  .byte 7C 28 03 01 FE FF 11 00 00 CD D5 30 5E 16 00 3A ; |(.........0^..:
7F7A  .byte D5 D2 87 4F 42 21 3D 34 09 7E 23 66 6F 19 7E E6 ; ...OB!=4.~#fo.~.
7F8A  .byte 3F 7A 5A 20 0C 3A 04 D4 ED 5B 05 D4 CB 2A CB 1B ; ?zZ .:...[...*..
7F9A  .byte 1F DD 6E 02 DD 66 03 DD 86 01 ED 5A DD 77 01 DD ; ..n..f.....Z.w..
7FAA  .byte 75 02 DD 74 03 32 FE D3 22 FF D3 ED 5B 04 D4 CB ; u..t.2.."...[...
7FBA  .byte 7A 28 07 7B 2F 5F 7A 2F 57 13 DD 6E 12 DD 66 13 ; z(.{/_z/W..n..f.
7FCA  .byte 19 7C FE 09 38 31 D6 09 67 18 2C 00 00 00 00 00 ; .|..81..g.,.....
7FDA  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
7FEA  .byte 00 00 00 00 00 00 54 4D 52 20 53 45 47 41 00 00 ; ......TMR SEGA..
7FFA  .byte 00 00 08 24 00 60 00 DD 75 12 DD 74 13 5F 16 00 ; ...$.`..u..t._..
800A  .byte 21 31 80 19 5E 21 3A 80 19 DD 75 0F DD 74 10 18 ; !1..^!:...u..t..
801A  .byte 08 DD 36 0F 3A DD 36 10 80 DD 34 11 DD 7E 11 FE ; ..6.:.6...4..~..
802A  .byte 28 D8 DD 36 11 00 C9 00 00 00 12 12 12 24 24 24 ; (..6.........$$$
803A  .byte FE FF FF FF FF FF 3A 3C FF FF FF FF FF FF FF FF ; ......:<........
804A  .byte FF FF FE FF FF FF FF FF 36 38 FF FF FF FF FF FF ; ........68......
805A  .byte FF FF FF FF FE FF FF FF FF FF 4C 4E FF FF FF FF ; ..........LN....
806A  .byte FF DD CB 18 EE DD 36 0D 20 DD 36 0E 1C DD CB 18 ; ......6. .6.....
807A  .byte 46 20 48 2A 02 D4 11 E0 00 A7 ED 52 D0 3A 15 D4 ; F H*.......R.:..
808A  .byte 07 D0 21 B7 A8 11 00 20 3E 09 CD 06 04 3E 11 32 ; ..!.... >....>.2
809A  .byte 2D D2 3E 0B DF AF 32 ED D2 2A 54 D2 22 6D D2 22 ; -.>...2..*T."m."
80AA  .byte 6F D2 2A 57 D2 22 71 D2 22 73 D2 21 E0 01 22 75 ; o.*W."q."s.!.."u
80BA  .byte D2 21 50 00 22 77 D2 DD CB 18 C6 CD DA 7C DD CB ; .!P."w.......|..
80CA  .byte 11 46 20 2E DD 36 0F 09 DD 36 10 82 DD 36 0A 80 ; .F ..6...6...6..
80DA  .byte DD 36 0B 00 DD 36 0C 00 DD 6E 05 DD 66 06 11 70 ; .6...6...n..f..p
80EA  .byte 00 AF ED 52 D8 DD 77 0A DD 77 0B DD 77 0C DD CB ; ...R..w..w..w...
80FA  .byte 11 C6 DD 7E 12 A7 C2 5F 81 DD 6E 02 DD 66 03 DD ; ...~..._..n..f..
810A  .byte CB 11 4E 20 28 DD 36 0F 09 DD 36 10 82 DD CB 18 ; ..N (.6...6.....
811A  .byte 8E DD 36 07 80 DD 36 08 FF DD 36 09 FF 11 1C 02 ; ..6...6...6.....
812A  .byte A7 ED 52 D2 FC 81 DD 36 12 57 C3 FC 81 DD 36 0F ; ..R....6.W....6.
813A  .byte 1B DD 36 10 82 DD CB 18 CE DD 36 07 80 DD 36 08 ; ..6.......6...6.
814A  .byte 00 DD 36 09 00 11 88 02 A7 ED 52 DA FC 81 DD 36 ; ..6.......R....6
815A  .byte 12 57 C3 FC 81 AF DD 77 07 DD 77 08 DD 77 09 21 ; .W.....w..w..w.!
816A  .byte 01 00 DD 35 12 28 12 DD 7E 12 FE 38 30 0E 21 FF ; ...5.(..~..80.!.
817A  .byte FF FE 20 38 07 FE 34 28 0F 21 00 00 DD 36 0A 00 ; .. 8..4(.!...6..
818A  .byte DD 75 0B DD 74 0C 18 6A DD 7E 11 EE 02 DD 77 11 ; .u..t..j.~....w.
819A  .byte 3A ED D2 FE 08 30 5B CD AF 7C D8 DD 5E 02 DD 56 ; :....0[..|..^..V
81AA  .byte 03 DD 4E 05 DD 46 06 DD E5 E5 DD E1 DD 36 00 2B ; ..N..F.......6.+
81BA  .byte AF DD 77 01 21 0B 00 19 DD 75 02 DD 74 03 DD 77 ; ..w.!....u..t..w
81CA  .byte 04 21 30 00 09 DD 75 05 DD 74 06 DD 77 07 DD 77 ; .!0...u..t..w..w
81DA  .byte 08 DD 77 09 DD 77 0A DD 77 0B DD 77 0C DD 77 11 ; ..w..w..w..w..w.
81EA  .byte DD 77 16 DD 77 17 CD 88 06 E6 3F C6 64 DD 77 12 ; .w..w.....?.d.w.
81FA  .byte DD E1 21 5A 00                                  ; ..!Z.

; ==== sub_81FF (1 caller) ====
81FF  22 17 D2    LD ($D217),HL
8202  CD FD 77    CALL $77FD
8205  CD 36 7A    CALL $7A36
8208  C9          RET
8209  .byte 20 22 24 26 28 FF 40 42 44 46 48 FF 60 54 56 58 ;  "$&(.@BDFH.`TVX
8219  .byte 68 FF 2A 2C 2E 30 32 FF 4A 4C 4E 50 52 FF 6A 5A ; h.*,.02.JLNPR.jZ
8229  .byte 5C 5E 72 FF DD CB 18 AE DD 36 0D 0C DD 36 0E 10 ; \^r......6...6..
8239  .byte 21 02 02 22 15 D2 CD 28 33 D4 D9 2F DD 6E 07 DD ; !.."...(3../.n..
8249  .byte 66 08 DD 7E 09 11 02 00 0E 00 A7 FA 5B 82 0D 11 ; f..~........[...
8259  .byte FE FF 19 89 DD 75 07 DD 74 08 DD 77 09 DD 6E 0A ; .....u..t..w..n.
8269  .byte DD 66 0B DD 7E 0C 11 20 00 19 CE 00 4F 7C FE 03 ; .f..~.. ....O|..
8279  .byte 38 05 21 00 03 0E 00 DD 75 0A DD 74 0B DD 71 0C ; 8.!.....u..t..q.
8289  .byte 3A 24 D2 E6 01 DD 86 11 DD 77 11 DD 7E 11 DD BE ; :$.......w..~...
8299  .byte 12 30 0A 01 D6 82 11 E2 82 CD 75 7C C9 20 0D 3A ; .0........u|. .:
82A9  .byte 24 D2 E6 01 C8 DD 36 16 00 3E 01 EF AF DD 77 07 ; $.....6..>....w.
82B9  .byte DD 77 08 DD 77 09 01 DB 82 11 84 A3 CD 75 7C DD ; .w..w........u|.
82C9  .byte 7E 12 C6 12 DD BE 11 D0 DD 36 00 FF C9 00 04 01 ; ~........6......
82D9  .byte 04 FF 01 0C 02 0C 03 0C FF 08 0A FF FF FF FF FF ; ................
82E9  .byte FF FF FF FF FF FF FF FF FF FF FF 0C 0E FF FF FF ; ................
82F9  .byte FF FF DD 36 0D 10                               ; ...6..
82FF  DD 36 0E 0E LD (IX+14),$0E
8303  21 08 06    LD HL,$0608
8306  22 15 D2    LD ($D215),HL
8309  CD 28 33    CALL $3328
830C  D4 D9 2F    CALL NC,$2FD9
830F  DD 36 0D 14 LD (IX+13),$14
8313  DD 36 0E 20 LD (IX+14),$20
8317  21 06 10    LD HL,$1006
831A  22 15 D2    LD ($D215),HL
831D  CD 28 33    CALL $3328
8320  21 04 04    LD HL,$0404
8323  22 0F D2    LD ($D20F),HL
8326  D4 C1 2F    CALL NC,$2FC1
8329  DD 6E 0A    LD L,(IX+10)
832C  DD 66 0B    LD H,(IX+11)
832F  DD 7E 0C    LD A,(IX+12)
8332  11 20 00    LD DE,$0020
8335  19          ADD HL,DE
8336  CE 00       ADC A,$00
8338  DD 75 0A    LD (IX+10),L
833B  DD 74 0B    LD (IX+11),H
833E  DD 77 0C    LD (IX+12),A
8341  DD 7E 11    LD A,(IX+17)
8344  FE 50       CP $50
8346  38 18       JR C,$8360
8348  DD 36 07 40 LD (IX+7),$40
834C  DD 36 08 00 LD (IX+8),$00
8350  DD 36 09 00 LD (IX+9),$00
8354  11 93 83    LD DE,$8393
8357  01 8E 83    LD BC,$838E
835A  CD 75 7C    CALL $7C75
835D  C3 75 83    JP $8375
8360  DD 36 07 C0 LD (IX+7),$C0
8364  DD 36 08 FF LD (IX+8),$FF
8368  DD 36 09 FF LD (IX+9),$FF
836C  11 93 83    LD DE,$8393
836F  01 89 83    LD BC,$8389
8372  CD 75 7C    CALL $7C75
8375  3A 24 D2    LD A,($D224)
8378  E6 07       AND $07
837A  C0          RET NZ
837B  DD 34 11    INC (IX+17)
837E  DD 7E 11    LD A,(IX+17)
8381  FE A0       CP $A0
8383  D8          RET C
8384  DD 36 11 00 LD (IX+17),$00
8388  C9          RET
8389  .byte 00 06 01 06 FF 02 06 03 06 FF FE 00 02 FF FF FF ; ................
8399  .byte 20 22 24 FF FF FF FF FF FF FF FF FF FE 00 02 FF ;  "$.............
83A9  .byte FF FF 26 28 2A FF FF FF FF FF FF FF FF FF 40 42 ; ..&(*.........@B
83B9  .byte FF FF FF FF 4A 4C 4E FF FF FF FF FF FF FF FF FF ; ....JLN.........
83C9  .byte 40 42 FF FF FF FF 44 46 48 FF FF FF FF DD CB 18 ; @B....DFH.......
83D9  .byte EE DD 36 0D 0E DD 36 0E 0C DD CB 18 46 20 54 AF ; ..6...6.....F T.
83E9  .byte DD 77 0F DD 77 10 6F 67 22 0F D2 DD CB 18 4E 20 ; .w..w.og".....N 
83F9  .byte 0D CD 88 06 E6 1F 3C DD 77 11 DD CB 18 CE DD 35 ; ......<.w......5
8409  .byte 11 C2 7C 84 DD 36 11 01 3A AC D2 E6 80 CA 7C 84 ; ..|..6..:.....|.
8419  .byte DD 6E 02 DD 66 03 22 AB D2 DD 6E 05 DD 66 06 11 ; .n..f."...n..f..
8429  .byte 0E 00 19 22 AD D2 21 A3 84 22 AF D2 DD CB 18 C6 ; ..."..!.."......
8439  .byte 3E 20 EF DD 36 0F 96 DD 36 10 84 DD 6E 0A DD 66 ; > ..6...6...n..f
8449  .byte 0B DD 7E 0C 11 20 00 19 CE 00 4F 7C FE 04 38 02 ; ..~.. ....O|..8.
8459  .byte 26 04 DD 75 0A DD 74 0B DD 71 0C 22 0F D2 ED 5B ; &..u..t..q."...[
8469  .byte 57 D2 14 DD 6E 05 DD 66 06 A7 ED 52 38 05 DD 36 ; W...n..f...R8..6
8479  .byte 00 FF C9 21 02 04 22 15 D2 CD 28 33 D8 3A 09 D4 ; ...!.."...(3.:..
8489  .byte A7 F8 ED 5B 0F D2 01 10 00 CD F5 7C C9 FE FF FF ; ...[.......|....
8499  .byte FF FF FF 70 72 FF FF FF FF FF 00 00 00 00 00 00 ; ...pr...........
84A9  .byte 00 00 DD CB 18 EE DD 36 0D 1E DD 36 0E 1C CD DA ; .......6...6....
84B9  .byte 7C DD 36 0F 6F DD 36 10 86 DD CB 18 46 20 24 21 ; |.6.o.6.....F $!
84C9  .byte D0 05 11 08 03 CD C0 7C 21 DE E5 11 00 20 3E 0C ; .......|!.... >.
84D9  .byte CD 06 04 3E 11 32 2D D2 AF 32 ED D2 3E 0B DF DD ; ...>.2-..2..>...
84E9  .byte CB 18 C6 DD 7E 11 A7 20 28 CD 88 06 E6 01 87 87 ; ....~.. (.......
84F9  .byte 5F 16 00 21 47 86 19 7E DD 77 02 23 7E 23 DD 77 ; _..!G..~.w.#~#.w
8509  .byte 03 7E 23 DD 77 05 7E 23 DD 77 06 DD 34 11 C3 D9 ; .~#.w.~#.w..4...
8519  .byte 85 3D 20 24 DD 36 0A 80 DD 36 0B FF DD 36 0C FF ; .= $.6...6...6..
8529  .byte 21 70 03 DD 5E 05 DD 56 06 AF ED 52 DA D9 85 DD ; !p..^..V...R....
8539  .byte 34 11 DD 77 12 C3 D9 85 3D 20 78 AF DD 77 0A DD ; 4..w....= x..w..
8549  .byte 77 0B DD 77 0C DD 34 12 DD 7E 12 FE 64 C2 D9 85 ; w..w..4..~..d...
8559  .byte DD 34 11 3A ED D2 FE 08 30 76 2A FF D3 DD 5E 02 ; .4.:....0v*...^.
8569  .byte DD 56 03 A7 ED 52 21 4F 86 38 03 21 5F 86 5E 23 ; .V...R!O.8.!_.^#
8579  .byte 56 23 4E 23 46 23 E5 DD 6E 02 DD 66 03 19 22 0F ; V#N#F#..n..f..".
8589  .byte D2 DD 6E 05 DD 66 06 09 22 11 D2 E1 06 03 C5 7E ; ..n..f.."......~
8599  .byte 32 13 D2 23 7E 32 14 D2 23 7E 32 15 D2 23 7E 32 ; 2..#~2..#~2..#~2
85A9  .byte 16 D2 23 E5 0E 10 CD E6 85 E1 C1 10 E1 3E 01 EF ; ..#..........>..
85B9  .byte C3 D9 85 DD 36 0A 80 DD 36 0B 00 DD 36 0C 00 21 ; ....6...6...6..!
85C9  .byte D0 03 DD 5E 05 DD 56 06 AF ED 52 30 03 DD 77 11 ; ...^..V...R0..w.
85D9  .byte 21 A2 00 22 17 D2 CD FD 77 CD 02 94 C9 C5 CD AF ; !.."....w.......
85E9  .byte 7C C1 D8 DD E5 E5 DD E1 AF DD 36 00 0D 2A 0F D2 ; |.........6..*..
85F9  .byte DD 77 01 DD 75 02 DD 74 03 2A 11 D2 DD 77 04 DD ; .w..u..t.*...w..
8609  .byte 75 05 DD 74 06 DD 77 11 DD 71 13 DD 77 14 DD 77 ; u..t..w..q..w..w
8619  .byte 15 DD 77 16 DD 77 17 2A 13 D2 AF CB 7C 28 01 3D ; ..w..w.*....|(.=
8629  .byte DD 75 07 DD 74 08 DD 77 09 2A 15 D2 AF CB 7C 28 ; .u..t..w.*....|(
8639  .byte 01 3D DD 75 0A DD 74 0B DD 77 0C DD E1 C9 0E 06 ; .=.u..t..w......
8649  .byte D0 03 6E 06 D0 03 00 00 F6 FF C0 FE 00 FC 60 FE ; ..n...........`.
8659  .byte 80 FD C0 FD 00 FF 20 00 F6 FF 40 01 00 FC A0 01 ; ...... ...@.....
8669  .byte 80 FD 40 02 00 FF 20 22 24 26 28 FF 40 42 44 46 ; ..@... "$&(.@BDF
8679  .byte 48 FF 60 62 64 66 68 FF DD CB 18 EE DD CB 18 46 ; H.`bdfh........F
8689  .byte 20 18 DD 36 11 1C DD 6E 02 DD 66 03 11 F0 FF 19 ;  ..6...n..f.....
8699  .byte DD 75 02 DD 74 03 DD CB 18 C6 DD 6E 14 DD 66 15 ; .u..t......n..f.
86A9  .byte DD 7E 16 DD 5E 12 DD 56 13 0E 00 CB 7A 28 01 0D ; .~..^..V....z(..
86B9  .byte 19 89 DD 75 14 DD 74 15 DD 77 16 4C 47 21 38 00 ; ...u..t..w.LG!8.
86C9  .byte 19 DD 75 12 DD 74 13 CB 7C 20 5C 07 38 59 DD 7E ; ..u..t..| \.8Y.~
86D9  .byte 11 A7 28 3F DD CB 18 4E 28 26 7D B4 20 0E 3A E9 ; ..(?...N(&}. .:.
86E9  .byte D2 2A E7 D2 22 07 D4 32 09 D4 18 14 7D 2F 6F 7C ; .*.."..2....}/o|
86F9  .byte 2F 67 23 ED 5B E7 D2 19 22 07 D4 3E FF 32 09 D4 ; /g#.[..."..>.2..
8709  .byte 3E 1C 91 DD 77 11 28 02 30 1D DD CB 18 4E 28 03 ; >...w.(.0....N(.
8719  .byte 3E 04 EF AF DD 77 11 DD 77 12 DD 77 13 DD 77 14 ; >....w..w..w..w.
8729  .byte DD 36 15 1C DD 77 16 DD 6E 02 DD 66 03 22 0F D2 ; .6...w..n..f."..
8739  .byte DD 6E 05 DD 66 06 22 11 D2 21 00 00 22 13 D2 DD ; .n..f."..!.."...
8749  .byte 6E 11 11 10 00 19 22 15 D2 21 45 88 CD 2F 88 21 ; n....."..!E../.!
8759  .byte 28 00 22 13 D2 3E 1C DD 96 11 6F 26 00 11 10 00 ; (."..>....o&....
8769  .byte 19 22 15 D2 21 45 88 CD 2F 88 21 2C 00 22 13 D2 ; ."..!E../.!,."..
8779  .byte DD 6E 15 DD 66 16 22 15 D2 21 49 88 CD 2F 88 DD ; .n..f."..!I../..
8789  .byte CB 18 8E DD 36 0D 14 3E 02 32 15 D2 DD 7E 11 4F ; ....6..>.2...~.O
8799  .byte C6 08 DD 77 0E 79 C6 04 32 16 D2 CD 28 33 30 28 ; ...w.y..2...(30(
87A9  .byte 3A 09 D4 A7 F8 DD 36 0D 3C 3E 2A 32 15 D2 3E 1C ; :.....6.<>*2..>.
87B9  .byte DD 96 11 C6 08 DD 77 0E 3E 1C DD 96 11 C6 04 32 ; ......w.>......2
87C9  .byte 16 D2 CD 28 33 30 32 C9 DD CB 18 CE 3A 09 D4 A7 ; ...(302.....:...
87D9  .byte F8 DD 7E 11 FE 1C 28 21 2A 07 D4 7D 2F 6F 7C 2F ; ..~...(!*..}/o|/
87E9  .byte 67 23 DD 75 12 DD 74 13 3A 08 D4 DD 86 11 DD 77 ; g#.u..t.:......w
87F9  .byte 11 FE 1C 38 10 DD 36 11 1C 3A E9 D2 2A E7 D2 22 ; ...8..6..:..*.."
8809  .byte 07 D4 32 09 D4 DD 6E 05 DD 66 06 01 10 00 09 3A ; ..2...n..f.....:
8819  .byte 16 D2 D6 04 4F 09 3A 0B D4 4F AF ED 42 22 02 D4 ; ....O.:..O..B"..
8829  .byte 21 15 D4 CB FE C9 7E A7 F8 E5 CD 5D 2F 2A 13 D2 ; !.....~....]/*..
8839  .byte 11 08 00 19 22 13 D2 E1 23 C3 2F 88 36 38 3A FF ; ...."...#./.68:.
8849  .byte 3C 3E FF DD CB 18 EE DD 7E 11 FE 80 30 31 DD 36 ; <>......~...01.6
8859  .byte 07 20 DD 36 08 00 DD 36 09 00 DD 36 0D 14 DD 36 ; . .6...6...6...6
8869  .byte 0E 0C 21 02 0A 22 15 D2 CD 28 33 21 08 00 22 0F ; ..!.."...(3!..".
8879  .byte D2 D4 C1 2F 11 D3 88 01 C9 88 CD 75 7C 18 2F DD ; .../.......u|./.
8889  .byte 36 07 E0 DD 36 08 FF DD 36 09 FF DD 36 0D 0C DD ; 6...6...6...6...
8899  .byte 36 0E 0C 21 02 02 22 15 D2 CD 28 33 21 00 00 22 ; 6..!.."...(3!.."
88A9  .byte 0F D2 D4 C1 2F 11 D3 88 01 CE 88 CD 75 7C 3A 24 ; ..../.......u|:$
88B9  .byte D2 E6 07 C0 DD 34 11 CD 88 06 E6 1E CC F5 91 C9 ; .....4..........
88C9  .byte 00 04 01 04 FF 02 04 03 04 FF 04 2A 2C FF FF FF ; ...........*,...
88D9  .byte FF FF FF FF FF FF FF FF FF FF FF FF 0C 2A 2C FF ; .............*,.
88E9  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF FF 0E 10 ; ................
88F9  .byte 0A FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF ; ................
8909  .byte 0E 10 0C FF FF FF FF DD CB 18 EE DD 36 0D 08 DD ; ............6...
8919  .byte 36 0E 0C DD CB 18 46 20 21 DD 6E 02 DD 66 03 11 ; 6.....F !.n..f..
8929  .byte 08 00 19 DD 75 12 DD 74 13 DD 6E 05 DD 66 06 19 ; ....u..t..n..f..
8939  .byte DD 75 14 DD 74 15 DD CB 18 C6 DD 6E 11 26 00 29 ; .u..t......n.&.)
8949  .byte 11 A0 89 19 5E 23 4E 16 00 42 CB 7B 28 01 15 CB ; ....^#N..B.{(...
8959  .byte 79 28 01 05 DD 6E 12 DD 66 13 19 DD 75 02 DD 74 ; y(...n..f...u..t
8969  .byte 03 DD 6E 14 DD 66 15 09 DD 75 05 DD 74 06 21 04 ; ..n..f...u..t.!.
8979  .byte 02 22 15 D2 CD 28 33 D4 D9 2F DD 36 0F 99 DD 36 ; ."...(3../.6...6
8989  .byte 10 89 DD 34 11 DD 7E 11 FE B4 D8 DD 36 11 00 C9 ; ...4..~.....6...
8999  .byte 60 62 FF FF FF FF FF 40 00 40 02 40 04 40 07 3F ; `b.....@.@.@.@.?
89A9  .byte 09 3F 0B 3F 0D 3E 0F 3E 12 3D 14 3C 16 3B 18 3A ; .?.?.>.>.=.<.;.:
89B9  .byte 1A 3A 1C 39 1E 37 20 36 22 35 24 34 26 32 27 31 ; .:.9.7 6"5$4&2'1
89C9  .byte 29 30 2B 2E 2C 2C 2E 2B 30 29 31 27 32 26 34 24 ; )0+.,,.+0)1'2&4$
89D9  .byte 35 22 36 20 37 1E 39 1C 3A 1A 3A 18 3B 16 3C 14 ; 5"6 7.9.:.:.;.<.
89E9  .byte 3D 12 3E 0F 3E 0D 3F 0B 3F 09 3F 07 40 04 40 02 ; =.>.>.?.?.?.@.@.
89F9  .byte 40 00 40 FE 40 FC 40 F9 40 F7 3F F5 3F F3 3F F1 ; @.@.@.@.@.?.?.?.
8A09  .byte 3E EE 3E EC 3D EA 3C E8 3B E6 3A E4 3A E2 39 E0 ; >.>.=.<.;.:.:.9.
8A19  .byte 37 DE 36 DC 35 DA 34 D9 32 D7 31 D5 30 D4 2E D2 ; 7.6.5.4.2.1.0...
8A29  .byte 2C D0 2B CF 29 CE 27 CC 26 CB 24 CA 22 C9 20 C7 ; ,.+.).'.&.$.". .
8A39  .byte 1E C6 1C C6 1A C5 18 C4 16 C3 14 C2 12 C2 0F C1 ; ................
8A49  .byte 0D C1 0B C1 09 C0 07 C0 04 C0 02 C0 00 C0 FE C0 ; ................
8A59  .byte FC C0 F9 C1 F7 C1 F5 C1 F3 C2 F1 C2 EE C3 EC C4 ; ................
8A69  .byte EA C5 E8 C6 E6 C6 E4 C7 E2 C9 E0 CA DE CB DC CC ; ................
8A79  .byte DA CE D9 CF D7 D0 D5 D2 D4 D4 D2 D5 D0 D7 CF D9 ; ................
8A89  .byte CE DA CC DC CB DE CA E0 C9 E2 C7 E4 C6 E6 C6 E8 ; ................
8A99  .byte C5 EA C4 EC C3 EE C2 F1 C2 F3 C1 F5 C1 F7 C1 F9 ; ................
8AA9  .byte C0 FC C0 FE C0 00 C0 02 C0 04 C0 07 C0 09 C1 0B ; ................
8AB9  .byte C1 0D C1 0F C2 12 C2 14 C3 16 C4 18 C5 1A C6 1C ; ................
8AC9  .byte C6 1E C7 20 C9 22 CA 24 CB 26 CC 27 CE 29 CF 2B ; ... .".$.&.'.).+
8AD9  .byte D0 2C D2 2E D4 30 D5 31 D7 32 D9 34 DA 35 DC 36 ; .,...0.1.2.4.5.6
8AE9  .byte DE 37 E0 39 E2 3A E4 3A E6 3B E8 3C EA 3D EC 3E ; .7.9.:.:.;.<.=.>
8AF9  .byte EE 3E F1 3F F3 3F F5 3F F7 40 F9 40 FC 40 FE DD ; .>.?.?.?.@.@.@..
8B09  .byte CB 18 EE DD CB 18 46 20 14 DD 6E 02 DD 66 03 11 ; ......F ..n..f..
8B19  .byte 0C 00 19 DD 75 02 DD 74 03 DD CB 18 C6 DD 6E 02 ; ....u..t......n.
8B29  .byte DD 66 03 22 0F D2 DD 6E 05 DD 66 06 22 11 D2 21 ; .f."...n..f."..!
8B39  .byte 00 00 22 13 D2 3A 24 D2 07 07 E6 03 20 14 21 CE ; .."..:$..... .!.
8B49  .byte 8B 3A 24 D2 E6 3F 5F FE 08 38 2F 21 DF 8B 1E 00 ; .:$..?_..8/!....
8B59  .byte 18 28 FE 01 20 07 21 DF 8B 1E 00 18 1D FE 02 20 ; .(.. .!........ 
8B69  .byte 14 21 D6 8B 3A 24 D2 E6 3F 5F FE 08 38 0C 21 DE ; .!..:$..?_..8.!.
8B79  .byte 8B 1E 00 18 05 21 DE 8B 1E 00 16 00 19 7E 21 E0 ; .....!.......~!.
8B89  .byte 8B 87 87 87 5F 19 06 03 C5 7E 23 5E 23 A7 FA A5 ; ...._....~#^#...
8B99  .byte 8B E5 16 00 ED 53 15 D2 CD 5D 2F E1 C1 10 E9 DD ; .....S...]/.....
8BA9  .byte 70 0F DD 70 10 56 1E 04 ED 53 15 D2 23 7E DD 36 ; p..p.V...S..#~.6
8BB9  .byte 0D 01 DD 77 0E CD 28 33 D4 D9 2F 3A 24 D2 FE 80 ; ...w..(3../:$...
8BC9  .byte C0 3E 1D EF C9 00 01 02 03 04 05 06 07 07 06 05 ; .>..............
8BD9  .byte 04 03 02 01 00 00 08 12 00 32 10 32 20 01 30 12 ; .........2.2 .0.
8BE9  .byte 04 32 14 32 20 02 30 12 08 32 18 32 20 06 30 12 ; .2.2 .0..2.2 .0.
8BF9  .byte 0C 32 1C 32 20 0A 30 12 10 32 20 FF 00 0E 30 12 ; .2.2 .0..2 ...0.
8C09  .byte 14 32 20 FF 00 12 30 12 18 32 20 FF 00 16 30 12 ; .2 ...0..2 ...0.
8C19  .byte 1C 32 20 FF 00 1A 30 12 20 FF 00 FF 00 1E 30 DD ; .2 ...0. .....0.
8C29  .byte CB 18 AE DD 36 0D 04 DD 36 0E 0A DD CB 18 46 20 ; ....6...6.....F 
8C39  .byte 45 DD 6E 02 DD 66 03 11 0A 00 19 DD 75 02 DD 74 ; E.n..f......u..t
8C49  .byte 03 DD 75 12 DD 74 13 DD 6E 05 DD 66 06 11 08 00 ; ..u..t..n..f....
8C59  .byte 19 DD 75 05 DD 74 06 DD 75 14 DD 74 15 DD 36 11 ; ..u..t..u..t..6.
8C69  .byte 96 DD CB 18 C6 01 00 00 59 50 CD D5 30 7E FE 52 ; ........YP..0~.R
8C79  .byte 28 04 DD CB 18 CE DD 7E 11 A7 28 19 DD 35 11 28 ; (......~..(..5.(
8C89  .byte 11 AF DD 77 0F DD 77 10 DD 77 07 DD 77 08 DD 77 ; ...w..w..w..w..w
8C99  .byte 09 C9 3E 18 EF AF DD CB 18 4E 20 16 DD 36 07 00 ; ..>......N ..6..
8CA9  .byte DD 36 08 FF DD 36 09 FF DD 36 0F 4A DD 36 10 8D ; .6...6...6.J.6..
8CB9  .byte 18 12 DD 77 07 DD 36 08 01 DD 77 09 DD 36 0F 52 ; ...w..6...w..6.R
8CC9  .byte DD 36 10 8D DD 77 0A DD 77 0B DD 77 0C DD CB 18 ; .6...w..w..w....
8CD9  .byte 76 20 4F DD CB 18 7E 20 49 21 02 04 22 15 D2 CD ; v O...~ I!.."...
8CE9  .byte 28 33 D4 D9 2F DD 5E 02 DD 56 03 2A 54 D2 01 F0 ; (3../.^..V.*T...
8CF9  .byte FF 09 A7 ED 52 30 2B 2A 54 D2 01 10 01 09 A7 ED ; ....R0+*T.......
8D09  .byte 52 38 1F DD 5E 05 DD 56 06 2A 57 D2 01 F0 FF 09 ; R8..^..V.*W.....
8D19  .byte A7 ED 52 30 0D 2A 57 D2 01 D0 00 09 A7 ED 52 38 ; ..R0.*W.......R8
8D29  .byte 01 C9 DD 6E 12 DD 66 13 DD 75 02 DD 74 03 DD 6E ; ...n..f..u..t..n
8D39  .byte 14 DD 66 15 DD 75 05 DD 74 06 DD 36 11 96 C3 8A ; ..f..u..t..6....
8D49  .byte 8C 2E FF FF FF FF FF FF FF 30 FF FF FF FF FF FF ; .........0......
8D59  .byte DD CB 18 EE DD 7E 11 5F 16 00 21 4A 8E 19 5E 7A ; .....~._..!J..^z
8D69  .byte CB 7B 28 02 3D 15 DD 6E 04 DD 66 05 19 DD 8E 06 ; .{(.=..n..f.....
8D79  .byte DD 75 04 DD 74 05 DD 77 06 6C DD 66 06 3A 24 D2 ; .u..t..w.l.f.:$.
8D89  .byte E6 0F 20 0E DD 34 11 DD 7E 11 FE 20 38 04 DD 36 ; .. ..4..~.. 8..6
8D99  .byte 11 00 22 DD D2 ED 5B 57 D2 A7 3E FF ED 52 38 13 ; .."...[W..>..R8.
8DA9  .byte EB 21 0C 00 3E FF ED 52 30 09 21 B4 00 AF ED 52 ; .!..>..R0.!....R
8DB9  .byte 38 01 7B 32 DC D2 A7 C8 FE FF C8 C6 09 6F 26 00 ; 8.{2.........o&.
8DC9  .byte 22 15 D2 2A 54 D2 22 0F D2 2A 57 D2 22 11 D2 FD ; "..*T."..*W."...
8DD9  .byte 7E 0A 2A 36 D2 F5 E5 21 00 D0 22 36 D2 DD 7E 12 ; ~.*6...!.."6..~.
8DE9  .byte 87 4F 06 00 21 3E 8E 09 06 02 C5 4E 23 E5 DD 7E ; .O..!>.....N#..~
8DF9  .byte 13 81 6F 26 00 22 13 D2 3E 00 CD 5D 2F 2A 13 D2 ; ..o&."..>..]/*..
8E09  .byte 11 08 00 19 22 13 D2 3E 02 CD 5D 2F E1 C1 10 DA ; ...."..>..]/....
8E19  .byte E1 F1 22 36 D2 FD 77 0A DD 34 12 DD 7E 12 FE 06 ; .."6..w..4..~...
8E29  .byte D8 DD 36 12 00 DD 7E 13 C6 02 DD 77 13 FE 10 D8 ; ..6...~....w....
8E39  .byte DD 36 13 00 C9 30 80 40 90 50 A0 60 B0 70 C0 80 ; .6...0.@.P.`.p..
8E49  .byte D0 FE FC F8 F0 E8 D8 C8 C8 C8 C8 D8 E8 F0 F8 FC ; ................
8E59  .byte FE 02 04 08 10 18 28 38 38 38 38 28 18 10 08 04 ; ......(8888(....
8E69  .byte 02 DD CB 18 EE DD 7E 12 E6 7F 20 11 CD 88 06 E6 ; ......~... .....
8E79  .byte 07 5F 16 00 21 C7 8E 19 CB 46 C4 F5 91 11 9C 8E ; ._..!....F......
8E89  .byte 01 93 8E CD 75 7C DD 34 12 C9 00 0A 01 0A 02 0A ; ....u|.4........
8E99  .byte 01 0A FF FE 0A FF FF FF FF FF FF FF FF FF FF FF ; ................
8EA9  .byte FF FF FF FF FF FE 0C FF FF FF FF FF FF FF FF FF ; ................
8EB9  .byte FF FF FF FF FF FF FF FE 04 FF FF FF FF FF 01 00 ; ................
8EC9  .byte 01 01 00 01 00 01 DD CB 18 EE AF DD 77 0F DD 77 ; ............w..w
8ED9  .byte 10 DD 7E 11 E6 0F 20 1C CD 88 06 01 28 00 16 00 ; ..~... .....(...
8EE9  .byte E6 3F FE 20 38 05 01 D8 FF 16 FF DD 71 07 DD 70 ; .?. 8.......q..p
8EF9  .byte 08 DD 72 09 DD 36 0A 60 DD 36 0B FF DD 36 0C FF ; ..r..6.`.6...6..
8F09  .byte DD 6E 02 DD 66 03 22 0F D2 EB 2A 54 D2 01 08 00 ; .n..f."...*T....
8F19  .byte AF ED 42 30 02 6F 67 A7 ED 52 30 36 2A 54 D2 01 ; ..B0.og..R06*T..
8F29  .byte 00 01 09 A7 ED 52 38 2A DD 6E 05 DD 66 06 22 11 ; .....R8*.n..f.".
8F39  .byte D2 EB 2A DD D2 A7 ED 52 30 18 2A 57 D2 01 F0 FF ; ..*....R0.*W....
8F49  .byte 09 A7 ED 52 30 0C 2A 57 D2 01 C0 00 09 A7 ED 52 ; ...R0.*W.......R
8F59  .byte 30 04 DD 36 00 FF 21 00 00 22 13 D2 22 15 D2 3E ; 0..6..!..".."..>
8F69  .byte 0C CD 5D 2F DD 34 11 C9 C9 DD 36 0D 0C DD 36 0E ; ..]/.4....6...6.
8F79  .byte 20 21 02 02 22 15 D2 CD 28 33 21 00 08 22 0F D2 ;  !.."...(3!.."..
8F89  .byte D4 C1 2F DD 6E 0A DD 66 0B DD 7E 0C 11 10 00 19 ; ../.n..f..~.....
8F99  .byte CE 00 4F FA A9 8F 7C FE 04 38 05 21 00 03 0E 00 ; ..O...|..8.!....
8FA9  .byte DD 75 0A DD 74 0B DD 71 0C DD CB 18 46 C2 2E 90 ; .u..t..q....F...
8FB9  .byte 11 D0 FF DD 6E 02 DD 66 03 19 ED 5B FF D3 A7 ED ; ....n..f...[....
8FC9  .byte 52 30 1F 01 30 00 DD 6E 02 DD 66 03 09 A7 ED 52 ; R0..0..n..f....R
8FD9  .byte 38 10 DD CB 18 C6 DD 36 0A 80 DD 36 0B FD DD 36 ; 8......6...6...6
8FE9  .byte 0C FF DD 6E 02 DD 66 03 ED 5B FF D3 A7 ED 52 38 ; ...n..f..[....R8
8FF9  .byte 1A DD 36 07 C0 DD 36 08 FF DD 36 09 FF 11 5E 90 ; ..6...6...6...^.
9009  .byte 01 4F 90 CD 75 7C DD CB 18 CE C9 DD 36 07 40 DD ; .O..u|......6.@.
9019  .byte 36 08 00 DD 36 09 00 11 5E 90 01 4A 90 CD 75 7C ; 6...6...^..J..u|
9029  .byte DD CB 18 8E C9 01 59 90 DD CB 18 4E 20 03 01 54 ; ......Y....N ..T
9039  .byte 90 11 5E 90 CD 75 7C DD CB 18 7E C8 DD CB 18 86 ; ..^..u|...~.....
9049  .byte C9 00 04 01 04 FF 02 04 03 04 FF 04 04 04 04 FF ; ................
9059  .byte 05 04 05 04 FF 44 46 FF FF FF FF 64 66 FF FF FF ; .....DF....df...
9069  .byte FF FF FF FF FF FF FF 44 46 FF FF FF FF 48 4A FF ; .......DF....HJ.
9079  .byte FF FF FF FF FF FF FF FF FF 50 52 FF FF FF FF 70 ; .........PR....p
9089  .byte 72 FF FF FF FF FF FF FF FF FF FF 50 52 FF FF FF ; r..........PR...
9099  .byte FF 4C 4E FF FF FF FF FF FF FF FF FF FF 44 46 FF ; .LN..........DF.
90A9  .byte FF FF FF 68 6A FF FF FF FF FF FF FF FF FF FF 50 ; ...hj..........P
90B9  .byte 52 FF FF FF FF 6C 6E FF FF FF FF FF DD CB 18 EE ; R....ln.........
90C9  .byte DD 36 0D 1E DD 36 0E 1C DD 36 0F E8 DD 36 10 91 ; .6...6...6...6..
90D9  .byte DD CB 18 4E 20 26 DD 6E 02 DD 66 03 DD 75 11 DD ; ...N &.n..f..u..
90E9  .byte 74 12 DD 6E 05 DD 66 06 11 FF FF 19 DD 75 05 DD ; t..n..f......u..
90F9  .byte 74 06 DD 75 13 DD 74 14 DD CB 18 CE 01 10 00 11 ; t..u..t.........
9109  .byte 20 00 CD D5 30 5E 16 00 3A D5 D2 87 4F 42 21 3D ;  ...0^..:...OB!=
9119  .byte 34 09 7E 23 66 6F 19 7E E6 3F 0E 00 69 61 FE 1E ; 4.~#fo.~.?..ia..
9129  .byte 28 22 DD CB 18 46 28 25 DD 6E 0A DD 66 0B DD 7E ; ("...F(%.n..f..~
9139  .byte 0C 11 F8 FF 19 CE FF 4F 7C ED 44 FE 02 38 05 21 ; .......O|.D..8.!
9149  .byte 00 FF 0E FF DD 75 0A DD 74 0B DD 71 0C DD 5E 02 ; .....u..t..q..^.
9159  .byte DD 56 03 2A 54 D2 01 E0 FF 09 A7 ED 52 30 27 2A ; .V.*T.......R0'*
9169  .byte 54 D2 24 A7 ED 52 38 1E DD 5E 05 DD 56 06 2A 57 ; T.$..R8..^..V.*W
9179  .byte D2 01 E0 FF 09 A7 ED 52 30 0C 2A 57 D2 01 E0 00 ; .......R0.*W....
9189  .byte 09 A7 ED 52 30 2D DD 6E 11 DD 66 12 DD 75 02 DD ; ...R0-.n..f..u..
9199  .byte 74 03 DD 6E 13 DD 66 14 DD 75 05 DD 74 06 AF DD ; t..n..f..u..t...
91A9  .byte 77 01 DD 77 04 DD 77 0A DD 77 0B DD 77 0C DD CB ; w..w..w..w..w...
91B9  .byte 18 86 C9 21 02 0E 22 15 D2 CD 28 33 D8 DD CB 18 ; ...!.."...(3....
91C9  .byte C6 3A 08 D4 A7 F2 D6 91 ED 44 FE 02 D0 FD CB 06 ; .:.......D......
91D9  .byte 76 C0 DD 5E 0A DD 56 0B 01 10 00 CD F5 7C C9 FE ; v..^..V......|..
91E9  .byte FF FF FF FF FF 16 18 1A 1C FF FF FF CD AF 7C D8 ; ..............|.
91F9  .byte 0E 42 DD 7E 00 FE 41 20 0F E5 CD 88 06 E6 0F 5F ; .B.~..A ......._
9209  .byte 16 00 21 61 92 19 4E E1 79 DD 5E 02 DD 56 03 DD ; ..!a..N.y.^..V..
9219  .byte 4E 05 DD 46 06 DD E5 E5 DD E1 DD 77 00 AF DD 77 ; N..F.......w...w
9229  .byte 01 CD 88 06 E6 0F 6F 26 00 19 DD 75 02 DD 74 03 ; ......o&...u..t.
9239  .byte DD 36 04 00 CD 88 06 E6 0F 6F AF 67 09 DD 75 05 ; .6.......o.g..u.
9249  .byte DD 74 06 DD 77 11 DD 77 12 DD 77 18 DD 77 07 DD ; .t..w..w..w..w..
9259  .byte 77 08 DD 77 09 DD E1 C9 42 20 20 20 42 20 20 20 ; w..w....B   B   
9269  .byte 42 20 20 20 42 20 20 20 DD CB 18 EE DD 36 0D 20 ; B   B   .....6. 
9279  .byte DD 36 0E 1C CD DA 7C DD 36 0F 95 DD 36 10 94 DD ; .6....|.6...6...
9289  .byte CB 18 46 20 23 21 60 05 11 00 02 CD C0 7C FD CB ; ..F #!`......|..
9299  .byte 09 CE 21 DE E5 11 00 20 3E 0C CD 06 04 AF 32 ED ; ..!.... >.....2.
92A9  .byte D2 3E 0B DF DD CB 18 C6 DD 7E 11 A7 20 26 DD 7E ; .>.......~.. &.~
92B9  .byte 13 87 87 5F 16 00 21 7D 94 19 7E DD 77 02 23 7E ; ..._..!}..~.w.#~
92C9  .byte 23 DD 77 03 7E 23 DD 77 05 7E 23 DD 77 06 DD 34 ; #.w.~#.w.~#.w..4
92D9  .byte 11 C3 F9 93 3D 20 46 DD 7E 13 A7 20 0F DD 36 0A ; ....= F.~.. ..6.
92E9  .byte 80 DD 36 0B FF DD 36 0C FF C3 01 93 DD 36 0A 80 ; ..6...6......6..
92F9  .byte DD 36 0B 00 DD 36 0C 00 21 89 94 DD 7E 13 87 5F ; .6...6..!...~.._
9309  .byte 16 00 19 7E 23 66 6F DD 5E 05 DD 56 06 A7 ED 52 ; ...~#fo.^..V...R
9319  .byte C2 F9 93 DD 34 11 DD 36 12 00 C3 F9 93 3D C2 AD ; ....4..6.....=..
9329  .byte 93 AF DD 77 0A DD 77 0B DD 77 0C DD 34 12 DD 7E ; ...w..w..w..4..~
9339  .byte 12 FE 64 C2 F9 93 DD 34 11 DD 6E 02 DD 66 03 11 ; ..d....4..n..f..
9349  .byte 0F 00 19 22 0F D2 DD 6E 05 DD 66 06 01 22 00 09 ; ..."...n..f.."..
9359  .byte 22 11 D2 DD 7E 13 A7 CA 34 94 3A ED D2 FE 08 D2 ; "...~...4.:.....
9369  .byte F9 93 CD AF 7C DA F9 93 DD E5 E5 DD E1 AF DD 36 ; ....|..........6
9379  .byte 00 2F 2A 0F D2 DD 77 01 DD 75 02 DD 74 03 2A 11 ; ./*...w..u..t.*.
9389  .byte D2 DD 77 04 DD 75 05 DD 74 06 DD 77 18 DD 77 07 ; ..w..u..t..w..w.
9399  .byte DD 77 08 DD 77 09 DD 77 0A DD 77 0B DD 77 0C DD ; .w..w..w..w..w..
93A9  .byte E1 C3 F9 93 DD 7E 13 A7 20 0F DD 36 0A 80 DD 36 ; .....~.. ..6...6
93B9  .byte 0B 00 DD 36 0C 00 C3 CE 93 DD 36 0A 80 DD 36 0B ; ...6......6...6.
93C9  .byte FF DD 36 0C FF 21 8F 94 DD 7E 13 87 5F 16 00 19 ; ..6..!...~.._...
93D9  .byte 7E 23 66 6F DD 5E 05 DD 56 06 AF ED 52 20 11 DD ; ~#fo.^..V...R ..
93E9  .byte 77 11 DD 34 13 DD 7E 13 FE 03 38 04 DD 36 13 00 ; w..4..~...8..6..
93F9  .byte 21 A2 00 22 17 D2 CD FD 77 3A ED D2 FE 08 D0 DD ; !.."....w:......
9409  .byte CB 0C 7E C8 DD 6E 02 DD 66 03 22 0F D2 DD 6E 05 ; ..~..n..f."...n.
9419  .byte DD 66 06 22 11 D2 21 10 00 22 13 D2 21 30 00 22 ; .f."..!.."..!0."
9429  .byte 15 D2 3A 24 D2 E6 02 CD 5D 2F C9 DD 6E 02 DD 66 ; ..:$....]/..n..f
9439  .byte 03 11 04 00 19 22 0F D2 DD 6E 05 DD 66 06 11 FA ; ....."...n..f...
9449  .byte FF 19 22 11 D2 21 00 FF 22 13 D2 21 00 FF 22 15 ; .."..!.."..!..".
9459  .byte D2 0E 04 CD E6 85 DD 6E 02 DD 66 03 11 20 00 19 ; .......n..f.. ..
9469  .byte 22 0F D2 21 00 01 22 13 D2 0E 04 CD E6 85 3E 01 ; "..!..".......>.
9479  .byte EF C3 F9 93 CC 05 C0 02 9C 05 C0 01 FC 05 C0 01 ; ................
9489  .byte 88 02 30 02 30 02 C0 02 C0 01 C0 01 20 22 24 26 ; ..0.0....... "$&
9499  .byte 28 FF 40 42 44 46 48 FF 60 62 64 66 68 FF DD CB ; (.@BDFH.`bdfh...
94A9  .byte 18 EE DD 36 0D 08 DD 36 0E 0A 21 04 04 22 15 D2 ; ...6...6..!.."..
94B9  .byte CD 28 33 D4 D9 2F DD CB 18 4E 20 1F DD CB 18 CE ; .(3../...N .....
94C9  .byte 2A FF D3 11 08 00 19 EB DD 6E 02 DD 66 03 01 08 ; *........n..f...
94D9  .byte 00 09 A7 ED 52 30 04 DD CB 18 D6 DD CB 18 46 20 ; ....R0........F 
94E9  .byte 30 DD 36 0A 40 DD 36 0B 00 DD 36 0C 00 21 9A 96 ; 0.6.@.6...6..!..
94F9  .byte DD CB 18 56 28 03 21 8A 96 DD 75 0F DD 74 10 2A ; ...V(.!...u..t.*
9509  .byte 02 D4 DD 5E 05 DD 56 06 A7 ED 52 D0 DD CB 18 C6 ; ...^..V...R.....
9519  .byte C9 DD 4E 02 DD 46 03 21 F0 FF 09 ED 5B 54 D2 A7 ; ..N..F.!....[T..
9529  .byte ED 52 38 24 69 60 14 A7 ED 52 30 1C DD 4E 05 DD ; .R8$i`...R0..N..
9539  .byte 46 06 21 F0 FF 09 ED 5B 57 D2 A7 ED 52 38 09 21 ; F.!....[W...R8.!
9549  .byte C0 00 19 A7 ED 42 30 04 DD 36 00 FF AF 21 02 00 ; .....B0..6...!..
9559  .byte DD CB 18 56 20 04 3D 21 FE FF DD 5E 07 DD 56 08 ; ...V .=!...^..V.
9569  .byte 19 DD 8E 09 4F 7C 11 00 01 CB 79 28 0B 7D 2F 5F ; ....O|....y(.}/_
9579  .byte 7C 2F 57 13 7A 11 00 FF A7 28 01 EB DD 75 07 DD ; |/W.z....(...u..
9589  .byte 74 08 DD 71 09 2A 02 D4 11 10 00 19 EB DD 6E 05 ; t..q.*........n.
9599  .byte DD 66 06 01 08 00 09 A7 ED 52 3E FF 21 FE FF DD ; .f.......R>.!...
95A9  .byte CB 0C 7E 20 03 21 FC FF 30 0D 3C 21 02 00 DD CB ; ..~ .!..0.<!....
95B9  .byte 0C 7E 28 03 21 04 00 DD 5E 0A DD 56 0B 19 DD 8E ; .~(.!...^..V....
95C9  .byte 0C 4F 7C 11 00 01 CB 79 28 0B 7D 2F 5F 7C 2F 57 ; .O|....y(.}/_|/W
95D9  .byte 13 7A 11 00 FF A7 28 01 EB DD 75 0A DD 74 0B DD ; .z....(...u..t..
95E9  .byte 71 0C 21 8A 96 DD CB 09 7E 28 03 21 9A 96 E5 DD ; q.!.....~(.!....
95F9  .byte 6E 07 DD 66 08 CB 7C 28 07 7D 2F 6F 7C 2F 67 23 ; n..f..|(.}/o|/g#
9609  .byte DD 5E 11 DD 56 12 19 DD 75 11 DD 74 12 7C E6 08 ; .^..V...u..t.|..
9619  .byte 5F 16 00 E1 19 DD 75 0F DD 74 10 DD 6E 02 DD 66 ; _.....u..t..n..f
9629  .byte 03 11 F9 FF DD CB 09 7E 28 03 11 0F 00 19 22 0F ; .......~(.....".
9639  .byte D2 DD 6E 05 DD 66 06 22 11 D2 3A 24 D2 E6 0F C0 ; ..n..f."..:$....
9649  .byte CD AF 7C D8 DD E5 E5 DD E1 AF DD 36 00 2A 2A 0F ; ..|........6.**.
9659  .byte D2 DD 77 01 DD 75 02 DD 74 03 2A 11 D2 DD 77 04 ; ..w..u..t.*...w.
9669  .byte DD 75 05 DD 74 06 DD 77 11 DD 77 12 DD 77 07 DD ; .u..t..w..w..w..
9679  .byte 77 08 DD 77 09 DD 77 0A DD 77 0B DD 77 0C DD E1 ; w..w..w..w..w...
9689  .byte C9 3C 3E FF FF FF FF FF FF 38 3A FF FF FF FF FF ; .<>......8:.....
9699  .byte FF 56 58 FF FF FF FF FF FF 5A 5C FF FF FF FF FF ; .VX......Z\.....
96A9  .byte FF DD CB 18 EE AF DD 77 0F DD 77 10 DD 6E 02 DD ; .......w..w..n..
96B9  .byte 66 03 22 0F D2 DD 6E 05 DD 66 06 22 11 D2 6F 67 ; f."...n..f."..og
96C9  .byte 22 13 D2 22 15 D2 DD 5E 12 16 00 21 F7 96 19 7E ; ".."...^...!...~
96D9  .byte CD 5D 2F DD 34 11 DD 7E 11 FE 0C D8 DD 36 11 00 ; .]/.4..~.....6..
96E9  .byte DD 34 12 DD 7E 12 FE 03 D8 DD 36 00 FF C9 1C 1E ; .4..~.....6.....
96F9  .byte 5E DD CB 18 EE AF DD 77 0F DD 77 10 FD 7E 0A 2A ; ^......w..w..~.*
9709  .byte 36 D2 F5 E5 3A DF D2 FE 24 30 55 5F 16 00 21 00 ; 6...:...$0U_..!.
9719  .byte D0 19 22 36 D2 DD 6E 02 DD 66 03 22 0F D2 DD 6E ; .."6..n..f."...n
9729  .byte 05 DD 66 06 22 11 D2 21 00 00 22 13 D2 22 15 D2 ; ..f."..!..".."..
9739  .byte DD 7E 12 A7 28 0E FE 08 30 0A 21 04 00 22 13 D2 ; .~..(...0.!.."..
9749  .byte 3E 0C 18 11 3E 40 CD 5D 2F 2A 13 D2 11 08 00 19 ; >...>@.]/*......
9759  .byte 22 13 D2 3E 42 CD 5D 2F 3A DF D2 C6 06 32 DF D2 ; "..>B.]/:....2..
9769  .byte E1 F1 22 36 D2 FD 77 0A DD 36 0D 0A DD 36 0E 0C ; .."6..w..6...6..
9779  .byte DD 7E 12 A7 28 1A 0E 00 41 51 DD 71 0A DD 71 0B ; .~..(...AQ.q..q.
9789  .byte DD 71 0C DD 35 12 C2 0B 98 DD 36 00 FF C3 0B 98 ; .q..5.....6.....
9799  .byte 21 06 02 22 15 D2 CD 28 33 38 41 ED 4B 02 D4 DD ; !.."...(38A.K...
97A9  .byte 5E 05 DD 56 06 21 F8 FF 19 A7 ED 42 30 2E 21 06 ; ^..V.!.....B0.!.
97B9  .byte 00 19 A7 ED 42 38 25 DD 7E 12 A7 20 1F AF 6F 67 ; ....B8%.~.. ..og
97C9  .byte 22 07 D4 32 09 D4 32 88 D2 22 96 D2 FD CB 08 D6 ; "..2..2.."......
97D9  .byte 3E 20 32 F6 D2 DD 36 12 10 3E 22 EF DD 36 0A 98 ; > 2...6..>"..6..
97E9  .byte DD 36 0B FF DD 36 0C FF DD 7E 11 E6 0F 20 1C CD ; .6...6...~... ..
97F9  .byte 88 06 01 20 00 16 00 E6 3F FE 20 38 05 01 E0 FF ; ... ....?. 8....
9809  .byte 16 FF DD 71 07 DD 70 08 DD 72 09 DD 6E 02 DD 66 ; ...q..p..r..n..f
9819  .byte 03 EB 2A 54 D2 01 08 00 AF ED 42 30 02 6F 67 A7 ; ..*T......B0.og.
9829  .byte ED 52 30 33 2A 54 D2 01 00 01 09 A7 ED 52 38 27 ; .R03*T.......R8'
9839  .byte DD 6E 05 DD 66 06 EB 2A DD D2 A7 ED 52 30 18 2A ; .n..f..*....R0.*
9849  .byte 57 D2 01 F0 FF 09 A7 ED 52 30 0C 2A 57 D2 01 C0 ; W.......R0.*W...
9859  .byte 00 09 A7 ED 52 30 04 DD 36 00 FF DD 34 11 C9 DD ; ....R0..6...4...
9869  .byte CB 18 EE DD 36 0F 53 DD 36 10 9A FD CB 03 6E 20 ; ....6.S.6.....n 
9879  .byte 13 DD 7E 11 DD 77 12 DD 7E 11 FE 05 30 0F DD 34 ; ..~..w..~...0..4
9889  .byte 11 C3 96 98 DD 7E 11 A7 28 03 DD 35 11 DD 7E 11 ; .....~..(..5..~.
9899  .byte FE 01 30 21 21 0C 14 22 15 D2 DD 36 0D 1E DD 36 ; ..0!!.."...6...6
98A9  .byte 0E 16 CD 28 33 D8 01 73 99 CD 84 9A D0 0E FF 11 ; ...(3..s........
98B9  .byte FC FF C3 58 99 FE 04 D2 32 99 DD 36 0F 65 DD 36 ; ...X....2..6.e.6
98C9  .byte 10 9A 21 0F 08 22 15 D2 DD 36 0D 1E DD 36 0E 16 ; ..!.."...6...6..
98D9  .byte CD 28 33 D8 01 93 99 CD 84 9A D0 DD 7E 12 DD BE ; .(3.........~...
98E9  .byte 11 D0 3A FF D3 C6 08 E6 1F 87 4F 06 00 21 D3 99 ; ..:.......O..!..
98F9  .byte 09 5E 23 56 2A 04 D4 3A 06 D4 19 CE FF 22 04 D4 ; .^#V*..:....."..
9909  .byte 32 06 D4 21 13 9A 09 5E 23 56 2A 07 D4 7D 2F 6F ; 2..!...^#V*..}/o
9919  .byte 7C 2F 67 3A 09 D4 2F 19 CE FF 22 07 D4 32 09 D4 ; |/g:../..."..2..
9929  .byte C9 0E 00 11 08 00 C3 58 99 DD 36 0F 77 DD 36 10 ; .......X..6.w.6.
9939  .byte 9A 21 1A 02 22 15 D2 DD 36 0D 1E DD 36 0E 16 CD ; .!.."...6...6...
9949  .byte 28 33 D8 01 B3 99 CD 84 9A D0 0E 00 11 1A 00 3A ; (3.............:
9959  .byte E9 D2 2A E7 D2 22 07 D4 32 09 D4 2A 04 D4 3A 06 ; ..*.."..2..*..:.
9969  .byte D4 19 89 22 04 D4 32 06 D4 C9 FF FF FE FE FE FD ; ..."..2.........
9979  .byte FD FD FC FC FC FC FB FB FB FB FA FA FA FA FA F9 ; ................
9989  .byte F9 F9 F9 F9 F9 FA FA FB FC FE EA EA EA F6 F7 F8 ; ................
9999  .byte F8 F8 F9 F9 F9 FA FA FA FB FB FB FB FC FC FC FC ; ................
99A9  .byte FD FD FD FD FE FE FF 00 02 04 EA EA EA EA EA EA ; ................
99B9  .byte EA EA EA EA EA EA EE ED EC EC EC ED EE EF F0 F2 ; ................
99C9  .byte F3 F4 F5 F7 F8 F9 FA FB FD FF 00 F8 00 F8 00 F9 ; ................
99D9  .byte 00 FA 00 FB 00 FC E0 FC 80 FD C0 FD 00 FE 40 FE ; ..............@.
99E9  .byte 80 FE C0 FE 00 FF 20 FF 40 FF 60 FF 80 FF A0 FF ; ...... .@.`.....
99F9  .byte C0 FF E0 FF E8 FF EA FF EC FF EE FF F0 FF F2 FF ; ................
9A09  .byte F4 FF F6 FF F8 FF FC FF FE FF 00 FC 00 FC 00 FC ; ................
9A19  .byte 00 FB 00 FA 00 F9 00 F8 00 F7 00 F6 80 F5 00 F5 ; ................
9A29  .byte C0 F4 80 F4 40 F4 00 F4 00 F4 00 F4 00 F4 40 F4 ; ....@.........@.
9A39  .byte 80 F4 C0 F4 00 F5 00 F6 00 F7 00 F9 00 FA 00 FC ; ................
9A49  .byte 80 FC 00 FD C0 FD 00 FF 00 FF FE FF FF FF FF FF ; ................
9A59  .byte 38 3A 3C 3E FF FF FF FF FF FF FF FF 48 4A 4C 4E ; 8:<>........HJLN
9A69  .byte FF FF 68 6A 6C 6E FF FF FF FF FF FF FF FF FE 12 ; ..hjln..........
9A79  .byte 14 16 FF FF FE 32 34 36 FF FF FF 3A 09 D4 A7 F8 ; .....246...:....
9A89  .byte 3A FF D3 C6 08 E6 1F 6F 26 00 09 06 00 4E CB 79 ; :......o&....N.y
9A99  .byte 28 01 05 DD 6E 05 DD 66 06 09 22 02 D4 3A 08 D4 ; (...n..f.."..:..
9AA9  .byte FE 03 30 02 37 C9 11 01 00 2A 07 D4 7D 2F 6F 7C ; ..0.7....*..}/o|
9AB9  .byte 2F 67 3A 09 D4 2F 19 CE 00 CB 2F CB 1C CB 1D 22 ; /g:../..../...."
9AC9  .byte 07 D4 32 09 D4 A7 C9 DD CB 18 EE DD 36 0D 1C DD ; ..2.........6...
9AD9  .byte 36 0E 06 DD 36 0F 43 DD 36 10 9B 21 01 00 DD 7E ; 6...6.C.6..!...~
9AE9  .byte 12 FE 60 30 03 21 FF FF DD 36 07 00 DD 75 08 DD ; ..`0.!...6...u..
9AF9  .byte 74 09 3C FE C0 38 01 AF DD 77 12 DD 7E 11 A7 20 ; t.<..8...w..~.. 
9B09  .byte 35 21 02 06 22 15 D2 CD 28 33 D8 3A E9 D2 ED 5B ; 5!.."...(3.:...[
9B19  .byte E7 D2 4F 2A 07 D4 7D 2F 6F 7C 2F 67 3A 09 D4 2F ; ..O*..}/o|/g:../
9B29  .byte 19 89 11 01 00 19 CE 00 22 07 D4 32 09 D4 DD 36 ; ........"..2...6
9B39  .byte 11 08 3E 07 EF C9 DD 35 11 C9 08 0A 28 2A FF FF ; ..>....5....(*..
9B49  .byte FF DD CB 18 EE DD 36 0D 1E DD 36 0E 60 21 18 00 ; ......6...6.`!..
9B59  .byte 22 15 D2 CD 28 33 38 45 DD 6E 02 DD 66 03 7D 87 ; "...(38E.n..f.}.
9B69  .byte CB 14 87 CB 14 87 CB 14 5C DD 6E 05 DD 66 06 7D ; ........\.n..f.}
9B79  .byte 87 CB 14 87 CB 14 87 CB 14 54 21 AE 9B 06 05 7E ; .........T!....~
9B89  .byte 23 BB 20 15 7E BA 20 11 23 7E 32 D4 D2 3E 01 32 ; #. .~. .#~2..>.2
9B99  .byte 83 D2 FD CB 06 E6 C3 A6 9B 23 23 10 E2 AF DD 77 ; .........##....w
9BA9  .byte 0F DD 77 10 C9 7C 19 15 7C 01 14 01 3B 18 01 01 ; ..w..|..|...;...
9BB9  .byte 19 14 0F 1A DD 36 07 80 DD 36 08 01 DD 36 09 00 ; .....6...6...6..
9BC9  .byte DD 36 0F 3E DD 36 10 9C DD CB 18 EE DD CB 18 46 ; .6.>.6.........F
9BD9  .byte 20 13 DD 7E 02 DD 77 11 DD 7E 03 DD 77 12 3E 18 ;  ..~..w..~..w.>.
9BE9  .byte EF DD CB 18 C6 DD 36 0D 06 DD 36 0E 08 DD 7E 13 ; ......6...6...~.
9BF9  .byte FE 64 30 0C 21 00 04 22 15 D2 CD 28 33 D4 D9 2F ; .d0.!.."...(3../
9C09  .byte DD 34 13 DD 7E 13 FE 64 D8 FE F0 38 17 AF DD 77 ; .4..~..d...8...w
9C19  .byte 01 DD 77 13 DD 7E 11 DD 77 02 DD 7E 12 DD 77 03 ; ..w..~..w..~..w.
9C29  .byte 3E 18 EF C9 AF DD 77 0F DD 77 10 DD 77 07 DD 77 ; >.....w..w..w..w
9C39  .byte 08 DD 77 09 C9 0C 0E FF FF FF FF FF DD 36 07 80 ; ..w..........6..
9C49  .byte DD 36 08 FE DD 36 09 FF DD 36 0F 5C DD 36 10 9C ; .6...6...6.\.6..
9C59  .byte C3 D1 9B 2C 2E FF FF FF FF FF DD CB 18 EE DD CB ; ...,............
9C69  .byte 18 46 20 1A DD 6E 02 DD 66 03 11 0C 00 19 DD 75 ; .F ..n..f......u
9C79  .byte 02 DD 74 03 CD 88 06 DD 77 11 DD CB 18 C6 21 33 ; ..t.....w.....!3
9C89  .byte 9D CD DF 9C 21 13 9D 19 7E 23 66 6F B4 28 33 DD ; ....!...~#fo.(3.
9C99  .byte 7E 11 87 87 87 E6 1F 5F 16 00 19 06 04 C5 7E 23 ; ~......_......~#
9CA9  .byte 5E 23 16 00 E5 ED 53 15 D2 CD 5D 2F E1 C1 10 ED ; ^#....S...]/....
9CB9  .byte DD 7E 0E A7 28 0C 21 02 02 22 15 D2 CD 28 33 D4 ; .~..(.!.."...(3.
9CC9  .byte D9 2F DD 34 11 AF DD 77 0F DD 77 10 DD 7E 11 FE ; ./.4...w..w..~..
9CD9  .byte 70 C0 3E 17 EF C9 DD 7E 11 CB 3F CB 3F CB 3F CB ; p.>....~..?.?.?.
9CE9  .byte 3F 4F 06 00 87 5F 16 00 09 7E DD 77 0E DD 36 0D ; ?O..._...~.w..6.
9CF9  .byte 06 DD 6E 02 DD 66 03 22 0F D2 DD 6E 05 DD 66 06 ; ..n..f."...n..f.
9D09  .byte 22 11 D2 21 00 00 22 13 D2 C9 00 00 00 00 00 00 ; "..!..".........
9D19  .byte 00 00 00 00 00 00 00 00 63 9D 83 9D A3 9D 43 9D ; ........c.....C.
9D29  .byte 43 9D 43 9D A3 9D 83 9D 63 9D 00 00 00 00 00 00 ; C.C.....c.......
9D39  .byte 00 1B 1F 22 25 25 25 22 1F 1B 00 15 1E 0E 1E 07 ; ..."%%%"........
9D49  .byte 1E 00 00 17 1E 10 1E 09 1E 02 00 19 1E 12 1E 0B ; ................
9D59  .byte 1E 04 00 1B 1E 14 1E 0D 1E 06 00 0C 1E 08 1E 04 ; ................
9D69  .byte 1E 00 00 0E 1E 0A 1E 06 1E 02 00 10 1E 0C 1E 08 ; ................
9D79  .byte 1E 04 00 11 1E 0E 1E 0A 1E 06 00 0F 1E 0A 1E 05 ; ................
9D89  .byte 1E 00 00 11 1E 0C 1E 07 1E 02 00 13 1E 0E 1E 09 ; ................
9D99  .byte 1E 04 00 15 1E 10 1E 0B 1E 06 00 12 1E 0C 1E 06 ; ................
9DA9  .byte 1E 00 00 14 1E 0E 1E 08 1E 02 00 16 1E 10 1E 0A ; ................
9DB9  .byte 1E 04 00 18 1E 12 1E 0C 1E 06 DD CB 18 EE CD 9D ; ................
9DC9  .byte 9E DD 7E 11 FE 28 30 2B 21 05 00 22 15 D2 CD 28 ; ..~..(0+!.."...(
9DD9  .byte 33 38 20 11 05 00 3A 06 D4 A7 FA E9 9D 11 F4 FF ; 38 ...:.........
9DE9  .byte DD 6E 02 DD 66 03 19 22 FF D3 AF 6F 67 22 04 D4 ; .n..f.."...og"..
9DF9  .byte 32 06 D4 DD 6E 02 DD 66 03 11 D0 FF 19 ED 5B FF ; 2...n..f......[.
9E09  .byte D3 AF ED 52 30 32 DD 6E 02 DD 66 03 A7 ED 52 38 ; ...R02.n..f...R8
9E19  .byte 27 DD 6E 05 DD 66 06 11 E0 FF 19 ED 5B 02 D4 AF ; '.n..f......[...
9E29  .byte ED 52 30 14 DD 6E 05 DD 66 06 01 50 00 09 A7 ED ; .R0..n..f..P....
9E39  .byte 52 38 05 CD 7D 9E 18 03 CD 8D 9E 11 F4 9E DD 7E ; R8..}..........~
9E49  .byte 11 E6 0F 4F 06 00 DD 6E 12 DD 66 13 A7 ED 42 DD ; ...O...n..f...B.
9E59  .byte 75 05 DD 74 06 DD 7E 11 CB 3F CB 3F CB 3F CB 3F ; u..t..~..?.?.?.?
9E69  .byte E6 03 87 4F 87 87 87 81 4F 06 00 EB 09 DD 75 0F ; ...O....O.....u.
9E79  .byte DD 74 10 C9 DD 7E 11 FE 30 D0 3C DD 77 11 3D C0 ; .t...~..0.<.w.=.
9E89  .byte 3E 19 EF C9 DD 7E 11 A7 C8 3D DD 77 11 FE 2F C0 ; >....~...=.w../.
9E99  .byte 3E 19 EF C9 DD 36 0D 04 DD 7E 11 CB 3F CB 3F CB ; >....6...~..?.?.
9EA9  .byte 3F CB 3F E6 03 5F 3E 03 93 87 87 87 87 DD 77 0E ; ?.?.._>.......w.
9EB9  .byte DD CB 18 46 C0 01 00 00 11 F0 FF CD D5 30 11 14 ; ...F.........0..
9EC9  .byte 00 7E FE A3 28 07 11 04 00 DD CB 18 CE DD 6E 02 ; .~..(.........n.
9ED9  .byte DD 66 03 19 DD 75 02 DD 74 03 DD 7E 05 DD 77 12 ; .f...u..t..~..w.
9EE9  .byte DD 7E 06 DD 77 13 DD CB 18 C6 C9 0A FF FF FF FF ; .~..w...........
9EF9  .byte FF 3E FF FF FF FF FF 0A FF FF FF FF FF 3E FF FF ; .>...........>..
9F09  .byte FF FF FF 0A FF FF FF FF FF FF FF FF FF FF FF 0A ; ................
9F19  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF ; ................
9F29  .byte FF FF DD CB 18 EE CD 9D 9E DD 7E 11 FE 28 30 2C ; ..........~..(0,
9F39  .byte 21 05 00 22 15 D2 CD 28 33 38 21 11 05 00 3A 06 ; !.."...(38!...:.
9F49  .byte D4 A7 FA 51 9F 11 F4 FF DD 6E 02 DD 66 03 19 22 ; ...Q.....n..f.."
9F59  .byte FF D3 AF 32 04 D4 32 05 D4 32 06 D4 DD 6E 02 DD ; ...2..2..2...n..
9F69  .byte 66 03 11 F4 FF 19 ED 5B FF D3 AF ED 52 30 36 DD ; f......[....R06.
9F79  .byte 6E 02 DD 66 03 01 24 00 09 A7 ED 52 38 27 DD 6E ; n..f..$....R8'.n
9F89  .byte 05 DD 66 06 11 E0 FF 19 ED 5B 02 D4 AF ED 52 30 ; ..f......[....R0
9F99  .byte 14 DD 6E 05 DD 66 06 01 50 00 09 A7 ED 52 38 05 ; ..n..f..P....R8.
9FA9  .byte CD 7D 9E 18 03 CD 8D 9E 11 B7 9F C3 47 9E 36 FF ; .}..........G.6.
9FB9  .byte FF FF FF FF 3E FF FF FF FF FF 36 FF FF FF FF FF ; ....>.....6.....
9FC9  .byte 3E FF FF FF FF FF 36 FF FF FF FF FF FF FF FF FF ; >.....6.........
9FD9  .byte FF FF 36 FF FF FF FF FF FF FF FF FF FF FF FF FF ; ..6.............
9FE9  .byte FF FF FF FF FF DD CB 18 EE CD 9D 9E DD 7E 11 FE ; .............~..
9FF9  .byte 28 30 2C 21 05 00 22 15 D2 CD 28 33 38 21 11 05 ; (0,!.."...(38!..
A009  .byte 00 3A 06 D4 A7 FA 14 A0 11 F4 FF DD 6E 02 DD 66 ; .:..........n..f
A019  .byte 03 19 22 FF D3 AF 32 04 D4 32 05 D4 32 06 D4 DD ; .."...2..2..2...
A029  .byte 6E 02 DD 66 03 11 D0 FF 19 ED 5B FF D3 AF ED 52 ; n..f......[....R
A039  .byte 30 36 DD 6E 02 DD 66 03 01 24 00 09 A7 ED 52 38 ; 06.n..f..$....R8
A049  .byte 27 DD 6E 05 DD 66 06 11 E0 FF 19 ED 5B 02 D4 AF ; '.n..f......[...
A059  .byte ED 52 30 14 DD 6E 05 DD 66 06 01 50 00 09 A7 ED ; .R0..n..f..P....
A069  .byte 52 38 05 CD 7D 9E 18 03 CD 8D 9E 11 7A A0 C3 47 ; R8..}.......z..G
A079  .byte 9E 38 FF FF FF FF FF 3E FF FF FF FF FF 38 FF FF ; .8.....>.....8..
A089  .byte FF FF FF 3E FF FF FF FF FF 38 FF FF FF FF FF FF ; ...>.....8......
A099  .byte FF FF FF FF FF 38 FF FF FF FF FF FF FF FF FF FF ; .....8..........
A0A9  .byte FF FF FF FF FF FF FF FF DD CB 18 EE DD 36 0D 2A ; .............6.*
A0B9  .byte DD 36 0E 0C DD CB 18 46 20 24 DD 6E 02 DD 66 03 ; .6.....F $.n..f.
A0C9  .byte 11 18 00 19 DD 75 02 DD 74 03 DD 6E 05 DD 66 06 ; .....u..t..n..f.
A0D9  .byte 11 10 00 19 DD 75 05 DD 74 06 DD CB 18 C6 DD 7E ; .....u..t......~
A0E9  .byte 11 FE 64 38 1D 20 03 3E 13 EF 21 03 02 22 15 D2 ; ..d8. .>..!.."..
A0F9  .byte CD 28 33 D4 D9 2F 11 3C A1 01 30 A1 CD 75 7C C3 ; .(3../.<..0..u|.
A109  .byte 22 A1 FE 46 30 0A AF DD 77 0F DD 77 10 C3 22 A1 ; "..F0...w..w..".
A119  .byte 11 3C A1 01 37 A1 CD 75 7C DD 34 11 DD 7E 11 FE ; .<..7..u|.4..~..
A129  .byte A0 D8 DD 36 11 00 C9 00 01 01 01 02 01 FF 02 01 ; ...6............
A139  .byte 03 01 FF 02 04 FF FF FF FF FF FF FF FF FF FF FF ; ................
A149  .byte FF FF FF FF FF FE FE FE FE 02 04 FF FF FF FF FF ; ................
A159  .byte FF FF FF FF FF FF FF FE FE 16 18 FF FF FF FF FF ; ................
A169  .byte FF FF FF FF FF FF FF FF FF FF DD 36 0D 0A DD 36 ; ...........6...6
A179  .byte 0E 20 21 03 08 22 15 D2 CD 28 33 21 00 0E 22 0F ; . !.."...(3!..".
A189  .byte D2 D4 C1 2F DD 36 0A 00 DD 36 0B 01 DD 36 0C 00 ; .../.6...6...6..
A199  .byte DD 6E 02 DD 66 03 11 0A 00 19 EB 2A FF D3 01 08 ; .n..f......*....
A1A9  .byte 00 09 A7 ED 52 30 76 01 9B A2 DD 7E 11 FE EB 38 ; ....R0v....~...8
A1B9  .byte 09 20 04 DD 36 16 00 01 A0 A2 11 A3 A2 CD 75 7C ; . ..6.........u|
A1C9  .byte DD 7E 11 FE ED C2 97 A2 CD AF 7C DA 97 A2 DD 5E ; .~........|....^
A1D9  .byte 02 DD 56 03 DD 4E 05 DD 46 06 DD E5 E5 DD E1 AF ; ..V..N..F.......
A1E9  .byte DD 36 00 1C DD 77 01 DD 73 02 DD 72 03 21 06 00 ; .6...w..s..r.!..
A1F9  .byte 09 DD 77 04 DD 75 05 DD 74 06 DD 77 11 DD 77 16 ; ..w..u..t..w..w.
A209  .byte DD 77 17 DD 77 07 DD 36 08 FF DD 36 09 FF DD 77 ; .w..w..6...6...w
A219  .byte 0A DD 36 0B 01 DD 77 0C DD E1 C3 97 A2 01 9B A2 ; ..6...w.........
A229  .byte DD 7E 11 FE EB 38 09 20 04 DD 36 16 00 01 A0 A2 ; .~...8. ..6.....
A239  .byte 11 D4 A2 CD 75 7C DD 7E 11 FE ED 20 51 CD AF 7C ; ....u|.~... Q..|
A249  .byte DA 97 A2 DD 5E 02 DD 56 03 DD 4E 05 DD 46 06 DD ; ....^..V..N..F..
A259  .byte E5 E5 DD E1 AF DD 36 00 1C DD 77 01 DD 73 02 DD ; ......6...w..s..
A269  .byte 72 03 21 06 00 09 DD 77 04 DD 75 05 DD 74 06 DD ; r.!....w..u..t..
A279  .byte 77 11 DD 77 16 DD 77 17 DD 77 07 DD 36 08 01 DD ; w..w..w..w..6...
A289  .byte 77 09 DD 77 0A DD 36 0B 01 DD 77 0C DD E1 DD 34 ; w..w..6...w....4
A299  .byte 11 C9 00 1C 01 06 FF 02 18 FF 40 42 FF FF FF FF ; ..........@B....
A2A9  .byte 60 62 FF FF FF FF FF FF FF FF FF FF 44 46 FF FF ; `b..........DF..
A2B9  .byte FF FF 64 66 FF FF FF FF FF FF FF FF FF FF 40 42 ; ..df..........@B
A2C9  .byte FF FF FF FF 68 6A FF FF FF FF FF 50 52 FF FF FF ; ....hj.....PR...
A2D9  .byte FF 70 72 FF FF FF FF FF FF FF FF FF FF 4C 4E FF ; .pr..........LN.
A2E9  .byte FF FF FF 6C 6E FF FF FF FF FF FF FF FF FF FF 50 ; ...ln..........P
A2F9  .byte 52 FF FF FF FF 48 4A FF FF FF FF FF DD CB 18 AE ; R....HJ.........
A309  .byte DD 36 0D 0A DD 36 0E 0F 21 01 01 22 15 D2 CD 28 ; .6...6..!.."...(
A319  .byte 33 D4 D9 2F DD CB 18 7E 28 0C DD 36 0A 00 DD 36 ; 3../...~(..6...6
A329  .byte 0B FD DD 36 0C FF DD 6E 0A DD 66 0B DD 7E 0C 11 ; ...6...n..f..~..
A339  .byte 1F 00 19 CE 00 DD 75 0A DD 74 0B DD 77 0C DD 7E ; ......u..t..w..~
A349  .byte 11 FE 82 30 0C 01 7A A3 11 84 A3 CD 75 7C C3 6C ; ...0..z.....u|.l
A359  .byte A3 20 07 DD 36 16 00 3E 01 EF 01 7D A3 11 84 A3 ; . ..6..>...}....
A369  .byte CD 75 7C DD 34 11 DD 7E 11 FE A5 D8 DD 36 00 FF ; .u|.4..~.....6..
A379  .byte C9 00 08 FF 01 0C 02 0C 03 0C FF 20 22 FF FF FF ; ........... "...
A389  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF 74 76 FF ; .............tv.
A399  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF 78 ; ...............x
A3A9  .byte 7A FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF ; z...............
A3B9  .byte FF 7C 7E FF FF FF FF FF DD 36 0D 0A DD 36 0E 11 ; .|~......6...6..
A3C9  .byte DD CB 18 46 20 14 DD 6E 02 DD 66 03 11 08 00 19 ; ...F ..n..f.....
A3D9  .byte DD 75 02 DD 74 03 DD CB 18 C6 21 01 00 22 15 D2 ; .u..t.....!.."..
A3E9  .byte CD 28 33 38 3F 3A 09 D4 A7 FA 2D A4 DD 36 0F 54 ; .(38?:....-..6.T
A3F9  .byte DD 36 10 A4 3A D5 D2 FE 03 20 08 DD 36 0F 64 DD ; .6..:.... ..6.d.
A409  .byte 36 10 A4 01 06 00 11 00 00 CD F5 7C DD CB 18 4E ; 6..........|...N
A419  .byte 20 2D DD CB 18 CE 21 18 D3 CD 8D 0B 7E A9 77 3E ;  -....!.....~.w>
A429  .byte 1A EF 18 1B DD CB 18 8E DD 36 0F 5C DD 36 10 A4 ; .........6.\.6..
A439  .byte 3A D5 D2 FE 03 20 08 DD 36 0F 6C DD 36 10 A4 AF ; :.... ..6.l.6...
A449  .byte DD 77 0A DD 36 0B 02 DD 77 0C C9 1A 1C FF FF FF ; .w..6...w.......
A459  .byte FF FF FF 3A 3C FF FF FF FF FF FF 38 3A FF FF FF ; ...:<......8:...
A469  .byte FF FF FF 34 36 FF FF FF FF FF FF DD CB 18 EE CD ; ...46...........
A479  .byte 9D 9E DD 7E 11 FE 28 30 2C 21 05 00 22 15 D2 CD ; ...~..(0,!.."...
A489  .byte 28 33 38 21 11 05 00 3A 06 D4 A7 FA 9A A4 11 F4 ; (38!...:........
A499  .byte FF DD 6E 02 DD 66 03 19 22 FF D3 AF 32 04 D4 32 ; ..n..f.."...2..2
A4A9  .byte 05 D4 32 06 D4 21 18 D3 CD 8D 0B DD CB 18 4E 28 ; ..2..!........N(
A4B9  .byte 06 7E A1 20 14 18 04 7E A1 28 0E DD 7E 11 FE 30 ; .~. ...~.(..~..0
A4C9  .byte 30 12 3C 3C DD 77 11 18 0B DD 7E 11 A7 28 05 3D ; 0.<<.w....~..(.=
A4D9  .byte 3D DD 77 11 11 E3 A4 C3 47 9E 3E FF FF FF FF FF ; =.w.....G.>.....
A4E9  .byte 38 FF FF FF FF FF 3E FF FF FF FF FF 38 FF FF FF ; 8.....>.....8...
A4F9  .byte FF FF 3E FF FF FF FF FF FF FF FF FF FF FF 3E FF ; ..>...........>.
A509  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF ; ................
A519  .byte FF DD 36 0D 06 DD 36 0E 10 3A 24 D2 E6 01 20 53 ; ..6...6..:$... S
A529  .byte 21 82 A6 DD CB 18 4E 28 03 21 32 A7 DD 5E 11 CB ; !.....N(.!2..^..
A539  .byte 23 16 00 19 4E 23 46 DD 6E 01 DD 66 02 DD 7E 03 ; #...N#F.n..f..~.
A549  .byte 09 CB 78 28 04 CE FF 18 02 CE 00 DD 75 01 DD 74 ; ..x(........u..t
A559  .byte 02 DD 77 03 21 AE A6 19 5E 23 56 DD 6E 12 DD 66 ; ..w.!...^#V.n..f
A569  .byte 13 19 DD 75 12 DD 74 13 0E 00 CB 7C 28 02 0E FF ; ...u..t....|(...
A579  .byte DD 71 14 DD 6E 02 DD 66 03 22 0F D2 DD 6E 05 DD ; .q..n..f."...n..
A589  .byte 66 06 22 11 D2 DD CB 18 4E 20 49 21 DA A6 DD 5E ; f.".....N I!...^
A599  .byte 11 16 00 19 3E 24 CD 51 A6 3E 26 CD 6B A6 3E 26 ; ....>$.Q.>&.k.>&
A5A9  .byte CD 51 A6 3E 26 CD 6B A6 DD 36 0D 06 21 02 08 22 ; .Q.>&.k..6..!.."
A5B9  .byte 15 D2 CD 28 33 21 00 00 22 0F D2 38 05 CD C1 2F ; ...(3!.."..8.../
A5C9  .byte 18 59 DD 36 0D 16 21 06 08 22 15 D2 CD 28 33 D4 ; .Y.6..!.."...(3.
A5D9  .byte D9 2F 18 47 21 5E A7 DD 5E 11 16 00 19 3E 2A CD ; ./.G!^..^....>*.
A5E9  .byte 51 A6 3E 28 CD 6B A6 3E 28 CD 51 A6 3E 28 CD 6B ; Q.>(.k.>(.Q.>(.k
A5F9  .byte A6 DD 36 0D 10 21 01 04 22 15 D2 CD 28 33 38 05 ; ..6..!.."...(38.
A609  .byte CD D9 2F 18 16 DD 36 0D 16 21 10 04 22 15 D2 CD ; ../...6..!.."...
A619  .byte 28 33 21 00 00 22 0F D2 D4 C1 2F DD 36 0B 01 3A ; (3!.."..../.6..:
A629  .byte 24 D2 E6 01 C0 DD 34 11 DD 7E 11 FE 16 D8 DD 36 ; $.....4..~.....6
A639  .byte 11 00 DD 34 15 DD 7E 15 FE 14 D8 DD 36 15 00 DD ; ...4..~.....6...
A649  .byte 7E 18 EE 02 DD 77 18 C9 E5 5E 16 00 ED 53 13 D2 ; ~....w...^...S..
A659  .byte DD 6E 13 DD 66 14 22 15 D2 CD 5D 2F E1 11 16 00 ; .n..f."...]/....
A669  .byte 19 C9 E5 5E 16 00 ED 53 13 D2 21 00 00 22 15 D2 ; ...^...S..!.."..
A679  .byte CD 5D 2F E1 11 16 00 19 C9 00 00 00 00 00 00 00 ; .]/.............
A689  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 E0 ; ................
A699  .byte FF E0 FF E0 FF E0 FF C0 FF C0 FF 80 FF 80 FF 00 ; ................
A6A9  .byte FF 00 FF 00 FE 00 FF 80 FF 80 FF C0 FF C0 FF E0 ; ................
A6B9  .byte FF E0 FF F0 FF F0 FF F0 FF F0 FF 10 00 10 00 10 ; ................
A6C9  .byte 00 10 00 20 00 20 00 40 00 40 00 80 00 80 00 00 ; ... . .@.@......
A6D9  .byte 01 00 01 02 02 03 03 03 03 03 03 03 03 03 03 03 ; ................
A6E9  .byte 03 03 03 02 02 01 00 07 07 07 07 07 07 07 07 07 ; ................
A6F9  .byte 07 07 07 07 07 07 07 07 07 07 07 07 07 0E 0D 0C ; ................
A709  .byte 0C 0B 0B 0B 0B 0B 0B 0B 0B 0B 0B 0B 0B 0B 0B 0C ; ................
A719  .byte 0C 0D 0E 15 13 12 11 10 10 0F 0F 0F 0F 0F 0F 0F ; ................
A729  .byte 0F 0F 0F 10 10 11 12 13 15 00 00 00 00 00 00 00 ; ................
A739  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 20 ; ............... 
A749  .byte 00 20 00 20 00 20 00 40 00 40 00 80 00 80 00 00 ; . . . .@.@......
A759  .byte 01 00 01 00 02 15 14 13 13 12 12 12 12 12 12 12 ; ................
A769  .byte 12 12 12 12 12 12 12 13 13 14 15 0E 0E 0E 0E 0E ; ................
A779  .byte 0E 0E 0E 0E 0E 0E 0E 0E 0E 0E 0E 0E 0E 0E 0E 0E ; ................
A789  .byte 0E 07 08 09 09 0A 0A 0A 0A 0A 0A 0A 0A 0A 0A 0A ; ................
A799  .byte 0A 0A 0A 09 09 08 07 00 02 03 04 05 05 06 06 06 ; ................
A7A9  .byte 06 06 06 06 06 06 06 05 05 04 03 02 00 DD 36 0D ; ..............6.
A7B9  .byte 1E DD 36 0E 2F DD CB 18 46 20 32 21 40 03 22 6D ; ..6./...F 2!@."m
A7C9  .byte D2 21 40 05 22 6F D2 2A 57                      ; .!@."o.*W

; ==== sub_A7D2 (1 caller) ====
A7D2  D2 22 71    JP NC,$7122
A7D5  D2 22 73    JP NC,$7322
A7D8  D2 21 20    JP NC,$2021
A7DB  02          LD (BC),A
A7DC  22 77 D2    LD ($D277),HL
A7DF  21 0C F0    LD HL,$F00C
A7E2  11 00 20    LD DE,$2000
A7E5  3E 0C       LD A,$0C
A7E7  CD 06 04    CALL $0406
A7EA  3E 11       LD A,$11
A7EC  32 2D D2    LD ($D22D),A
A7EF  3E 0B       LD A,$0B
A7F1  DF          RST $18
A7F2  DD CB 18 C6 SET 0,(IX+24)
A7F6  DD CB 18 4E BIT 1,(IX+24)
A7FA  20 5D       JR NZ,$A859
A7FC  2A 54 D2    LD HL,($D254)
A7FF  22 6D D2    LD ($D26D),HL
A802  11 0A BB    LD DE,$BB0A
A805  01 7D A9    LD BC,$A97D
A808  CD 75 7C    CALL $7C75
A80B  DD 6E 02    LD L,(IX+2)
A80E  DD 66 03    LD H,(IX+3)
A811  ED 5B FF D3 LD DE,($D3FF)
A815  AF          XOR A
A816  ED 52       SBC HL,DE
A818  11 40 00    LD DE,$0040
A81B  AF          XOR A
A81C  ED 4B 04 D4 LD BC,($D404)
A820  CB 78       BIT 7,B
A822  20 04       JR NZ,$A828
A824  ED 52       SBC HL,DE
A826  38 03       JR C,$A82B
A828  01 80 FF    LD BC,$FF80
A82B  04          INC B
A82C  DD 71 07    LD (IX+7),C
A82F  DD 70 08    LD (IX+8),B
A832  DD 77 09    LD (IX+9),A
A835  DD 6E 02    LD L,(IX+2)
A838  DD 66 03    LD H,(IX+3)
A83B  11 A0 05    LD DE,$05A0
A83E  AF          XOR A
A83F  ED 52       SBC HL,DE
A841  DA 3A A9    JP C,$A93A
A844  6F          LD L,A
A845  67          LD H,A
A846  DD 77 07    LD (IX+7),A
A849  DD 77 08    LD (IX+8),A
A84C  22 04 D4    LD ($D404),HL
A84F  32 06 D4    LD ($D406),A
A852  DD CB 18 CE SET 1,(IX+24)
A856  C3 3A A9    JP $A93A
A859  DD CB 18 56 BIT 2,(IX+24)
A85D  20 34       JR NZ,$A893
A85F  21 30 05    LD HL,$0530
A862  11 20 02    LD DE,$0220
A865  CD C0 7C    CALL $7CC0
A868  FD 36 03 FF LD (IY+3),$FF
A86C  21 A0 05    LD HL,$05A0
A86F  DD 36 01 00 LD (IX+1),$00
A873  DD 75 02    LD (IX+2),L
A876  DD 74 03    LD (IX+3),H
A879  DD 36 0F 0A LD (IX+15),$0A
A87D  DD 36 10 BB LD (IX+16),$BB
A881  DD 34 11    INC (IX+17)
A884  DD 7E 11    LD A,(IX+17)
A887  FE C0       CP $C0
A889  DA 3A A9    JP C,$A93A
A88C  DD CB 18 D6 SET 2,(IX+24)
A890  C3 3A A9    JP $A93A
A893  DD CB 18 5E BIT 3,(IX+24)
A897  20 18       JR NZ,$A8B1
A899  FD 36 03 FF LD (IY+3),$FF
A89D  AF          XOR A
A89E  DD 77 0F    LD (IX+15),A
A8A1  DD 77 10    LD (IX+16),A
A8A4  DD 35 11    DEC (IX+17)
A8A7  C2 3A A9    JP NZ,$A93A
A8AA  DD CB 18 DE SET 3,(IX+24)
A8AE  C3 3A A9    JP $A93A
A8B1  DD CB 18 66 BIT 4,(IX+24)
A8B5  20 7A       JR NZ,$A931
A8B7  ED 5B FF D3 LD DE,($D3FF)
A8BB  21 96 05    LD HL,$0596
A8BE  A7          AND A
A8BF  ED 52       SBC HL,DE
A8C1  30 77       JR NC,$A93A
A8C3  21 C0 05    LD HL,$05C0
A8C6  AF          XOR A
A8C7  ED 52       SBC HL,DE
A8C9  38 6F       JR C,$A93A
A8CB  DD B6 11    OR (IX+17)
A8CE  20 13       JR NZ,$A8E3
A8D0  2A 02 D4    LD HL,($D402)
A8D3  11 8D 02    LD DE,$028D
A8D6  AF          XOR A
A8D7  ED 52       SBC HL,DE
A8D9  38 5F       JR C,$A93A
A8DB  6F          LD L,A
A8DC  67          LD H,A
A8DD  22 04 D4    LD ($D404),HL
A8E0  32 06 D4    LD ($D406),A
A8E3  3E 80       LD A,$80
A8E5  32 15 D4    LD ($D415),A
A8E8  21 A0 05    LD HL,$05A0
A8EB  22 FF D3    LD ($D3FF),HL
A8EE  FD 36 03 FF LD (IY+3),$FF
A8F2  DD 5E 11    LD E,(IX+17)
A8F5  16 00       LD D,$00
A8F7  21 8E 02    LD HL,$028E
A8FA  AF          XOR A
A8FB  ED 52       SBC HL,DE
A8FD  32 01 D4    LD ($D401),A
A900  22 02 D4    LD ($D402),HL
A903  3A E9 D2    LD A,($D2E9)
A906  2A E7 D2    LD HL,($D2E7)
A909  22 07 D4    LD ($D407),HL
A90C  32 09 D4    LD ($D409),A
A90F  DD 34 11    INC (IX+17)
A912  DD 7E 11    LD A,(IX+17)
A915  FE C0       CP $C0
A917  20 21       JR NZ,$A93A
A919  2A 54 D2    LD HL,($D254)
A91C  24          INC H
A91D  22 FF D3    LD ($D3FF),HL
A920  DD CB 18 E6 SET 4,(IX+24)
A924  3E 09       LD A,$09
A926  DF          RST $18
A927  3E A0       LD A,$A0
A929  32 83 D2    LD ($D283),A
A92C  FD CB 06 CE SET 1,(IY+6)
A930  C9          RET
A931  DD 7E 11    LD A,(IX+17)
A934  A7          AND A
A935  28 03       JR Z,$A93A
A937  DD 35 11    DEC (IX+17)
A93A  DD 5E 11    LD E,(IX+17)
A93D  16 00       LD D,$00
A93F  21 80 02    LD HL,$0280
A942  AF          XOR A
A943  ED 52       SBC HL,DE
A945  DD 77 04    LD (IX+4),A
A948  DD 75 05    LD (IX+5),L
A94B  DD 74 06    LD (IX+6),H
A94E  DD 5E 11    LD E,(IX+17)
A951  16 00       LD D,$00
A953  21 AF 02    LD HL,$02AF
A956  A7          AND A
A957  ED 52       SBC HL,DE
A959  ED 4B 57 D2 LD BC,($D257)
A95D  A7          AND A
A95E  ED 42       SBC HL,BC
A960  EB          EX DE,HL
A961  21 A0 05    LD HL,$05A0
A964  ED 4B 54 D2 LD BC,($D254)
A968  A7          AND A
A969  ED 42       SBC HL,BC
A96B  01 86 A9    LD BC,$A986
A96E  CD 07 2F    CALL $2F07
A971  DD 7E 11    LD A,(IX+17)
A974  E6 1F       AND $1F
A976  FE 0F       CP $0F
A978  C0          RET NZ
A979  3E 19       LD A,$19
A97B  EF          RST $28
A97C  C9          RET
A97D  .byte 03 08 04 07 05 08 04 07 FF 74 76 76 78 FF FF FF ; .........tvvx...
A98D  .byte DD CB 18 EE FD 7E 0A 2A 36 D2 F5 E5 3A DF D2 FE ; .....~.*6...:...
A99D  .byte 24 30 42 5F 16 00 21 00 D0 19 22 36 D2 3A 9E D2 ; $0B_..!..."6.:..
A9AD  .byte 4F ED 5B 9C D2 DD 6E 04 DD 66 05 DD 7E 06 19 89 ; O.[...n..f..~...
A9BD  .byte 6C 67 ED 4B 57 D2 A7 ED 42 EB DD 6E 02 DD 66 03 ; lg.KW...B..n..f.
A9CD  .byte ED 4B 54 D2 A7 ED 42 01 29 AA CD 07 2F 3A DF D2 ; .KT...B.).../:..
A9DD  .byte C6 0C 32 DF D2 E1 F1 22 36 D2 FD 77 0A 2A 54 D2 ; ..2...."6..w.*T.
A9ED  .byte 11 E0 FF 19 EB DD 6E 02 DD 66 03 A7 ED 52 30 17 ; ......n..f...R0.
A9FD  .byte CD 88 06 06 00 87 4F CB 10 2A 54 D2 11 B4 01 19 ; ......O..*T.....
AA0D  .byte 09 DD 75 02 DD 74 03 DD 36 07 00 DD 36 08 FD DD ; ..u..t..6...6...
AA1D  .byte 36 09 FF DD 36 0F 00 DD 36 10 00 C9 40 42 44 46 ; 6...6...6...@BDF
AA2D  .byte FF FF FF DD CB 18 EE DD 36 0D 05 DD 36 0E 14 DD ; ........6...6...
AA3D  .byte CB 18 46 20 24 DD 6E 02 DD 66 03 11 0F 00 19 DD ; ..F $.n..f......
AA4D  .byte 75 02 DD 74 03 DD 6E 05 DD 66 06 11 FA FF 19 DD ; u..t..n..f......
AA5D  .byte 75 05 DD 74 06 DD CB 18 C6 DD 6E 02 DD 66 03 22 ; u..t......n..f."
AA6D  .byte 0F D2 DD 6E 05 DD 66 06 22 11 D2 DD 5E 11 16 00 ; ...n..f."...^...
AA7D  .byte 21 C7 AA 19 5E 23 56 06 02 C5 1A 6F 26 00 22 13 ; !...^#V....o&.".
AA8D  .byte D2 13 1A 6F 22 15 D2 13 1A 13 A7 FA A0 AA D5 CD ; ...o"...........
AA9D  .byte 5D 2F D1 C1 10 E3 21 02 02 22 15 D2 CD 28 33 D4 ; ]/....!.."...(3.
AAAD  .byte D9 2F DD 36 0F 00 DD 36 10 00 DD 7E 11 3C 3C FE ; ./.6...6...~.<<.
AABD  .byte 08 DD 77 11 D8 DD 36 11 00 C9 CF AA D5 AA DB AA ; ..w...6.........
AACD  .byte E1 AA 00 00 1C 00 18 3C 00 00 1E 00 18 3E 00 00 ; .......<.....>..
AADD  .byte 38 00 18 3A 00 08 1A 00 00 FF DD 36 0D 08 DD 36 ; 8..:.......6...6
AAED  .byte 0E 10 DD 7E 11 FE 64 30 2A DD 6E 02 DD 66 03 11 ; ...~..d0*.n..f..
AAFD  .byte D0 FF 19 EB 2A FF D3 A7 ED 52 38 17 DD 6E 02 DD ; ....*....R8..n..
AB0D  .byte 66 03 11 28 00 19 EB 2A FF D3 A7 ED 52 30 04 DD ; f..(...*....R0..
AB1D  .byte 36 11 64 DD 7E 11 FE 1E 30 18 DD 36 07 F8 DD 36 ; 6.d.~...0..6...6
AB2D  .byte 08 FF DD 36 09 FF 11 D1 AC 01 B7 AC CD 75 7C C3 ; ...6.........u|.
AB3D  .byte 30 AC DD 7E 11 FE 64 DA E4 AB DD 36 07 00 DD 36 ; 0..~..d....6...6
AB4D  .byte 08 00 DD 36 09 00 FE 66 30 0C 11 D1 AC 01 C7 AC ; ...6...f0.......
AB5D  .byte CD 75 7C C3 30 AC DD 36 0F 19 DD 36 10 AD FE 67 ; .u|.0..6...6...g
AB6D  .byte C2 30 AC 21 FE FF 22 13 D2 21 FC FF 22 15 D2 CD ; .0.!.."..!.."...
AB7D  .byte AF 7C DA 3C AC 11 00 00 4B 42 CD 5C AC 21 03 00 ; .|.<....KB.\.!..
AB8D  .byte 22 13 D2 21 FC FF 22 15 D2 CD AF 7C DA 3C AC 11 ; "..!.."....|.<..
AB9D  .byte 08 00 01 00 00 CD 5C AC 21 FE FF 22 13 D2 21 FE ; ......\.!.."..!.
ABAD  .byte FF 22 15 D2 CD AF 7C DA 3C AC 11 00 00 01 08 00 ; ."....|.<.......
ABBD  .byte CD 5C AC 21 03 00 22 13 D2 21 FE FF 22 15 D2 CD ; .\.!.."..!.."...
ABCD  .byte AF 7C DA 3C AC 11 08 00 01 08 00 CD 5C AC DD 36 ; .|.<........\..6
ABDD  .byte 00 FF 3E 1B EF 18 58 FE 23 30 15 AF DD 77 07 DD ; ..>...X.#0...w..
ABED  .byte 77 08 DD 77 09 11 D1 AC 01 BC AC CD 75 7C 18 33 ; w..w........u|.3
ABFD  .byte DD 7E 11 FE 41 30 17 DD 36 07 08 DD 36 08 00 DD ; .~..A0..6...6...
AC0D  .byte 36 09 00 11 D1 AC 01 BF AC CD 75 7C 18 15 DD 36 ; 6.........u|...6
AC1D  .byte 07 00 DD 36 08 00 DD 36 09 00 11 D1 AC 01 C4 AC ; ...6...6........
AC2D  .byte CD 75 7C DD 36 0A 80 DD 36 0B 00 DD 36 0C 00 21 ; .u|.6...6...6..!
AC3D  .byte 02 04 22 15 D2 CD 28 33 D4 D9 2F 3A 24 D2 E6 3F ; .."...(3../:$..?
AC4D  .byte C0 DD 34 11 DD 7E 11 FE 46 C0 DD 36 11 00 C9 DD ; ..4..~..F..6....
AC5D  .byte E5 E5 DD 6E 02 DD 66 03 19 EB DD 6E 05 DD 66 06 ; ...n..f....n..f.
AC6D  .byte 09 4D 44 DD E1 AF DD 36 00 0D DD 77 01 DD 73 02 ; .MD....6...w..s.
AC7D  .byte DD 72 03 DD 77 04 DD 71 05 DD 70 06 DD 77 11 DD ; .r..w..q..p..w..
AC8D  .byte 36 13 24 DD 77 14 DD 77 15 DD 77 16 DD 77 17 DD ; 6.$.w..w..w..w..
AC9D  .byte 77 07 2A 13 D2 DD 75 08 DD 74 09 DD 77 0A 2A 15 ; w.*...u..t..w.*.
ACAD  .byte D2 DD 75 0B DD 74 0C DD E1 C9 00 20 01 20 FF 01 ; ..u..t..... . ..
ACBD  .byte 20 FF 02 20 03 20 FF 03 20 FF 01 02 04 02 FF 03 ;  .. . .. .......
ACCD  .byte 02 05 02 FF 0A 0C FF FF FF FF FF FF FF FF FF FF ; ................
ACDD  .byte FF FF FF FF FF FF 0E 10 FF FF FF FF FF FF FF FF ; ................
ACED  .byte FF FF FF FF FF FF FF FF 2A 2C FF FF FF FF FF FF ; ........*,......
ACFD  .byte FF FF FF FF FF FF FF FF FF FF 2E 30 FF FF FF FF ; ...........0....
AD0D  .byte FF FF FF FF FF FF FF FF FF FF FF FF 12 14 FF FF ; ................
AD1D  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF FF 32 34 ; ..............24
AD2D  .byte FF FF FF FF FF DD CB 18 EE DD CB 18 46 20 1A DD ; ............F ..
AD3D  .byte 6E 02 DD 66 03 11 FC FF 19 DD 75 02 DD 74 03 CD ; n..f......u..t..
AD4D  .byte 88 06 DD 77 11 DD CB 18 C6 DD 7E 11 FE 64 20 46 ; ...w......~..d F
AD5D  .byte CD AF 7C 38 41 DD E5 DD 5E 02 DD 56 03 DD 4E 05 ; ..|8A...^..V..N.
AD6D  .byte DD 46 06 E5 DD E1 AF DD 36 00 34 DD 77 01 21 04 ; .F......6.4.w.!.
AD7D  .byte 00 19 DD 75 02 DD 74 03 DD 77 04 21 10 00 09 DD ; ...u..t..w.!....
AD8D  .byte 75 05 DD 74 06 DD E1 3E 1C EF DD 36 12 18 DD 36 ; u..t...>...6...6
AD9D  .byte 16 00 DD 36 17 00 DD 7E 12 A7 28 10 11 CA AD 01 ; ...6...~..(.....
ADAD  .byte C3 AD CD 75 7C DD 35 12 DD 34 11 C9 DD 77 0F DD ; ...u|.5..4...w..
ADBD  .byte 77 10 DD 34 11 C9 00 08 01 08 02 08 FF FE FF FF ; w..4............
ADCD  .byte FF FF FF 74 76 FF FF FF FF FF FF FF FF FF FF FE ; ...tv...........
ADDD  .byte FF FF FF FF FF 78 7A FF FF FF FF FF FF FF FF FF ; .....xz.........
ADED  .byte FF FE FF FF FF FF FF 7C 7E FF FF FF FF FF DD CB ; .......|~.......
ADFD  .byte 18 EE DD 36 0D 0C DD 36 0E 0C 2A 54 D2 11 10 01 ; ...6...6..*T....
AE0D  .byte 19 DD 5E 02 DD 56 03 A7 ED 52 30 04 DD 36 00 FF ; ..^..V...R0..6..
AE1D  .byte 21 02 02 22 15 D2 CD 28 33 D4 D9 2F AF DD 36 07 ; !.."...(3../..6.
AE2D  .byte 80 DD 36 08 02 DD 77 09 DD 77 0A DD 77 0B DD 77 ; ..6...w..w..w..w
AE3D  .byte 0C DD 36 0F 47 DD 36 10 AE C9 02 04 FF FF FF FF ; ..6.G.6.........
AE4D  .byte FF DD CB 18 EE DD CB 18 46 20 14 DD 36 11 00 DD ; ........F ..6...
AE5D  .byte 36 12 2A DD 36 13 52 DD 36 14 7C DD CB 18 C6 DD ; 6.*.6.R.6.|.....
AE6D  .byte 6E 02 DD 66 03 ED 5B FF D3 A7 ED 52 38 23 DD 36 ; n..f..[....R8#.6
AE7D  .byte 07 F8 DD 36 08 FF DD 36 09 FF DD 36 0F 9B DD 36 ; ...6...6...6...6
AE8D  .byte 10 B0 21 80 FF 22 17 D2 CD 5E AF DD 36 16 01 18 ; ..!.."...^..6...
AE9D  .byte 21 DD 36 07 08 DD 36 08 00 DD 36 09 00 DD 36 0F ; !.6...6...6...6.
AEAD  .byte AD DD 36 10 B0 21 80 00 22 17 D2 CD 5E AF DD 36 ; ..6..!.."...^..6
AEBD  .byte 16 FF DD 36 0D 1C DD 36 0E 1C 21 12 12 22 15 D2 ; ...6...6..!.."..
AECD  .byte CD 28 33 21 10 10 22 0F D2 D4 C1 2F DD 6E 02 DD ; .(3!.."..../.n..
AEDD  .byte 66 03 22 0F D2 DD 6E 05 DD 66 06 22 11 D2 DD E5 ; f."...n..f."....
AEED  .byte E1 11 11 00 19 06 04 C5 E5 7E FE FE 28 38 E6 FE ; .........~..(8..
AEFD  .byte 5F 16 00 21 F7 AF 19 E5 5E ED 53 13 D2 23 5E ED ; _..!....^.S..#^.
AF0D  .byte 53 15 D2 3E 24 CD 5D 2F E1 7E 3C 3C 32 15 D2 C6 ; S..>$.]/.~<<2...
AF1D  .byte 04 DD 77 0D 23 7E 3C 3C 32 16 D2 C6 04 DD 77 0E ; ..w.#~<<2.....w.
AF2D  .byte CD 28 33 D4 D9 2F E1 C1 7E FE FE 28 10 DD 86 16 ; .(3../..~..(....
AF3D  .byte FE FF 20 04 3E A3 18 05 FE A4 20 01 AF 77 23 10 ; .. .>..... ..w#.
AF4D  .byte A6 3A 24 D2 E6 07 C8 DD 7E 15 FE C8 D0 DD 34 15 ; .:$.....~.....4.
AF5D  .byte C9 DD 7E 15 FE C8 C0 3A D5 D2 FE 03 C0 DD 6E 05 ; ..~....:......n.
AF6D  .byte DD 66 06 11 D0 FF 19 ED 5B 02 D4 A7 ED 52 D0 DD ; .f......[....R..
AF7D  .byte 6E 05 DD 66 06 01 2C 00 09 A7 ED 52 D8 DD E5 E1 ; n..f..,....R....
AF8D  .byte 11 11 00 19 06 04 C5 E5 7E FE 4A CC A1 AF E1 C1 ; ........~.J.....
AF9D  .byte 23 10 F3 C9 36 FE CD AF 7C D8 DD E5 DD 5E 02 DD ; #...6...|....^..
AFAD  .byte 56 03 DD 4E 05 DD 46 06 E5 DD E1 AF DD 36 00 36 ; V..N..F......6.6
AFBD  .byte DD 77 01 21 12 00 19 DD 75 02 DD 74 03 DD 77 04 ; .w.!....u..t..w.
AFCD  .byte 21 1E 00 09 DD 75 05 DD 74 06 2A 17 D2 DD 75 07 ; !....u..t.*...u.
AFDD  .byte DD 74 08 AF CB 7C 28 02 3E FF DD 77 09 AF DD 77 ; .t...|(.>..w...w
AFED  .byte 0A DD 77 0B DD 77 0C DD E1 C9 0C 03 0D 03 0E 03 ; ..w..w..........
AFFD  .byte 0E 04 0F 04 10 04 10 05 11 05 11 06 12 06 12 07 ; ................
B00D  .byte 13 07 13 08 13 09 14 09 14 0A 14 0B 15 0B 15 0C ; ................
B01D  .byte 15 0D 15 0E 15 0F 15 10 15 11 14 11 14 12 14 13 ; ................
B02D  .byte 13 13 13 14 13 15 12 15 12 16 11 16 11 17 10 17 ; ................
B03D  .byte 10 18 0F 18 0E 18 0E 19 0D 19 0C 19 0B 19 0A 19 ; ................
B04D  .byte 09 19 09 18 08 18 07 18 07 17 06 17 06 16 05 16 ; ................
B05D  .byte 05 15 04 15 04 14 04 13 03 13 03 12 03 11 02 11 ; ................
B06D  .byte 02 10 02 0F 02 0E 02 0D 02 0C 02 0B 03 0B 03 0A ; ................
B07D  .byte 03 09 04 09 04 08 04 07 05 07 05 06 06 06 06 05 ; ................
B08D  .byte 07 05 07 04 08 04 09 04 09 03 0A 03 0B 03 FE FF ; ................
B09D  .byte FF FF FF FF FE 26 28 FF FF FF FF FF FF FF FF FF ; .....&(.........
B0AD  .byte FE FF FF FF FF FF FE 20 22 FF FF FF FF DD CB 18 ; ....... ".......
B0BD  .byte EE DD 36 0F 00 DD 36 10 00 DD 36 0D 04 DD 36 0E ; ..6...6...6...6.
B0CD  .byte 04 21 02 06 22 15 D2 CD 28 33 D4 D9 2F DD 6E 02 ; .!.."...(3../.n.
B0DD  .byte DD 66 03 22 0F D2 EB 2A 54 D2 01 F0 FF 09 A7 ED ; .f."...*T.......
B0ED  .byte 52 30 3D 2A 54 D2 01 10 01 09 A7 ED 52 38 31 DD ; R0=*T.......R81.
B0FD  .byte 6E 05 DD 66 06 22 11 D2 EB 2A 57 D2 01 F0 FF 09 ; n..f."...*W.....
B10D  .byte A7 ED 52 30 1B 2A 57 D2 01 D0 00 09 A7 ED 52 38 ; ..R0.*W.......R8
B11D  .byte 0F 21 00 00 22 13 D2 22 15 D2 3E 24 CD 5D 2F C9 ; .!..".."..>$.]/.
B12D  .byte DD 36 00 FF C9 DD CB 18 EE DD CB 18 46 20 0C CD ; .6..........F ..
B13D  .byte 88 06 E6 07 DD 77 11 DD CB 18 C6 DD 36 0F 00 DD ; .....w......6...
B14D  .byte 36 10 00 DD 6E 02 DD 66 03 22 0F D2 DD 6E 05 DD ; 6...n..f."...n..
B15D  .byte 66 06 22 11 D2 DD 7E 11 87 87 87 5F 16 00 21 ED ; f."...~...._..!.
B16D  .byte B1 19 06 02 C5 16 00 5E CB 7B 28 02 16 FF ED 53 ; .......^.{(....S
B17D  .byte 13 D2 23 16 00 5E CB 7B 28 02 16 FF ED 53 15 D2 ; ..#..^.{(....S..
B18D  .byte 23 7E 23 23 FE FF 28 05 E5 CD 5D 2F E1 C1 10 D4 ; #~##..(...]/....
B19D  .byte 3A 24 D2 E6 3F 20 09 DD 7E 11 3C E6 07 DD 77 11 ; :$..? ..~.<...w.
B1AD  .byte DD 34 12 DD 7E 12 FE 1A C0 DD 36 12 00 DD 7E 11 ; .4..~.....6...~.
B1BD  .byte 87 5F 87 83 5F 16 00 21 2D B2 19 5E 23 56 23 ED ; ._.._..!-..^#V#.
B1CD  .byte 53 13 D2 5E 23 56 ED 53 15 D2 23 5E 16 00 CB 7B ; S..^#V.S..#^...{
B1DD  .byte 28 01 15 23 4E 06 00 CB 79 28 01 05 CD 9C B5 C9 ; (..#N...y(......
B1ED  .byte 08 F8 66 00 00 00 FF 00 0C FA 70 00 14 FA 72 00 ; ..f.......p...r.
B1FD  .byte 0F 07 4C 00 17 07 4E 00 0D 0C 6C 00 15 0C 6E 00 ; ..L...N...l...n.
B20D  .byte 08 0F 64 00 00 00 FF 00 FC 0C 68 00 04 0C 6A 00 ; ..d.......h...j.
B21D  .byte F9 07 48 00 01 07 4A 00 FB F9 50 00 03 F9 52 00 ; ..H...J...P...R.
B22D  .byte 00 00 00 FE 08 F0 00 01 00 FF 18 F8 00 02 00 00 ; ................
B23D  .byte 1E 07 00 01 00 01 16 16 00 00 00 02 08 20 00 FF ; ............. ..
B24D  .byte 00 01 F8 18 00 FE 00 00 F2 07 00 FF 00 FF F7 F6 ; ................
B25D  .byte DD CB 18 EE DD CB 18 46 20 16 DD 7E 04 DD 77 12 ; .......F ..~..w.
B26D  .byte DD 7E 05 DD 77 13 DD 7E 06 DD 77 14 DD CB 18 C6 ; .~..w..~..w.....
B27D  .byte 3A 9E D2 4F ED 5B 9C D2 DD 6E 12 DD 66 13 DD 7E ; :..O.[...n..f..~
B28D  .byte 14 19 89 DD 75 04 DD 74 05 DD 77 06 3A 09 D4 A7 ; ....u..t..w.:...
B29D  .byte FA 03 B3 DD 36 0D 1E DD 36 0E 10 21 02 0A 22 15 ; ....6...6..!..".
B2AD  .byte D2 CD 28 33 38 50 01 10 00 11 00 00 CD F5 7C 01 ; ..(38P........|.
B2BD  .byte 20 00 11 10 00 CD D5 30 5E 16 00 3A D5 D2 87 4F ;  ......0^..:...O
B2CD  .byte 42 21 3D 34 09 7E 23 66 6F 19 7E E6 3F 20 27 DD ; B!=4.~#fo.~.? '.
B2DD  .byte 6E 01 DD 66 02 DD 7E 03 11 80 00 19 CE 00 DD 75 ; n..f..~........u
B2ED  .byte 01 DD 74 02 DD 77 03 2A FE D3 3A 00 D4 19 CE 00 ; ..t..w.*..:.....
B2FD  .byte 22 FE D3 32 00 D4 DD 6E 02 DD 66 03 22 0F D2 DD ; "..2...n..f."...
B30D  .byte 6E 05 DD 66 06 22 11 D2 21 F8 FF 22 13 D2 DD 5E ; n..f."..!.."...^
B31D  .byte 11 16 00 21 62 B3 19 06 02 C5 5E 16 00 23 ED 53 ; ...!b.....^..#.S
B32D  .byte 15 D2 7E 23 FE FF 28 05 E5 CD 5D 2F E1 C1 10 E9 ; ..~#..(...]/....
B33D  .byte DD 36 0F 55 DD 36 10 B3 DD 7E 11 C6 04 DD 77 11 ; .6.U.6...~....w.
B34D  .byte FE 10 D8 DD 36 11 00 C9 FE FF FF FF FF FF 36 36 ; ....6.........66
B35D  .byte 36 36 FF FF FF 08 1C 18 3C 08 1E 18 3E 08 38 18 ; 66......<...>.8.
B36D  .byte 3A 0C 1A 00 FF DD CB 18 EE DD CB 18 46 20 10 DD ; :...........F ..
B37D  .byte 6E 02 DD 66 03 DD 75 11 DD 74 12 DD CB 18 C6 DD ; n..f..u..t......
B38D  .byte 36 0D 0C DD 36 0E 2E DD 36 0F 35 DD 36 10 B4 21 ; 6...6...6.5.6..!
B39D  .byte 02 02 22 15 D2 CD 28 33 D4 D9 2F DD 6E 01 DD 66 ; .."...(3../.n..f
B3AD  .byte 02 DD 7E 03 11 80 00 19 CE 00 6C 67 22 0F D2 DD ; ..~.......lg"...
B3BD  .byte 6E 05 DD 66 06 22 11 D2 21 00 00 22 13 D2 21 F0 ; n..f."..!.."..!.
B3CD  .byte FF 22 15 D2 3E 16 CD 5D 2F 21 08 00 22 13 D2 3E ; ."..>..]/!.."..>
B3DD  .byte 18 CD 5D 2F DD 6E 02 DD 66 03 11 80 05 AF DD 77 ; ..]/.n..f......w
B3ED  .byte 07 DD 77 08 DD 77 09 ED 52 D0 DD 4E 05 DD 46 06 ; ..w..w..R..N..F.
B3FD  .byte 21 40 00 09 ED 5B 57 D2 A7 ED 52 30 0C DD 7E 11 ; !@...[W...R0..~.
B40D  .byte DD 77 02 DD 7E 12 DD 77 03 ED 5B 02 D4 21 E0 FF ; .w..~..w..[..!..
B41D  .byte 09 AF ED 52 D0 21 2C 00 09 AF ED 52 D8 DD 36 07 ; ...R.!,....R..6.
B42D  .byte 80 DD 77 08 DD 77 09 C9 16 18 FF FF FF FF 16 18 ; ..w..w..........
B43D  .byte FF FF FF FF 16 18 FF FF FF FF DD CB 18 EE DD CB ; ................
B44D  .byte 18 46 20 15 01 00 00 59 50 CD D5 30 7E D6 3C FE ; .F ....YP..0~.<.
B45D  .byte 04 D0 DD 77 11 DD CB 18 C6 DD 34 12 DD 7E 12 CB ; ...w......4..~..
B46D  .byte 77 C0 E6 0F C0 DD 7E 11 87 5F 87 87 83 5F 16 00 ; w.....~.._..._..
B47D  .byte 21 C0 B4 19 5E 23 56 23 ED 53 13 D2 5E 23 56 23 ; !...^#V#.S..^#V#
B48D  .byte ED 53 15 D2 5E 23 56 23 4E 23 46 23 D9 DD 5E 02 ; .S..^#V#N#F#..^.
B49D  .byte DD 56 03 2A FF D3 A7 ED 52 7C D9 BE C0 23 D9 DD ; .V.*....R|...#..
B4AD  .byte 5E 05 DD 56 06 2A 02 D4 A7 ED 52 7C D9 BE C0 CD ; ^..V.*....R|....
B4BD  .byte 9C B5 C9 80 FE 80 FE 00 00 F8 FF FF FF 80 01 80 ; ................
B4CD  .byte FE 18 00 F8 FF 00 FF 80 FE 80 01 00 00 10 00 FF ; ................
B4DD  .byte 00 80 01 80 01 18 00 10 00 00 00 DD CB 18 EE 21 ; ...............!
B4ED  .byte 55 B3 3A D5 D2 FE 01 20 03 21 8F B5 DD 75 0F DD ; U.:.... .!...u..
B4FD  .byte 74 10 3E 50 32 17 D2 CD 15 B5 DD 34 11 DD 7E 11 ; t.>P2......4..~.
B50D  .byte FE A0 D8 DD 36 11 00 C9 3A 17 D2 6F 11 10 00 0E ; ....6...:..o....
B51D  .byte 00 DD 7E 11 BD 38 04 0D 11 F0 FF DD 6E 0A DD 66 ; ..~..8......n..f
B52D  .byte 0B DD 7E 0C 19 89 DD 75 0A DD 74 0B DD 77 0C 7C ; ..~....u..t..w.|
B53D  .byte A7 F2 5B B5 7D 2F 6F 7C 2F 67 23 7C FE 02 38 1E ; ..[.}/o|/g#|..8.
B54D  .byte DD 36 0A 00 DD 36 0B FE DD 36 0C FF 18 10 FE 02 ; .6...6...6......
B55D  .byte 38 0C DD 36 0A 00 DD 36 0B 02 DD 36 0C 00 3A 09 ; 8..6...6...6..:.
B56D  .byte D4 A7 F8 DD 36 0D 1E DD 36 0E 1C 21 02 08 22 15 ; ....6...6..!..".
B57D  .byte D2 CD 28 33 D8 DD 5E 0A DD 56 0B 01 10 00 CD F5 ; ..(3..^..V......
B58D  .byte 7C C9 FE FF FF FF FF FF 6C 6E 6C 6E FF FF FF C5 ; |.......lnln....
B59D  .byte D5 CD AF 7C D1 C1 D8 DD E5 E5 DD 6E 02 DD 66 03 ; ...|.......n..f.
B5AD  .byte 19 EB DD 6E 05 DD 66 06 09 4D 44 DD E1 AF DD 36 ; ...n..f..MD....6
B5BD  .byte 00 0D DD 77 01 DD 73 02 DD 72 03 DD 77 04 DD 71 ; ...w..s..r..w..q
B5CD  .byte 05 DD 70 06 DD 77 11 DD 77 13 DD 77 14 DD 77 15 ; ..p..w..w..w..w.
B5DD  .byte DD 77 16 DD 77 17 2A 13 D2 CB 7C 28 02 3E FF DD ; .w..w.*...|(.>..
B5ED  .byte 75 07 DD 74 08 DD 77 09 AF 2A 15 D2 CB 7C 28 02 ; u..t..w..*...|(.
B5FD  .byte 3E FF DD 75 0A DD 74 0B DD 77 0C DD E1 3E 01 EF ; >..u..t..w...>..
B60D  .byte C9 DD 36 0D 1E DD 36 0E 2F DD CB 18 EE DD CB 18 ; ..6...6./.......
B61D  .byte 56 C2 1C B8 CD DA 7C CD CB B7 DD CB 18 46 20 4F ; V.....|......F O
B62D  .byte 21 0C F0 11 00 20 3E 0C CD 06 04 21 60 03 11 38 ; !.... >....!`..8
B63D  .byte 01 CD C0 7C DD 6E 02 DD 66 03 11 08 00 19 DD 75 ; ...|.n..f......u
B64D  .byte 02 DD 74 03 DD 75 11 DD 74 12 DD 6E 05 DD 66 06 ; ..t..u..t..n..f.
B65D  .byte 11 10 00 19 DD 75 05 DD 74 06 DD 75 13 DD 74 14 ; .....u..t..u..t.
B66D  .byte AF 32 ED D2 3E 0D DF FD CB 08 E6 DD CB 18 C6 DD ; .2..>...........
B67D  .byte 7E 15 A7 C2 B9 B6 CD B0 B9 3A 24 D2 E6 07 C2 78 ; ~........:$....x
B68D  .byte B7 DD 7E 16 FE 1C 30 0B DD 34 17 DD 7E 17 FE 02 ; ..~...0..4..~...
B69D  .byte DA A4 B6 DD 36 17 00 DD 34 16 DD 7E 16 FE 28 DA ; ....6...4..~..(.
B6AD  .byte 78 B7 DD 36 16 00 DD 34 15 C3 78 B7 3D 20 2A DD ; x..6...4..x.= *.
B6BD  .byte 36 0A 40 DD 36 0B FE DD 36 0C FF DD 34 15 DD 6E ; 6.@.6...6...4..n
B6CD  .byte 11 DD 66 12 11 04 00 19 DD 75 02 DD 74 03 DD 36 ; ..f......u..t..6
B6DD  .byte 0F 2E DD 36 10 BB C3 78 B7 3D C2 41 B7 DD 6E 0A ; ...6...x.=.A..n.
B6ED  .byte DD 66 0B DD 7E 0C 11 0E 00 19 CE 00 4F FA 05 B7 ; .f..~.......O...
B6FD  .byte 7C FE 02 38 03 21 00 02 DD 75 0A DD 74 0B DD 71 ; |..8.!...u..t..q
B70D  .byte 0C DD 36 0F 2E DD 36 10 BB DD 6E 05 DD 66 06 2B ; ..6...6...n..f.+
B71D  .byte DD 5E 13 DD 56 14 A7 ED 52 38 50 DD 73 05 DD 72 ; .^..V...R8P.s..r
B72D  .byte 06 AF DD 77 16 DD 77 0A DD 77 0B DD 77 0C DD 34 ; ...w..w..w..w..4
B73D  .byte 15 C3 78 B7 3D C2 78 B7 DD 6E 11 DD 66 12 DD 75 ; ..x.=.x..n..f..u
B74D  .byte 02 DD 74 03 DD 7E 16 A7 CC E6 B9 DD 36 17 02 DD ; ..t..~......6...
B75D  .byte CB 18 CE CD B0 B9 DD 34 16 DD 7E 16 FE 12 38 0B ; .......4..~...8.
B76D  .byte DD CB 18 8E AF DD 77 15 DD 77 16 21 42 BA DD CB ; ......w..w.!B...
B77D  .byte 18 4E 28 03 21 4C BA 11 0F D2 ED A0 ED A0 ED A0 ; .N(.!L..........
B78D  .byte ED A0 ED A0 ED A0 ED A0 ED A0 7E 23 E5 CD 5D 2F ; ..........~#..]/
B79D  .byte 2A 13 D2 11 08 00 19 22 13 D2 E1 7E CD 5D 2F 3A ; *......"...~.]/:
B7AD  .byte ED D2 FE 0C D8 AF DD 77 11 DD 77 16 DD 77 17 DD ; .......w..w..w..
B7BD  .byte CB 18 D6 FD CB 08 A6 3E 04 DF 3E 21 EF C9 2A FF ; .......>..>!..*.
B7CD  .byte D3 11 F8 03 AF ED 52 38 08 6F 67 32 04 D4 22 05 ; ......R8.og2..".
B7DD  .byte D4 3A B1 D2 A7 C0 FD CB 05 46 C0 3A 15 D4 0F 38 ; .:.......F.:...8
B7ED  .byte 03 E6 02 C8 2A FF D3 11 F8 03 A7 ED 52 D8 21 00 ; ....*.......R.!.
B7FD  .byte FD 3E FF 22 04 D4 32 06 D4 21 B1 D2 36 18 23 36 ; .>."..2..!..6.#6
B80D  .byte 00 23 36 FF 23 36 0F 3E 01 EF 21 ED D2 34 C9 DD ; .#6.#6.>..!..4..
B81D  .byte CB 18 5E C2 6C B9 DD CB 18 AE DD 7E 11 FE 0F 30 ; ..^.l......~...0
B82D  .byte 37 87 87 5F 87 83 5F 16 00 21 56 BA 19 5E 23 56 ; 7.._.._..!V..^#V
B83D  .byte 23 ED 53 AB D2 5E 23 56 23 ED 53 AD D2 22 AF D2 ; #.S..^#V#.S.."..
B84D  .byte DD 34 11 DD 7E 11 FE 0F 20 0E FD CB 00 EE FD CB ; .4..~... .......
B85D  .byte 02 8E 21 50 05 22 6F D2 DD 5E 02 DD 56 03 21 E0 ; ..!P."o..^..V.!.
B86D  .byte 05 AF ED 52 30 05 4F 47 C3 94 B8 EB ED 5B FF D3 ; ...R0.OG.....[..
B87D  .byte AF ED 52 11 40 00 AF ED 4B 04 D4 CB 78 20 04 ED ; ..R.@...K...x ..
B88D  .byte 52 38 03 01 80 FF 04 DD 71 07 DD 70 08 DD 77 09 ; R8......q..p..w.
B89D  .byte DD 7E 17 FE 06 20 18 DD 7E 16 3D 20 12 DD CB 18 ; .~... ..~.= ....
B8AD  .byte 7E 28 0C DD 36 0A 00 DD 36 0B FF DD 36 0C FF 11 ; ~(..6...6...6...
B8BD  .byte 17 00 01 36 00 CD D5 30 5E 16 00 21 10 39 19 7E ; ...6...0^..!.9.~
B8CD  .byte E6 3F A7 28 12 DD CB 18 7E 28 0C DD 36 0A 80 DD ; .?.(....~(..6...
B8DD  .byte 36 0B FD DD 36 0C FF 11 00 00 01 08 00 CD D5 30 ; 6...6..........0
B8ED  .byte 7E FE 49 20 4C DD CB 18 7E 28 46 AF DD 77 16 DD ; ~.I L...~(F..w..
B8FD  .byte 77 17 DD 77 07 DD 77 08 DD 77 09 DD 36 11 E0 DD ; w..w..w..w..6...
B90D  .byte 36 12 05 DD 36 13 60 DD 36 14 01 21 50 05 11 20 ; 6...6.`.6..!P.. 
B91D  .byte 01 CD C0 7C 3E 03 32 FE FF 32 2F D2 21 08 00 CD ; ...|>.2..2/.!...
B92D  .byte 0C 40 3E 01 32 FE FF 32 2F D2 DD CB 18 DE C3 6C ; .@>.2..2/......l
B93D  .byte B9 DD 6E 0A DD 66 0B DD 7E 0C 11 0E 00 19 CE 00 ; ..n..f..~.......
B94D  .byte 4F FA 59 B9 7C FE 02 38 03 21 00 02 DD 75 0A DD ; O.Y.|..8.!...u..
B95D  .byte 74 0B DD 71 0C 01 39 BA 11 0A BB CD 75 7C C9 FD ; t..q..9.....u|..
B96D  .byte 36 03 FF CD B0 B9 DD 7E 16 FE 30 30 21 4F 3A 24 ; 6......~..00!O:$
B97D  .byte D2 E6 07 20 0C DD 7E 17 3C E6 01 DD 77 17 DD 34 ; ... ..~.<...w..4
B98D  .byte 16 79 FE 2C D8 DD 36 0F 88 DD 36 10 BB C9 AF DD ; .y.,..6...6.....
B99D  .byte 77 0F DD 77 10 DD 34 16 DD 7E 16 FE 70 D8 DD 36 ; w..w..4..~..p..6
B9AD  .byte 00 FF C9 21 2D BA DD 7E 17 87 87 5F 16 00 42 19 ; ...!-..~..._..B.
B9BD  .byte 4E 23 5E 23 7E 23 66 6F DD 75 0F DD 74 10 DD 6E ; N#^#~#fo.u..t..n
B9CD  .byte 11 DD 66 12 09 DD 75 02 DD 74 03 DD 6E 13 DD 66 ; ..f...u..t..n..f
B9DD  .byte 14 19 DD 75 05 DD 74 06 C9 FD CB 08 6E C0 CD AF ; ...u..t.....n...
B9ED  .byte 7C D8 DD E5 E5 DD E1 AF DD 36 00 47 DD 77 01 21 ; |........6.G.w.!
B9FD  .byte C8 03 DD 75 02 DD 74 03 DD 77 04 21 5F 01 DD 75 ; ...u..t..w.!_..u
BA0D  .byte 05 DD 74 06 DD 77 11 DD 77 18 DD 77 07 DD 77 08 ; ..t..w..w..w..w.
BA1D  .byte DD 77 09 DD 77 0A DD 77 0B DD 77 0C DD E1 C9 C9 ; .w..w..w..w.....
BA2D  .byte 00 00 0A BB 00 02 1C BB 00 07 1C BB 03 08 04 07 ; ................
BA3D  .byte 05 08 04 07 FF 10 04 A0 01 00 00 00 00 20 22 10 ; ............. ".
BA4D  .byte 04 A0 01 00 00 00 00 24 26 00 04 60 01 37 10 38 ; .......$&..`.7.8
BA5D  .byte 10 4A 10 4B 10 10 04 60 01 28 10 19 10 4C 10 4D ; .J.K...`.(...L.M
BA6D  .byte 10 20 04 60 01 00 10 2D 10 4E 10 4F 10 00 04 70 ; . .`...-.N.O...p
BA7D  .byte 01 00 00 00 00 00 00 00 00 10 04 70 01 00 00 00 ; ...........p....
BA8D  .byte 00 00 00 00 00 20 04 70 01 00 00 00 00 00 00 00 ; ..... .p........
BA9D  .byte 00 00 04 80 01 00 00 00 00 00 00 00 00 10 04 80 ; ................
BAAD  .byte 01 00 00 00 00 00 00 00 00 20 04 80 01 00 00 00 ; ......... ......
BABD  .byte 00 00 00 00 00 00 04 90 01 00 00 00 00 00 00 00 ; ................
BACD  .byte 00 10 04 90 01 00 00 00 00 00 00 00 00 20 04 90 ; ............. ..
BADD  .byte 01 00 00 00 00 00 00 00 00 00 04 A0 01 5A 10 5B ; .............Z.[
BAED  .byte 10 37 10 3B 10 10 04 A0 01 5C 10 5D 10 3C 10 00 ; .7.;.....\.].<..
BAFD  .byte 10 20 04 A0 01 5E 10 5F 10 00 10 2D 10 FE 0A 0C ; . ...^._...-....
BB0D  .byte 0E FF FF 28 2A 2C 2E FF FF FE 4A 4C 4E FF FF FE ; ...(*,....JLN...
BB1D  .byte 0A 0C 0E FF FF 28 2A 2C 2E FF FF FE 02 04 06 FF ; .....(*,........
BB2D  .byte FF 10 12 14 16 FF FF 30 32 34 FE FF FF 50 52 54 ; .......024...PRT
BB3D  .byte FE FF FF 18 1A 1C 1E FF FF FE 3A 3C 3E FF FF FE ; ..........:<>...
BB4D  .byte 64 66 68 FF FF 18 1A 1C 1E FF FF FE 3A 3C 3E FF ; dfh.........:<>.
BB5D  .byte FF FE 6A 6C 6E FF FF 18 1A 1C 1E FF FF FE 3A 3C ; ..jln.........:<
BB6D  .byte 3E FF FF 70 72 5A 5C 5E FF 00 0A 0C 0E FF FF 28 ; >..prZ\^.......(
BB7D  .byte 2A 2C 2E FF FF 00 4A 4C 4E FF FF FE FF FF FF FF ; *,....JLN.......
BB8D  .byte FF FE 44 46 FF FF FF FF DD CB 18 EE DD CB 18 46 ; ..DF...........F
BB9D  .byte 20 14 DD 6E 02 DD 66 03 11 0C 00 19 DD 75 02 DD ;  ..n..f......u..
BBAD  .byte 74 03 DD CB 18 C6 FD CB 08 66 20 05 DD 36 00 FF ; t........f ..6..
BBBD  .byte C9 21 5C BC CD DF 9C 21 3C BC 19 7E 23 66 6F B4 ; .!\....!<..~#fo.
BBCD  .byte 28 34 DD 7E 11 87 87 87 E6 1F 5F 16 00 19 06 04 ; (4.~......_.....
BBDD  .byte C5 4E 23 5E 23 16 00 E5 ED 53 15 D2 79 CD 5D 2F ; .N#^#....S..y.]/
BBED  .byte E1 C1 10 EC DD 7E 0E A7 28 0C 21 02 0A 22 15 D2 ; .....~..(.!.."..
BBFD  .byte CD 28 33 D4 D9 2F DD 34 11 AF DD 77 0F DD 77 10 ; .(3../.4...w..w.
BC0D  .byte DD 7E 11 A7 28 07 FE 70 C0 3E 17 EF C9 DD CB 18 ; .~..(..p.>......
BC1D  .byte 86 DD 7E 12 3C FE 03 38 01 AF DD 77 12 87 5F 16 ; ..~.<..8...w.._.
BC2D  .byte 00 21 6C BC 19 7E DD 77 02 23 7E DD 77 03 C9 00 ; .!l..~.w.#~.w...
BC3D  .byte 00 00 00 00 00 00 00 00 00 00 00 00 00 92 BC B2 ; ................
BC4D  .byte BC D2 BC 72 BC 72 BC 72 BC D2 BC B2 BC 92 BC 00 ; ...r.r.r........
BC5D  .byte 00 00 00 00 00 00 1B 1F 22 25 25 25 22 1F 1B A0 ; ........"%%%"...
BC6D  .byte 03 E0 03 C0 03 48 07 08 0E 48 15 08 1C 08 05 48 ; .....H...H.....H
BC7D  .byte 0C 08 13 48 1A 48 03 08 0A 48 11 08 18 08 01 48 ; ...H.H...H.....H
BC8D  .byte 08 08 0F 48 16 48 10 08 14 48 18 08 1C 08 0E 48 ; ...H.H...H.....H
BC9D  .byte 12 08 16 48 1A 48 0C 08 10 48 14 08 18 08 0A 48 ; ...H.H...H.....H
BCAD  .byte 0E 08 12 48 16 48 0D 08 12 48 17 08 1C 08 0B 48 ; ...H.H...H.....H
BCBD  .byte 10 08 15 48 1A 48 09 08 0E 48 13 08 18 08 07 48 ; ...H.H...H.....H
BCCD  .byte 0C 08 11 48 16 48 0C 08 10 48 16 08 1C 08 0A 48 ; ...H.H...H.....H
BCDD  .byte 0E 08 14 48 1A 48 08 08 0C 48 12 08 18 08 06 48 ; ...H.H...H.....H
BCED  .byte 0A 08 10 48 16 DD CB 18 EE DD 36 0D 08 DD 36 0E ; ...H......6...6.
BCFD  .byte 08 FD CB 08 EE 21 04 04 22 15 D2 CD 28 33 38 0A ; .....!.."...(38.
BD0D  .byte FD CB 05 46 CC D9 2F C3 DC BD DD 7E 11 FE C8 DA ; ...F../....~....
BD1D  .byte CB BD DD 5E 02 DD 56 03 2A 54 D2 01 28 00 09 A7 ; ...^..V.*T..(...
BD2D  .byte ED 52 D2 DC BD 2A 54 D2 01 D0 00 19 A7 ED 52 DA ; .R...*T.......R.
BD3D  .byte DC BD 2A FF D3 A7 ED 52 DD 6E 07 DD 66 08 DD 7E ; ..*....R.n..f..~
BD4D  .byte 09 30 0E 0E FF 11 F8 FF CB 7F 20 11 11 F0 FF 18 ; .0........ .....
BD5D  .byte 0C 0E 00 11 08 00 CB 7F 28 03 11 10 00 19 89 DD ; ........(.......
BD6D  .byte 75 07 DD 74 08 DD 77 09 DD 5E 05 DD 56 06 2A 57 ; u..t..w..^..V.*W
BD7D  .byte D2 01 10 00 09 A7 ED 52 30 55 2A 57 D2 01 A8 00 ; .......R0U*W....
BD8D  .byte 09 A7 ED 52 38 49 2A 02 D4 A7 ED 52 DD 6E 0A DD ; ...R8I*....R.n..
BD9D  .byte 66 0B DD 7E 0C 30 0E 0E FF 11 FB FF CB 7F 20 11 ; f..~.0........ .
BDAD  .byte 11 FE FF 18 0C 11 05 00 0E 00 CB 7F 28 03 11 02 ; ............(...
BDBD  .byte 00 19 89 DD 75 0A DD 74 0B DD 77 0C 18 03 DD 34 ; ....u..t..w....4
BDCD  .byte 11 01 E5 BD 11 EA BD CD 75 7C FD CB 08 66 C0 DD ; ........u|...f..
BDDD  .byte 36 00 FF FD CB 08 AE C9 00 02 01 02 FF 44 46 FF ; 6............DF.
BDED  .byte FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF 60 ; ...............`
BDFD  .byte 62 FF FF FF FF FF DD CB 18 EE FD 36 03 FF DD CB ; b..........6....
BE0D  .byte 18 4E 20 1C 3E 11 32 2D D2 3E FF 32 FD D3 21 00 ; .N .>.2-.>.2..!.
BE1D  .byte 00 22 02 D4 DD 36 12 FF FD CB 07 F6 DD CB 18 CE ; ."...6..........
BE2D  .byte 3A 24 D2 0F 38 30 DD 7E 12 A7 28 2A DD 35 12 20 ; :$..80.~..(*.5. 
BE3D  .byte 25 DD 6E 02 DD 66 03 11 3C 00 19 22 FF D3 DD 6E ; %.n..f..<.."...n
BE4D  .byte 05 DD 66 06 11 C0 FF 19 22 02 D4 AF 32 FD D3 FD ; ..f....."...2...
BE5D  .byte CB 08 F6 3E 06 EF DD 36 0D 20 DD 36 0E 1C AF DD ; ...>...6. .6....
BE6D  .byte 77 07 DD 36 08 01 DD 77 09 DD 77 0A DD 77 0B DD ; w..6...w..w..w..
BE7D  .byte 77 0C FD CB 07 76 28 18 ED 5B 54 D2 21 40 00 19 ; w....v(..[T.!@..
BE8D  .byte DD 4E 02 DD 46 03 A7 ED 42 30 05 13 ED 53 54 D2 ; .N..F...B0...ST.
BE9D  .byte DD 36 0F 28 DD 36 10 BF DD CB 18 46 20 33 21 08 ; .6.(.6.....F 3!.
BEAD  .byte 10 22 15 D2 CD 28 33 38 28 11 01 00 2A 07 D4 7D ; ."...(38(...*..}
BEBD  .byte 2F 6F 7C 2F 67 3A 09 D4 2F 19 CE 00 22 07 D4 32 ; /o|/g:../..."..2
BECD  .byte 09 D4 FD CB 07 B6 DD CB 18 C6 DD 36 11 01 3E 01 ; ...........6..>.
BEDD  .byte EF CD 36 7A DD CB 18 46 C8 AF DD 36 0A 40 DD 77 ; ..6z...F...6.@.w
BEED  .byte 0B DD 77 0C DD 36 0F 3A DD 36 10 BF DD 35 11 C0 ; ..w..6.:.6...5..
BEFD  .byte CD 76 7A DD 36 11 18 DD 34 13 DD 7E 13 FE 0A D8 ; .vz.6...4..~....
BF0D  .byte 3A 79 D2 FE 06 38 05 FD CB 08 FE C9 3A 83 D2 A7 ; :y...8......:...
BF1D  .byte C0 3E 20 32 83 D2 FD CB 0D D6 C9 2A 2C 2E 30 32 ; .> 2.......*,.02
BF2D  .byte FF 4A 4C 4E 50 52 FF 6A 6C 6E 70 72 FF 2A 34 36 ; .JLNPR.jlnpr.*46
BF3D  .byte 38 32 FF 4A 4C 4E 50 52 FF 6A 6C 6E 70 72 FF 5C ; 82.JLNPR.jlnpr.\
BF4D  .byte 5E FF FF FF FF FF DD CB 18 EE 21 80 54 CD A8 0B ; ^.........!.T...
BF5D  .byte DD CB 18 46 20 22 AF DD 77 0F DD 77 10 DD 77 07 ; ...F "..w..w..w.
BF6D  .byte DD 77 08 DD 77 09 DD 34 11 DD 7E 11 FE 50 D8 DD ; .w..w..4..~..P..
BF7D  .byte CB 18 C6 DD 36 11 64 C9 DD 7E 11 A7 28 05 DD 35 ; ....6.d..~..(..5
BF8D  .byte 11 18 0C DD 36 0A 80 DD 36 0B FF DD 36 0C FF 21 ; ....6...6...6..!
BF9D  .byte F8 BF 3A 24 D2 0F 30 37 FD 7E 0A 2A 36 D2 F5 E5 ; ..:$..07.~.*6...
BFAD  .byte 21 00 D0 22 36 D2 DD 6E 05 DD 66 06 ED 5B 57 D2 ; !.."6..n..f..[W.
BFBD  .byte A7 ED 52 EB DD 6E 02 DD 66 03 ED 4B 54 D2 A7 ED ; ..R..n..f..KT...
BFCD  .byte 42 01 F8 BF CD 07 2F E1 F1 22 36 D2 FD 77 0A DD ; B...../.."6..w..
BFDD  .byte 6E 05 DD 66 06 11 20 00 19 ED 5B 57 D2 A7 ED 52 ; n..f.. ...[W...R
BFED  .byte D0 3E 01 32 83 D2 FD CB 0D D6 C9 5C 5E FF FF FF ; .>.2.......\^...
BFFD  .byte FF FF 48 C3 3A 42 C3 18 40 C3 2D 41 C3 E5 41 C3 ; ..H.:B..@.-A..A.
C00D  .byte 24 42 C3 71 41 C3 EB 46 C3 FF 46 F5 C5 D5 E5 DD ; $B.qA..F..F.....
C01D  .byte E5 4D 44 DD 21 1C DC 3E 05 5E 23 56 23 EB 09 DD ; .MD.!..>.^#V#...
C02D  .byte 75 00 DD 23 DD 74 00 DD 23 EB 3D C2 26 40 21 70 ; u..#.t..#.=.&@!p
C03D  .byte 40 5E 23 56 7A 3C 28 08 23 ED A0 ED A0 C3 3E 40 ; @^#Vz<(.#.....>@
C04D  .byte 21 D6 40 5E 23 56 7A 3C 28 06 23 ED A0 C3 50 40 ; !.@^#Vz<(.#...P@
C05D  .byte DD E1 E1 D1 C1 F1 22 4F DC 22 7C DC 22 A9 DC 22 ; ......"O."|.".."
C06D  .byte D6 DC C9 48 DC 00 00 75 DC 00 00 A2 DC 00 00 CF ; ...H...u........
C07D  .byte DC 00 00 46 DC 07 DD 73 DC 08 DD A0 DC 09 DD CD ; ...F...s........
C08D  .byte DC 0A DD 28 DC 01 00 55 DC 01 00 82 DC 01 00 AF ; ...(...U........
C09D  .byte DC 01 00 3D DC 00 00 42 DC 00 00 6A DC 00 00 6F ; ...=...B...j...o
C0AD  .byte DC 00 00 97 DC 00 00 9C DC 00 00 C4 DC 00 00 C9 ; ................
C0BD  .byte DC 00 00 2E DC 00 00 5B DC 00 00 88 DC 00 00 B5 ; .......[........
C0CD  .byte DC 00 00 0A DC 01 00 FF FF 26 DC 80 27 DC 90 53 ; .........&..'..S
C0DD  .byte DC A0 54 DC B0 80 DC C0 81 DC D0 AD DC E0 AE DC ; ..T.............
C0ED  .byte F0 4E DC 02 7B DC 02 A8 DC 02 D5 DC 02 02 DD 00 ; .N..{...........
C0FD  .byte 3A DC 00 67 DC 00 94 DC 00 C1 DC 00 3B DC 00 68 ; :..g........;..h
C10D  .byte DC 00 95 DC 00 C2 DC 00 51 DC 00 7E DC 01 AB DC ; ........Q..~....
C11D  .byte 02 D8 DC 03 06 DC 00 04 DC 00 FF FF 9F BF DF FF ; ................
C12D  .byte F5 E5 C5 3A 4E DC E6 FD 32 4E DC 3A 7B DC E6 FD ; ...:N...2N.:{...
C13D  .byte 32 7B DC 3A A8 DC E6 FD 32 A8 DC 3A D5 DC E6 FD ; 2{.:....2..:....
C14D  .byte 32 D5 DC 3A 02 DD E6 FD 32 02 DD AF 32 06 DC 06 ; 2..:....2...2...
C15D  .byte 04 0E 7F 21 29 41 ED B3 3A 04 DC E6 F7 32 04 DC ; ...!)A..:....2..
C16D  .byte C1 E1 F1 C9 F5 D5 E5 5F 3A 06 DC A7 28 03 BB 38 ; ......._:...(..8
C17D  .byte 5B 7B 32 06 DC 22 03 DD 3A DB DC F6 0F D3 7F 7E ; [{2.."..:......~
C18D  .byte 32 05 DC 23 5E 23 56 23 ED 53 00 DD 5E 23 56 23 ; 2..#^#V#.S..^#V#
C19D  .byte ED 53 0E DC 23 22 24 DC 21 DD 41 87 5F 16 00 19 ; .S..#"$.!.A._...
C1AD  .byte 7E 32 DA DC 23 7E 32 DB DC 21 00 00 22 FC DC 22 ; ~2..#~2..!..".."
C1BD  .byte F1 DC 22 F6 DC 22 E2 DC 3E 04 32 05 DD 23 22 DC ; ..".."..>.2..#".
C1CD  .byte DC 21 0B DD 22 FA DC 3E 02 32 02 DD E1 D1 F1 C9 ; .!.."..>.2......
C1DD  .byte 80 90 A0 B0 C0 D0 E0 F0 F5 3A 4E DC F6 02 32 4E ; .........:N...2N
C1ED  .byte DC 3A 7B DC F6 02 32 7B DC 3A A8 DC F6 02 32 A8 ; .:{...2{.:....2.
C1FD  .byte DC 3A D5 DC F6 02 32 D5 DC 3A 52 DC 32 2B DC 3A ; .:....2..:R.2+.:
C20D  .byte 7F DC 32 58 DC 3A AC DC 32 85 DC 3A D9 DC 32 B2 ; ..2X.:..2..:..2.
C21D  .byte DC AF 32 04 DC F1 C9 F5 E5 22 12 DC 3A 04 DC F6 ; ..2......"..:...
C22D  .byte 08 32 04 DC 21 00 10 22 10 DC E1 F1 C9 DD 21 26 ; .2..!.."......!&
C23D  .byte DC ED 5B 1C DC ED 4B 0A DC CD F4 42 DD 22 14 DC ; ..[...K....B."..
C24D  .byte ED 53 1C DC DD 21 53 DC ED 5B 1E DC ED 4B 0A DC ; .S...!S..[...K..
C25D  .byte CD F4 42 DD 22 16 DC ED 53 1E DC DD 21 80 DC ED ; ..B."...S...!...
C26D  .byte 5B 20 DC ED 4B 0A DC CD F4 42 DD 22 18 DC ED 53 ; [ ..K....B."...S
C27D  .byte 20 DC DD 21 AD DC ED 5B 22 DC ED 4B 0A DC CD F4 ;  ..!...["..K....
C28D  .byte 42 DD 22 1A DC ED 53 22 DC DD 21 DA DC ED 5B 24 ; B."...S"..!...[$
C29D  .byte DC ED 4B 0E DC CD F4 42 ED 53 24 DC DD CB 28 4E ; ..K....B.S$...(N
C2AD  .byte 28 10 21 14 DC 3A 05 DC 87 4F 06 00 09 36 DA 23 ; (.!..:...O...6.#
C2BD  .byte 36 DC DD 2A 14 DC CD DE 43 DD 2A 16 DC CD DE 43 ; 6..*....C.*....C
C2CD  .byte DD 2A 18 DC CD DE 43 DD 2A 1A DC CD DE 43 3A 04 ; .*....C.*....C:.
C2DD  .byte DC E6 08 C8 2A 10 DC ED 4B 12 DC A7 ED 42 30 03 ; ....*...K....B0.
C2ED  .byte CD 2D 41 22 10 DC C9 DD CB 28 4E C8 DD 6E 02 DD ; .-A".....(N..n..
C2FD  .byte 66 03 A7 ED 42 DD 75 02 DD 74 03 28 03 D2 C9 43 ; f...B.u..t.(...C
C30D  .byte 1A A7 FA F3 44 FE 70 38 35 FE 7F 20 07 DD 36 1E ; ....D.p85.. ..6.
C31D  .byte 00 C3 9F 43 D5 DD E5 E1 01 0E 00 09 EB E6 0F 6F ; ...C...........o
C32D  .byte 26 00 29 29 29 01 CE 43 09 7E DD 77 25 23 ED A0 ; &.)))..C.~.w%#..
C33D  .byte ED A0 ED A0 ED A0 ED A0 ED A0 D1 C3 6E 43 E6 0F ; ............nC..
C34D  .byte 21 D5 44 87 4F 06 00 09 7E DD 77 06 23 7E DD 77 ; !.D.O...~.w.#~.w
C35D  .byte 07 1A 0F 0F 0F 0F E6 0F DD 77 1F DD CB 28 46 20 ; .........w...(F 
C36D  .byte 31 DD 7E 14 DD 77 19 DD 7E 15 DD 77 1A DD 7E 16 ; 1.~..w..~..w..~.
C37D  .byte CB 3F DD 77 1B DD 7E 17 DD 77 1C DD 7E 18 DD 77 ; .?.w..~..w..~..w
C38D  .byte 1D AF DD 77 0A DD 77 0B DD 77 0D DD 77 0C DD 36 ; ...w..w..w..w..6
C39D  .byte 1E 0F 13 1A 13 A7 20 03 DD 7E 24 D5 4F DD 6E 26 ; ...... ..~$.O.n&
C3AD  .byte DD 66 27 7D B4 20 03 2A 08 DC CD D8 46 D1 7D DD ; .f'}. .*....F.}.
C3BD  .byte 86 02 DD 77 02 7C DD 8E 03 DD 77 03 DD CB 28 86 ; ...w.|....w...(.
C3CD  .byte C9 05 FF BE 0A 04 05 02 00 05 E6 24 5A 14 28 08 ; ...........$Z.(.
C3DD  .byte 00 DD CB 28 4E C8 DD 7E 0D A7 CA 45 45 3D CA 5C ; ...(N..~...EE=.\
C3ED  .byte 45 3D CA 79 45 3D CA 97 45 DD 7E 00 FE E0 20 15 ; E=.yE=..E.~... .
C3FD  .byte DD 4E 25 3A 07 DC B9 CA 8F 44 79 32 07 DC F6 E0 ; .N%:.....Dy2....
C40D  .byte D3 7F C3 8F 44 DD 5E 0A DD 56 0B DD 7E 19 A7 28 ; ....D.^..V..~..(
C41D  .byte 06 DD 35 19 C3 5A 44 DD 35 1A C2 5A 44 DD 7E 15 ; ..5..ZD.5..ZD.~.
C42D  .byte DD 77 1A DD 6E 1C DD 66 1D DD 35 1B C2 52 44 DD ; .w..n..f..5..RD.
C43D  .byte 7E 16 DD 77 1B 7D 2F 6F 7C 2F 67 23 DD 75 1C DD ; ~..w.}/o|/g#.u..
C44D  .byte 74 1D C3 5A 44 19 DD 75 0A DD 74 0B EB DD 6E 06 ; t..ZD..u..t...n.
C45D  .byte DD 66 07 DD 4E 08 DD 46 09 09 19 DD 7E 1F A7 28 ; .f..N..F....~..(
C46D  .byte 07 47 CB 3C CB 1D 10 FA 7D E6 0F DD B6 00 D3 7F ; .G.<....}.......
C47D  .byte 7C 07 07 07 07 E6 F0 4F 7D 0F 0F 0F 0F E6 0F B1 ; |......O}.......
C48D  .byte D3 7F DD 7E 05 A7 28 12 4F DD 7E 0C A7 28 0B 6F ; ...~..(.O.~..(.o
C49D  .byte 26 00 CD D8 46 CB 15 3E 00 8C DD A6 1E EE 0F DD ; &...F..>........
C4AD  .byte B6 01 D3 7F 3A 04 DC E6 08 C8 DD 7E 2B FE 04 C8 ; ....:......~+...
C4BD  .byte DD 6E 04 DD 66 05 ED 4B 12 DC ED 42 30 03 21 00 ; .n..f..K...B0.!.
C4CD  .byte 00 DD 75 04 DD 74 05 C9 56 03 26 03 F9 02 CE 02 ; ..u..t..V.&.....
C4DD  .byte A5 02 80 02 5C 02 3A 02 1A 02 FB 01 DF 01 C4 01 ; ....\.:.........
C4ED  .byte F7 03 BE 03 88 03 FE FF CA 0B 45 FE FE CA 19 45 ; ..........E....E
C4FD  .byte 13 21 29 45 87 4F 06 00 09 7E 23 66 6F E9 DD 6E ; .!)E.O...~#fo..n
C50D  .byte 22 DD 66 23 7D B4 28 08 EB C3 0D 43 AF 32 06 DC ; ".f#}.(....C.2..
C51D  .byte DD CB 28 8E 3E 0F DD B6 01 D3 7F C9 AE 45 D1 45 ; ..(.>........E.E
C52D  .byte F2 45 0A 46 20 46 2D 46 32 46 47 46 7D 46 86 46 ; .E.F F-F2FGF}F.F
C53D  .byte 8E 46 96 46 B4 46 D1 46 DD 7E 0E DD 86 0C D2 50 ; .F.F.F.F.~.....P
C54D  .byte 45 3E FF DD 77 0C D2 F6 43 DD 34 0D C3 F6 43 DD ; E>..w...C.4...C.
C55D  .byte 4E 10 DD 7E 0C DD 96 0F 38 06 DD BE 10 38 01 4F ; N..~....8....8.O
C56D  .byte DD 71 0C D2 F6 43 DD 34 0D C3 F6 43 DD 4E 12 DD ; .q...C.4...C.N..
C57D  .byte 7E 0C DD 96 11 38 07 DD BE 12 DA 8B 45 4F DD 71 ; ~....8......EO.q
C58D  .byte 0C D2 F6 43 DD 34 0D C3 F6 43 DD 7E 0C DD 96 13 ; ...C.4...C.~....
C59D  .byte D2 A2 45 3E 00 DD 77 0C D2 F6 43 DD 34 0D C3 F6 ; ..E>..w...C.4...
C5AD  .byte 43 1A DD 77 26 32 08 DC 13 1A DD 77 27 32 09 DC ; C..w&2.....w'2..
C5BD  .byte 13 1A 32 0A DC 32 0C DC 13 1A 32 0B DC 32 0D DC ; ..2..2....2..2..
C5CD  .byte 13 C3 0D 43 1A DD 77 2C 13 DD 7E 2B FE 04 28 08 ; ...C..w,..~+..(.
C5DD  .byte 3A 04 DC E6 08 C2 0D 43 DD 7E 2C DD 77 05 DD 36 ; :......C.~,.w..6
C5ED  .byte 04 00 C3 0D 43 DD E5 E1 01 0E 00 09 EB ED A0 ED ; ....C...........
C5FD  .byte A0 ED A0 ED A0 ED A0 ED A0 EB C3 0D 43 DD E5 E1 ; ............C...
C60D  .byte 01 14 00 09 EB ED A0 ED A0 ED A0 ED A0 ED A0 EB ; ................
C61D  .byte C3 0D 43 1A DD 77 08 13 1A DD 77 09 13 C3 0D 43 ; ..C..w....w....C
C62D  .byte 1A 13 C3 0D 43 DD 6E 20 DD 66 21 36 00 01 05 00 ; ....C.n .f!6....
C63D  .byte 09 DD 75 20 DD 74 21 C3 0D 43 DD 6E 20 DD 66 21 ; ..u .t!..C.n .f!
C64D  .byte 01 FB FF 09 7E A7 20 08 1A 3D 28 18 77 C3 60 46 ; ....~. ..=(.w.`F
C65D  .byte 35 28 11 EB 23 7E 23 66 6F DD 4E 29 DD 46 2A 09 ; 5(..#~#fo.N).F*.
C66D  .byte EB C3 0D 43 DD 75 20 DD 74 21 13 13 13 C3 0D 43 ; ...C.u .t!.....C
C67D  .byte DD 73 22 DD 72 23 C3 0D 43 1A DD 77 25 13 C3 0D ; .s".r#..C..w%...
C68D  .byte 43 1A DD 77 24 13 C3 0D 43 DD 7E 2C 3C FE 10 38 ; C..w$...C.~,<..8
C69D  .byte 02 3E 0F DD 77 2C 3A 04 DC E6 08 C2 0D 43 DD 7E ; .>..w,:......C.~
C6AD  .byte 2C DD 77 05 C3 0D 43 DD 7E 2C 3D FE 10 38 01 AF ; ,.w...C.~,=..8..
C6BD  .byte DD 77 2C 3A 04 DC E6 08 C2 0D 43 DD 7E 2C DD 77 ; .w,:......C.~,.w
C6CD  .byte 05 C3 0D 43 DD CB 28 C6 C3 0D 43 AF 06 07 EB 6F ; ...C..(...C....o
C6DD  .byte 67 CB 11 D2 E4 46 19 29 10 F7 B1 C8 19 C9 E5 21 ; g....F.).......!
C6ED  .byte 16 47 87 85 6F 3E 00 8C 67 7E 23 66 6F CD 18 40 ; .G..o>..g~#fo..@
C6FD  .byte E1 C9 E5 D5 21 40 47 87 87 5F 16 00 19 5E 23 56 ; ....!@G.._...^#V
C70D  .byte 23 7E EB CD 71 41 D1 E1 C9 D0 47 4A 57 4A 52 0C ; #~..qA....GJWJR.
C71D  .byte 76 4F 5B A7 61 C3 64 3C 66 04 67 B4 68 91 69 C0 ; vO[.a.d<f.g.h.i.
C72D  .byte 6A C0 6A C0 6A 54 6D D0 47 2C 71 D0 47 D0 47 8C ; j.j.jTm.G,q.G.G.
C73D  .byte 79 32 7A 33 7B 02 00 4F 7B 02 00 80 7B 02 00 A4 ; y2z3{..O{...{...
C74D  .byte 7B 02 00 CB 7B 02 00 F2 7B 02 00 24 7C 02 00 4E ; {...{...{..$|..N
C75D  .byte 7C 02 00 6A 7C 01 00 9A 7C 01 00 C3 7C 02 00 E4 ; |..j|...|...|...
C76D  .byte 7C 01 00 09 7D 01 00 30 7D 01 00 6E 7D 02 00 6E ; |...}..0}..n}..n
C77D  .byte 7D 01 00 6E 7D 02 00 6E 7D 02 00 94 7D 02 00 BD ; }..n}..n}...}...
C78D  .byte 7D 01 00 BD 7D 02 00 BD 7D 02 00 BD 7D 02 00 F2 ; }...}...}...}...
C79D  .byte 7D 01 00 18 7E 01 00 3B 7E 02 00 54 7E 01 00 68 ; }...~..;~..T~..h
C7AD  .byte 7E 01 00 80 7E 01 00 B0 7E 01 00 D8 7E 02 00 D8 ; ~...~...~...~...
C7BD  .byte 7E 02 00 F4 7E 02 00 14 7F 01 00 5A 7F 01 00 8F ; ~...~......Z....
C7CD  .byte 7F 02 00 0A 00 74 01 CD 02 FC 04 00 00 80 01 00 ; .....t..........
C7DD  .byte 01 00 82 FF 14 96 00 32 0A 85 FF 83 0C 01 04 05 ; .......2........
C7ED  .byte 00 81 0C 8A 06 29 00 25 00 29 00 25 00 2B 00 27 ; .....).%.).%.+.'
C7FD  .byte 00 2B 00 27 00 30 00 29 00 30 00 29 00 32 00 2B ; .+.'.0.).0.).2.+
C80D  .byte 00 32 00 2B 00 8A 0C 1B 18 7F 00 19 18 7F 00 1B ; .2.+............
C81D  .byte 18 7F 00 19 18 7F 00 1B 00 7F 00 19 00 7F 00 20 ; ............... 
C82D  .byte 18 7F 00 1B 18 7F 00 19 18 8D 19 30 8D 19 24 7F ; ...........0..$.
C83D  .byte 00 19 18 7F 00 1B 18 7F 00 20 18 19 18 7F 00 1B ; ......... ......
C84D  .byte 18 7F 00 20 18 20 24 1B 00 8D 1B 30 8D 1B 30 8D ; ... . $....0..0.
C85D  .byte 1B 18 7F 18 88 7F 30 20 00 19 18 20 00 1B 18 20 ; ......0 ... ... 
C86D  .byte 00 1B 18 17 24 8D 17 00 7F 18 19 00 24 00 22 18 ; ....$.......$.".
C87D  .byte 20 00 1B 18 20 00 1B 18 17 24 7F 30 20 00 19 18 ;  ... ....$.0 ...
C88D  .byte 20 00 1B 18 20 00 1B 18 17 24 8D 17 00 7F 18 19 ;  ... ....$......
C89D  .byte 00 19 00 15 18 19 00 17 18 19 00 17 18 10 24 7F ; ..............$.
C8AD  .byte 30 20 00 19 18 20 00 1B 18 20 00 1B 18 17 24 8D ; 0 ... ... ....$.
C8BD  .byte 17 00 7F 18 19 00 24 00 22 18 20 00 1B 18 20 00 ; ......$.". ... .
C8CD  .byte 1B 18 17 24 7F 30 20 00 19 18 20 00 1B 18 20 00 ; ...$.0 ... ... .
C8DD  .byte 1B 18 17 24 8D 17 00 7F 18 19 00 19 00 15 18 19 ; ...$............
C8ED  .byte 00 17 18 19 00 17 18 10 18 14 00 12 30 8D 12 30 ; ............0..0
C8FD  .byte 8D 12 30 8D 12 00 10 00 12 00 14 00 8D 14 30 8D ; ..0...........0.
C90D  .byte 14 30 8D 14 30 8D 14 00 10 00 19 00 13 00 8D 13 ; .0..0...........
C91D  .byte 30 8D 13 30 8D 13 30 8D 13 00 10 00 13 00 12 00 ; 0..0..0.........
C92D  .byte 8D 12 48 8D 12 00 24 18 24 00 25 00 24 00 27 00 ; ..H...$.$.%.$.'.
C93D  .byte 24 00 24 00 20 00 FF 82 FF 1E 96 00 32 0A 81 0D ; $.$. .......2...
C94D  .byte 8A 0C 7F 00 0C 00 09 00 0C 00 0D 00 0A 00 0E 00 ; ................
C95D  .byte 0B 00 86 00 06 8C 8C 8C 8C 00 06 8B 8B 8B 8B 87 ; ................
C96D  .byte 18 90 01 00 00 00 00 0C 00 0C 00 0D 00 0D 00 0E ; ................
C97D  .byte 00 0E 00 86 00 06 8C 8C 8C 8C 00 06 8B 8B 8B 8B ; ................
C98D  .byte 87 1E B1 01 02 00 04 00 88 86 05 00 05 00 15 00 ; ................
C99D  .byte 05 00 05 00 05 00 15 00 05 00 04 00 04 00 14 00 ; ................
C9AD  .byte 04 00 04 00 00 00 02 00 04 00 87 02 C7 01 05 00 ; ................
C9BD  .byte 05 00 15 00 05 00 05 00 05 00 15 00 05 00 04 00 ; ................
C9CD  .byte 04 00 14 00 04 00 04 00 04 00 14 00 04 00 02 00 ; ................
C9DD  .byte 02 00 12 00 02 00 02 00 02 00 12 00 02 00 00 00 ; ................
C9ED  .byte 00 00 10 00 00 00 00 00 00 00 02 00 04 00 86 05 ; ................
C9FD  .byte 00 05 00 15 00 05 00 05 00 05 00 15 00 05 00 04 ; ................
CA0D  .byte 00 04 00 14 00 04 00 04 00 00 00 02 00 04 00 87 ; ................
CA1D  .byte 02 2C 02 05 00 05 00 15 00 05 00 05 00 05 00 15 ; .,..............
CA2D  .byte 00 05 00 04 00 04 00 14 00 04 00 04 00 04 00 14 ; ................
CA3D  .byte 00 04 00 02 00 02 00 12 00 02 00 02 00 02 00 12 ; ................
CA4D  .byte 00 02 00 00 00 00 00 10 00 00 00 00 00 00 00 00 ; ................
CA5D  .byte 00 00 00 0A 24 09 24 07 24 05 24 04 18 02 18 8A ; ....$.$.$.$.....
CA6D  .byte 24 0C 00 0E 00 00 00 02 00 04 18 09 18 08 00 07 ; $...............
CA7D  .byte 00 05 00 03 00 02 18 00 18 07 24 02 24 07 24 8A ; ..........$.$.$.
CA8D  .byte 0C 07 00 04 00 04 00 05 00 05 00 06 00 07 00 FF ; ................
CA9D  .byte 82 FF 1E 82 00 32 0A 81 09 8A 06 19 00 15 00 19 ; .....2..........
CAAD  .byte 00 15 00 1B 00 17 00 1B 00 17 00 20 00 19 00 20 ; ........... ... 
CABD  .byte 00 19 00 22 00 1B 00 22 00 1B 00 30 00 81 04 30 ; ..."..."...0...0
CACD  .byte 00 81 09 2B 00 81 04 30 00 81 09 29 00 81 04 2B ; ...+...0...)...+
CADD  .byte 00 81 09 27 00 81 04 29 00 86 81 09 30 00 81 04 ; ...'...)....0...
CAED  .byte 27 00 81 09 2B 00 81 04 30 00 81 09 29 00 81 04 ; '...+...0...)...
CAFD  .byte 2B 00 81 09 27 00 81 04 29 00 87 0F 17 03 88 84 ; +...'...).......
CB0D  .byte 04 00 81 09 8A 0C 7F 30 20 00 19 18 20 00 1B 18 ; .......0 ... ...
CB1D  .byte 20 00 1B 18 8C 27 06 29 06 30 00 29 00 7F 24 8B ;  ....'.).0.)..$.
CB2D  .byte 19 00 24 00 22 18 20 00 1B 18 20 00 1B 18 8C 27 ; ..$.". ... ....'
CB3D  .byte 06 29 06 30 00 34 00 8B 7F 30 20 00 19 18 20 00 ; .).0.4...0 ... .
CB4D  .byte 1B 18 20 00 1B 18 8C 27 06 29 06 30 00 29 00 7F ; .. ....'.).0.)..
CB5D  .byte 24 8B 19 00 19 00 15 18 19 00 17 18 19 00 17 18 ; $...............
CB6D  .byte 10 24 8A 06 86 86 81 09 30 00 81 05 30 00 81 09 ; .$......0...0...
CB7D  .byte 29 00 81 05 30 00 81 09 25 00 81 05 29 00 81 09 ; )...0...%...)...
CB8D  .byte 22 00 81 05 25 00 87 02 A3 03 81 09 2B 00 81 05 ; "...%.......+...
CB9D  .byte 22 00 81 09 27 00 81 05 2B 00 81 09 32 00 81 05 ; "...'...+...2...
CBAD  .byte 27 00 81 09 2B 00 81 05 32 00 81 09 2B 00 81 05 ; '...+...2...+...
CBBD  .byte 2B 00 81 09 27 00 81 05 2B 00 81 09 27 00 81 05 ; +...'...+...'...
CBCD  .byte 27 00 81 09 24 00 81 05 27 00 87 03 A2 03 81 09 ; '...$...'.......
CBDD  .byte 30 00 81 05 24 00 81 09 29 00 81 05 30 00 81 09 ; 0...$...)...0...
CBED  .byte 25 00 81 05 29 00 81 09 22 00 81 05 25 00 81 09 ; %...)..."...%...
CBFD  .byte 30 00 81 05 30 00 81 09 29 00 81 05 30 00 81 09 ; 0...0...)...0...
CC0D  .byte 25 00 81 05 29 00 8A 0C 81 09 34 00 81 05 34 00 ; %...).....4...4.
CC1D  .byte 81 09 30 00 81 05 34 00 81 09 29 00 81 05 30 00 ; ..0...4...)...0.
CC2D  .byte 81 09 29 00 30 00 34 00 8A 06 86 81 09 2A 00 81 ; ..).0.4......*..
CC3D  .byte 05 2A 00 81 09 25 00 81 05 2A 00 81 09 32 00 81 ; .*...%...*...2..
CC4D  .byte 05 25 00 81 09 25 00 81 05 32 00 87 04 68 04 86 ; .%...%...2...h..
CC5D  .byte 81 09 29 00 81 05 29 00 81 09 24 00 81 05 29 00 ; ..)...)...$...).
CC6D  .byte 81 09 30 00 81 05 24 00 81 09 24 00 81 05 30 00 ; ..0...$...$...0.
CC7D  .byte 87 04 8D 04 86 81 09 28 00 81 05 28 00 81 09 23 ; .......(...(...#
CC8D  .byte 00 81 05 28 00 81 09 30 00 81 05 23 00 81 09 23 ; ...(...0...#...#
CC9D  .byte 00 81 05 30 00 87 04 B2 04 86 81 09 30 00 81 05 ; ...0........0...
CCAD  .byte 30 00 81 09 29 00 81 05 30 00 81 09 34 00 81 05 ; 0...)...0...4...
CCBD  .byte 29 00 81 09 29 00 81 05 34 00 87 04 D7 04 FF 81 ; )...)...4.......
CCCD  .byte 09 8A 06 8A 0C 81 09 70 00 70 00 81 0C 71 00 81 ; .......p.p...q..
CCDD  .byte 09 70 00 70 00 81 0C 71 00 71 00 71 00 88 86 81 ; .p.p...q.q.q....
CCED  .byte 09 70 00 70 00 81 0C 71 00 81 09 70 00 87 0F 1C ; .p.p...q...p....
CCFD  .byte 05 70 00 81 0C 71 00 71 00 71 00 FF 00 0A 00 D8 ; .p...q.q.q......
CD0D  .byte 00 CC 01 05 05 00 00 80 01 00 01 00 85 FF 82 AF ; ................
CD1D  .byte 14 A0 00 05 01 8A 0C 81 0B 09 00 0B 00 10 00 14 ; ................
CD2D  .byte 00 88 81 0C 83 0C 01 04 06 00 86 1B 18 1B 00 19 ; ................
CD3D  .byte 00 87 03 2E 00 1B 00 19 00 14 00 10 00 17 18 19 ; ................
CD4D  .byte 00 15 30 8D 15 30 8D 15 30 7F 00 86 19 18 19 00 ; ..0..0..0.......
CD5D  .byte 17 00 87 03 4F 00 19 18 1B 18 15 24 14 30 8D 14 ; ....O......$.0..
CD6D  .byte 30 7F 00 09 00 0B 00 10 00 14 00 86 1B 18 1B 00 ; 0...............
CD7D  .byte 19 00 87 03 6F 00 1B 00 19 00 14 00 10 00 17 18 ; ....o...........
CD8D  .byte 19 00 15 30 8D 15 30 8D 15 30 7F 00 19 30 8D 19 ; ...0..0..0...0..
CD9D  .byte 18 1B 18 18 30 8D 18 12 7F 06 1B 00 7F 00 1B 24 ; ....0..........$
CDAD  .byte 19 00 8D 19 30 8D 19 30 8D 19 30 86 29 00 30 06 ; ....0..0..0.).0.
CDBD  .byte 29 06 30 00 29 00 2B 00 27 00 22 00 2B 00 25 00 ; ).0.).+.'.".+.%.
CDCD  .byte 29 06 25 06 29 00 25 00 27 00 29 00 2B 00 27 00 ; ).%.).%.'.).+.'.
CDDD  .byte 87 02 AF 00 FF 81 0A 82 FF 14 82 14 32 01 7F 30 ; ............2..0
CDED  .byte 88 8A 0C 81 0D 86 0C 00 0C 00 04 00 04 00 02 00 ; ................
CDFD  .byte 02 00 04 00 04 00 87 02 E9 00 86 02 00 02 00 09 ; ................
CE0D  .byte 00 09 00 05 00 05 00 09 00 09 00 87 02 FE 00 86 ; ................
CE1D  .byte 0E 00 0E 00 07 00 07 00 02 00 02 00 07 00 07 00 ; ................
CE2D  .byte 87 02 13 01 00 00 00 00 07 00 07 00 04 00 04 00 ; ................
CE3D  .byte 07 00 07 00 0E 00 0E 00 05 00 05 00 04 00 04 00 ; ................
CE4D  .byte 0E 00 0E 00 86 0C 00 0C 00 04 00 04 00 02 00 02 ; ................
CE5D  .byte 00 04 00 04 00 87 02 48 01 86 02 00 02 00 09 00 ; .......H........
CE6D  .byte 09 00 05 00 05 00 09 00 09 00 87 02 5D 01 0E 00 ; ............]...
CE7D  .byte 0E 00 05 00 05 00 02 00 02 00 05 00 05 00 04 00 ; ................
CE8D  .byte 04 00 0B 00 0B 00 08 00 08 00 0B 00 0B 00 86 0C ; ................
CE9D  .byte 00 0C 00 04 00 04 00 02 00 02 00 04 00 04 00 87 ; ................
CEAD  .byte 02 92 01 86 09 00 09 00 09 00 09 00 07 00 07 00 ; ................
CEBD  .byte 07 00 07 00 05 00 05 00 05 00 05 00 07 00 07 00 ; ................
CECD  .byte 07 00 07 00 87 02 A7 01 FF 84 04 00 82 B9 14 82 ; ................
CEDD  .byte 00 05 01 8A 0C 81 0A 09 00 0B 00 10 00 14 00 88 ; ................
CEED  .byte 81 09 83 0C 01 04 06 00 7F 0C 86 1B 18 1B 00 19 ; ................
CEFD  .byte 00 87 03 EE 01 1B 00 19 00 14 00 10 00 17 18 19 ; ................
CF0D  .byte 00 15 00 8D 15 24 8A 03 8B 84 00 00 35 00 8C 8C ; .....$......5...
CF1D  .byte 8C 35 00 8B 8B 8B 32 00 8C 8C 8C 35 00 8B 8B 8B ; .5....2....5....
CF2D  .byte 29 00 8C 8C 8C 32 00 8B 8B 8B 25 00 8C 8C 8C 29 ; )....2....%....)
CF3D  .byte 00 8B 8B 8B 32 00 8C 8C 8C 25 00 8B 8B 8B 29 00 ; ....2....%....).
CF4D  .byte 8C 8C 8C 32 00 8B 8B 8B 25 00 8C 8C 8C 29 00 8B ; ...2....%....)..
CF5D  .byte 8B 8B 22 00 8C 8C 8C 25 00 8B 8B 8B 29 00 8C 8C ; .."....%....)...
CF6D  .byte 8C 22 00 8B 8B 8B 25 00 8C 8C 8C 29 00 8B 8B 8B ; ."....%....)....
CF7D  .byte 22 00 8C 8C 8C 25 00 8B 8B 8B 19 00 8C 8C 8C 22 ; "....%........."
CF8D  .byte 00 8B 8B 8B 25 00 8C 8C 8C 19 00 8B 8B 8B 22 00 ; ....%.........".
CF9D  .byte 8C 8C 8C 25 00 8B 8B 8B 19 00 8C 8C 8C 22 00 8B ; ...%........."..
CFAD  .byte 8B 8B 15 00 8C 8C 8C 19 00 8B 8B 8B 8C 84 04 00 ; ................
CFBD  .byte 8A 0C 7F 00 86 19 18 19 00 17 00 87 03 B8 02 19 ; ................
CFCD  .byte 18 1B 18 15 24 14 00 8D 14 18 84 00 00 8B 35 0C ; ....$.........5.
CFDD  .byte 8C 8C 8C 35 0C 8B 8B 8B 35 06 8C 8C 8C 35 06 8B ; ...5....5....5..
CFED  .byte 8B 8B 32 0C 8C 8C 8C 35 0C 8B 8B 8B 2B 06 8C 8C ; ..2....5....+...
CFFD  .byte 8C 32 06 8B 8B 8B 28 24 81 09 7F 0C 86 1B 18 1B ; .2....($........
D00D  .byte 00 19 00 87 03 00 03 1B 00 19 00 14 00 10 00 17 ; ................
D01D  .byte 18 19 00 15 00 8D 15 24 8A 03 8B 84 00 00 35 00 ; .......$......5.
D02D  .byte 8C 8C 8C 35 00 8B 8B 8B 32 00 8C 8C 8C 35 00 8B ; ...5....2....5..
D03D  .byte 8B 8B 29 00 8C 8C 8C 32 00 8B 8B 8B 25 00 8C 8C ; ..)....2....%...
D04D  .byte 8C 29 00 8B 8B 8B 32 00 8C 8C 8C 25 00 8B 8B 8B ; .)....2....%....
D05D  .byte 29 00 8C 8C 8C 32 00 8B 8B 8B 25 00 8C 8C 8C 29 ; )....2....%....)
D06D  .byte 00 8B 8B 8B 22 00 8C 8C 8C 25 00 8B 8B 8B 29 00 ; ...."....%....).
D07D  .byte 8C 8C 8C 22 00 8B 8B 8B 25 00 8C 8C 8C 29 00 8B ; ..."....%....)..
D08D  .byte 8B 8B 22 00 8C 8C 8C 25 00 8B 8B 8B 19 00 8C 8C ; .."....%........
D09D  .byte 8C 22 00 8B 8B 8B 25 00 8C 8C 8C 19 00 8B 8B 8B ; ."....%.........
D0AD  .byte 22 00 8C 8C 8C 25 00 8B 8B 8B 19 00 8C 8C 8C 22 ; "....%........."
D0BD  .byte 00 8B 8B 8B 15 00 8C 8C 8C 19 00 8B 8B 8B 8C 8A ; ................
D0CD  .byte 0C 7F 0C 19 30 8D 19 18 1B 18 18 30 8D 18 12 7F ; ....0......0....
D0DD  .byte 06 1B 00 7F 00 1B 24 19 00 8D 19 18 8B 34 00 8C ; ......$......4..
D0ED  .byte 8C 8C 19 00 8B 8B 8B 34 00 32 00 8C 8C 8C 34 00 ; .......4.2....4.
D0FD  .byte 8B 8B 8B 30 06 8C 8C 8C 34 06 8B 8B 8B 2B 24 81 ; ...0....4....+$.
D10D  .byte 0B 8A 06 7F 0C 24 00 8C 8C 8C 24 00 8B 8B 8B 20 ; .....$....$.... 
D11D  .byte 00 8C 8C 8C 24 00 8B 8B 8B 86 19 00 8C 8C 8C 24 ; ....$..........$
D12D  .byte 00 8C 87 02 1D 04 81 0B 22 00 8C 8C 8C 19 00 8B ; ........".......
D13D  .byte 8B 8B 1B 00 8C 8C 8C 22 00 8B 8B 8B 86 17 00 8C ; ......."........
D14D  .byte 8C 8C 1B 00 8C 87 02 40 04 81 0B 20 00 8C 8C 8C ; .......@... ....
D15D  .byte 17 00 8B 8B 8B 19 00 8C 8C 8C 20 00 8B 8B 8B 86 ; .......... .....
D16D  .byte 15 00 8C 8C 8C 19 00 8C 87 02 63 04 81 0B 22 00 ; ..........c...".
D17D  .byte 8C 8C 8C 19 00 8B 8B 8B 1B 00 8C 8C 8C 22 00 8B ; ............."..
D18D  .byte 8B 8B 86 17 00 8C 8C 8C 1B 00 8C 87 02 86 04 81 ; ................
D19D  .byte 0B 24 00 8C 8C 8C 17 00 8B 8B 8B 20 00 8C 8C 8C ; .$......... ....
D1AD  .byte 24 00 8B 8B 8B 86 19 00 8C 8C 8C 24 00 8C 87 02 ; $..........$....
D1BD  .byte A9 04 81 0B 22 00 8C 8C 8C 19 00 8B 8B 8B 1B 00 ; ...."...........
D1CD  .byte 8C 8C 8C 22 00 8B 8B 8B 86 17 00 8C 8C 8C 1B 00 ; ..."............
D1DD  .byte 8C 87 02 CC 04 81 0B 20 00 8C 8C 8C 17 00 8B 8B ; ....... ........
D1ED  .byte 8B 19 00 8C 8C 8C 20 00 8B 8B 8B 15 00 8C 8C 8C ; ...... .........
D1FD  .byte 19 00 8B 8B 8B 8A 0C 81 0C 09 00 0B 00 10 00 14 ; ................
D20D  .byte 00 FF 81 09 8A 0C 70 00 7F 00 70 00 7F 00 88 86 ; ......p...p.....
D21D  .byte 81 09 70 00 70 00 81 0C 71 00 81 09 70 00 87 20 ; ..p.p...q...p.. 
D22D  .byte 13 05 86 81 0C 71 00 81 09 70 00 70 00 81 0C 71 ; .....q...p.p...q
D23D  .byte 00 87 08 26 05 FF 00 00 00 00 00 00 00 0A 00 98 ; ...&............
D24D  .byte 01 25 03 AA 04 00 00 80 01 00 01 00 85 FF 88 83 ; .%..............
D25D  .byte 10 01 04 07 00 82 FF 14 96 00 14 0A 81 0D 8A 06 ; ................
D26D  .byte 7F 12 29 00 7F 0C 26 0C 29 00 7F 0C 2B 00 7F 0C ; ..)...&.)...+...
D27D  .byte 26 00 7F 0C 26 00 7F 0C 24 00 22 00 7F 0C 1B 00 ; &...&...$.".....
D28D  .byte 7F 0C 22 00 7F 00 22 00 24 00 7F 00 26 00 7F 0C ; .."...".$...&...
D29D  .byte 24 0C 8D 24 30 7F 12 7F 12 1B 12 22 12 2B 12 29 ; $..$0......".+.)
D2AD  .byte 12 26 0C 22 00 7F 0C 24 00 26 00 7F 0C 27 00 7F ; .&."...$.&...'..
D2BD  .byte 0C 27 00 7F 0C 28 00 7F 0C 28 00 7F 00 29 0C 8D ; .'...(...(...)..
D2CD  .byte 29 30 7F 12 7F 12 29 00 7F 0C 26 0C 29 00 7F 0C ; )0....)...&.)...
D2DD  .byte 2B 00 7F 0C 26 00 7F 0C 26 00 7F 0C 24 00 22 00 ; +...&...&...$.".
D2ED  .byte 7F 0C 1B 00 7F 0C 22 00 7F 00 22 00 24 00 7F 00 ; ......"...".$...
D2FD  .byte 26 00 7F 0C 24 0C 8D 24 30 7F 12 7F 12 1B 12 22 ; &...$..$0......"
D30D  .byte 12 2B 12 29 12 26 0C 22 00 7F 0C 24 00 26 00 7F ; .+.).&."...$.&..
D31D  .byte 0C 24 00 7F 0C 24 00 7F 00 24 00 19 0C 1B 00 21 ; .$...$...$.....!
D32D  .byte 0C 22 0C 8D 22 30 7F 12 8C 8C 7F 12 2B 03 27 03 ; .".."0......+.'.
D33D  .byte 7F 0C 2B 03 27 03 7F 0C 29 03 22 03 7F 00 27 03 ; ..+.'...)."...'.
D34D  .byte 22 03 7F 0C 27 03 2B 03 7F 0C 27 03 2B 03 2B 03 ; "...'.+...'.+.+.
D35D  .byte 32 03 7F 0C 27 03 2B 03 7F 0C 7F 0C 26 03 29 03 ; 2...'.+.....&.).
D36D  .byte 7F 0C 26 03 29 03 29 03 32 03 7F 0C 22 03 26 03 ; ..&.).).2...".&.
D37D  .byte 7F 00 86 26 03 8C 29 03 87 03 36 01 86 86 26 03 ; ...&..)...6...&.
D38D  .byte 29 03 87 02 41 01 8B 87 05 40 01 8C 8C 7F 12 2B ; )...A....@.....+
D39D  .byte 03 27 03 7F 0C 2B 03 27 03 7F 0C 29 03 22 03 7F ; .'...+.'...)."..
D3AD  .byte 00 27 03 22 03 7F 0C 8B 8B 27 00 7F 0C 27 00 29 ; .'.".....'...'.)
D3BD  .byte 00 7F 0C 2B 00 7F 0C 86 29 00 7F 00 7F 00 8C 8C ; ...+....).......
D3CD  .byte 8C 87 04 7B 01 81 0D 29 00 7F 00 2B 00 7F 0C 29 ; ...{...)...+...)
D3DD  .byte 00 8D 29 24 FF 88 82 FF 14 96 00 14 0A 81 0E 8A ; ..)$............
D3ED  .byte 06 02 0C 0C 00 0E 00 7F 00 02 00 0C 00 7F 00 0E ; ................
D3FD  .byte 00 7F 0C 02 12 02 00 7F 0C 02 00 7F 0C 04 00 06 ; ................
D40D  .byte 00 7F 0C 07 00 7F 0C 07 00 7F 00 07 00 08 00 7F ; ................
D41D  .byte 0C 08 00 7F 00 09 00 7F 0C 09 00 09 00 7F 0C 0C ; ................
D42D  .byte 00 7F 00 0C 00 0E 00 7F 00 0C 00 07 0C 02 00 07 ; ................
D43D  .byte 00 7F 00 02 12 04 00 07 00 7F 0C 06 0C 02 00 06 ; ................
D44D  .byte 00 7F 00 0E 00 7F 0C 01 00 02 00 7F 0C 04 00 7F ; ................
D45D  .byte 0C 04 00 7F 0C 04 00 7F 0C 04 00 7F 00 09 00 7F ; ................
D46D  .byte 0C 04 00 11 0C 0B 00 09 0C 07 00 04 00 7F 00 0C ; ................
D47D  .byte 00 02 0C 0C 00 0E 00 7F 00 02 00 0C 00 7F 00 0E ; ................
D48D  .byte 00 7F 0C 02 12 02 00 7F 0C 02 00 7F 0C 04 00 06 ; ................
D49D  .byte 00 7F 0C 07 00 7F 0C 07 00 7F 00 07 00 08 00 7F ; ................
D4AD  .byte 0C 08 00 7F 00 09 00 7F 0C 09 00 09 00 7F 0C 0C ; ................
D4BD  .byte 00 7F 00 0C 00 0E 00 7F 00 0C 00 07 0C 02 00 07 ; ................
D4CD  .byte 00 7F 00 02 12 04 00 07 00 7F 0C 06 0C 02 00 06 ; ................
D4DD  .byte 00 7F 00 0E 00 7F 0C 01 00 02 00 7F 0C 04 0C 02 ; ................
D4ED  .byte 00 04 00 7F 00 04 00 09 00 7F 00 0C 00 0E 00 7F ; ................
D4FD  .byte 00 01 00 02 00 7F 0C 0C 00 7F 0C 02 00 7F 00 02 ; ................
D50D  .byte 00 04 00 7F 00 06 00 8A 12 07 00 02 00 04 00 02 ; ................
D51D  .byte 00 07 00 06 00 04 00 02 00 02 00 0C 00 0E 00 0C ; ................
D52D  .byte 00 02 00 01 00 0E 00 0C 00 07 00 02 00 04 00 02 ; ................
D53D  .byte 00 07 00 06 00 04 00 02 00 8A 06 09 00 7F 0C 0C ; ................
D54D  .byte 00 7F 00 0D 00 0E 00 7F 00 0D 00 0C 00 7F 0C 09 ; ................
D55D  .byte 00 7F 00 0C 00 7F 0C 0C 00 0E 00 7F 0C 01 00 7F ; ................
D56D  .byte 0C FF 88 84 04 00 82 FF 14 82 00 14 0A 81 0B 8A ; ................
D57D  .byte 06 7F 12 26 00 7F 0C 22 0C 26 00 7F 0C 26 00 7F ; ...&...".&...&..
D58D  .byte 0C 22 00 7F 0C 22 00 7F 0C 19 00 19 00 7F 0C 17 ; ."..."..........
D59D  .byte 00 7F 0C 17 00 7F 0C 1B 00 7F 00 1B 00 7F 0C 19 ; ................
D5AD  .byte 00 7F 0C 19 00 14 00 7F 00 11 00 19 00 7F 00 19 ; ................
D5BD  .byte 00 1B 00 7F 00 19 00 7F 12 17 12 1B 12 22 12 26 ; .............".&
D5CD  .byte 12 19 0C 19 00 7F 0C 1B 00 1B 00 7F 0C 1B 00 7F ; ................
D5DD  .byte 0C 1B 00 7F 0C 1B 00 7F 0C 1B 00 7F 00 21 00 7F ; .............!..
D5ED  .byte 0C 22 00 24 00 7F 00 26 00 24 00 7F 00 21 00 19 ; .".$...&.$...!..
D5FD  .byte 00 7F 00 14 00 7F 12 26 00 7F 0C 22 0C 26 00 7F ; .......&...".&..
D60D  .byte 0C 26 00 7F 0C 22 00 7F 0C 22 00 7F 0C 19 00 19 ; .&..."..."......
D61D  .byte 00 7F 0C 17 00 7F 0C 17 00 7F 0C 1B 00 7F 00 1B ; ................
D62D  .byte 00 7F 0C 19 00 7F 0C 19 00 14 00 7F 00 11 00 19 ; ................
D63D  .byte 00 7F 00 19 00 1B 00 7F 00 19 00 7F 12 17 12 1B ; ................
D64D  .byte 12 22 12 26 12 19 0C 19 00 7F 0C 1B 00 1B 00 7F ; .".&............
D65D  .byte 0C 1B 00 7F 0C 1B 00 7F 0C 14 0C 14 00 14 00 7F ; ................
D66D  .byte 00 19 00 8D 19 24 02 00 04 00 06 00 09 00 12 00 ; .....$..........
D67D  .byte 16 00 27 00 7F 0C 27 00 7F 0C 27 00 7F 0C 22 00 ; ..'...'...'...".
D68D  .byte 7F 00 1B 00 7F 0C 27 00 7F 0C 27 00 2B 00 7F 0C ; ......'...'.+...
D69D  .byte 27 00 7F 0C 7F 0C 26 00 7F 0C 26 00 29 00 7F 0C ; '.....&...&.)...
D6AD  .byte 22 00 7F 00 26 00 8D 26 12 12 12 14 12 16 12 17 ; "...&..&........
D6BD  .byte 12 27 00 7F 0C 27 00 7F 0C 22 00 7F 00 1B 00 7F ; .'...'..."......
D6CD  .byte 0C 22 00 7F 0C 22 00 22 00 7F 0C 22 0C 7F 00 24 ; ."..."."..."...$
D6DD  .byte 00 7F 0C 7F 12 7F 12 7F 12 24 00 7F 00 24 00 7F ; .........$...$..
D6ED  .byte 0C 24 00 8D 24 24 FF 8A 06 88 86 81 09 70 00 7F ; .$..$$.......p..
D6FD  .byte 00 70 00 81 0C 71 00 7F 00 81 09 70 00 87 1F AE ; .p...q.....p....
D70D  .byte 04 70 00 7F 00 70 00 81 0C 71 00 71 00 71 00 86 ; .p...p...q.q.q..
D71D  .byte 81 09 70 00 7F 00 70 00 81 0C 71 00 7F 00 81 09 ; ..p...p...q.....
D72D  .byte 70 00 87 0C D3 04 86 70 00 7F 0C 87 04 EA 04 81 ; p......p........
D73D  .byte 0C 86 71 00 87 0C F5 04 FF 00 00 00 00 0A 00 FB ; ..q.............
D74D  .byte 00 BC 01 E0 03 00 00 80 01 00 01 00 85 FF 82 FF ; ................
D75D  .byte 14 A0 00 32 01 83 14 01 04 05 00 8A 0C 81 0C 88 ; ...2............
D76D  .byte 7F 18 29 00 7F 00 27 12 30 06 7F 00 25 00 7F 00 ; ..)...'.0...%...
D77D  .byte 25 00 25 00 7F 00 24 12 29 06 7F 00 22 00 7F 00 ; %.%...$.)..."...
D78D  .byte 22 00 22 00 7F 00 20 12 25 06 7F 00 2A 00 7F 00 ; "."... .%...*...
D79D  .byte 29 00 27 00 25 00 25 18 27 18 7F 18 29 00 7F 00 ; ).'.%.%.'...)...
D7AD  .byte 27 12 30 06 7F 00 25 00 7F 00 25 00 25 00 7F 00 ; '.0...%...%.%...
D7BD  .byte 24 12 29 06 7F 00 22 00 7F 00 22 00 22 00 7F 00 ; $.)..."..."."...
D7CD  .byte 20 12 25 06 7F 00 27 30 8D 27 30 7F 00 25 30 8D ;  .%...'0.'0..%0.
D7DD  .byte 25 00 25 00 27 00 28 00 28 24 27 30 7F 00 28 30 ; %.%.'.(.($'0..(0
D7ED  .byte 8D 28 00 28 00 2A 00 30 00 2A 00 33 00 81 08 2A ; .(.(.*.0.*.3...*
D7FD  .byte 00 81 0C 2A 00 81 08 33 00 81 0C 27 00 81 08 2A ; ...*...3...'...*
D80D  .byte 00 81 0C 25 00 8D 25 30 7F 00 25 00 27 00 28 00 ; ...%..%0..%.'.(.
D81D  .byte 27 00 2A 00 81 08 27 00 81 0C 27 00 81 08 2A 00 ; '.*...'...'...*.
D82D  .byte 81 0C 23 00 81 08 27 00 81 0C 25 30 8D 25 30 8D ; ..#...'...%0.%0.
D83D  .byte 25 30 8D 25 30 7F 00 FF 82 FF 1E 82 00 32 01 8A ; %0.%0........2..
D84D  .byte 0C 81 0D 88 05 00 05 00 15 00 05 00 04 00 04 00 ; ................
D85D  .byte 14 00 04 00 02 00 02 00 12 00 02 00 00 00 00 00 ; ................
D86D  .byte 10 00 00 00 0D 00 0D 00 0A 00 0D 00 0C 00 0C 00 ; ................
D87D  .byte 09 00 0C 00 0D 00 0D 00 0A 00 0D 00 00 00 00 00 ; ................
D88D  .byte 10 00 00 00 05 00 05 00 15 00 05 00 04 00 04 00 ; ................
D89D  .byte 14 00 04 00 02 00 02 00 12 00 02 00 00 00 00 00 ; ................
D8AD  .byte 10 00 00 00 0D 00 0D 00 0A 00 0D 00 0C 00 0C 00 ; ................
D8BD  .byte 09 00 0C 00 03 00 03 00 13 00 03 00 00 00 00 00 ; ................
D8CD  .byte 10 00 00 00 86 86 01 00 01 00 11 00 01 00 87 02 ; ................
D8DD  .byte 89 01 86 03 00 03 00 13 00 03 00 87 02 96 01 87 ; ................
D8ED  .byte 03 88 01 86 05 00 05 00 15 00 05 00 87 03 A7 01 ; ................
D8FD  .byte 05 00 00 00 02 00 04 00 FF 82 FF 14 96 00 32 01 ; ..............2.
D90D  .byte 83 14 01 04 05 00 81 08 88 8A 0C 84 04 00 7F 12 ; ................
D91D  .byte 7F 18 29 00 7F 00 27 12 30 06 7F 00 25 00 7F 00 ; ..)...'.0...%...
D92D  .byte 25 00 25 00 7F 00 24 12 29 06 7F 00 22 00 7F 00 ; %.%...$.)..."...
D93D  .byte 22 00 22 00 7F 00 20 12 25 06 7F 00 2A 00 7F 00 ; "."... .%...*...
D94D  .byte 29 00 27 00 25 00 25 18 27 18 7F 18 29 00 7F 00 ; ).'.%.%.'...)...
D95D  .byte 27 12 30 06 7F 00 25 00 7F 00 25 00 25 00 7F 00 ; '.0...%...%.%...
D96D  .byte 24 12 29 06 7F 00 22 00 7F 00 22 00 22 00 7F 00 ; $.)..."..."."...
D97D  .byte 20 12 25 06 7F 00 27 30 8D 27 18 8D 27 12 84 00 ;  .%...'0.'..'...
D98D  .byte 00 8A 03 86 86 11 00 81 04 11 00 81 08 08 00 81 ; ................
D99D  .byte 04 11 00 81 08 11 00 81 04 08 00 81 08 15 00 81 ; ................
D9AD  .byte 04 11 00 81 08 18 00 81 04 15 00 81 08 15 00 81 ; ................
D9BD  .byte 04 18 00 81 08 11 00 81 04 15 00 81 08 08 00 81 ; ................
D9CD  .byte 04 11 00 81 08 87 02 48 02 86 13 00 81 04 0A 00 ; .......H........
D9DD  .byte 81 08 0A 00 81 04 13 00 81 08 13 00 81 04 0A 00 ; ................
D9ED  .byte 81 08 17 00 81 04 13 00 81 08 1A 00 81 04 17 00 ; ................
D9FD  .byte 81 08 17 00 81 04 1A 00 81 08 13 00 81 04 17 00 ; ................
DA0D  .byte 81 08 0A 00 81 04 13 00 81 08 87 02 8D 02 87 02 ; ................
DA1D  .byte 47 02 86 15 00 81 04 11 00 81 08 11 00 81 04 15 ; G...............
DA2D  .byte 00 81 08 15 00 81 04 11 00 81 08 18 00 81 04 15 ; ................
DA3D  .byte 00 81 08 21 00 81 04 18 00 81 08 18 00 81 04 21 ; ...!...........!
DA4D  .byte 00 81 08 15 00 81 04 18 00 81 08 11 00 81 04 15 ; ................
DA5D  .byte 00 81 08 87 02 D6 02 17 00 81 04 11 00 81 08 13 ; ................
DA6D  .byte 00 81 04 17 00 81 08 17 00 81 04 13 00 81 08 1A ; ................
DA7D  .byte 00 81 04 17 00 81 08 23 00 81 04 1A 00 81 08 1A ; .......#........
DA8D  .byte 00 81 04 23 00 81 08 17 00 81 04 1A 00 81 08 13 ; ...#............
DA9D  .byte 00 81 04 17 00 81 08 17 00 81 04 13 00 81 08 13 ; ................
DAAD  .byte 00 81 04 17 00 81 08 17 00 81 04 13 00 81 08 1A ; ................
DABD  .byte 00 81 04 17 00 81 08 23 00 81 04 1A 00 81 08 27 ; .......#.......'
DACD  .byte 00 81 04 23 00 81 08 2A 00 81 04 27 00 81 08 27 ; ...#...*...'...'
DADD  .byte 00 81 04 2A 00 81 08 86 19 00 81 04 29 00 81 08 ; ...*........)...
DAED  .byte 15 00 81 04 19 00 81 08 19 00 81 04 15 00 81 08 ; ................
DAFD  .byte 20 00 81 04 19 00 81 08 25 00 81 04 20 00 81 08 ;  .......%... ...
DB0D  .byte 20 00 81 04 25 00 81 08 25 00 81 04 20 00 81 08 ;  ...%...%... ...
DB1D  .byte 29 00 81 04 25 00 81 08 87 04 9B 03 FF 8A 0C 88 ; )...%...........
DB2D  .byte 86 81 09 70 00 70 00 81 0C 71 00 81 09 70 00 87 ; ...p.p...q...p..
DB3D  .byte 0F E4 03 81 09 70 00 70 00 81 0C 71 00 71 06 71 ; .....p.p...q.q.q
DB4D  .byte 06 FF 0A 00 3F 02 1D 03 04 06 00 00 80 04 00 03 ; ....?...........
DB5D  .byte 00 85 FF 82 FF 14 96 00 32 01 8A 06 81 0C 83 10 ; ........2.......
DB6D  .byte 01 04 06 00 22 00 24 00 25 00 27 00 88 81 0C 29 ; ....".$.%.'....)
DB7D  .byte 30 8D 29 0C 27 00 81 09 29 00 81 0C 25 00 24 00 ; 0.).'...)...%.$.
DB8D  .byte 81 09 25 00 81 0C 22 0C 8D 22 30 8C 8C 25 00 24 ; ..%...".."0..%.$
DB9D  .byte 00 25 00 8B 8B 22 00 24 00 25 00 27 00 29 30 8D ; .%...".$.%.'.)0.
DBAD  .byte 29 0C 27 00 81 09 29 00 81 0C 25 00 24 00 81 09 ; ).'...)...%.$...
DBBD  .byte 25 00 81 0C 27 00 8D 27 30 8D 27 18 8D 27 0C 81 ; %...'..'0.'..'..
DBCD  .byte 09 12 00 14 00 81 0C 24 00 24 00 22 00 24 00 20 ; .......$.$.".$. 
DBDD  .byte 00 17 00 81 08 20 00 81 0C 20 00 8D 20 30 22 00 ; ..... ... .. 0".
DBED  .byte 22 00 24 00 25 00 22 00 19 00 81 08 22 00 81 0C ; ".$.%."....."...
DBFD  .byte 22 00 8D 22 30 24 00 24 00 25 00 27 00 24 00 20 ; ".."0$.$.%.'.$. 
DC0D  .byte 00 7F 00 24 12 24 0C 25 00 81 09 24 00 81 0C 27 ; ...$.$.%...$...'
DC1D  .byte 00 81 09 25 00 81 0C 29 00 81 09 27 00 81 0C 29 ; ...%...)...'...)
DC2D  .byte 00 81 09 29 00 81 0C 30 00 2A 00 81 09 30 00 81 ; ...)...0.*...0..
DC3D  .byte 0C 29 00 8D 29 18 22 00 24 00 25 00 27 00 29 30 ; .)..).".$.%.'.)0
DC4D  .byte 8D 29 0C 27 00 8C 8C 8C 29 00 8B 8B 8B 25 00 24 ; .).'....)....%.$
DC5D  .byte 00 8C 8C 8C 25 00 8B 8B 8B 22 0C 8D 22 30 8C 8C ; ....%....".."0..
DC6D  .byte 25 00 24 00 25 00 8B 8B 22 00 24 00 25 00 27 00 ; %.$.%...".$.%.'.
DC7D  .byte 29 30 8D 29 0C 27 00 8C 8C 8C 29 00 8B 8B 8B 25 ; )0.).'....)....%
DC8D  .byte 00 24 00 8C 8C 8C 25 00 8B 8B 8B 27 00 8D 27 30 ; .$....%....'..'0
DC9D  .byte 8D 27 18 8D 27 0C 8C 8C 8C 12 00 14 00 8B 8B 8B ; .'..'...........
DCAD  .byte 24 00 24 00 22 00 24 00 20 00 17 00 81 08 20 00 ; $.$.".$. ..... .
DCBD  .byte 81 0C 20 00 8D 20 30 22 00 22 00 24 00 25 00 22 ; .. .. 0".".$.%."
DCCD  .byte 00 19 00 81 08 22 00 81 0C 22 00 8D 22 30 24 00 ; ....."...".."0$.
DCDD  .byte 24 00 25 00 27 00 24 00 20 00 7F 00 24 12 24 0C ; $.%.'.$. ...$.$.
DCED  .byte 25 00 8C 8C 8C 24 00 8B 8B 8B 27 00 8C 8C 8C 25 ; %....$....'....%
DCFD  .byte 00 8B 8B 8B 29 00 8C 8C 8C 27 00 8B 8B 8B 29 00 ; ....)....'....).
DD0D  .byte 8C 8C 8C 29 00 8B 8B 8B 30 00 2A 00 8C 8C 8C 30 ; ...)....0.*....0
DD1D  .byte 00 8B 8B 8B 29 30 81 08 29 00 81 0C 8A 0C 22 30 ; ....)0..)....."0
DD2D  .byte 8D 22 00 22 00 24 00 25 00 27 30 8D 27 00 27 00 ; .".".$.%.'0.'.'.
DD3D  .byte 25 00 24 00 25 30 8D 25 00 22 00 24 00 25 00 24 ; %.$.%0.%.".$.%.$
DD4D  .byte 30 8D 24 00 27 00 25 00 24 00 22 30 8D 22 00 22 ; 0.$.'.%.$."0."."
DD5D  .byte 00 24 00 25 00 27 30 8D 27 00 27 00 25 00 24 00 ; .$.%.'0.'.'.%.$.
DD6D  .byte 25 30 8D 25 00 22 00 24 00 25 00 24 30 8D 24 06 ; %0.%.".$.%.$0.$.
DD7D  .byte 7F 06 8A 06 19 00 7F 00 22 00 24 00 25 00 27 00 ; ........".$.%.'.
DD8D  .byte FF 82 FF 1E 82 00 32 01 8A 06 81 0D 7F 18 88 86 ; ......2.........
DD9D  .byte 86 02 00 02 00 09 00 09 00 05 00 05 00 09 00 09 ; ................
DDAD  .byte 00 87 04 4F 02 86 0D 00 0D 00 05 00 05 00 02 00 ; ...O............
DDBD  .byte 02 00 05 00 05 00 87 04 64 02 86 00 00 00 00 07 ; ........d.......
DDCD  .byte 00 07 00 04 00 04 00 07 00 07 00 87 02 79 02 86 ; .............y..
DDDD  .byte 02 00 02 00 09 00 09 00 05 00 05 00 09 00 09 00 ; ................
DDED  .byte 87 02 8E 02 00 00 00 00 07 00 07 00 04 00 04 00 ; ................
DDFD  .byte 07 00 07 00 00 00 00 00 07 00 07 00 0D 00 0D 00 ; ................
DE0D  .byte 07 00 07 00 0C 00 7F 00 0C 00 7F 00 00 00 00 00 ; ................
DE1D  .byte 7F 00 0C 00 7F 00 0C 00 0D 00 0C 00 0C 00 0C 00 ; ................
DE2D  .byte 00 00 01 00 87 02 4E 02 02 30 8D 02 30 00 30 8D ; ......N..0..0.0.
DE3D  .byte 00 30 0D 30 8D 0D 30 0C 30 8D 0C 30 02 30 8D 02 ; .0.0..0.0..0.0..
DE4D  .byte 30 00 30 8D 00 30 0D 30 8D 0D 30 0C 30 8D 0C 00 ; 0.0..0.0..0.0...
DE5D  .byte 0C 00 09 00 0C 00 0C 00 0E 00 00 00 01 00 FF 84 ; ................
DE6D  .byte 04 00 82 FF 14 96 00 32 01 8A 06 81 07 7F 00 22 ; .......2......."
DE7D  .byte 00 24 00 25 00 88 81 0A 8A 06 86 12 00 09 00 15 ; .$.%............
DE8D  .byte 00 81 07 09 00 81 0A 14 00 10 00 81 07 14 00 81 ; ................
DE9D  .byte 0A 12 00 81 07 10 00 81 0A 09 12 8D 09 00 09 00 ; ................
DEAD  .byte 12 00 10 00 12 00 09 00 15 00 81 07 09 00 81 0A ; ................
DEBD  .byte 14 00 10 00 81 07 14 00 81 0A 12 00 81 07 10 00 ; ................
DECD  .byte 81 0A 12 00 10 00 12 00 0A 00 10 00 12 00 14 00 ; ................
DEDD  .byte 15 00 12 00 19 00 81 07 12 00 81 0A 17 00 19 00 ; ................
DEED  .byte 81 07 17 00 81 0A 15 0C 8D 15 18 15 00 14 00 10 ; ................
DEFD  .byte 00 12 00 09 00 19 00 81 07 09 00 81 0A 17 00 19 ; ................
DF0D  .byte 00 81 07 17 00 81 0A 22 0C 8A 03 09 00 81 07 22 ; ......."......."
DF1D  .byte 00 81 0A 12 00 81 07 09 00 81 0A 09 00 81 07 12 ; ................
DF2D  .byte 00 81 0A 12 00 81 07 09 00 81 0A 14 00 81 07 12 ; ................
DF3D  .byte 00 81 0A 15 00 81 07 14 00 81 0A 17 00 81 07 15 ; ................
DF4D  .byte 00 81 0A 86 10 00 81 07 17 00 81 0A 17 00 81 07 ; ................
DF5D  .byte 10 00 81 0A 14 00 81 07 17 00 81 0A 17 00 81 07 ; ................
DF6D  .byte 14 00 81 0A 87 02 02 04 10 00 81 07 17 00 81 0A ; ................
DF7D  .byte 17 00 81 07 10 00 81 0A 27 06 81 07 17 06 81 0A ; ........'.......
DF8D  .byte 25 00 81 07 27 00 81 0A 24 12 86 12 00 81 07 12 ; %...'...$.......
DF9D  .byte 00 81 0A 19 00 81 07 12 00 81 0A 15 00 81 07 19 ; ................
DFAD  .byte 00 81 0A 19 00 81 07 15 00 81 0A 87 02 49 04 12 ; .............I..
DFBD  .byte 00 81 07 19 00 81 0A 19 00 81 07 12 00 81 0A 29 ; ...............)
DFCD  .byte 06 81 07 19 06 81 0A 27 00 81 07 29 00 81 0A 25 ; .......'...)...%
DFDD  .byte 12 86 24 00 81 07 20 00 81 0A 20 00 81 07 24 00 ; ..$... ... ...$.
DFED  .byte 81 0A 17 00 81 07 20 00 81 0A 20 00 81 07 17 00 ; ...... ... .....
DFFD  .byte 81 0A 87 03 90 04 24 00 81 07 20 00 81 0A 25 00 ; ......$... ...%.
E00D  .byte 81 07 24 00 81 0A 24 00 81 07 25 00 81 0A 20 00 ; ..$...$...%... .
E01D  .byte 81 07 24 00 81 0A 8A 06 24 00 81 07 20 00 81 0A ; ..$.....$... ...
E02D  .byte 24 00 81 07 24 00 81 0A 27 00 25 00 81 07 27 00 ; $...$...'.%...'.
E03D  .byte 81 0A 24 00 8D 24 18 09 00 0B 00 10 00 11 00 87 ; ..$..$..........
E04D  .byte 02 39 03 8A 0C 15 06 09 00 14 00 12 00 15 00 09 ; .9..............
E05D  .byte 00 12 06 14 00 12 00 15 06 09 00 14 00 12 00 15 ; ................
E06D  .byte 00 09 00 10 06 14 00 12 00 15 06 0A 00 14 00 15 ; ................
E07D  .byte 00 17 00 0A 00 15 06 14 00 12 00 14 06 09 00 11 ; ................
E08D  .byte 00 12 00 14 00 09 00 14 06 12 00 11 00 15 06 09 ; ................
E09D  .byte 00 14 00 12 00 15 00 09 00 12 06 14 00 12 00 15 ; ................
E0AD  .byte 06 09 00 14 00 12 00 15 00 09 00 10 06 14 00 12 ; ................
E0BD  .byte 00 15 06 0A 00 14 00 15 00 17 00 0A 00 15 06 14 ; ................
E0CD  .byte 00 12 00 8A 03 09 00 81 07 12 00 81 0A 0B 00 81 ; ................
E0DD  .byte 07 09 00 81 0A 11 00 81 07 0B 00 81 0A 12 00 81 ; ................
E0ED  .byte 07 11 00 81 0A 0B 00 81 07 12 00 81 0A 11 00 81 ; ................
E0FD  .byte 07 0B 00 81 0A 12 00 81 07 11 00 81 0A 14 00 81 ; ................
E10D  .byte 07 12 00 81 0A 11 00 81 07 14 00 81 0A 12 00 81 ; ................
E11D  .byte 07 11 00 81 0A 14 00 81 07 12 00 81 0A 17 00 81 ; ................
E12D  .byte 07 14 00 81 0A 19 00 81 07 17 00 81 0A 21 00 81 ; .............!..
E13D  .byte 07 19 00 81 0A 22 00 81 07 21 00 81 0A 24 00 81 ; ....."...!...$..
E14D  .byte 07 22 00 81 0A FF 81 09 8A 06 70 00 7F 00 70 00 ; ."........p...p.
E15D  .byte 7F 00 88 86 81 09 70 00 70 00 81 0C 71 00 81 09 ; ......p.p...q...
E16D  .byte 70 00 87 1C 12 06 86 81 0C 71 00 81 09 70 00 87 ; p........q...p..
E17D  .byte 02 25 06 81 0C 71 00 71 00 81 09 70 00 81 0C 71 ; .%...q.q...p...q
E18D  .byte 00 81 09 70 00 81 0C 71 00 71 00 71 00 71 00 71 ; ...p...q.q.q.q.q
E19D  .byte 00 71 00 71 00 FF 00 00 00 00 0A 00 58 01 E2 01 ; .q.q........X...
E1AD  .byte C3 02 00 00 80 01 00 01 00 85 FF 88 83 10 01 04 ; ................
E1BD  .byte 07 00 82 FF 14 96 0A 14 0A 81 0D 8A 06 86 22 00 ; ..............".
E1CD  .byte 20 00 22 00 81 0A 20 00 81 0D 25 00 81 0A 22 00 ;  ."... ...%...".
E1DD  .byte 25 00 81 0D 86 25 00 81 09 25 00 81 0D 87 02 3B ; %....%...%.....;
E1ED  .byte 00 22 00 81 0A 25 00 81 0D 19 00 17 00 15 00 22 ; ."...%........."
E1FD  .byte 00 20 00 22 00 81 0A 20 00 81 0D 25 00 81 0A 22 ; . ."... ...%..."
E20D  .byte 00 25 00 81 0D 86 25 00 81 0A 25 00 81 0D 87 02 ; .%....%...%.....
E21D  .byte 6C 00 22 00 81 0A 25 00 81 0D 18 00 17 00 15 00 ; l."...%.........
E22D  .byte 87 02 24 00 86 19 00 19 00 81 0A 19 00 81 0D 17 ; ..$.............
E23D  .byte 00 81 0A 19 00 81 0D 15 00 81 0A 17 00 81 0D 14 ; ................
E24D  .byte 00 81 0A 15 00 81 0D 12 00 81 0A 14 00 81 0D 14 ; ................
E25D  .byte 00 81 0A 12 00 81 0D 15 00 81 0A 14 00 81 0D 17 ; ................
E26D  .byte 00 1A 00 1A 00 81 0A 1A 00 81 0D 19 00 81 0A 1A ; ................
E27D  .byte 00 81 0D 17 00 81 0A 19 00 81 0D 15 00 81 0A 17 ; ................
E28D  .byte 00 81 0D 14 00 81 0A 15 00 81 0D 15 00 81 0A 14 ; ................
E29D  .byte 00 81 0D 17 00 81 0A 15 00 81 0D 18 00 87 02 8B ; ................
E2AD  .byte 00 21 00 19 00 21 00 22 00 81 0A 21 00 81 0D 1A ; .!...!."...!....
E2BD  .byte 00 22 00 21 00 81 0A 22 00 81 0D 21 00 19 00 24 ; .".!..."...!...$
E2CD  .byte 00 25 00 81 0A 24 00 81 0D 22 00 25 00 24 00 21 ; .%...$...".%.$.!
E2DD  .byte 00 24 00 25 00 81 0A 24 00 81 0D 22 00 25 00 21 ; .$.%...$...".%.!
E2ED  .byte 00 24 00 27 00 2A 00 31 00 3A 00 41 00 44 00 47 ; .$.'.*.1.:.A.D.G
E2FD  .byte 00 FF 88 82 FF 14 96 14 14 0A 81 0E 8A 06 86 02 ; ................
E30D  .byte 0C 02 0C 02 00 02 0C 02 0C 02 00 00 00 02 00 02 ; ................
E31D  .byte 00 05 00 04 00 00 00 0D 0C 0D 0C 0D 00 0D 0C 0D ; ................
E32D  .byte 0C 0D 00 0D 0C 0D 00 0E 00 00 00 01 00 87 02 65 ; ...............e
E33D  .byte 01 86 0C 0C 0C 0C 0C 00 0C 0C 0C 0C 0C 0C 0C 0C ; ................
E34D  .byte 0C 00 0C 00 0C 00 0D 0C 0D 0C 0D 00 0D 0C 0D 0C ; ................
E35D  .byte 0D 0C 0D 00 0D 00 0D 00 0D 00 0D 00 87 02 98 01 ; ................
E36D  .byte 86 0C 00 0C 00 0C 0C 0C 00 0C 0C 0C 0C 0C 0C 0C ; ................
E37D  .byte 0C 0C 00 0C 00 0C 00 87 02 C7 01 FF 88 84 04 00 ; ................
E38D  .byte 82 FF 14 96 00 14 0A 81 0A 8A 06 86 19 00 17 00 ; ................
E39D  .byte 19 00 7F 00 22 0C 7F 00 22 00 7F 00 22 00 7F 00 ; ...."..."..."...
E3AD  .byte 19 00 7F 00 15 00 12 00 12 00 18 00 17 00 18 00 ; ................
E3BD  .byte 7F 00 22 0C 7F 00 22 00 7F 00 22 00 7F 00 18 00 ; .."..."...".....
E3CD  .byte 7F 00 15 00 12 00 12 00 87 02 F2 01 83 18 01 FA ; ................
E3DD  .byte F4 FF 19 30 83 10 01 04 07 00 29 30 83 18 01 FA ; ...0......)0....
E3ED  .byte 0C 00 28 30 83 10 01 04 07 00 22 30 83 18 01 FA ; ..(0......"0....
E3FD  .byte F4 FF 19 30 83 10 01 04 07 00 29 30 83 1C 01 FA ; ...0......)0....
E40D  .byte F6 FF 28 30 83 10 01 04 07 00 32 30 19 00 14 00 ; ..(0......20....
E41D  .byte 19 00 1A 00 81 0A 19 00 81 0D 15 00 1A 00 19 00 ; ................
E42D  .byte 81 0A 1A 00 81 0D 19 00 14 00 21 00 22 00 81 0A ; ..........!."...
E43D  .byte 21 00 81 0D 1A 00 22 00 21 00 19 00 21 00 22 00 ; !.....".!...!.".
E44D  .byte 81 0A 21 00 81 0D 19 00 22 00 19 00 21 00 24 00 ; ..!....."...!.$.
E45D  .byte 27 00 2A 00 27 00 2A 00 31 00 34 00 FF 81 09 8A ; '.*.'.*.1.4.....
E46D  .byte 06 88 86 70 00 70 00 70 00 70 00 81 0C 71 00 81 ; ...p.p.p.p...q..
E47D  .byte 09 70 00 70 00 70 00 70 00 7F 00 70 00 70 00 81 ; .p.p.p.p...p.p..
E48D  .byte 0C 71 00 81 09 70 00 70 00 70 00 87 09 C9 02 70 ; .q...p.p.p.....p
E49D  .byte 00 70 00 70 00 70 00 81 0C 71 00 81 09 70 00 70 ; .p.p.p...q...p.p
E4AD  .byte 00 70 00 70 00 7F 00 70 00 70 00 81 0C 71 00 71 ; .p.p...p.p...q.q
E4BD  .byte 00 71 00 71 00 FF 0A 00 8C 00 DA 00 3C 01 00 00 ; .q.q........<...
E4CD  .byte 80 05 00 04 00 85 FF 81 0C 8A 06 81 0D 82 FA 1E ; ................
E4DD  .byte 96 1E 05 01 83 01 01 FA 1E 00 7F 0C 86 14 06 7F ; ................
E4ED  .byte 06 87 03 27 00 81 0C 82 FF 14 96 02 32 0A 83 10 ; ...'........2...
E4FD  .byte 01 04 06 00 7F 0C 21 18 21 0C 22 12 83 10 01 FA ; ......!.!.".....
E50D  .byte FE FF 1B 1E 83 10 01 04 06 00 21 00 7F 00 21 00 ; ..........!...!.
E51D  .byte 7F 00 21 00 7F 00 19 00 7F 00 17 12 1B 1E 19 00 ; ..!.............
E52D  .byte 7F 00 21 00 7F 00 29 00 7F 00 24 00 7F 0C 28 12 ; ..!...)...$...(.
E53D  .byte 86 29 00 7F 00 8C 8C 8C 87 04 7B 00 88 81 00 7F ; .)........{.....
E54D  .byte 00 FF 82 FF 1E 96 0A 0A 0A 81 0D 8A 0C 7F 30 0C ; ..............0.
E55D  .byte 00 09 00 04 00 04 00 07 00 07 06 06 00 07 06 06 ; ................
E56D  .byte 00 0C 00 09 00 04 00 04 00 02 00 02 06 01 00 02 ; ................
E57D  .byte 06 01 00 0C 00 09 00 04 00 01 00 0C 06 08 12 09 ; ................
E58D  .byte 00 82 FF 1E 96 01 0A 0A 0C 30 88 81 00 7F 00 FF ; .........0......
E59D  .byte 82 FF 14 82 02 32 0A 81 0B 8A 06 7F 30 83 10 01 ; .....2......0...
E5AD  .byte 04 06 00 7F 0C 19 18 19 0C 1B 12 83 10 01 FA FE ; ................
E5BD  .byte FF 17 1E 83 10 01 04 06 00 19 00 7F 00 19 00 7F ; ................
E5CD  .byte 00 19 00 7F 00 14 00 7F 00 12 12 17 1E 11 00 7F ; ................
E5DD  .byte 00 19 00 7F 00 24 00 7F 00 19 00 7F 0C 22 12 21 ; .....$.......".!
E5ED  .byte 00 7F 00 82 FF 1E 96 01 0A 0A 09 30 88 81 00 7F ; ...........0....
E5FD  .byte 00 FF 81 09 8A 06 8A 0C 70 00 71 00 71 00 71 00 ; ........p.q.q.q.
E60D  .byte 86 81 09 70 00 81 0B 71 00 81 09 70 06 70 06 81 ; ...p...q...p.p..
E61D  .byte 0B 71 00 81 09 70 00 81 0B 71 06 81 09 70 00 70 ; .q...p...q...p.p
E62D  .byte 06 81 0B 71 00 87 03 4B 01 88 81 00 7F 00 FF 0A ; ...q...K........
E63D  .byte 00 4D 00 8B 00 C2 00 00 00 80 01 00 01 00 85 FF ; .M..............
E64D  .byte 81 0D 82 FF 14 78 14 23 01 81 0C 07 08 0B 07 12 ; .....x.#........
E65D  .byte 07 15 06 17 06 1B 06 22 06 25 06 27 05 2B 05 32 ; .......".%.'.+.2
E66D  .byte 05 35 05 37 05 3B 05 42 05 45 05 86 47 05 7F 05 ; .5.7.;.B.E..G...
E67D  .byte 8C 8C 87 06 3D 00 88 81 00 7F 00 FF 81 0A 82 FF ; ....=...........
E68D  .byte 14 78 14 23 01 7F 0C 81 0C 07 08 0B 07 12 07 15 ; .x.#............
E69D  .byte 06 17 06 1B 06 22 06 25 06 27 05 2B 05 32 05 35 ; .....".%.'.+.2.5
E6AD  .byte 05 37 05 3B 05 42 05 45 05 86 47 05 7F 05 8C 8C ; .7.;.B.E..G.....
E6BD  .byte 87 05 7B 00 88 81 00 7F 00 FF 81 07 82 FF 14 78 ; ..{............x
E6CD  .byte 14 23 01 7F 18 81 0C 07 08 0B 07 12 07 15 06 17 ; .#..............
E6DD  .byte 06 1B 06 22 06 25 06 27 05 2B 05 32 05 35 05 37 ; ...".%.'.+.2.5.7
E6ED  .byte 05 3B 05 42 05 45 05 47 05 7F 0A 88 81 00 7F 00 ; .;.B.E.G........
E6FD  .byte FF 88 81 00 7F 00 FF 0A 00 A4 00 09 01 60 01 00 ; .............`..
E70D  .byte 00 80 01 00 01 00 85 FF 88 8A 06 86 81 0C 82 FF ; ................
E71D  .byte 14 96 00 32 0A 83 10 01 04 06 00 7F 0C 23 18 23 ; ...2.........#.#
E72D  .byte 0C 24 12 83 10 01 FA FE FF 21 1E 83 10 01 04 06 ; .$.......!......
E73D  .byte 00 23 00 7F 00 23 00 7F 00 23 00 7F 00 1B 00 7F ; .#...#...#......
E74D  .byte 00 19 12 21 1E 87 02 15 00 86 7F 00 09 12 0B 00 ; ...!............
E75D  .byte 7F 00 81 07 0B 00 7F 00 81 0D 87 02 53 00 8C 8C ; ............S...
E76D  .byte 82 FF 00 96 02 32 0A 8A 04 1B 00 21 00 23 00 24 ; .....2.....!.#.$
E77D  .byte 00 26 00 28 00 23 00 24 00 26 00 28 00 29 00 2B ; .&.(.#.$.&.(.).+
E78D  .byte 00 24 00 26 00 28 00 29 00 2B 00 31 00 8B 26 00 ; .$.&.(.).+.1..&.
E79D  .byte 28 00 2A 00 2B 00 31 00 33 00 FF 82 FF 14 96 00 ; (.*.+.1.3.......
E7AD  .byte 0A 0A 81 0E 88 86 8A 0C 0E 00 0B 00 06 00 06 00 ; ................
E7BD  .byte 09 00 09 06 08 00 09 06 08 00 0E 00 0B 00 06 00 ; ................
E7CD  .byte 06 00 04 00 04 06 03 00 04 06 03 00 87 02 AF 00 ; ................
E7DD  .byte 86 0C 06 0C 12 0E 00 0E 00 87 02 DA 00 0C 06 01 ; ................
E7ED  .byte 12 03 00 05 00 8A 04 01 00 03 00 04 00 06 00 08 ; ................
E7FD  .byte 00 09 00 03 00 04 00 06 00 08 00 09 00 10 00 FF ; ................
E80D  .byte 82 FF 14 82 00 32 0A 88 86 8A 06 81 0A 83 10 01 ; .....2..........
E81D  .byte 04 06 00 7F 0C 1B 18 1B 0C 21 12 83 10 01 FA FE ; .........!......
E82D  .byte FF 19 1E 83 10 01 04 06 00 1B 00 7F 00 1B 00 7F ; ................
E83D  .byte 00 1B 00 7F 00 16 00 7F 00 14 12 19 1E 87 02 12 ; ................
E84D  .byte 01 86 7F 00 11 12 13 00 7F 00 81 06 13 00 7F 00 ; ................
E85D  .byte 81 0B 87 04 4B 01 FF 88 8A 0C 86 81 09 70 00 81 ; ....K........p..
E86D  .byte 0B 71 00 81 09 70 06 70 06 81 0B 71 00 81 09 70 ; .q...p.p...q...p
E87D  .byte 00 81 0B 71 06 81 09 70 00 70 06 81 0B 71 00 87 ; ...q...p.p...q..
E88D  .byte 04 64 01 8A 06 86 81 09 70 00 81 0B 71 0C 81 09 ; .d......p...q...
E89D  .byte 70 00 81 0C 71 0C 81 09 70 00 81 0B 71 00 87 04 ; p...q...p...q...
E8AD  .byte 8F 01 FF 00 00 00 00 0A 00 44 00 6F 00 A2 00 00 ; .........D.o....
E8BD  .byte 00 80 04 00 03 00 85 FF 81 0C 8A 0C 81 0C 82 64 ; ...............d
E8CD  .byte 14 96 00 32 0A 83 10 01 04 06 00 19 00 22 00 21 ; ...2.........".!
E8DD  .byte 00 19 00 22 00 21 00 19 00 22 00 21 18 32 06 31 ; ...".!...".!.2.1
E8ED  .byte 06 29 06 2B 60 88 81 00 7F 00 FF 82 FF 1E 96 00 ; .).+`...........
E8FD  .byte 0A 0A 81 0D 8A 0C 09 06 7F 06 0C 00 0E 00 01 00 ; ................
E90D  .byte 02 00 04 00 06 00 07 00 09 18 0C 00 7F 06 0E 60 ; ...............`
E91D  .byte 88 81 00 7F 00 FF 82 64 14 82 00 32 0A 81 0B 8A ; .......d...2....
E92D  .byte 0C 83 10 01 04 06 00 19 06 7F 06 09 00 0B 00 11 ; ................
E93D  .byte 00 12 00 14 00 16 00 17 00 19 18 21 06 1B 06 21 ; ...........!...!
E94D  .byte 06 24 60 88 81 00 7F 00 FF 81 09 8A 06 86 81 0B ; .$`.............
E95D  .byte 71 00 71 00 81 09 70 00 7F 00 70 00 7F 00 87 02 ; q.q...p...p.....
E96D  .byte A7 00 81 09 70 00 70 00 81 0C 71 00 71 00 71 00 ; ....p.p...q.q.q.
E97D  .byte 71 00 71 00 81 09 70 00 70 00 81 0C 71 00 88 81 ; q.q...p.p...q...
E98D  .byte 00 7F 00 FF 0A 00 64 00 9B 00 E9 00 00 00 80 01 ; ......d.........
E99D  .byte 00 01 00 82 FF 14 96 0A 14 0A 85 FF 81 0F 8A 01 ; ................
E9AD  .byte 86 00 00 0C 00 0E 00 8C 8C 40 00 0C 00 7F 00 8C ; .........@......
E9BD  .byte 8C 87 03 1D 00 8A 06 81 0C 30 18 30 0C 31 12 83 ; .........0.0.1..
E9CD  .byte 0B 01 FA FE FF 2A 1E 83 10 01 04 04 00 30 00 7F ; .....*.......0..
E9DD  .byte 00 30 00 7F 00 30 00 7F 00 28 00 7F 00 86 26 00 ; .0...0...(....&.
E9ED  .byte 7F 00 8C 87 08 5A 00 FF 82 FF 1E 96 0A 14 0A 81 ; .....Z..........
E9FD  .byte 0D 8A 0C 7F 06 09 00 19 00 14 00 14 00 17 00 17 ; ................
EA0D  .byte 06 16 00 17 06 16 00 09 00 19 00 14 00 14 00 82 ; ................
EA1D  .byte FF 10 96 01 14 0A 83 10 01 05 0C 00 13 60 FF 82 ; .............`..
EA2D  .byte FF 14 82 0A 14 0A 81 0E 8A 0C 83 01 01 FA 05 00 ; ................
EA3D  .byte 7F 06 4B 0C 8A 06 81 0B 83 24 01 01 01 00 28 18 ; ..K......$....(.
EA4D  .byte 28 0C 29 12 83 0B 01 FA FE FF 26 1E 83 10 01 04 ; (.).......&.....
EA5D  .byte 04 00 28 00 7F 00 28 00 7F 00 28 00 7F 00 24 00 ; ..(...(...(...$.
EA6D  .byte 7F 00 86 22 00 7F 00 8C 87 08 DF 00 FF 81 09 8A ; ..."............
EA7D  .byte 06 7F 06 8A 0C 70 00 81 0B 71 00 81 09 70 06 70 ; .....p...q...p.p
EA8D  .byte 06 81 0B 71 00 81 09 70 00 81 0B 71 06 81 09 70 ; ...q...p...q...p
EA9D  .byte 00 70 06 81 0B 71 00 81 09 70 00 81 0B 71 00 81 ; .p...q...p...q..
EAAD  .byte 09 70 06 70 06 81 0B 71 00 81 09 86 71 04 87 18 ; .p.p...q....q...
EABD  .byte 28 01 FF 0A 00 4B 01 B2 01 2A 02 00 00 80 04 00 ; (....K...*......
EACD  .byte 05 00 85 FF 88 83 10 01 04 07 00 82 FF 14 96 00 ; ................
EADD  .byte 14 0A 81 0D 8A 06 7F 0C 25 00 7F 00 81 09 25 00 ; ........%.....%.
EAED  .byte 81 0D 7F 00 25 00 7F 00 81 09 25 00 81 0D 22 00 ; ....%.....%...".
EAFD  .byte 7F 00 81 09 22 00 81 0D 25 0C 22 00 20 0C 22 00 ; ...."...%.". .".
EB0D  .byte 7F 00 81 09 22 00 81 0D 19 18 7F 0C 25 00 7F 00 ; ....".......%...
EB1D  .byte 81 09 25 00 81 0D 7F 00 25 00 7F 00 81 09 25 00 ; ..%.....%.....%.
EB2D  .byte 81 0D 22 00 7F 00 81 09 22 00 81 0D 25 0C 22 00 ; .."....."...%.".
EB3D  .byte 20 0C 22 00 7F 00 81 09 22 00 81 0D 29 18 7F 0C ;  ."....."...)...
EB4D  .byte 2A 00 7F 00 81 09 2A 00 81 0D 7F 00 2A 00 7F 00 ; *.....*.....*...
EB5D  .byte 81 09 2A 00 81 0D 27 00 7F 00 81 09 27 00 81 0D ; ..*...'.....'...
EB6D  .byte 2A 0C 27 00 25 0C 27 00 7F 00 81 09 27 00 81 0D ; *.'.%.'.....'...
EB7D  .byte 22 18 7F 0C 2A 00 7F 00 81 09 2A 00 81 0D 7F 00 ; "...*.....*.....
EB8D  .byte 2A 00 7F 00 81 09 2A 00 81 0D 27 00 7F 00 81 09 ; *.....*...'.....
EB9D  .byte 27 00 81 0D 2A 0C 27 00 25 0C 27 00 7F 00 81 09 ; '...*.'.%.'.....
EBAD  .byte 27 00 81 0D 32 18 86 82 FF 14 96 00 14 0A 8B 29 ; '...2..........)
EBBD  .byte 00 8C 7F 00 29 00 82 FF 00 96 00 14 0A 81 08 83 ; ....)...........
EBCD  .byte 0C 01 F0 EA FF 09 12 83 0C 01 F0 16 00 19 12 83 ; ................
EBDD  .byte 0C 01 F0 EA FF 09 12 83 0C 01 F0 16 00 19 12 83 ; ................
EBED  .byte 0C 01 F0 EA FF 09 12 83 0C 01 F0 16 00 19 12 83 ; ................
EBFD  .byte 0C 01 F0 EA FF 09 12 81 0D 87 02 F4 00 FF 88 82 ; ................
EC0D  .byte FF 14 96 01 14 0A 81 0E 8A 12 86 02 00 02 00 00 ; ................
EC1D  .byte 00 0E 00 0D 00 0E 00 00 00 01 00 87 02 58 01 86 ; .............X..
EC2D  .byte 07 00 07 00 06 00 05 00 04 00 05 00 06 00 07 00 ; ................
EC3D  .byte 87 02 6D 01 8B 09 0C 8C 09 06 0C 12 0C 12 0C 12 ; ..m.............
EC4D  .byte 0C 12 0C 12 0C 12 0C 12 8B 09 0C 8C 09 06 0C 12 ; ................
EC5D  .byte 0C 12 0C 12 0C 0C 0D 06 7F 0C 0E 06 7F 0C 00 06 ; ................
EC6D  .byte 7F 0C 01 06 FF 88 84 04 00 82 FF 14 82 00 14 0A ; ................
EC7D  .byte 81 0A 8A 06 86 7F 0C 22 00 86 19 00 7F 0C 87 03 ; ......."........
EC8D  .byte C7 01 19 18 8D 19 0C 8D 19 24 87 02 C2 01 86 7F ; .........$......
EC9D  .byte 0C 22 00 7F 12 22 00 7F 0C 22 00 7F 0C 22 0C 22 ; ."..."..."..."."
ECAD  .byte 00 22 0C 22 00 7F 0C 22 18 87 02 DC 01 8B 19 00 ; ."."..."........
ECBD  .byte 7F 00 19 00 86 09 00 7F 0C 87 07 02 02 19 00 7F ; ................
ECCD  .byte 00 19 00 86 09 00 7F 0C 87 03 11 02 09 0C 0A 00 ; ................
ECDD  .byte 7F 0C 0B 00 7F 0C 10 00 7F 0C 11 00 FF 81 09 8A ; ................
ECED  .byte 06 88 86 81 09 70 00 7F 00 70 00 81 0B 71 0C 81 ; .....p...p...q..
ECFD  .byte 09 70 00 87 10 30 02 81 0C 71 00 7F 00 71 00 81 ; .p...0...q...q..
ED0D  .byte 09 70 12 70 12 70 12 86 70 00 7F 00 70 00 81 0B ; .p.p.p..p...p...
ED1D  .byte 71 0C 81 09 70 00 87 02 55 02 81 0C 71 00 7F 00 ; q...p...U...q...
ED2D  .byte 71 00 81 09 70 12 70 12 70 12 70 00 7F 00 70 00 ; q...p.p.p.p...p.
ED3D  .byte 81 0B 71 0C 81 09 70 00 81 0B 71 00 71 00 71 00 ; ..q...p...q.q.q.
ED4D  .byte 71 00 71 00 71 00 FF 0A 00 64 01 54 02 92 03 00 ; q.q.q....d.T....
ED5D  .byte 00 80 02 00 01 00 85 FF 7F 30 83 10 01 04 07 00 ; .........0......
ED6D  .byte 82 FF 14 96 00 14 0A 81 0D 8A 06 86 21 00 7F 00 ; ............!...
ED7D  .byte 19 0C 8D 19 24 7F 00 22 00 7F 00 22 00 22 00 24 ; ....$.."...".".$
ED8D  .byte 00 21 00 7F 00 19 0C 8D 19 24 7F 00 16 00 7F 00 ; .!.......$......
ED9D  .byte 19 00 19 00 1B 00 87 03 25 00 21 00 7F 00 19 0C ; ........%.!.....
EDAD  .byte 8D 19 24 7F 00 22 00 7F 00 22 00 22 00 24 00 21 ; ..$.."...".".$.!
EDBD  .byte 00 7F 00 19 1E 29 00 8D 29 30 7F 18 20 00 19 0C ; .....)..)0.. ...
EDCD  .byte 20 00 1B 0C 20 00 1B 0C 17 12 7F 12 19 00 24 00 ;  ... .........$.
EDDD  .byte 22 0C 20 00 1B 0C 20 00 1B 0C 17 12 7F 18 20 00 ; ". ... ....... .
EDED  .byte 19 0C 20 00 1B 0C 20 00 1B 0C 17 12 7F 12 19 00 ; .. ... .........
EDFD  .byte 19 00 15 0C 19 00 17 0C 19 00 17 0C 10 0C 14 00 ; ................
EE0D  .byte 12 30 8D 12 1E 10 00 12 00 14 0C 8D 14 48 10 00 ; .0...........H..
EE1D  .byte 19 00 13 48 8D 13 0C 10 00 13 00 12 30 14 18 1B ; ...H........0...
EE2D  .byte 0C 21 00 22 00 24 00 86 21 00 7F 00 19 0C 8D 19 ; .!.".$..!.......
EE3D  .byte 24 7F 00 22 00 7F 00 22 00 22 00 24 00 21 00 7F ; $.."...".".$.!..
EE4D  .byte 00 19 0C 8D 19 24 7F 00 16 00 7F 00 19 00 19 00 ; .....$..........
EE5D  .byte 1B 00 87 03 E1 00 21 00 7F 00 19 0C 8D 19 24 7F ; ......!.......$.
EE6D  .byte 00 22 00 7F 00 22 00 22 00 24 00 21 00 7F 00 19 ; ."...".".$.!....
EE7D  .byte 1E 29 0C 29 00 26 00 7F 00 29 00 2B 00 7F 00 29 ; .).).&...).+...)
EE8D  .byte 00 8D 29 30 8D 29 30 8D 29 30 8D 29 30 7F 30 7F ; ..)0.)0.)0.)0.0.
EE9D  .byte 30 7F 00 22 00 7F 0C 22 00 7F 00 22 00 21 00 8D ; 0.."..."...".!..
EEAD  .byte 21 36 8D 21 30 81 00 88 7F 00 FF 7F 30 82 FF 14 ; !6.!0.......0...
EEBD  .byte 96 14 14 0A 81 0E 8A 06 86 0C 0C 0C 0C 01 00 11 ; ................
EECD  .byte 00 01 00 02 0C 02 00 12 00 02 00 04 00 04 00 14 ; ................
EEDD  .byte 00 04 00 87 08 72 01 86 05 00 05 00 10 00 05 0C ; .....r..........
EEED  .byte 05 00 00 00 05 00 04 00 04 00 10 00 04 0C 00 00 ; ................
EEFD  .byte 02 00 04 00 87 04 91 01 0D 0C 0D 00 0C 0C 0C 00 ; ................
EF0D  .byte 0D 0C 0D 00 00 0C 00 00 02 0C 04 0C 0C 0C 0C 00 ; ................
EF1D  .byte 0E 0C 0E 00 00 0C 00 00 02 0C 02 00 04 0C 00 0C ; ................
EF2D  .byte 08 0C 08 00 07 0C 07 00 05 0C 05 00 03 0C 03 00 ; ................
EF3D  .byte 02 0C 00 0C 07 0C 07 00 02 0C 02 00 07 0C 07 00 ; ................
EF4D  .byte 8C 8C 8C 8C 04 00 04 00 8B 04 00 8B 04 00 8B 04 ; ................
EF5D  .byte 00 8B 04 00 04 00 86 0C 0C 0C 0C 01 00 11 00 01 ; ................
EF6D  .byte 00 02 0C 02 00 12 00 02 00 04 00 04 00 14 00 04 ; ................
EF7D  .byte 00 87 0B 10 02 05 00 05 00 15 00 05 00 07 00 17 ; ................
EF8D  .byte 00 07 00 09 0C 04 0C 01 00 82 FF 14 96 14 00 0A ; ................
EF9D  .byte 0C 30 8D 0C 18 81 00 88 7F 00 FF 7F 30 84 04 00 ; .0..........0...
EFAD  .byte 82 FF 14 82 00 14 0A 8A 03 86 81 0B 39 00 81 07 ; ............9...
EFBD  .byte 34 00 81 0B 38 00 81 07 39 00 81 0B 36 00 81 07 ; 4...8...9...6...
EFCD  .byte 38 00 81 0B 34 00 81 07 36 00 87 20 63 02 8A 06 ; 8...4...6.. c...
EFDD  .byte 81 0B 86 19 00 15 00 10 00 09 00 09 00 10 00 15 ; ................
EFED  .byte 00 19 00 17 00 14 00 10 00 07 00 07 00 10 00 14 ; ................
EFFD  .byte 00 17 00 87 03 8C 02 19 00 15 00 10 00 09 00 09 ; ................
F00D  .byte 00 10 00 15 00 19 00 10 00 14 00 17 00 1B 00 20 ; ............... 
F01D  .byte 00 24 00 27 00 2B 00 1A 00 22 00 25 00 29 00 2A ; .$.'.+...".%.).*
F02D  .byte 00 32 00 35 00 39 00 3A 00 39 00 35 00 32 00 2A ; .2.5.9.:.9.5.2.*
F03D  .byte 00 29 00 25 00 22 00 09 00 0B 00 10 00 14 00 19 ; .).%."..........
F04D  .byte 00 1B 00 20 00 24 00 29 00 24 00 20 00 1B 00 19 ; ... .$.).$. ....
F05D  .byte 00 14 00 10 00 0B 00 08 00 0A 00 10 00 13 00 18 ; ................
F06D  .byte 00 1A 00 20 00 23 00 28 00 2A 00 30 00 33 00 38 ; ... .#.(.*.0.3.8
F07D  .byte 00 33 00 30 00 27 00 07 00 09 00 0B 00 12 00 17 ; .3.0.'..........
F08D  .byte 00 19 00 1B 00 17 00 14 00 08 00 09 00 0B 00 11 ; ................
F09D  .byte 00 12 00 14 00 16 00 8A 03 86 81 0B 39 00 81 07 ; ............9...
F0AD  .byte 34 00 81 0B 38 00 81 07 39 00 81 0B 36 00 81 07 ; 4...8...9...6...
F0BD  .byte 38 00 81 0B 34 00 81 07 36 00 87 2C 53 03 81 0B ; 8...4...6..,S...
F0CD  .byte 8A 06 7F 00 19 00 7F 0C 19 00 7F 00 19 00 14 36 ; ...............6
F0DD  .byte 8D 14 30 81 00 88 7F 00 FF 8A 06 81 09 70 0C 70 ; ..0..........p.p
F0ED  .byte 0C 70 0C 70 0C 86 86 70 00 70 00 81 0D 71 00 81 ; .p.p...p.p...q..
F0FD  .byte 09 70 00 87 0F A0 03 70 00 81 0D 71 00 71 00 71 ; .p.....p...q.q.q
F10D  .byte 00 87 06 9F 03 86 81 09 70 00 70 00 81 0D 71 00 ; ........p.p...q.
F11D  .byte 81 09 70 00 87 0F BF 03 81 00 88 7F 00 FF 00 0A ; ..p.............
F12D  .byte 00 79 02 59 03 9F 04 00 00 80 07 00 06 00 85 FF ; .y.Y............
F13D  .byte 88 83 10 01 04 07 00 82 FF 14 96 00 14 0A 81 0D ; ................
F14D  .byte 8A 06 86 29 30 7F 0C 26 00 81 09 29 00 81 0D 29 ; ...)0..&...)...)
F15D  .byte 00 81 09 26 00 81 0D 2B 00 81 09 29 00 81 0D 22 ; ...&...+...)..."
F16D  .byte 00 7F 00 22 00 7F 00 22 00 1B 00 7F 00 22 00 7F ; ..."..."....."..
F17D  .byte 00 7F 00 1B 00 7F 00 22 00 7F 00 24 00 7F 00 29 ; ......."...$...)
F18D  .byte 30 7F 0C 26 00 81 09 29 00 81 0D 29 00 81 09 26 ; 0..&...)...)...&
F19D  .byte 00 81 0D 2B 00 81 09 29 00 81 0D 32 00 7F 12 2B ; ...+...)...2...+
F1AD  .byte 00 7F 0C 26 00 29 00 7F 0C 2B 00 7F 0C 26 00 7F ; ...&.)...+...&..
F1BD  .byte 00 29 30 7F 0C 26 00 81 09 29 00 81 0D 29 00 81 ; .)0..&...)...)..
F1CD  .byte 09 26 00 81 0D 2B 00 81 09 29 00 81 0D 22 00 7F ; .&...+...)..."..
F1DD  .byte 00 22 00 7F 00 22 00 1B 00 7F 00 22 00 7F 00 7F ; ."..."....."....
F1ED  .byte 00 1B 00 7F 00 22 00 7F 00 24 00 7F 00 29 30 7F ; ....."...$...)0.
F1FD  .byte 0C 26 00 81 09 29 00 81 0D 29 00 81 09 26 00 81 ; .&...)...)...&..
F20D  .byte 0D 2B 00 81 09 29 00 81 0D 32 00 7F 12 2B 00 7F ; .+...)...2...+..
F21D  .byte 0C 35 00 7F 00 35 00 34 00 7F 00 32 00 2B 0C 7F ; .5...5.4...2.+..
F22D  .byte 00 7F 0C 26 00 81 09 26 00 81 0D 27 00 81 09 26 ; ...&...&...'...&
F23D  .byte 00 81 0D 26 00 81 09 27 00 81 0D 2A 00 81 09 26 ; ...&...'...*...&
F24D  .byte 00 81 0D 2A 00 81 09 26 00 81 0D 2B 00 81 09 2A ; ...*...&...+...*
F25D  .byte 00 81 0D 31 00 81 09 2B 00 81 0D 32 00 7F 12 31 ; ...1...+...2...1
F26D  .byte 00 7F 0C 2B 00 7F 00 2B 00 7F 00 31 00 2B 00 7F ; ...+...+...1.+..
F27D  .byte 00 26 00 7F 00 7F 0C 24 00 7F 00 26 00 7F 00 24 ; .&.....$...&...$
F28D  .byte 00 7F 00 2B 00 7F 00 2B 00 7F 00 31 00 7F 00 2B ; ...+...+...1...+
F29D  .byte 00 7F 00 29 30 7F 0C 29 00 81 09 29 00 81 0D 2B ; ...)0..)...)...+
F2AD  .byte 00 81 09 29 00 81 0D 31 00 81 09 2B 00 81 0D 87 ; ...)...1...+....
F2BD  .byte 02 24 00 86 82 FF 14 96 0A 14 0A 81 0D 32 00 7F ; .$...........2..
F2CD  .byte 00 32 00 7F 00 32 00 7F 18 82 FF 00 96 00 14 0A ; .2...2..........
F2DD  .byte 81 09 86 42 01 44 01 87 02 B4 01 7F 02 7F 00 86 ; ...B.D..........
F2ED  .byte 42 01 44 01 87 02 C1 01 7F 02 86 42 01 44 01 87 ; B.D........B.D..
F2FD  .byte 02 CC 01 7F 02 7F 00 86 42 01 44 01 87 06 D9 01 ; ........B.D.....
F30D  .byte 82 FF 14 96 0A 14 0A 81 0D 32 00 7F 00 32 00 7F ; .........2...2..
F31D  .byte 00 32 00 82 FF 00 96 00 14 0A 81 09 86 39 01 3B ; .2...........9.;
F32D  .byte 01 87 02 FE 01 7F 02 86 42 01 44 01 87 02 09 02 ; ........B.D.....
F33D  .byte 7F 02 86 42 01 44 01 87 06 14 02 86 39 01 3B 01 ; ...B.D......9.;.
F34D  .byte 87 02 1D 02 7F 02 86 42 01 44 01 87 02 28 02 7F ; .......B.D...(..
F35D  .byte 02 86 86 39 01 3B 01 87 02 34 02 7F 02 87 05 33 ; ...9.;...4.....3
F36D  .byte 02 87 03 95 01 82 FF 14 96 0A 14 0A 81 0D 32 00 ; ..............2.
F37D  .byte 7F 00 32 00 7F 00 32 00 7F 30 1B 00 1A 00 19 00 ; ..2...2..0......
F38D  .byte 7F 0C 82 FF 1E 96 14 0A 0A 83 01 01 FA 3C 00 2B ; .............<.+
F39D  .byte 0C 22 0C 12 0C 7F 30 FF 88 82 FF 12 96 10 14 0A ; ."....0.........
F3AD  .byte 81 0E 8A 06 86 86 02 0C 12 0C 02 0C 12 00 02 0C ; ................
F3BD  .byte 02 00 12 00 02 00 02 0C 12 0C 07 0C 17 0C 07 0C ; ................
F3CD  .byte 17 00 07 0C 07 00 17 00 07 00 07 0C 17 0C 87 04 ; ................
F3DD  .byte 87 02 06 0C 16 0C 06 0C 16 00 06 0C 06 00 16 0C ; ................
F3ED  .byte 06 0C 16 00 06 00 0E 0C 0B 0C 0E 00 0E 00 0B 00 ; ................
F3FD  .byte 0E 0C 0E 00 0B 00 0E 00 0E 0C 0B 0C 04 0C 14 0C ; ................
F40D  .byte 04 00 04 00 14 00 04 0C 04 00 14 00 04 00 04 0C ; ................
F41D  .byte 14 0C 09 0C 19 0C 0A 0C 1A 0C 0B 0C 1B 0C 11 0C ; ................
F42D  .byte 21 0C 87 02 86 02 86 02 0C 12 0C 02 0C 12 00 02 ; !...............
F43D  .byte 0C 02 00 12 00 02 00 02 0C 12 0C 07 0C 17 0C 07 ; ................
F44D  .byte 0C 17 00 07 0C 07 00 17 00 07 00 07 0C 17 0C 87 ; ................
F45D  .byte 03 08 03 02 0C 12 0C 02 0C 12 00 02 0C 02 00 12 ; ................
F46D  .byte 00 02 00 02 0C 12 0C 09 0C 0C 0C 0C 0C 0C 0C 0C ; ................
F47D  .byte 0C 0E 0C 00 0C 01 0C FF 88 82 FF 14 82 00 14 0A ; ................
F48D  .byte 81 0A 8A 06 86 86 26 00 81 06 26 00 81 0A 22 00 ; ......&...&...".
F49D  .byte 81 06 26 00 81 0A 32 00 81 06 22 00 81 0A 31 00 ; ..&...2..."...1.
F4AD  .byte 32 00 81 06 31 00 81 0A 22 00 81 06 32 00 81 0A ; 2...1..."...2...
F4BD  .byte 22 00 26 00 81 06 22 00 81 0A 26 00 81 06 26 00 ; ".&..."...&...&.
F4CD  .byte 81 0A 1B 00 81 06 26 00 81 0A 1B 00 81 06 1B 00 ; ......&.........
F4DD  .byte 81 0A 2B 00 81 06 1B 00 81 0A 29 00 27 00 81 06 ; ..+.......).'...
F4ED  .byte 29 00 81 0A 27 00 81 06 27 00 81 0A 27 00 1B 00 ; )...'...'...'...
F4FD  .byte 81 06 27 00 81 0A 1B 00 81 06 1B 00 81 0A 87 04 ; ..'.............
F50D  .byte 67 03 21 18 1A 18 1B 18 21 18 1B 00 7F 00 1B 00 ; g.!.....!.......
F51D  .byte 7F 00 26 00 27 00 26 00 22 00 7F 00 22 00 1B 00 ; ..&.'.&."..."...
F52D  .byte 7F 00 16 00 7F 00 12 00 7F 00 24 18 14 18 16 18 ; ..........$.....
F53D  .byte 18 18 19 00 7F 00 19 00 7F 00 17 00 7F 00 17 00 ; ................
F54D  .byte 7F 00 16 00 17 00 16 00 14 00 7F 00 14 00 7F 00 ; ................
F55D  .byte 14 00 87 02 66 03 86 81 0A 12 00 86 16 00 19 00 ; ....f...........
F56D  .byte 21 00 32 00 8C 8C 87 03 3D 04 16 00 19 00 21 00 ; !.2.....=.....!.
F57D  .byte 81 0A 17 00 86 1B 00 22 00 26 00 27 00 8C 8C 87 ; .......".&.'....
F58D  .byte 03 56 04 1B 00 22 00 26 00 87 03 38 04 81 0A 12 ; .V...".&...8....
F59D  .byte 00 86 16 00 19 00 21 00 32 00 8C 8C 87 03 73 04 ; ......!.2.....s.
F5AD  .byte 16 00 19 00 21 00 19 00 86 21 00 24 00 27 00 29 ; ....!....!.$.'.)
F5BD  .byte 00 8C 8C 87 03 8A 04 24 00 27 00 29 00 FF 81 09 ; .......$.'.)....
F5CD  .byte 8A 06 88 86 81 09 70 00 70 00 81 0C 71 00 81 09 ; ......p.p...q...
F5DD  .byte 70 00 87 7C A5 04 81 0D 71 00 7F 00 81 09 70 00 ; p..|....q.....p.
F5ED  .byte 7F 00 70 00 7F 00 70 00 7F 00 70 00 7F 00 81 0C ; ..p...p...p.....
F5FD  .byte 71 00 7F 00 71 00 71 00 71 00 7F 00 FF 00 00 0A ; q...q.q.q.......
F60D  .byte 00 E0 01 9D 02 51 03 00 00 80 01 00 01 00 85 FF ; .....Q..........
F61D  .byte 83 10 01 04 07 00 82 FF 14 96 00 14 0A 81 0D 8A ; ................
F62D  .byte 06 88 27 48 8D 27 24 23 0C 25 00 7F 00 81 09 25 ; ..'H.'$#.%.....%
F63D  .byte 00 81 0D 23 00 7F 00 81 09 23 00 81 0D 20 00 8D ; ...#.....#... ..
F64D  .byte 20 48 23 00 7F 00 81 09 23 00 81 0D 25 0C 26 00 ;  H#.....#...%.&.
F65D  .byte 7F 00 81 09 26 00 81 0D 27 00 8D 27 48 8D 27 24 ; ....&...'..'H.'$
F66D  .byte 23 0C 25 00 7F 00 81 09 25 00 81 0D 23 00 7F 00 ; #.%.....%...#...
F67D  .byte 81 09 23 00 81 0D 20 00 8D 20 48 23 00 7F 00 81 ; ..#... .. H#....
F68D  .byte 09 23 00 81 0D 25 0C 23 00 7F 0C 20 00 7F 00 81 ; .#...%.#... ....
F69D  .byte 09 20 00 81 0D 20 00 1A 12 17 0C 23 00 7F 00 81 ; . ... .....#....
F6AD  .byte 09 23 00 81 0D 25 12 25 00 23 00 7F 00 81 09 23 ; .#...%.%.#.....#
F6BD  .byte 00 81 0D 17 0C 1A 00 7F 00 81 09 1A 00 81 0D 20 ; ............... 
F6CD  .byte 00 8D 20 48 7F 00 81 09 20 00 81 0D 86 27 00 7F ; .. H.... ....'..
F6DD  .byte 00 81 09 27 00 81 0D 87 03 CE 00 20 00 7F 00 81 ; ...'....... ....
F6ED  .byte 09 20 00 81 0D 20 00 1A 12 17 0C 23 00 7F 00 81 ; . ... .....#....
F6FD  .byte 09 23 00 81 0D 25 12 25 00 23 00 7F 00 81 09 23 ; .#...%.%.#.....#
F70D  .byte 00 81 0D 17 0C 1A 00 7F 00 81 09 1A 00 81 0D 20 ; ............... 
F71D  .byte 00 8D 20 48 7F 00 81 09 20 00 81 0D 86 27 00 7F ; .. H.... ....'..
F72D  .byte 00 81 09 27 00 81 0D 87 03 1E 01 30 00 8D 30 0C ; ...'.......0..0.
F73D  .byte 30 00 2A 12 27 0C 23 00 7F 00 81 09 23 00 81 0D ; 0.*.'.#.....#...
F74D  .byte 25 12 25 00 23 12 25 0C 27 00 7F 00 81 09 27 00 ; %.%.#.%.'.....'.
F75D  .byte 81 0D 25 12 25 00 23 00 7F 00 81 09 23 00 81 0D ; ..%.%.#.....#...
F76D  .byte 20 0C 23 00 7F 00 81 09 23 00 81 0D 25 12 25 00 ;  .#.....#...%.%.
F77D  .byte 25 00 7F 00 81 09 25 00 81 0D 25 0C 27 00 7F 00 ; %.....%...%.'...
F78D  .byte 81 09 27 00 81 0D 30 00 8D 30 0C 30 00 2A 12 27 ; ..'...0..0.0.*.'
F79D  .byte 0C 23 00 7F 00 81 09 23 00 81 0D 25 12 25 00 23 ; .#.....#...%.%.#
F7AD  .byte 12 25 0C 27 00 7F 00 81 09 27 00 81 0D 25 12 25 ; .%.'.....'...%.%
F7BD  .byte 00 23 00 7F 00 81 09 23 00 81 0D 20 0C 23 00 7F ; .#.....#... .#..
F7CD  .byte 00 81 09 23 00 81 0D 25 12 25 00 25 00 7F 00 81 ; ...#...%.%.%....
F7DD  .byte 09 25 00 81 0D 25 0C 26 00 7F 0C 27 00 8D FF 82 ; .%...%.&...'....
F7ED  .byte FF 14 96 00 14 0A 81 0E 8A 06 88 00 12 86 03 12 ; ................
F7FD  .byte 05 12 06 0C 07 00 7F 00 81 0A 07 00 81 0E 07 00 ; ................
F80D  .byte 0C 12 0D 12 0E 0C 00 00 7F 00 81 0A 00 00 81 0E ; ................
F81D  .byte 00 00 87 07 EF 01 03 12 05 12 06 0C 07 00 7F 00 ; ................
F82D  .byte 81 0A 07 00 81 0E 07 00 0C 12 0D 12 0E 0C 00 00 ; ................
F83D  .byte 86 00 12 87 03 32 02 00 00 00 00 00 00 86 0D 12 ; .....2..........
F84D  .byte 87 03 3F 02 0D 00 0D 00 0D 00 86 08 12 87 03 4C ; ..?............L
F85D  .byte 02 08 00 08 00 08 00 86 07 12 87 03 59 02 07 00 ; ............Y...
F86D  .byte 07 00 07 00 86 00 12 87 03 66 02 00 00 00 00 00 ; .........f......
F87D  .byte 00 86 0D 12 87 03 73 02 0D 00 0D 00 0D 00 86 08 ; ......s.........
F88D  .byte 12 87 03 80 02 08 00 08 00 08 00 07 12 07 12 0D ; ................
F89D  .byte 00 0D 00 0D 00 0E 00 0E 00 0E 00 FF 84 04 00 82 ; ................
F8AD  .byte FF 14 8C 00 14 0A 81 0B 8A 06 88 86 81 0B 86 30 ; ...............0
F8BD  .byte 00 7F 00 8C 8C 8C 8C 30 00 8B 8B 8B 87 06 B0 02 ; .......0........
F8CD  .byte 81 0A 20 0C 20 00 7F 0C 20 00 7F 0C 17 00 8D 17 ; .. . ... .......
F8DD  .byte 24 23 00 25 00 26 00 27 00 26 00 25 00 23 00 20 ; $#.%.&.'.&.%.#. 
F8ED  .byte 00 1A 00 20 00 1A 00 17 0C 1A 00 17 00 87 02 AD ; ... ............
F8FD  .byte 02 86 7F 0C 17 00 17 0C 7F 00 12 0C 1A 00 7F 0C ; ................
F90D  .byte 20 12 20 00 1A 00 7F 0C 12 0C 15 00 7F 0C 17 12 ;  . .............
F91D  .byte 17 00 13 00 7F 0C 15 0C 16 00 7F 0C 17 00 7F 0C ; ................
F92D  .byte 17 00 13 12 15 0C 13 00 7F 0C 17 00 87 02 F3 02 ; ................
F93D  .byte 86 17 48 17 48 20 48 20 12 22 00 23 00 25 00 27 ; ..H.H H .".#.%.'
F94D  .byte 00 25 00 23 00 22 00 1A 00 17 00 87 02 32 03 FF ; .%.#.".......2..
F95D  .byte 81 09 8A 06 88 86 81 09 70 00 7F 00 70 00 81 0B ; ........p...p...
F96D  .byte 71 0C 81 09 70 00 87 2F 57 03 81 09 70 00 7F 00 ; q...p../W...p...
F97D  .byte 70 00 81 0B 71 00 71 00 71 00 FF 00 00 00 00 0A ; p...q.q.q.......
F98D  .byte 00 44 00 77 00 A2 00 00 00 80 01 00 02 00 85 FF ; .D.w............
F99D  .byte 83 01 01 FE F8 FF 82 FF 00 96 00 14 0A 81 08 7F ; ................
F9AD  .byte 30 7F 30 86 14 24 8B 87 04 25 00 14 24 14 24 14 ; 0.0..$...%..$.$.
F9BD  .byte 24 86 14 24 8C 87 09 33 00 88 83 00 00 00 00 00 ; $..$...3........
F9CD  .byte 7F 00 FF 83 01 01 FE F8 FF 82 FF 00 96 00 14 0A ; ................
F9DD  .byte 81 05 7F 30 7F 3C 86 14 24 8B 87 04 58 00 14 24 ; ...0.<..$...X..$
F9ED  .byte 14 24 14 24 86 14 24 8C 87 09 66 00             ; .$.$..$...f.
F9F9  88          ADC A,B
F9FA  83          ADD A,E
F9FB  00          NOP
F9FC  00          NOP
F9FD  00          NOP
F9FE  00          NOP
F9FF  00          NOP
FA00  7F          LD A,A
FA01  00          NOP
FA02  FF          RST $38
FA03  82          ADD A,D
FA04  FF          RST $38
FA05  14          INC D
FA06  82          ADD A,D
FA07  0A          LD A,(BC)
FA08  14          INC D
FA09  0A          LD A,(BC)
FA0A  81          ADD A,C
FA0B  06 8A       LD B,$8A
FA0D  03          INC BC
FA0E  86          ADD A,(HL)
FA0F  4B          LD C,E
FA10  00          NOP
FA11  3B          DEC SP
FA12  00          NOP
FA13  4B          LD C,E
FA14  00          NOP
FA15  3B          DEC SP
FA16  00          NOP
FA17  8B          ADC A,E
FA18  87          ADD A,A
FA19  07          RLCA
FA1A  83          ADD A,E
FA1B  00          NOP
FA1C  86          ADD A,(HL)
FA1D  4B          LD C,E
FA1E  00          NOP
FA1F  3B          DEC SP
FA20  00          NOP
FA21  4B          LD C,E
FA22  00          NOP
FA23  3B          DEC SP
FA24  00          NOP
FA25  8C          ADC A,H
FA26  87          ADD A,A
FA27  0D          DEC C
FA28  91          SUB C
FA29  00          NOP
FA2A  88          ADC A,B
FA2B  7F          LD A,A
FA2C  00          NOP
FA2D  FF          RST $38
FA2E  88          ADC A,B
FA2F  7F          LD A,A
FA30  10 FF       DJNZ $FA31
FA32  0A          LD A,(BC)
FA33  00          NOP
FA34  86          ADD A,(HL)
FA35  00          NOP
FA36  BC          CP H
FA37  00          NOP
FA38  FB          EI
FA39  00          NOP
FA3A  00          NOP
FA3B  00          NOP
FA3C  80          ADD A,B
FA3D  05          DEC B
FA3E  00          NOP
FA3F  06 00       LD B,$00
FA41  85          ADD A,L
FA42  FF          RST $38
FA43  83          ADD A,E
FA44  10 01       DJNZ $FA47
FA46  04          INC B
FA47  07          RLCA
FA48  00          NOP
FA49  82          ADD A,D
FA4A  FF          RST $38
FA4B  0A          LD A,(BC)
FA4C  96          SUB (HL)
FA4D  0A          LD A,(BC)
FA4E  14          INC D
FA4F  0A          LD A,(BC)
FA50  81          ADD A,C
FA51  0D          DEC C
FA52  8A          ADC A,D
FA53  06 24       LD B,$24
FA55  00          NOP
FA56  81          ADD A,C
FA57  0A          LD A,(BC)
FA58  1B          DEC DE
FA59  00          NOP
FA5A  81          ADD A,C
FA5B  0D          DEC C
FA5C  24          INC H
FA5D  00          NOP
FA5E  24          INC H
FA5F  00          NOP
FA60  24          INC H
FA61  0C          INC C
FA62  24          INC H
FA63  00          NOP
FA64  81          ADD A,C
FA65  0A          LD A,(BC)
FA66  24          INC H
FA67  00          NOP
FA68  81          ADD A,C
FA69  0D          DEC C
FA6A  1B          DEC DE
FA6B  00          NOP
FA6C  81          ADD A,C
FA6D  0A          LD A,(BC)
FA6E  24          INC H
FA6F  00          NOP
FA70  81          ADD A,C
FA71  0D          DEC C
FA72  24          INC H
FA73  00          NOP
FA74  81          ADD A,C
FA75  0A          LD A,(BC)
FA76  1B          DEC DE
FA77  00          NOP
FA78  81          ADD A,C
FA79  0D          DEC C
FA7A  26 00       LD H,$00
FA7C  81          ADD A,C
FA7D  0A          LD A,(BC)
FA7E  24          INC H
FA7F  00          NOP
FA80  81          ADD A,C
FA81  0D          DEC C
FA82  22 00 81    LD ($8100),HL
FA85  0A          LD A,(BC)
FA86  26 00       LD H,$00
FA88  81          ADD A,C
FA89  0D          DEC C
FA8A  26 00       LD H,$00
FA8C  81          ADD A,C
FA8D  0A          LD A,(BC)
FA8E  22 00 81    LD ($8100),HL
FA91  0D          DEC C
FA92  27          DAA
FA93  00          NOP
FA94  81          ADD A,C
FA95  0A          LD A,(BC)
FA96  26 00       LD H,$00
FA98  81          ADD A,C
FA99  0D          DEC C
FA9A  24          INC H
FA9B  00          NOP
FA9C  81          ADD A,C
FA9D  0A          LD A,(BC)
FA9E  27          DAA
FA9F  00          NOP
FAA0  81          ADD A,C
FAA1  0D          DEC C
FAA2  27          DAA
FAA3  00          NOP
FAA4  81          ADD A,C
FAA5  0A          LD A,(BC)
FAA6  24          INC H
FAA7  00          NOP
FAA8  82          ADD A,D
FAA9  FF          RST $38
FAAA  00          NOP
FAAB  96          SUB (HL)
FAAC  00          NOP
FAAD  14          INC D
FAAE  0A          LD A,(BC)
FAAF  28 48       JR Z,$FAF9
FAB1  8D          ADC A,L
FAB2  28 24       JR Z,$FAD8
FAB4  88          ADC A,B
FAB5  7F          LD A,A
FAB6  00          NOP
FAB7  FF          RST $38
FAB8  82          ADD A,D
FAB9  FF          RST $38
FABA  14          INC D
FABB  96          SUB (HL)
FABC  14          INC D
FABD  14          INC D
FABE  0A          LD A,(BC)
FABF  81          ADD A,C
FAC0  0E 8A       LD C,$8A
FAC2  06 04       LD B,$04
FAC4  12          LD (DE),A
FAC5  04          INC B
FAC6  00          NOP
FAC7  04          INC B
FAC8  0C          INC C
FAC9  04          INC B
FACA  0C          INC C
FACB  0E 0C       LD C,$0C
FACD  04          INC B
FACE  0C          INC C
FACF  02          LD (BC),A
FAD0  0C          INC C
FAD1  0C          INC C
FAD2  0C          INC C
FAD3  02          LD (BC),A
FAD4  0C          INC C
FAD5  00          NOP
FAD6  0C          INC C
FAD7  00          NOP
FAD8  0C          INC C
FAD9  00          NOP
FADA  0C          INC C
FADB  04          INC B
FADC  0C          INC C
FADD  04          INC B
FADE  0C          INC C
FADF  02          LD (BC),A
FAE0  0C          INC C
FAE1  82          ADD A,D
FAE2  FF          RST $38
FAE3  00          NOP
FAE4  96          SUB (HL)
FAE5  00          NOP
FAE6  14          INC D
FAE7  0A          LD A,(BC)
FAE8  04          INC B
FAE9  48          LD C,B
FAEA  88          ADC A,B
FAEB  7F          LD A,A
FAEC  00          NOP
FAED  FF          RST $38
FAEE  84          ADD A,H
FAEF  04          INC B
FAF0  00          NOP
FAF1  82          ADD A,D
FAF2  FF          RST $38
FAF3  14          INC D
FAF4  82          ADD A,D
FAF5  0A          LD A,(BC)
FAF6  14          INC D
FAF7  0A          LD A,(BC)
FAF8  81          ADD A,C
FAF9  0B          DEC BC
FAFA  8A          ADC A,D
FAFB  06 18       LD B,$18
FAFD  12          LD (DE),A
FAFE  18 00       JR $FB00
FB00  18 0C       JR $FB0E
FB02  .byte 18 0C 18 0C 18 0C 12 00 19 00 22 00             ; ..........".
FB0E  24          INC H
FB0F  00          NOP
FB10  26 00       LD H,$00
FB12  29          ADD HL,HL
FB13  00          NOP
FB14  20 00       JR NZ,$FB16
FB16  22 00 24    LD ($2400),HL
FB19  00          NOP
FB1A  27          DAA
FB1B  00          NOP
FB1C  30 00       JR NC,$FB1E
FB1E  32 00 86    LD ($8600),A
FB21  34          INC (HL)
FB22  04          INC B
FB23  36 04       LD (HL),$04
FB25  87          ADD A,A
FB26  0F          RRCA
FB27  EF          RST $28
FB28  00          NOP
FB29  88          ADC A,B
FB2A  7F          LD A,A
FB2B  00          NOP
FB2C  FF          RST $38
FB2D  81          ADD A,C
FB2E  00          NOP
FB2F  88          ADC A,B
FB30  7F          LD A,A
FB31  00          NOP
FB32  FF          RST $38
FB33  02          LD (BC),A
FB34  01 00 01    LD BC,$0100
FB37  00          NOP
FB38  00          NOP
FB39  82          ADD A,D
FB3A  FF          RST $38
FB3B  00          NOP
FB3C  FA 00 32    JP M,$3200
FB3F  0A          LD A,(BC)
FB40  83          ADD A,E
FB41  03          INC BC
FB42  01 FA F0    LD BC,$F0FA
FB45  FF          RST $38
FB46  81          ADD A,C
FB47  0D          DEC C
FB48  15          DEC D
FB49  03          INC BC
FB4A  1A          LD A,(DE)
FB4B  15          DEC D
FB4C  81          ADD A,C
FB4D  00          NOP
FB4E  FE 02       CP $02
FB50  01 00 01    LD BC,$0100
FB53  00          NOP
FB54  00          NOP
FB55  82          ADD A,D
FB56  FF          RST $38
FB57  00          NOP
FB58  FA 00 32    JP M,$3200
FB5B  0A          LD A,(BC)
FB5C  8A          ADC A,D
FB5D  01 81 0F    LD BC,$0F81
FB60  10 00       DJNZ $FB62
FB62  07          RLCA
FB63  00          NOP
FB64  04          INC B
FB65  00          NOP
FB66  00          NOP
FB67  00          NOP
FB68  86          ADD A,(HL)
FB69  17          RLA
FB6A  00          NOP
FB6B  15          DEC D
FB6C  00          NOP
FB6D  14          INC D
FB6E  00          NOP
FB6F  12          LD (DE),A
FB70  00          NOP
FB71  10 00       DJNZ $FB73
FB73  0B          DEC BC
FB74  00          NOP
FB75  8C          ADC A,H
FB76  8C          ADC A,H
FB77  8C          ADC A,H
FB78  8C          ADC A,H
FB79  87          ADD A,A
FB7A  03          INC BC
FB7B  1A          LD A,(DE)
FB7C  00          NOP
FB7D  81          ADD A,C
FB7E  00          NOP
FB7F  FE 02       CP $02
FB81  01 00 01    LD BC,$0100
FB84  00          NOP
FB85  00          NOP
FB86  82          ADD A,D
FB87  FF          RST $38
FB88  00          NOP
FB89  FA 00 32    JP M,$3200
FB8C  0A          LD A,(BC)
FB8D  81          ADD A,C
FB8E  0F          RRCA
FB8F  34          INC (HL)
FB90  04          INC B
FB91  37          SCF
FB92  04          INC B
FB93  40          LD B,B
FB94  04          INC B
FB95  8C          ADC A,H
FB96  8C          ADC A,H
FB97  40          LD B,B
FB98  04          INC B
FB99  8C          ADC A,H
FB9A  8C          ADC A,H
FB9B  40          LD B,B
FB9C  04          INC B
FB9D  8C          ADC A,H
FB9E  8C          ADC A,H
FB9F  40          LD B,B
FBA0  04          INC B
FBA1  81          ADD A,C
FBA2  00          NOP
FBA3  FE 02       CP $02
FBA5  01 00 01    LD BC,$0100
FBA8  00          NOP
FBA9  00          NOP
FBAA  82          ADD A,D
FBAB  FF          RST $38
FBAC  00          NOP
FBAD  FA 00 32    JP M,$3200
FBB0  0A          LD A,(BC)
FBB1  81          ADD A,C
FBB2  0F          RRCA
FBB3  8A          ADC A,D
FBB4  01 28 00    LD BC,$0028
FBB7  2A 00 28    LD HL,($2800)
FBBA  00          NOP
FBBB  2A 00 7F    LD HL,($7F00)
FBBE  02          LD (BC),A
FBBF  86          ADD A,(HL)
FBC0  28 00       JR Z,$FBC2
FBC2  2A 00 87    LD HL,($8700)
FBC5  09          ADD HL,BC
FBC6  1C          INC E
FBC7  00          NOP
FBC8  81          ADD A,C
FBC9  00          NOP
FBCA  FE 02       CP $02
FBCC  01 00 01    LD BC,$0100
FBCF  00          NOP
FBD0  00          NOP
FBD1  82          ADD A,D
FBD2  FF          RST $38
FBD3  00          NOP
FBD4  FA 00 32    JP M,$3200
FBD7  0A          LD A,(BC)
FBD8  81          ADD A,C
FBD9  0F          RRCA
FBDA  8A          ADC A,D
FBDB  01 86 17    LD BC,$1786
FBDE  00          NOP
FBDF  19          ADD HL,DE
FBE0  00          NOP
FBE1  1B          DEC DE
FBE2  00          NOP
FBE3  20 00       JR NZ,$FBE5
FBE5  22 00 24    LD ($2400),HL
FBE8  00          NOP
FBE9  8C          ADC A,H
FBEA  8C          ADC A,H
FBEB  87          ADD A,A
FBEC  07          RLCA
FBED  12          LD (DE),A
FBEE  00          NOP
FBEF  81          ADD A,C
FBF0  00          NOP
FBF1  FE 02       CP $02
FBF3  01 00 01    LD BC,$0100
FBF6  00          NOP
FBF7  00          NOP
FBF8  83          ADD A,E
FBF9  01 01 FA    LD BC,$FA01

; ==== sub_FBFC (1 caller) ====
FBFC  F2 FF 82    JP P,$82FF
FBFF  FF          RST $38
FC00  00          NOP
FC01  FA 00 32    JP M,$3200
FC04  0A          LD A,(BC)
FC05  81          ADD A,C
FC06  0A          LD A,(BC)
FC07  86          ADD A,(HL)
FC08  3B          DEC SP
FC09  02          LD (BC),A
FC0A  8D          ADC A,L
FC0B  8B          ADC A,E
FC0C  87          ADD A,A
FC0D  03          INC BC
FC0E  16 00       LD D,$00
FC10  3B          DEC SP
FC11  02          LD (BC),A
FC12  83          ADD A,E
FC13  01 01 FA    LD BC,$FA01
FC16  17          RLA
FC17  00          NOP
FC18  86          ADD A,(HL)
FC19  4B          LD C,E
FC1A  03          INC BC
FC1B  8D          ADC A,L
FC1C  8C          ADC A,H
FC1D  87          ADD A,A
FC1E  0E 27       LD C,$27
FC20  00          NOP
FC21  81          ADD A,C
FC22  00          NOP
FC23  FE 02       CP $02
FC25  01 00 01    LD BC,$0100
FC28  00          NOP
FC29  00          NOP
FC2A  83          ADD A,E
FC2B  01 01 FA    LD BC,$FA01
FC2E  FE FF       CP $FF
FC30  82          ADD A,D
FC31  FF          RST $38
FC32  00          NOP
FC33  FA 00 32    JP M,$3200
FC36  0A          LD A,(BC)
FC37  81          ADD A,C
FC38  0A          LD A,(BC)
FC39  86          ADD A,(HL)
FC3A  3B          DEC SP
FC3B  0C          INC C
FC3C  8D          ADC A,L
FC3D  8B          ADC A,E
FC3E  87          ADD A,A
FC3F  04          INC B
FC40  16 00       LD D,$00
FC42  86          ADD A,(HL)
FC43  3B          DEC SP
FC44  0C          INC C
FC45  8D          ADC A,L
FC46  8C          ADC A,H
FC47  87          ADD A,A
FC48  06 1F       LD B,$1F
FC4A  00          NOP
FC4B  81          ADD A,C
FC4C  00          NOP
FC4D  FE 02       CP $02
FC4F  01 00 01    LD BC,$0100
FC52  00          NOP
FC53  00          NOP
FC54  82          ADD A,D
FC55  FF          RST $38
FC56  00          NOP
FC57  FA 00 32    JP M,$3200
FC5A  0A          LD A,(BC)
FC5B  81          ADD A,C
FC5C  0F          RRCA
FC5D  86          ADD A,(HL)
FC5E  10 01       DJNZ $FC61
FC60  30 01       JR NC,$FC63
FC62  8C          ADC A,H
FC63  87          ADD A,A
FC64  0F          RRCA
FC65  10 00       DJNZ $FC67
FC67  81          ADD A,C
FC68  00          NOP
FC69  FE 02       CP $02
FC6B  01 00 01    LD BC,$0100
FC6E  00          NOP
FC6F  00          NOP
FC70  83          ADD A,E
FC71  01 01 FA    LD BC,$FA01
FC74  BF          CP A
FC75  FF          RST $38
FC76  82          ADD A,D
FC77  FF          RST $38
FC78  00          NOP
FC79  FA 00 32    JP M,$3200
FC7C  0A          LD A,(BC)
FC7D  8A          ADC A,D
FC7E  06 81       LD B,$81
FC80  0F          RRCA
FC81  86          ADD A,(HL)
FC82  10 00       DJNZ $FC84
FC84  8C          ADC A,H
FC85  12          LD (DE),A
FC86  00          NOP
FC87  8C          ADC A,H
FC88  14          INC D
FC89  00          NOP
FC8A  8C          ADC A,H
FC8B  15          DEC D
FC8C  00          NOP
FC8D  8C          ADC A,H
FC8E  17          RLA
FC8F  00          NOP
FC90  8C          ADC A,H
FC91  19          ADD HL,DE
FC92  00          NOP
FC93  87          ADD A,A
FC94  03          INC BC
FC95  18 00       JR $FC97
FC97  81          ADD A,C
FC98  00          NOP
FC99  FE 02       CP $02
FC9B  01 00 01    LD BC,$0100
FC9E  00          NOP
FC9F  00          NOP
FCA0  82          ADD A,D
FCA1  FF          RST $38
FCA2  00          NOP
FCA3  FA 00 32    JP M,$3200
FCA6  0A          LD A,(BC)
FCA7  8A          ADC A,D
FCA8  04          INC B
FCA9  81          ADD A,C
FCAA  0F          RRCA
FCAB  38 00       JR C,$FCAD
FCAD  40          LD B,B
FCAE  00          NOP
FCAF  43          LD B,E
FCB0  00          NOP
FCB1  40          LD B,B
FCB2  00          NOP
FCB3  43          LD B,E
FCB4  00          NOP
FCB5  86          ADD A,(HL)
FCB6  48          LD C,B
FCB7  00          NOP
FCB8  7F          LD A,A
FCB9  00          NOP
FCBA  8C          ADC A,H
FCBB  8C          ADC A,H
FCBC  87          ADD A,A
FCBD  07          RLCA
FCBE  1C          INC E
FCBF  00          NOP
FCC0  81          ADD A,C
FCC1  00          NOP
FCC2  FE 02       CP $02
FCC4  01 00 01    LD BC,$0100
FCC7  00          NOP
FCC8  00          NOP
FCC9  82          ADD A,D
FCCA  FF          RST $38
FCCB  00          NOP
FCCC  FA 00 32    JP M,$3200
FCCF  0A          LD A,(BC)
FCD0  81          ADD A,C
FCD1  0F          RRCA
FCD2  86          ADD A,(HL)
FCD3  18 01       JR $FCD6
FCD5  .byte 7F                                              ; .
FCD6  01 8C 10    LD BC,$108C
FCD9  01 7F 01    LD BC,$017F
FCDC  8C          ADC A,H
FCDD  87          ADD A,A
FCDE  07          RLCA
FCDF  10 00       DJNZ $FCE1
FCE1  81          ADD A,C
FCE2  00          NOP
FCE3  FE 02       CP $02
FCE5  01 00 01    LD BC,$0100
FCE8  00          NOP
FCE9  00          NOP
FCEA  82          ADD A,D
FCEB  FF          RST $38
FCEC  00          NOP
FCED  FA 00 32    JP M,$3200
FCF0  0A          LD A,(BC)
FCF1  8A          ADC A,D
FCF2  01 81 0F    LD BC,$0F81
FCF5  86          ADD A,(HL)
FCF6  3B          DEC SP
FCF7  00          NOP
FCF8  49          LD C,C
FCF9  00          NOP
FCFA  7F          LD A,A
FCFB  02          LD (BC),A
FCFC  3B          DEC SP
FCFD  00          NOP
FCFE  49          LD C,C
FCFF  00          NOP
FD00  7F          LD A,A
FD01  02          LD (BC),A
FD02  87          ADD A,A
FD03  FF          RST $38
FD04  12          LD (DE),A
FD05  00          NOP
FD06  81          ADD A,C
FD07  00          NOP
FD08  FE 02       CP $02
FD0A  01 00 01    LD BC,$0100
FD0D  00          NOP
FD0E  00          NOP
FD0F  82          ADD A,D
FD10  FF          RST $38
FD11  1E C8       LD E,$C8
FD13  1E 0A       LD E,$0A
FD15  01 83 01    LD BC,$0183
FD18  01 FA F0    LD BC,$F0FA
FD1B  FF          RST $38
FD1C  81          ADD A,C
FD1D  0F          RRCA
FD1E  10 02       DJNZ $FD22
FD20  7F          LD A,A
FD21  02          LD (BC),A
FD22  86          ADD A,(HL)
FD23  19          ADD HL,DE
FD24  0C          INC C
FD25  8C          ADC A,H
FD26  8C          ADC A,H
FD27  8C          ADC A,H
FD28  8C          ADC A,H
FD29  87          ADD A,A
FD2A  03          INC BC
FD2B  1A          LD A,(DE)
FD2C  00          NOP
FD2D  81          ADD A,C
FD2E  00          NOP
FD2F  FE 02       CP $02
FD31  01 00 01    LD BC,$0100
FD34  00          NOP
FD35  00          NOP
FD36  82          ADD A,D
FD37  FF          RST $38
FD38  00          NOP
FD39  FA 00 00    JP M,$0000
FD3C  0A          LD A,(BC)
FD3D  83          ADD A,E
FD3E  01 01 FA    LD BC,$FA01
FD41  C4 FF 81    CALL NZ,$81FF
FD44  00          NOP
FD45  86          ADD A,(HL)
FD46  00          NOP
FD47  09          ADD HL,BC
FD48  8B          ADC A,E
FD49  87          ADD A,A
FD4A  0A          LD A,(BC)
FD4B  16 00       LD D,$00
FD4D  86          ADD A,(HL)
FD4E  86          ADD A,(HL)
FD4F  00          NOP
FD50  09          ADD HL,BC
FD51  00          NOP
FD52  09          ADD HL,BC
FD53  8B          ADC A,E
FD54  87          ADD A,A
FD55  04          INC B
FD56  1F          RRA
FD57  00          NOP
FD58  86          ADD A,(HL)
FD59  00          NOP
FD5A  09          ADD HL,BC
FD5B  87          ADD A,A
FD5C  08          EX AF,AF'
FD5D  29          ADD HL,HL
FD5E  00          NOP
FD5F  86          ADD A,(HL)
FD60  00          NOP
FD61  09          ADD HL,BC
FD62  00          NOP
FD63  09          ADD HL,BC
FD64  8C          ADC A,H
FD65  87          ADD A,A
FD66  04          INC B
FD67  30 00       JR NC,$FD69
FD69  87          ADD A,A
FD6A  FF          RST $38
FD6B  1E 00       LD E,$00
FD6D  FE 02       CP $02
FD6F  01 00 01    LD BC,$0100
FD72  00          NOP
FD73  00          NOP
FD74  82          ADD A,D
FD75  FF          RST $38
FD76  00          NOP
FD77  FF          RST $38
FD78  00          NOP
FD79  0A          LD A,(BC)
FD7A  01 81 0F    LD BC,$0F81
FD7D  86          ADD A,(HL)
FD7E  00          NOP
FD7F  01 01 01    LD BC,$0101
FD82  87          ADD A,A
FD83  02          LD (BC),A
FD84  10 00       DJNZ $FD86
FD86  7F          LD A,A
FD87  05          DEC B
FD88  86          ADD A,(HL)
FD89  00          NOP
FD8A  01 01 01    LD BC,$0101
FD8D  87          ADD A,A
FD8E  10 1B       DJNZ $FDAB
FD90  00          NOP
FD91  81          ADD A,C
FD92  00          NOP
FD93  FE 03       CP $03
FD95  01 00 01    LD BC,$0100
FD98  00          NOP
FD99  00          NOP
FD9A  81          ADD A,C
FD9B  0D          DEC C
FD9C  82          ADD A,D
FD9D  FF          RST $38
FD9E  00          NOP
FD9F  FA 00 32    JP M,$3200
FDA2  0A          LD A,(BC)
FDA3  8A          ADC A,D
FDA4  02          LD (BC),A
FDA5  89          ADC A,C
FDA6  05          DEC B
FDA7  00          NOP
FDA8  00          NOP
FDA9  7F          LD A,A
FDAA  00          NOP
FDAB  89          ADC A,C
FDAC  04          INC B
FDAD  81          ADD A,C
FDAE  0C          INC C
FDAF  00          NOP
FDB0  00          NOP
FDB1  8B          ADC A,E
FDB2  86          ADD A,(HL)
FDB3  00          NOP
FDB4  00          NOP
FDB5  8C          ADC A,H
FDB6  87          ADD A,A
FDB7  0D          DEC C
FDB8  1F          RRA
FDB9  00          NOP
FDBA  81          ADD A,C
FDBB  00          NOP
FDBC  FE 02       CP $02
FDBE  01 00 01    LD BC,$0100
FDC1  00          NOP
FDC2  00          NOP
FDC3  82          ADD A,D
FDC4  FF          RST $38
FDC5  00          NOP
FDC6  FA 00 32    JP M,$3200
FDC9  0A          LD A,(BC)
FDCA  81          ADD A,C
FDCB  0F          RRCA
FDCC  8A          ADC A,D
FDCD  01 01 00    LD BC,$0001
FDD0  7F          LD A,A
FDD1  00          NOP
FDD2  02          LD (BC),A
FDD3  00          NOP
FDD4  7F          LD A,A
FDD5  00          NOP
FDD6  03          INC BC
FDD7  00          NOP
FDD8  7F          LD A,A
FDD9  00          NOP
FDDA  86          ADD A,(HL)
FDDB  04          INC B
FDDC  00          NOP
FDDD  7F          LD A,A
FDDE  00          NOP
FDDF  87          ADD A,A
FDE0  0F          RRCA
FDE1  1E 00       LD E,$00
FDE3  86          ADD A,(HL)
FDE4  04          INC B
FDE5  00          NOP
FDE6  7F          LD A,A
FDE7  00          NOP
FDE8  7F          LD A,A
FDE9  00          NOP
FDEA  8C          ADC A,H
FDEB  87          ADD A,A
FDEC  0F          RRCA
FDED  27          DAA
FDEE  00          NOP
FDEF  81          ADD A,C
FDF0  00          NOP
FDF1  FE 03       CP $03
FDF3  01 00 01    LD BC,$0100
FDF6  00          NOP
FDF7  00          NOP
FDF8  81          ADD A,C
FDF9  07          RLCA
FDFA  82          ADD A,D
FDFB  FF          RST $38
FDFC  00          NOP
FDFD  FA 00 32    JP M,$3200
FE00  0A          LD A,(BC)
FE01  89          ADC A,C
FE02  06 86       LD B,$86
FE04  00          NOP
FE05  0C          INC C
FE06  8B          ADC A,E
FE07  87          ADD A,A
FE08  07          RLCA
FE09  12          LD (DE),A
FE0A  00          NOP
FE0B  00          NOP
FE0C  30 86       JR NC,$FD94
FE0E  00          NOP
FE0F  02          LD (BC),A
FE10  8C          ADC A,H
FE11  87          ADD A,A
FE12  0E 1C       LD C,$1C
FE14  00          NOP
FE15  81          ADD A,C
FE16  00          NOP
FE17  FE 03       CP $03
FE19  01 00 01    LD BC,$0100
FE1C  00          NOP
FE1D  00          NOP
FE1E  81          ADD A,C
FE1F  0F          RRCA
FE20  82          ADD A,D
FE21  FF          RST $38
FE22  00          NOP
FE23  FA 00 32    JP M,$3200
FE26  0A          LD A,(BC)
FE27  89          ADC A,C
FE28  06 86       LD B,$86
FE2A  00          NOP
FE2B  03          INC BC
FE2C  7F          LD A,A
FE2D  03          INC BC
FE2E  8C          ADC A,H
FE2F  8C          ADC A,H
FE30  8C          ADC A,H
FE31  8C          ADC A,H
FE32  87          ADD A,A
FE33  03          INC BC
FE34  12          LD (DE),A
FE35  00          NOP
FE36  00          NOP
FE37  03          INC BC
FE38  81          ADD A,C
FE39  00          NOP
FE3A  FE 02       CP $02
FE3C  01 00 01    LD BC,$0100
FE3F  00          NOP
FE40  00          NOP
FE41  82          ADD A,D
FE42  FF          RST $38
FE43  00          NOP
FE44  FA 00 00    JP M,$0000
FE47  0A          LD A,(BC)
FE48  81          ADD A,C
FE49  0E 86       LD C,$86
FE4B  0C          INC C
FE4C  02          LD (BC),A
FE4D  0D          DEC C
FE4E  02          LD (BC),A
FE4F  87          ADD A,A
FE50  0A          LD A,(BC)
FE51  10 00       DJNZ $FE53
FE53  FE 02       CP $02
FE55  01 00 01    LD BC,$0100
FE58  00          NOP
FE59  00          NOP
FE5A  82          ADD A,D
FE5B  FF          RST $38
FE5C  00          NOP
FE5D  FA 00 32    JP M,$3200
FE60  0A          LD A,(BC)
FE61  81          ADD A,C
FE62  0D          DEC C
FE63  49          LD C,C
FE64  03          INC BC
FE65  81          ADD A,C
FE66  00          NOP
FE67  FE 03       CP $03
FE69  01 00 01    LD BC,$0100
FE6C  00          NOP
FE6D  00          NOP
FE6E  81          ADD A,C
FE6F  0F          RRCA
FE70  82          ADD A,D
FE71  FF          RST $38
FE72  0A          LD A,(BC)
FE73  96          SUB (HL)
FE74  14          INC D
FE75  50          LD D,B
FE76  0A          LD A,(BC)
FE77  8A          ADC A,D
FE78  10 89       DJNZ $FE03
FE7A  06 00       LD B,$00
FE7C  12          LD (DE),A
FE7D  81          ADD A,C
FE7E  00          NOP
FE7F  FE 02       CP $02
FE81  01 00 01    LD BC,$0100
FE84  00          NOP
FE85  00          NOP
FE86  82          ADD A,D
FE87  FF          RST $38
FE88  00          NOP
FE89  FA 00 32    JP M,$3200
FE8C  0A          LD A,(BC)
FE8D  8A          ADC A,D
FE8E  01 81 0F    LD BC,$0F81
FE91  0C          INC C
FE92  02          LD (BC),A
FE93  7F          LD A,A
FE94  02          LD (BC),A
FE95  86          ADD A,(HL)
FE96  11 00 09    LD DE,$0900
FE99  00          NOP
FE9A  07          RLCA
FE9B  00          NOP
FE9C  04          INC B
FE9D  00          NOP
FE9E  03          INC BC
FE9F  00          NOP
FEA0  00          NOP
FEA1  00          NOP
FEA2  0E 00       LD C,$00
FEA4  0C          INC C
FEA5  00          NOP
FEA6  8C          ADC A,H
FEA7  8C          ADC A,H
FEA8  8C          ADC A,H
FEA9  87          ADD A,A
FEAA  04          INC B
FEAB  16 00       LD D,$00
FEAD  81          ADD A,C
FEAE  00          NOP
FEAF  FE 02       CP $02
FEB1  01 00 01    LD BC,$0100
FEB4  00          NOP
FEB5  00          NOP
FEB6  82          ADD A,D
FEB7  FF          RST $38
FEB8  00          NOP
FEB9  FA 00 32    JP M,$3200
FEBC  0A          LD A,(BC)
FEBD  81          ADD A,C
FEBE  0F          RRCA
FEBF  8A          ADC A,D
FEC0  01 17 00    LD BC,$0017
FEC3  10 00       DJNZ $FEC5
FEC5  07          RLCA
FEC6  00          NOP
FEC7  00          NOP
FEC8  00          NOP
FEC9  0C          INC C
FECA  00          NOP
FECB  86          ADD A,(HL)
FECC  4B          LD C,E
FECD  00          NOP
FECE  0B          DEC BC
FECF  00          NOP
FED0  8C          ADC A,H
FED1  87          ADD A,A
FED2  0F          RRCA
FED3  1C          INC E
FED4  00          NOP
FED5  81          ADD A,C
FED6  00          NOP
FED7  FE 02       CP $02
FED9  01 00 01    LD BC,$0100
FEDC  00          NOP
FEDD  00          NOP
FEDE  82          ADD A,D
FEDF  FF          RST $38
FEE0  00          NOP
FEE1  FA 00 32    JP M,$3200
FEE4  0A          LD A,(BC)
FEE5  81          ADD A,C
FEE6  0F          RRCA
FEE7  86          ADD A,(HL)
FEE8  19          ADD HL,DE
FEE9  01 39 01    LD BC,$0139
FEEC  8C          ADC A,H
FEED  87          ADD A,A
FEEE  0F          RRCA
FEEF  10 00       DJNZ $FEF1
FEF1  81          ADD A,C
FEF2  00          NOP
FEF3  FE 03       CP $03
FEF5  01 00 01    LD BC,$0100
FEF8  00          NOP
FEF9  00          NOP
FEFA  82          ADD A,D
FEFB  FF          RST $38
FEFC  00          NOP
FEFD  FA 00 32    JP M,$3200
FF00  0A          LD A,(BC)
FF01  8A          ADC A,D
FF02  03          INC BC
FF03  81          ADD A,C
FF04  0E 89       LD C,$89
FF06  06 86       LD B,$86
FF08  00          NOP
FF09  00          NOP
FF0A  7F          LD A,A
FF0B  00          NOP
FF0C  8C          ADC A,H
FF0D  87          ADD A,A
FF0E  0E 14       LD C,$14
FF10  00          NOP
FF11  81          ADD A,C
FF12  00          NOP
FF13  FE 02       CP $02
FF15  01 00 01    LD BC,$0100
FF18  00          NOP
FF19  00          NOP
FF1A  82          ADD A,D
FF1B  FF          RST $38
FF1C  00          NOP
FF1D  FA 00 32    JP M,$3200
FF20  0A          LD A,(BC)
FF21  81          ADD A,C
FF22  0F          RRCA
FF23  8A          ADC A,D
FF24  01 09 00    LD BC,$0009
FF27  7F          LD A,A
FF28  00          NOP
FF29  47          LD B,A
FF2A  00          NOP
FF2B  7F          LD A,A
FF2C  00          NOP
FF2D  40          LD B,B
FF2E  00          NOP
FF2F  5B          LD E,E
FF30  00          NOP
FF31  7F          LD A,A
FF32  00          NOP
FF33  57          LD D,A
FF34  02          LD (BC),A
FF35  54          LD D,H
FF36  00          NOP
FF37  4B          LD C,E
FF38  00          NOP
FF39  7F          LD A,A
FF3A  02          LD (BC),A
FF3B  5B          LD E,E
FF3C  00          NOP
FF3D  7F          LD A,A
FF3E  00          NOP
FF3F  54          LD D,H
FF40  00          NOP
FF41  7F          LD A,A
FF42  00          NOP
FF43  5B          LD E,E
FF44  00          NOP
FF45  7F          LD A,A
FF46  00          NOP
FF47  81          ADD A,C
FF48  0C          INC C
FF49  86          ADD A,(HL)
FF4A  5B          LD E,E
FF4B  00          NOP
FF4C  0C          INC C
FF4D  00          NOP
FF4E  7F          LD A,A
FF4F  00          NOP
FF50  57          LD D,A
FF51  00          NOP
FF52  8C          ADC A,H
FF53  87          ADD A,A
FF54  06 36       LD B,$36
FF56  00          NOP
FF57  81          ADD A,C
FF58  00          NOP
FF59  FE 02       CP $02
FF5B  01 00 01    LD BC,$0100
FF5E  00          NOP
FF5F  00          NOP
FF60  82          ADD A,D
FF61  FF          RST $38
FF62  00          NOP
FF63  FA 00 00    JP M,$0000
FF66  0A          LD A,(BC)
FF67  81          ADD A,C
FF68  0B          DEC BC
FF69  8A          ADC A,D
FF6A  01 19 00    LD BC,$0019
FF6D  8B          ADC A,E
FF6E  15          DEC D
FF6F  00          NOP
FF70  8B          ADC A,E
FF71  22 00 27    LD ($2700),HL
FF74  00          NOP
FF75  7F          LD A,A
FF76  03          INC BC
FF77  2B          DEC HL
FF78  00          NOP
FF79  32 00 36    LD ($3600),A
FF7C  00          NOP
FF7D  39          ADD HL,SP
FF7E  00          NOP
FF7F  7F          LD A,A
FF80  06 81       LD B,$81
FF82  05          DEC B
FF83  2B          DEC HL
FF84  00          NOP
FF85  32 00 36    LD ($3600),A
FF88  00          NOP
FF89  39          ADD HL,SP
FF8A  00          NOP
FF8B  81          ADD A,C
FF8C  00          NOP
FF8D  FE 00       CP $00
FF8F  02          LD (BC),A
FF90  01 00 01    LD BC,$0100
FF93  00          NOP
FF94  00          NOP
FF95  83          ADD A,E
FF96  01 01 FA    LD BC,$FA01
FF99  F2 FF 82    JP P,$82FF
FF9C  FF          RST $38
FF9D  00          NOP
FF9E  FA 00 00    JP M,$0000
FFA1  0A          LD A,(BC)
FFA2  86          ADD A,(HL)
FFA3  81          ADD A,C
FFA4  0A          LD A,(BC)
FFA5  86          ADD A,(HL)
FFA6  30 06       JR NC,$FFAE
FFA8  8B          ADC A,E
FFA9  87          ADD A,A
FFAA  04          INC B
FFAB  17          RLA
FFAC  00          NOP
FFAD  86          ADD A,(HL)
FFAE  30 06       JR NC,$FFB6
FFB0  8C          ADC A,H
FFB1  8C          ADC A,H
FFB2  87          ADD A,A
FFB3  05          DEC B
FFB4  1F          RRA
FFB5  00          NOP
FFB6  87          ADD A,A
FFB7  FF          RST $38
FFB8  14          INC D
FFB9  00          NOP
FFBA  81          ADD A,C
FFBB  00          NOP
FFBC  FE 6D       CP $6D
FFBE  20 26       JR NZ,$FFE6
FFC0  20 47       JR NZ,$0009
FFC2  61          LD H,C
FFC3  6D          LD L,L
FFC4  65          LD H,L
FFC5  20 47       JR NZ,$000E
FFC7  65          LD H,L
FFC8  61          LD H,C
FFC9  72          LD (HL),D
FFCA  20 56       JR NZ,$0022
FFCC  65          LD H,L
FFCD  72          LD (HL),D
FFCE  73          LD (HL),E
FFCF  69          LD L,C
FFD0  6F          LD L,A
FFD1  6E          LD L,(HL)
FFD2  2E 20       LD L,$20
FFD4  20 27       JR NZ,$FFFD
FFD6  31 39 39    LD SP,$3939
FFD9  31 20 28    LD SP,$2820
FFDC  43          LD B,E
FFDD  29          ADD HL,HL
FFDE  41          LD B,C
FFDF  6E          LD L,(HL)
FFE0  63          LD H,E
FFE1  69          LD L,C
FFE2  65          LD H,L
FFE3  6E          LD L,(HL)
FFE4  74          LD (HL),H
FFE5  2E 20       LD L,$20
FFE7  28 42       JR Z,$002B
FFE9  41          LD B,C
FFEA  4E          LD C,(HL)
FFEB  4B          LD C,E
FFEC  30 2D       JR NC,$001B
FFEE  34          INC (HL)
FFEF  29          ADD HL,HL
FFF0  A2          AND D
FFF1  53          LD D,E
FFF2  4F          LD C,A
FFF3  4E          LD C,(HL)
FFF4  49          LD C,C
FFF5  43          LD B,E
FFF6  20 54       JR NZ,$004C
FFF8  48          LD C,B
FFF9  45          LD B,L
FFFA  20 48       JR NZ,$0044
FFFC  45          LD B,L
FFFD  44          LD B,H
FFFE  47          LD B,A
FFFF  45          LD B,L
