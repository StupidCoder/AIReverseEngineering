
; --- _ovstart  $070000 — overlay entry — the launcher enters here via call_seglist with d0=mode, a0=image filename. mode 1: set up graphics and display the splash; mode 0: tear the display back down. ---
070000  23 C0 00 07 00 A0             MOVE.l d0,$700A0.l
070006  23 C8 00 07 00 A4             MOVE.l a0,$700A4.l
07000C  23 F8 00 04 00 07 00 64       MOVE.l $4.w,$70064.l
070014  48 E7 7E FE                   MOVEM.l d1-d6/a0-a6,-(a7)
070018  23 CF 00 07 00 98             MOVE.l a7,$70098.l
07001E  2C 79 00 07 00 64             MOVEA.l $70064.l,a6
070024  61 00 00 20                   BSR $070046
070028  2F 39 00 07 00 A4             MOVE.l $700A4.l,-(a7)
07002E  2F 39 00 07 00 A0             MOVE.l $700A0.l,-(a7)
070034  4E B9 00 07 07 0A             JSR $7070A.l

; ==== _exit  $07003A  (1 caller) —  ====
07003A  2E 79 00 07 00 98             MOVEA.l $70098.l,a7
070040  4C DF                         .dc.w $4CDF
070042  .dc.b 7F 7E 4E 75                                     ; .~Nu

; ==== openDOS  $070046  (1 caller) — OpenLibrary("dos.library") into _DOSBase. ====
070046  43 F9 00 07 00 AC             LEA $700AC.l,a1
07004C  4E AE FE 68                   JSR -$198(a6)
070050  23 C0 00 07 00 A8             MOVE.l d0,$700A8.l
070056  67 00 00 04                   BEQ $07005C
07005A  4E 75                         RTS

; --- noDOS  $07005C — dos.library failed to open: abort. ---
07005C  70 01                         MOVEQ #$1,d0

; --- _XCEXIT  $07005E —  ---
07005E  60 DA                         BRA $07003A

; --- NULL  $070060 —  (data) ---
070060  .dc.b 00 00 00 00                                     ; ....

; --- _SysBase  $070064 —  (data) ---
070064  .dc.b 00 00 00 00                                     ; ....

; --- _SP  $070068 —  (data) ---
070068  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070078  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................

; --- _mbase  $070088 —  (data) ---
070088  .dc.b 00 00 00 00                                     ; ....

; --- _mnext  $07008C —  (data) ---
07008C  .dc.b 00 00 00 00                                     ; ....

; --- _msize  $070090 —  (data) ---
070090  .dc.b 00 00 00 00                                     ; ....

; --- _tsize  $070094 —  (data) ---
070094  .dc.b 00 00 00 00                                     ; ....

; --- StackFrame  $070098 —  (data) ---
070098  .dc.b 00 00 00 00                                     ; ....

; --- __oserr  $07009C —  (data) ---
07009C  .dc.b 00 00 00 00                                     ; ....

; --- dosCmdLen  $0700A0 —  (data) ---
0700A0  .dc.b 00 00 00 00                                     ; ....

; --- dosCmdBuf  $0700A4 —  (data) ---
0700A4  .dc.b 00 00 00 00                                     ; ....

; --- _DOSBase  $0700A8 —  (data) ---
0700A8  .dc.b 00 00 00 00                                     ; ....

; --- DOSName  $0700AC —  (data) ---
0700AC  .dc.b 64 6F 73 2E 6C 69 62 72 61 72 79 00             ; dos.library.

; --- _IffErr  $0700B8 —  (data) ---
0700B8  .dc.b 4E 56 FF FC 20 39 00 07 07 34 42 B9 00 07 07 34 ; NV.. 9...4B....4
0700C8  .dc.b 4E 5E 4E 75                                     ; N^Nu

; --- _GetFoILBM  $0700CC — open the First Of an ILBM FORM in the file (EA IFF reader). ---
0700CC  4E 56 FF 6E                   LINK a6,#-$92
0700D0  48 E7 00 20                   MOVEM.l a2,-(a7)
0700D4  20 6E 00 08                   MOVEA.l $8(a6),a0
0700D8  0C A8 49 4C 42 4D 00 1C       CMPI.l #$494C424D,$1C(a0)
0700E0  67 0A                         BEQ $0700EC
0700E2  70 00                         MOVEQ #$0,d0
0700E4  4C DF                         .dc.w $4CDF
0700E6  .dc.b 04 00 4E 5E 4E 75                               ; ..N^Nu
0700EC  22 6E 00 08                   MOVEA.l $8(a6),a1
0700F0  20 69 00 04                   MOVEA.l $4(a1),a0
0700F4  43 EE FF 72                   LEA -$8E(a6),a1
0700F8  70 65                         MOVEQ #$65,d0
0700FA  12 D8                         MOVE.b (a0)+,(a1)+
0700FC  51 C8 FF FC                   DBRA d0,$0700FA
070100  48 6E FF D8                   PEA -$28(a6)
070104  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
070108  4E B9 00 07 0C A4             JSR $70CA4.l
07010E  50 8F                         ADDQ.l #8,a7
070110  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070114  4A 80                         TST.l d0
070116  67 08                         BEQ $070120
070118  4C DF                         .dc.w $4CDF
07011A  .dc.b 04 00 4E 5E 4E 75                               ; ..N^Nu
070120  48 6E FF D8                   PEA -$28(a6)
070124  4E B9 00 07 11 70             JSR $71170.l
07012A  58 8F                         ADDQ.l #4,a7
07012C  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070130  72 20                         MOVEQ #$20,d1
070132  04 81 00 00 00 08             SUBI.l #$8,d1
070138  6B 00 01 B0                   BMI $0702EA
07013C  B0 BB 18 08                   CMP.l $8(pc,d1.l),d0
070140  66 F0                         BNE $070132
070142  4E FB 18 06                   JMP $6(pc,d1.l)
070146  .dc.b FF FF FF FF 60 00 01 98 42 4F 44 59 60 00 00 9E ; ....`...BODY`...
070156  .dc.b 43 4D 41 50 60 00 00 70 42 4D 48 44 60 00 00 02 ; CMAP`..pBMHD`...
070166  .dc.b 1D 7C 00 01 FF 82 41 EE FF 84 70 14 2F 00 2F 08 ; .|....A...p././.
070176  .dc.b 48 6E FF D8 4E B9 00 07 0F 12 4F EF 00 0C 72 00 ; Hn..N.....O...r.
070186  .dc.b 32 2E FF 84 23 C1 00 07 07 20 72 00 32 2E FF 86 ; 2...#.... r.2...
070196  .dc.b 23 C1 00 07 07 24 72 00 12 2E FF 8C 23 C1 00 07 ; #....$r.....#...
0701A6  .dc.b 07 28 2D 40 FF FC 0C 81 00 00 00 05 6F 00 01 36 ; .(-@........o..6
0701B6  .dc.b 70 05 1D 40 FF 8C 72 00 12 2E FF 8C 23 C1 00 07 ; p..@..r.....#...
0701C6  .dc.b 07 28 60 00 01 20 1D 7C 00 20 FF 83 41 EE FF 98 ; .(`.. .|. ..A...
0701D6  .dc.b 48 6E FF 83 2F 08 48 6E FF D8 4E B9 00 07 08 78 ; Hn../.Hn..N....x
0701E6  .dc.b 4F EF 00 0C 2D 40 FF FC 60 00 00 FA 4A 2E FF 82 ; O...-@..`...J...
0701F6  .dc.b 66 0A 70 F9 4C DF 04 00 4E 5E 4E 75 2F 39 00 07 ; f.p.L...N^Nu/9..
070206  .dc.b 07 24 2F 39 00 07 07 20 2F 39 00 07 07 28 2F 39 ; .$/9... /9...(/9
070216  .dc.b 00 07 07 78 4E B9 00 07 1F A8 4F EF 00 10 20 39 ; ...xN.....O... 9
070226  .dc.b 00 07 07 20 72 08 4E B9 00 07 20 18 22 39 00 07 ; ... r.N... ."9..
070236  .dc.b 07 24 4E B9 00 07 20 5C 23 C0 00 07 07 2C 22 39 ; .$N... \#....,"9
070246  .dc.b 00 07 07 28 4E B9 00 07 20 5C 72 02 2F 01 2F 00 ; ...(N... \r././.
070256  .dc.b 4E B9 00 07 1E 10 50 8F 20 79 00 07 07 78 21 40 ; N.....P. y...x!@
070266  .dc.b 00 08 4A 80 67 42 70 01 2D 40 FF 6E 20 2E FF 6E ; ..J.gBp.-@.n ..n
070276  .dc.b B0 B9 00 07 07 28 6C 30 E5 80 20 79 00 07 07 78 ; .....(l0.. y...x
070286  .dc.b 50 88 D1 C0 20 39 00 07 07 2C 22 2E FF 6E 4E B9 ; P... 9...,"..nN.
070296  .dc.b 00 07 20 5C 24 79 00 07 07 78 22 6A 00 08 D3 C0 ; .. \$y...x"j....
0702A6  .dc.b 20 89 52 AE FF 6E 60 C4 2F 39 00 07 07 84 2F 39 ;  .R..n`./9..../9
0702B6  .dc.b 00 07 07 80 48 6E FF 84 42 A7 2F 39 00 07 07 78 ; ....Hn..B./9...x
0702C6  .dc.b 48 6E FF D8 4E B9 00 07 09 0E 4F EF 00 18 2D 40 ; Hn..N.....O...-@
0702D6  .dc.b FF FC 4A 80 66 0E 70 FE 2D 40 FF FC 60 06 70 F9 ; ..J.f.p.-@..`.p.
0702E6  .dc.b 2D 40 FF FC                                     ; -@..
0702EA  4A AE FF FC                   TST.l -$4(a6)
0702EE  6A 00 FE 30                   BPL $070120
0702F2  20 2E FF FC                   MOVE.l -$4(a6),d0
0702F6  0C 80 FF FF FF FE             CMPI.l #$FFFFFFFE,d0
0702FC  67 08                         BEQ $070306
0702FE  4C DF                         .dc.w $4CDF
070300  .dc.b 04 00 4E 5E 4E 75                               ; ..N^Nu
070306  48 6E FF D8                   PEA -$28(a6)
07030A  4E B9 00 07 0D 14             JSR $70D14.l
070310  58 8F                         ADDQ.l #4,a7
070312  70 00                         MOVEQ #$0,d0
070314  10 2E FF 83                   MOVE.b -$7D(a6),d0
070318  23 C0 00 07 07 30             MOVE.l d0,$70730.l
07031E  42 AE FF 6E                   CLR.l -$92(a6)
070322  20 2E FF 6E                   MOVE.l -$92(a6),d0
070326  B0 B9 00 07 07 30             CMP.l $70730.l,d0
07032C  6C 14                         BGE $070342
07032E  E3 80                         ASL.l #1,d0
070330  20 79 00 07 07 7C             MOVEA.l $7077C.l,a0
070336  D1 C0                         ADDA.l d0,a0
070338  30 B6 08 98                   MOVE.w -$68(a6,d0.l),(a0)
07033C  52 AE FF 6E                   ADDQ.l #1,-$92(a6)
070340  60 E0                         BRA $070322
070342  20 2E FF FC                   MOVE.l -$4(a6),d0
070346  4C DF                         .dc.w $4CDF
070348  .dc.b 04 00 4E 5E 4E 75                               ; ..N^Nu

; --- _GetPrILBM  $07034E — get the PROPerties of an ILBM (BMHD/CMAP). ---
07034E  4E 56 FF D4                   LINK a6,#-$2C
070352  20 6E 00 08                   MOVEA.l $8(a6),a0
070356  2D 68 00 04 FF D4             MOVE.l $4(a0),-$2C(a6)
07035C  0C A8 49 4C 42 4D 00 1C       CMPI.l #$494C424D,$1C(a0)
070364  67 06                         BEQ $07036C
070366  70 00                         MOVEQ #$0,d0
070368  4E 5E                         UNLK a6
07036A  4E 75                         RTS
07036C  48 6E FF D8                   PEA -$28(a6)
070370  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
070374  4E B9 00 07 0C A4             JSR $70CA4.l
07037A  50 8F                         ADDQ.l #8,a7
07037C  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070380  4A 80                         TST.l d0
070382  67 04                         BEQ $070388
070384  4E 5E                         UNLK a6
070386  4E 75                         RTS
070388  48 6E FF D8                   PEA -$28(a6)
07038C  4E B9 00 07 12 32             JSR $71232.l
070392  58 8F                         ADDQ.l #4,a7
070394  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070398  0C 80 43 4D 41 50             CMPI.l #$434D4150,d0
07039E  67 32                         BEQ $0703D2
0703A0  0C 80 42 4D 48 44             CMPI.l #$424D4844,d0
0703A6  66 58                         BNE $070400
0703A8  20 6E FF D4                   MOVEA.l -$2C(a6),a0
0703AC  11 7C 00 01 00 10             MOVE.b #$1,$10(a0)
0703B2  D0 FC 00 12                   ADDA.w #$12,a0
0703B6  20 08                         MOVE.l a0,d0
0703B8  72 14                         MOVEQ #$14,d1
0703BA  2F 01                         MOVE.l d1,-(a7)
0703BC  2F 00                         MOVE.l d0,-(a7)
0703BE  48 6E FF D8                   PEA -$28(a6)
0703C2  4E B9 00 07 0F 12             JSR $70F12.l
0703C8  4F EF 00 0C                   LEA $C(a7),a7
0703CC  2D 40 FF FC                   MOVE.l d0,-$4(a6)
0703D0  60 2E                         BRA $070400
0703D2  20 6E FF D4                   MOVEA.l -$2C(a6),a0
0703D6  11 7C 00 20 00 11             MOVE.b #$20,$11(a0)
0703DC  D0 FC 00 26                   ADDA.w #$26,a0
0703E0  20 08                         MOVE.l a0,d0
0703E2  20 6E FF D4                   MOVEA.l -$2C(a6),a0
0703E6  D0 FC 00 11                   ADDA.w #$11,a0
0703EA  2F 08                         MOVE.l a0,-(a7)
0703EC  2F 00                         MOVE.l d0,-(a7)
0703EE  48 6E FF D8                   PEA -$28(a6)
0703F2  4E B9 00 07 08 78             JSR $70878.l
0703F8  4F EF 00 0C                   LEA $C(a7),a7
0703FC  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070400  4A AE FF FC                   TST.l -$4(a6)
070404  6A 82                         BPL $070388
070406  48 6E FF D8                   PEA -$28(a6)
07040A  4E B9 00 07 0D 14             JSR $70D14.l
070410  58 8F                         ADDQ.l #4,a7
070412  0C AE FF FF FF FF FF FC       CMPI.l #$FFFFFFFF,-$4(a6)
07041A  66 04                         BNE $070420
07041C  70 00                         MOVEQ #$0,d0
07041E  60 04                         BRA $070424
070420  20 2E FF FC                   MOVE.l -$4(a6),d0
070424  4E 5E                         UNLK a6
070426  4E 75                         RTS

; --- _GetLiILBM  $070428 — iterate the chunks of an ILBM LIST. ---
070428  4E 56 FF 9A                   LINK a6,#-$66
07042C  22 6E 00 08                   MOVEA.l $8(a6),a1
070430  20 69 00 04                   MOVEA.l $4(a1),a0
070434  43 EE FF 9A                   LEA -$66(a6),a1
070438  70 65                         MOVEQ #$65,d0
07043A  12 D8                         MOVE.b (a0)+,(a1)+
07043C  51 C8 FF FC                   DBRA d0,$07043A
070440  41 EE FF 9A                   LEA -$66(a6),a0
070444  2F 08                         MOVE.l a0,-(a7)
070446  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
07044A  4E B9 00 07 10 36             JSR $71036.l
070450  50 8F                         ADDQ.l #8,a7
070452  4E 5E                         UNLK a6
070454  4E 75                         RTS

; ==== _GetPicture  $070456  (1 caller) — load an ILBM into memory: open the IFF, walk it for BMHD/CMAP/BODY, and fill the bitmap planes. ====
070456  4E 56 FF 9A                   LINK a6,#-$66
07045A  41 FA FF CC                   LEA $070428(pc),a0
07045E  2D 48 FF 9A                   MOVE.l a0,-$66(a6)
070462  41 FA FE EA                   LEA $07034E(pc),a0
070466  2D 48 FF 9E                   MOVE.l a0,-$62(a6)
07046A  41 FA FC 60                   LEA $0700CC(pc),a0
07046E  2D 48 FF A2                   MOVE.l a0,-$5E(a6)
070472  2D 7C 00 07 11 5C FF A6       MOVE.l #$7115C,-$5A(a6)
07047A  70 00                         MOVEQ #$0,d0
07047C  1D 40 FF AA                   MOVE.b d0,-$56(a6)
070480  1D 40 FF AB                   MOVE.b d0,-$55(a6)
070484  23 EE 00 0C 00 07 07 78       MOVE.l $C(a6),$70778.l
07048C  23 EE 00 10 00 07 07 7C       MOVE.l $10(a6),$7077C.l
070494  23 EE 00 14 00 07 07 80       MOVE.l $14(a6),$70780.l
07049C  23 EE 00 18 00 07 07 84       MOVE.l $18(a6),$70784.l
0704A4  41 EE FF 9A                   LEA -$66(a6),a0
0704A8  2F 08                         MOVE.l a0,-(a7)
0704AA  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0704AE  4E B9 00 07 0F 94             JSR $70F94.l
0704B4  50 8F                         ADDQ.l #8,a7
0704B6  23 C0 00 07 07 34             MOVE.l d0,$70734.l
0704BC  54 80                         ADDQ.l #2,d0
0704BE  57 C1                         SEQ d1
0704C0  44 01                         NEG.b d1
0704C2  48 81                         EXT.W d1
0704C4  48 C1                         EXT.L d1
0704C6  20 01                         MOVE.l d1,d0
0704C8  4E 5E                         UNLK a6
0704CA  4E 75                         RTS

; ==== _main0  $0704CC  (1 caller) — C main wrapper. ====
0704CC  4E 56 FF F8                   LINK a6,#-$8
0704D0  4A AE 00 08                   TST.l $8(a6)
0704D4  67 00 01 C6                   BEQ $07069C
0704D8  42 A7                         CLR.l -(a7)
0704DA  48 79 00 07 07 88             PEA $70788.l
0704E0  4E B9 00 07 1E 7C             JSR $71E7C.l
0704E6  50 8F                         ADDQ.l #8,a7
0704E8  23 C0 00 07 07 9C             MOVE.l d0,$7079C.l
0704EE  4A 80                         TST.l d0
0704F0  66 0C                         BNE $0704FE
0704F2  70 01                         MOVEQ #$1,d0
0704F4  2F 00                         MOVE.l d0,-(a7)
0704F6  4E B9 00 07 00 3A             JSR $7003A.l
0704FC  58 8F                         ADDQ.l #4,a7
0704FE  20 79 00 07 07 9C             MOVEA.l $7079C.l,a0
070504  23 E8 00 22 00 07 08 72       MOVE.l $22(a0),$70872.l
07050C  33 FC 01 00 00 DF F0 96       MOVE.w #$100,$DFF096.l
070514  2F 3C 00 00 03 ED             MOVE.l #$3ED,-(a7)
07051A  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
07051E  4E B9 00 07 1D A8             JSR $71DA8.l
070524  50 8F                         ADDQ.l #8,a7
070526  72 01                         MOVEQ #$1,d1
070528  2F 01                         MOVE.l d1,-(a7)
07052A  2F 3C 00 00 01 2C             MOVE.l #$12C,-(a7)
070530  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070534  4E B9 00 07 1E 10             JSR $71E10.l
07053A  50 8F                         ADDQ.l #8,a7
07053C  2D 40 FF F8                   MOVE.l d0,-$8(a6)
070540  4A AE FF FC                   TST.l -$4(a6)
070544  67 24                         BEQ $07056A
070546  4A 80                         TST.l d0
070548  67 20                         BEQ $07056A
07054A  2F 3C 00 00 01 2C             MOVE.l #$12C,-(a7)
070550  2F 00                         MOVE.l d0,-(a7)
070552  48 79 00 07 07 38             PEA $70738.l
070558  48 79 00 07 08 04             PEA $70804.l
07055E  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
070562  61 00 FE F2                   BSR $070456
070566  4F EF 00 14                   LEA $14(a7),a7
07056A  4A AE FF F8                   TST.l -$8(a6)
07056E  67 12                         BEQ $070582
070570  2F 3C 00 00 01 2C             MOVE.l #$12C,-(a7)
070576  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
07057A  4E B9 00 07 1E 28             JSR $71E28.l
070580  50 8F                         ADDQ.l #8,a7
070582  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
070586  4E B9 00 07 1D C4             JSR $71DC4.l
07058C  58 8F                         ADDQ.l #4,a7
07058E  4A B9 00 07 08 0C             TST.l $7080C.l
070594  67 00 01 70                   BEQ $070706
070598  48 79 00 07 08 38             PEA $70838.l
07059E  4E B9 00 07 1F 94             JSR $71F94.l
0705A4  58 8F                         ADDQ.l #4,a7
0705A6  23 FC 00 07 08 4A 00 07 08 38 MOVE.l #$7084A,$70838.l
0705B0  48 79 00 07 07 A0             PEA $707A0.l
0705B6  4E B9 00 07 1F 0C             JSR $71F0C.l
0705BC  58 8F                         ADDQ.l #4,a7
0705BE  41 F9 00 07 08 04             LEA $70804.l,a0
0705C4  23 C8 00 07 07 A4             MOVE.l a0,$707A4.l
0705CA  23 C8 00 07 08 30             MOVE.l a0,$70830.l
0705D0  70 00                         MOVEQ #$0,d0
0705D2  33 C0 00 07 08 34             MOVE.w d0,$70834.l
0705D8  33 C0 00 07 08 36             MOVE.w d0,$70836.l
0705DE  48 79 00 07 08 4A             PEA $7084A.l
0705E4  4E B9 00 07 1F 20             JSR $71F20.l
0705EA  58 8F                         ADDQ.l #4,a7
0705EC  20 39 00 07 07 20             MOVE.l $70720.l,d0
0705F2  33 C0 00 07 08 62             MOVE.w d0,$70862.l
0705F8  22 39 00 07 07 24             MOVE.l $70724.l,d1
0705FE  33 C1 00 07 08 64             MOVE.w d1,$70864.l
070604  0C 80 00 00 01 40             CMPI.l #$140,d0
07060A  6E 08                         BGT $070614
07060C  42 79 00 07 08 6A             CLR.w $7086A.l
070612  60 08                         BRA $07061C
070614  33 FC 80 00 00 07 08 6A       MOVE.w #$8000,$7086A.l
07061C  0C B9 00 00 00 C8 00 07 07 24 CMPI.l #$C8,$70724.l
070626  6F 08                         BLE $070630
070628  33 FC 00 04 00 07 08 48       MOVE.w #$4,$70848.l
070630  23 FC 00 07 08 2C 00 07 08 6E MOVE.l #$7082C,$7086E.l
07063A  48 79 00 07 08 4A             PEA $7084A.l
070640  48 79 00 07 08 38             PEA $70838.l
070646  4E B9 00 07 1F 48             JSR $71F48.l
07064C  50 8F                         ADDQ.l #8,a7
07064E  48 79 00 07 08 38             PEA $70838.l
070654  4E B9 00 07 1F 34             JSR $71F34.l
07065A  58 8F                         ADDQ.l #4,a7
07065C  48 79 00 07 08 38             PEA $70838.l
070662  4E B9 00 07 1F 60             JSR $71F60.l
070668  58 8F                         ADDQ.l #4,a7
07066A  4E B9 00 07 1F 74             JSR $71F74.l
070670  4E B9 00 07 1F 84             JSR $71F84.l
070676  2F 39 00 07 07 30             MOVE.l $70730.l,-(a7)
07067C  48 79 00 07 07 38             PEA $70738.l
070682  48 79 00 07 08 4A             PEA $7084A.l
070688  4E B9 00 07 1E F0             JSR $71EF0.l
07068E  4F EF 00 0C                   LEA $C(a7),a7
070692  33 FC 81 00 00 DF F0 96       MOVE.w #$8100,$DFF096.l
07069A  60 6A                         BRA $070706
07069C  33 FC 01 00 00 DF F0 96       MOVE.w #$100,$DFF096.l
0706A4  2F 39 00 07 08 72             MOVE.l $70872.l,-(a7)
0706AA  4E B9 00 07 1F 60             JSR $71F60.l
0706B0  58 8F                         ADDQ.l #4,a7
0706B2  4A B9 00 07 08 0C             TST.l $7080C.l
0706B8  67 3E                         BEQ $0706F8
0706BA  20 39 00 07 07 28             MOVE.l $70728.l,d0
0706C0  22 39 00 07 07 2C             MOVE.l $7072C.l,d1
0706C6  4E B9 00 07 20 5C             JSR $7205C.l
0706CC  2F 00                         MOVE.l d0,-(a7)
0706CE  2F 39 00 07 08 0C             MOVE.l $7080C.l,-(a7)
0706D4  4E B9 00 07 1E 28             JSR $71E28.l
0706DA  50 8F                         ADDQ.l #8,a7
0706DC  48 79 00 07 08 4A             PEA $7084A.l
0706E2  4E B9 00 07 1F C8             JSR $71FC8.l
0706E8  58 8F                         ADDQ.l #4,a7
0706EA  2F 39 00 07 08 3C             MOVE.l $7083C.l,-(a7)
0706F0  4E B9 00 07 1F DC             JSR $71FDC.l
0706F6  58 8F                         ADDQ.l #4,a7
0706F8  2F 39 00 07 07 9C             MOVE.l $7079C.l,-(a7)
0706FE  4E B9 00 07 1E 94             JSR $71E94.l
070704  58 8F                         ADDQ.l #4,a7
070706  4E 5E                         UNLK a6
070708  4E 75                         RTS

; ==== _main  $07070A  (1 caller) — boot-screen driver: open the libraries, load the picture (_GetPicture), build a viewport/bitmap and show it (graphics calls below). ====
07070A  4E 56 00 00                   LINK a6,#$0
07070E  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
070712  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
070716  61 00 FD B4                   BSR $0704CC
07071A  50 8F                         ADDQ.l #8,a7
07071C  4E 5E                         UNLK a6
07071E  4E 75                         RTS

; --- _WIDTH  $070720 —  (data) ---
070720  .dc.b 00 00 00 00                                     ; ....

; --- _HEIGHT  $070724 —  (data) ---
070724  .dc.b 00 00 00 00                                     ; ....

; --- _DEPTH  $070728 —  (data) ---
070728  .dc.b 00 00 00 00                                     ; ....

; --- _plsize  $07072C —  (data) ---
07072C  .dc.b 00 00 00 00                                     ; ....

; --- _aqNColorReg  $070730 —  (data) ---
070730  .dc.b 00 00 00 00 00 00 00 00                         ; ........

; --- _colors  $070738 —  (data) ---
070738  .dc.b 00 00 00 0A 00 A0 00 CA 0A 00 0A 0A 0C A0 0A AA ; ................
070748  .dc.b 08 88 00 0F 00 F0 00 FF 0F 00 0F 0F 0F F0 0F FF ; ................
070758  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070768  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070778  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070788  .dc.b 67 72 61 70 68 69 63 73 2E 6C 69 62 72 61 72 79 ; graphics.library
070798  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0707A8  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0707B8  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0707C8  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0707D8  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0707E8  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0707F8  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070808  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070818  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070828  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070838  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070848  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070858  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
070868  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................

; ==== _GetCMAP  $070878  (1 caller) — read the CMAP chunk — the colour map — into _colors. ====
070878  4E 56 FF F4                   LINK a6,#-$C
07087C  48 E7 03 00                   MOVEM.l d6-d7,-(a7)
070880  20 6E 00 08                   MOVEA.l $8(a6),a0
070884  20 28 00 18                   MOVE.l $18(a0),d0
070888  72 03                         MOVEQ #$3,d1
07088A  4E B9 00 07 20 18             JSR $72018.l
070890  2E 00                         MOVE.l d0,d7
070892  70 00                         MOVEQ #$0,d0
070894  20 6E 00 10                   MOVEA.l $10(a6),a0
070898  10 10                         MOVE.b (a0),d0
07089A  B0 87                         CMP.l d7,d0
07089C  64 04                         BCC $0708A2
07089E  7E 00                         MOVEQ #$0,d7
0708A0  1E 10                         MOVE.b (a0),d7
0708A2  20 07                         MOVE.l d7,d0
0708A4  20 6E 00 10                   MOVEA.l $10(a6),a0
0708A8  10 80                         MOVE.b d0,(a0)
0708AA  4A 87                         TST.l d7
0708AC  6F 56                         BLE $070904
0708AE  41 EE FF F5                   LEA -$B(a6),a0
0708B2  70 03                         MOVEQ #$3,d0
0708B4  2F 00                         MOVE.l d0,-(a7)
0708B6  2F 08                         MOVE.l a0,-(a7)
0708B8  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0708BC  4E B9 00 07 0F 12             JSR $70F12.l
0708C2  4F EF 00 0C                   LEA $C(a7),a7
0708C6  2C 00                         MOVE.l d0,d6
0708C8  4A 86                         TST.l d6
0708CA  67 0A                         BEQ $0708D6
0708CC  20 06                         MOVE.l d6,d0
0708CE  4C DF                         .dc.w $4CDF
0708D0  .dc.b 00 C0 4E 5E 4E 75                               ; ..N^Nu
0708D6  20 6E 00 0C                   MOVEA.l $C(a6),a0
0708DA  54 AE 00 0C                   ADDQ.l #2,$C(a6)
0708DE  70 00                         MOVEQ #$0,d0
0708E0  10 2E FF F5                   MOVE.b -$B(a6),d0
0708E4  E8 88                         LSR.l #4,d0
0708E6  E1 80                         ASL.l #8,d0
0708E8  72 00                         MOVEQ #$0,d1
0708EA  12 2E FF F6                   MOVE.b -$A(a6),d1
0708EE  E8 89                         LSR.l #4,d1
0708F0  E9 81                         ASL.l #4,d1
0708F2  80 81                         OR.l d1,d0
0708F4  72 00                         MOVEQ #$0,d1
0708F6  12 2E FF F7                   MOVE.b -$9(a6),d1
0708FA  E8 89                         LSR.l #4,d1
0708FC  80 81                         OR.l d1,d0
0708FE  30 80                         MOVE.w d0,(a0)
070900  53 87                         SUBQ.l #1,d7
070902  60 A6                         BRA $0708AA
070904  70 00                         MOVEQ #$0,d0
070906  4C DF                         .dc.w $4CDF
070908  .dc.b 00 C0 4E 5E 4E 75                               ; ..N^Nu

; --- _GetBODY  $07090E — read the BODY chunk: per-row, per-plane, decompressing with _UnPackRow into the bitplanes. ---
07090E  4E 56 FF 88                   LINK a6,#-$78
070912  48 E7 2F 00                   MOVEM.l d2/d4-d7,-(a7)
070916  20 6E 00 14                   MOVEA.l $14(a6),a0
07091A  1D 68 00 08 FF FB             MOVE.b $8(a0),-$5(a6)
070920  70 00                         MOVEQ #$0,d0
070922  30 10                         MOVE.w (a0),d0
070924  06 80 00 00 00 0F             ADDI.l #$F,d0
07092A  E8 88                         LSR.l #4,d0
07092C  E3 80                         ASL.l #1,d0
07092E  2D 40 FF F6                   MOVE.l d0,-$A(a6)
070932  06 80 00 00 00 7F             ADDI.l #$7F,d0
070938  EE 80                         ASR.l #7,d0
07093A  22 2E FF F6                   MOVE.l -$A(a6),d1
07093E  D2 80                         ADD.l d0,d1
070940  70 00                         MOVEQ #$0,d0
070942  30 28 00 02                   MOVE.w $2(a0),d0
070946  14 28 00 0A                   MOVE.b $A(a0),d2
07094A  2D 40 FF EE                   MOVE.l d0,-$12(a6)
07094E  2D 41 FF F2                   MOVE.l d1,-$E(a6)
070952  1D 42 FF ED                   MOVE.b d2,-$13(a6)
070956  0C 02 00 01                   CMPI.b #$1,d2
07095A  63 0A                         BLS $070966
07095C  70 FA                         MOVEQ #$FA,d0
07095E  4C DF                         .dc.w $4CDF
070960  .dc.b 00 F4 4E 5E 4E 75                               ; ..N^Nu
070966  70 00                         MOVEQ #$0,d0
070968  20 6E 00 0C                   MOVEA.l $C(a6),a0
07096C  30 10                         MOVE.w (a0),d0
07096E  22 2E FF F6                   MOVE.l -$A(a6),d1
070972  B2 80                         CMP.l d0,d1
070974  66 16                         BNE $07098C
070976  20 2E FF F2                   MOVE.l -$E(a6),d0
07097A  E3 80                         ASL.l #1,d0
07097C  22 2E 00 1C                   MOVE.l $1C(a6),d1
070980  B2 80                         CMP.l d0,d1
070982  6D 08                         BLT $07098C
070984  0C 2E 00 11 FF FB             CMPI.b #$11,-$5(a6)
07098A  63 0A                         BLS $070996
07098C  70 FA                         MOVEQ #$FA,d0
07098E  4C DF                         .dc.w $4CDF
070990  .dc.b 00 F4 4E 5E 4E 75                               ; ..N^Nu
070996  70 00                         MOVEQ #$0,d0
070998  20 6E 00 0C                   MOVEA.l $C(a6),a0
07099C  30 28 00 02                   MOVE.w $2(a0),d0
0709A0  22 2E FF EE                   MOVE.l -$12(a6),d1
0709A4  B2 80                         CMP.l d0,d1
0709A6  63 0A                         BLS $0709B2
0709A8  72 00                         MOVEQ #$0,d1
0709AA  32 28 00 02                   MOVE.w $2(a0),d1
0709AE  2D 41 FF EE                   MOVE.l d1,-$12(a6)
0709B2  7C 00                         MOVEQ #$0,d6
0709B4  70 00                         MOVEQ #$0,d0
0709B6  20 6E 00 0C                   MOVEA.l $C(a6),a0
0709BA  10 28 00 05                   MOVE.b $5(a0),d0
0709BE  BC 80                         CMP.l d0,d6
0709C0  64 12                         BCC $0709D4
0709C2  20 06                         MOVE.l d6,d0
0709C4  22 06                         MOVE.l d6,d1
0709C6  E5 81                         ASL.l #2,d1
0709C8  50 88                         ADDQ.l #8,a0
0709CA  D1 C1                         ADDA.l d1,a0
0709CC  2D 90 18 88                   MOVE.l (a0),-$78(a6,d1.l)
0709D0  52 86                         ADDQ.l #1,d6
0709D2  60 E0                         BRA $0709B4
0709D4  0C 86 00 00 00 11             CMPI.l #$11,d6
0709DA  6C 0E                         BGE $0709EA
0709DC  20 06                         MOVE.l d6,d0
0709DE  22 06                         MOVE.l d6,d1
0709E0  E5 81                         ASL.l #2,d1
0709E2  42 B6 18 88                   CLR.l -$78(a6,d1.l)
0709E6  52 86                         ADDQ.l #1,d6
0709E8  60 EA                         BRA $0709D4
0709EA  20 6E 00 14                   MOVEA.l $14(a6),a0
0709EE  10 28 00 09                   MOVE.b $9(a0),d0
0709F2  53 00                         SUBQ.b #1,d0
0709F4  66 26                         BNE $070A1C
0709F6  4A AE 00 10                   TST.l $10(a6)
0709FA  67 10                         BEQ $070A0C
0709FC  70 00                         MOVEQ #$0,d0
0709FE  10 2E FF FB                   MOVE.b -$5(a6),d0
070A02  E5 80                         ASL.l #2,d0
070A04  2D AE 00 10 08 88             MOVE.l $10(a6),-$78(a6,d0.l)
070A0A  60 0C                         BRA $070A18
070A0C  70 00                         MOVEQ #$0,d0
070A0E  10 2E FF FB                   MOVE.b -$5(a6),d0
070A12  E5 80                         ASL.l #2,d0
070A14  42 B6 08 88                   CLR.l -$78(a6,d0.l)
070A18  52 2E FF FB                   ADDQ.b #1,-$5(a6)
070A1C  20 6E 00 18                   MOVEA.l $18(a6),a0
070A20  20 2E FF F2                   MOVE.l -$E(a6),d0
070A24  2D 48 FF D4                   MOVE.l a0,-$2C(a6)
070A28  D1 C0                         ADDA.l d0,a0
070A2A  22 2E 00 1C                   MOVE.l $1C(a6),d1
070A2E  92 80                         SUB.l d0,d1
070A30  2D 48 00 18                   MOVE.l a0,$18(a6)
070A34  D1 C1                         ADDA.l d1,a0
070A36  2D 48 FF D8                   MOVE.l a0,-$28(a6)
070A3A  2A 2E FF EE                   MOVE.l -$12(a6),d5
070A3E  2D 41 00 1C                   MOVE.l d1,$1C(a6)
070A42  4A 85                         TST.l d5
070A44  6F 00 01 1C                   BLE $070B62
070A48  7C 00                         MOVEQ #$0,d6
070A4A  70 00                         MOVEQ #$0,d0
070A4C  10 2E FF FB                   MOVE.b -$5(a6),d0
070A50  BC 80                         CMP.l d0,d6
070A52  64 00 01 08                   BCC $070B5C
070A56  20 06                         MOVE.l d6,d0
070A58  22 06                         MOVE.l d6,d1
070A5A  E5 81                         ASL.l #2,d1
070A5C  41 EE FF 88                   LEA -$78(a6),a0
070A60  D1 C1                         ADDA.l d1,a0
070A62  2D 48 FF CC                   MOVE.l a0,-$34(a6)
070A66  4A 90                         TST.l (a0)
070A68  66 0E                         BNE $070A78
070A6A  2D 6E FF D4 FF D0             MOVE.l -$2C(a6),-$30(a6)
070A70  41 EE FF D0                   LEA -$30(a6),a0
070A74  2D 48 FF CC                   MOVE.l a0,-$34(a6)
070A78  4A 2E FF ED                   TST.b -$13(a6)
070A7C  66 36                         BNE $070AB4
070A7E  2F 2E FF F6                   MOVE.l -$A(a6),-(a7)
070A82  20 6E FF CC                   MOVEA.l -$34(a6),a0
070A86  2F 10                         MOVE.l (a0),-(a7)
070A88  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
070A8C  4E B9 00 07 0F 12             JSR $70F12.l
070A92  4F EF 00 0C                   LEA $C(a7),a7
070A96  2E 00                         MOVE.l d0,d7
070A98  4A 87                         TST.l d7
070A9A  67 0A                         BEQ $070AA6
070A9C  20 07                         MOVE.l d7,d0
070A9E  4C DF                         .dc.w $4CDF
070AA0  .dc.b 00 F4 4E 5E 4E 75                               ; ..N^Nu
070AA6  20 2E FF F6                   MOVE.l -$A(a6),d0
070AAA  20 6E FF CC                   MOVEA.l -$34(a6),a0
070AAE  D1 90                         ADD.l d0,(a0)
070AB0  60 00 00 A4                   BRA $070B56
070AB4  20 2E FF D8                   MOVE.l -$28(a6),d0
070AB8  90 AE 00 18                   SUB.l $18(a6),d0
070ABC  28 00                         MOVE.l d0,d4
070ABE  20 2E 00 1C                   MOVE.l $1C(a6),d0
070AC2  90 84                         SUB.l d4,d0
070AC4  2D 40 FF DC                   MOVE.l d0,-$24(a6)
070AC8  B0 AE FF F2                   CMP.l -$E(a6),d0
070ACC  6C 60                         BGE $070B2E
070ACE  2F 00                         MOVE.l d0,-(a7)
070AD0  2F 2E 00 18                   MOVE.l $18(a6),-(a7)
070AD4  2F 2E FF D8                   MOVE.l -$28(a6),-(a7)
070AD8  4E B9 00 07 1F F0             JSR $71FF0.l
070ADE  4F EF 00 0C                   LEA $C(a7),a7
070AE2  20 6E 00 08                   MOVEA.l $8(a6),a0
070AE6  20 28 00 18                   MOVE.l $18(a0),d0
070AEA  90 A8 00 20                   SUB.l $20(a0),d0
070AEE  B8 80                         CMP.l d0,d4
070AF0  6F 0C                         BLE $070AFE
070AF2  28 00                         MOVE.l d0,d4
070AF4  20 2E FF DC                   MOVE.l -$24(a6),d0
070AF8  D0 84                         ADD.l d4,d0
070AFA  2D 40 00 1C                   MOVE.l d0,$1C(a6)
070AFE  20 6E 00 18                   MOVEA.l $18(a6),a0
070B02  D1 EE FF DC                   ADDA.l -$24(a6),a0
070B06  2F 04                         MOVE.l d4,-(a7)
070B08  2F 08                         MOVE.l a0,-(a7)
070B0A  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
070B0E  4E B9 00 07 0F 12             JSR $70F12.l
070B14  4F EF 00 0C                   LEA $C(a7),a7
070B18  2E 00                         MOVE.l d0,d7
070B1A  4A 87                         TST.l d7
070B1C  67 0A                         BEQ $070B28
070B1E  20 07                         MOVE.l d7,d0
070B20  4C DF                         .dc.w $4CDF
070B22  .dc.b 00 F4 4E 5E 4E 75                               ; ..N^Nu
070B28  2D 6E 00 18 FF D8             MOVE.l $18(a6),-$28(a6)
070B2E  2F 2E FF F6                   MOVE.l -$A(a6),-(a7)
070B32  2F 2E FF F2                   MOVE.l -$E(a6),-(a7)
070B36  2F 2E FF CC                   MOVE.l -$34(a6),-(a7)
070B3A  48 6E FF D8                   PEA -$28(a6)
070B3E  4E B9 00 07 0B 6C             JSR $70B6C.l
070B44  4F EF 00 10                   LEA $10(a7),a7
070B48  4A 40                         TST.w d0
070B4A  67 0A                         BEQ $070B56
070B4C  70 F9                         MOVEQ #$F9,d0
070B4E  4C DF                         .dc.w $4CDF
070B50  .dc.b 00 F4 4E 5E 4E 75                               ; ..N^Nu
070B56  52 86                         ADDQ.l #1,d6
070B58  60 00 FE F0                   BRA $070A4A
070B5C  53 85                         SUBQ.l #1,d5
070B5E  60 00 FE E2                   BRA $070A42
070B62  70 00                         MOVEQ #$0,d0
070B64  4C DF                         .dc.w $4CDF
070B66  .dc.b 00 F4 4E 5E 4E 75                               ; ..N^Nu

; ==== _UnPackRow  $070B6C  (1 caller) — ByteRun1 (PackBits) row decompressor — the standard IFF ILBM run-length scheme. ====
070B6C  4E 56 FF F0                   LINK a6,#-$10
070B70  48 E7 0F 0C                   MOVEM.l d4-d7/a4-a5,-(a7)
070B74  20 6E 00 08                   MOVEA.l $8(a6),a0
070B78  2A 50                         MOVEA.l (a0),a5
070B7A  20 6E 00 0C                   MOVEA.l $C(a6),a0
070B7E  28 50                         MOVEA.l (a0),a4
070B80  2A 2E 00 10                   MOVE.l $10(a6),d5
070B84  28 2E 00 14                   MOVE.l $14(a6),d4
070B88  3D 7C 00 01 FF F2             MOVE.w #$1,-$E(a6)
070B8E  1D 7C 00 80 FF F1             MOVE.b #$80,-$F(a6)

; --- UP.5  $070B94 —  ---
070B94  4A 04                         TST.b d4
070B96  6F 48                         BLE $070BE0
070B98  53 05                         SUBQ.b #1,d5
070B9A  6D 48                         BLT $070BE4
070B9C  1E 1D                         MOVE.b (a5)+,d7
070B9E  6B 18                         BMI $070BB8
070BA0  52 07                         ADDQ.b #1,d7
070BA2  9A 07                         SUB.b d7,d5
070BA4  6D 3E                         BLT $070BE4
070BA6  98 07                         SUB.b d7,d4
070BA8  6D 3A                         BLT $070BE4
070BAA  02 47 00 FF                   ANDI.w #$FF,d7
070BAE  53 47                         SUBQ.w #1,d7

; --- LUP.4  $070BB0 —  ---
070BB0  18 DD                         MOVE.b (a5)+,(a4)+
070BB2  51 CF FF FC                   DBRA d7,$070BB0
070BB6  60 DC                         BRA $070B94

; --- UP.3  $070BB8 —  ---
070BB8  10 2E FF F1                   MOVE.b -$F(a6),d0
070BBC  B0 07                         CMP.b d7,d0
070BBE  67 D4                         BEQ $070B94
070BC0  20 07                         MOVE.l d7,d0
070BC2  44 00                         NEG.b d0
070BC4  52 00                         ADDQ.b #1,d0
070BC6  2E 00                         MOVE.l d0,d7
070BC8  53 05                         SUBQ.b #1,d5
070BCA  6D 18                         BLT $070BE4
070BCC  98 07                         SUB.b d7,d4
070BCE  6D 14                         BLT $070BE4
070BD0  1C 1D                         MOVE.b (a5)+,d6
070BD2  02 47 00 FF                   ANDI.w #$FF,d7
070BD6  53 47                         SUBQ.w #1,d7

; --- LUP.6  $070BD8 —  ---
070BD8  18 C6                         MOVE.b d6,(a4)+
070BDA  51 CF FF FC                   DBRA d7,$070BD8
070BDE  60 B4                         BRA $070B94

; --- UP.1  $070BE0 —  ---
070BE0  42 6E FF F2                   CLR.w -$E(a6)

; --- UP.2  $070BE4 —  ---
070BE4  20 6E 00 08                   MOVEA.l $8(a6),a0
070BE8  20 8D                         MOVE.l a5,(a0)
070BEA  20 6E 00 0C                   MOVEA.l $C(a6),a0
070BEE  20 8C                         MOVE.l a4,(a0)
070BF0  30 2E FF F2                   MOVE.w -$E(a6),d0
070BF4  4C DF                         .dc.w $4CDF
070BF6  .dc.b 30 F0 4E 5E 4E 75                               ; 0.N^Nu

; ==== _OpenRIFF  $070BFC  (1 caller) — EA IFF: open a read context on a file. ====
070BFC  4E 56 FF F4                   LINK a6,#-$C
070C00  48 E7 01 04                   MOVEM.l d7/a5,-(a7)
070C04  2E 2E 00 08                   MOVE.l $8(a6),d7
070C08  2A 6E 00 0C                   MOVEA.l $C(a6),a5
070C0C  70 00                         MOVEQ #$0,d0
070C0E  2A 80                         MOVE.l d0,(a5)
070C10  2B 6E 00 10 00 04             MOVE.l $10(a6),$4(a5)
070C16  2B 47 00 08                   MOVE.l d7,$8(a5)
070C1A  2B 40 00 0C                   MOVE.l d0,$C(a5)
070C1E  2B 40 00 1C                   MOVE.l d0,$1C(a5)
070C22  2B 40 00 14                   MOVE.l d0,$14(a5)
070C26  2B 40 00 20                   MOVE.l d0,$20(a5)
070C2A  2B 40 00 18                   MOVE.l d0,$18(a5)
070C2E  2D 40 FF F4                   MOVE.l d0,-$C(a6)
070C32  4A 87                         TST.l d7
070C34  6E 0A                         BGT $070C40
070C36  70 FB                         MOVEQ #$FB,d0
070C38  4C DF                         .dc.w $4CDF
070C3A  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu
070C40  70 01                         MOVEQ #$1,d0
070C42  2F 00                         MOVE.l d0,-(a7)
070C44  42 A7                         CLR.l -(a7)
070C46  2F 07                         MOVE.l d7,-(a7)
070C48  4E B9 00 07 1D F4             JSR $71DF4.l
070C4E  4F EF 00 0C                   LEA $C(a7),a7
070C52  70 00                         MOVEQ #$0,d0
070C54  2F 00                         MOVE.l d0,-(a7)
070C56  2F 00                         MOVE.l d0,-(a7)
070C58  2F 07                         MOVE.l d7,-(a7)
070C5A  4E B9 00 07 1D F4             JSR $71DF4.l
070C60  4F EF 00 0C                   LEA $C(a7),a7
070C64  2B 40 00 10                   MOVE.l d0,$10(a5)
070C68  4A 80                         TST.l d0
070C6A  6A 0A                         BPL $070C76
070C6C  70 FD                         MOVEQ #$FD,d0
070C6E  4C DF                         .dc.w $4CDF
070C70  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu
070C76  70 FF                         MOVEQ #$FF,d0
070C78  2F 00                         MOVE.l d0,-(a7)
070C7A  42 A7                         CLR.l -(a7)
070C7C  2F 07                         MOVE.l d7,-(a7)
070C7E  4E B9 00 07 1D F4             JSR $71DF4.l
070C84  4F EF 00 0C                   LEA $C(a7),a7
070C88  0C AD 00 00 00 08 00 10       CMPI.l #$8,$10(a5)
070C90  6C 06                         BGE $070C98
070C92  70 FC                         MOVEQ #$FC,d0
070C94  2D 40 FF F4                   MOVE.l d0,-$C(a6)
070C98  20 2E FF F4                   MOVE.l -$C(a6),d0
070C9C  4C DF                         .dc.w $4CDF
070C9E  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu

; ==== _OpenRGroup  $070CA4  (3 callers) — EA IFF: descend into a FORM/LIST/CAT group. ====
070CA4  4E 56 FF F4                   LINK a6,#-$C
070CA8  48 E7 20 0C                   MOVEM.l d2/a4-a5,-(a7)
070CAC  2A 6E 00 08                   MOVEA.l $8(a6),a5
070CB0  28 6E 00 0C                   MOVEA.l $C(a6),a4
070CB4  70 00                         MOVEQ #$0,d0
070CB6  28 8D                         MOVE.l a5,(a4)
070CB8  29 6D 00 04 00 04             MOVE.l $4(a5),$4(a4)
070CBE  29 6D 00 08 00 08             MOVE.l $8(a5),$8(a4)
070CC4  22 2D 00 0C                   MOVE.l $C(a5),d1
070CC8  29 41 00 0C                   MOVE.l d1,$C(a4)
070CCC  22 2D 00 18                   MOVE.l $18(a5),d1
070CD0  92 AD 00 20                   SUB.l $20(a5),d1
070CD4  24 2D 00 0C                   MOVE.l $C(a5),d2
070CD8  D4 81                         ADD.l d1,d2
070CDA  29 42 00 10                   MOVE.l d2,$10(a4)
070CDE  29 40 00 1C                   MOVE.l d0,$1C(a4)
070CE2  29 40 00 14                   MOVE.l d0,$14(a4)
070CE6  29 40 00 20                   MOVE.l d0,$20(a4)
070CEA  29 40 00 18                   MOVE.l d0,$18(a4)
070CEE  2D 40 FF F4                   MOVE.l d0,-$C(a6)
070CF2  20 2C 00 10                   MOVE.l $10(a4),d0
070CF6  B0 AD 00 10                   CMP.l $10(a5),d0
070CFA  6E 06                         BGT $070D02
070CFC  08 00 00 00                   BTST.l #$0,d0
070D00  67 06                         BEQ $070D08
070D02  70 F7                         MOVEQ #$F7,d0
070D04  2D 40 FF F4                   MOVE.l d0,-$C(a6)
070D08  20 2E FF F4                   MOVE.l -$C(a6),d0
070D0C  4C DF                         .dc.w $4CDF
070D0E  .dc.b 30 04 4E 5E 4E 75                               ; 0.N^Nu

; ==== _CloseRGroup  $070D14  (4 callers) — EA IFF: leave a group. ====
070D14  4E 56 FF FC                   LINK a6,#-$4
070D18  48 E7 01 00                   MOVEM.l d7,-(a7)
070D1C  20 6E 00 08                   MOVEA.l $8(a6),a0
070D20  4A 90                         TST.l (a0)
070D22  67 18                         BEQ $070D3C
070D24  20 6E 00 08                   MOVEA.l $8(a6),a0
070D28  2E 28 00 0C                   MOVE.l $C(a0),d7
070D2C  20 50                         MOVEA.l (a0),a0
070D2E  20 07                         MOVE.l d7,d0
070D30  90 A8 00 0C                   SUB.l $C(a0),d0
070D34  D1 A8 00 20                   ADD.l d0,$20(a0)
070D38  21 47 00 0C                   MOVE.l d7,$C(a0)
070D3C  70 00                         MOVEQ #$0,d0
070D3E  4C DF                         .dc.w $4CDF
070D40  .dc.b 00 80 4E 5E 4E 75                               ; ..N^Nu

; ==== _SkipFwd  $070D46  (1 caller) — EA IFF: skip forward over chunk data. ====
070D46  4E 56 FF FC                   LINK a6,#-$4
070D4A  70 00                         MOVEQ #$0,d0
070D4C  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070D50  4A AE 00 0C                   TST.l $C(a6)
070D54  6F 30                         BLE $070D86
070D56  2F 00                         MOVE.l d0,-(a7)
070D58  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
070D5C  20 6E 00 08                   MOVEA.l $8(a6),a0
070D60  2F 28 00 08                   MOVE.l $8(a0),-(a7)
070D64  4E B9 00 07 1D F4             JSR $71DF4.l
070D6A  4F EF 00 0C                   LEA $C(a7),a7
070D6E  52 80                         ADDQ.l #1,d0
070D70  66 08                         BNE $070D7A
070D72  70 F7                         MOVEQ #$F7,d0
070D74  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070D78  60 0C                         BRA $070D86
070D7A  20 2E 00 0C                   MOVE.l $C(a6),d0
070D7E  20 6E 00 08                   MOVEA.l $8(a6),a0
070D82  D1 A8 00 0C                   ADD.l d0,$C(a0)
070D86  20 2E FF FC                   MOVE.l -$4(a6),d0
070D8A  4E 5E                         UNLK a6
070D8C  4E 75                         RTS

; ==== _GetChunkHdr  $070D8E  (5 callers) — EA IFF: read the next chunk's id + size. ====
070D8E  4E 56 FF F4                   LINK a6,#-$C
070D92  48 E7 01 04                   MOVEM.l d7/a5,-(a7)
070D96  2A 6E 00 08                   MOVEA.l $8(a6),a5
070D9A  20 2D 00 18                   MOVE.l $18(a5),d0
070D9E  90 AD 00 20                   SUB.l $20(a5),d0
070DA2  22 2D 00 18                   MOVE.l $18(a5),d1
070DA6  02 81 00 00 00 01             ANDI.l #$1,d1
070DAC  D0 81                         ADD.l d1,d0
070DAE  2F 00                         MOVE.l d0,-(a7)
070DB0  2F 0D                         MOVE.l a5,-(a7)
070DB2  61 92                         BSR $070D46
070DB4  50 8F                         ADDQ.l #8,a7
070DB6  2E 00                         MOVE.l d0,d7
070DB8  4A 87                         TST.l d7
070DBA  67 0A                         BEQ $070DC6
070DBC  20 07                         MOVE.l d7,d0
070DBE  4C DF                         .dc.w $4CDF
070DC0  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu
070DC6  70 F7                         MOVEQ #$F7,d0
070DC8  2B 40 00 14                   MOVE.l d0,$14(a5)
070DCC  70 00                         MOVEQ #$0,d0
070DCE  2B 40 00 1C                   MOVE.l d0,$1C(a5)
070DD2  2B 40 00 20                   MOVE.l d0,$20(a5)
070DD6  20 2D 00 10                   MOVE.l $10(a5),d0
070DDA  90 AD 00 0C                   SUB.l $C(a5),d0
070DDE  2D 40 FF F4                   MOVE.l d0,-$C(a6)
070DE2  4A 80                         TST.l d0
070DE4  66 0E                         BNE $070DF4
070DE6  42 AD 00 18                   CLR.l $18(a5)
070DEA  70 FF                         MOVEQ #$FF,d0
070DEC  2B 40 00 14                   MOVE.l d0,$14(a5)
070DF0  60 00 01 14                   BRA $070F06
070DF4  20 2E FF F4                   MOVE.l -$C(a6),d0
070DF8  0C 80 00 00 00 08             CMPI.l #$8,d0
070DFE  6C 08                         BGE $070E08
070E00  2B 40 00 18                   MOVE.l d0,$18(a5)
070E04  60 00 01 00                   BRA $070F06
070E08  20 4D                         MOVEA.l a5,a0
070E0A  D0 FC 00 14                   ADDA.w #$14,a0
070E0E  70 08                         MOVEQ #$8,d0
070E10  2F 00                         MOVE.l d0,-(a7)
070E12  2F 08                         MOVE.l a0,-(a7)
070E14  2F 2D 00 08                   MOVE.l $8(a5),-(a7)
070E18  4E B9 00 07 1D D8             JSR $71DD8.l
070E1E  4F EF 00 0C                   LEA $C(a7),a7
070E22  4A 80                         TST.l d0
070E24  67 16                         BEQ $070E3C
070E26  0C 80 FF FF FF FF             CMPI.l #$FFFFFFFF,d0
070E2C  66 1C                         BNE $070E4A
070E2E  70 FD                         MOVEQ #$FD,d0
070E30  2B 40 00 14                   MOVE.l d0,$14(a5)
070E34  4C DF                         .dc.w $4CDF
070E36  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu
070E3C  70 F7                         MOVEQ #$F7,d0
070E3E  2B 40 00 14                   MOVE.l d0,$14(a5)
070E42  4C DF                         .dc.w $4CDF
070E44  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu
070E4A  4A 95                         TST.l (a5)
070E4C  66 2C                         BNE $070E7A
070E4E  20 2D 00 14                   MOVE.l $14(a5),d0
070E52  0C 80 43 41 54 20             CMPI.l #$43415420,d0
070E58  67 20                         BEQ $070E7A
070E5A  0C 80 4C 49 53 54             CMPI.l #$4C495354,d0
070E60  67 18                         BEQ $070E7A
070E62  0C 80 46 4F 52 4D             CMPI.l #$464F524D,d0
070E68  67 10                         BEQ $070E7A
070E6A  4E 71                         NOP
070E6C  70 FC                         MOVEQ #$FC,d0
070E6E  2B 40 00 14                   MOVE.l d0,$14(a5)
070E72  4C DF                         .dc.w $4CDF
070E74  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu
070E7A  50 AD 00 0C                   ADDQ.l #8,$C(a5)
070E7E  51 AE FF F4                   SUBQ.l #8,-$C(a6)
070E82  4A AD 00 14                   TST.l $14(a5)
070E86  6E 08                         BGT $070E90
070E88  70 F7                         MOVEQ #$F7,d0
070E8A  2B 40 00 14                   MOVE.l d0,$14(a5)
070E8E  60 76                         BRA $070F06
070E90  4A AD 00 18                   TST.l $18(a5)
070E94  6B 0A                         BMI $070EA0
070E96  20 2D 00 18                   MOVE.l $18(a5),d0
070E9A  B0 AE FF F4                   CMP.l -$C(a6),d0
070E9E  6F 0E                         BLE $070EAE
070EA0  2B 6E FF F4 00 18             MOVE.l -$C(a6),$18(a5)
070EA6  70 F7                         MOVEQ #$F7,d0
070EA8  2B 40 00 14                   MOVE.l d0,$14(a5)
070EAC  60 58                         BRA $070F06
070EAE  20 2D 00 14                   MOVE.l $14(a5),d0
070EB2  72 20                         MOVEQ #$20,d1
070EB4  04 81 00 00 00 08             SUBI.l #$8,d1
070EBA  6B 4A                         BMI $070F06
070EBC  B0 BB 18 08                   CMP.l $8(pc,d1.l),d0
070EC0  66 F2                         BNE $070EB4
070EC2  4E FB 18 06                   JMP $6(pc,d1.l)
070EC6  .dc.b 43 41 54 20 60 00 00 1A 50 52 4F 50 60 00 00 12 ; CAT `...PROP`...
070ED6  .dc.b 46 4F 52 4D 60 00 00 0A 4C 49 53 54 60 00 00 02 ; FORM`...LIST`...
070EE6  .dc.b 20 4D D0 FC 00 1C 20 08 72 04 2F 01 2F 00 2F 0D ;  M.... .r./././.
070EF6  .dc.b 61 1A 4F EF 00 0C 2E 00 4A 87 67 04 2B 47 00 14 ; a.O.....J.g.+G..
070F06  20 2D 00 14                   MOVE.l $14(a5),d0
070F0A  4C DF                         .dc.w $4CDF
070F0C  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu

; ==== _IFFReadBytes  $070F12  (4 callers) — EA IFF: read raw bytes from the current chunk. ====
070F12  4E 56 FF FC                   LINK a6,#-$4
070F16  48 E7 01 00                   MOVEM.l d7,-(a7)
070F1A  7E 00                         MOVEQ #$0,d7
070F1C  4A AE 00 10                   TST.l $10(a6)
070F20  6A 04                         BPL $070F26
070F22  7E FA                         MOVEQ #$FA,d7
070F24  60 5C                         BRA $070F82
070F26  20 6E 00 08                   MOVEA.l $8(a6),a0
070F2A  20 28 00 18                   MOVE.l $18(a0),d0
070F2E  90 A8 00 20                   SUB.l $20(a0),d0
070F32  22 2E 00 10                   MOVE.l $10(a6),d1
070F36  B2 80                         CMP.l d0,d1
070F38  6F 04                         BLE $070F3E
070F3A  7E F8                         MOVEQ #$F8,d7
070F3C  60 44                         BRA $070F82
070F3E  4A AE 00 10                   TST.l $10(a6)
070F42  6F 3E                         BLE $070F82
070F44  2F 2E 00 10                   MOVE.l $10(a6),-(a7)
070F48  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
070F4C  20 6E 00 08                   MOVEA.l $8(a6),a0
070F50  2F 28 00 08                   MOVE.l $8(a0),-(a7)
070F54  4E B9 00 07 1D D8             JSR $71DD8.l
070F5A  4F EF 00 0C                   LEA $C(a7),a7
070F5E  4A 80                         TST.l d0
070F60  67 0C                         BEQ $070F6E
070F62  0C 80 FF FF FF FF             CMPI.l #$FFFFFFFF,d0
070F68  66 08                         BNE $070F72
070F6A  7E FD                         MOVEQ #$FD,d7
070F6C  60 14                         BRA $070F82
070F6E  7E F7                         MOVEQ #$F7,d7
070F70  60 10                         BRA $070F82
070F72  20 2E 00 10                   MOVE.l $10(a6),d0
070F76  20 6E 00 08                   MOVEA.l $8(a6),a0
070F7A  D1 A8 00 0C                   ADD.l d0,$C(a0)
070F7E  D1 A8 00 20                   ADD.l d0,$20(a0)
070F82  20 07                         MOVE.l d7,d0
070F84  4C DF                         .dc.w $4CDF
070F86  .dc.b 00 80 4E 5E 4E 75                               ; ..N^Nu

; --- _SkipGroup  $070F8C —  ---
070F8C  4E 56 00 00                   LINK a6,#$0
070F90  4E 5E                         UNLK a6
070F92  4E 75                         RTS

; ==== _ReadIFF  $070F94  (1 caller) — EA IFF: top-level reader dispatch (FORM/LIST/CAT). ====
070F94  4E 56 FF D8                   LINK a6,#-$28
070F98  48 6E FF D8                   PEA -$28(a6)
070F9C  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
070FA0  61 00 FC 5A                   BSR $070BFC
070FA4  50 8F                         ADDQ.l #8,a7
070FA6  2D 6E 00 0C FF DC             MOVE.l $C(a6),-$24(a6)
070FAC  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070FB0  4A 80                         TST.l d0
070FB2  66 64                         BNE $071018
070FB4  48 6E FF D8                   PEA -$28(a6)
070FB8  61 00 FD D4                   BSR $070D8E
070FBC  58 8F                         ADDQ.l #4,a7
070FBE  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070FC2  0C 80 43 41 54 20             CMPI.l #$43415420,d0
070FC8  67 3A                         BEQ $071004
070FCA  0C 80 4C 49 53 54             CMPI.l #$4C495354,d0
070FD0  67 1E                         BEQ $070FF0
070FD2  0C 80 46 4F 52 4D             CMPI.l #$464F524D,d0
070FD8  66 3E                         BNE $071018
070FDA  22 6E 00 0C                   MOVEA.l $C(a6),a1
070FDE  20 69 00 08                   MOVEA.l $8(a1),a0
070FE2  48 6E FF D8                   PEA -$28(a6)
070FE6  4E 90                         JSR (a0)
070FE8  58 8F                         ADDQ.l #4,a7
070FEA  2D 40 FF FC                   MOVE.l d0,-$4(a6)
070FEE  60 28                         BRA $071018
070FF0  22 6E 00 0C                   MOVEA.l $C(a6),a1
070FF4  20 51                         MOVEA.l (a1),a0
070FF6  48 6E FF D8                   PEA -$28(a6)
070FFA  4E 90                         JSR (a0)
070FFC  58 8F                         ADDQ.l #4,a7
070FFE  2D 40 FF FC                   MOVE.l d0,-$4(a6)
071002  60 14                         BRA $071018
071004  22 6E 00 0C                   MOVEA.l $C(a6),a1
071008  20 69 00 0C                   MOVEA.l $C(a1),a0
07100C  48 6E FF D8                   PEA -$28(a6)
071010  4E 90                         JSR (a0)
071012  58 8F                         ADDQ.l #4,a7
071014  2D 40 FF FC                   MOVE.l d0,-$4(a6)
071018  48 6E FF D8                   PEA -$28(a6)
07101C  61 00 FC F6                   BSR $070D14
071020  58 8F                         ADDQ.l #4,a7
071022  4A AE FF FC                   TST.l -$4(a6)
071026  6F 06                         BLE $07102E
071028  70 FC                         MOVEQ #$FC,d0
07102A  2D 40 FF FC                   MOVE.l d0,-$4(a6)
07102E  20 2E FF FC                   MOVE.l -$4(a6),d0
071032  4E 5E                         UNLK a6
071034  4E 75                         RTS

; ==== _ReadIList  $071036  (2 callers) —  ====
071036  4E 56 FF D6                   LINK a6,#-$2A
07103A  3D 7C 00 01 FF D6             MOVE.w #$1,-$2A(a6)
071040  48 6E FF DC                   PEA -$24(a6)
071044  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
071048  61 00 FC 5A                   BSR $070CA4
07104C  50 8F                         ADDQ.l #8,a7
07104E  2D 40 FF D8                   MOVE.l d0,-$28(a6)
071052  4A 80                         TST.l d0
071054  67 04                         BEQ $07105A
071056  4E 5E                         UNLK a6
071058  4E 75                         RTS
07105A  20 6E 00 08                   MOVEA.l $8(a6),a0
07105E  0C A8 43 41 54 20 00 14       CMPI.l #$43415420,$14(a0)
071066  66 06                         BNE $07106E
071068  42 6E FF D6                   CLR.w -$2A(a6)
07106C  60 06                         BRA $071074
07106E  2D 6E 00 0C FF E0             MOVE.l $C(a6),-$20(a6)
071074  48 6E FF DC                   PEA -$24(a6)
071078  61 00 FD 14                   BSR $070D8E
07107C  58 8F                         ADDQ.l #4,a7
07107E  2D 40 FF D8                   MOVE.l d0,-$28(a6)
071082  72 20                         MOVEQ #$20,d1
071084  04 81 00 00 00 08             SUBI.l #$8,d1
07108A  6B 00 00 8E                   BMI $07111A
07108E  B0 BB 18 08                   CMP.l $8(pc,d1.l),d0
071092  66 F0                         BNE $071084
071094  4E FB 18 06                   JMP $6(pc,d1.l)
071098  .dc.b 43 41 54 20 60 00 00 68 4C 49 53 54 60 00 00 4C ; CAT `..hLIST`..L
0710A8  .dc.b 46 4F 52 4D 60 00 00 2E 50 52 4F 50 60 00 00 02 ; FORM`...PROP`...
0710B8  .dc.b 4A 6E FF D6 67 16 22 6E 00 0C 20 69 00 04 48 6E ; Jn..g."n.. i..Hn
0710C8  .dc.b FF DC 4E 90 58 8F 2D 40 FF D8 60 46 70 F7 2D 40 ; ..N.X.-@..`Fp.-@
0710D8  .dc.b FF D8 60 3E 22 6E 00 0C 20 69 00 08 48 6E FF DC ; ..`>"n.. i..Hn..
0710E8  .dc.b 4E 90 58 8F 2D 40 FF D8 60 28 22 6E 00 0C 20 51 ; N.X.-@..`("n.. Q
0710F8  .dc.b 48 6E FF DC 4E 90 58 8F 2D 40 FF D8 60 14 22 6E ; Hn..N.X.-@..`."n
071108  .dc.b 00 0C 20 69 00 0C 48 6E FF DC 4E 90 58 8F 2D 40 ; .. i..Hn..N.X.-@
071118  .dc.b FF D8                                           ; ..
07111A  0C AE 50 52 4F 50 FF F0       CMPI.l #$50524F50,-$10(a6)
071122  67 04                         BEQ $071128
071124  42 6E FF D6                   CLR.w -$2A(a6)
071128  4A AE FF D8                   TST.l -$28(a6)
07112C  67 00 FF 46                   BEQ $071074
071130  48 6E FF DC                   PEA -$24(a6)
071134  61 00 FB DE                   BSR $070D14
071138  58 8F                         ADDQ.l #4,a7
07113A  4A AE FF D8                   TST.l -$28(a6)
07113E  6F 06                         BLE $071146
071140  70 F7                         MOVEQ #$F7,d0
071142  2D 40 FF D8                   MOVE.l d0,-$28(a6)
071146  0C AE FF FF FF FF FF D8       CMPI.l #$FFFFFFFF,-$28(a6)
07114E  66 04                         BNE $071154
071150  70 00                         MOVEQ #$0,d0
071152  60 04                         BRA $071158
071154  20 2E FF D8                   MOVE.l -$28(a6),d0
071158  4E 5E                         UNLK a6
07115A  4E 75                         RTS

; --- _ReadICat  $07115C —  ---
07115C  4E 56 00 00                   LINK a6,#$0
071160  42 A7                         CLR.l -(a7)
071162  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
071166  61 00 FE CE                   BSR $071036
07116A  50 8F                         ADDQ.l #8,a7
07116C  4E 5E                         UNLK a6
07116E  4E 75                         RTS

; ==== _GetFChunkHdr  $071170  (1 caller) —  ====
071170  4E 56 FF FC                   LINK a6,#-$4
071174  48 E7 01 00                   MOVEM.l d7,-(a7)
071178  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
07117C  61 00 FC 10                   BSR $070D8E
071180  58 8F                         ADDQ.l #4,a7
071182  2E 00                         MOVE.l d0,d7
071184  0C 87 50 52 4F 50             CMPI.l #$50524F50,d7
07118A  66 0A                         BNE $071196
07118C  7E F7                         MOVEQ #$F7,d7
07118E  20 6E 00 08                   MOVEA.l $8(a6),a0
071192  21 47 00 14                   MOVE.l d7,$14(a0)
071196  20 07                         MOVE.l d7,d0
071198  4C DF                         .dc.w $4CDF
07119A  .dc.b 00 80 4E 5E 4E 75                               ; ..N^Nu

; --- _GetF1ChunkHdr  $0711A0 —  ---
0711A0  4E 56 FF F8                   LINK a6,#-$8
0711A4  48 E7 01 04                   MOVEM.l d7/a5,-(a7)
0711A8  20 6E 00 08                   MOVEA.l $8(a6),a0
0711AC  2A 68 00 04                   MOVEA.l $4(a0),a5
0711B0  2F 08                         MOVE.l a0,-(a7)
0711B2  61 00 FB DA                   BSR $070D8E
0711B6  58 8F                         ADDQ.l #4,a7
0711B8  2E 00                         MOVE.l d0,d7
0711BA  20 07                         MOVE.l d7,d0
0711BC  72 20                         MOVEQ #$20,d1
0711BE  04 81 00 00 00 08             SUBI.l #$8,d1
0711C4  6B 5A                         BMI $071220
0711C6  B0 BB 18 08                   CMP.l $8(pc,d1.l),d0
0711CA  66 F2                         BNE $0711BE
0711CC  4E FB 18 06                   JMP $6(pc,d1.l)
0711D0  .dc.b 43 41 54 20 60 00 00 3C 4C 49 53 54 60 00 00 26 ; CAT `..<LIST`..&
0711E0  .dc.b 46 4F 52 4D 60 00 00 0E 50 52 4F 50 60 00 00 02 ; FORM`...PROP`...
0711F0  .dc.b 7E F7 60 2C 20 6D 00 08 2F 2E 00 08 4E 90 58 8F ; ~.`, m../...N.X.
071200  .dc.b 2E 00 60 1C 20 55 2F 2E 00 08 4E 90 58 8F 2E 00 ; ..`. U/...N.X...
071210  .dc.b 60 0E 20 6D 00 0C 2F 2E 00 08 4E 90 58 8F 2E 00 ; `. m../...N.X...
071220  20 07                         MOVE.l d7,d0
071222  20 6E 00 08                   MOVEA.l $8(a6),a0
071226  21 40 00 14                   MOVE.l d0,$14(a0)
07122A  4C DF                         .dc.w $4CDF
07122C  .dc.b 20 80 4E 5E 4E 75                               ;  .N^Nu

; ==== _GetPChunkHdr  $071232  (1 caller) —  ====
071232  4E 56 FF FC                   LINK a6,#-$4
071236  48 E7 01 00                   MOVEM.l d7,-(a7)
07123A  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
07123E  61 00 FB 4E                   BSR $070D8E
071242  58 8F                         ADDQ.l #4,a7
071244  2E 00                         MOVE.l d0,d7
071246  20 07                         MOVE.l d7,d0
071248  72 20                         MOVEQ #$20,d1
07124A  04 81 00 00 00 08             SUBI.l #$8,d1
071250  6B 36                         BMI $071288
071252  B0 BB 18 08                   CMP.l $8(pc,d1.l),d0
071256  66 F2                         BNE $07124A
071258  4E FB 18 06                   JMP $6(pc,d1.l)
07125C  .dc.b 43 41 54 20 60 00 00 1A 50 52 4F 50 60 00 00 12 ; CAT `...PROP`...
07126C  .dc.b 46 4F 52 4D 60 00 00 0A 4C 49 53 54 60 00 00 02 ; FORM`...LIST`...
07127C  .dc.b 70 F7 20 6E 00 08 21 40 00 14 2E 00             ; p. n..!@....
071288  20 07                         MOVE.l d7,d0
07128A  4C DF                         .dc.w $4CDF
07128C  .dc.b 00 80 4E 5E 4E 75 00 00                         ; ..N^Nu..

; --- _MFOpen  $071294 — mini-filesystem: open a file by reading the disk directly through trackdisk.device (bypasses AmigaDOS). ---
071294  4E 56 FF F8                   LINK a6,#-$8
071298  91 C8                         SUBA.l a0,a0
07129A  23 C8 00 07 1B F8             MOVE.l a0,$71BF8.l
0712A0  2D 48 FF FC                   MOVE.l a0,-$4(a6)
0712A4  20 2E 00 0C                   MOVE.l $C(a6),d0
0712A8  0C 80 00 00 03 ED             CMPI.l #$3ED,d0
0712AE  67 00 00 FE                   BEQ $0713AE
0712B2  0C 80 00 00 03 EE             CMPI.l #$3EE,d0
0712B8  66 00 01 AA                   BNE $071464
0712BC  61 00 06 1C                   BSR $0718DA
0712C0  4A 80                         TST.l d0
0712C2  67 0E                         BEQ $0712D2
0712C4  70 01                         MOVEQ #$1,d0
0712C6  23 C0 00 07 1B F8             MOVE.l d0,$71BF8.l
0712CC  70 00                         MOVEQ #$0,d0
0712CE  4E 5E                         UNLK a6
0712D0  4E 75                         RTS
0712D2  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0712D6  2F 39 00 07 1C 00             MOVE.l $71C00.l,-(a7)
0712DC  61 00 07 44                   BSR $071A22
0712E0  50 8F                         ADDQ.l #8,a7
0712E2  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0712E6  52 80                         ADDQ.l #1,d0
0712E8  67 0A                         BEQ $0712F4
0712EA  61 00 06 D2                   BSR $0719BE
0712EE  70 00                         MOVEQ #$0,d0
0712F0  4E 5E                         UNLK a6
0712F2  4E 75                         RTS
0712F4  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0712F8  2F 39 00 07 1C 00             MOVE.l $71C00.l,-(a7)
0712FE  61 00 07 7A                   BSR $071A7A
071302  50 8F                         ADDQ.l #8,a7
071304  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
07130A  72 2C                         MOVEQ #$2C,d1
07130C  2F 01                         MOVE.l d1,-(a7)
07130E  2D 40 FF F8                   MOVE.l d0,-$8(a6)
071312  4E B9 00 07 1E 10             JSR $71E10.l
071318  50 8F                         ADDQ.l #8,a7
07131A  2D 40 FF FC                   MOVE.l d0,-$4(a6)
07131E  0C AE FF FF FF FF FF F8       CMPI.l #$FFFFFFFF,-$8(a6)
071326  66 0E                         BNE $071336
071328  70 07                         MOVEQ #$7,d0
07132A  23 C0 00 07 1B F8             MOVE.l d0,$71BF8.l
071330  70 00                         MOVEQ #$0,d0
071332  4E 5E                         UNLK a6
071334  4E 75                         RTS
071336  20 2E FF F8                   MOVE.l -$8(a6),d0
07133A  72 1A                         MOVEQ #$1A,d1
07133C  4E B9 00 07 20 5C             JSR $7205C.l
071342  20 79 00 07 1C 00             MOVEA.l $71C00.l,a0
071348  D0 FC 00 10                   ADDA.w #$10,a0
07134C  D1 C0                         ADDA.l d0,a0
07134E  22 79 00 07 1C 00             MOVEA.l $71C00.l,a1
071354  20 A9 00 0C                   MOVE.l $C(a1),(a0)
071358  20 6E FF FC                   MOVEA.l -$4(a6),a0
07135C  21 6E FF F8 00 10             MOVE.l -$8(a6),$10(a0)
071362  21 7C 00 00 03 EE 00 14       MOVE.l #$3EE,$14(a0)
07136A  42 A8 00 0C                   CLR.l $C(a0)
07136E  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
071374  2F 3C 00 00 08 00             MOVE.l #$800,-(a7)
07137A  4E B9 00 07 1E 10             JSR $71E10.l
071380  50 8F                         ADDQ.l #8,a7
071382  20 6E FF FC                   MOVEA.l -$4(a6),a0
071386  21 40 00 1C                   MOVE.l d0,$1C(a0)
07138A  20 40                         MOVEA.l d0,a0
07138C  D0 FC 08 00                   ADDA.w #$800,a0
071390  22 6E FF FC                   MOVEA.l -$4(a6),a1
071394  23 48 00 20                   MOVE.l a0,$20(a1)
071398  23 69 00 1C 00 18             MOVE.l $1C(a1),$18(a1)
07139E  20 79 00 07 1C 00             MOVEA.l $71C00.l,a0
0713A4  23 68 00 0C 00 04             MOVE.l $C(a0),$4(a1)
0713AA  60 00 00 C6                   BRA $071472
0713AE  61 00 05 2A                   BSR $0718DA
0713B2  4A 80                         TST.l d0
0713B4  67 0E                         BEQ $0713C4
0713B6  70 01                         MOVEQ #$1,d0
0713B8  23 C0 00 07 1B F8             MOVE.l d0,$71BF8.l
0713BE  70 00                         MOVEQ #$0,d0
0713C0  4E 5E                         UNLK a6
0713C2  4E 75                         RTS
0713C4  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0713C8  2F 39 00 07 1C 00             MOVE.l $71C00.l,-(a7)
0713CE  61 00 06 52                   BSR $071A22
0713D2  50 8F                         ADDQ.l #8,a7
0713D4  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0713D8  52 80                         ADDQ.l #1,d0
0713DA  66 0A                         BNE $0713E6
0713DC  61 00 05 E0                   BSR $0719BE
0713E0  70 00                         MOVEQ #$0,d0
0713E2  4E 5E                         UNLK a6
0713E4  4E 75                         RTS
0713E6  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
0713EC  70 2C                         MOVEQ #$2C,d0
0713EE  2F 00                         MOVE.l d0,-(a7)
0713F0  4E B9 00 07 1E 10             JSR $71E10.l
0713F6  50 8F                         ADDQ.l #8,a7
0713F8  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
0713FE  2F 3C 00 00 08 00             MOVE.l #$800,-(a7)
071404  2D 40 FF FC                   MOVE.l d0,-$4(a6)
071408  4E B9 00 07 1E 10             JSR $71E10.l
07140E  50 8F                         ADDQ.l #8,a7
071410  20 6E FF FC                   MOVEA.l -$4(a6),a0
071414  21 40 00 1C                   MOVE.l d0,$1C(a0)
071418  21 7C 00 00 03 ED 00 14       MOVE.l #$3ED,$14(a0)
071420  22 68 00 1C                   MOVEA.l $1C(a0),a1
071424  D2 FC 08 00                   ADDA.w #$800,a1
071428  21 49 00 20                   MOVE.l a1,$20(a0)
07142C  21 49 00 18                   MOVE.l a1,$18(a0)
071430  20 2E FF F8                   MOVE.l -$8(a6),d0
071434  21 40 00 10                   MOVE.l d0,$10(a0)
071438  72 1A                         MOVEQ #$1A,d1
07143A  4E B9 00 07 20 5C             JSR $7205C.l
071440  22 79 00 07 1C 00             MOVEA.l $71C00.l,a1
071446  D2 FC 00 10                   ADDA.w #$10,a1
07144A  D3 C0                         ADDA.l d0,a1
07144C  21 51 00 04                   MOVE.l (a1),$4(a0)
071450  22 79 00 07 1C 00             MOVEA.l $71C00.l,a1
071456  D2 FC 00 10                   ADDA.w #$10,a1
07145A  D3 C0                         ADDA.l d0,a1
07145C  21 69 00 08 00 0C             MOVE.l $8(a1),$C(a0)
071462  60 0E                         BRA $071472
071464  70 05                         MOVEQ #$5,d0
071466  23 C0 00 07 1B F8             MOVE.l d0,$71BF8.l
07146C  70 00                         MOVEQ #$0,d0
07146E  4E 5E                         UNLK a6
071470  4E 75                         RTS
071472  20 6E FF FC                   MOVEA.l -$4(a6),a0
071476  42 90                         CLR.l (a0)
071478  21 7C 39 72 62 83 00 28       MOVE.l #$39726283,$28(a0)
071480  20 08                         MOVE.l a0,d0
071482  4E 5E                         UNLK a6
071484  4E 75                         RTS

; --- _MFClose  $071486 —  ---
071486  4E 56 FF F8                   LINK a6,#-$8
07148A  42 B9 00 07 1B F8             CLR.l $71BF8.l
071490  20 6E 00 08                   MOVEA.l $8(a6),a0
071494  20 28 00 14                   MOVE.l $14(a0),d0
071498  0C 80 00 00 03 ED             CMPI.l #$3ED,d0
07149E  67 00 00 EC                   BEQ $07158C
0714A2  0C 80 00 00 03 EE             CMPI.l #$3EE,d0
0714A8  66 00 01 18                   BNE $0715C2
0714AC  20 6E 00 08                   MOVEA.l $8(a6),a0
0714B0  20 28 00 10                   MOVE.l $10(a0),d0
0714B4  2D 40 FF FC                   MOVE.l d0,-$4(a6)
0714B8  52 80                         ADDQ.l #1,d0
0714BA  66 0E                         BNE $0714CA
0714BC  70 07                         MOVEQ #$7,d0
0714BE  23 C0 00 07 1B F8             MOVE.l d0,$71BF8.l
0714C4  70 00                         MOVEQ #$0,d0
0714C6  4E 5E                         UNLK a6
0714C8  4E 75                         RTS
0714CA  20 6E 00 08                   MOVEA.l $8(a6),a0
0714CE  20 10                         MOVE.l (a0),d0
0714D0  06 80 00 00 01 FF             ADDI.l #$1FF,d0
0714D6  72 09                         MOVEQ #$9,d1
0714D8  E2 A0                         ASR.l d1,d0
0714DA  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0714DE  20 2E FF FC                   MOVE.l -$4(a6),d0
0714E2  72 1A                         MOVEQ #$1A,d1
0714E4  4E B9 00 07 20 5C             JSR $7205C.l
0714EA  20 79 00 07 1C 00             MOVEA.l $71C00.l,a0
0714F0  D0 FC 00 10                   ADDA.w #$10,a0
0714F4  D1 C0                         ADDA.l d0,a0
0714F6  22 2E FF F8                   MOVE.l -$8(a6),d1
0714FA  21 41 00 04                   MOVE.l d1,$4(a0)
0714FE  20 79 00 07 1C 00             MOVEA.l $71C00.l,a0
071504  D0 FC 00 10                   ADDA.w #$10,a0
071508  D1 C0                         ADDA.l d0,a0
07150A  22 6E 00 08                   MOVEA.l $8(a6),a1
07150E  20 A9 00 04                   MOVE.l $4(a1),(a0)
071512  20 79 00 07 1C 00             MOVEA.l $71C00.l,a0
071518  20 28 00 0C                   MOVE.l $C(a0),d0
07151C  D2 80                         ADD.l d0,d1
07151E  20 79 00 07 1C 00             MOVEA.l $71C00.l,a0
071524  21 41 00 0C                   MOVE.l d1,$C(a0)
071528  B2 A8 00 08                   CMP.l $8(a0),d1
07152C  70 03                         MOVEQ #$3,d0
07152E  23 C0 00 07 1B F8             MOVE.l d0,$71BF8.l
071534  20 2E FF FC                   MOVE.l -$4(a6),d0
071538  72 1A                         MOVEQ #$1A,d1
07153A  4E B9 00 07 20 5C             JSR $7205C.l
071540  20 79 00 07 1C 00             MOVEA.l $71C00.l,a0
071546  D0 FC 00 10                   ADDA.w #$10,a0
07154A  D1 C0                         ADDA.l d0,a0
07154C  22 6E 00 08                   MOVEA.l $8(a6),a1
071550  21 51 00 08                   MOVE.l (a1),$8(a0)
071554  2F 09                         MOVE.l a1,-(a7)
071556  61 00 02 EC                   BSR $071844
07155A  58 8F                         ADDQ.l #4,a7
07155C  61 00 04 3A                   BSR $071998
071560  61 00 04 5C                   BSR $0719BE
071564  2F 3C 00 00 08 00             MOVE.l #$800,-(a7)
07156A  20 6E 00 08                   MOVEA.l $8(a6),a0
07156E  2F 28 00 1C                   MOVE.l $1C(a0),-(a7)
071572  4E B9 00 07 1E 28             JSR $71E28.l
071578  50 8F                         ADDQ.l #8,a7
07157A  70 2C                         MOVEQ #$2C,d0
07157C  2F 00                         MOVE.l d0,-(a7)
07157E  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
071582  4E B9 00 07 1E 28             JSR $71E28.l
071588  50 8F                         ADDQ.l #8,a7
07158A  60 44                         BRA $0715D0
07158C  20 6E 00 08                   MOVEA.l $8(a6),a0
071590  2D 68 00 10 FF FC             MOVE.l $10(a0),-$4(a6)
071596  61 00 04 26                   BSR $0719BE
07159A  2F 3C 00 00 08 00             MOVE.l #$800,-(a7)
0715A0  20 6E 00 08                   MOVEA.l $8(a6),a0
0715A4  2F 28 00 1C                   MOVE.l $1C(a0),-(a7)
0715A8  4E B9 00 07 1E 28             JSR $71E28.l
0715AE  50 8F                         ADDQ.l #8,a7
0715B0  70 2C                         MOVEQ #$2C,d0
0715B2  2F 00                         MOVE.l d0,-(a7)
0715B4  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0715B8  4E B9 00 07 1E 28             JSR $71E28.l
0715BE  50 8F                         ADDQ.l #8,a7
0715C0  60 0E                         BRA $0715D0
0715C2  70 05                         MOVEQ #$5,d0
0715C4  23 C0 00 07 1B F8             MOVE.l d0,$71BF8.l
0715CA  70 00                         MOVEQ #$0,d0
0715CC  4E 5E                         UNLK a6
0715CE  4E 75                         RTS
0715D0  70 01                         MOVEQ #$1,d0
0715D2  4E 5E                         UNLK a6
0715D4  4E 75                         RTS

; --- _MFRead  $0715D6 — mini-filesystem: read bytes, refilling from raw disk sectors. ---
0715D6  4E 56 FF EC                   LINK a6,#-$14
0715DA  70 00                         MOVEQ #$0,d0
0715DC  2D 40 FF F0                   MOVE.l d0,-$10(a6)
0715E0  2D 40 FF F4                   MOVE.l d0,-$C(a6)
0715E4  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0715E8  2D 40 FF FC                   MOVE.l d0,-$4(a6)
0715EC  20 6E 00 08                   MOVEA.l $8(a6),a0
0715F0  4A 90                         TST.l (a0)
0715F2  66 58                         BNE $07164C
0715F4  70 09                         MOVEQ #$9,d0
0715F6  22 2E 00 10                   MOVE.l $10(a6),d1
0715FA  E0 A1                         ASR.l d0,d1
0715FC  2D 41 FF F4                   MOVE.l d1,-$C(a6)
071600  E1 A1                         ASL.l d0,d1
071602  20 2E 00 10                   MOVE.l $10(a6),d0
071606  02 80 00 00 01 FF             ANDI.l #$1FF,d0
07160C  2D 40 FF F8                   MOVE.l d0,-$8(a6)
071610  2D 41 FF F0                   MOVE.l d1,-$10(a6)
071614  4A AE FF F4                   TST.l -$C(a6)
071618  67 32                         BEQ $07164C
07161A  2F 2E FF F4                   MOVE.l -$C(a6),-(a7)
07161E  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
071622  2F 28 00 04                   MOVE.l $4(a0),-(a7)
071626  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
07162C  4E B9 00 07 1C BE             JSR $71CBE.l
071632  4F EF 00 10                   LEA $10(a7),a7
071636  20 2E FF F0                   MOVE.l -$10(a6),d0
07163A  20 6E 00 08                   MOVEA.l $8(a6),a0
07163E  20 80                         MOVE.l d0,(a0)
071640  91 AE 00 10                   SUB.l d0,$10(a6)
071644  D1 AE FF FC                   ADD.l d0,-$4(a6)
071648  D1 AE 00 0C                   ADD.l d0,$C(a6)
07164C  20 2E 00 10                   MOVE.l $10(a6),d0
071650  2D 40 FF EC                   MOVE.l d0,-$14(a6)
071654  4A 80                         TST.l d0
071656  66 06                         BNE $07165E
071658  70 00                         MOVEQ #$0,d0
07165A  4E 5E                         UNLK a6
07165C  4E 75                         RTS
07165E  20 6E 00 08                   MOVEA.l $8(a6),a0
071662  20 10                         MOVE.l (a0),d0
071664  B0 A8 00 0C                   CMP.l $C(a0),d0
071668  6C 40                         BGE $0716AA
07166A  22 6E 00 08                   MOVEA.l $8(a6),a1
07166E  20 69 00 18                   MOVEA.l $18(a1),a0
071672  B1 E9 00 20                   CMPA.l $20(a1),a0
071676  66 08                         BNE $071680
071678  2F 09                         MOVE.l a1,-(a7)
07167A  61 00 01 6E                   BSR $0717EA
07167E  58 8F                         ADDQ.l #4,a7
071680  22 6E 00 08                   MOVEA.l $8(a6),a1
071684  20 69 00 18                   MOVEA.l $18(a1),a0
071688  52 A9 00 18                   ADDQ.l #1,$18(a1)
07168C  22 6E 00 0C                   MOVEA.l $C(a6),a1
071690  12 90                         MOVE.b (a0),(a1)
071692  52 AE 00 0C                   ADDQ.l #1,$C(a6)
071696  53 AE FF EC                   SUBQ.l #1,-$14(a6)
07169A  52 AE FF FC                   ADDQ.l #1,-$4(a6)
07169E  20 6E 00 08                   MOVEA.l $8(a6),a0
0716A2  52 90                         ADDQ.l #1,(a0)
0716A4  4A AE FF EC                   TST.l -$14(a6)
0716A8  6E B4                         BGT $07165E
0716AA  20 2E FF FC                   MOVE.l -$4(a6),d0
0716AE  4E 5E                         UNLK a6
0716B0  4E 75                         RTS

; --- _MFWrite  $0716B2 —  ---
0716B2  4E 56 FF F0                   LINK a6,#-$10
0716B6  70 00                         MOVEQ #$0,d0
0716B8  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0716BC  2D 40 FF FC                   MOVE.l d0,-$4(a6)
0716C0  20 6E 00 08                   MOVEA.l $8(a6),a0
0716C4  4A 90                         TST.l (a0)
0716C6  66 4A                         BNE $071712
0716C8  70 09                         MOVEQ #$9,d0
0716CA  22 2E 00 10                   MOVE.l $10(a6),d1
0716CE  E0 A1                         ASR.l d0,d1
0716D0  2D 41 FF F8                   MOVE.l d1,-$8(a6)
0716D4  E1 A1                         ASL.l d0,d1
0716D6  2D 41 FF F4                   MOVE.l d1,-$C(a6)
0716DA  4A AE FF F8                   TST.l -$8(a6)
0716DE  67 32                         BEQ $071712
0716E0  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
0716E4  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
0716E8  2F 28 00 04                   MOVE.l $4(a0),-(a7)
0716EC  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
0716F2  4E B9 00 07 1D 04             JSR $71D04.l
0716F8  4F EF 00 10                   LEA $10(a7),a7
0716FC  20 2E FF F4                   MOVE.l -$C(a6),d0
071700  20 6E 00 08                   MOVEA.l $8(a6),a0
071704  20 80                         MOVE.l d0,(a0)
071706  91 AE 00 10                   SUB.l d0,$10(a6)
07170A  D1 AE FF FC                   ADD.l d0,-$4(a6)
07170E  D1 AE 00 0C                   ADD.l d0,$C(a6)
071712  20 2E 00 10                   MOVE.l $10(a6),d0
071716  2D 40 FF F0                   MOVE.l d0,-$10(a6)
07171A  4A 80                         TST.l d0
07171C  66 06                         BNE $071724
07171E  70 00                         MOVEQ #$0,d0
071720  4E 5E                         UNLK a6
071722  4E 75                         RTS
071724  22 6E 00 08                   MOVEA.l $8(a6),a1
071728  20 69 00 18                   MOVEA.l $18(a1),a0
07172C  B1 E9 00 20                   CMPA.l $20(a1),a0
071730  66 08                         BNE $07173A
071732  2F 09                         MOVE.l a1,-(a7)
071734  61 00 01 0E                   BSR $071844
071738  58 8F                         ADDQ.l #4,a7
07173A  22 6E 00 08                   MOVEA.l $8(a6),a1
07173E  20 69 00 18                   MOVEA.l $18(a1),a0
071742  52 A9 00 18                   ADDQ.l #1,$18(a1)
071746  22 6E 00 0C                   MOVEA.l $C(a6),a1
07174A  10 91                         MOVE.b (a1),(a0)
07174C  52 AE 00 0C                   ADDQ.l #1,$C(a6)
071750  53 AE FF F0                   SUBQ.l #1,-$10(a6)
071754  52 AE FF FC                   ADDQ.l #1,-$4(a6)
071758  20 6E 00 08                   MOVEA.l $8(a6),a0
07175C  52 90                         ADDQ.l #1,(a0)
07175E  4A AE FF F0                   TST.l -$10(a6)
071762  6E C0                         BGT $071724
071764  20 2E FF FC                   MOVE.l -$4(a6),d0
071768  4E 5E                         UNLK a6
07176A  4E 75                         RTS

; --- _MFSeek  $07176C — mini-filesystem: seek within a file. ---
07176C  4E 56 FF FC                   LINK a6,#-$4
071770  20 6E 00 08                   MOVEA.l $8(a6),a0
071774  2D 50 FF FC                   MOVE.l (a0),-$4(a6)
071778  0C A8 00 00 03 ED 00 14       CMPI.l #$3ED,$14(a0)
071780  67 06                         BEQ $071788
071782  70 FF                         MOVEQ #$FF,d0
071784  4E 5E                         UNLK a6
071786  4E 75                         RTS
071788  20 2E 00 10                   MOVE.l $10(a6),d0
07178C  0C 80 00 00 00 01             CMPI.l #$1,d0
071792  67 22                         BEQ $0717B6
071794  4A 80                         TST.l d0
071796  67 12                         BEQ $0717AA
071798  0C 80 FF FF FF FF             CMPI.l #$FFFFFFFF,d0
07179E  66 26                         BNE $0717C6
0717A0  20 6E 00 08                   MOVEA.l $8(a6),a0
0717A4  20 AE 00 0C                   MOVE.l $C(a6),(a0)
0717A8  60 22                         BRA $0717CC
0717AA  20 2E 00 0C                   MOVE.l $C(a6),d0
0717AE  20 6E 00 08                   MOVEA.l $8(a6),a0
0717B2  D1 90                         ADD.l d0,(a0)
0717B4  60 16                         BRA $0717CC
0717B6  20 6E 00 08                   MOVEA.l $8(a6),a0
0717BA  20 28 00 0C                   MOVE.l $C(a0),d0
0717BE  D0 AE 00 0C                   ADD.l $C(a6),d0
0717C2  20 80                         MOVE.l d0,(a0)
0717C4  60 06                         BRA $0717CC
0717C6  70 FF                         MOVEQ #$FF,d0
0717C8  4E 5E                         UNLK a6
0717CA  4E 75                         RTS
0717CC  20 6E 00 08                   MOVEA.l $8(a6),a0
0717D0  21 68 00 20 00 18             MOVE.l $20(a0),$18(a0)
0717D6  20 28 00 0C                   MOVE.l $C(a0),d0
0717DA  22 10                         MOVE.l (a0),d1
0717DC  B2 80                         CMP.l d0,d1
0717DE  6F 02                         BLE $0717E2
0717E0  20 80                         MOVE.l d0,(a0)
0717E2  20 2E FF FC                   MOVE.l -$4(a6),d0
0717E6  4E 5E                         UNLK a6
0717E8  4E 75                         RTS

; ==== _MFillBuf  $0717EA  (1 caller) —  ====
0717EA  4E 56 FF FC                   LINK a6,#-$4
0717EE  70 09                         MOVEQ #$9,d0
0717F0  20 6E 00 08                   MOVEA.l $8(a6),a0
0717F4  22 10                         MOVE.l (a0),d1
0717F6  E0 A1                         ASR.l d0,d1
0717F8  D2 A8 00 04                   ADD.l $4(a0),d1
0717FC  70 04                         MOVEQ #$4,d0
0717FE  2F 00                         MOVE.l d0,-(a7)
071800  2F 28 00 1C                   MOVE.l $1C(a0),-(a7)
071804  2F 01                         MOVE.l d1,-(a7)
071806  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
07180C  2D 41 FF FC                   MOVE.l d1,-$4(a6)
071810  4E B9 00 07 1C BE             JSR $71CBE.l
071816  4F EF 00 10                   LEA $10(a7),a7
07181A  20 6E 00 08                   MOVEA.l $8(a6),a0
07181E  20 10                         MOVE.l (a0),d0
071820  02 80 00 00 01 FF             ANDI.l #$1FF,d0
071826  20 68 00 1C                   MOVEA.l $1C(a0),a0
07182A  D1 C0                         ADDA.l d0,a0
07182C  22 6E 00 08                   MOVEA.l $8(a6),a1
071830  23 48 00 18                   MOVE.l a0,$18(a1)
071834  20 69 00 1C                   MOVEA.l $1C(a1),a0
071838  D0 FC 08 00                   ADDA.w #$800,a0
07183C  23 48 00 20                   MOVE.l a0,$20(a1)
071840  4E 5E                         UNLK a6
071842  4E 75                         RTS

; ==== _MFlushBuf  $071844  (2 callers) —  ====
071844  4E 56 FF F4                   LINK a6,#-$C
071848  48 E7 20 00                   MOVEM.l d2,-(a7)
07184C  22 6E 00 08                   MOVEA.l $8(a6),a1
071850  20 69 00 18                   MOVEA.l $18(a1),a0
071854  B1 E9 00 1C                   CMPA.l $1C(a1),a0
071858  66 0A                         BNE $071864
07185A  70 00                         MOVEQ #$0,d0
07185C  4C DF                         .dc.w $4CDF
07185E  .dc.b 00 04 4E 5E 4E 75                               ; ..N^Nu
071864  20 6E 00 08                   MOVEA.l $8(a6),a0
071868  20 28 00 18                   MOVE.l $18(a0),d0
07186C  90 A8 00 1C                   SUB.l $1C(a0),d0
071870  22 10                         MOVE.l (a0),d1
071872  92 80                         SUB.l d0,d1
071874  74 09                         MOVEQ #$9,d2
071876  2D 41 FF F4                   MOVE.l d1,-$C(a6)
07187A  E4 A1                         ASR.l d2,d1
07187C  D2 A8 00 04                   ADD.l $4(a0),d1
071880  06 80 00 00 01 FF             ADDI.l #$1FF,d0
071886  E4 A0                         ASR.l d2,d0
071888  2F 00                         MOVE.l d0,-(a7)
07188A  2F 28 00 1C                   MOVE.l $1C(a0),-(a7)
07188E  2F 01                         MOVE.l d1,-(a7)
071890  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
071896  2D 40 FF F8                   MOVE.l d0,-$8(a6)
07189A  2D 41 FF FC                   MOVE.l d1,-$4(a6)
07189E  4E B9 00 07 1D 04             JSR $71D04.l
0718A4  4F EF 00 10                   LEA $10(a7),a7
0718A8  22 6E 00 08                   MOVEA.l $8(a6),a1
0718AC  20 69 00 1C                   MOVEA.l $1C(a1),a0
0718B0  23 48 00 18                   MOVE.l a0,$18(a1)
0718B4  20 69 00 1C                   MOVEA.l $1C(a1),a0
0718B8  D0 FC 08 00                   ADDA.w #$800,a0
0718BC  23 48 00 20                   MOVE.l a0,$20(a1)
0718C0  4C DF                         .dc.w $4CDF
0718C2  .dc.b 00 04 4E 5E 4E 75                               ; ..N^Nu

; --- _MFDrive  $0718C8 —  ---
0718C8  4E 56 00 00                   LINK a6,#$0
0718CC  23 EE 00 08 00 07 1C 04       MOVE.l $8(a6),$71C04.l
0718D4  70 00                         MOVEQ #$0,d0
0718D6  4E 5E                         UNLK a6
0718D8  4E 75                         RTS

; ==== _MGetDir  $0718DA  (2 callers) — mini-filesystem: read the volume directory off the disk. ====
0718DA  4E 56 FF F8                   LINK a6,#-$8
0718DE  70 00                         MOVEQ #$0,d0
0718E0  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0718E4  2D 40 FF FC                   MOVE.l d0,-$4(a6)
0718E8  20 39 00 07 1C 04             MOVE.l $71C04.l,d0
0718EE  0C 80 FF FF FF FF             CMPI.l #$FFFFFFFF,d0
0718F4  67 04                         BEQ $0718FA
0718F6  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0718FA  4A B9 00 07 1C 00             TST.l $71C00.l
071900  67 06                         BEQ $071908
071902  70 00                         MOVEQ #$0,d0
071904  4E 5E                         UNLK a6
071906  4E 75                         RTS
071908  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
07190E  70 38                         MOVEQ #$38,d0
071910  2F 00                         MOVE.l d0,-(a7)
071912  4E B9 00 07 1E 10             JSR $71E10.l
071918  50 8F                         ADDQ.l #8,a7
07191A  23 C0 00 07 1B FC             MOVE.l d0,$71BFC.l
071920  2F 00                         MOVE.l d0,-(a7)
071922  4E B9 00 07 1C 1C             JSR $71C1C.l
071928  58 8F                         ADDQ.l #4,a7
07192A  42 A7                         CLR.l -(a7)
07192C  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
071932  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
071936  48 79 00 07 1C 08             PEA $71C08.l
07193C  4E B9 00 07 1E A8             JSR $71EA8.l
071942  4F EF 00 10                   LEA $10(a7),a7
071946  2D 40 FF FC                   MOVE.l d0,-$4(a6)
07194A  4A 80                         TST.l d0
07194C  67 04                         BEQ $071952
07194E  4E 5E                         UNLK a6
071950  4E 75                         RTS
071952  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
071958  2F 3C 00 00 04 00             MOVE.l #$400,-(a7)
07195E  4E B9 00 07 1E 10             JSR $71E10.l
071964  50 8F                         ADDQ.l #8,a7
071966  23 C0 00 07 1C 00             MOVE.l d0,$71C00.l
07196C  4A 80                         TST.l d0
07196E  66 06                         BNE $071976
071970  70 06                         MOVEQ #$6,d0
071972  4E 5E                         UNLK a6
071974  4E 75                         RTS
071976  70 02                         MOVEQ #$2,d0
071978  2F 00                         MOVE.l d0,-(a7)
07197A  2F 39 00 07 1C 00             MOVE.l $71C00.l,-(a7)
071980  70 16                         MOVEQ #$16,d0
071982  2F 00                         MOVE.l d0,-(a7)
071984  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
07198A  4E B9 00 07 1C BE             JSR $71CBE.l
071990  4F EF 00 10                   LEA $10(a7),a7
071994  4E 5E                         UNLK a6
071996  4E 75                         RTS

; ==== _MFlushDir  $071998  (1 caller) —  ====
071998  4E 56 FF FC                   LINK a6,#-$4
07199C  70 02                         MOVEQ #$2,d0
07199E  2F 00                         MOVE.l d0,-(a7)
0719A0  2F 39 00 07 1C 00             MOVE.l $71C00.l,-(a7)
0719A6  70 16                         MOVEQ #$16,d0
0719A8  2F 00                         MOVE.l d0,-(a7)
0719AA  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
0719B0  4E B9 00 07 1D 04             JSR $71D04.l
0719B6  4F EF 00 10                   LEA $10(a7),a7
0719BA  4E 5E                         UNLK a6
0719BC  4E 75                         RTS

; ==== _MCloseDir  $0719BE  (4 callers) —  ====
0719BE  4A B9 00 07 1C 00             TST.l $71C00.l
0719C4  66 04                         BNE $0719CA
0719C6  70 01                         MOVEQ #$1,d0
0719C8  4E 75                         RTS
0719CA  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
0719D0  4E B9 00 07 1D 4A             JSR $71D4A.l
0719D6  58 8F                         ADDQ.l #4,a7
0719D8  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
0719DE  4E B9 00 07 1D 78             JSR $71D78.l
0719E4  58 8F                         ADDQ.l #4,a7
0719E6  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
0719EC  4E B9 00 07 1E C8             JSR $71EC8.l
0719F2  58 8F                         ADDQ.l #4,a7
0719F4  2F 39 00 07 1B FC             MOVE.l $71BFC.l,-(a7)
0719FA  4E B9 00 07 1C 72             JSR $71C72.l
071A00  58 8F                         ADDQ.l #4,a7
071A02  2F 3C 00 00 04 00             MOVE.l #$400,-(a7)
071A08  2F 39 00 07 1C 00             MOVE.l $71C00.l,-(a7)
071A0E  4E B9 00 07 1E 28             JSR $71E28.l
071A14  50 8F                         ADDQ.l #8,a7
071A16  91 C8                         SUBA.l a0,a0
071A18  23 C8 00 07 1C 00             MOVE.l a0,$71C00.l
071A1E  20 08                         MOVE.l a0,d0
071A20  4E 75                         RTS

; ==== _MFindEntry  $071A22  (2 callers) — mini-filesystem: look a name up in the directory. ====
071A22  4E 56 FF FC                   LINK a6,#-$4
071A26  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
071A2A  61 00 00 E4                   BSR $071B10
071A2E  58 8F                         ADDQ.l #4,a7
071A30  42 AE FF FC                   CLR.l -$4(a6)
071A34  20 2E FF FC                   MOVE.l -$4(a6),d0
071A38  0C 80 00 00 00 26             CMPI.l #$26,d0
071A3E  6C 34                         BGE $071A74
071A40  72 1A                         MOVEQ #$1A,d1
071A42  4E B9 00 07 20 5C             JSR $7205C.l
071A48  20 6E 00 08                   MOVEA.l $8(a6),a0
071A4C  D0 FC 00 10                   ADDA.w #$10,a0
071A50  D1 C0                         ADDA.l d0,a0
071A52  D0 FC 00 0C                   ADDA.w #$C,a0
071A56  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
071A5A  2F 08                         MOVE.l a0,-(a7)
071A5C  61 00 00 F4                   BSR $071B52
071A60  50 8F                         ADDQ.l #8,a7
071A62  4A 80                         TST.l d0
071A64  66 08                         BNE $071A6E
071A66  20 2E FF FC                   MOVE.l -$4(a6),d0
071A6A  4E 5E                         UNLK a6
071A6C  4E 75                         RTS
071A6E  52 AE FF FC                   ADDQ.l #1,-$4(a6)
071A72  60 C0                         BRA $071A34
071A74  70 FF                         MOVEQ #$FF,d0
071A76  4E 5E                         UNLK a6
071A78  4E 75                         RTS

; ==== _MAddEntry  $071A7A  (1 caller) —  ====
071A7A  4E 56 FF FC                   LINK a6,#-$4
071A7E  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
071A82  61 00 00 8C                   BSR $071B10
071A86  58 8F                         ADDQ.l #4,a7
071A88  42 AE FF FC                   CLR.l -$4(a6)
071A8C  20 2E FF FC                   MOVE.l -$4(a6),d0
071A90  0C 80 00 00 00 26             CMPI.l #$26,d0
071A96  6C 72                         BGE $071B0A
071A98  72 1A                         MOVEQ #$1A,d1
071A9A  4E B9 00 07 20 5C             JSR $7205C.l
071AA0  20 6E 00 08                   MOVEA.l $8(a6),a0
071AA4  D0 FC 00 10                   ADDA.w #$10,a0
071AA8  D1 C0                         ADDA.l d0,a0
071AAA  4A A8 00 08                   TST.l $8(a0)
071AAE  66 54                         BNE $071B04
071AB0  20 6E 00 08                   MOVEA.l $8(a6),a0
071AB4  D0 FC 00 10                   ADDA.w #$10,a0
071AB8  D1 C0                         ADDA.l d0,a0
071ABA  D0 FC 00 0C                   ADDA.w #$C,a0
071ABE  2F 08                         MOVE.l a0,-(a7)
071AC0  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
071AC4  61 00 00 FC                   BSR $071BC2
071AC8  50 8F                         ADDQ.l #8,a7
071ACA  20 2E FF FC                   MOVE.l -$4(a6),d0
071ACE  72 1A                         MOVEQ #$1A,d1
071AD0  4E B9 00 07 20 5C             JSR $7205C.l
071AD6  20 6E 00 08                   MOVEA.l $8(a6),a0
071ADA  D0 FC 00 10                   ADDA.w #$10,a0
071ADE  D1 C0                         ADDA.l d0,a0
071AE0  22 6E 00 08                   MOVEA.l $8(a6),a1
071AE4  20 A9 00 0C                   MOVE.l $C(a1),(a0)
071AE8  20 2E FF FC                   MOVE.l -$4(a6),d0
071AEC  4E B9 00 07 20 5C             JSR $7205C.l
071AF2  D2 FC 00 10                   ADDA.w #$10,a1
071AF6  D3 C0                         ADDA.l d0,a1
071AF8  42 A9 00 04                   CLR.l $4(a1)
071AFC  20 2E FF FC                   MOVE.l -$4(a6),d0
071B00  4E 5E                         UNLK a6
071B02  4E 75                         RTS
071B04  52 AE FF FC                   ADDQ.l #1,-$4(a6)
071B08  60 82                         BRA $071A8C
071B0A  70 FF                         MOVEQ #$FF,d0
071B0C  4E 5E                         UNLK a6
071B0E  4E 75                         RTS

; ==== _toupper  $071B10  (2 callers) —  ====
071B10  4E 56 FF FC                   LINK a6,#-$4
071B14  42 AE FF FC                   CLR.l -$4(a6)
071B18  20 6E 00 08                   MOVEA.l $8(a6),a0
071B1C  D1 EE FF FC                   ADDA.l -$4(a6),a0
071B20  10 10                         MOVE.b (a0),d0
071B22  0C 00 00 61                   CMPI.b #$61,d0
071B26  6D 16                         BLT $071B3E
071B28  0C 00 00 7A                   CMPI.b #$7A,d0
071B2C  6E 10                         BGT $071B3E
071B2E  20 6E 00 08                   MOVEA.l $8(a6),a0
071B32  D1 EE FF FC                   ADDA.l -$4(a6),a0
071B36  10 10                         MOVE.b (a0),d0
071B38  04 00 00 20                   SUBI.b #$20,d0
071B3C  10 80                         MOVE.b d0,(a0)
071B3E  52 AE FF FC                   ADDQ.l #1,-$4(a6)
071B42  20 6E 00 08                   MOVEA.l $8(a6),a0
071B46  D1 EE FF FC                   ADDA.l -$4(a6),a0
071B4A  4A 10                         TST.b (a0)
071B4C  66 CA                         BNE $071B18
071B4E  4E 5E                         UNLK a6
071B50  4E 75                         RTS

; ==== _strcmp  $071B52  (1 caller) —  ====
071B52  4E 56 FF FC                   LINK a6,#-$4
071B56  42 AE FF FC                   CLR.l -$4(a6)
071B5A  20 2E FF FC                   MOVE.l -$4(a6),d0
071B5E  20 6E 00 08                   MOVEA.l $8(a6),a0
071B62  D1 C0                         ADDA.l d0,a0
071B64  12 10                         MOVE.b (a0),d1
071B66  4A 01                         TST.b d1
071B68  66 16                         BNE $071B80
071B6A  20 6E 00 0C                   MOVEA.l $C(a6),a0
071B6E  D1 C0                         ADDA.l d0,a0
071B70  10 10                         MOVE.b (a0),d0
071B72  4A 00                         TST.b d0
071B74  66 04                         BNE $071B7A
071B76  70 00                         MOVEQ #$0,d0
071B78  60 02                         BRA $071B7C
071B7A  70 FF                         MOVEQ #$FF,d0
071B7C  4E 5E                         UNLK a6
071B7E  4E 75                         RTS
071B80  20 2E FF FC                   MOVE.l -$4(a6),d0
071B84  20 6E 00 08                   MOVEA.l $8(a6),a0
071B88  D1 C0                         ADDA.l d0,a0
071B8A  22 6E 00 0C                   MOVEA.l $C(a6),a1
071B8E  D3 C0                         ADDA.l d0,a1
071B90  10 10                         MOVE.b (a0),d0
071B92  12 11                         MOVE.b (a1),d1
071B94  B0 01                         CMP.b d1,d0
071B96  6F 06                         BLE $071B9E
071B98  70 01                         MOVEQ #$1,d0
071B9A  4E 5E                         UNLK a6
071B9C  4E 75                         RTS
071B9E  20 2E FF FC                   MOVE.l -$4(a6),d0
071BA2  20 6E 00 08                   MOVEA.l $8(a6),a0
071BA6  D1 C0                         ADDA.l d0,a0
071BA8  22 6E 00 0C                   MOVEA.l $C(a6),a1
071BAC  D3 C0                         ADDA.l d0,a1
071BAE  10 10                         MOVE.b (a0),d0
071BB0  12 11                         MOVE.b (a1),d1
071BB2  B0 01                         CMP.b d1,d0
071BB4  6C 06                         BGE $071BBC
071BB6  70 FF                         MOVEQ #$FF,d0
071BB8  4E 5E                         UNLK a6
071BBA  4E 75                         RTS
071BBC  52 AE FF FC                   ADDQ.l #1,-$4(a6)
071BC0  60 98                         BRA $071B5A

; ==== _strcopy  $071BC2  (1 caller) —  ====
071BC2  4E 56 FF FC                   LINK a6,#-$4
071BC6  42 AE FF FC                   CLR.l -$4(a6)
071BCA  20 6E 00 08                   MOVEA.l $8(a6),a0
071BCE  22 6E 00 0C                   MOVEA.l $C(a6),a1
071BD2  12 90                         MOVE.b (a0),(a1)
071BD4  52 AE 00 08                   ADDQ.l #1,$8(a6)
071BD8  52 AE 00 0C                   ADDQ.l #1,$C(a6)
071BDC  52 AE FF FC                   ADDQ.l #1,-$4(a6)
071BE0  20 6E 00 08                   MOVEA.l $8(a6),a0
071BE4  10 10                         MOVE.b (a0),d0
071BE6  4A 00                         TST.b d0
071BE8  66 E0                         BNE $071BCA
071BEA  20 6E 00 0C                   MOVEA.l $C(a6),a0
071BEE  42 10                         CLR.b (a0)
071BF0  20 2E FF FC                   MOVE.l -$4(a6),d0
071BF4  4E 5E                         UNLK a6
071BF6  4E 75                         RTS

; --- _MERR  $071BF8 —  (data) ---
071BF8  .dc.b 00 00 00 00                                     ; ....

; --- _mf_ios  $071BFC —  (data) ---
071BFC  .dc.b 00 00 00 00                                     ; ....

; --- _minidir  $071C00 —  (data) ---
071C00  .dc.b 00 00 00 00                                     ; ....

; --- _MF_UseDrive  $071C04 —  (data) ---
071C04  .dc.b FF FF FF FF 74 72 61 63 6B 64 69 73 6B 2E 64 65 ; ....trackdisk.de
071C14  .dc.b 76 69 63 65 00 00 00 00                         ; vice....

; ==== MotorOff  $071C1C  (1 caller) — spin the floppy motor down. ====
071C1C  4E 56 FF FC                   LINK a6,#-$4
071C20  2F 3C 00 01 00 00             MOVE.l #$10000,-(a7)
071C26  70 22                         MOVEQ #$22,d0
071C28  2F 00                         MOVE.l d0,-(a7)
071C2A  4E B9 00 07 1E 10             JSR $71E10.l
071C30  50 8F                         ADDQ.l #8,a7
071C32  42 A7                         CLR.l -(a7)
071C34  2D 40 FF FC                   MOVE.l d0,-$4(a6)
071C38  4E B9 00 07 1E 40             JSR $71E40.l
071C3E  58 8F                         ADDQ.l #4,a7
071C40  20 6E FF FC                   MOVEA.l -$4(a6),a0
071C44  21 40 00 10                   MOVE.l d0,$10(a0)
071C48  4E B9 00 07 1E 54             JSR $71E54.l
071C4E  20 6E FF FC                   MOVEA.l -$4(a6),a0
071C52  11 40 00 0F                   MOVE.b d0,$F(a0)
071C56  70 00                         MOVEQ #$0,d0
071C58  11 40 00 0E                   MOVE.b d0,$E(a0)
071C5C  11 40 00 09                   MOVE.b d0,$9(a0)
071C60  22 6E 00 08                   MOVEA.l $8(a6),a1
071C64  23 48 00 0E                   MOVE.l a0,$E(a1)
071C68  33 7C 00 38 00 12             MOVE.w #$38,$12(a1)
071C6E  4E 5E                         UNLK a6
071C70  4E 75                         RTS

; ==== DeleteStdIO  $071C72  (1 caller) —  ====
071C72  4E 56 FF FC                   LINK a6,#-$4
071C76  20 6E 00 08                   MOVEA.l $8(a6),a0
071C7A  2D 68 00 0E FF FC             MOVE.l $E(a0),-$4(a6)
071C80  70 00                         MOVEQ #$0,d0
071C82  22 6E FF FC                   MOVEA.l -$4(a6),a1
071C86  10 29 00 0F                   MOVE.b $F(a1),d0
071C8A  2F 00                         MOVE.l d0,-(a7)
071C8C  2F 29 00 10                   MOVE.l $10(a1),-(a7)
071C90  4E B9 00 07 1E 68             JSR $71E68.l
071C96  50 8F                         ADDQ.l #8,a7
071C98  70 22                         MOVEQ #$22,d0
071C9A  2F 00                         MOVE.l d0,-(a7)
071C9C  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
071CA0  4E B9 00 07 1E 28             JSR $71E28.l
071CA6  50 8F                         ADDQ.l #8,a7
071CA8  70 38                         MOVEQ #$38,d0
071CAA  2F 00                         MOVE.l d0,-(a7)
071CAC  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
071CB0  4E B9 00 07 1E 28             JSR $71E28.l
071CB6  50 8F                         ADDQ.l #8,a7
071CB8  70 00                         MOVEQ #$0,d0
071CBA  4E 5E                         UNLK a6
071CBC  4E 75                         RTS

; ==== ReadSecs  $071CBE  (3 callers) — read raw sectors via trackdisk.device (CMD_READ + DoIO). ====
071CBE  4E 56 FF FC                   LINK a6,#-$4
071CC2  20 6E 00 10                   MOVEA.l $10(a6),a0
071CC6  C1 88                         EXG d0,a0
071CC8  02 40 FF FE                   ANDI.w #$FFFE,d0
071CCC  C1 88                         EXG d0,a0
071CCE  22 6E 00 08                   MOVEA.l $8(a6),a1
071CD2  23 48 00 28                   MOVE.l a0,$28(a1)
071CD6  33 7C 00 02 00 1C             MOVE.w #$2,$1C(a1)
071CDC  70 09                         MOVEQ #$9,d0
071CDE  22 2E 00 0C                   MOVE.l $C(a6),d1
071CE2  E1 A1                         ASL.l d0,d1
071CE4  23 41 00 2C                   MOVE.l d1,$2C(a1)
071CE8  22 2E 00 14                   MOVE.l $14(a6),d1
071CEC  E1 A1                         ASL.l d0,d1
071CEE  23 41 00 24                   MOVE.l d1,$24(a1)
071CF2  42 29 00 1E                   CLR.b $1E(a1)
071CF6  2F 09                         MOVE.l a1,-(a7)
071CF8  4E B9 00 07 1E DC             JSR $71EDC.l
071CFE  58 8F                         ADDQ.l #4,a7
071D00  4E 5E                         UNLK a6
071D02  4E 75                         RTS

; ==== WriteSecs  $071D04  (3 callers) — write raw sectors via trackdisk.device. ====
071D04  4E 56 FF FC                   LINK a6,#-$4
071D08  20 6E 00 10                   MOVEA.l $10(a6),a0
071D0C  C1 88                         EXG d0,a0
071D0E  02 40 FF FE                   ANDI.w #$FFFE,d0
071D12  C1 88                         EXG d0,a0
071D14  22 6E 00 08                   MOVEA.l $8(a6),a1
071D18  23 48 00 28                   MOVE.l a0,$28(a1)
071D1C  33 7C 00 03 00 1C             MOVE.w #$3,$1C(a1)
071D22  70 09                         MOVEQ #$9,d0
071D24  22 2E 00 0C                   MOVE.l $C(a6),d1
071D28  E1 A1                         ASL.l d0,d1
071D2A  23 41 00 2C                   MOVE.l d1,$2C(a1)
071D2E  22 2E 00 14                   MOVE.l $14(a6),d1
071D32  E1 A1                         ASL.l d0,d1
071D34  23 41 00 24                   MOVE.l d1,$24(a1)
071D38  42 29 00 1E                   CLR.b $1E(a1)
071D3C  2F 09                         MOVE.l a1,-(a7)
071D3E  4E B9 00 07 1E DC             JSR $71EDC.l
071D44  58 8F                         ADDQ.l #4,a7
071D46  4E 5E                         UNLK a6
071D48  4E 75                         RTS

; ==== UpdateTD  $071D4A  (1 caller) —  ====
071D4A  4E 56 FF FC                   LINK a6,#-$4
071D4E  91 C8                         SUBA.l a0,a0
071D50  22 6E 00 08                   MOVEA.l $8(a6),a1
071D54  23 48 00 28                   MOVE.l a0,$28(a1)
071D58  33 7C 00 04 00 1C             MOVE.w #$4,$1C(a1)
071D5E  23 48 00 2C                   MOVE.l a0,$2C(a1)
071D62  23 48 00 24                   MOVE.l a0,$24(a1)
071D66  42 29 00 1E                   CLR.b $1E(a1)
071D6A  2F 09                         MOVE.l a1,-(a7)
071D6C  4E B9 00 07 1E DC             JSR $71EDC.l
071D72  58 8F                         ADDQ.l #4,a7
071D74  4E 5E                         UNLK a6
071D76  4E 75                         RTS

; ==== sub_071D78 (1 caller) ====
071D78  4E 56 FF FC                   LINK a6,#-$4
071D7C  91 C8                         SUBA.l a0,a0
071D7E  22 6E 00 08                   MOVEA.l $8(a6),a1
071D82  23 48 00 28                   MOVE.l a0,$28(a1)
071D86  33 7C 00 09 00 1C             MOVE.w #$9,$1C(a1)
071D8C  23 48 00 2C                   MOVE.l a0,$2C(a1)
071D90  23 48 00 24                   MOVE.l a0,$24(a1)
071D94  42 29 00 1E                   CLR.b $1E(a1)
071D98  2F 09                         MOVE.l a1,-(a7)
071D9A  4E B9 00 07 1E DC             JSR $71EDC.l
071DA0  58 8F                         ADDQ.l #4,a7
071DA2  4E 5E                         UNLK a6
071DA4  4E 75                         RTS
071DA6  .dc.b 00 00                                           ; ..

; ==== _Open  $071DA8  (1 caller) —  ====
071DA8  48 E7 20 02                   MOVEM.l d2/a6,-(a7)
071DAC  4C EF                         .dc.w $4CEF
071DAE  .dc.b 00 06 00 0C 2C 79 00 07 00 A8 4E AE FF E2 4C DF ; ....,y....N...L.
071DBE  .dc.b 40 04 4E 75 00 00                               ; @.Nu..

; ==== _Close  $071DC4  (1 caller) —  ====
071DC4  2F 0E                         MOVE.l a6,-(a7)
071DC6  22 2F 00 08                   MOVE.l $8(a7),d1
071DCA  2C 79 00 07 00 A8             MOVEA.l $700A8.l,a6
071DD0  4E AE FF DC                   JSR -$24(a6)
071DD4  2C 5F                         MOVEA.l (a7)+,a6
071DD6  4E 75                         RTS

; ==== _Read  $071DD8  (2 callers) —  ====
071DD8  48 E7 30 02                   MOVEM.l d2-d3/a6,-(a7)
071DDC  4C EF                         .dc.w $4CEF
071DDE  .dc.b 00 0E 00 10 2C 79 00 07 00 A8 4E AE FF D6 4C DF ; ....,y....N...L.
071DEE  .dc.b 40 0C 4E 75 00 00                               ; @.Nu..

; ==== _Seek  $071DF4  (4 callers) —  ====
071DF4  48 E7 30 02                   MOVEM.l d2-d3/a6,-(a7)
071DF8  4C EF                         .dc.w $4CEF
071DFA  .dc.b 00 0E 00 10 2C 79 00 07 00 A8 4E AE FF BE 4C DF ; ....,y....N...L.
071E0A  .dc.b 40 0C 4E 75 00 00                               ; @.Nu..

; ==== _AllocMem  $071E10  (8 callers) —  ====
071E10  2F 0E                         MOVE.l a6,-(a7)
071E12  4C EF                         .dc.w $4CEF
071E14  .dc.b 00 03 00 08 2C 79 00 07 00 64 4E AE FF 3A 2C 5F ; ....,y...dN..:,_
071E24  .dc.b 4E 75 00 00                                     ; Nu..

; ==== _FreeMem  $071E28  (9 callers) —  ====
071E28  2F 0E                         MOVE.l a6,-(a7)
071E2A  22 6F 00 08                   MOVEA.l $8(a7),a1
071E2E  20 2F 00 0C                   MOVE.l $C(a7),d0
071E32  2C 79 00 07 00 64             MOVEA.l $70064.l,a6
071E38  4E AE FF 2E                   JSR -$D2(a6)
071E3C  2C 5F                         MOVEA.l (a7)+,a6
071E3E  4E 75                         RTS

; ==== _FindTask  $071E40  (1 caller) —  ====
071E40  2F 0E                         MOVE.l a6,-(a7)
071E42  22 6F 00 08                   MOVEA.l $8(a7),a1
071E46  2C 79 00 07 00 64             MOVEA.l $70064.l,a6
071E4C  4E AE FE DA                   JSR -$126(a6)
071E50  2C 5F                         MOVEA.l (a7)+,a6
071E52  4E 75                         RTS

; ==== _AllocSignal  $071E54  (1 caller) —  ====
071E54  2F 0E                         MOVE.l a6,-(a7)
071E56  20 2F 00 08                   MOVE.l $8(a7),d0
071E5A  2C 79 00 07 00 64             MOVEA.l $70064.l,a6
071E60  4E AE FE B6                   JSR -$14A(a6)
071E64  2C 5F                         MOVEA.l (a7)+,a6
071E66  4E 75                         RTS

; ==== _FreeSignal  $071E68  (1 caller) —  ====
071E68  2F 0E                         MOVE.l a6,-(a7)
071E6A  20 2F 00 08                   MOVE.l $8(a7),d0
071E6E  2C 79 00 07 00 64             MOVEA.l $70064.l,a6
071E74  4E AE FE B0                   JSR -$150(a6)
071E78  2C 5F                         MOVEA.l (a7)+,a6
071E7A  4E 75                         RTS

; ==== _OpenLibrary  $071E7C  (1 caller) — exec OpenLibrary (graphics.library / intuition for the display). ====
071E7C  2F 0E                         MOVE.l a6,-(a7)
071E7E  22 6F 00 08                   MOVEA.l $8(a7),a1
071E82  20 2F 00 0C                   MOVE.l $C(a7),d0
071E86  2C 79 00 07 00 64             MOVEA.l $70064.l,a6
071E8C  4E AE FE 68                   JSR -$198(a6)
071E90  2C 5F                         MOVEA.l (a7)+,a6
071E92  4E 75                         RTS

; ==== _CloseLibrary  $071E94  (1 caller) —  ====
071E94  2F 0E                         MOVE.l a6,-(a7)
071E96  22 6F 00 08                   MOVEA.l $8(a7),a1
071E9A  2C 79 00 07 00 64             MOVEA.l $70064.l,a6
071EA0  4E AE FE 62                   JSR -$19E(a6)
071EA4  2C 5F                         MOVEA.l (a7)+,a6
071EA6  4E 75                         RTS

; ==== _OpenDevice  $071EA8  (1 caller) — exec OpenDevice — opens trackdisk.device for the mini-filesystem. ====
071EA8  2F 0E                         MOVE.l a6,-(a7)
071EAA  20 6F 00 08                   MOVEA.l $8(a7),a0
071EAE  4C EF                         .dc.w $4CEF
071EB0  .dc.b 02 01 00 0C 22 2F 00 14 2C 79 00 07 00 64 4E AE ; ...."/..,y...dN.
071EC0  .dc.b FE 44 2C 5F 4E 75 00 00                         ; .D,_Nu..

; ==== _CloseDevice  $071EC8  (1 caller) —  ====
071EC8  2F 0E                         MOVE.l a6,-(a7)
071ECA  22 6F 00 08                   MOVEA.l $8(a7),a1
071ECE  2C 79 00 07 00 64             MOVEA.l $70064.l,a6
071ED4  4E AE FE 3E                   JSR -$1C2(a6)
071ED8  2C 5F                         MOVEA.l (a7)+,a6
071EDA  4E 75                         RTS

; ==== _DoIO  $071EDC  (4 callers) — exec DoIO — synchronous device request (trackdisk reads). ====
071EDC  2F 0E                         MOVE.l a6,-(a7)
071EDE  22 6F 00 08                   MOVEA.l $8(a7),a1
071EE2  2C 79 00 07 00 64             MOVEA.l $70064.l,a6
071EE8  4E AE FE 38                   JSR -$1C8(a6)
071EEC  2C 5F                         MOVEA.l (a7)+,a6
071EEE  4E 75                         RTS

; ==== _LoadRGB4  $071EF0  (1 caller) — graphics LoadRGB4 — load the colour registers from _colors. ====
071EF0  2F 0E                         MOVE.l a6,-(a7)
071EF2  4C EF                         .dc.w $4CEF
071EF4  .dc.b 03 00 00 08 20 2F 00 10 2C 79 00 07 07 9C 4E AE ; .... /..,y....N.
071F04  .dc.b FF 40 2C 5F 4E 75 00 00                         ; .@,_Nu..

; ==== _InitRastPort  $071F0C  (1 caller) — graphics InitRastPort. ====
071F0C  2F 0E                         MOVE.l a6,-(a7)
071F0E  22 6F 00 08                   MOVEA.l $8(a7),a1
071F12  2C 79 00 07 07 9C             MOVEA.l $7079C.l,a6
071F18  4E AE FF 3A                   JSR -$C6(a6)
071F1C  2C 5F                         MOVEA.l (a7)+,a6
071F1E  4E 75                         RTS

; ==== _InitVPort  $071F20  (1 caller) — graphics InitVPort. ====
071F20  2F 0E                         MOVE.l a6,-(a7)
071F22  20 6F 00 08                   MOVEA.l $8(a7),a0
071F26  2C 79 00 07 07 9C             MOVEA.l $7079C.l,a6
071F2C  4E AE FF 34                   JSR -$CC(a6)
071F30  2C 5F                         MOVEA.l (a7)+,a6
071F32  4E 75                         RTS

; ==== _MrgCop  $071F34  (1 caller) — graphics MrgCop — merge the copper lists. ====
071F34  2F 0E                         MOVE.l a6,-(a7)
071F36  22 6F 00 08                   MOVEA.l $8(a7),a1
071F3A  2C 79 00 07 07 9C             MOVEA.l $7079C.l,a6
071F40  4E AE FF 2E                   JSR -$D2(a6)
071F44  2C 5F                         MOVEA.l (a7)+,a6
071F46  4E 75                         RTS

; ==== _MakeVPort  $071F48  (1 caller) — graphics MakeVPort — build the copper list for the viewport. ====
071F48  2F 0E                         MOVE.l a6,-(a7)
071F4A  4C EF                         .dc.w $4CEF
071F4C  .dc.b 03 00 00 08 2C 79 00 07 07 9C 4E AE FF 28 2C 5F ; ....,y....N..(,_
071F5C  .dc.b 4E 75 00 00                                     ; Nu..

; ==== _LoadView  $071F60  (2 callers) — graphics LoadView — install the View, putting the splash on screen. ====
071F60  2F 0E                         MOVE.l a6,-(a7)
071F62  22 6F 00 08                   MOVEA.l $8(a7),a1
071F66  2C 79 00 07 07 9C             MOVEA.l $7079C.l,a6
071F6C  4E AE FF 22                   JSR -$DE(a6)
071F70  2C 5F                         MOVEA.l (a7)+,a6
071F72  4E 75                         RTS

; ==== _WaitBlit  $071F74  (1 caller) —  ====
071F74  2F 0E                         MOVE.l a6,-(a7)
071F76  2C 79 00 07 07 9C             MOVEA.l $7079C.l,a6
071F7C  4E AE FF 1C                   JSR -$E4(a6)
071F80  2C 5F                         MOVEA.l (a7)+,a6
071F82  4E 75                         RTS

; ==== _WaitTOF  $071F84  (1 caller) — graphics WaitTOF — sync to vertical blank. ====
071F84  2F 0E                         MOVE.l a6,-(a7)
071F86  2C 79 00 07 07 9C             MOVEA.l $7079C.l,a6
071F8C  4E AE FE F2                   JSR -$10E(a6)
071F90  2C 5F                         MOVEA.l (a7)+,a6
071F92  4E 75                         RTS

; ==== _InitView  $071F94  (1 caller) — graphics InitView. ====
071F94  2F 0E                         MOVE.l a6,-(a7)
071F96  22 6F 00 08                   MOVEA.l $8(a7),a1
071F9A  2C 79 00 07 07 9C             MOVEA.l $7079C.l,a6
071FA0  4E AE FE 98                   JSR -$168(a6)
071FA4  2C 5F                         MOVEA.l (a7)+,a6
071FA6  4E 75                         RTS

; --- _InitBitMap  $071FA8 — graphics InitBitMap for the splash bitplanes. ---
071FA8  48 E7 20 02                   MOVEM.l d2/a6,-(a7)
071FAC  20 6F 00 0C                   MOVEA.l $C(a7),a0
071FB0  4C EF                         .dc.w $4CEF
071FB2  .dc.b 00 07 00 10 2C 79 00 07 07 9C 4E AE FE 7A 4C DF ; ....,y....N..zL.
071FC2  .dc.b 40 04 4E 75 00 00                               ; @.Nu..

; ==== _FreeVPortCopLists  $071FC8  (1 caller) — graphics: free the viewport copper lists (teardown). ====
071FC8  2F 0E                         MOVE.l a6,-(a7)
071FCA  20 6F 00 08                   MOVEA.l $8(a7),a0
071FCE  2C 79 00 07 07 9C             MOVEA.l $7079C.l,a6
071FD4  4E AE FD E4                   JSR -$21C(a6)
071FD8  2C 5F                         MOVEA.l (a7)+,a6
071FDA  4E 75                         RTS

; ==== _FreeCprList  $071FDC  (1 caller) — graphics: free a copper list (teardown). ====
071FDC  2F 0E                         MOVE.l a6,-(a7)
071FDE  20 6F 00 08                   MOVEA.l $8(a7),a0
071FE2  2C 79 00 07 07 9C             MOVEA.l $7079C.l,a6
071FE8  4E AE FD CC                   JSR -$234(a6)
071FEC  2C 5F                         MOVEA.l (a7)+,a6
071FEE  4E 75                         RTS

; ==== sub_071FF0 (1 caller) ====
071FF0  20 6F 00 04                   MOVEA.l $4(a7),a0
071FF4  22 6F 00 08                   MOVEA.l $8(a7),a1
071FF8  20 2F 00 0C                   MOVE.l $C(a7),d0
071FFC  6F 16                         BLE $072014
071FFE  B3 C8                         CMPA.l a0,a1
072000  65 0C                         BCS $07200E
072002  D1 C0                         ADDA.l d0,a0
072004  D3 C0                         ADDA.l d0,a1
072006  13 20                         MOVE.b -(a0),-(a1)
072008  53 80                         SUBQ.l #1,d0
07200A  66 FA                         BNE $072006
07200C  4E 75                         RTS
07200E  12 D8                         MOVE.b (a0)+,(a1)+
072010  53 80                         SUBQ.l #1,d0
072012  66 FA                         BNE $07200E
072014  4E 75                         RTS
072016  .dc.b 00 00                                           ; ..

; ==== sub_072018 (1 caller) ====
072018  48 E7 3C 00                   MOVEM.l d2-d5,-(a7)
07201C  2A 01                         MOVE.l d1,d5
07201E  67 32                         BEQ $072052
072020  6A 02                         BPL $072024
072022  44 81                         NEG.l d1
072024  28 00                         MOVE.l d0,d4
072026  67 28                         BEQ $072050
072028  6A 02                         BPL $07202C
07202A  44 80                         NEG.l d0
07202C  42 82                         CLR.l d2
07202E  76 1F                         MOVEQ #$1F,d3
072030  E3 80                         ASL.l #1,d0
072032  E3 92                         ROXL.l #1,d2
072034  B4 81                         CMP.l d1,d2
072036  65 04                         BCS $07203C
072038  94 81                         SUB.l d1,d2
07203A  52 80                         ADDQ.l #1,d0
07203C  51 CB FF F2                   DBRA d3,$072030
072040  22 02                         MOVE.l d2,d1
072042  B9 85                         EOR.l d4,d5
072044  6A 02                         BPL $072048
072046  44 80                         NEG.l d0
072048  B3 84                         EOR.l d1,d4
07204A  6A 08                         BPL $072054
07204C  44 81                         NEG.l d1
07204E  60 04                         BRA $072054
072050  42 81                         CLR.l d1
072052  42 80                         CLR.l d0
072054  4C DF                         .dc.w $4CDF
072056  .dc.b 00 3C 4E 75 00 00                               ; .<Nu..

; ==== sub_07205C (9 callers) ====
07205C  48 E7 78 00                   MOVEM.l d1-d4,-(a7)
072060  28 00                         MOVE.l d0,d4
072062  B3 84                         EOR.l d1,d4
072064  4A 80                         TST.l d0
072066  67 30                         BEQ $072098
072068  6A 02                         BPL $07206C
07206A  44 80                         NEG.l d0
07206C  24 00                         MOVE.l d0,d2
07206E  4A 81                         TST.l d1
072070  66 04                         BNE $072076
072072  42 80                         CLR.l d0
072074  60 22                         BRA $072098
072076  6A 02                         BPL $07207A
072078  44 81                         NEG.l d1
07207A  26 00                         MOVE.l d0,d3
07207C  C6 C1                         MULU.W d1,d3
07207E  48 42                         SWAP d2
072080  C4 C1                         MULU.W d1,d2
072082  48 42                         SWAP d2
072084  42 42                         CLR.w d2
072086  D6 82                         ADD.l d2,d3
072088  48 41                         SWAP d1
07208A  C0 C1                         MULU.W d1,d0
07208C  48 40                         SWAP d0
07208E  42 40                         CLR.w d0
072090  D0 83                         ADD.l d3,d0
072092  4A 84                         TST.l d4
072094  6A 02                         BPL $072098
072096  44 80                         NEG.l d0
072098  4C DF                         .dc.w $4CDF
07209A  .dc.b 00 1E 4E 75 00 00                               ; ..Nu..
