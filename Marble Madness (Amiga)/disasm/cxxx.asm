
; --- cstart  $060000 — C runtime entry: save d0/a0 (the CLI/control-block args) to globals, cache ExecBase ($4.w -> $60064), MOVEM the regs, save SP; OpenLibrary dos.library; call main(arg0,arg1); fall through to exit. ---
060000  23 C0 00 06 00 A0             MOVE.l d0,$600A0.l
060006  23 C8 00 06 00 A4             MOVE.l a0,$600A4.l
06000C  23 F8 00 04 00 06 00 64       MOVE.l $4.w,$60064.l
060014  48 E7 7E FE                   MOVEM.l d1-d6/a0-a6,-(a7)
060018  23 CF 00 06 00 98             MOVE.l a7,$60098.l
06001E  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
060024  61 00 00 20                   BSR $060046
060028  2F 39 00 06 00 A4             MOVE.l $600A4.l,-(a7)
06002E  2F 39 00 06 00 A0             MOVE.l $600A0.l,-(a7)
060034  4E B9 00 06 00 B8             JSR $600B8.l

; ==== exit  $06003A  (1 caller) — restore the saved SP, MOVEM the regs back, RTS to AmigaDOS. ====
06003A  2E 79 00 06 00 98             MOVEA.l $60098.l,a7
060040  4C DF                         .dc.w $4CDF
060042  .dc.b 7F 7E 4E 75                                     ; .~Nu

; ==== open_dos  $060046  (1 caller) — OldOpenLibrary("dos.library") (exec -$228); store the base at $600A8. ====
060046  42 80                         CLR.l d0
060048  43 F9 00 06 00 AC             LEA $600AC.l,a1
06004E  4E AE FD D8                   JSR -$228(a6)
060052  23 C0 00 06 00 A8             MOVE.l d0,$600A8.l
060058  4E 75                         RTS
06005A  .dc.b 70 01 60 DC 00 00 00 00 00 00 00 00 00 00 00 00 ; p.`.............
06006A  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
06007A  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
06008A  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
06009A  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
0600AA  .dc.b 00 00                                           ; ..

; --- s_dos_library  $0600AC — the string "dos.library". (data) ---
0600AC  .dc.b 64 6F 73 2E 6C 69 62 72 61 72 79 00             ; dos.library.

; ==== main  $0600B8  (1 caller) — main(ctrl): AllocMem a work buffer of ctrl->8 * 4 bytes (the load area), record it at ctrl->C, then call run_loader(ctrl); clear ctrl->10 on return. ====
0600B8  4E 56 00 00                   LINK a6,#$0
0600BC  20 6E 00 08                   MOVEA.l $8(a6),a0
0600C0  20 28 00 08                   MOVE.l $8(a0),d0
0600C4  E5 80                         ASL.l #2,d0
0600C6  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
0600CC  2F 00                         MOVE.l d0,-(a7)
0600CE  4E B9 00 06 10 04             JSR $61004.l
0600D4  50 8F                         ADDQ.l #8,a7
0600D6  20 6E 00 08                   MOVEA.l $8(a6),a0
0600DA  21 40 00 0C                   MOVE.l d0,$C(a0)
0600DE  2F 08                         MOVE.l a0,-(a7)
0600E0  4E B9 00 06 01 08             JSR $60108.l
0600E6  58 8F                         ADDQ.l #4,a7
0600E8  20 6E 00 08                   MOVEA.l $8(a6),a0
0600EC  42 A8 00 10                   CLR.l $10(a0)
0600F0  60 10                         BRA $060102
0600F2  .dc.b 08 68 0F 00 23 95 2F 08 4E B9 00 06 01 08 58 8F ; .h..#./.N.....X.
060102  4E 5E                         UNLK a6
060104  4E 75                         RTS
060106  .dc.b 00 00                                           ; ..

; ==== run_loader  $060108  (1 caller) — the loader's real entry: pull the source/dest descriptors out of the control block and call load_session; the spine that walks the request and reads each track. ====
060108  4E 56 FF E0                   LINK a6,#-$20
06010C  42 AE FF EC                   CLR.l -$14(a6)
060110  20 6E 00 08                   MOVEA.l $8(a6),a0
060114  2F 28 00 04                   MOVE.l $4(a0),-(a7)
060118  2F 10                         MOVE.l (a0),-(a7)
06011A  61 00 02 86                   BSR $0603A2
06011E  50 8F                         ADDQ.l #8,a7
060120  4A 80                         TST.l d0
060122  67 06                         BEQ $06012A
060124  70 01                         MOVEQ #$1,d0
060126  4E 5E                         UNLK a6
060128  4E 75                         RTS
06012A  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
060130  2F 3C 00 00 3E 80             MOVE.l #$3E80,-(a7)
060136  4E B9 00 06 10 04             JSR $61004.l
06013C  50 8F                         ADDQ.l #8,a7
06013E  2D 40 FF E8                   MOVE.l d0,-$18(a6)
060142  4A 80                         TST.l d0
060144  66 0C                         BNE $060152
060146  70 01                         MOVEQ #$1,d0
060148  2F 00                         MOVE.l d0,-(a7)
06014A  4E B9 00 06 00 3A             JSR $6003A.l
060150  58 8F                         ADDQ.l #4,a7
060152  2F 2E FF E8                   MOVE.l -$18(a6),-(a7)
060156  20 6E 00 08                   MOVEA.l $8(a6),a0
06015A  2F 10                         MOVE.l (a0),-(a7)
06015C  48 79 00 06 0D F4             PEA $60DF4.l
060162  61 00 03 0A                   BSR $06046E
060166  4F EF 00 0C                   LEA $C(a7),a7
06016A  48 79 00 06 0D F4             PEA $60DF4.l
060170  2D 40 FF F0                   MOVE.l d0,-$10(a6)
060174  61 00 07 3A                   BSR $0608B0
060178  58 8F                         ADDQ.l #4,a7
06017A  48 79 00 06 0D F4             PEA $60DF4.l
060180  61 00 03 DC                   BSR $06055E
060184  58 8F                         ADDQ.l #4,a7
060186  42 39 00 06 0D F6             CLR.b $60DF6.l
06018C  20 6E 00 08                   MOVEA.l $8(a6),a0
060190  2F 28 00 04                   MOVE.l $4(a0),-(a7)
060194  48 79 00 06 0D F4             PEA $60DF4.l
06019A  61 00 04 58                   BSR $0605F4
06019E  50 8F                         ADDQ.l #8,a7
0601A0  48 79 00 06 0D F4             PEA $60DF4.l
0601A6  61 00 06 22                   BSR $0607CA
0601AA  58 8F                         ADDQ.l #4,a7
0601AC  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0601B0  4A 80                         TST.l d0
0601B2  67 16                         BEQ $0601CA
0601B4  20 2E FF EC                   MOVE.l -$14(a6),d0
0601B8  52 80                         ADDQ.l #1,d0
0601BA  2D 40 FF EC                   MOVE.l d0,-$14(a6)
0601BE  0C 80 00 00 00 05             CMPI.l #$5,d0
0601C4  6C 00 01 26                   BGE $0602EC
0601C8  60 C2                         BRA $06018C
0601CA  48 79 00 06 0D F4             PEA $60DF4.l
0601D0  61 00 07 9E                   BSR $060970
0601D4  58 8F                         ADDQ.l #4,a7
0601D6  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0601DA  4A 80                         TST.l d0
0601DC  67 16                         BEQ $0601F4
0601DE  20 2E FF EC                   MOVE.l -$14(a6),d0
0601E2  52 80                         ADDQ.l #1,d0
0601E4  2D 40 FF EC                   MOVE.l d0,-$14(a6)
0601E8  0C 80 00 00 00 05             CMPI.l #$5,d0
0601EE  6C 00 00 FC                   BGE $0602EC
0601F2  60 98                         BRA $06018C
0601F4  13 FC 00 01 00 06 0D F6       MOVE.b #$1,$60DF6.l
0601FC  10 39 00 06 0D F8             MOVE.b $60DF8.l,d0
060202  0C 00 00 0B                   CMPI.b #$B,d0
060206  67 84                         BEQ $06018C
060208  48 79 00 06 0D F4             PEA $60DF4.l
06020E  61 00 01 10                   BSR $060320
060212  58 8F                         ADDQ.l #4,a7
060214  2D 40 FF F4                   MOVE.l d0,-$C(a6)
060218  0C 80 00 00 05 DC             CMPI.l #$5DC,d0
06021E  4E 71                         NOP
060220  2D 7C 00 00 05 DB FF F4       MOVE.l #$5DB,-$C(a6)
060228  42 AE FF FC                   CLR.l -$4(a6)
06022C  20 2E FF FC                   MOVE.l -$4(a6),d0
060230  20 6E 00 08                   MOVEA.l $8(a6),a0
060234  B0 A8 00 08                   CMP.l $8(a0),d0
060238  6C 34                         BGE $06026E
06023A  E5 80                         ASL.l #2,d0
06023C  20 68 00 0C                   MOVEA.l $C(a0),a0
060240  D1 C0                         ADDA.l d0,a0
060242  20 2E FF FC                   MOVE.l -$4(a6),d0
060246  52 80                         ADDQ.l #1,d0
060248  22 3C 00 00 01 2C             MOVE.l #$12C,d1
06024E  4E B9 00 06 12 48             JSR $61248.l
060254  2F 40 00 04                   MOVE.l d0,$4(a7)
060258  20 2E FF F4                   MOVE.l -$C(a6),d0
06025C  22 2F 00 04                   MOVE.l $4(a7),d1
060260  4E B9 00 06 12 04             JSR $61204.l
060266  20 80                         MOVE.l d0,(a0)
060268  52 AE FF FC                   ADDQ.l #1,-$4(a6)
06026C  60 BE                         BRA $06022C
06026E  2D 7C 00 00 04 F8 FF F4       MOVE.l #$4F8,-$C(a6)
060276  4E 71                         NOP
060278  70 00                         MOVEQ #$0,d0
06027A  60 02                         BRA $06027E
06027C  .dc.b 70 01                                           ; p.
06027E  20 6E 00 08                   MOVEA.l $8(a6),a0
060282  21 40 00 10                   MOVE.l d0,$10(a0)
060286  20 2E FF F4                   MOVE.l -$C(a6),d0
06028A  22 3C 00 00 01 2C             MOVE.l #$12C,d1
060290  4E B9 00 06 12 04             JSR $61204.l
060296  22 3C 00 00 01 2C             MOVE.l #$12C,d1
06029C  4E B9 00 06 12 48             JSR $61248.l
0602A2  21 40 00 14                   MOVE.l d0,$14(a0)
0602A6  4E 71                         NOP
0602A8  48 79 00 06 0D F4             PEA $60DF4.l
0602AE  61 00 06 3E                   BSR $0608EE
0602B2  58 8F                         ADDQ.l #4,a7
0602B4  48 79 00 06 0D F4             PEA $60DF4.l
0602BA  2F 2E FF F0                   MOVE.l -$10(a6),-(a7)
0602BE  61 00 02 4C                   BSR $06050C
0602C2  50 8F                         ADDQ.l #8,a7
0602C4  2F 3C 00 00 3E 80             MOVE.l #$3E80,-(a7)
0602CA  2F 2E FF E8                   MOVE.l -$18(a6),-(a7)
0602CE  4E B9 00 06 10 1C             JSR $6101C.l
0602D4  50 8F                         ADDQ.l #8,a7
0602D6  20 6E 00 08                   MOVEA.l $8(a6),a0
0602DA  2F 28 00 04                   MOVE.l $4(a0),-(a7)
0602DE  2F 10                         MOVE.l (a0),-(a7)
0602E0  61 00 00 C0                   BSR $0603A2
0602E4  50 8F                         ADDQ.l #8,a7
0602E6  70 00                         MOVEQ #$0,d0
0602E8  4E 5E                         UNLK a6
0602EA  4E 75                         RTS
0602EC  48 79 00 06 0D F4             PEA $60DF4.l
0602F2  61 00 05 FA                   BSR $0608EE
0602F6  58 8F                         ADDQ.l #4,a7
0602F8  48 79 00 06 0D F4             PEA $60DF4.l
0602FE  2F 2E FF F0                   MOVE.l -$10(a6),-(a7)
060302  61 00 02 08                   BSR $06050C
060306  50 8F                         ADDQ.l #8,a7
060308  2F 3C 00 00 3E 80             MOVE.l #$3E80,-(a7)
06030E  2F 2E FF E8                   MOVE.l -$18(a6),-(a7)
060312  4E B9 00 06 10 1C             JSR $6101C.l
060318  50 8F                         ADDQ.l #8,a7
06031A  70 01                         MOVEQ #$1,d0
06031C  4E 5E                         UNLK a6
06031E  4E 75                         RTS

; ==== sub_060320 (1 caller) ====
060320  4E 56 FF E4                   LINK a6,#-$1C
060324  42 AE FF FC                   CLR.l -$4(a6)
060328  2F 2E FF FC                   MOVE.l -$4(a6),-(a7)
06032C  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
060330  61 00 07 CA                   BSR $060AFC
060334  50 8F                         ADDQ.l #8,a7
060336  20 40                         MOVEA.l d0,a0
060338  2D 48 FF E4                   MOVE.l a0,-$1C(a6)
06033C  50 88                         ADDQ.l #8,a0
06033E  2F 08                         MOVE.l a0,-(a7)
060340  61 00 0A 12                   BSR $060D54
060344  58 8F                         ADDQ.l #4,a7
060346  52 AE FF FC                   ADDQ.l #1,-$4(a6)
06034A  2D 40 FF E8                   MOVE.l d0,-$18(a6)
06034E  10 2E FF EB                   MOVE.b -$15(a6),d0
060352  53 00                         SUBQ.b #1,d0
060354  66 D2                         BNE $060328
060356  20 2E FF FC                   MOVE.l -$4(a6),d0
06035A  72 0B                         MOVEQ #$B,d1
06035C  4E B9 00 06 12 04             JSR $61204.l
060362  20 2E FF FC                   MOVE.l -$4(a6),d0
060366  53 80                         SUBQ.l #1,d0
060368  2F 00                         MOVE.l d0,-(a7)
06036A  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
06036E  2D 41 FF F8                   MOVE.l d1,-$8(a6)
060372  61 00 07 88                   BSR $060AFC
060376  50 8F                         ADDQ.l #8,a7
060378  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
06037C  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
060380  2D 40 FF F4                   MOVE.l d0,-$C(a6)
060384  61 00 07 76                   BSR $060AFC
060388  50 8F                         ADDQ.l #8,a7
06038A  72 00                         MOVEQ #$0,d1
06038C  12 2E FF E9                   MOVE.b -$17(a6),d1
060390  2D 40 FF F0                   MOVE.l d0,-$10(a6)
060394  90 AE FF F4                   SUB.l -$C(a6),d0
060398  04 80 00 00 04 40             SUBI.l #$440,d0
06039E  4E 5E                         UNLK a6
0603A0  4E 75                         RTS

; ==== load_session  $0603A2  (2 callers) — one load session: AllocMem a $38-byte device/IO control struct (the loader's IORequest-like block) and a 512-byte CHIP sector buffer, set up the drive, run the track reads (read_track / mfm chain), then FreeMem both. Returns 0 on success, 1 on alloc failure. ====
0603A2  4E 56 FF F4                   LINK a6,#-$C
0603A6  42 AE FF FC                   CLR.l -$4(a6)
0603AA  2F 3C 00 01 00 00             MOVE.l #$10000,-(a7)
0603B0  70 38                         MOVEQ #$38,d0
0603B2  2F 00                         MOVE.l d0,-(a7)
0603B4  4E B9 00 06 10 04             JSR $61004.l
0603BA  50 8F                         ADDQ.l #8,a7
0603BC  2F 00                         MOVE.l d0,-(a7)
0603BE  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0603C2  4E B9 00 06 0E 1C             JSR $60E1C.l
0603C8  58 8F                         ADDQ.l #4,a7
0603CA  42 A7                         CLR.l -(a7)
0603CC  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
0603D0  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0603D4  48 79 00 06 0D E0             PEA $60DE0.l
0603DA  4E B9 00 06 10 98             JSR $61098.l
0603E0  4F EF 00 10                   LEA $10(a7),a7
0603E4  2D 40 FF FC                   MOVE.l d0,-$4(a6)
0603E8  4A 80                         TST.l d0
0603EA  67 12                         BEQ $0603FE
0603EC  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
0603F0  4E B9 00 06 0E 40             JSR $60E40.l
0603F6  58 8F                         ADDQ.l #4,a7
0603F8  70 01                         MOVEQ #$1,d0
0603FA  4E 5E                         UNLK a6
0603FC  4E 75                         RTS
0603FE  2F 3C 00 01 00 02             MOVE.l #$10002,-(a7)
060404  2F 3C 00 00 02 00             MOVE.l #$200,-(a7)
06040A  4E B9 00 06 10 04             JSR $61004.l
060410  50 8F                         ADDQ.l #8,a7
060412  2D 40 FF F4                   MOVE.l d0,-$C(a6)
060416  20 2E 00 0C                   MOVE.l $C(a6),d0
06041A  72 0B                         MOVEQ #$B,d1
06041C  4E B9 00 06 12 48             JSR $61248.l
060422  72 01                         MOVEQ #$1,d1
060424  2F 01                         MOVE.l d1,-(a7)
060426  2F 2E FF F4                   MOVE.l -$C(a6),-(a7)
06042A  2F 00                         MOVE.l d0,-(a7)
06042C  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
060430  4E B9 00 06 0E 70             JSR $60E70.l
060436  4F EF 00 10                   LEA $10(a7),a7
06043A  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
06043E  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060442  4E B9 00 06 0F 9E             JSR $60F9E.l
060448  58 8F                         ADDQ.l #4,a7
06044A  2F 2E FF F8                   MOVE.l -$8(a6),-(a7)
06044E  4E B9 00 06 0E 40             JSR $60E40.l
060454  58 8F                         ADDQ.l #4,a7
060456  2F 3C 00 00 02 00             MOVE.l #$200,-(a7)
06045C  2F 2E FF F4                   MOVE.l -$C(a6),-(a7)
060460  4E B9 00 06 10 1C             JSR $6101C.l
060466  50 8F                         ADDQ.l #8,a7
060468  70 00                         MOVEQ #$0,d0
06046A  4E 5E                         UNLK a6
06046C  4E 75                         RTS

; ==== disk_hw_init  $06046E  (1 caller) — bring up the raw floppy hardware: copy a 20-byte parameter/identity table ($60E08) into the IO struct, set CIA-A DDRA ($BFE201) for the drive-status inputs, and idle CIA-B PRB ($BFD100 = $FF: all /SELx high, motor off). $DFF000 kept as the custom-chip base. ====
06046E  4E 56 FF F4                   LINK a6,#-$C
060472  2D 7C 00 DF F0 00 FF FC       MOVE.l #$DFF000,-$4(a6)
06047A  41 F9 00 06 0E 08             LEA $60E08.l,a0
060480  22 6E 00 08                   MOVEA.l $8(a6),a1
060484  70 13                         MOVEQ #$13,d0
060486  12 D8                         MOVE.b (a0)+,(a1)+
060488  51 C8 FF FC                   DBRA d0,$060486
06048C  10 39 00 BF E2 01             MOVE.b $BFE201.l,d0
060492  02 00 00 C3                   ANDI.b #$C3,d0
060496  13 C0 00 BF E2 01             MOVE.b d0,$BFE201.l
06049C  70 FF                         MOVEQ #$FF,d0
06049E  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
0604A4  13 C0 00 BF D3 00             MOVE.b d0,$BFD300.l
0604AA  20 2E 00 0C                   MOVE.l $C(a6),d0
0604AE  56 80                         ADDQ.l #3,d0
0604B0  72 01                         MOVEQ #$1,d1
0604B2  E1 A1                         ASL.l d0,d1
0604B4  20 6E 00 08                   MOVEA.l $8(a6),a0
0604B8  10 81                         MOVE.b d1,(a0)
0604BA  21 6E 00 10 00 08             MOVE.l $10(a6),$8(a0)
0604C0  20 6E FF FC                   MOVEA.l -$4(a6),a0
0604C4  30 28 00 1C                   MOVE.w $1C(a0),d0
0604C8  31 7C 7F FF 00 9A             MOVE.w #$7FFF,$9A(a0)
0604CE  31 7C 82 10 00 96             MOVE.w #$8210,$96(a0)
0604D4  32 28 00 10                   MOVE.w $10(a0),d1
0604D8  31 7C 7F 00 00 9E             MOVE.w #$7F00,$9E(a0)
0604DE  31 7C 85 00 00 9E             MOVE.w #$8500,$9E(a0)
0604E4  31 7C 44 89 00 7E             MOVE.w #$4489,$7E(a0)
0604EA  3D 40 FF FA                   MOVE.w d0,-$6(a6)
0604EE  02 80 00 00 FF FF             ANDI.l #$FFFF,d0
0604F4  74 10                         MOVEQ #$10,d2
0604F6  2D 40 FF F4                   MOVE.l d0,-$C(a6)
0604FA  E5 A0                         ASL.l d2,d0
0604FC  3D 41 FF F8                   MOVE.w d1,-$8(a6)
060500  02 81 00 00 FF FF             ANDI.l #$FFFF,d1
060506  80 81                         OR.l d1,d0
060508  4E 5E                         UNLK a6
06050A  4E 75                         RTS

; ==== dma_setup  $06050C  (2 callers) — program Paula for a disk transfer: write DMACON ($DFF09A) — clear all ($7FFF) then set the requested bits|$8000 — and ADKCON ($DFF09E) the same way (MFM/word-sync/precomp). Args are the DMACON/ADKCON masks; tail-calls $6075E. ====
06050C  4E 56 FF F8                   LINK a6,#-$8
060510  2D 7C 00 DF F0 00 FF F8       MOVE.l #$DFF000,-$8(a6)
060518  20 2E 00 08                   MOVE.l $8(a6),d0
06051C  02 40 FF FF                   ANDI.w #$FFFF,d0
060520  72 10                         MOVEQ #$10,d1
060522  24 2E 00 08                   MOVE.l $8(a6),d2
060526  E2 A2                         ASR.l d1,d2
060528  32 3C 7F FF                   MOVE.w #$7FFF,d1
06052C  20 6E FF F8                   MOVEA.l -$8(a6),a0
060530  31 41 00 9A                   MOVE.w d1,$9A(a0)
060534  3D 42 FF FC                   MOVE.w d2,-$4(a6)
060538  00 42 80 00                   ORI.w #$8000,d2
06053C  31 42 00 9A                   MOVE.w d2,$9A(a0)
060540  31 41 00 9E                   MOVE.w d1,$9E(a0)
060544  3D 40 FF FE                   MOVE.w d0,-$2(a6)
060548  00 40 80 00                   ORI.w #$8000,d0
06054C  31 40 00 9E                   MOVE.w d0,$9E(a0)
060550  2F 2E 00 0C                   MOVE.l $C(a6),-(a7)
060554  61 00 02 08                   BSR $06075E
060558  58 8F                         ADDQ.l #4,a7
06055A  4E 5E                         UNLK a6
06055C  4E 75                         RTS

; ==== seek_track0  $06055E  (1 caller) — recalibrate: motor/select on, then pulse /STEP via CIA-B PRB ($BFD100, toggling the step/dir bits) while polling CIA-A PRA ($BFE001) /TK0 until the head reaches track 0; $3A98 (15000) head-settle delay. ====
06055E  4E 56 00 00                   LINK a6,#$0
060562  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
060566  61 00 01 B4                   BSR $06071C
06056A  58 8F                         ADDQ.l #4,a7
06056C  20 6E 00 08                   MOVEA.l $8(a6),a0
060570  42 68 00 06                   CLR.w $6(a0)
060574  10 39 00 BF D1 00             MOVE.b $BFD100.l,d0
06057A  00 00 00 04                   ORI.b #$4,d0
06057E  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
060584  02 00 00 FD                   ANDI.b #$FD,d0
060588  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
06058E  70 00                         MOVEQ #$0,d0
060590  10 39 00 BF E0 01             MOVE.b $BFE001.l,d0
060596  00 80 FF FF FF EF             ORI.l #$FFFFFFEF,d0
06059C  52 80                         ADDQ.l #1,d0
06059E  67 0C                         BEQ $0605AC
0605A0  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0605A4  61 00 01 F8                   BSR $06079E
0605A8  58 8F                         ADDQ.l #4,a7
0605AA  60 E2                         BRA $06058E
0605AC  10 39 00 BF D1 00             MOVE.b $BFD100.l,d0
0605B2  00 00 00 02                   ORI.b #$2,d0
0605B6  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
0605BC  70 00                         MOVEQ #$0,d0
0605BE  10 39 00 BF E0 01             MOVE.b $BFE001.l,d0
0605C4  00 80 FF FF FF EF             ORI.l #$FFFFFFEF,d0
0605CA  52 80                         ADDQ.l #1,d0
0605CC  66 0C                         BNE $0605DA
0605CE  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0605D2  61 00 01 CA                   BSR $06079E
0605D6  58 8F                         ADDQ.l #4,a7
0605D8  60 E2                         BRA $0605BC
0605DA  2F 3C 00 00 3A 98             MOVE.l #$3A98,-(a7)
0605E0  61 00 01 94                   BSR $060776
0605E4  58 8F                         ADDQ.l #4,a7
0605E6  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0605EA  61 00 01 72                   BSR $06075E
0605EE  58 8F                         ADDQ.l #4,a7
0605F0  4E 5E                         UNLK a6
0605F2  4E 75                         RTS

; ==== sub_0605F4 (1 caller) ====
0605F4  4E 56 FF E4                   LINK a6,#-$1C
0605F8  48 E7 08 00                   MOVEM.l d4,-(a7)
0605FC  20 2E 00 0C                   MOVE.l $C(a6),d0
060600  02 80 00 00 00 01             ANDI.l #$1,d0
060606  22 2E 00 0C                   MOVE.l $C(a6),d1
06060A  E2 81                         ASR.l #1,d1
06060C  20 6E 00 08                   MOVEA.l $8(a6),a0
060610  34 28 00 06                   MOVE.w $6(a0),d2
060614  48 C2                         EXT.L d2
060616  2F 42 00 04                   MOVE.l d2,$4(a7)
06061A  02 82 00 00 00 01             ANDI.l #$1,d2
060620  26 2F 00 04                   MOVE.l $4(a7),d3
060624  E2 83                         ASR.l #1,d3
060626  78 00                         MOVEQ #$0,d4
060628  2D 44 FF EC                   MOVE.l d4,-$14(a6)
06062C  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060630  2D 41 FF F8                   MOVE.l d1,-$8(a6)
060634  2D 42 FF F4                   MOVE.l d2,-$C(a6)
060638  2D 43 FF F0                   MOVE.l d3,-$10(a6)
06063C  20 2F 00 04                   MOVE.l $4(a7),d0
060640  B0 AE 00 0C                   CMP.l $C(a6),d0
060644  66 0A                         BNE $060650
060646  20 04                         MOVE.l d4,d0
060648  4C DF                         .dc.w $4CDF
06064A  .dc.b 00 10 4E 5E 4E 75                               ; ..N^Nu
060650  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
060654  61 00 00 C6                   BSR $06071C
060658  58 8F                         ADDQ.l #4,a7
06065A  20 2E FF FC                   MOVE.l -$4(a6),d0
06065E  22 2E FF F4                   MOVE.l -$C(a6),d1
060662  B2 80                         CMP.l d0,d1
060664  67 32                         BEQ $060698
060666  4A 80                         TST.l d0
060668  67 12                         BEQ $06067C
06066A  10 39 00 BF D1 00             MOVE.b $BFD100.l,d0
060670  02 00 00 FB                   ANDI.b #$FB,d0
060674  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
06067A  60 10                         BRA $06068C
06067C  10 39 00 BF D1 00             MOVE.b $BFD100.l,d0
060682  00 00 00 04                   ORI.b #$4,d0
060686  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
06068C  2F 3C 00 00 00 C8             MOVE.l #$C8,-(a7)
060692  61 00 00 E2                   BSR $060776
060696  58 8F                         ADDQ.l #4,a7
060698  20 2E FF F8                   MOVE.l -$8(a6),d0
06069C  22 2E FF F0                   MOVE.l -$10(a6),d1
0606A0  B2 80                         CMP.l d0,d1
0606A2  67 5C                         BEQ $060700
0606A4  B2 80                         CMP.l d0,d1
0606A6  6C 18                         BGE $0606C0
0606A8  14 39 00 BF D1 00             MOVE.b $BFD100.l,d2
0606AE  02 02 00 FD                   ANDI.b #$FD,d2
0606B2  13 C2 00 BF D1 00             MOVE.b d2,$BFD100.l
0606B8  90 81                         SUB.l d1,d0
0606BA  2D 40 FF E8                   MOVE.l d0,-$18(a6)
0606BE  60 1C                         BRA $0606DC
0606C0  10 39 00 BF D1 00             MOVE.b $BFD100.l,d0
0606C6  00 00 00 02                   ORI.b #$2,d0
0606CA  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
0606D0  20 2E FF F0                   MOVE.l -$10(a6),d0
0606D4  90 AE FF F8                   SUB.l -$8(a6),d0
0606D8  2D 40 FF E8                   MOVE.l d0,-$18(a6)
0606DC  20 2E FF E8                   MOVE.l -$18(a6),d0
0606E0  53 AE FF E8                   SUBQ.l #1,-$18(a6)
0606E4  4A 80                         TST.l d0
0606E6  67 0C                         BEQ $0606F4
0606E8  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0606EC  61 00 00 B0                   BSR $06079E
0606F0  58 8F                         ADDQ.l #4,a7
0606F2  60 E8                         BRA $0606DC
0606F4  2F 3C 00 00 3A 98             MOVE.l #$3A98,-(a7)
0606FA  61 00 00 7A                   BSR $060776
0606FE  58 8F                         ADDQ.l #4,a7
060700  20 2E 00 0C                   MOVE.l $C(a6),d0
060704  20 6E 00 08                   MOVEA.l $8(a6),a0
060708  31 40 00 06                   MOVE.w d0,$6(a0)
06070C  2F 08                         MOVE.l a0,-(a7)
06070E  61 00 00 4E                   BSR $06075E
060712  58 8F                         ADDQ.l #4,a7
060714  4C DF                         .dc.w $4CDF
060716  .dc.b 00 10 4E 5E 4E 75                               ; ..N^Nu

; ==== drive_select  $06071C  (5 callers) — select drive 0 and turn the motor on via CIA-B PRB ($BFD100). ====
06071C  4E 56 00 00                   LINK a6,#$0
060720  10 39 00 BF D1 00             MOVE.b $BFD100.l,d0
060726  00 00 00 80                   ORI.b #$80,d0
06072A  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
060730  72 00                         MOVEQ #$0,d1
060732  20 6E 00 08                   MOVEA.l $8(a6),a0
060736  12 28 00 01                   MOVE.b $1(a0),d1
06073A  EF 81                         ASL.l #7,d1
06073C  46 81                         NOT.l d1
06073E  74 00                         MOVEQ #$0,d2
060740  14 39 00 BF D1 00             MOVE.b $BFD100.l,d2
060746  C4 81                         AND.l d1,d2
060748  13 C2 00 BF D1 00             MOVE.b d2,$BFD100.l
06074E  10 10                         MOVE.b (a0),d0
060750  46 00                         NOT.b d0
060752  C4 00                         AND.b d0,d2
060754  13 C2 00 BF D1 00             MOVE.b d2,$BFD100.l
06075A  4E 5E                         UNLK a6
06075C  4E 75                         RTS

; ==== dma_wait  $06075E  (6 callers) — wait for the disk-DMA / blit to finish (paired with dma_setup). ====
06075E  4E 56 00 00                   LINK a6,#$0
060762  10 39 00 BF D1 00             MOVE.b $BFD100.l,d0
060768  00 00 00 78                   ORI.b #$78,d0
06076C  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
060772  4E 5E                         UNLK a6
060774  4E 75                         RTS

; ==== sub_060776 (5 callers) ====
060776  4E 56 00 00                   LINK a6,#$0
06077A  20 2E 00 08                   MOVE.l $8(a6),d0
06077E  E4 80                         ASR.l #2,d0
060780  2D 40 00 08                   MOVE.l d0,$8(a6)
060784  20 2E 00 08                   MOVE.l $8(a6),d0
060788  53 80                         SUBQ.l #1,d0
06078A  2D 40 00 08                   MOVE.l d0,$8(a6)
06078E  4A 80                         TST.l d0
060790  67 04                         BEQ $060796
060792  61 06                         BSR $06079A
060794  60 EE                         BRA $060784
060796  4E 5E                         UNLK a6
060798  4E 75                         RTS

; ==== sub_06079A (1 caller) ====
06079A  70 00                         MOVEQ #$0,d0
06079C  4E 75                         RTS

; ==== step_wait  $06079E  (3 callers) — a short busy-wait used between drive-status polls while seeking. ====
06079E  4E 56 00 00                   LINK a6,#$0
0607A2  10 39 00 BF D1 00             MOVE.b $BFD100.l,d0
0607A8  02 00 00 FE                   ANDI.b #$FE,d0
0607AC  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
0607B2  00 00 00 01                   ORI.b #$1,d0
0607B6  13 C0 00 BF D1 00             MOVE.b d0,$BFD100.l
0607BC  2F 3C 00 00 0B B8             MOVE.l #$BB8,-(a7)
0607C2  61 B2                         BSR $060776
0607C4  58 8F                         ADDQ.l #4,a7
0607C6  4E 5E                         UNLK a6
0607C8  4E 75                         RTS

; ==== read_track  $0607CA  (1 caller) — read one raw MFM track via Paula disk DMA: clear INTREQ DSKBLK ($DFF09C #2), set DSKPT ($DFF020) to the track buffer, check CIA-A PRA /RDY ($BFE001), then write DSKLEN ($DFF024) = (len/2)|$8000 TWICE — the standard Amiga "arm disk DMA" sequence — pulling a full $36F2-byte (14066) MFM track; $30D40 (200000) read timeout. ====
0607CA  4E 56 FF E4                   LINK a6,#-$1C
0607CE  48 E7 00 20                   MOVEM.l a2,-(a7)
0607D2  2D 7C 00 DF F0 00 FF FC       MOVE.l #$DFF000,-$4(a6)
0607DA  22 6E 00 08                   MOVEA.l $8(a6),a1
0607DE  20 69 00 08                   MOVEA.l $8(a1),a0
0607E2  5C 88                         ADDQ.l #6,a0
0607E4  2D 7C 00 00 36 F2 FF F4       MOVE.l #$36F2,-$C(a6)
0607EC  70 00                         MOVEQ #$0,d0
0607EE  2D 40 FF E8                   MOVE.l d0,-$18(a6)
0607F2  2D 7C 00 03 0D 40 FF E4       MOVE.l #$30D40,-$1C(a6)
0607FA  24 6E FF FC                   MOVEA.l -$4(a6),a2
0607FE  35 7C 00 02 00 9C             MOVE.w #$2,$9C(a2)
060804  22 08                         MOVE.l a0,d1
060806  25 41 00 20                   MOVE.l d1,$20(a2)
06080A  2F 09                         MOVE.l a1,-(a7)
06080C  2D 40 FF EC                   MOVE.l d0,-$14(a6)
060810  2D 41 FF F8                   MOVE.l d1,-$8(a6)
060814  61 00 FF 06                   BSR $06071C
060818  58 8F                         ADDQ.l #4,a7
06081A  70 00                         MOVEQ #$0,d0
06081C  10 39 00 BF E0 01             MOVE.b $BFE001.l,d0
060822  00 80 FF FF FF FB             ORI.l #$FFFFFFFB,d0
060828  52 80                         ADDQ.l #1,d0
06082A  66 68                         BNE $060894
06082C  20 6E FF FC                   MOVEA.l -$4(a6),a0
060830  42 68 00 24                   CLR.w $24(a0)
060834  20 2E FF F4                   MOVE.l -$C(a6),d0
060838  E2 80                         ASR.l #1,d0
06083A  00 80 00 00 80 00             ORI.l #$8000,d0
060840  31 40 00 24                   MOVE.w d0,$24(a0)
060844  31 40 00 24                   MOVE.w d0,$24(a0)
060848  2D 40 FF F0                   MOVE.l d0,-$10(a6)
06084C  20 2E FF E4                   MOVE.l -$1C(a6),d0
060850  53 80                         SUBQ.l #1,d0
060852  2D 40 FF E4                   MOVE.l d0,-$1C(a6)
060856  4A 80                         TST.l d0
060858  6A 0A                         BPL $060864
06085A  70 16                         MOVEQ #$16,d0
06085C  4C DF                         .dc.w $4CDF
06085E  .dc.b 04 00 4E 5E 4E 75                               ; ..N^Nu
060864  70 00                         MOVEQ #$0,d0
060866  20 6E FF FC                   MOVEA.l -$4(a6),a0
06086A  30 28 00 1E                   MOVE.w $1E(a0),d0
06086E  2D 40 FF F0                   MOVE.l d0,-$10(a6)
060872  08 2E 00 01 FF F3             BTST.b #$1,-$D(a6)
060878  67 D2                         BEQ $06084C
06087A  20 6E FF FC                   MOVEA.l -$4(a6),a0
06087E  42 68 00 24                   CLR.w $24(a0)
060882  70 00                         MOVEQ #$0,d0
060884  10 39 00 BF E0 01             MOVE.b $BFE001.l,d0
06088A  00 80 FF FF FF FB             ORI.l #$FFFFFFFB,d0
060890  52 80                         ADDQ.l #1,d0
060892  67 06                         BEQ $06089A
060894  70 19                         MOVEQ #$19,d0
060896  2D 40 FF EC                   MOVE.l d0,-$14(a6)
06089A  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
06089E  61 00 FE BE                   BSR $06075E
0608A2  58 8F                         ADDQ.l #4,a7
0608A4  20 2E FF EC                   MOVE.l -$14(a6),d0
0608A8  4C DF                         .dc.w $4CDF
0608AA  .dc.b 04 00 4E 5E 4E 75                               ; ..N^Nu

; ==== sub_0608B0 (1 caller) ====
0608B0  4E 56 00 00                   LINK a6,#$0
0608B4  20 6E 00 08                   MOVEA.l $8(a6),a0
0608B8  4A 28 00 01                   TST.b $1(a0)
0608BC  67 04                         BEQ $0608C2
0608BE  4E 5E                         UNLK a6
0608C0  4E 75                         RTS
0608C2  20 6E 00 08                   MOVEA.l $8(a6),a0
0608C6  11 7C 00 01 00 01             MOVE.b #$1,$1(a0)
0608CC  2F 08                         MOVE.l a0,-(a7)
0608CE  61 00 FE 4C                   BSR $06071C
0608D2  58 8F                         ADDQ.l #4,a7
0608D4  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0608D8  61 00 FE 84                   BSR $06075E
0608DC  58 8F                         ADDQ.l #4,a7
0608DE  2F 3C 00 07 A1 20             MOVE.l #$7A120,-(a7)
0608E4  61 00 FE 90                   BSR $060776
0608E8  58 8F                         ADDQ.l #4,a7
0608EA  4E 5E                         UNLK a6
0608EC  4E 75                         RTS

; ==== sub_0608EE (2 callers) ====
0608EE  4E 56 00 00                   LINK a6,#$0
0608F2  20 6E 00 08                   MOVEA.l $8(a6),a0
0608F6  4A 28 00 01                   TST.b $1(a0)
0608FA  66 04                         BNE $060900
0608FC  4E 5E                         UNLK a6
0608FE  4E 75                         RTS
060900  20 6E 00 08                   MOVEA.l $8(a6),a0
060904  42 28 00 01                   CLR.b $1(a0)
060908  2F 08                         MOVE.l a0,-(a7)
06090A  61 00 FE 10                   BSR $06071C
06090E  58 8F                         ADDQ.l #4,a7
060910  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
060914  61 00 FE 48                   BSR $06075E
060918  58 8F                         ADDQ.l #4,a7
06091A  4E 5E                         UNLK a6
06091C  4E 75                         RTS

; ==== sub_06091E (1 caller) ====
06091E  4E 56 FF F8                   LINK a6,#-$8
060922  20 6E 00 08                   MOVEA.l $8(a6),a0
060926  20 08                         MOVE.l a0,d0
060928  2D 40 FF FC                   MOVE.l d0,-$4(a6)
06092C  D0 AE 00 0C                   ADD.l $C(a6),d0
060930  2D 40 FF F8                   MOVE.l d0,-$8(a6)
060934  20 6E FF FC                   MOVEA.l -$4(a6),a0
060938  B1 EE FF F8                   CMPA.l -$8(a6),a0
06093C  64 2C                         BCC $06096A
06093E  30 10                         MOVE.w (a0),d0
060940  0C 40 44 89                   CMPI.w #$4489,d0
060944  66 1E                         BNE $060964
060946  30 28 00 02                   MOVE.w $2(a0),d0
06094A  0C 40 44 89                   CMPI.w #$4489,d0
06094E  66 08                         BNE $060958
060950  59 88                         SUBQ.l #4,a0
060952  20 08                         MOVE.l a0,d0
060954  4E 5E                         UNLK a6
060956  4E 75                         RTS
060958  20 6E FF FC                   MOVEA.l -$4(a6),a0
06095C  5D 88                         SUBQ.l #6,a0
06095E  20 08                         MOVE.l a0,d0
060960  4E 5E                         UNLK a6
060962  4E 75                         RTS
060964  54 AE FF FC                   ADDQ.l #2,-$4(a6)
060968  60 CA                         BRA $060934
06096A  70 00                         MOVEQ #$0,d0
06096C  4E 5E                         UNLK a6
06096E  4E 75                         RTS

; ==== sub_060970 (1 caller) ====
060970  4E 56 FF E8                   LINK a6,#-$18
060974  22 6E 00 08                   MOVEA.l $8(a6),a1
060978  20 69 00 08                   MOVEA.l $8(a1),a0
06097C  5C 88                         ADDQ.l #6,a0
06097E  42 AE FF F0                   CLR.l -$10(a6)
060982  30 10                         MOVE.w (a0),d0
060984  2D 48 FF FC                   MOVE.l a0,-$4(a6)
060988  0C 40 44 89                   CMPI.w #$4489,d0
06098C  67 0A                         BEQ $060998
06098E  70 16                         MOVEQ #$16,d0
060990  2D 40 FF F0                   MOVE.l d0,-$10(a6)
060994  60 00 01 5E                   BRA $060AF4
060998  20 6E FF FC                   MOVEA.l -$4(a6),a0
06099C  30 28 00 02                   MOVE.w $2(a0),d0
0609A0  0C 40 44 89                   CMPI.w #$4489,d0
0609A4  66 0A                         BNE $0609B0
0609A6  59 88                         SUBQ.l #4,a0
0609A8  20 08                         MOVE.l a0,d0
0609AA  2D 40 FF F4                   MOVE.l d0,-$C(a6)
0609AE  60 0C                         BRA $0609BC
0609B0  20 6E FF FC                   MOVEA.l -$4(a6),a0
0609B4  5D 88                         SUBQ.l #6,a0
0609B6  20 08                         MOVE.l a0,d0
0609B8  2D 40 FF F4                   MOVE.l d0,-$C(a6)
0609BC  30 3C AA AA                   MOVE.w #$AAAA,d0
0609C0  20 6E FF F4                   MOVEA.l -$C(a6),a0
0609C4  30 80                         MOVE.w d0,(a0)
0609C6  31 40 00 02                   MOVE.w d0,$2(a0)
0609CA  31 7C 44 89 00 04             MOVE.w #$4489,$4(a0)
0609D0  22 6E 00 08                   MOVEA.l $8(a6),a1
0609D4  23 48 00 0C                   MOVE.l a0,$C(a1)
0609D8  50 88                         ADDQ.l #8,a0
0609DA  2F 08                         MOVE.l a0,-(a7)
0609DC  61 00 03 76                   BSR $060D54
0609E0  58 8F                         ADDQ.l #4,a7
0609E2  2D 40 FF F8                   MOVE.l d0,-$8(a6)
0609E6  48 6E FF F8                   PEA -$8(a6)
0609EA  2F 2E FF F4                   MOVE.l -$C(a6),-(a7)
0609EE  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
0609F2  61 00 01 74                   BSR $060B68
0609F6  4F EF 00 0C                   LEA $C(a7),a7
0609FA  2D 40 FF F0                   MOVE.l d0,-$10(a6)
0609FE  4A 80                         TST.l d0
060A00  66 00 00 F2                   BNE $060AF4
060A04  10 2E FF FB                   MOVE.b -$5(a6),d0
060A08  20 6E 00 08                   MOVEA.l $8(a6),a0
060A0C  11 40 00 04                   MOVE.b d0,$4(a0)
060A10  11 6E FF FA 00 03             MOVE.b -$6(a6),$3(a0)
060A16  10 28 00 04                   MOVE.b $4(a0),d0
060A1A  0C 00 00 0B                   CMPI.b #$B,d0
060A1E  64 40                         BCC $060A60
060A20  72 00                         MOVEQ #$0,d1
060A22  12 28 00 04                   MOVE.b $4(a0),d1
060A26  20 3C 00 00 04 40             MOVE.l #$440,d0
060A2C  4E B9 00 06 12 48             JSR $61248.l
060A32  20 6E FF F4                   MOVEA.l -$C(a6),a0
060A36  D1 C0                         ADDA.l d0,a0
060A38  2F 3C 00 00 0D 64             MOVE.l #$D64,-(a7)
060A3E  2F 08                         MOVE.l a0,-(a7)
060A40  61 00 FE DC                   BSR $06091E
060A44  50 8F                         ADDQ.l #8,a7
060A46  20 6E 00 08                   MOVEA.l $8(a6),a0
060A4A  21 40 00 10                   MOVE.l d0,$10(a0)
060A4E  2D 40 FF F4                   MOVE.l d0,-$C(a6)
060A52  4A 80                         TST.l d0
060A54  66 12                         BNE $060A68
060A56  70 17                         MOVEQ #$17,d0
060A58  2D 40 FF F0                   MOVE.l d0,-$10(a6)
060A5C  60 00 00 96                   BRA $060AF4
060A60  20 6E 00 08                   MOVEA.l $8(a6),a0
060A64  42 A8 00 10                   CLR.l $10(a0)
060A68  42 AE FF EC                   CLR.l -$14(a6)
060A6C  20 2E FF EC                   MOVE.l -$14(a6),d0
060A70  0C 80 00 00 00 0B             CMPI.l #$B,d0
060A76  6C 7C                         BGE $060AF4
060A78  2F 00                         MOVE.l d0,-(a7)
060A7A  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
060A7E  61 7C                         BSR $060AFC
060A80  50 8F                         ADDQ.l #8,a7
060A82  20 40                         MOVEA.l d0,a0
060A84  2D 48 FF F4                   MOVE.l a0,-$C(a6)
060A88  50 88                         ADDQ.l #8,a0
060A8A  2F 08                         MOVE.l a0,-(a7)
060A8C  61 00 02 C6                   BSR $060D54
060A90  58 8F                         ADDQ.l #4,a7
060A92  2D 40 FF F8                   MOVE.l d0,-$8(a6)
060A96  48 6E FF F8                   PEA -$8(a6)
060A9A  2F 2E FF F4                   MOVE.l -$C(a6),-(a7)
060A9E  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
060AA2  61 00 00 C4                   BSR $060B68
060AA6  4F EF 00 0C                   LEA $C(a7),a7
060AAA  2D 40 FF F0                   MOVE.l d0,-$10(a6)
060AAE  4A 80                         TST.l d0
060AB0  66 42                         BNE $060AF4
060AB2  20 6E FF F4                   MOVEA.l -$C(a6),a0
060AB6  D0 FC 00 40                   ADDA.w #$40,a0
060ABA  2F 3C 00 00 04 00             MOVE.l #$400,-(a7)
060AC0  2F 08                         MOVE.l a0,-(a7)
060AC2  61 00 01 5E                   BSR $060C22
060AC6  50 8F                         ADDQ.l #8,a7
060AC8  20 6E FF F4                   MOVEA.l -$C(a6),a0
060ACC  D0 FC 00 38                   ADDA.w #$38,a0
060AD0  2F 08                         MOVE.l a0,-(a7)
060AD2  2F 40 00 04                   MOVE.l d0,$4(a7)
060AD6  61 00 02 7C                   BSR $060D54
060ADA  58 8F                         ADDQ.l #4,a7
060ADC  22 2F 00 00                   MOVE.l $0(a7),d1
060AE0  B2 80                         CMP.l d0,d1
060AE2  67 08                         BEQ $060AEC
060AE4  70 18                         MOVEQ #$18,d0
060AE6  2D 40 FF F0                   MOVE.l d0,-$10(a6)
060AEA  60 08                         BRA $060AF4
060AEC  52 AE FF EC                   ADDQ.l #1,-$14(a6)
060AF0  60 00 FF 7A                   BRA $060A6C
060AF4  20 2E FF F0                   MOVE.l -$10(a6),d0
060AF8  4E 5E                         UNLK a6
060AFA  4E 75                         RTS

; ==== sub_060AFC (4 callers) ====
060AFC  4E 56 FF F4                   LINK a6,#-$C
060B00  70 00                         MOVEQ #$0,d0
060B02  20 6E 00 08                   MOVEA.l $8(a6),a0
060B06  10 28 00 03                   MOVE.b $3(a0),d0
060B0A  72 00                         MOVEQ #$0,d1
060B0C  12 28 00 04                   MOVE.b $4(a0),d1
060B10  2D 68 00 0C FF F4             MOVE.l $C(a0),-$C(a6)
060B16  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060B1A  2D 41 FF F8                   MOVE.l d1,-$8(a6)
060B1E  20 2E 00 0C                   MOVE.l $C(a6),d0
060B22  B0 AE FF FC                   CMP.l -$4(a6),d0
060B26  66 08                         BNE $060B30
060B28  20 2E FF F4                   MOVE.l -$C(a6),d0
060B2C  4E 5E                         UNLK a6
060B2E  4E 75                         RTS
060B30  06 AE 00 00 04 40 FF F4       ADDI.l #$440,-$C(a6)
060B38  20 2E FF F8                   MOVE.l -$8(a6),d0
060B3C  53 80                         SUBQ.l #1,d0
060B3E  2D 40 FF F8                   MOVE.l d0,-$8(a6)
060B42  4A 80                         TST.l d0
060B44  66 0A                         BNE $060B50
060B46  20 6E 00 08                   MOVEA.l $8(a6),a0
060B4A  2D 68 00 10 FF F4             MOVE.l $10(a0),-$C(a6)
060B50  20 2E FF FC                   MOVE.l -$4(a6),d0
060B54  52 80                         ADDQ.l #1,d0
060B56  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060B5A  0C 80 00 00 00 0B             CMPI.l #$B,d0
060B60  6D BC                         BLT $060B1E
060B62  42 AE FF FC                   CLR.l -$4(a6)
060B66  60 B6                         BRA $060B1E

; ==== sub_060B68 (2 callers) ====
060B68  4E 56 FF F8                   LINK a6,#-$8
060B6C  20 6E 00 0C                   MOVEA.l $C(a6),a0
060B70  30 28 00 06                   MOVE.w $6(a0),d0
060B74  0C 40 44 89                   CMPI.w #$4489,d0
060B78  67 08                         BEQ $060B82
060B7A  70 04                         MOVEQ #$4,d0
060B7C  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060B80  60 2E                         BRA $060BB0
060B82  20 6E 00 0C                   MOVEA.l $C(a6),a0
060B86  50 88                         ADDQ.l #8,a0
060B88  70 28                         MOVEQ #$28,d0
060B8A  2F 00                         MOVE.l d0,-(a7)
060B8C  2F 08                         MOVE.l a0,-(a7)
060B8E  61 00 00 92                   BSR $060C22
060B92  50 8F                         ADDQ.l #8,a7
060B94  20 6E 00 0C                   MOVEA.l $C(a6),a0
060B98  D0 FC 00 30                   ADDA.w #$30,a0
060B9C  2F 08                         MOVE.l a0,-(a7)
060B9E  2F 40 00 04                   MOVE.l d0,$4(a7)
060BA2  61 00 01 B0                   BSR $060D54
060BA6  58 8F                         ADDQ.l #4,a7
060BA8  22 2F 00 00                   MOVE.l $0(a7),d1
060BAC  B2 80                         CMP.l d0,d1
060BAE  67 08                         BEQ $060BB8
060BB0  20 2E FF FC                   MOVE.l -$4(a6),d0
060BB4  4E 5E                         UNLK a6
060BB6  4E 75                         RTS
060BB8  20 6E 00 10                   MOVEA.l $10(a6),a0
060BBC  10 10                         MOVE.b (a0),d0
060BBE  0C 00 00 FF                   CMPI.b #$FF,d0
060BC2  67 08                         BEQ $060BCC
060BC4  70 05                         MOVEQ #$5,d0
060BC6  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060BCA  60 E4                         BRA $060BB0
060BCC  70 00                         MOVEQ #$0,d0
060BCE  20 6E 00 10                   MOVEA.l $10(a6),a0
060BD2  10 28 00 01                   MOVE.b $1(a0),d0
060BD6  20 6E 00 08                   MOVEA.l $8(a6),a0
060BDA  32 28 00 06                   MOVE.w $6(a0),d1
060BDE  48 C1                         EXT.L d1
060BE0  B0 81                         CMP.l d1,d0
060BE2  67 08                         BEQ $060BEC
060BE4  70 06                         MOVEQ #$6,d0
060BE6  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060BEA  60 C4                         BRA $060BB0
060BEC  20 6E 00 10                   MOVEA.l $10(a6),a0
060BF0  10 28 00 02                   MOVE.b $2(a0),d0
060BF4  0C 00 00 0B                   CMPI.b #$B,d0
060BF8  65 08                         BCS $060C02
060BFA  70 07                         MOVEQ #$7,d0
060BFC  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060C00  60 AE                         BRA $060BB0
060C02  20 6E 00 10                   MOVEA.l $10(a6),a0
060C06  10 28 00 03                   MOVE.b $3(a0),d0
060C0A  4A 00                         TST.b d0
060C0C  67 06                         BEQ $060C14
060C0E  0C 00 00 0B                   CMPI.b #$B,d0
060C12  63 08                         BLS $060C1C
060C14  70 08                         MOVEQ #$8,d0
060C16  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060C1A  60 94                         BRA $060BB0
060C1C  70 00                         MOVEQ #$0,d0
060C1E  4E 5E                         UNLK a6
060C20  4E 75                         RTS

; ==== sub_060C22 (2 callers) ====
060C22  4E 56 FF F8                   LINK a6,#-$8
060C26  20 2E 00 0C                   MOVE.l $C(a6),d0
060C2A  E4 80                         ASR.l #2,d0
060C2C  42 AE FF F8                   CLR.l -$8(a6)
060C30  2D 40 FF FC                   MOVE.l d0,-$4(a6)
060C34  4A AE FF FC                   TST.l -$4(a6)
060C38  67 14                         BEQ $060C4E
060C3A  53 AE FF FC                   SUBQ.l #1,-$4(a6)
060C3E  20 6E 00 08                   MOVEA.l $8(a6),a0
060C42  20 10                         MOVE.l (a0),d0
060C44  B1 AE FF F8                   EOR.l d0,-$8(a6)
060C48  58 AE 00 08                   ADDQ.l #4,$8(a6)
060C4C  60 E6                         BRA $060C34
060C4E  20 2E FF F8                   MOVE.l -$8(a6),d0
060C52  02 80 55 55 55 55             ANDI.l #$55555555,d0
060C58  4E 5E                         UNLK a6
060C5A  4E 75                         RTS
060C5C  .dc.b 4E 56 FF EE 70 00 30 2E 00 12 72 0B 4E B9 00 06 ; NV..p.0...r.N...
060C6C  .dc.b 11 D0 42 AE FF F2 72 0A 2D 41 FF EE 2D 40 FF FC ; ..B...r.-A..-@..
060C7C  .dc.b 20 6E 00 08 30 28 00 06 48 C0 22 2E FF FC B2 80 ;  n..0(..H.".....
060C8C  .dc.b 66 06 4A 28 00 02 66 3C 2F 2E FF FC 2F 2E 00 08 ; f.J(..f</.../...
060C9C  .dc.b 61 00 F9 56 50 8F 2F 2E 00 08 61 00 FB 22 58 8F ; a..VP./...a.."X.
060CAC  .dc.b 2D 40 FF F2 4A 80 66 6C 2F 2E 00 08 61 00 FC B6 ; -@..J.fl/...a...
060CBC  .dc.b 58 8F 2D 40 FF F2 4A 80 66 5A 20 6E 00 08 11 7C ; X.-@..J.fZ n...|
060CCC  .dc.b 00 01 00 02 20 2E FF FC 72 0B 4E B9 00 06 12 48 ; .... ...r.N....H
060CDC  .dc.b 72 00 32 2E 00 12 92 80 3D 41 FF F6 02 81 00 00 ; r.2.....=A......
060CEC  .dc.b FF FF 2F 01 2F 2E 00 08 61 00 FE 06 50 8F 20 40 ; .././...a...P. @
060CFC  .dc.b 2D 48 FF F8 D0 FC 00 40 2F 08 2F 2E 00 0C 2F 3C ; -H.....@/./.../<
060D0C  .dc.b 00 00 02 00 61 00 00 64 4F EF 00 0C 20 2E FF F2 ; ....a..dO... ...
060D1C  .dc.b 4E 5E 4E 75 20 6E 00 08 42 28 00 02 20 2E FF EE ; N^Nu n..B(.. ...
060D2C  .dc.b 53 80 2D 40 FF EE 4A 80 6F E2 20 2E FF EE 02 80 ; S.-@..J.o. .....
060D3C  .dc.b 00 00 00 03 4A 80 66 00 FF 38 2F 2E 00 08 61 00 ; ....J.f..8/...a.
060D4C  .dc.b F8 12 58 8F 60 00 FF 2A                         ; ..X.`..*

; ==== sub_060D54 (5 callers) ====
060D54  4E 56 FF FC                   LINK a6,#-$4
060D58  20 3C 55 55 55 55             MOVE.l #$55555555,d0
060D5E  20 6E 00 08                   MOVEA.l $8(a6),a0
060D62  22 10                         MOVE.l (a0),d1
060D64  C2 80                         AND.l d0,d1
060D66  E3 81                         ASL.l #1,d1
060D68  24 28 00 04                   MOVE.l $4(a0),d2
060D6C  C4 80                         AND.l d0,d2
060D6E  82 82                         OR.l d2,d1
060D70  20 01                         MOVE.l d1,d0
060D72  4E 5E                         UNLK a6
060D74  4E 75                         RTS
060D76  .dc.b 4E 56 FF EC 2D 7C 55 55 55 55 FF EC 20 6E 00 10 ; NV..-|UUUU.. n..
060D86  .dc.b 20 08 22 2E 00 08 2D 40 FF FC D0 81 2D 6E 00 0C ;  ."...-@....-n..
060D96  .dc.b FF F4 2D 40 FF F8 2D 41 FF F0 0C AE 00 00 00 00 ; ..-@..-A........
060DA6  .dc.b FF F0 63 30 20 6E FF F4 58 AE FF F4 20 2E FF EC ; ..c0 n..X... ...
060DB6  .dc.b 22 6E FF FC 22 11 C2 80 58 AE FF FC E3 81 22 6E ; "n.."...X....."n
060DC6  .dc.b FF F8 24 11 C4 80 58 AE FF F8 82 82 20 81 59 AE ; ..$...X..... .Y.
060DD6  .dc.b FF F0 60 C6 4E 5E 4E 75 00 00                   ; ..`.N^Nu..

; --- load_descriptor  $060DE0 — the static source/destination descriptor passed to load_session. (data) ---
060DE0  .dc.b 74 72 61 63 6B 64 69 73 6B 2E 64 65 76 69 63 65 ; trackdisk.device
060DF0  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
060E00  .dc.b 00 00 00 00 00 00 00 00                         ; ........

; --- disk_params  $060E08 — 20-byte parameter/identity table copied into the IO struct by disk_hw_init (drive timing / unit data). (data) ---
060E08  .dc.b 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ; ................
060E18  .dc.b 00 00 00 00                                     ; ....

; ==== io_init  $060E1C  (1 caller) — initialise the $38 IO control struct (zero it / link the message port). ====
060E1C  4E 56 FF FC                   LINK a6,#-$4
060E20  70 00                         MOVEQ #$0,d0
060E22  2F 00                         MOVE.l d0,-(a7)
060E24  2F 00                         MOVE.l d0,-(a7)
060E26  4E B9 00 06 10 E0             JSR $610E0.l
060E2C  50 8F                         ADDQ.l #8,a7
060E2E  20 6E 00 08                   MOVEA.l $8(a6),a0
060E32  21 40 00 0E                   MOVE.l d0,$E(a0)
060E36  31 7C 00 38 00 12             MOVE.w #$38,$12(a0)
060E3C  4E 5E                         UNLK a6
060E3E  4E 75                         RTS

; ==== io_free  $060E40  (2 callers) — tear down / FreeMem the IO control struct. ====
060E40  4E 56 FF FC                   LINK a6,#-$4
060E44  22 6E 00 08                   MOVEA.l $8(a6),a1
060E48  20 69 00 0E                   MOVEA.l $E(a1),a0
060E4C  2F 08                         MOVE.l a0,-(a7)
060E4E  2D 48 FF FC                   MOVE.l a0,-$4(a6)
060E52  4E B9 00 06 11 88             JSR $61188.l
060E58  58 8F                         ADDQ.l #4,a7
060E5A  70 38                         MOVEQ #$38,d0
060E5C  2F 00                         MOVE.l d0,-(a7)
060E5E  2F 2E 00 08                   MOVE.l $8(a6),-(a7)
060E62  4E B9 00 06 10 1C             JSR $6101C.l
060E68  50 8F                         ADDQ.l #8,a7
060E6A  70 00                         MOVEQ #$0,d0
060E6C  4E 5E                         UNLK a6
060E6E  4E 75                         RTS

; ==== io_doio  $060E70  (1 caller) — drive a trackdisk-style request through the IO struct (motor/seek/read) — uses the exec DoIO stub. ====
060E70  4E 56 FF FC                   LINK a6,#-$4
060E74  20 6E 00 10                   MOVEA.l $10(a6),a0
060E78  C1 88                         EXG d0,a0
060E7A  02 40 FF FE                   ANDI.w #$FFFE,d0
060E7E  C1 88                         EXG d0,a0
060E80  22 6E 00 08                   MOVEA.l $8(a6),a1
060E84  23 48 00 28                   MOVE.l a0,$28(a1)
060E88  33 7C 00 02 00 1C             MOVE.w #$2,$1C(a1)
060E8E  70 09                         MOVEQ #$9,d0
060E90  22 2E 00 0C                   MOVE.l $C(a6),d1
060E94  E1 A1                         ASL.l d0,d1
060E96  23 41 00 2C                   MOVE.l d1,$2C(a1)
060E9A  22 2E 00 14                   MOVE.l $14(a6),d1
060E9E  E1 A1                         ASL.l d0,d1
060EA0  23 41 00 24                   MOVE.l d1,$24(a1)
060EA4  42 29 00 1E                   CLR.b $1E(a1)
060EA8  2F 09                         MOVE.l a1,-(a7)
060EAA  4E B9 00 06 10 B8             JSR $610B8.l
060EB0  58 8F                         ADDQ.l #4,a7
060EB2  4E 5E                         UNLK a6
060EB4  4E 75                         RTS
060EB6  .dc.b 4E 56 FF FC 20 6E 00 10 C1 88 02 40 FF FE C1 88 ; NV.. n.....@....
060EC6  .dc.b 22 6E 00 08 23 48 00 28 33 7C 00 03 00 1C 70 09 ; "n..#H.(3|....p.
060ED6  .dc.b 22 2E 00 0C E1 A1 23 41 00 2C 22 2E 00 14 E1 A1 ; ".....#A.,".....
060EE6  .dc.b 23 41 00 24 42 29 00 1E 2F 09 4E B9 00 06 10 B8 ; #A.$B)../.N.....
060EF6  .dc.b 58 8F 4E 5E 4E 75 4E 56 FF FC 20 6E 00 10 C1 88 ; X.N^NuNV.. n....
060F06  .dc.b 02 40 FF FE C1 88 22 6E 00 08 23 48 00 28 33 7C ; .@...."n..#H.(3|
060F16  .dc.b 00 0B 00 1C 70 09 22 2E 00 0C E1 A1 23 41 00 2C ; ....p.".....#A.,
060F26  .dc.b 22 2E 00 14 E1 A1 23 41 00 24 42 29 00 1E 20 2E ; ".....#A.$B).. .
060F36  .dc.b 00 14 72 0B 4E B9 00 06 12 04 4A 81 67 06 70 01 ; ..r.N.....J.g.p.
060F46  .dc.b 4E 5E 4E 75 20 2E 00 0C 72 0B 4E B9 00 06 12 04 ; N^Nu ...r.N.....
060F56  .dc.b 4A 81 67 06 70 02 4E 5E 4E 75 2F 2E 00 08 4E B9 ; J.g.p.N^Nu/...N.
060F66  .dc.b 00 06 10 B8 58 8F 4E 5E 4E 75 4E 56 FF FC 91 C8 ; ....X.N^NuNV....
060F76  .dc.b 22 6E 00 08 23 48 00 28 33 7C 00 04 00 1C 23 48 ; "n..#H.(3|....#H
060F86  .dc.b 00 2C 23 48 00 24 42 29 00 1E 2F 09 4E B9 00 06 ; .,#H.$B)../.N...
060F96  .dc.b 10 B8 58 8F 4E 5E 4E 75                         ; ..X.N^Nu

; ==== load_finish  $060F9E  (1 caller) — finish a load session (flush / stop the motor). ====
060F9E  4E 56 FF FC                   LINK a6,#-$4
060FA2  91 C8                         SUBA.l a0,a0
060FA4  22 6E 00 08                   MOVEA.l $8(a6),a1
060FA8  23 48 00 28                   MOVE.l a0,$28(a1)
060FAC  33 7C 00 09 00 1C             MOVE.w #$9,$1C(a1)
060FB2  23 48 00 2C                   MOVE.l a0,$2C(a1)
060FB6  23 48 00 24                   MOVE.l a0,$24(a1)
060FBA  42 29 00 1E                   CLR.b $1E(a1)
060FBE  2F 09                         MOVE.l a1,-(a7)
060FC0  4E B9 00 06 10 B8             JSR $610B8.l
060FC6  58 8F                         ADDQ.l #4,a7
060FC8  4E 5E                         UNLK a6
060FCA  4E 75                         RTS
060FCC  .dc.b 4E 56 FF FC 91 C8 22 6E 00 08 23 48 00 28 20 2E ; NV...."n..#H.( .
060FDC  .dc.b 00 0C 33 40 00 1C 23 48 00 2C 23 48 00 24 42 29 ; ..3@..#H.,#H.$B)
060FEC  .dc.b 00 1E 2F 09 4E B9 00 06 10 B8 58 8F 20 6E 00 08 ; ../.N.....X. n..
060FFC  .dc.b 20 28 00 20 4E 5E 4E 75                         ;  (. N^Nu

; ==== _AllocMem  $061004  (5 callers) — exec AllocMem(size, flags) (-$C6) ====
061004  2F 0E                         MOVE.l a6,-(a7)
061006  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
06100C  4C EF                         .dc.w $4CEF
06100E  .dc.b 00 03 00 08 4E AE FF 3A 2C 5F 4E 75 00 00       ; ....N..:,_Nu..

; ==== _FreeMem  $06101C  (5 callers) — exec FreeMem(addr, size) (-$D2) ====
06101C  2F 0E                         MOVE.l a6,-(a7)
06101E  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
061024  22 6F 00 08                   MOVEA.l $8(a7),a1
061028  20 2F 00 0C                   MOVE.l $C(a7),d0
06102C  4E AE FF 2E                   JSR -$D2(a6)
061030  2C 5F                         MOVEA.l (a7)+,a6
061032  4E 75                         RTS

; ==== _FindTask  $061034  (1 caller) — exec FindTask(name|0) (-$126) ====
061034  2F 0E                         MOVE.l a6,-(a7)
061036  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
06103C  22 6F 00 08                   MOVEA.l $8(a7),a1
061040  4E AE FE DA                   JSR -$126(a6)
061044  2C 5F                         MOVEA.l (a7)+,a6
061046  4E 75                         RTS

; ==== _AllocSignal  $061048  (1 caller) — exec AllocSignal(signum) (-$14A) — for the IO reply port ====
061048  2F 0E                         MOVE.l a6,-(a7)
06104A  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
061050  20 2F 00 08                   MOVE.l $8(a7),d0
061054  4E AE FE B6                   JSR -$14A(a6)
061058  2C 5F                         MOVEA.l (a7)+,a6
06105A  4E 75                         RTS

; ==== _FreeSignal  $06105C  (2 callers) — exec FreeSignal(signum) (-$150) ====
06105C  2F 0E                         MOVE.l a6,-(a7)
06105E  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
061064  20 2F 00 08                   MOVE.l $8(a7),d0
061068  4E AE FE B0                   JSR -$150(a6)
06106C  2C 5F                         MOVEA.l (a7)+,a6
06106E  4E 75                         RTS

; ==== _AddPort  $061070  (1 caller) — exec AddPort(port) (-$162) — public message port for the IORequest ====
061070  2F 0E                         MOVE.l a6,-(a7)
061072  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
061078  22 6F 00 08                   MOVEA.l $8(a7),a1
06107C  4E AE FE 9E                   JSR -$162(a6)
061080  2C 5F                         MOVEA.l (a7)+,a6
061082  4E 75                         RTS

; ==== _RemPort  $061084  (1 caller) — exec RemPort(port) (-$168) ====
061084  2F 0E                         MOVE.l a6,-(a7)
061086  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
06108C  22 6F 00 08                   MOVEA.l $8(a7),a1
061090  4E AE FE 98                   JSR -$168(a6)
061094  2C 5F                         MOVEA.l (a7)+,a6
061096  4E 75                         RTS

; ==== parse_request  $061098  (1 caller) — validate the load request out of the control block before allocating buffers. ====
061098  2F 0E                         MOVE.l a6,-(a7)
06109A  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
0610A0  20 6F 00 08                   MOVEA.l $8(a7),a0
0610A4  4C EF                         .dc.w $4CEF
0610A6  .dc.b 02 01 00 0C 22 2F 00 14 4E AE FE 44 2C 5F 4E 75 ; ...."/..N..D,_Nu
0610B6  .dc.b 00 00                                           ; ..

; ==== _DoIO  $0610B8  (2 callers) — exec DoIO(ioreq) (-$1C8) — synchronous device request ====
0610B8  2F 0E                         MOVE.l a6,-(a7)
0610BA  2C 79 00 06 00 64             MOVEA.l $60064.l,a6
0610C0  22 6F 00 08                   MOVEA.l $8(a7),a1
0610C4  4E AE FE 38                   JSR -$1C8(a6)
0610C8  2C 5F                         MOVEA.l (a7)+,a6
0610CA  4E 75                         RTS

; ==== sub_0610CC (1 caller) ====
0610CC  20 6F 00 04                   MOVEA.l $4(a7),a0
0610D0  20 88                         MOVE.l a0,(a0)
0610D2  58 90                         ADDQ.l #4,(a0)
0610D4  42 A8 00 04                   CLR.l $4(a0)
0610D8  21 48 00 08                   MOVE.l a0,$8(a0)
0610DC  4E 75                         RTS
0610DE  .dc.b 00 00                                           ; ..

; ==== sub_0610E0 (1 caller) ====
0610E0  48 E7 3F 20                   MOVEM.l d2-d7/a2,-(a7)
0610E4  28 2F 00 20                   MOVE.l $20(a7),d4
0610E8  16 2F 00 27                   MOVE.b $27(a7),d3
0610EC  2F 3C FF FF FF FF             MOVE.l #$FFFFFFFF,-(a7)
0610F2  4E B9 00 06 10 48             JSR $61048.l
0610F8  24 00                         MOVE.l d0,d2
0610FA  1C 02                         MOVE.b d2,d6
0610FC  1A 06                         MOVE.b d6,d5
0610FE  74 00                         MOVEQ #$0,d2
061100  14 06                         MOVE.b d6,d2
061102  0C 82 FF FF FF FF             CMPI.l #$FFFFFFFF,d2
061108  58 8F                         ADDQ.l #4,a7
06110A  66 06                         BNE $061112
06110C  70 00                         MOVEQ #$0,d0
06110E  60 00 00 72                   BRA $061182
061112  2F 3C 00 01 00 01             MOVE.l #$10001,-(a7)
061118  48 78 00 22                   PEA $22.w
06111C  4E B9 00 06 10 04             JSR $61004.l
061122  24 40                         MOVEA.l d0,a2
061124  CF 8A                         EXG d7,a2
061126  4A 87                         TST.l d7
061128  CF 8A                         EXG d7,a2
06112A  50 8F                         ADDQ.l #8,a7
06112C  66 12                         BNE $061140
06112E  74 00                         MOVEQ #$0,d2
061130  14 05                         MOVE.b d5,d2
061132  2F 02                         MOVE.l d2,-(a7)
061134  4E B9 00 06 10 5C             JSR $6105C.l
06113A  70 00                         MOVEQ #$0,d0
06113C  58 8F                         ADDQ.l #4,a7
06113E  60 42                         BRA $061182
061140  25 44 00 0A                   MOVE.l d4,$A(a2)
061144  15 43 00 09                   MOVE.b d3,$9(a2)
061148  15 7C 00 04 00 08             MOVE.b #$4,$8(a2)
06114E  42 2A 00 0E                   CLR.b $E(a2)
061152  15 45 00 0F                   MOVE.b d5,$F(a2)
061156  42 A7                         CLR.l -(a7)
061158  4E B9 00 06 10 34             JSR $61034.l
06115E  25 40 00 10                   MOVE.l d0,$10(a2)
061162  4A 84                         TST.l d4
061164  58 8F                         ADDQ.l #4,a7
061166  67 0C                         BEQ $061174
061168  2F 0A                         MOVE.l a2,-(a7)
06116A  4E B9 00 06 10 70             JSR $61070.l
061170  58 8F                         ADDQ.l #4,a7
061172  60 0C                         BRA $061180
061174  48 6A 00 14                   PEA $14(a2)
061178  4E B9 00 06 10 CC             JSR $610CC.l
06117E  58 8F                         ADDQ.l #4,a7
061180  20 0A                         MOVE.l a2,d0
061182  4C DF                         .dc.w $4CDF
061184  .dc.b 04 FC 4E 75                                     ; ..Nu

; ==== sub_061188 (1 caller) ====
061188  48 E7 20 20                   MOVEM.l d2/a2,-(a7)
06118C  24 6F 00 0C                   MOVEA.l $C(a7),a2
061190  4A AA 00 0A                   TST.l $A(a2)
061194  67 0A                         BEQ $0611A0
061196  2F 0A                         MOVE.l a2,-(a7)
061198  4E B9 00 06 10 84             JSR $61084.l
06119E  58 8F                         ADDQ.l #4,a7
0611A0  15 7C 00 FF 00 08             MOVE.b #$FF,$8(a2)
0611A6  74 FF                         MOVEQ #$FF,d2
0611A8  25 42 00 14                   MOVE.l d2,$14(a2)
0611AC  74 00                         MOVEQ #$0,d2
0611AE  14 2A 00 0F                   MOVE.b $F(a2),d2
0611B2  2F 02                         MOVE.l d2,-(a7)
0611B4  4E B9 00 06 10 5C             JSR $6105C.l
0611BA  48 78 00 22                   PEA $22.w
0611BE  2F 0A                         MOVE.l a2,-(a7)
0611C0  4E B9 00 06 10 1C             JSR $6101C.l
0611C6  4F EF 00 0C                   LEA $C(a7),a7
0611CA  4C DF                         .dc.w $4CDF
0611CC  .dc.b 04 04 4E 75 2F 02 2F 03 4A 81 67 22 4A 80 67 1C ; ..Nu/./.J.g"J.g.
0611DC  .dc.b 42 82 76 1F E3 80 E3 92 B4 81 65 08 94 81 D0 BC ; B.v.......e.....
0611EC  .dc.b 00 00 00 01 51 CB FF EE 22 02 60 04 42 81 42 80 ; ....Q...".`.B.B.
0611FC  .dc.b 26 1F 24 1F 4E 75 00 00                         ; &.$.Nu..

; ==== sub_061204 (3 callers) ====
061204  48 E7 3C 00                   MOVEM.l d2-d5,-(a7)
061208  2A 01                         MOVE.l d1,d5
06120A  67 32                         BEQ $06123E
06120C  6A 02                         BPL $061210
06120E  44 81                         NEG.l d1
061210  28 00                         MOVE.l d0,d4
061212  67 28                         BEQ $06123C
061214  6A 02                         BPL $061218
061216  44 80                         NEG.l d0
061218  42 82                         CLR.l d2
06121A  76 1F                         MOVEQ #$1F,d3
06121C  E3 80                         ASL.l #1,d0
06121E  E3 92                         ROXL.l #1,d2
061220  B4 81                         CMP.l d1,d2
061222  65 04                         BCS $061228
061224  94 81                         SUB.l d1,d2
061226  52 80                         ADDQ.l #1,d0
061228  51 CB FF F2                   DBRA d3,$06121C
06122C  22 02                         MOVE.l d2,d1
06122E  B9 85                         EOR.l d4,d5
061230  6A 02                         BPL $061234
061232  44 80                         NEG.l d0
061234  B3 84                         EOR.l d1,d4
061236  6A 08                         BPL $061240
061238  44 81                         NEG.l d1
06123A  60 04                         BRA $061240
06123C  42 81                         CLR.l d1
06123E  42 80                         CLR.l d0
061240  4C DF                         .dc.w $4CDF
061242  .dc.b 00 3C 4E 75 00 00                               ; .<Nu..

; ==== div_helper  $061248  (4 callers) — small arithmetic helper (track/offset split; d0/d1 in). ====
061248  48 E7 78 00                   MOVEM.l d1-d4,-(a7)
06124C  28 00                         MOVE.l d0,d4
06124E  B3 84                         EOR.l d1,d4
061250  4A 80                         TST.l d0
061252  67 30                         BEQ $061284
061254  6A 02                         BPL $061258
061256  44 80                         NEG.l d0
061258  24 00                         MOVE.l d0,d2
06125A  4A 81                         TST.l d1
06125C  66 04                         BNE $061262
06125E  42 80                         CLR.l d0
061260  60 22                         BRA $061284
061262  6A 02                         BPL $061266
061264  44 81                         NEG.l d1
061266  26 00                         MOVE.l d0,d3
061268  C6 C1                         MULU.W d1,d3
06126A  48 42                         SWAP d2
06126C  C4 C1                         MULU.W d1,d2
06126E  48 42                         SWAP d2
061270  42 42                         CLR.w d2
061272  D6 82                         ADD.l d2,d3
061274  48 41                         SWAP d1
061276  C0 C1                         MULU.W d1,d0
061278  48 40                         SWAP d0
06127A  42 40                         CLR.w d0
06127C  D0 83                         ADD.l d3,d0
06127E  4A 84                         TST.l d4
061280  6A 02                         BPL $061284
061282  44 80                         NEG.l d0
061284  4C DF                         .dc.w $4CDF
061286  .dc.b 00 1E 4E 75 00 00                               ; ..Nu..
