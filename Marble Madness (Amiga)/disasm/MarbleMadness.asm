
; --- cstart  $050000 — C runtime entry: AmigaDOS jumps here. Save regs, store ExecBase ($501A8) and the CLI arg length/ptr (d0/a0); FindTask self; on pr_CLI ($AC) split CLI-vs-Workbench, build an argv from the command line for CLI; then call main and exit. ---
050000  48 E7 1F FE                   MOVEM.l d3-d7/a0-a6,-(a7)
050004  23 CF 00 05 01 C0             MOVE.l a7,$501C0.l
05000A  23 C0 00 05 01 C8             MOVE.l d0,$501C8.l
050010  23 C8 00 05 01 CC             MOVE.l a0,$501CC.l
050016  42 B9 00 05 01 C4             CLR.l $501C4.l
05001C  2C 79 00 00 00 04             MOVEA.l $4.l,a6
050022  23 CE 00 05 01 A8             MOVE.l a6,$501A8.l
050028  93 C9                         SUBA.l a1,a1
05002A  4E AE FE DA                   JSR -$126(a6)
05002E  28 40                         MOVEA.l d0,a4
050030  4A AC 00 AC                   TST.l $AC(a4)
050034  67 00 00 AA                   BEQ $0500E0
050038  61 00 01 52                   BSR $05018C
05003C  20 6C 00 AC                   MOVEA.l $AC(a4),a0
050040  D1 C8                         ADDA.l a0,a0
050042  D1 C8                         ADDA.l a0,a0
050044  20 68 00 10                   MOVEA.l $10(a0),a0
050048  D1 C8                         ADDA.l a0,a0
05004A  D1 C8                         ADDA.l a0,a0
05004C  48 E7 20 30                   MOVEM.l d2/a2-a3,-(a7)
050050  45 F9 00 05 02 50             LEA $50250.l,a2
050056  47 F9 00 05 01 D0             LEA $501D0.l,a3
05005C  74 01                         MOVEQ #$1,d2
05005E  70 00                         MOVEQ #$0,d0
050060  10 18                         MOVE.b (a0)+,d0
050062  26 CA                         MOVE.l a2,(a3)+
050064  60 02                         BRA $050068
050066  14 D8                         MOVE.b (a0)+,(a2)+
050068  51 C8 FF FC                   DBRA d0,$050066
05006C  42 1A                         CLR.b (a2)+
05006E  20 39 00 05 01 C8             MOVE.l $501C8.l,d0
050074  20 79 00 05 01 CC             MOVEA.l $501CC.l,a0
05007A  12 18                         MOVE.b (a0)+,d1
05007C  53 80                         SUBQ.l #1,d0
05007E  6F 1E                         BLE $05009E
050080  0C 01 00 20                   CMPI.b #$20,d1
050084  6F F4                         BLE $05007A
050086  52 82                         ADDQ.l #1,d2
050088  26 CA                         MOVE.l a2,(a3)+
05008A  60 0A                         BRA $050096
05008C  12 18                         MOVE.b (a0)+,d1
05008E  53 80                         SUBQ.l #1,d0
050090  0C 01 00 20                   CMPI.b #$20,d1
050094  6F 04                         BLE $05009A
050096  14 C1                         MOVE.b d1,(a2)+
050098  60 F2                         BRA $05008C
05009A  42 1A                         CLR.b (a2)+
05009C  60 DC                         BRA $05007A
05009E  42 1A                         CLR.b (a2)+
0500A0  42 9B                         CLR.l (a3)+
0500A2  20 02                         MOVE.l d2,d0
0500A4  4C DF                         .dc.w $4CDF
0500A6  .dc.b 0C 04 48 79 00 05 01 D0 2F 00 4E B9 00 05 0D A4 ; ..Hy..../.N.....
0500B6  .dc.b 23 C0 00 05 01 B4 4E B9 00 05 0D B4 23 C0 00 05 ; #.....N.....#...
0500C6  .dc.b 01 B8 23 C0 00 05 01 BC 4E B9 00 05 03 8C 2E 79 ; ..#.....N......y
0500D6  .dc.b 00 05 01 C0 4C DF 7F F8 4E 75                   ; ....L...Nu
0500E0  61 00 00 AA                   BSR $05018C
0500E4  4E B9 00 05 0D A4             JSR $50DA4.l
0500EA  23 C0 00 05 01 B4             MOVE.l d0,$501B4.l
0500F0  4E B9 00 05 0D B4             JSR $50DB4.l
0500F6  23 C0 00 05 01 B8             MOVE.l d0,$501B8.l
0500FC  23 C0 00 05 01 BC             MOVE.l d0,$501BC.l
050102  61 00 00 76                   BSR $05017A
050106  23 C0 00 05 01 C4             MOVE.l d0,$501C4.l
05010C  42 A7                         CLR.l -(a7)
05010E  42 A7                         CLR.l -(a7)
050110  4E B9 00 05 03 8C             JSR $5038C.l
050116  50 8F                         ADDQ.l #8,a7
050118  60 04                         BRA $05011E

; ==== c_exit  $05011A  (1 caller) — exit(code): restore the saved stack ($501C0), reply/free the Workbench startup message if one was held, return the code in d0 to AmigaDOS. ====
05011A  20 2F 00 04                   MOVE.l $4(a7),d0
05011E  2E 79 00 05 01 C0             MOVEA.l $501C0.l,a7
050124  2F 00                         MOVE.l d0,-(a7)
050126  2C 79 00 00 00 04             MOVEA.l $4.l,a6
05012C  22 79 00 05 01 AC             MOVEA.l $501AC.l,a1
050132  4E AE FE 62                   JSR -$19E(a6)
050136  4A B9 00 05 01 C4             TST.l $501C4.l
05013C  67 14                         BEQ $050152
05013E  2C 79 00 00 00 04             MOVEA.l $4.l,a6
050144  4E AE FF 7C                   JSR -$84(a6)
050148  22 79 00 05 01 C4             MOVEA.l $501C4.l,a1
05014E  4E AE FE 86                   JSR -$17A(a6)
050152  20 1F                         MOVE.l (a7)+,d0
050154  2E 79 00 05 01 C0             MOVEA.l $501C0.l,a7
05015A  4C DF                         .dc.w $4CDF
05015C  .dc.b 7F F8 4E 75                                     ; ..Nu

; --- alert_no_dos  $050160 — dos.library failed to open: Alert($00038007) via exec -$6C, then retry/abort. ---
050160  48 E7 01 06                   MOVEM.l d7/a5-a6,-(a7)
050164  2E 3C 00 03 80 07             MOVE.l #$38007,d7
05016A  2C 78 00 04                   MOVEA.l $4.w,a6
05016E  4E AE FF 94                   JSR -$6C(a6)
050172  4C DF                         .dc.w $4CDF
050174  .dc.b 60 80 70 64 60 A4                               ; `.pd`.

; ==== reply_wb_msg  $05017A  (1 caller) — hand the Workbench startup message back through the process message port ($5C of the Process). ====
05017A  41 EC 00 5C                   LEA $5C(a4),a0
05017E  4E AE FE 80                   JSR -$180(a6)
050182  41 EC 00 5C                   LEA $5C(a4),a0
050186  4E AE FE 8C                   JSR -$174(a6)
05018A  4E 75                         RTS

; ==== open_dos  $05018C  (2 callers) — OpenLibrary("dos.library") -> $501AC (exec -$228); jump to alert_no_dos on failure. ====
05018C  43 F9 00 05 03 50             LEA $50350.l,a1
050192  42 80                         CLR.l d0
050194  4E AE FD D8                   JSR -$228(a6)
050198  23 C0 00 05 01 AC             MOVE.l d0,$501AC.l
05019E  67 C0                         BEQ $050160
0501A0  4E 75                         RTS
0501A2  .dc.b 00 00 00 01 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0501B2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0501C2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0501D2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0501E2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0501F2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050202  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050212  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050222  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050232  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050242  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050252  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050262  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050272  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050282  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050292  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0502A2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0502B2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0502C2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0502D2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0502E2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0502F2  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050302  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050312  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050322  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050332  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050342  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 64 6F ; ..............do
050352  .dc.b 73 2E 6C 69 62 72 61 72 79 00                   ; s.library.

; --- avail_mem  $05035C — AvailMem(d1=2) via exec -$D8 — query free memory before loading. ---
05035C  2F 0E                         MOVE.l a6,-(a7)
05035E  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050364  22 3C 00 00 00 02             MOVE.l #$2,d1
05036A  4E AE FF 28                   JSR -$D8(a6)
05036E  2C 5F                         MOVEA.l (a7)+,a6

; --- call_func  $050370 — call a C function pointer: a1=func, d0=arg1, a0=arg2; save regs and JSR (a1). ---
050370  22 6F 00 04                   MOVEA.l $4(a7),a1
050374  20 2F 00 08                   MOVE.l $8(a7),d0
050378  20 6F 00 0C                   MOVEA.l $C(a7),a0
05037C  48 E7 7F 3E                   MOVEM.l d1-d7/a2-a6,-(a7)
050380  4E 91                         JSR (a1)
050382  4C DF                         .dc.w $4CDF
050384  .dc.b 7C FE 4E 75 4E 71 00 00                         ; |.NuNq..

; ==== main  $05038C  (1 caller) — C main(argc, argv): thin wrapper — calls launcher_main and returns 0. ====
05038C  4E 56 00 00                   LINK a6,#$0
050390  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
050394  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
050398  61 08                         BSR $0503A2
05039A  50 8F                         ADDQ.l #8,a7
05039C  70 00                         MOVEQ #$0,d0
05039E  4E 5E                         UNLK a6
0503A0  4E 75                         RTS

; ==== launcher_main  $0503A2  (1 caller) — the launcher's real entry. FindTask self; branch on whether a Workbench startup message was passed ($C(a6)); CurrentDir to the program's own drawer; create the reply port; then put up the boot screen and load+decrypt the game (see the load sequence below). ====
0503A2  4E 56 FF 40                   LINK a6,#-$C0
0503A6  42 A7                         CLR.l -(a7)
0503A8  4E B9 00 05 0E 7C             JSR $50E7C.l
0503AE  58 8F                         ADDQ.l #4,a7
0503B0  72 00                         MOVEQ #$0,d1
0503B2  2D 40 FF B6                   MOVE.l d0,-$4A(a6)
0503B6  2D 41 FF 40                   MOVE.l d1,-$C0(a6)
0503BA  2D 41 FF 44                   MOVE.l d1,-$BC(a6)
0503BE  2D 41 FF 48                   MOVE.l d1,-$B8(a6)
0503C2  2D 41 FF 5C                   MOVE.l d1,-$A4(a6)
0503C6  4A AE 00 0C                   TST.l $C(a6)
0503CA  66 72                         BNE $05043E
0503CC  2D 79 00 05 01 C4 FF FC       MOVE.l $501C4.l,-$4(a6)
0503D4  20 6E FF FC                   MOVEA.l -$4(a6),a0
0503D8  2D 68 00 24 FF F4             MOVE.l $24(a0),-$C(a6)
0503DE  2D 68 00 1C 00 08             MOVE.l $1C(a0),$8(a6)
0503E4  48 6E FF 8C                   PEA -$74(a6)
0503E8  20 6E FF F4                   MOVEA.l -$C(a6),a0
0503EC  2F 28 00 04                   MOVE.l $4(a0),-(a7)
0503F0  61 00 03 72                   BSR $050764
0503F4  50 8F                         ADDQ.l #8,a7
0503F6  48 79 00 05 09 C0             PEA $509C0.l
0503FC  20 6E FF F4                   MOVEA.l -$C(a6),a0
050400  2F 28 00 04                   MOVE.l $4(a0),-(a7)
050404  61 00 03 F0                   BSR $0507F6
050408  50 8F                         ADDQ.l #8,a7
05040A  42 A7                         CLR.l -(a7)
05040C  48 79 00 05 09 B8             PEA $509B8.l
050412  4E B9 00 05 0F 20             JSR $50F20.l
050418  50 8F                         ADDQ.l #8,a7
05041A  20 79 00 05 01 C4             MOVEA.l $501C4.l,a0
050420  2D 68 00 0E FF 50             MOVE.l $E(a0),-$B0(a6)
050426  20 79 00 05 01 C4             MOVEA.l $501C4.l,a0
05042C  21 40 00 0E                   MOVE.l d0,$E(a0)
050430  20 6E FF F4                   MOVEA.l -$C(a6),a0
050434  2D 50 FF 44                   MOVE.l (a0),-$BC(a6)
050438  2D 40 FF 54                   MOVE.l d0,-$AC(a6)
05043C  60 22                         BRA $050460
05043E  48 6E FF 8C                   PEA -$74(a6)
050442  20 6E 00 0C                   MOVEA.l $C(a6),a0
050446  2F 10                         MOVE.l (a0),-(a7)
050448  61 00 03 1A                   BSR $050764
05044C  50 8F                         ADDQ.l #8,a7
05044E  48 79 00 05 09 C0             PEA $509C0.l
050454  20 6E 00 0C                   MOVEA.l $C(a6),a0
050458  2F 10                         MOVE.l (a0),-(a7)
05045A  61 00 03 9A                   BSR $0507F6
05045E  50 8F                         ADDQ.l #8,a7
050460  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
050466  70 24                         MOVEQ #$24,d0
050468  2F 00                         MOVE.l d0,-(a7)
05046A  4E B9 00 05 0E 4C             JSR $50E4C.l
050470  50 8F                         ADDQ.l #8,a7
050472  2F 00                         MOVE.l d0,-(a7)
050474  2F 2E FF 44                   MOVE.l -$BC(a6),-(a7)
050478  2D 40 FF 4C                   MOVE.l d0,-$B4(a6)
05047C  4E B9 00 05 0D F4             JSR $50DF4.l
050482  50 8F                         ADDQ.l #8,a7
050484  48 79 00 05 09 E8             PEA $509E8.l
05048A  2F 2E FF 4C                   MOVE.l -$B4(a6),-(a7)
05048E  61 00 04 A4                   BSR $050934
050492  50 8F                         ADDQ.l #8,a7
050494  70 24                         MOVEQ #$24,d0
050496  2F 00                         MOVE.l d0,-(a7)
050498  2F 2E FF 4C                   MOVE.l -$B4(a6),-(a7)
05049C  4E B9 00 05 0E 64             JSR $50E64.l
0504A2  50 8F                         ADDQ.l #8,a7
0504A4  2F 2E FF 44                   MOVE.l -$BC(a6),-(a7)
0504A8  4E B9 00 05 0E 10             JSR $50E10.l
0504AE  58 8F                         ADDQ.l #4,a7
0504B0  72 00                         MOVEQ #$0,d1
0504B2  32 39 00 DF F0 06             MOVE.w $DFF006.l,d1
0504B8  02 81 00 00 00 0F             ANDI.l #$F,d1
0504BE  2D 40 FF 40                   MOVE.l d0,-$C0(a6)
0504C2  2D 41 FF 5C                   MOVE.l d1,-$A4(a6)
0504C6  4A 81                         TST.l d1
0504C8  67 04                         BEQ $0504CE
0504CA  4A 81                         TST.l d1
0504CC  6A 06                         BPL $0504D4
0504CE  70 01                         MOVEQ #$1,d0
0504D0  2D 40 FF 5C                   MOVE.l d0,-$A4(a6)
0504D4  20 2E FF 5C                   MOVE.l -$A4(a6),d0
0504D8  E7 80                         ASL.l #3,d0
0504DA  72 02                         MOVEQ #$2,d1
0504DC  2F 01                         MOVE.l d1,-(a7)
0504DE  2F 00                         MOVE.l d0,-(a7)
0504E0  2D 40 FF 5C                   MOVE.l d0,-$A4(a6)

; --- ld_alloc_ctrl  $0504E4 — AllocMem the control block / key-array buffer that c/zzz's decode is driven from. ---
0504E4  4E B9 00 05 0E 4C             JSR $50E4C.l
0504EA  50 8F                         ADDQ.l #8,a7
0504EC  72 00                         MOVEQ #$0,d1
0504EE  2D 41 FF D2                   MOVE.l d1,-$2E(a6)
0504F2  72 01                         MOVEQ #$1,d1
0504F4  2D 41 FF D6                   MOVE.l d1,-$2A(a6)
0504F8  42 AE FF DA                   CLR.l -$26(a6)

; --- ld_name_zzz  $0504FC — point at the string "c/zzz" (the decrypter). ---
0504FC  48 79 00 05 09 B2             PEA $509B2.l
050502  2D 40 FF 58                   MOVE.l d0,-$A8(a6)

; --- ld_loadseg_zzz  $050506 — LoadSeg("c/zzz") -> the decrypter's seglist (-$46(a6)). c/zzz is a clean hunk, so AmigaDOS loads it normally. ---
050506  4E B9 00 05 0E 24             JSR $50E24.l
05050C  58 8F                         ADDQ.l #4,a7
05050E  48 79 00 05 09 AC             PEA $509AC.l
050514  48 6E FF D2                   PEA -$2E(a6)
050518  2F 00                         MOVE.l d0,-(a7)
05051A  2D 40 FF BA                   MOVE.l d0,-$46(a6)

; --- ld_run_zzz_xxx  $05051E — call_seglist: run c/zzz with the control block and the filename "c/xxx" — i.e. decrypt and load the second-stage loader c/xxx. Its returned seglist is kept in -$32(a6). ---
05051E  4E B9 00 05 0A 10             JSR $50A10.l
050524  4F EF 00 0C                   LEA $C(a7),a7
050528  3D 6E FF EA FF B4             MOVE.w -$16(a6),-$4C(a6)
05052E  72 FE                         MOVEQ #$FE,d1
050530  2F 01                         MOVE.l d1,-(a7)
050532  48 79 00 05 09 C0             PEA $509C0.l
050538  2D 40 FF CE                   MOVE.l d0,-$32(a6)
05053C  4E B9 00 05 0D C4             JSR $50DC4.l
050542  50 8F                         ADDQ.l #8,a7
050544  2D 40 FF 48                   MOVE.l d0,-$B8(a6)
050548  4A 80                         TST.l d0
05054A  67 0A                         BEQ $050556
05054C  2F 00                         MOVE.l d0,-(a7)
05054E  4E B9 00 05 0D E0             JSR $50DE0.l
050554  58 8F                         ADDQ.l #4,a7
050556  70 14                         MOVEQ #$14,d0
050558  2D 40 FF DA                   MOVE.l d0,-$26(a6)
05055C  2F 2E FF CE                   MOVE.l -$32(a6),-(a7)
050560  4E B9 00 05 0D 4A             JSR $50D4A.l
050566  58 8F                         ADDQ.l #4,a7
050568  48 C0                         EXT.L d0
05056A  32 2E FF B4                   MOVE.w -$4C(a6),d1
05056E  48 C1                         EXT.L d1
050570  B3 80                         EOR.l d1,d0
050572  02 80 00 00 FF FF             ANDI.l #$FFFF,d0
050578  42 A7                         CLR.l -(a7)
05057A  48 6E FF D2                   PEA -$2E(a6)
05057E  2F 2E FF CE                   MOVE.l -$32(a6),-(a7)
050582  2D 40 FF BE                   MOVE.l d0,-$42(a6)
050586  4E B9 00 05 0A 10             JSR $50A10.l
05058C  4F EF 00 0C                   LEA $C(a7),a7
050590  2F 2E FF CE                   MOVE.l -$32(a6),-(a7)
050594  4E B9 00 05 0E 38             JSR $50E38.l
05059A  58 8F                         ADDQ.l #4,a7

; --- ld_mutate_key  $05059C — XOR-mutate the key array in place before the next decrypt pass (the 16-bit constant computed just above). ---
05059C  42 AE FF C2                   CLR.l -$3E(a6)
0505A0  20 2E FF C2                   MOVE.l -$3E(a6),d0
0505A4  B0 AE FF DA                   CMP.l -$26(a6),d0
0505A8  6C 14                         BGE $0505BE
0505AA  E5 80                         ASL.l #2,d0
0505AC  20 6E FF DE                   MOVEA.l -$22(a6),a0
0505B0  D1 C0                         ADDA.l d0,a0
0505B2  20 2E FF BE                   MOVE.l -$42(a6),d0
0505B6  B1 90                         EOR.l d0,(a0)
0505B8  52 AE FF C2                   ADDQ.l #1,-$3E(a6)
0505BC  60 E2                         BRA $0505A0
0505BE  70 FE                         MOVEQ #$FE,d0
0505C0  2F 00                         MOVE.l d0,-(a7)
0505C2  48 79 00 05 09 E8             PEA $509E8.l
0505C8  4E B9 00 05 0D C4             JSR $50DC4.l
0505CE  50 8F                         ADDQ.l #8,a7
0505D0  2D 40 FF 48                   MOVE.l d0,-$B8(a6)
0505D4  4A 80                         TST.l d0
0505D6  67 0A                         BEQ $0505E2
0505D8  2F 00                         MOVE.l d0,-(a7)
0505DA  4E B9 00 05 0D E0             JSR $50DE0.l
0505E0  58 8F                         ADDQ.l #4,a7

; --- ld_name_bootscr  $0505E2 — point at the string "c/bootscr". ---
0505E2  48 79 00 05 09 88             PEA $50988.l

; --- ld_loadseg_bootscr  $0505E8 — LoadSeg("c/bootscr") -> the boot-screen overlay's seglist (-$3A(a6)). bootscr is also a clean, unencrypted hunk, so it loads through plain LoadSeg — this is the load the question asks about. ---
0505E8  4E B9 00 05 0E 24             JSR $50E24.l
0505EE  58 8F                         ADDQ.l #4,a7
0505F0  2D 40 FF C6                   MOVE.l d0,-$3A(a6)

; --- ld_bootscr_check  $0505F4 — if LoadSeg returned 0 the boot screen is missing: exit(1). ---
0505F4  4A 80                         TST.l d0
0505F6  66 0C                         BNE $050604
0505F8  70 01                         MOVEQ #$1,d0
0505FA  2F 00                         MOVE.l d0,-(a7)
0505FC  4E B9 00 05 01 1A             JSR $5011A.l
050602  58 8F                         ADDQ.l #4,a7
050604  70 FE                         MOVEQ #$FE,d0
050606  2F 00                         MOVE.l d0,-(a7)
050608  48 79 00 05 09 92             PEA $50992.l

; --- ld_lock_splash  $05060E — Lock("c/splash", ACCESS_READ=$FE) to test whether the splash image file is present. ---
05060E  4E B9 00 05 0D C4             JSR $50DC4.l
050614  50 8F                         ADDQ.l #8,a7
050616  2D 40 FF EC                   MOVE.l d0,-$14(a6)
05061A  4A 80                         TST.l d0
05061C  67 24                         BEQ $050642
05061E  2F 00                         MOVE.l d0,-(a7)
050620  4E B9 00 05 0D E0             JSR $50DE0.l
050626  58 8F                         ADDQ.l #4,a7

; --- ld_splash_present  $050628 — splash file exists: UnLock it, then... ---
050628  48 79 00 05 09 92             PEA $50992.l
05062E  70 01                         MOVEQ #$1,d0
050630  2F 00                         MOVE.l d0,-(a7)
050632  2F 2E FF C6                   MOVE.l -$3A(a6),-(a7)

; --- ld_run_bootscr_splash  $050636 — call_seglist: run bootscr(seglist, 1, "c/splash") — hand the overlay the splash filename so it loads the IFF and paints it on screen. ---
050636  4E B9 00 05 0A 10             JSR $50A10.l
05063C  4F EF 00 0C                   LEA $C(a7),a7
050640  60 18                         BRA $05065A

; --- ld_splash_missing  $050642 — splash file absent: fall back to the alternate image path. ---
050642  48 79 00 05 09 9C             PEA $5099C.l
050648  70 01                         MOVEQ #$1,d0
05064A  2F 00                         MOVE.l d0,-(a7)
05064C  2F 2E FF C6                   MOVE.l -$3A(a6),-(a7)

; --- ld_run_bootscr_alt  $050650 — call_seglist: run bootscr(seglist, 1, "lo-res/paintcan") — the fallback image. ---
050650  4E B9 00 05 0A 10             JSR $50A10.l
050656  4F EF 00 0C                   LEA $C(a7),a7
05065A  48 6E FF 8C                   PEA -$74(a6)
05065E  48 6E FF D2                   PEA -$2E(a6)
050662  2F 2E FF BA                   MOVE.l -$46(a6),-(a7)

; --- ld_run_loader  $050666 — call_seglist into the c/zzz/loader stage again to continue bringing the main game into memory (control block + work buffer). ---
050666  4E B9 00 05 0A 10             JSR $50A10.l
05066C  4F EF 00 0C                   LEA $C(a7),a7
050670  22 2E FF DA                   MOVE.l -$26(a6),d1
050674  E5 81                         ASL.l #2,d1
050676  2F 01                         MOVE.l d1,-(a7)
050678  2F 2E FF DE                   MOVE.l -$22(a6),-(a7)
05067C  2D 40 FF F0                   MOVE.l d0,-$10(a6)
050680  4E B9 00 05 0E 64             JSR $50E64.l
050686  50 8F                         ADDQ.l #8,a7

; --- ld_free_key  $050688 — free the key-array buffer once decryption is done. ---
050688  70 00                         MOVEQ #$0,d0
05068A  2F 00                         MOVE.l d0,-(a7)
05068C  2F 00                         MOVE.l d0,-(a7)
05068E  2F 2E FF C6                   MOVE.l -$3A(a6),-(a7)

; --- ld_run_bootscr_off  $050692 — call_seglist: run bootscr(seglist, 0, 0) — tear the boot screen back down. ---
050692  4E B9 00 05 0A 10             JSR $50A10.l
050698  4F EF 00 0C                   LEA $C(a7),a7
05069C  2F 2E FF C6                   MOVE.l -$3A(a6),-(a7)

; --- ld_unload_bootscr  $0506A0 — UnLoadSeg the boot-screen overlay now that it has done its job. ---
0506A0  4E B9 00 05 0E 38             JSR $50E38.l
0506A6  58 8F                         ADDQ.l #4,a7
0506A8  2F 2E FF 5C                   MOVE.l -$A4(a6),-(a7)
0506AC  2F 2E FF 58                   MOVE.l -$A8(a6),-(a7)
0506B0  4E B9 00 05 0E 64             JSR $50E64.l
0506B6  50 8F                         ADDQ.l #8,a7
0506B8  2F 2E FF BA                   MOVE.l -$46(a6),-(a7)

; --- ld_unload_zzz  $0506BC — UnLoadSeg the decrypter c/zzz. ---
0506BC  4E B9 00 05 0E 38             JSR $50E38.l
0506C2  58 8F                         ADDQ.l #4,a7
0506C4  4A AE 00 0C                   TST.l $C(a6)
0506C8  66 56                         BNE $050720
0506CA  20 6E FF B6                   MOVEA.l -$4A(a6),a0
0506CE  D0 FC 00 5C                   ADDA.w #$5C,a0
0506D2  2F 39 00 05 01 C4             MOVE.l $501C4.l,-(a7)
0506D8  2F 08                         MOVE.l a0,-(a7)
0506DA  4E B9 00 05 0E E0             JSR $50EE0.l
0506E0  50 8F                         ADDQ.l #8,a7
0506E2  70 00                         MOVEQ #$0,d0
0506E4  2F 00                         MOVE.l d0,-(a7)
0506E6  2F 00                         MOVE.l d0,-(a7)
0506E8  2F 2E FF F0                   MOVE.l -$10(a6),-(a7)
0506EC  4E B9 00 05 0A 10             JSR $50A10.l
0506F2  4F EF 00 0C                   LEA $C(a7),a7
0506F6  2F 2E FF 54                   MOVE.l -$AC(a6),-(a7)
0506FA  4E B9 00 05 0E F8             JSR $50EF8.l
050700  58 8F                         ADDQ.l #4,a7
050702  4A 80                         TST.l d0
050704  67 F0                         BEQ $0506F6
050706  2F 2E FF 54                   MOVE.l -$AC(a6),-(a7)
05070A  4E B9 00 05 0F C8             JSR $50FC8.l
050710  58 8F                         ADDQ.l #4,a7
050712  20 79 00 05 01 C4             MOVEA.l $501C4.l,a0
050718  21 6E FF 50 00 0E             MOVE.l -$B0(a6),$E(a0)
05071E  60 34                         BRA $050754
050720  20 2E 00 08                   MOVE.l $8(a6),d0
050724  53 80                         SUBQ.l #1,d0
050726  20 6E 00 0C                   MOVEA.l $C(a6),a0
05072A  58 88                         ADDQ.l #4,a0
05072C  48 6E FF 64                   PEA -$9C(a6)
050730  2F 08                         MOVE.l a0,-(a7)
050732  2F 00                         MOVE.l d0,-(a7)
050734  61 00 01 84                   BSR $0508BA
050738  4F EF 00 0C                   LEA $C(a7),a7
05073C  48 6E FF 64                   PEA -$9C(a6)
050740  2F 00                         MOVE.l d0,-(a7)
050742  2F 2E FF F0                   MOVE.l -$10(a6),-(a7)
050746  2D 40 FF CA                   MOVE.l d0,-$36(a6)
05074A  4E B9 00 05 0A 10             JSR $50A10.l
050750  4F EF 00 0C                   LEA $C(a7),a7
050754  2F 2E FF F0                   MOVE.l -$10(a6),-(a7)
050758  4E B9 00 05 0E 38             JSR $50E38.l
05075E  58 8F                         ADDQ.l #4,a7
050760  4E 5E                         UNLK a6
050762  4E 75                         RTS

; ==== build_path  $050764  (2 callers) — assemble a "drawer/file" path string into a local buffer. ====
050764  4E 56 FF F8                   LINK a6,#-$8
050768  42 AE FF FC                   CLR.l -$4(a6)
05076C  20 6E 00 0C                   MOVEA.l $C(a6),a0
050770  10 BC 00 63                   MOVE.b #$63,(a0)
050774  11 7C 00 2F 00 01             MOVE.b #$2F,$1(a0)
05077A  70 02                         MOVEQ #$2,d0
05077C  2D 40 FF F8                   MOVE.l d0,-$8(a6)
050780  20 2E FF F8                   MOVE.l -$8(a6),d0
050784  20 6E 00 0C                   MOVEA.l $C(a6),a0
050788  D1 C0                         ADDA.l d0,a0
05078A  22 2E FF FC                   MOVE.l -$4(a6),d1
05078E  22 6E 00 08                   MOVEA.l $8(a6),a1
050792  D3 C1                         ADDA.l d1,a1
050794  10 91                         MOVE.b (a1),(a0)
050796  52 AE FF FC                   ADDQ.l #1,-$4(a6)
05079A  52 AE FF F8                   ADDQ.l #1,-$8(a6)
05079E  20 6E 00 08                   MOVEA.l $8(a6),a0
0507A2  D1 EE FF FC                   ADDA.l -$4(a6),a0
0507A6  4A 10                         TST.b (a0)
0507A8  67 08                         BEQ $0507B2
0507AA  10 10                         MOVE.b (a0),d0
0507AC  0C 00 00 20                   CMPI.b #$20,d0
0507B0  66 CE                         BNE $050780
0507B2  20 6E 00 0C                   MOVEA.l $C(a6),a0
0507B6  D1 EE FF F8                   ADDA.l -$8(a6),a0
0507BA  20 2E FF F8                   MOVE.l -$8(a6),d0
0507BE  52 80                         ADDQ.l #1,d0
0507C0  10 BC 00 2E                   MOVE.b #$2E,(a0)
0507C4  20 6E 00 0C                   MOVEA.l $C(a6),a0
0507C8  D1 C0                         ADDA.l d0,a0
0507CA  52 80                         ADDQ.l #1,d0
0507CC  10 BC 00 64                   MOVE.b #$64,(a0)
0507D0  20 6E 00 0C                   MOVEA.l $C(a6),a0
0507D4  D1 C0                         ADDA.l d0,a0
0507D6  52 80                         ADDQ.l #1,d0
0507D8  10 BC 00 61                   MOVE.b #$61,(a0)
0507DC  20 6E 00 0C                   MOVEA.l $C(a6),a0
0507E0  D1 C0                         ADDA.l d0,a0
0507E2  52 80                         ADDQ.l #1,d0
0507E4  10 BC 00 74                   MOVE.b #$74,(a0)
0507E8  20 6E 00 0C                   MOVEA.l $C(a6),a0
0507EC  D1 C0                         ADDA.l d0,a0
0507EE  52 80                         ADDQ.l #1,d0
0507F0  42 10                         CLR.b (a0)
0507F2  4E 5E                         UNLK a6
0507F4  4E 75                         RTS

; ==== build_path2  $0507F6  (2 callers) — assemble a second path string (e.g. the data file under the program drawer). ====
0507F6  4E 56 FF F8                   LINK a6,#-$8
0507FA  70 00                         MOVEQ #$0,d0
0507FC  2D 40 FF F8                   MOVE.l d0,-$8(a6)
050800  2D 40 FF FC                   MOVE.l d0,-$4(a6)
050804  20 2E FF F8                   MOVE.l -$8(a6),d0
050808  20 6E 00 0C                   MOVEA.l $C(a6),a0
05080C  D1 C0                         ADDA.l d0,a0
05080E  22 2E FF FC                   MOVE.l -$4(a6),d1
050812  22 6E 00 08                   MOVEA.l $8(a6),a1
050816  D3 C1                         ADDA.l d1,a1
050818  10 91                         MOVE.b (a1),(a0)
05081A  52 AE FF FC                   ADDQ.l #1,-$4(a6)
05081E  52 AE FF F8                   ADDQ.l #1,-$8(a6)
050822  20 6E 00 08                   MOVEA.l $8(a6),a0
050826  D1 EE FF FC                   ADDA.l -$4(a6),a0
05082A  10 10                         MOVE.b (a0),d0
05082C  4A 00                         TST.b d0
05082E  66 D4                         BNE $050804
050830  20 6E 00 0C                   MOVEA.l $C(a6),a0
050834  D1 EE FF F8                   ADDA.l -$8(a6),a0
050838  20 2E FF F8                   MOVE.l -$8(a6),d0
05083C  52 80                         ADDQ.l #1,d0
05083E  10 BC 00 3A                   MOVE.b #$3A,(a0)
050842  20 6E 00 0C                   MOVEA.l $C(a6),a0
050846  D1 C0                         ADDA.l d0,a0
050848  52 80                         ADDQ.l #1,d0
05084A  10 BC 00 63                   MOVE.b #$63,(a0)
05084E  20 6E 00 0C                   MOVEA.l $C(a6),a0
050852  D1 C0                         ADDA.l d0,a0
050854  52 80                         ADDQ.l #1,d0
050856  10 BC 00 2F                   MOVE.b #$2F,(a0)
05085A  20 6E 00 0C                   MOVEA.l $C(a6),a0
05085E  D1 C0                         ADDA.l d0,a0
050860  52 80                         ADDQ.l #1,d0
050862  10 BC 00 73                   MOVE.b #$73,(a0)
050866  20 6E 00 0C                   MOVEA.l $C(a6),a0
05086A  D1 C0                         ADDA.l d0,a0
05086C  52 80                         ADDQ.l #1,d0
05086E  72 69                         MOVEQ #$69,d1
050870  10 81                         MOVE.b d1,(a0)
050872  20 6E 00 0C                   MOVEA.l $C(a6),a0
050876  D1 C0                         ADDA.l d0,a0
050878  52 80                         ADDQ.l #1,d0
05087A  10 BC 00 67                   MOVE.b #$67,(a0)
05087E  20 6E 00 0C                   MOVEA.l $C(a6),a0
050882  D1 C0                         ADDA.l d0,a0
050884  52 80                         ADDQ.l #1,d0
050886  10 BC 00 66                   MOVE.b #$66,(a0)
05088A  20 6E 00 0C                   MOVEA.l $C(a6),a0
05088E  D1 C0                         ADDA.l d0,a0
050890  52 80                         ADDQ.l #1,d0
050892  10 81                         MOVE.b d1,(a0)
050894  20 6E 00 0C                   MOVEA.l $C(a6),a0
050898  D1 C0                         ADDA.l d0,a0
05089A  52 80                         ADDQ.l #1,d0
05089C  10 BC 00 6C                   MOVE.b #$6C,(a0)
0508A0  20 6E 00 0C                   MOVEA.l $C(a6),a0
0508A4  D1 C0                         ADDA.l d0,a0
0508A6  52 80                         ADDQ.l #1,d0
0508A8  10 BC 00 65                   MOVE.b #$65,(a0)
0508AC  20 6E 00 0C                   MOVEA.l $C(a6),a0
0508B0  D1 C0                         ADDA.l d0,a0
0508B2  52 80                         ADDQ.l #1,d0
0508B4  42 10                         CLR.b (a0)
0508B6  4E 5E                         UNLK a6
0508B8  4E 75                         RTS

; ==== build_argv  $0508BA  (1 caller) — build a NULL-terminated argv-style pointer array from a packed string. ====
0508BA  4E 56 FF F0                   LINK a6,#-$10
0508BE  70 00                         MOVEQ #$0,d0
0508C0  2D 40 FF FC                   MOVE.l d0,-$4(a6)
0508C4  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0508C8  20 2E FF FC                   MOVE.l -$4(a6),d0
0508CC  B0 AE 00 08                   CMP.l $8(a6),d0
0508D0  6C 4E                         BGE $050920
0508D2  E5 80                         ASL.l #2,d0
0508D4  20 6E 00 0C                   MOVEA.l $C(a6),a0
0508D8  D1 C0                         ADDA.l d0,a0
0508DA  2D 50 FF F0                   MOVE.l (a0),-$10(a6)
0508DE  42 AE FF F4                   CLR.l -$C(a6)
0508E2  20 6E 00 10                   MOVEA.l $10(a6),a0
0508E6  D1 EE FF F8                   ADDA.l -$8(a6),a0
0508EA  52 AE FF F8                   ADDQ.l #1,-$8(a6)
0508EE  22 6E FF F0                   MOVEA.l -$10(a6),a1
0508F2  D3 EE FF F4                   ADDA.l -$C(a6),a1
0508F6  52 AE FF F4                   ADDQ.l #1,-$C(a6)
0508FA  10 91                         MOVE.b (a1),(a0)
0508FC  20 6E FF F0                   MOVEA.l -$10(a6),a0
050900  D1 EE FF F4                   ADDA.l -$C(a6),a0
050904  10 10                         MOVE.b (a0),d0
050906  4A 00                         TST.b d0
050908  66 D8                         BNE $0508E2
05090A  20 6E 00 10                   MOVEA.l $10(a6),a0
05090E  D1 EE FF F8                   ADDA.l -$8(a6),a0
050912  52 AE FF F8                   ADDQ.l #1,-$8(a6)
050916  10 BC 00 20                   MOVE.b #$20,(a0)
05091A  52 AE FF FC                   ADDQ.l #1,-$4(a6)
05091E  60 A8                         BRA $0508C8
050920  20 6E 00 10                   MOVEA.l $10(a6),a0
050924  D1 EE FF F8                   ADDA.l -$8(a6),a0
050928  20 2E FF F8                   MOVE.l -$8(a6),d0
05092C  52 80                         ADDQ.l #1,d0
05092E  42 10                         CLR.b (a0)
050930  4E 5E                         UNLK a6
050932  4E 75                         RTS

; ==== copy_toolname  $050934  (1 caller) — copy the Workbench tool name / CLI command name into a local buffer. ====
050934  4E 56 FF F4                   LINK a6,#-$C
050938  20 6E 00 08                   MOVEA.l $8(a6),a0
05093C  20 28 00 1C                   MOVE.l $1C(a0),d0
050940  E5 80                         ASL.l #2,d0
050942  20 40                         MOVEA.l d0,a0
050944  12 10                         MOVE.b (a0),d1
050946  48 81                         EXT.W d1
050948  48 C1                         EXT.L d1
05094A  42 AE FF F4                   CLR.l -$C(a6)
05094E  2D 40 FF FC                   MOVE.l d0,-$4(a6)
050952  2D 41 FF F8                   MOVE.l d1,-$8(a6)
050956  20 2E FF F4                   MOVE.l -$C(a6),d0
05095A  B0 AE FF F8                   CMP.l -$8(a6),d0
05095E  6C 16                         BGE $050976
050960  20 6E 00 0C                   MOVEA.l $C(a6),a0
050964  D1 C0                         ADDA.l d0,a0
050966  52 80                         ADDQ.l #1,d0
050968  22 6E FF FC                   MOVEA.l -$4(a6),a1
05096C  D3 C0                         ADDA.l d0,a1
05096E  10 91                         MOVE.b (a1),(a0)
050970  52 AE FF F4                   ADDQ.l #1,-$C(a6)
050974  60 E0                         BRA $050956
050976  20 6E 00 0C                   MOVEA.l $C(a6),a0
05097A  D1 EE FF F8                   ADDA.l -$8(a6),a0
05097E  42 10                         CLR.b (a0)
050980  70 00                         MOVEQ #$0,d0
050982  4E 5E                         UNLK a6
050984  4E 75                         RTS
050986  .dc.b 00 00 63 2F 62 6F 6F 74 73 63 72 00 63 2F 73 70 ; ..c/bootscr.c/sp
050996  .dc.b 6C 61 73 68 00 00 6C 6F 2D 72 65 73 2F 70 61 69 ; lash..lo-res/pai
0509A6  .dc.b 6E 74 63 61 6E 00 63 2F 78 78 78 00 63 2F 7A 7A ; ntcan.c/xxx.c/zz
0509B6  .dc.b 7A 00 52 50 4C 59 00 00 00 00 00 00 00 00 00 00 ; z.RPLY..........
0509C6  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0509D6  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0509E6  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0509F6  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
050A06  .dc.b 00 00 00 00 00 00 00 00 00 00                   ; ..........

; ==== call_seglist  $050A10  (8 callers) — run a LoadSeg'd module: take its BPTR seglist, convert to an address ((bptr+1)<<2) and JSR into the first hunk, passing d0 (arg1) and a0 (arg2). This is how every loaded overlay — bootscr, c/zzz, the decrypted c/xxx — is entered. ====
050A10  22 6F 00 04                   MOVEA.l $4(a7),a1
050A14  20 2F 00 08                   MOVE.l $8(a7),d0
050A18  20 6F 00 0C                   MOVEA.l $C(a7),a0
050A1C  48 E7 7F 3E                   MOVEM.l d1-d7/a2-a6,-(a7)
050A20  22 09                         MOVE.l a1,d1
050A22  52 81                         ADDQ.l #1,d1
050A24  E5 89                         LSL.l #2,d1
050A26  22 41                         MOVEA.l d1,a1
050A28  4E 91                         JSR (a1)
050A2A  4C DF                         .dc.w $4CDF
050A2C  .dc.b 7C FE 4E 75 4E 71 00 00                         ; |.NuNq..

; ==== zz_seed_table  $050A34  (2 callers) — [embedded, unused] build the 55-entry table from seed $57319753 by the ×31 hash (mirror of c/zzz sub_BEC). ====
050A34  4E 56 FF FC                   LINK a6,#-$4
050A38  20 6E 00 08                   MOVEA.l $8(a6),a0
050A3C  20 BC 57 31 97 53             MOVE.l #$57319753,(a0)
050A42  70 01                         MOVEQ #$1,d0
050A44  2D 40 FF FC                   MOVE.l d0,-$4(a6)
050A48  20 2E FF FC                   MOVE.l -$4(a6),d0
050A4C  0C 80 00 00 00 37             CMPI.l #$37,d0
050A52  6C 22                         BGE $050A76
050A54  E5 80                         ASL.l #2,d0
050A56  20 6E 00 08                   MOVEA.l $8(a6),a0
050A5A  D1 C0                         ADDA.l d0,a0
050A5C  59 80                         SUBQ.l #4,d0
050A5E  22 6E 00 08                   MOVEA.l $8(a6),a1
050A62  D3 C0                         ADDA.l d0,a1
050A64  20 11                         MOVE.l (a1),d0
050A66  EB 80                         ASL.l #5,d0
050A68  90 91                         SUB.l (a1),d0
050A6A  D0 AE FF FC                   ADD.l -$4(a6),d0
050A6E  20 80                         MOVE.l d0,(a0)
050A70  52 AE FF FC                   ADDQ.l #1,-$4(a6)
050A74  60 D2                         BRA $050A48
050A76  20 6E 00 08                   MOVEA.l $8(a6),a0
050A7A  23 C8 00 05 0C E8             MOVE.l a0,$50CE8.l
050A80  D0 FC 00 6C                   ADDA.w #$6C,a0
050A84  23 C8 00 05 0C EC             MOVE.l a0,$50CEC.l
050A8A  20 6E 00 08                   MOVEA.l $8(a6),a0
050A8E  D0 FC 00 DC                   ADDA.w #$DC,a0
050A92  23 C8 00 05 0C F0             MOVE.l a0,$50CF0.l
050A98  4E 5E                         UNLK a6
050A9A  4E 75                         RTS

; --- zz_key_init  $050A9C — [embedded, unused] AllocMem the table, seed it, XOR in the key array, run the protection (mirror of c/zzz sub_D06). ---
050A9C  4E 56 FF F4                   LINK a6,#-$C
050AA0  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
050AA6  2F 3C 00 00 00 DC             MOVE.l #$DC,-(a7)
050AAC  4E B9 00 05 0E 4C             JSR $50E4C.l
050AB2  50 8F                         ADDQ.l #8,a7
050AB4  2F 00                         MOVE.l d0,-(a7)
050AB6  2D 40 FF F8                   MOVE.l d0,-$8(a6)
050ABA  61 00 FF 78                   BSR $050A34
050ABE  58 8F                         ADDQ.l #4,a7
050AC0  0C AE 00 00 00 37 00 0C       CMPI.l #$37,$C(a6)
050AC8  6F 04                         BLE $050ACE
050ACA  70 37                         MOVEQ #$37,d0
050ACC  60 04                         BRA $050AD2
050ACE  20 2E 00 0C                   MOVE.l $C(a6),d0
050AD2  42 AE FF FC                   CLR.l -$4(a6)
050AD6  2D 40 00 0C                   MOVE.l d0,$C(a6)
050ADA  20 2E FF FC                   MOVE.l -$4(a6),d0
050ADE  B0 AE 00 0C                   CMP.l $C(a6),d0
050AE2  6C 18                         BGE $050AFC
050AE4  E5 80                         ASL.l #2,d0
050AE6  20 6E FF F8                   MOVEA.l -$8(a6),a0
050AEA  D1 C0                         ADDA.l d0,a0
050AEC  22 6E 00 08                   MOVEA.l $8(a6),a1
050AF0  D3 C0                         ADDA.l d0,a1
050AF2  20 11                         MOVE.l (a1),d0
050AF4  B1 90                         EOR.l d0,(a0)
050AF6  52 AE FF FC                   ADDQ.l #1,-$4(a6)
050AFA  60 DE                         BRA $050ADA
050AFC  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
050B00  61 00 00 F0                   BSR $050BF2
050B04  58 8F                         ADDQ.l #4,a7
050B06  42 AE FF FC                   CLR.l -$4(a6)
050B0A  20 2E FF FC                   MOVE.l -$4(a6),d0
050B0E  B0 AE 00 14                   CMP.l $14(a6),d0
050B12  6C 24                         BGE $050B38
050B14  E5 80                         ASL.l #2,d0
050B16  20 6E 00 10                   MOVEA.l $10(a6),a0
050B1A  D1 C0                         ADDA.l d0,a0
050B1C  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
050B20  2F 48 00 04                   MOVE.l a0,$4(a7)
050B24  4E B9 00 05 0C F4             JSR $50CF4.l
050B2A  58 8F                         ADDQ.l #4,a7
050B2C  20 6F 00 00                   MOVEA.l $0(a7),a0
050B30  B1 90                         EOR.l d0,(a0)
050B32  52 AE FF FC                   ADDQ.l #1,-$4(a6)
050B36  60 D2                         BRA $050B0A
050B38  2F 3C 00 00 00 DC             MOVE.l #$DC,-(a7)
050B3E  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
050B42  4E B9 00 05 0E 64             JSR $50E64.l
050B48  50 8F                         ADDQ.l #8,a7
050B4A  4E 5E                         UNLK a6
050B4C  4E 75                         RTS

; --- zz_free_key  $050B4E — [embedded, unused] FreeMem the key table. ---
050B4E  4E 56 FF F8                   LINK a6,#-$8
050B52  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
050B58  2F 3C 00 00 00 DC             MOVE.l #$DC,-(a7)
050B5E  4E B9 00 05 0E 4C             JSR $50E4C.l
050B64  50 8F                         ADDQ.l #8,a7
050B66  2F 00                         MOVE.l d0,-(a7)
050B68  2D 40 FF F8                   MOVE.l d0,-$8(a6)
050B6C  61 00 FE C6                   BSR $050A34
050B70  58 8F                         ADDQ.l #4,a7
050B72  0C AE 00 00 00 37 00 0C       CMPI.l #$37,$C(a6)
050B7A  6F 04                         BLE $050B80
050B7C  70 37                         MOVEQ #$37,d0
050B7E  60 04                         BRA $050B84
050B80  20 2E 00 0C                   MOVE.l $C(a6),d0
050B84  42 AE FF FC                   CLR.l -$4(a6)
050B88  2D 40 00 0C                   MOVE.l d0,$C(a6)
050B8C  20 2E FF FC                   MOVE.l -$4(a6),d0
050B90  B0 AE 00 0C                   CMP.l $C(a6),d0
050B94  6C 18                         BGE $050BAE
050B96  E5 80                         ASL.l #2,d0
050B98  20 6E FF F8                   MOVEA.l -$8(a6),a0
050B9C  D1 C0                         ADDA.l d0,a0
050B9E  22 6E 00 08                   MOVEA.l $8(a6),a1
050BA2  D3 C0                         ADDA.l d0,a1
050BA4  20 11                         MOVE.l (a1),d0
050BA6  B1 90                         EOR.l d0,(a0)
050BA8  52 AE FF FC                   ADDQ.l #1,-$4(a6)
050BAC  60 DE                         BRA $050B8C
050BAE  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
050BB2  61 00 00 3E                   BSR $050BF2
050BB6  58 8F                         ADDQ.l #4,a7
050BB8  20 2E FF F8                   MOVE.l -$8(a6),d0
050BBC  4E 5E                         UNLK a6
050BBE  4E 75                         RTS

; --- zz_freemem  $050BC0 — [embedded, unused] FreeMem wrapper. ---
050BC0  4E 56 00 00                   LINK a6,#$0
050BC4  2F 3C 00 00 00 DC             MOVE.l #$DC,-(a7)
050BCA  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
050BCE  4E B9 00 05 0E 64             JSR $50E64.l
050BD4  50 8F                         ADDQ.l #8,a7
050BD6  4E 5E                         UNLK a6
050BD8  4E 75                         RTS

; ==== zz_byte_hi  $050BDA  (5 callers) — [embedded, unused] return (arg>>16)&0xFF (mirror of c/zzz sub_D92). ====
050BDA  4E 56 00 00                   LINK a6,#$0
050BDE  70 10                         MOVEQ #$10,d0
050BE0  22 2E 00 08                   MOVE.l $8(a6),d1
050BE4  E0 A1                         ASR.l d0,d1
050BE6  02 81 00 00 00 FF             ANDI.l #$FF,d1
050BEC  20 01                         MOVE.l d1,d0
050BEE  4E 5E                         UNLK a6
050BF0  4E 75                         RTS

; ==== zz_protection  $050BF2  (2 callers) — [embedded, unused] THE COPY PROTECTION: XOR the host's CPU exception/TRAP vectors ($8-$38, $80-$BC) and the running task's tc_ExceptCode/tc_TrapCode into table entries 10-16 / 30-31 / 32-47 (mirror of c/zzz sub_DAA). Binds the decode to live machine state. ====
050BF2  4E 56 FF EC                   LINK a6,#-$14
050BF6  91 C8                         SUBA.l a0,a0
050BF8  2F 08                         MOVE.l a0,-(a7)
050BFA  2D 48 FF FC                   MOVE.l a0,-$4(a6)
050BFE  4E B9 00 05 0E 7C             JSR $50E7C.l
050C04  58 8F                         ADDQ.l #4,a7
050C06  72 0A                         MOVEQ #$A,d1
050C08  2D 41 FF F8                   MOVE.l d1,-$8(a6)
050C0C  2D 40 FF F4                   MOVE.l d0,-$C(a6)
050C10  20 2E FF F8                   MOVE.l -$8(a6),d0
050C14  0C 80 00 00 00 11             CMPI.l #$11,d0
050C1A  6C 2A                         BGE $050C46
050C1C  E5 80                         ASL.l #2,d0
050C1E  20 6E 00 08                   MOVEA.l $8(a6),a0
050C22  D1 C0                         ADDA.l d0,a0
050C24  04 80 00 00 00 20             SUBI.l #$20,d0
050C2A  22 6E FF FC                   MOVEA.l -$4(a6),a1
050C2E  D3 C0                         ADDA.l d0,a1
050C30  2F 11                         MOVE.l (a1),-(a7)
050C32  2F 48 00 08                   MOVE.l a0,$8(a7)
050C36  61 A2                         BSR $050BDA
050C38  58 8F                         ADDQ.l #4,a7
050C3A  20 6F 00 04                   MOVEA.l $4(a7),a0
050C3E  B1 90                         EOR.l d0,(a0)
050C40  52 AE FF F8                   ADDQ.l #1,-$8(a6)
050C44  60 CA                         BRA $050C10
050C46  70 0A                         MOVEQ #$A,d0
050C48  2D 40 FF F8                   MOVE.l d0,-$8(a6)
050C4C  20 2E FF F8                   MOVE.l -$8(a6),d0
050C50  0C 80 00 00 00 0F             CMPI.l #$F,d0
050C56  6C 26                         BGE $050C7E
050C58  E5 80                         ASL.l #2,d0
050C5A  20 6E 00 08                   MOVEA.l $8(a6),a0
050C5E  D1 C0                         ADDA.l d0,a0
050C60  22 6E FF FC                   MOVEA.l -$4(a6),a1
050C64  D3 C0                         ADDA.l d0,a1
050C66  2F 11                         MOVE.l (a1),-(a7)
050C68  2F 48 00 08                   MOVE.l a0,$8(a7)
050C6C  61 00 FF 6C                   BSR $050BDA
050C70  58 8F                         ADDQ.l #4,a7
050C72  20 6F 00 04                   MOVEA.l $4(a7),a0
050C76  B1 90                         EOR.l d0,(a0)
050C78  52 AE FF F8                   ADDQ.l #1,-$8(a6)
050C7C  60 CE                         BRA $050C4C
050C7E  70 20                         MOVEQ #$20,d0
050C80  2D 40 FF F8                   MOVE.l d0,-$8(a6)
050C84  20 2E FF F8                   MOVE.l -$8(a6),d0
050C88  0C 80 00 00 00 30             CMPI.l #$30,d0
050C8E  6C 26                         BGE $050CB6
050C90  E5 80                         ASL.l #2,d0
050C92  20 6E 00 08                   MOVEA.l $8(a6),a0
050C96  D1 C0                         ADDA.l d0,a0
050C98  22 6E FF FC                   MOVEA.l -$4(a6),a1
050C9C  D3 C0                         ADDA.l d0,a1
050C9E  2F 11                         MOVE.l (a1),-(a7)
050CA0  2F 48 00 08                   MOVE.l a0,$8(a7)
050CA4  61 00 FF 34                   BSR $050BDA
050CA8  58 8F                         ADDQ.l #4,a7
050CAA  20 6F 00 04                   MOVEA.l $4(a7),a0
050CAE  B1 90                         EOR.l d0,(a0)
050CB0  52 AE FF F8                   ADDQ.l #1,-$8(a6)
050CB4  60 CE                         BRA $050C84
050CB6  20 6E FF F4                   MOVEA.l -$C(a6),a0
050CBA  2F 28 00 2A                   MOVE.l $2A(a0),-(a7)
050CBE  61 00 FF 1A                   BSR $050BDA
050CC2  58 8F                         ADDQ.l #4,a7
050CC4  20 6E 00 08                   MOVEA.l $8(a6),a0
050CC8  B1 A8 00 78                   EOR.l d0,$78(a0)
050CCC  20 6E FF F4                   MOVEA.l -$C(a6),a0
050CD0  2F 28 00 32                   MOVE.l $32(a0),-(a7)
050CD4  61 00 FF 04                   BSR $050BDA
050CD8  58 8F                         ADDQ.l #4,a7
050CDA  E8 80                         ASR.l #4,d0
050CDC  20 6E 00 08                   MOVEA.l $8(a6),a0
050CE0  B1 A8 00 7C                   EOR.l d0,$7C(a0)
050CE4  4E 5E                         UNLK a6
050CE6  4E 75                         RTS
050CE8  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00             ; ............

; ==== zz_keystream  $050CF4  (1 caller) — [embedded, unused] the additive lagged-Fibonacci keystream generator over the table (mirror of c/zzz sub_$EAC): table[p]+=table[q]; p+=1; q+=2; mod 55. ====
050CF4  20 79 00 05 0C E8             MOVEA.l $50CE8.l,a0
050CFA  22 79 00 05 0C EC             MOVEA.l $50CEC.l,a1
050D00  22 39 00 05 0C F0             MOVE.l $50CF0.l,d1
050D06  20 11                         MOVE.l (a1),d0
050D08  D1 98                         ADD.l d0,(a0)+
050D0A  B1 C1                         CMPA.l d1,a0
050D0C  66 04                         BNE $050D12
050D0E  90 FC 00 DC                   SUBA.w #$DC,a0
050D12  50 89                         ADDQ.l #8,a1
050D14  B3 C1                         CMPA.l d1,a1
050D16  65 04                         BCS $050D1C
050D18  92 FC 00 DC                   SUBA.w #$DC,a1
050D1C  23 C8 00 05 0C E8             MOVE.l a0,$50CE8.l
050D22  23 C9 00 05 0C EC             MOVE.l a1,$50CEC.l
050D28  20 10                         MOVE.l (a0),d0
050D2A  4E 75                         RTS

; ==== zz_decrypt_word  $050D2C  (1 caller) — [embedded, unused] read one longword and XOR it with the keystream. ====
050D2C  4E 56 00 00                   LINK a6,#$0
050D30  20 6E 00 08                   MOVEA.l $8(a6),a0
050D34  20 28 FF FC                   MOVE.l -$4(a0),d0
050D38  4E 5E                         UNLK a6
050D3A  4E 75                         RTS

; ==== sub_050D3C (2 callers) ====
050D3C  4E 56 00 00                   LINK a6,#$0
050D40  20 2E 00 08                   MOVE.l $8(a6),d0
050D44  E5 80                         ASL.l #2,d0
050D46  4E 5E                         UNLK a6
050D48  4E 75                         RTS

; ==== sub_050D4A (1 caller) ====
050D4A  4E 56 FF F2                   LINK a6,#-$E
050D4E  48 E7 03 04                   MOVEM.l d6-d7/a5,-(a7)
050D52  7C 00                         MOVEQ #$0,d6
050D54  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
050D58  61 E2                         BSR $050D3C
050D5A  58 8F                         ADDQ.l #4,a7
050D5C  2D 40 FF F2                   MOVE.l d0,-$E(a6)
050D60  2F 2E FF F2                   MOVE.l -$E(a6),-(a7)
050D64  61 C6                         BSR $050D2C
050D66  58 8F                         ADDQ.l #4,a7
050D68  E2 80                         ASR.l #1,d0
050D6A  59 80                         SUBQ.l #4,d0
050D6C  2E 00                         MOVE.l d0,d7
050D6E  67 16                         BEQ $050D86
050D70  20 6E FF F2                   MOVEA.l -$E(a6),a0
050D74  58 88                         ADDQ.l #4,a0
050D76  2A 48                         MOVEA.l a0,a5
050D78  30 15                         MOVE.w (a5),d0
050D7A  54 8D                         ADDQ.l #2,a5
050D7C  B1 46                         EOR.w d0,d6
050D7E  53 87                         SUBQ.l #1,d7
050D80  20 07                         MOVE.l d7,d0
050D82  4A 80                         TST.l d0
050D84  66 F2                         BNE $050D78
050D86  20 6E FF F2                   MOVEA.l -$E(a6),a0
050D8A  2F 10                         MOVE.l (a0),-(a7)
050D8C  61 AE                         BSR $050D3C
050D8E  58 8F                         ADDQ.l #4,a7
050D90  2D 40 FF F2                   MOVE.l d0,-$E(a6)
050D94  4A 80                         TST.l d0
050D96  66 C8                         BNE $050D60
050D98  20 06                         MOVE.l d6,d0
050D9A  4C DF                         .dc.w $4CDF
050D9C  .dc.b 20 C0 4E 5E 4E 75 00 00                         ;  .N^Nu..

; ==== _Input  $050DA4  (1 caller) — dos.library Input()  (-$36) ====
050DA4  2F 0E                         MOVE.l a6,-(a7)
050DA6  2C 79 00 05 01 AC             MOVEA.l $501AC.l,a6
050DAC  4E AE FF CA                   JSR -$36(a6)
050DB0  2C 5F                         MOVEA.l (a7)+,a6
050DB2  4E 75                         RTS

; ==== _Output  $050DB4  (1 caller) — dos.library Output() (-$3C) ====
050DB4  2F 0E                         MOVE.l a6,-(a7)
050DB6  2C 79 00 05 01 AC             MOVEA.l $501AC.l,a6
050DBC  4E AE FF C4                   JSR -$3C(a6)
050DC0  2C 5F                         MOVEA.l (a7)+,a6
050DC2  4E 75                         RTS

; ==== _Lock  $050DC4  (3 callers) — dos.library Lock(name, mode) (-$54) ====
050DC4  48 E7 20 02                   MOVEM.l d2/a6,-(a7)
050DC8  4C EF                         .dc.w $4CEF
050DCA  .dc.b 00 06 00 0C 2C 79 00 05 01 AC 4E AE FF AC 4C DF ; ....,y....N...L.
050DDA  .dc.b 40 04 4E 75 00 00                               ; @.Nu..

; ==== _UnLock  $050DE0  (3 callers) — dos.library UnLock(lock) (-$5A) ====
050DE0  2F 0E                         MOVE.l a6,-(a7)
050DE2  22 2F 00 08                   MOVE.l $8(a7),d1
050DE6  2C 79 00 05 01 AC             MOVEA.l $501AC.l,a6
050DEC  4E AE FF A6                   JSR -$5A(a6)
050DF0  2C 5F                         MOVEA.l (a7)+,a6
050DF2  4E 75                         RTS

; ==== _Info  $050DF4  (1 caller) — dos.library Info(lock, infoblock) (-$72) ====
050DF4  48 E7 20 02                   MOVEM.l d2/a6,-(a7)
050DF8  4C EF                         .dc.w $4CEF
050DFA  .dc.b 00 06 00 0C 2C 79 00 05 01 AC 4E AE FF 8E 4C DF ; ....,y....N...L.
050E0A  .dc.b 40 04 4E 75 00 00                               ; @.Nu..

; ==== _CurrentDir  $050E10  (1 caller) — dos.library CurrentDir(lock) (-$7E) ====
050E10  2F 0E                         MOVE.l a6,-(a7)
050E12  22 2F 00 08                   MOVE.l $8(a7),d1
050E16  2C 79 00 05 01 AC             MOVEA.l $501AC.l,a6
050E1C  4E AE FF 82                   JSR -$7E(a6)
050E20  2C 5F                         MOVEA.l (a7)+,a6
050E22  4E 75                         RTS

; ==== _LoadSeg  $050E24  (2 callers) — dos.library LoadSeg(name) -> BPTR seglist (-$96) ====
050E24  2F 0E                         MOVE.l a6,-(a7)
050E26  22 2F 00 08                   MOVE.l $8(a7),d1
050E2A  2C 79 00 05 01 AC             MOVEA.l $501AC.l,a6
050E30  4E AE FF 6A                   JSR -$96(a6)
050E34  2C 5F                         MOVEA.l (a7)+,a6
050E36  4E 75                         RTS

; ==== _UnLoadSeg  $050E38  (4 callers) — dos.library UnLoadSeg(seglist) (-$9C) ====
050E38  2F 0E                         MOVE.l a6,-(a7)
050E3A  22 2F 00 08                   MOVE.l $8(a7),d1
050E3E  2C 79 00 05 01 AC             MOVEA.l $501AC.l,a6
050E44  4E AE FF 64                   JSR -$9C(a6)
050E48  2C 5F                         MOVEA.l (a7)+,a6
050E4A  4E 75                         RTS

; ==== _AllocMem  $050E4C  (5 callers) — exec.library AllocMem(size, flags) (-$C6) ====
050E4C  2F 0E                         MOVE.l a6,-(a7)
050E4E  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050E54  4C EF                         .dc.w $4CEF
050E56  .dc.b 00 03 00 08 4E AE FF 3A 2C 5F 4E 75 00 00       ; ....N..:,_Nu..

; ==== _FreeMem  $050E64  (6 callers) — exec.library FreeMem(addr, size) (-$D2) ====
050E64  2F 0E                         MOVE.l a6,-(a7)
050E66  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050E6C  22 6F 00 08                   MOVEA.l $8(a7),a1
050E70  20 2F 00 0C                   MOVE.l $C(a7),d0
050E74  4E AE FF 2E                   JSR -$D2(a6)
050E78  2C 5F                         MOVEA.l (a7)+,a6
050E7A  4E 75                         RTS

; ==== _FindTask  $050E7C  (3 callers) — exec.library FindTask(name|0) (-$126) ====
050E7C  2F 0E                         MOVE.l a6,-(a7)
050E7E  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050E84  22 6F 00 08                   MOVEA.l $8(a7),a1
050E88  4E AE FE DA                   JSR -$126(a6)
050E8C  2C 5F                         MOVEA.l (a7)+,a6
050E8E  4E 75                         RTS

; ==== _AllocSignal  $050E90  (1 caller) — exec.library AllocSignal(num) (-$14A) ====
050E90  2F 0E                         MOVE.l a6,-(a7)
050E92  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050E98  20 2F 00 08                   MOVE.l $8(a7),d0
050E9C  4E AE FE B6                   JSR -$14A(a6)
050EA0  2C 5F                         MOVEA.l (a7)+,a6
050EA2  4E 75                         RTS

; ==== _FreeSignal  $050EA4  (2 callers) — exec.library FreeSignal(num) (-$150) ====
050EA4  2F 0E                         MOVE.l a6,-(a7)
050EA6  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050EAC  20 2F 00 08                   MOVE.l $8(a7),d0
050EB0  4E AE FE B0                   JSR -$150(a6)
050EB4  2C 5F                         MOVEA.l (a7)+,a6
050EB6  4E 75                         RTS

; ==== _AddPort  $050EB8  (1 caller) — exec.library AddPort(port) (-$162) ====
050EB8  2F 0E                         MOVE.l a6,-(a7)
050EBA  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050EC0  22 6F 00 08                   MOVEA.l $8(a7),a1
050EC4  4E AE FE 9E                   JSR -$162(a6)
050EC8  2C 5F                         MOVEA.l (a7)+,a6
050ECA  4E 75                         RTS

; ==== _RemPort  $050ECC  (1 caller) — exec.library RemPort(port) (-$168) ====
050ECC  2F 0E                         MOVE.l a6,-(a7)
050ECE  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050ED4  22 6F 00 08                   MOVEA.l $8(a7),a1
050ED8  4E AE FE 98                   JSR -$168(a6)
050EDC  2C 5F                         MOVEA.l (a7)+,a6
050EDE  4E 75                         RTS

; ==== _PutMsg  $050EE0  (1 caller) — exec.library PutMsg(port, msg) (-$16E) ====
050EE0  2F 0E                         MOVE.l a6,-(a7)
050EE2  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050EE8  4C EF                         .dc.w $4CEF
050EEA  .dc.b 03 00 00 08 4E AE FE 92 2C 5F 4E 75 00 00       ; ....N...,_Nu..

; ==== _GetMsg  $050EF8  (1 caller) — exec.library GetMsg(port) (-$174) ====
050EF8  2F 0E                         MOVE.l a6,-(a7)
050EFA  2C 79 00 05 01 A8             MOVEA.l $501A8.l,a6
050F00  20 6F 00 08                   MOVEA.l $8(a7),a0
050F04  4E AE FE 8C                   JSR -$174(a6)
050F08  2C 5F                         MOVEA.l (a7)+,a6
050F0A  4E 75                         RTS

; ==== _NewList  $050F0C  (1 caller) — exec.library NewList(list) (-$192) ====
050F0C  20 6F 00 04                   MOVEA.l $4(a7),a0
050F10  20 88                         MOVE.l a0,(a0)
050F12  58 90                         ADDQ.l #4,(a0)
050F14  42 A8 00 04                   CLR.l $4(a0)
050F18  21 48 00 08                   MOVE.l a0,$8(a0)
050F1C  4E 75                         RTS
050F1E  .dc.b 00 00                                           ; ..

; ==== _CreatePort  $050F20  (1 caller) — create a public message port: AllocSignal, AllocMem the MsgPort, fill it in and AddPort. ====
050F20  48 E7 3F 20                   MOVEM.l d2-d7/a2,-(a7)
050F24  28 2F 00 20                   MOVE.l $20(a7),d4
050F28  16 2F 00 27                   MOVE.b $27(a7),d3
050F2C  2F 3C FF FF FF FF             MOVE.l #$FFFFFFFF,-(a7)
050F32  4E B9 00 05 0E 90             JSR $50E90.l
050F38  24 00                         MOVE.l d0,d2
050F3A  1C 02                         MOVE.b d2,d6
050F3C  1A 06                         MOVE.b d6,d5
050F3E  74 00                         MOVEQ #$0,d2
050F40  14 06                         MOVE.b d6,d2
050F42  0C 82 FF FF FF FF             CMPI.l #$FFFFFFFF,d2
050F48  58 8F                         ADDQ.l #4,a7
050F4A  66 06                         BNE $050F52
050F4C  70 00                         MOVEQ #$0,d0
050F4E  60 00 00 72                   BRA $050FC2
050F52  2F 3C 00 01 00 01             MOVE.l #$10001,-(a7)
050F58  48 78 00 22                   PEA $22.w
050F5C  4E B9 00 05 0E 4C             JSR $50E4C.l
050F62  24 40                         MOVEA.l d0,a2
050F64  CF 8A                         EXG d7,a2
050F66  4A 87                         TST.l d7
050F68  CF 8A                         EXG d7,a2
050F6A  50 8F                         ADDQ.l #8,a7
050F6C  66 12                         BNE $050F80
050F6E  74 00                         MOVEQ #$0,d2
050F70  14 05                         MOVE.b d5,d2
050F72  2F 02                         MOVE.l d2,-(a7)
050F74  4E B9 00 05 0E A4             JSR $50EA4.l
050F7A  70 00                         MOVEQ #$0,d0
050F7C  58 8F                         ADDQ.l #4,a7
050F7E  60 42                         BRA $050FC2
050F80  25 44 00 0A                   MOVE.l d4,$A(a2)
050F84  15 43 00 09                   MOVE.b d3,$9(a2)
050F88  15 7C 00 04 00 08             MOVE.b #$4,$8(a2)
050F8E  42 2A 00 0E                   CLR.b $E(a2)
050F92  15 45 00 0F                   MOVE.b d5,$F(a2)
050F96  42 A7                         CLR.l -(a7)
050F98  4E B9 00 05 0E 7C             JSR $50E7C.l
050F9E  25 40 00 10                   MOVE.l d0,$10(a2)
050FA2  4A 84                         TST.l d4
050FA4  58 8F                         ADDQ.l #4,a7
050FA6  67 0C                         BEQ $050FB4
050FA8  2F 0A                         MOVE.l a2,-(a7)
050FAA  4E B9 00 05 0E B8             JSR $50EB8.l
050FB0  58 8F                         ADDQ.l #4,a7
050FB2  60 0C                         BRA $050FC0
050FB4  48 6A 00 14                   PEA $14(a2)
050FB8  4E B9 00 05 0F 0C             JSR $50F0C.l
050FBE  58 8F                         ADDQ.l #4,a7
050FC0  20 0A                         MOVE.l a2,d0
050FC2  4C DF                         .dc.w $4CDF
050FC4  .dc.b 04 FC 4E 75                                     ; ..Nu

; ==== _DeletePort  $050FC8  (1 caller) — tear a message port down: RemPort, FreeSignal, FreeMem. ====
050FC8  48 E7 20 20                   MOVEM.l d2/a2,-(a7)
050FCC  24 6F 00 0C                   MOVEA.l $C(a7),a2
050FD0  4A AA 00 0A                   TST.l $A(a2)
050FD4  67 0A                         BEQ $050FE0
050FD6  2F 0A                         MOVE.l a2,-(a7)
050FD8  4E B9 00 05 0E CC             JSR $50ECC.l
050FDE  58 8F                         ADDQ.l #4,a7
050FE0  15 7C 00 FF 00 08             MOVE.b #$FF,$8(a2)
050FE6  74 FF                         MOVEQ #$FF,d2
050FE8  25 42 00 14                   MOVE.l d2,$14(a2)
050FEC  74 00                         MOVEQ #$0,d2
050FEE  14 2A 00 0F                   MOVE.b $F(a2),d2
050FF2  2F 02                         MOVE.l d2,-(a7)
050FF4  4E B9 00 05 0E A4             JSR $50EA4.l
050FFA  48 78 00 22                   PEA $22.w
050FFE  2F 0A                         MOVE.l a2,-(a7)
051000  4E B9 00 05 0E 64             JSR $50E64.l
051006  4F EF 00 0C                   LEA $C(a7),a7
05100A  4C DF                         .dc.w $4CDF
05100C  .dc.b 04 04 4E 75                                     ; ..Nu
