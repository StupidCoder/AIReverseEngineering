
; --- cstart  $040000 — entry: AmigaDOS/the launcher enters here. Save the d0/a0 args (control block, filename) to globals, OldOpenLibrary a library, then call main with (control-block, filename). ---
040000  23 C0 00 04 00 9C             MOVE.l d0,$4009C.l
040006  23 C8 00 04 00 A0             MOVE.l a0,$400A0.l
04000C  23 F8 00 04 00 04 00 60       MOVE.l $4.w,$40060.l
040014  48 E7 7E FE                   MOVEM.l d1-d6/a0-a6,-(a7)
040018  23 CF 00 04 00 94             MOVE.l a7,$40094.l
04001E  2C 79 00 04 00 60             MOVEA.l $40060.l,a6
040024  61 00 00 20                   BSR $040046
040028  2F 39 00 04 00 A0             MOVE.l $400A0.l,-(a7)
04002E  2F 39 00 04 00 9C             MOVE.l $4009C.l,-(a7)
040034  4E B9 00 04 00 B4             JSR $400B4.l

; ==== exit  $04003A  (6 callers) — return to the caller (restore the saved stack, MOVEM the regs, RTS); error paths jump here with a code. ====
04003A  2E 79 00 04 00 94             MOVEA.l $40094.l,a7
040040  4C DF                         .dc.w $4CDF
040042  .dc.b 7F 7E 4E 75                                     ; .~Nu

; ==== open_lib  $040046  (1 caller) — OldOpenLibrary the library named at the data pointer (exec -$198); store the base. ====
040046  43 F9 00 04 00 A8             LEA $400A8.l,a1
04004C  4E AE FE 68                   JSR -$198(a6)
040050  23 C0 00 04 00 A4             MOVE.l d0,$400A4.l
040056  4E 75                         RTS
040058  .dc.b 70 01 60 DE 00 00 00 00 00 00 00 00 00 00 00 00 ; p.`.............
040068  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
040078  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
040088  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
040098  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0400A8  .dc.b 64 6F 73 2E 6C 69 62 72 61 72 79 00             ; dos.library.

; ==== main  $0400B4  (1 caller) — main(control, filename): build the keystream table (key_init), decrunch+load the file (decrunch), free the table, return the seglist. ====
0400B4  4E 56 FF FC                   LINK a6,#-$4
0400B8  20 6E 00 08                   MOVEA.l $8(a6),a0
0400BC  2F 28 00 08                   MOVE.l $8(a0),-(a7)
0400C0  2F 28 00 0C                   MOVE.l $C(a0),-(a7)
0400C4  4E B9 00 04 0D 06             JSR $40D06.l
0400CA  50 8F                         ADDQ.l #8,a7
0400CC  23 C0 00 04 01 08             MOVE.l d0,$40108.l
0400D2  20 6E 00 08                   MOVEA.l $8(a6),a0
0400D6  D0 FC 00 18                   ADDA.w #$18,a0
0400DA  42 A7                         CLR.l -(a7)
0400DC  2F 08                         MOVE.l a0,-(a7)
0400DE  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
0400E2  4E B9 00 04 02 C8             JSR $402C8.l
0400E8  4F EF 00 0C                   LEA $C(a7),a7
0400EC  2F 39 00 04 01 08             MOVE.l $40108.l,-(a7)
0400F2  2D 40 FF FC                   MOVE.l d0,-$4(a6)
0400F6  4E B9 00 04 0D 78             JSR $40D78.l
0400FC  58 8F                         ADDQ.l #4,a7
0400FE  20 2E FF FC                   MOVE.l -$4(a6),d0
040102  4E 5E                         UNLK a6
040104  4E 75                         RTS
040106  .dc.b 00 00 00 00 00 00                               ; ......

; ==== bptr_helper  $04010C  (1 caller) — small BPTR/seglist helper used while building the loaded segment list. ====
04010C  4E 56 00 00                   LINK a6,#$0
040110  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040114  4E B9 00 04 00 3A             JSR $4003A.l
04011A  58 8F                         ADDQ.l #4,a7
04011C  4E 5E                         UNLK a6
04011E  4E 75                         RTS

; ==== hunk_base  $040120  (5 callers) — look hunk N up in the per-hunk pointer table ($40BC0) and return its allocated address (as an address-pointer). ====
040120  4E 56 00 00                   LINK a6,#$0
040124  20 2E 00 08                   MOVE.l $8(a6),d0
040128  E5 80                         ASL.l #2,d0
04012A  20 79 00 04 0B C0             MOVEA.l $40BC0.l,a0
040130  D1 C0                         ADDA.l d0,a0
040132  4A 90                         TST.l (a0)
040134  67 12                         BEQ $040148
040136  20 79 00 04 0B C0             MOVEA.l $40BC0.l,a0
04013C  D1 C0                         ADDA.l d0,a0
04013E  20 10                         MOVE.l (a0),d0
040140  52 80                         ADDQ.l #1,d0
040142  E5 80                         ASL.l #2,d0
040144  4E 5E                         UNLK a6
040146  4E 75                         RTS
040148  70 00                         MOVEQ #$0,d0
04014A  4E 5E                         UNLK a6
04014C  4E 75                         RTS

; ==== hunk_size_cell  $04014E  (1 caller) — fetch a hunk's size/cell from the hunk table. ====
04014E  4E 56 00 00                   LINK a6,#$0
040152  20 2E 00 08                   MOVE.l $8(a6),d0
040156  E5 80                         ASL.l #2,d0
040158  20 79 00 04 0B C0             MOVEA.l $40BC0.l,a0
04015E  D1 C0                         ADDA.l d0,a0
040160  4A 90                         TST.l (a0)
040162  67 10                         BEQ $040174
040164  20 79 00 04 0B C0             MOVEA.l $40BC0.l,a0
04016A  D1 C0                         ADDA.l d0,a0
04016C  20 10                         MOVE.l (a0),d0
04016E  E5 80                         ASL.l #2,d0
040170  4E 5E                         UNLK a6
040172  4E 75                         RTS
040174  70 01                         MOVEQ #$1,d0
040176  4E 5E                         UNLK a6
040178  4E 75                         RTS

; ==== alloc_hunk  $04017A  (1 caller) — AllocMem one hunk (size+2 longwords, MEMF flags from the 2-bit size tag), link it into the growing seglist, and record it in the $40BC0 table. ====
04017A  4E 56 FF F0                   LINK a6,#-$10
04017E  20 2E 00 0C                   MOVE.l $C(a6),d0
040182  54 80                         ADDQ.l #2,d0
040184  E5 80                         ASL.l #2,d0
040186  2F 2E 00 10                   MOVE.l $10(a6),-(a7)
04018A  2F 00                         MOVE.l d0,-(a7)
04018C  2D 40 FF F0                   MOVE.l d0,-$10(a6)
040190  4E B9 00 04 0F 30             JSR $40F30.l
040196  50 8F                         ADDQ.l #8,a7
040198  20 40                         MOVEA.l d0,a0
04019A  C1 88                         EXG d0,a0
04019C  02 40 FF FE                   ANDI.w #$FFFE,d0
0401A0  C1 88                         EXG d0,a0
0401A2  2D 48 FF FC                   MOVE.l a0,-$4(a6)
0401A6  B1 FC 00 00 00 00             CMPA.l #$0,a0
0401AC  66 0A                         BNE $0401B8
0401AE  70 01                         MOVEQ #$1,d0
0401B0  2F 00                         MOVE.l d0,-(a7)
0401B2  61 00 FF 58                   BSR $04010C
0401B6  58 8F                         ADDQ.l #4,a7
0401B8  20 6E FF FC                   MOVEA.l -$4(a6),a0
0401BC  20 AE FF F0                   MOVE.l -$10(a6),(a0)
0401C0  20 08                         MOVE.l a0,d0
0401C2  58 80                         ADDQ.l #4,d0
0401C4  E4 80                         ASR.l #2,d0
0401C6  72 00                         MOVEQ #$0,d1
0401C8  21 41 00 04                   MOVE.l d1,$4(a0)
0401CC  22 2E 00 08                   MOVE.l $8(a6),d1
0401D0  E5 81                         ASL.l #2,d1
0401D2  20 79 00 04 0B C0             MOVEA.l $40BC0.l,a0
0401D8  D1 C1                         ADDA.l d1,a0
0401DA  20 80                         MOVE.l d0,(a0)
0401DC  2D 40 FF F4                   MOVE.l d0,-$C(a6)
0401E0  4A AE 00 08                   TST.l $8(a6)
0401E4  6F 1E                         BLE $040204
0401E6  4A AE 00 14                   TST.l $14(a6)
0401EA  67 18                         BEQ $040204
0401EC  20 2E 00 08                   MOVE.l $8(a6),d0
0401F0  53 80                         SUBQ.l #1,d0
0401F2  2F 00                         MOVE.l d0,-(a7)
0401F4  61 00 FF 58                   BSR $04014E
0401F8  58 8F                         ADDQ.l #4,a7
0401FA  20 40                         MOVEA.l d0,a0
0401FC  20 AE FF F4                   MOVE.l -$C(a6),(a0)
040200  2D 40 FF F8                   MOVE.l d0,-$8(a6)
040204  4E 5E                         UNLK a6
040206  4E 75                         RTS

; ==== fillbytes  $040208  (7 callers) — buffered reader: copy N raw bytes from the 512-byte input buffer into a destination, refilling via _Read in $200-byte blocks. Raw — does NOT consume the keystream. ====
040208  4E 56 FF F8                   LINK a6,#-$8
04020C  48 E7 01 04                   MOVEM.l d7/a5,-(a7)
040210  2E 2E 00 10                   MOVE.l $10(a6),d7
040214  2A 6E 00 0C                   MOVEA.l $C(a6),a5
040218  4A 87                         TST.l d7
04021A  67 74                         BEQ $040290
04021C  20 79 00 04 0B DE             MOVEA.l $40BDE.l,a0
040222  B1 F9 00 04 0B E2             CMPA.l $40BE2.l,a0
040228  64 1E                         BCC $040248
04022A  1A 90                         MOVE.b (a0),(a5)
04022C  52 B9 00 04 0B DE             ADDQ.l #1,$40BDE.l
040232  52 8D                         ADDQ.l #1,a5
040234  53 87                         SUBQ.l #1,d7
040236  20 07                         MOVE.l d7,d0
040238  4A 80                         TST.l d0
04023A  66 E0                         BNE $04021C
04023C  20 2E 00 10                   MOVE.l $10(a6),d0
040240  4C DF                         .dc.w $4CDF
040242  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu
040248  20 79 00 04 0B DA             MOVEA.l $40BDA.l,a0
04024E  2F 3C 00 00 02 00             MOVE.l #$200,-(a7)
040254  2F 08                         MOVE.l a0,-(a7)
040256  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04025A  23 C8 00 04 0B DE             MOVE.l a0,$40BDE.l
040260  4E B9 00 04 0F 14             JSR $40F14.l
040266  4F EF 00 0C                   LEA $C(a7),a7
04026A  20 79 00 04 0B DE             MOVEA.l $40BDE.l,a0
040270  D1 C0                         ADDA.l d0,a0
040272  23 C8 00 04 0B E2             MOVE.l a0,$40BE2.l
040278  22 79 00 04 0B DE             MOVEA.l $40BDE.l,a1
04027E  B3 C8                         CMPA.l a0,a1
040280  66 96                         BNE $040218
040282  20 2E 00 10                   MOVE.l $10(a6),d0
040286  90 87                         SUB.l d7,d0
040288  4C DF                         .dc.w $4CDF
04028A  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu
040290  4C DF                         .dc.w $4CDF
040292  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu

; ==== readlong  $040298  (25 callers) — read one longword via fillbytes, then (if the decrypt flag $40BC8 is set) XOR it with the next keystream word. This is the only thing that advances the keystream. ====
040298  4E 56 00 00                   LINK a6,#$0
04029C  70 04                         MOVEQ #$4,d0
04029E  2F 00                         MOVE.l d0,-(a7)
0402A0  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
0402A4  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0402A8  61 00 FF 5E                   BSR $040208
0402AC  4F EF 00 0C                   LEA $C(a7),a7
0402B0  4A B9 00 04 0B C8             TST.l $40BC8.l
0402B6  67 0C                         BEQ $0402C4
0402B8  4E B9 00 04 0E AC             JSR $40EAC.l
0402BE  20 6E 00 0C                   MOVEA.l $C(a6),a0
0402C2  B1 90                         EOR.l d0,(a0)
0402C4  4E 5E                         UNLK a6
0402C6  4E 75                         RTS

; ==== decrunch  $0402C8  (1 caller) — decode-and-load: _Open the file (or use a preopened handle), allocate the work + 512-byte stream buffers, then loop reading each block's TYPE longword RAW, masking ($3FFFFFFF), subtracting $3E7, range-checking, and dispatching through the JMP table at $403DE to the per-type arm. Returns the loaded seglist (BCPL). ====
0402C8  4E 56 FF D8                   LINK a6,#-$28
0402CC  70 FF                         MOVEQ #$FF,d0
0402CE  2D 40 FF F4                   MOVE.l d0,-$C(a6)
0402D2  42 AE FF D8                   CLR.l -$28(a6)
0402D6  4A AE 00 08                   TST.l $8(a6)
0402DA  67 4C                         BEQ $040328
0402DC  2F 3C 00 00 03 ED             MOVE.l #$3ED,-(a7)
0402E2  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0402E6  4E B9 00 04 0E E4             JSR $40EE4.l
0402EC  50 8F                         ADDQ.l #8,a7
0402EE  2D 40 FF FC                   MOVE.l d0,-$4(a6)
0402F2  4A 80                         TST.l d0
0402F4  66 0C                         BNE $040302
0402F6  70 02                         MOVEQ #$2,d0
0402F8  2F 00                         MOVE.l d0,-(a7)
0402FA  4E B9 00 04 00 3A             JSR $4003A.l
040300  58 8F                         ADDQ.l #4,a7
040302  70 00                         MOVEQ #$0,d0
040304  2D 6E 00 0C FF D8             MOVE.l $C(a6),-$28(a6)
04030A  42 79 00 04 0B D8             CLR.w $40BD8.l
040310  72 01                         MOVEQ #$1,d1
040312  23 C1 00 04 0B C8             MOVE.l d1,$40BC8.l
040318  23 C0 00 04 0B CC             MOVE.l d0,$40BCC.l
04031E  2D 40 00 0C                   MOVE.l d0,$C(a6)
040322  2D 40 FF E8                   MOVE.l d0,-$18(a6)
040326  60 18                         BRA $040340
040328  2D 6E 00 10 FF FC             MOVE.l $10(a6),-$4(a6)
04032E  70 01                         MOVEQ #$1,d0
040330  2D 40 FF E8                   MOVE.l d0,-$18(a6)
040334  70 00                         MOVEQ #$0,d0
040336  2D 40 FF D8                   MOVE.l d0,-$28(a6)
04033A  23 C0 00 04 0B C8             MOVE.l d0,$40BC8.l
040340  70 02                         MOVEQ #$2,d0
040342  2F 00                         MOVE.l d0,-(a7)
040344  2F 3C 00 00 00 80             MOVE.l #$80,-(a7)
04034A  4E B9 00 04 0F 30             JSR $40F30.l
040350  50 8F                         ADDQ.l #8,a7
040352  23 C0 00 04 0B D4             MOVE.l d0,$40BD4.l
040358  4A 80                         TST.l d0
04035A  66 0C                         BNE $040368
04035C  70 02                         MOVEQ #$2,d0
04035E  2F 00                         MOVE.l d0,-(a7)
040360  4E B9 00 04 00 3A             JSR $4003A.l
040366  58 8F                         ADDQ.l #4,a7
040368  70 02                         MOVEQ #$2,d0
04036A  2F 00                         MOVE.l d0,-(a7)
04036C  2F 3C 00 00 02 00             MOVE.l #$200,-(a7)
040372  4E B9 00 04 0F 30             JSR $40F30.l
040378  50 8F                         ADDQ.l #8,a7
04037A  23 C0 00 04 0B DE             MOVE.l d0,$40BDE.l
040380  23 C0 00 04 0B E2             MOVE.l d0,$40BE2.l
040386  23 C0 00 04 0B DA             MOVE.l d0,$40BDA.l
04038C  4A 80                         TST.l d0
04038E  66 0C                         BNE $04039C
040390  70 02                         MOVEQ #$2,d0
040392  2F 00                         MOVE.l d0,-(a7)
040394  4E B9 00 04 00 3A             JSR $4003A.l
04039A  58 8F                         ADDQ.l #4,a7
04039C  70 04                         MOVEQ #$4,d0
04039E  2F 00                         MOVE.l d0,-(a7)
0403A0  48 6E FF F8                   PEA -$8(a6)
0403A4  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
0403A8  61 00 FE 5E                   BSR $040208
0403AC  4F EF 00 0C                   LEA $C(a7),a7
0403B0  2D 40 FF F0                   MOVE.l d0,-$10(a6)
0403B4  4A 80                         TST.l d0
0403B6  67 00 01 BC                   BEQ $040574
0403BA  20 2E FF F8                   MOVE.l -$8(a6),d0
0403BE  02 80 3F FF FF FF             ANDI.l #$3FFFFFFF,d0
0403C4  2D 40 FF E0                   MOVE.l d0,-$20(a6)
0403C8  04 80 00 00 03 E7             SUBI.l #$3E7,d0
0403CE  6D 00 01 94                   BLT $040564
0403D2  0C 80 00 00 00 10             CMPI.l #$10,d0
0403D8  6C 00 01 8A                   BGE $040564
0403DC  E5 80                         ASL.l #2,d0
0403DE  4E FB 08 02                   JMP $2(pc,d0.l)
0403E2  .dc.b 60 00 00 3E 60 00 00 48 60 00 00 52 60 00 00 68 ; `..>`..H`..R`..h
0403F2  .dc.b 60 00 00 7E 60 00 00 94 60 00 00 A2 60 00 00 AC ; `..~`...`...`...
040402  .dc.b 60 00 00 B6 60 00 00 C0 60 00 00 CE 60 00 00 D8 ; `...`...`...`...
040412  .dc.b 60 00 00 E2 60 00 01 0A 60 00 01 14 60 00 01 1C ; `...`...`...`...
040422  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
040426  61 00 01 E2                   BSR $04060A
04042A  58 8F                         ADDQ.l #4,a7
04042C  60 00 FF 6E                   BRA $04039C
040430  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
040434  61 00 01 FC                   BSR $040632
040438  58 8F                         ADDQ.l #4,a7
04043A  60 00 FF 60                   BRA $04039C
04043E  20 2E FF F4                   MOVE.l -$C(a6),d0
040442  52 80                         ADDQ.l #1,d0
040444  2F 00                         MOVE.l d0,-(a7)
040446  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
04044A  2D 40 FF F4                   MOVE.l d0,-$C(a6)
04044E  61 00 02 0A                   BSR $04065A
040452  50 8F                         ADDQ.l #8,a7
040454  60 00 FF 46                   BRA $04039C
040458  20 2E FF F4                   MOVE.l -$C(a6),d0
04045C  52 80                         ADDQ.l #1,d0
04045E  2F 00                         MOVE.l d0,-(a7)
040460  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
040464  2D 40 FF F4                   MOVE.l d0,-$C(a6)
040468  61 00 02 6A                   BSR $0406D4
04046C  50 8F                         ADDQ.l #8,a7
04046E  60 00 FF 2C                   BRA $04039C
040472  20 2E FF F4                   MOVE.l -$C(a6),d0
040476  52 80                         ADDQ.l #1,d0
040478  2F 00                         MOVE.l d0,-(a7)
04047A  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
04047E  2D 40 FF F4                   MOVE.l d0,-$C(a6)
040482  61 00 02 CA                   BSR $04074E
040486  50 8F                         ADDQ.l #8,a7
040488  60 00 FF 12                   BRA $04039C
04048C  2F 2E FF F4                   MOVE.l -$C(a6),-(a7)
040490  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
040494  61 00 02 CE                   BSR $040764
040498  50 8F                         ADDQ.l #8,a7
04049A  60 00 FF 00                   BRA $04039C
04049E  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
0404A2  61 00 03 78                   BSR $04081C
0404A6  58 8F                         ADDQ.l #4,a7
0404A8  60 00 FE F2                   BRA $04039C
0404AC  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
0404B0  61 00 03 AA                   BSR $04085C
0404B4  58 8F                         ADDQ.l #4,a7
0404B6  60 00 FE E4                   BRA $04039C
0404BA  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
0404BE  61 00 03 DC                   BSR $04089C
0404C2  58 8F                         ADDQ.l #4,a7
0404C4  60 00 FE D6                   BRA $04039C
0404C8  2F 2E FF F4                   MOVE.l -$C(a6),-(a7)
0404CC  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
0404D0  61 00 03 D8                   BSR $0408AA
0404D4  50 8F                         ADDQ.l #8,a7
0404D6  60 00 FE C4                   BRA $04039C
0404DA  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
0404DE  61 00 04 14                   BSR $0408F4
0404E2  58 8F                         ADDQ.l #4,a7
0404E4  60 00 FE B6                   BRA $04039C
0404E8  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
0404EC  61 00 04 2E                   BSR $04091C
0404F0  58 8F                         ADDQ.l #4,a7
0404F2  60 00 FE A8                   BRA $04039C

; --- arm_header  $0404F6 — HUNK_HEADER ($3F3) arm: call hdr_liblist then hdr_alloc. ---
0404F6  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
0404FA  61 00 04 28                   BSR $040924
0404FE  58 8F                         ADDQ.l #4,a7
040500  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
040504  2F 00                         MOVE.l d0,-(a7)
040506  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
04050A  2D 40 FF EC                   MOVE.l d0,-$14(a6)
04050E  61 00 05 B0                   BSR $040AC0
040512  4F EF 00 0C                   LEA $C(a7),a7
040516  2D 40 FF E4                   MOVE.l d0,-$1C(a6)
04051A  2D 40 FF F4                   MOVE.l d0,-$C(a6)
04051E  60 00 FE 7C                   BRA $04039C
040522  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
040526  61 00 04 4C                   BSR $040974
04052A  58 8F                         ADDQ.l #4,a7
04052C  60 00 FE 6E                   BRA $04039C
040530  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
040534  61 00 04 74                   BSR $0409AA
040538  58 8F                         ADDQ.l #4,a7
04053A  60 38                         BRA $040574
04053C  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
040540  61 00 05 20                   BSR $040A62
040544  58 8F                         ADDQ.l #4,a7
040546  4A AE FF E8                   TST.l -$18(a6)
04054A  67 00 FE 50                   BEQ $04039C
04054E  20 2E FF E4                   MOVE.l -$1C(a6),d0
040552  E5 80                         ASL.l #2,d0
040554  58 80                         ADDQ.l #4,d0
040556  20 79 00 04 0B C0             MOVEA.l $40BC0.l,a0
04055C  D1 C0                         ADDA.l d0,a0
04055E  2D 50 FF DC                   MOVE.l (a0),-$24(a6)
040562  60 4A                         BRA $0405AE
040564  70 01                         MOVEQ #$1,d0
040566  2F 00                         MOVE.l d0,-(a7)
040568  4E B9 00 04 00 3A             JSR $4003A.l
04056E  58 8F                         ADDQ.l #4,a7
040570  60 00 FE 2A                   BRA $04039C
040574  4A B9 00 04 0B CC             TST.l $40BCC.l
04057A  67 0C                         BEQ $040588
04057C  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
040580  61 00 04 92                   BSR $040A14
040584  58 8F                         ADDQ.l #4,a7
040586  60 1C                         BRA $0405A4
040588  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
04058C  4E B9 00 04 0F 00             JSR $40F00.l
040592  58 8F                         ADDQ.l #4,a7
040594  4A AE FF D8                   TST.l -$28(a6)
040598  67 0A                         BEQ $0405A4
04059A  20 6E FF D8                   MOVEA.l -$28(a6),a0
04059E  30 B9 00 04 0B D8             MOVE.w $40BD8.l,(a0)
0405A4  20 79 00 04 0B C0             MOVEA.l $40BC0.l,a0
0405AA  2D 50 FF DC                   MOVE.l (a0),-$24(a6)
0405AE  2F 3C 00 00 02 00             MOVE.l #$200,-(a7)
0405B4  2F 39 00 04 0B DA             MOVE.l $40BDA.l,-(a7)
0405BA  4E B9 00 04 0F 48             JSR $40F48.l
0405C0  50 8F                         ADDQ.l #8,a7
0405C2  2F 3C 00 00 00 80             MOVE.l #$80,-(a7)
0405C8  2F 39 00 04 0B D4             MOVE.l $40BD4.l,-(a7)
0405CE  4E B9 00 04 0F 48             JSR $40F48.l
0405D4  50 8F                         ADDQ.l #8,a7
0405D6  4A B9 00 04 0B CC             TST.l $40BCC.l
0405DC  66 24                         BNE $040602
0405DE  4A AE FF E8                   TST.l -$18(a6)
0405E2  66 1E                         BNE $040602
0405E4  20 79 00 04 0B C0             MOVEA.l $40BC0.l,a0
0405EA  59 88                         SUBQ.l #4,a0
0405EC  23 C8 00 04 0B C0             MOVE.l a0,$40BC0.l
0405F2  2F 39 00 04 0B C4             MOVE.l $40BC4.l,-(a7)
0405F8  2F 08                         MOVE.l a0,-(a7)
0405FA  4E B9 00 04 0F 48             JSR $40F48.l
040600  50 8F                         ADDQ.l #8,a7
040602  20 2E FF DC                   MOVE.l -$24(a6),d0
040606  4E 5E                         UNLK a6
040608  4E 75                         RTS

; ==== blk_unit  $04060A  (1 caller) — HUNK_UNIT ($3E7) arm handler: read and discard the unit longword. ====
04060A  4E 56 FF FC                   LINK a6,#-$4
04060E  48 6E FF FC                   PEA -$4(a6)
040612  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040616  61 00 FC 80                   BSR $040298
04061A  50 8F                         ADDQ.l #8,a7
04061C  20 2E FF FC                   MOVE.l -$4(a6),d0
040620  E5 80                         ASL.l #2,d0
040622  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040626  2F 00                         MOVE.l d0,-(a7)
040628  61 00 04 40                   BSR $040A6A
04062C  50 8F                         ADDQ.l #8,a7
04062E  4E 5E                         UNLK a6
040630  4E 75                         RTS

; ==== blk_name  $040632  (1 caller) — HUNK_NAME ($3E8) arm handler. ====
040632  4E 56 FF FC                   LINK a6,#-$4
040636  48 6E FF FC                   PEA -$4(a6)
04063A  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04063E  61 00 FC 58                   BSR $040298
040642  50 8F                         ADDQ.l #8,a7
040644  20 2E FF FC                   MOVE.l -$4(a6),d0
040648  E5 80                         ASL.l #2,d0
04064A  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04064E  2F 00                         MOVE.l d0,-(a7)
040650  61 00 04 18                   BSR $040A6A
040654  50 8F                         ADDQ.l #8,a7
040656  4E 5E                         UNLK a6
040658  4E 75                         RTS

; ==== blk_code  $04065A  (1 caller) — HUNK_CODE ($3E9) handler: read the size, get the hunk's memory (hunk_base), fillbytes the body (size*4 raw bytes), then XOR-decrypt each longword in place with the keystream. ====
04065A  4E 56 FF F0                   LINK a6,#-$10
04065E  48 E7 01 04                   MOVEM.l d7/a5,-(a7)
040662  48 6E FF FC                   PEA -$4(a6)
040666  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04066A  61 00 FC 2C                   BSR $040298
04066E  50 8F                         ADDQ.l #8,a7
040670  02 AE 3F FF FF FF FF FC       ANDI.l #$3FFFFFFF,-$4(a6)
040678  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
04067C  61 00 FA A2                   BSR $040120
040680  58 8F                         ADDQ.l #4,a7
040682  2A 40                         MOVEA.l d0,a5
040684  4A AE FF FC                   TST.l -$4(a6)
040688  67 16                         BEQ $0406A0
04068A  20 2E FF FC                   MOVE.l -$4(a6),d0
04068E  E5 80                         ASL.l #2,d0
040690  2F 00                         MOVE.l d0,-(a7)
040692  2F 0D                         MOVE.l a5,-(a7)
040694  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040698  61 00 FB 6E                   BSR $040208
04069C  4F EF 00 0C                   LEA $C(a7),a7
0406A0  4A B9 00 04 0B C8             TST.l $40BC8.l
0406A6  67 24                         BEQ $0406CC
0406A8  2E 2E FF FC                   MOVE.l -$4(a6),d7
0406AC  4A 87                         TST.l d7
0406AE  67 1C                         BEQ $0406CC
0406B0  20 4D                         MOVEA.l a5,a0
0406B2  58 8D                         ADDQ.l #4,a5
0406B4  2F 48 00 08                   MOVE.l a0,$8(a7)
0406B8  4E B9 00 04 0E AC             JSR $40EAC.l
0406BE  20 6F 00 08                   MOVEA.l $8(a7),a0
0406C2  B1 90                         EOR.l d0,(a0)
0406C4  53 87                         SUBQ.l #1,d7
0406C6  20 07                         MOVE.l d7,d0
0406C8  4A 80                         TST.l d0
0406CA  66 E4                         BNE $0406B0
0406CC  4C DF                         .dc.w $4CDF
0406CE  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu

; ==== blk_data  $0406D4  (1 caller) — HUNK_DATA ($3EA) handler: as blk_code, for data hunks. ====
0406D4  4E 56 FF F0                   LINK a6,#-$10
0406D8  48 E7 01 04                   MOVEM.l d7/a5,-(a7)
0406DC  48 6E FF FC                   PEA -$4(a6)
0406E0  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0406E4  61 00 FB B2                   BSR $040298
0406E8  50 8F                         ADDQ.l #8,a7
0406EA  02 AE 3F FF FF FF FF FC       ANDI.l #$3FFFFFFF,-$4(a6)
0406F2  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
0406F6  61 00 FA 28                   BSR $040120
0406FA  58 8F                         ADDQ.l #4,a7
0406FC  2A 40                         MOVEA.l d0,a5
0406FE  4A AE FF FC                   TST.l -$4(a6)
040702  67 16                         BEQ $04071A
040704  20 2E FF FC                   MOVE.l -$4(a6),d0
040708  E5 80                         ASL.l #2,d0
04070A  2F 00                         MOVE.l d0,-(a7)
04070C  2F 0D                         MOVE.l a5,-(a7)
04070E  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040712  61 00 FA F4                   BSR $040208
040716  4F EF 00 0C                   LEA $C(a7),a7
04071A  4A B9 00 04 0B C8             TST.l $40BC8.l
040720  67 24                         BEQ $040746
040722  2E 2E FF FC                   MOVE.l -$4(a6),d7
040726  4A 87                         TST.l d7
040728  67 1C                         BEQ $040746
04072A  20 4D                         MOVEA.l a5,a0
04072C  58 8D                         ADDQ.l #4,a5
04072E  2F 48 00 08                   MOVE.l a0,$8(a7)
040732  4E B9 00 04 0E AC             JSR $40EAC.l
040738  20 6F 00 08                   MOVEA.l $8(a7),a0
04073C  B1 90                         EOR.l d0,(a0)
04073E  53 87                         SUBQ.l #1,d7
040740  20 07                         MOVE.l d7,d0
040742  4A 80                         TST.l d0
040744  66 E4                         BNE $04072A
040746  4C DF                         .dc.w $4CDF
040748  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu

; ==== blk_bss  $04074E  (1 caller) — HUNK_BSS ($3EB) handler: read the size; no body (BSS is zero-filled, not stored). ====
04074E  4E 56 FF FC                   LINK a6,#-$4
040752  48 6E FF FC                   PEA -$4(a6)
040756  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04075A  61 00 FB 3C                   BSR $040298
04075E  50 8F                         ADDQ.l #8,a7
040760  4E 5E                         UNLK a6
040762  4E 75                         RTS

; ==== blk_reloc32  $040764  (1 caller) — HUNK_RELOC32 ($3EC) handler: loop { count; if 0 stop; target-hunk; count offsets } applying each 32-bit relocation (add the target hunk's base) into this hunk; also folds a word checksum into $40BD8. ====
040764  4E 56 FF E4                   LINK a6,#-$1C
040768  48 E7 07 00                   MOVEM.l d5-d7,-(a7)
04076C  48 6E FF F8                   PEA -$8(a6)
040770  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040774  61 00 FB 22                   BSR $040298
040778  50 8F                         ADDQ.l #8,a7
04077A  4A AE FF F8                   TST.l -$8(a6)
04077E  67 00 00 94                   BEQ $040814
040782  48 6E FF FC                   PEA -$4(a6)
040786  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04078A  61 00 FB 0C                   BSR $040298
04078E  50 8F                         ADDQ.l #8,a7
040790  2A 2E FF F8                   MOVE.l -$8(a6),d5
040794  4A 85                         TST.l d5
040796  67 6A                         BEQ $040802
040798  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
04079C  61 00 F9 82                   BSR $040120
0407A0  58 8F                         ADDQ.l #4,a7
0407A2  2E 00                         MOVE.l d0,d7
0407A4  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
0407A8  61 00 F9 76                   BSR $040120
0407AC  58 8F                         ADDQ.l #4,a7
0407AE  2C 00                         MOVE.l d0,d6
0407B0  48 6E FF F4                   PEA -$C(a6)
0407B4  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0407B8  61 00 FA DE                   BSR $040298
0407BC  50 8F                         ADDQ.l #8,a7
0407BE  20 07                         MOVE.l d7,d0
0407C0  D0 AE FF F4                   ADD.l -$C(a6),d0
0407C4  20 40                         MOVEA.l d0,a0
0407C6  C1 88                         EXG d0,a0
0407C8  02 40 FF FE                   ANDI.w #$FFFE,d0
0407CC  C1 88                         EXG d0,a0
0407CE  22 40                         MOVEA.l d0,a1
0407D0  C1 89                         EXG d0,a1
0407D2  02 40 FF FE                   ANDI.w #$FFFE,d0
0407D6  C1 89                         EXG d0,a1
0407D8  DD 91                         ADD.l d6,(a1)
0407DA  30 39 00 04 0B D8             MOVE.w $40BD8.l,d0
0407E0  32 10                         MOVE.w (a0),d1
0407E2  B3 40                         EOR.w d1,d0
0407E4  33 C0 00 04 0B D8             MOVE.w d0,$40BD8.l
0407EA  32 28 00 02                   MOVE.w $2(a0),d1
0407EE  B3 40                         EOR.w d1,d0
0407F0  33 C0 00 04 0B D8             MOVE.w d0,$40BD8.l
0407F6  2D 48 FF E4                   MOVE.l a0,-$1C(a6)
0407FA  53 85                         SUBQ.l #1,d5
0407FC  20 05                         MOVE.l d5,d0
0407FE  4A 80                         TST.l d0
040800  66 AE                         BNE $0407B0
040802  48 6E FF F8                   PEA -$8(a6)
040806  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04080A  61 00 FA 8C                   BSR $040298
04080E  50 8F                         ADDQ.l #8,a7
040810  60 00 FF 68                   BRA $04077A
040814  4C DF                         .dc.w $4CDF
040816  .dc.b 00 E0 4E 5E 4E 75                               ; ..N^Nu

; ==== blk_reloc16  $04081C  (1 caller) — HUNK_RELOC16 ($3ED) handler. ====
04081C  4E 56 FF FC                   LINK a6,#-$4
040820  48 6E FF FC                   PEA -$4(a6)
040824  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040828  61 00 FA 6E                   BSR $040298
04082C  50 8F                         ADDQ.l #8,a7
04082E  4A AE FF FC                   TST.l -$4(a6)
040832  67 24                         BEQ $040858
040834  20 2E FF FC                   MOVE.l -$4(a6),d0
040838  52 80                         ADDQ.l #1,d0
04083A  E5 80                         ASL.l #2,d0
04083C  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040840  2F 00                         MOVE.l d0,-(a7)
040842  61 00 02 26                   BSR $040A6A
040846  50 8F                         ADDQ.l #8,a7
040848  48 6E FF FC                   PEA -$4(a6)
04084C  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040850  61 00 FA 46                   BSR $040298
040854  50 8F                         ADDQ.l #8,a7
040856  60 D6                         BRA $04082E
040858  4E 5E                         UNLK a6
04085A  4E 75                         RTS

; ==== blk_reloc_a  $04085C  (1 caller) — relocation sub-handler (offset application). ====
04085C  4E 56 FF FC                   LINK a6,#-$4
040860  48 6E FF FC                   PEA -$4(a6)
040864  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040868  61 00 FA 2E                   BSR $040298
04086C  50 8F                         ADDQ.l #8,a7
04086E  4A AE FF FC                   TST.l -$4(a6)
040872  67 24                         BEQ $040898
040874  20 2E FF FC                   MOVE.l -$4(a6),d0
040878  52 80                         ADDQ.l #1,d0
04087A  E5 80                         ASL.l #2,d0
04087C  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040880  2F 00                         MOVE.l d0,-(a7)
040882  61 00 01 E6                   BSR $040A6A
040886  50 8F                         ADDQ.l #8,a7
040888  48 6E FF FC                   PEA -$4(a6)
04088C  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040890  61 00 FA 06                   BSR $040298
040894  50 8F                         ADDQ.l #8,a7
040896  60 D6                         BRA $04086E
040898  4E 5E                         UNLK a6
04089A  4E 75                         RTS

; ==== blk_reloc_b  $04089C  (1 caller) — relocation sub-handler. ====
04089C  4E 56 00 00                   LINK a6,#$0
0408A0  4E B9 00 04 00 3A             JSR $4003A.l
0408A6  4E 5E                         UNLK a6
0408A8  4E 75                         RTS

; ==== blk_symbol  $0408AA  (1 caller) — HUNK_SYMBOL ($3F0) handler: loop { name-length; read name (length longwords, RAW); value } until a 0 length. The names stay plaintext on disk. ====
0408AA  4E 56 FF FC                   LINK a6,#-$4
0408AE  48 6E FF FC                   PEA -$4(a6)
0408B2  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0408B6  61 00 F9 E0                   BSR $040298
0408BA  50 8F                         ADDQ.l #8,a7
0408BC  20 2E FF FC                   MOVE.l -$4(a6),d0
0408C0  E5 80                         ASL.l #2,d0
0408C2  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0408C6  2F 00                         MOVE.l d0,-(a7)
0408C8  61 00 01 A0                   BSR $040A6A
0408CC  50 8F                         ADDQ.l #8,a7
0408CE  48 6E FF FC                   PEA -$4(a6)
0408D2  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0408D6  61 00 F9 C0                   BSR $040298
0408DA  50 8F                         ADDQ.l #8,a7
0408DC  48 6E FF FC                   PEA -$4(a6)
0408E0  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0408E4  61 00 F9 B2                   BSR $040298
0408E8  50 8F                         ADDQ.l #8,a7
0408EA  4A AE FF FC                   TST.l -$4(a6)
0408EE  66 CC                         BNE $0408BC
0408F0  4E 5E                         UNLK a6
0408F2  4E 75                         RTS

; ==== blk_debug  $0408F4  (1 caller) — HUNK_DEBUG ($3F1) handler: read a length and skip that many longwords. ====
0408F4  4E 56 FF FC                   LINK a6,#-$4
0408F8  48 6E FF FC                   PEA -$4(a6)
0408FC  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040900  61 00 F9 96                   BSR $040298
040904  50 8F                         ADDQ.l #8,a7
040906  20 2E FF FC                   MOVE.l -$4(a6),d0
04090A  E5 80                         ASL.l #2,d0
04090C  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040910  2F 00                         MOVE.l d0,-(a7)
040912  61 00 01 56                   BSR $040A6A
040916  50 8F                         ADDQ.l #8,a7
040918  4E 5E                         UNLK a6
04091A  4E 75                         RTS

; ==== blk_end  $04091C  (1 caller) — HUNK_END ($3F2) handler: no-op (per-hunk terminator). ====
04091C  4E 56 00 00                   LINK a6,#$0
040920  4E 5E                         UNLK a6
040922  4E 75                         RTS

; ==== hdr_liblist  $040924  (1 caller) — HUNK_HEADER pre-pass: read the resident-library list (length-prefixed names, skipped RAW) and return the following table_size. ====
040924  4E 56 FF FC                   LINK a6,#-$4
040928  48 6E FF FC                   PEA -$4(a6)
04092C  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040930  61 00 F9 66                   BSR $040298
040934  50 8F                         ADDQ.l #8,a7
040936  4A AE FF FC                   TST.l -$4(a6)
04093A  67 22                         BEQ $04095E
04093C  48 6E FF FC                   PEA -$4(a6)
040940  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040944  61 00 F9 52                   BSR $040298
040948  50 8F                         ADDQ.l #8,a7
04094A  20 2E FF FC                   MOVE.l -$4(a6),d0
04094E  E5 80                         ASL.l #2,d0
040950  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040954  2F 00                         MOVE.l d0,-(a7)
040956  61 00 01 12                   BSR $040A6A
04095A  50 8F                         ADDQ.l #8,a7
04095C  60 D8                         BRA $040936
04095E  48 6E FF FC                   PEA -$4(a6)
040962  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040966  61 00 F9 30                   BSR $040298
04096A  50 8F                         ADDQ.l #8,a7
04096C  20 2E FF FC                   MOVE.l -$4(a6),d0
040970  4E 5E                         UNLK a6
040972  4E 75                         RTS

; ==== hdr_helper1  $040974  (1 caller) — HUNK_HEADER sub-helper. ====
040974  4E 56 FF FC                   LINK a6,#-$4
040978  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04097C  70 08                         MOVEQ #$8,d0
04097E  2F 00                         MOVE.l d0,-(a7)
040980  61 00 00 E8                   BSR $040A6A
040984  50 8F                         ADDQ.l #8,a7
040986  48 6E FF FC                   PEA -$4(a6)
04098A  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04098E  61 00 F9 08                   BSR $040298
040992  50 8F                         ADDQ.l #8,a7
040994  20 2E FF FC                   MOVE.l -$4(a6),d0
040998  E5 80                         ASL.l #2,d0
04099A  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
04099E  2F 00                         MOVE.l d0,-(a7)
0409A0  61 00 00 C8                   BSR $040A6A
0409A4  50 8F                         ADDQ.l #8,a7
0409A6  4E 5E                         UNLK a6
0409A8  4E 75                         RTS

; ==== hdr_helper2  $0409AA  (1 caller) — HUNK_HEADER sub-helper. ====
0409AA  4E 56 FF FC                   LINK a6,#-$4
0409AE  48 6E FF FC                   PEA -$4(a6)
0409B2  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0409B6  61 00 F8 E0                   BSR $040298
0409BA  50 8F                         ADDQ.l #8,a7
0409BC  20 2E FF FC                   MOVE.l -$4(a6),d0
0409C0  54 80                         ADDQ.l #2,d0
0409C2  E5 80                         ASL.l #2,d0
0409C4  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
0409CA  2F 00                         MOVE.l d0,-(a7)
0409CC  23 C0 00 04 0B D0             MOVE.l d0,$40BD0.l
0409D2  4E B9 00 04 0F 30             JSR $40F30.l
0409D8  50 8F                         ADDQ.l #8,a7
0409DA  20 40                         MOVEA.l d0,a0
0409DC  C1 88                         EXG d0,a0
0409DE  02 40 FF FE                   ANDI.w #$FFFE,d0
0409E2  C1 88                         EXG d0,a0
0409E4  23 C8 00 04 0B CC             MOVE.l a0,$40BCC.l
0409EA  58 88                         ADDQ.l #4,a0
0409EC  20 39 00 04 0B D0             MOVE.l $40BD0.l,d0
0409F2  59 80                         SUBQ.l #4,d0
0409F4  2F 00                         MOVE.l d0,-(a7)
0409F6  2F 08                         MOVE.l a0,-(a7)
0409F8  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0409FC  61 00 F8 0A                   BSR $040208
040A00  4F EF 00 0C                   LEA $C(a7),a7
040A04  20 79 00 04 0B CC             MOVEA.l $40BCC.l,a0
040A0A  20 B9 00 04 0B D0             MOVE.l $40BD0.l,(a0)
040A10  4E 5E                         UNLK a6
040A12  4E 75                         RTS

; ==== blk_misc  $040A14  (1 caller) — minor block/seglist helper. ====
040A14  4E 56 FF F0                   LINK a6,#-$10
040A18  2D 79 00 04 00 A4 FF F4       MOVE.l $400A4.l,-$C(a6)
040A20  20 6E FF F4                   MOVEA.l -$C(a6),a0
040A24  2D 68 00 26 FF F0             MOVE.l $26(a0),-$10(a6)
040A2A  42 A7                         CLR.l -(a7)
040A2C  61 00 F6 F2                   BSR $040120
040A30  58 8F                         ADDQ.l #4,a7
040A32  20 40                         MOVEA.l d0,a0
040A34  21 6E 00 08 00 08             MOVE.l $8(a6),$8(a0)
040A3A  22 79 00 04 0B CC             MOVEA.l $40BCC.l,a1
040A40  58 89                         ADDQ.l #4,a1
040A42  22 09                         MOVE.l a1,d1
040A44  21 41 00 0C                   MOVE.l d1,$C(a0)
040A48  22 39 00 04 0B C0             MOVE.l $40BC0.l,d1
040A4E  2D 41 FF F8                   MOVE.l d1,-$8(a6)
040A52  E4 81                         ASR.l #2,d1
040A54  21 41 00 10                   MOVE.l d1,$10(a0)
040A58  21 6E FF F0 00 14             MOVE.l -$10(a6),$14(a0)
040A5E  4E 5E                         UNLK a6
040A60  4E 75                         RTS

; ==== blk_idx15  $040A62  (1 caller) — arm 15 handler (block type $3F6). ====
040A62  4E 56 00 00                   LINK a6,#$0
040A66  4E 5E                         UNLK a6
040A68  4E 75                         RTS

; ==== read_skip  $040A6A  (9 callers) — read/consume N bytes through fillbytes in $80-byte chunks (used to skip library-name strings in the header). ====
040A6A  4E 56 00 00                   LINK a6,#$0
040A6E  4A AE 00 08                   TST.l $8(a6)
040A72  6E 06                         BGT $040A7A
040A74  70 00                         MOVEQ #$0,d0
040A76  4E 5E                         UNLK a6
040A78  4E 75                         RTS
040A7A  0C AE 00 00 00 80 00 08       CMPI.l #$80,$8(a6)
040A82  6F 22                         BLE $040AA6
040A84  2F 3C 00 00 00 80             MOVE.l #$80,-(a7)
040A8A  2F 39 00 04 0B D4             MOVE.l $40BD4.l,-(a7)
040A90  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
040A94  61 00 F7 72                   BSR $040208
040A98  4F EF 00 0C                   LEA $C(a7),a7
040A9C  04 AE 00 00 00 80 00 08       SUBI.l #$80,$8(a6)
040AA4  60 D4                         BRA $040A7A
040AA6  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040AAA  2F 39 00 04 0B D4             MOVE.l $40BD4.l,-(a7)
040AB0  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
040AB4  61 00 F7 52                   BSR $040208
040AB8  4F EF 00 0C                   LEA $C(a7),a7
040ABC  4E 5E                         UNLK a6
040ABE  4E 75                         RTS

; ==== hdr_alloc  $040AC0  (1 caller) — HUNK_HEADER allocation pass: read first/last hunk and the size table (each "size" carries a 2-bit mem-type tag in the top bits), and AllocMem every hunk via alloc_hunk, recording them in $40BC0. ====
040AC0  4E 56 FF E8                   LINK a6,#-$18
040AC4  20 2E 00 0C                   MOVE.l $C(a6),d0
040AC8  54 80                         ADDQ.l #2,d0
040ACA  E5 80                         ASL.l #2,d0
040ACC  23 C0 00 04 0B C4             MOVE.l d0,$40BC4.l
040AD2  4A AE 00 10                   TST.l $10(a6)
040AD6  66 38                         BNE $040B10
040AD8  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
040ADE  2F 00                         MOVE.l d0,-(a7)
040AE0  4E B9 00 04 0F 30             JSR $40F30.l
040AE6  50 8F                         ADDQ.l #8,a7
040AE8  20 40                         MOVEA.l d0,a0
040AEA  C1 88                         EXG d0,a0
040AEC  02 40 FF FE                   ANDI.w #$FFFE,d0
040AF0  C1 88                         EXG d0,a0
040AF2  2D 48 00 10                   MOVE.l a0,$10(a6)
040AF6  B1 FC 00 00 00 00             CMPA.l #$0,a0
040AFC  20 6E 00 10                   MOVEA.l $10(a6),a0
040B00  20 B9 00 04 0B C4             MOVE.l $40BC4.l,(a0)
040B06  58 88                         ADDQ.l #4,a0
040B08  23 C8 00 04 0B C0             MOVE.l a0,$40BC0.l
040B0E  60 0C                         BRA $040B1C
040B10  20 2E 00 10                   MOVE.l $10(a6),d0
040B14  E5 80                         ASL.l #2,d0
040B16  23 C0 00 04 0B C0             MOVE.l d0,$40BC0.l
040B1C  48 6E FF F8                   PEA -$8(a6)
040B20  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040B24  61 00 F7 72                   BSR $040298
040B28  50 8F                         ADDQ.l #8,a7
040B2A  48 6E FF F4                   PEA -$C(a6)
040B2E  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040B32  61 00 F7 64                   BSR $040298
040B36  50 8F                         ADDQ.l #8,a7
040B38  42 AE FF EC                   CLR.l -$14(a6)
040B3C  2D 6E FF F8 FF F0             MOVE.l -$8(a6),-$10(a6)
040B42  20 2E FF F0                   MOVE.l -$10(a6),d0
040B46  B0 AE FF F4                   CMP.l -$C(a6),d0
040B4A  6E 58                         BGT $040BA4
040B4C  48 6E FF FC                   PEA -$4(a6)
040B50  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040B54  61 00 F7 42                   BSR $040298
040B58  50 8F                         ADDQ.l #8,a7
040B5A  70 1E                         MOVEQ #$1E,d0
040B5C  22 2E FF FC                   MOVE.l -$4(a6),d1
040B60  E0 A1                         ASR.l d0,d1
040B62  02 81 00 00 00 03             ANDI.l #$3,d1
040B68  20 2E FF FC                   MOVE.l -$4(a6),d0
040B6C  02 80 3F FF FF FF             ANDI.l #$3FFFFFFF,d0
040B72  2D 40 FF FC                   MOVE.l d0,-$4(a6)
040B76  2D 41 FF E8                   MOVE.l d1,-$18(a6)
040B7A  E5 81                         ASL.l #2,d1
040B7C  20 41                         MOVEA.l d1,a0
040B7E  D1 FC 00 04 0B B0             ADDA.l #$40BB0,a0
040B84  2F 2E FF EC                   MOVE.l -$14(a6),-(a7)
040B88  2F 10                         MOVE.l (a0),-(a7)
040B8A  2F 00                         MOVE.l d0,-(a7)
040B8C  2F 2E FF F0                   MOVE.l -$10(a6),-(a7)
040B90  61 00 F5 E8                   BSR $04017A
040B94  4F EF 00 10                   LEA $10(a7),a7
040B98  70 01                         MOVEQ #$1,d0
040B9A  2D 40 FF EC                   MOVE.l d0,-$14(a6)
040B9E  52 AE FF F0                   ADDQ.l #1,-$10(a6)
040BA2  60 9E                         BRA $040B42
040BA4  20 2E FF F8                   MOVE.l -$8(a6),d0
040BA8  53 80                         SUBQ.l #1,d0
040BAA  4E 5E                         UNLK a6
040BAC  4E 75                         RTS
040BAE  .dc.b 00 00 00 01 00 01 00 01 00 02 00 01 00 04 00 01 ; ................
040BBE  .dc.b 00 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
040BCE  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
040BDE  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00       ; ..............

; ==== seed_table  $040BEC  (1 caller) — build the 55-entry table from seed $57319753 by the ×31 hash: table[0]=$57319753, table[i]=table[i-1]*31+i; record the table pointers in the globals ($40EA0/4/8) the keystream reads. ====
040BEC  4E 56 FF FC                   LINK a6,#-$4
040BF0  20 6E 00 08                   MOVEA.l $8(a6),a0
040BF4  20 BC 57 31 97 53             MOVE.l #$57319753,(a0)
040BFA  70 01                         MOVEQ #$1,d0
040BFC  2D 40 FF FC                   MOVE.l d0,-$4(a6)
040C00  20 2E FF FC                   MOVE.l -$4(a6),d0
040C04  0C 80 00 00 00 37             CMPI.l #$37,d0
040C0A  6C 22                         BGE $040C2E
040C0C  E5 80                         ASL.l #2,d0
040C0E  20 6E 00 08                   MOVEA.l $8(a6),a0
040C12  D1 C0                         ADDA.l d0,a0
040C14  59 80                         SUBQ.l #4,d0
040C16  22 6E 00 08                   MOVEA.l $8(a6),a1
040C1A  D3 C0                         ADDA.l d0,a1
040C1C  20 11                         MOVE.l (a1),d0
040C1E  EB 80                         ASL.l #5,d0
040C20  90 91                         SUB.l (a1),d0
040C22  D0 AE FF FC                   ADD.l -$4(a6),d0
040C26  20 80                         MOVE.l d0,(a0)
040C28  52 AE FF FC                   ADDQ.l #1,-$4(a6)
040C2C  60 D2                         BRA $040C00
040C2E  20 6E 00 08                   MOVEA.l $8(a6),a0
040C32  23 C8 00 04 0E A0             MOVE.l a0,$40EA0.l
040C38  D0 FC 00 6C                   ADDA.w #$6C,a0
040C3C  23 C8 00 04 0E A4             MOVE.l a0,$40EA4.l
040C42  20 6E 00 08                   MOVEA.l $8(a6),a0
040C46  D0 FC 00 DC                   ADDA.w #$DC,a0
040C4A  23 C8 00 04 0E A8             MOVE.l a0,$40EA8.l
040C50  4E 5E                         UNLK a6
040C52  4E 75                         RTS
040C54  .dc.b 4E 56 FF F4 2F 3C 00 01 00 02 2F 3C 00 00 00 DC ; NV../<..../<....
040C64  .dc.b 4E B9 00 04 0F 30 50 8F 2F 00 2D 40 FF F8 61 00 ; N....0P./.-@..a.
040C74  .dc.b FF 78 58 8F 0C AE 00 00 00 37 00 0C 6F 04 70 37 ; .xX......7..o.p7
040C84  .dc.b 60 04 20 2E 00 0C 42 AE FF FC 2D 40 00 0C 20 2E ; `. ...B...-@.. .
040C94  .dc.b FF FC B0 AE 00 0C 6C 18 E5 80 20 6E FF F8 D1 C0 ; ......l... n....
040CA4  .dc.b 22 6E 00 08 D3 C0 20 11 B1 90 52 AE FF FC 60 DE ; "n.... ...R...`.
040CB4  .dc.b 2F 2E FF F8 61 00 00 F0 58 8F 42 AE FF FC 20 2E ; /...a...X.B... .
040CC4  .dc.b FF FC B0 AE 00 14 6C 24 E5 80 20 6E 00 10 D1 C0 ; ......l$.. n....
040CD4  .dc.b 2F 2E FF F8 2F 48 00 04 4E B9 00 04 0E AC 58 8F ; /.../H..N.....X.
040CE4  .dc.b 20 6F 00 00 B1 90 52 AE FF FC 60 D2 2F 3C 00 00 ;  o....R...`./<..
040CF4  .dc.b 00 DC 2F 2E FF F8 4E B9 00 04 0F 48 50 8F 4E 5E ; ../...N....HP.N^
040D04  .dc.b 4E 75                                           ; Nu

; ==== key_init  $040D06  (1 caller) — AllocMem 220 bytes; seed_table; XOR in the caller's key array (count longwords from the control block); run the protection; return the table. ====
040D06  4E 56 FF F8                   LINK a6,#-$8
040D0A  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
040D10  2F 3C 00 00 00 DC             MOVE.l #$DC,-(a7)
040D16  4E B9 00 04 0F 30             JSR $40F30.l
040D1C  50 8F                         ADDQ.l #8,a7
040D1E  2F 00                         MOVE.l d0,-(a7)
040D20  2D 40 FF F8                   MOVE.l d0,-$8(a6)
040D24  61 00 FE C6                   BSR $040BEC
040D28  58 8F                         ADDQ.l #4,a7
040D2A  0C AE 00 00 00 37 00 0C       CMPI.l #$37,$C(a6)
040D32  6F 04                         BLE $040D38
040D34  70 37                         MOVEQ #$37,d0
040D36  60 04                         BRA $040D3C
040D38  20 2E 00 0C                   MOVE.l $C(a6),d0
040D3C  42 AE FF FC                   CLR.l -$4(a6)
040D40  2D 40 00 0C                   MOVE.l d0,$C(a6)
040D44  20 2E FF FC                   MOVE.l -$4(a6),d0
040D48  B0 AE 00 0C                   CMP.l $C(a6),d0
040D4C  6C 18                         BGE $040D66
040D4E  E5 80                         ASL.l #2,d0
040D50  20 6E FF F8                   MOVEA.l -$8(a6),a0
040D54  D1 C0                         ADDA.l d0,a0
040D56  22 6E 00 08                   MOVEA.l $8(a6),a1
040D5A  D3 C0                         ADDA.l d0,a1
040D5C  20 11                         MOVE.l (a1),d0
040D5E  B1 90                         EOR.l d0,(a0)
040D60  52 AE FF FC                   ADDQ.l #1,-$4(a6)
040D64  60 DE                         BRA $040D44
040D66  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
040D6A  61 00 00 3E                   BSR $040DAA
040D6E  58 8F                         ADDQ.l #4,a7
040D70  20 2E FF F8                   MOVE.l -$8(a6),d0
040D74  4E 5E                         UNLK a6
040D76  4E 75                         RTS

; ==== free_key  $040D78  (1 caller) — FreeMem the key table when decoding is done. ====
040D78  4E 56 00 00                   LINK a6,#$0
040D7C  2F 3C 00 00 00 DC             MOVE.l #$DC,-(a7)
040D82  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
040D86  4E B9 00 04 0F 48             JSR $40F48.l
040D8C  50 8F                         ADDQ.l #8,a7
040D8E  4E 5E                         UNLK a6
040D90  4E 75                         RTS

; ==== byte_hi  $040D92  (5 callers) — return (arg >> 16) & 0xFF — extracts a vector/handler page byte. ====
040D92  4E 56 00 00                   LINK a6,#$0
040D96  70 10                         MOVEQ #$10,d0
040D98  22 2E 00 08                   MOVE.l $8(a6),d1
040D9C  E0 A1                         ASR.l d0,d1
040D9E  02 81 00 00 00 FF             ANDI.l #$FF,d1
040DA4  20 01                         MOVE.l d1,d0
040DA6  4E 5E                         UNLK a6
040DA8  4E 75                         RTS

; ==== protection  $040DAA  (1 caller) — THE COPY PROTECTION: XOR byte_hi of the host's CPU exception/TRAP vectors (absolute low memory $8-$38, $80-$BC) into table entries 10-16/10-14/32-47, then FindTask and XOR byte_hi of the task's tc_ExceptCode ($2A) and tc_TrapCode ($32) into entries 30/31. Binds the keystream to live OS state. (Part III.) ====
040DAA  4E 56 FF EC                   LINK a6,#-$14
040DAE  91 C8                         SUBA.l a0,a0
040DB0  2F 08                         MOVE.l a0,-(a7)
040DB2  2D 48 FF FC                   MOVE.l a0,-$4(a6)
040DB6  4E B9 00 04 0F 60             JSR $40F60.l
040DBC  58 8F                         ADDQ.l #4,a7
040DBE  72 0A                         MOVEQ #$A,d1
040DC0  2D 41 FF F8                   MOVE.l d1,-$8(a6)
040DC4  2D 40 FF F4                   MOVE.l d0,-$C(a6)
040DC8  20 2E FF F8                   MOVE.l -$8(a6),d0
040DCC  0C 80 00 00 00 11             CMPI.l #$11,d0
040DD2  6C 2A                         BGE $040DFE
040DD4  E5 80                         ASL.l #2,d0
040DD6  20 6E 00 08                   MOVEA.l $8(a6),a0
040DDA  D1 C0                         ADDA.l d0,a0
040DDC  04 80 00 00 00 20             SUBI.l #$20,d0
040DE2  22 6E FF FC                   MOVEA.l -$4(a6),a1
040DE6  D3 C0                         ADDA.l d0,a1
040DE8  2F 11                         MOVE.l (a1),-(a7)
040DEA  2F 48 00 08                   MOVE.l a0,$8(a7)
040DEE  61 A2                         BSR $040D92
040DF0  58 8F                         ADDQ.l #4,a7
040DF2  20 6F 00 04                   MOVEA.l $4(a7),a0
040DF6  B1 90                         EOR.l d0,(a0)
040DF8  52 AE FF F8                   ADDQ.l #1,-$8(a6)
040DFC  60 CA                         BRA $040DC8
040DFE  70 0A                         MOVEQ #$A,d0
040E00  2D 40 FF F8                   MOVE.l d0,-$8(a6)
040E04  20 2E FF F8                   MOVE.l -$8(a6),d0
040E08  0C 80 00 00 00 0F             CMPI.l #$F,d0
040E0E  6C 26                         BGE $040E36
040E10  E5 80                         ASL.l #2,d0
040E12  20 6E 00 08                   MOVEA.l $8(a6),a0
040E16  D1 C0                         ADDA.l d0,a0
040E18  22 6E FF FC                   MOVEA.l -$4(a6),a1
040E1C  D3 C0                         ADDA.l d0,a1
040E1E  2F 11                         MOVE.l (a1),-(a7)
040E20  2F 48 00 08                   MOVE.l a0,$8(a7)
040E24  61 00 FF 6C                   BSR $040D92
040E28  58 8F                         ADDQ.l #4,a7
040E2A  20 6F 00 04                   MOVEA.l $4(a7),a0
040E2E  B1 90                         EOR.l d0,(a0)
040E30  52 AE FF F8                   ADDQ.l #1,-$8(a6)
040E34  60 CE                         BRA $040E04
040E36  70 20                         MOVEQ #$20,d0
040E38  2D 40 FF F8                   MOVE.l d0,-$8(a6)
040E3C  20 2E FF F8                   MOVE.l -$8(a6),d0
040E40  0C 80 00 00 00 30             CMPI.l #$30,d0
040E46  6C 26                         BGE $040E6E
040E48  E5 80                         ASL.l #2,d0
040E4A  20 6E 00 08                   MOVEA.l $8(a6),a0
040E4E  D1 C0                         ADDA.l d0,a0
040E50  22 6E FF FC                   MOVEA.l -$4(a6),a1
040E54  D3 C0                         ADDA.l d0,a1
040E56  2F 11                         MOVE.l (a1),-(a7)
040E58  2F 48 00 08                   MOVE.l a0,$8(a7)
040E5C  61 00 FF 34                   BSR $040D92
040E60  58 8F                         ADDQ.l #4,a7
040E62  20 6F 00 04                   MOVEA.l $4(a7),a0
040E66  B1 90                         EOR.l d0,(a0)
040E68  52 AE FF F8                   ADDQ.l #1,-$8(a6)
040E6C  60 CE                         BRA $040E3C
040E6E  20 6E FF F4                   MOVEA.l -$C(a6),a0
040E72  2F 28 00 2A                   MOVE.l $2A(a0),-(a7)
040E76  61 00 FF 1A                   BSR $040D92
040E7A  58 8F                         ADDQ.l #4,a7
040E7C  20 6E 00 08                   MOVEA.l $8(a6),a0
040E80  B1 A8 00 78                   EOR.l d0,$78(a0)
040E84  20 6E FF F4                   MOVEA.l -$C(a6),a0
040E88  2F 28 00 32                   MOVE.l $32(a0),-(a7)
040E8C  61 00 FF 04                   BSR $040D92
040E90  58 8F                         ADDQ.l #4,a7
040E92  E8 80                         ASR.l #4,d0
040E94  20 6E 00 08                   MOVEA.l $8(a6),a0
040E98  B1 A8 00 7C                   EOR.l d0,$7C(a0)
040E9C  4E 5E                         UNLK a6
040E9E  4E 75                         RTS
040EA0  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00             ; ............

; ==== keystream  $040EAC  (3 callers) — additive lagged-Fibonacci generator over the 55-entry table: read q-pointer, table[p]+=table[q], advance p by 1 entry and q by 2 (mod 220 bytes), return the updated table[p]. ====
040EAC  20 79 00 04 0E A0             MOVEA.l $40EA0.l,a0
040EB2  22 79 00 04 0E A4             MOVEA.l $40EA4.l,a1
040EB8  22 39 00 04 0E A8             MOVE.l $40EA8.l,d1
040EBE  20 11                         MOVE.l (a1),d0
040EC0  D1 98                         ADD.l d0,(a0)+
040EC2  B1 C1                         CMPA.l d1,a0
040EC4  66 04                         BNE $040ECA
040EC6  90 FC 00 DC                   SUBA.w #$DC,a0
040ECA  50 89                         ADDQ.l #8,a1
040ECC  B3 C1                         CMPA.l d1,a1
040ECE  65 04                         BCS $040ED4
040ED0  92 FC 00 DC                   SUBA.w #$DC,a1
040ED4  23 C8 00 04 0E A0             MOVE.l a0,$40EA0.l
040EDA  23 C9 00 04 0E A4             MOVE.l a1,$40EA4.l
040EE0  20 10                         MOVE.l (a0),d0
040EE2  4E 75                         RTS

; ==== _Open  $040EE4  (1 caller) — dos Open(name, mode) ====
040EE4  48 E7 20 02                   MOVEM.l d2/a6,-(a7)
040EE8  4C EF                         .dc.w $4CEF
040EEA  .dc.b 00 06 00 0C 2C 79 00 04 00 A4 4E AE FF E2 4C DF ; ....,y....N...L.
040EFA  .dc.b 40 04 4E 75 00 00                               ; @.Nu..

; ==== _Close  $040F00  (1 caller) — dos Close(handle) ====
040F00  2F 0E                         MOVE.l a6,-(a7)
040F02  22 2F 00 08                   MOVE.l $8(a7),d1
040F06  2C 79 00 04 00 A4             MOVEA.l $400A4.l,a6
040F0C  4E AE FF DC                   JSR -$24(a6)
040F10  2C 5F                         MOVEA.l (a7)+,a6
040F12  4E 75                         RTS

; ==== _Read  $040F14  (1 caller) — dos Read(handle, buf, length) ====
040F14  48 E7 30 02                   MOVEM.l d2-d3/a6,-(a7)
040F18  4C EF                         .dc.w $4CEF
040F1A  .dc.b 00 0E 00 10 2C 79 00 04 00 A4 4E AE FF D6 4C DF ; ....,y....N...L.
040F2A  .dc.b 40 0C 4E 75 00 00                               ; @.Nu..

; ==== _AllocMem  $040F30  (6 callers) — exec AllocMem(size, flags) ====
040F30  2F 0E                         MOVE.l a6,-(a7)
040F32  4C EF                         .dc.w $4CEF
040F34  .dc.b 00 03 00 08 2C 79 00 04 00 60 4E AE FF 3A 2C 5F ; ....,y...`N..:,_
040F44  .dc.b 4E 75 00 00                                     ; Nu..

; ==== _FreeMem  $040F48  (4 callers) — exec FreeMem(addr, size) ====
040F48  2F 0E                         MOVE.l a6,-(a7)
040F4A  22 6F 00 08                   MOVEA.l $8(a7),a1
040F4E  20 2F 00 0C                   MOVE.l $C(a7),d0
040F52  2C 79 00 04 00 60             MOVEA.l $40060.l,a6
040F58  4E AE FF 2E                   JSR -$D2(a6)
040F5C  2C 5F                         MOVEA.l (a7)+,a6
040F5E  4E 75                         RTS

; ==== _FindTask  $040F60  (1 caller) — exec FindTask(name|0) ====
040F60  2F 0E                         MOVE.l a6,-(a7)
040F62  22 6F 00 08                   MOVEA.l $8(a7),a1
040F66  2C 79 00 04 00 60             MOVEA.l $40060.l,a6
040F6C  4E AE FE DA                   JSR -$126(a6)
040F70  2C 5F                         MOVEA.l (a7)+,a6
040F72  4E 75                         RTS
