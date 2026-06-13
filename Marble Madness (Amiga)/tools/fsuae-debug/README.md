# Headless FS-UAE with GDB remote debug stub (macOS arm64)

A debugger-controllable Amiga emulator used to capture the *irreducible runtime
state* that the c/zzz copy-protection (`sub_DAA`) folds into its decryption
keystream — the CPU exception/TRAP vector page-bytes and the launcher task's
`tc_ExceptCode`/`tc_TrapCode` handler pages. These cannot be derived from the
disk image alone (they are set by booted AmigaDOS), so they are captured live
and fed to the Go reimplementation (`extract/cmd/runlauncher`) to complete the
decode. The decoder *algorithm* itself stays fully reimplemented in Go; the
emulator only supplies ~25 bytes of OS state and verifies the result.

Upstream: **prb28/fs-uae** — an FS-UAE fork (core derived from WinUAE 3300b2)
that adds a GDB remote-serial-protocol server (`src/remote_debug/`). Built at
commit `58feb4ccee1d0fa162ab25ef57a4529d433a5b3e`.

## Build (macOS arm64 / Apple Silicon)

```sh
brew install pkg-config glib sdl2 libpng gettext libmpeg2
git clone https://github.com/prb28/fs-uae /tmp/fs-uae
cd /tmp/fs-uae
git checkout 58feb4c
git apply "<repo>/Marble Madness (Amiga)/tools/fsuae-debug/fsuae-arm64.patch"

export PATH="/opt/homebrew/bin:$PATH"
export PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig"
export CPPFLAGS="-I/opt/homebrew/include"
export LDFLAGS="-L/opt/homebrew/lib"

./bootstrap            # autoreconf
./configure
# configure/Makefile emit a `-pagezero_size 0x2000` link flag that produces a
# malformed arm64 Mach-O (genblitter then SIGKILLs). Strip it:
sed -i '' 's/-pagezero_size 0x2000//g' Makefile config.status configure

make -j6 \
  CFLAGS="-g -O2 -Wno-narrowing -Wno-implicit-function-declaration" \
  CXXFLAGS="-g -O2 -Wno-c++11-narrowing -Wno-narrowing -Wno-implicit-function-declaration"
```

### What the arm64 port needed (see `fsuae-arm64.patch`)

| Symptom | Cause | Fix |
|---|---|---|
| `sysdeps.h: "unrecognized CPU type"` | no aarch64 branch | add `__aarch64__`/`_M_ARM64` case (`CPU_aarch64`, `CPU_64_BIT`) |
| genblitter killed; "Malformed Mach-o file" | `-pagezero_size 0x2000` invalid on arm64 | strip the flag (sed above) |
| `blkdev.cpp` C++11 narrowing errors | clang strict initializer-list narrowing | `-Wno-c++11-narrowing -Wno-narrowing` |
| `actions.c` implicit-decl error | `render.h` not included (symbol *does* exist) | `-Wno-implicit-function-declaration` |
| link: undefined `uae_ppc_wakeup_main()` | two calls sit outside the `#ifdef WITH_PPC` guard | guard them (patch) |

### The continue-freeze fix (the important one) — `remote_debug.cpp`

Out of the box, a bare GDB `c` (continue) **froze the whole emulated chipset**: the
video beam counter stopped, no interrupts fired, and any `STOP`-based OS idle loop
deadlocked. Root cause: the stub's `Tracing` service loop in `remote_debug_()`
spins on `sleep_millis(1)` and only breaks on `step_cpu` or quit — it never checks
`s_state`. `handle_continue_exec → remote_deactivate_debugger()` sets
`s_state = Running` but does **not** break that loop, so after `c` the CPU thread
stays trapped in C code, never calling `x_do_cycles()`. The 68000 `STOP` idle loop
(`newcpu.cpp` ~3870) only advances the chipset via `x_do_cycles()` *inside*
`while (SPCFLAG_STOP && !SPCFLAG_BRK)`, so with the CPU thread parked, the chipset
never ticks, no VBlank/CIA interrupt is generated, and `STOP` never wakes —
total deadlock. Single-step worked only because it sets `step_cpu`, which *does*
break the loop.

Fix: break the `Tracing` loop when a continue switches the state back to `Running`:

```c
if (s_state == Running) {   // continue requested -> resume the CPU/chipset
    break;
}
```

With this, `c` resumes the emulation thread and the chipset runs again
(verified: chip RAM changes across a continue).

### Two more stability fixes — `remote_debug.cpp`

**Speed.** `remote_debug_init_()` enables `debugmem_trace = true` and
`debugmem_enable_stackframe(true)` ("Full stack frame tracking enabled"), which
hook every call/return and slow emulation badly. Not needed for
memory/register/breakpoint capture — disabled (`false`/`false`) so emulation
runs at a usable speed.

**Crash on disconnect/shutdown (SIGSEGV).** `remote_debug_update_()` (called
every vsync) checks `s_conn` once at the top, then calls `remote_debug_()` —
which on the `is_quiting()` path does `rconn_destroy(s_conn); s_conn = 0;`.
Control returns and the function then calls `rconn_poll_read(s_conn)` →
`rconn_connected(NULL)` → dereferences `0x8` → `EXC_BAD_ACCESS`. This fired on
every emulator quit/disconnect (visible as macOS "fs-uae crashed" reports).
Fix: re-check `if (!s_conn) return;` after `remote_debug_()`. Verified: clean
connect/continue/disconnect now produces no crash report.

### Speed: enable the CPU-context debug hook — `remote_debug.cpp`

`remote_debug_init_()` left `debugging = 0`, so `do_specialties()` never called
`debug()` → `remote_debug()`; the whole debugger ran only off the per-vsync hook
(~50 Hz), so software breakpoints were only *sampled* 50×/sec and effectively
never hit. Setting `debugging = 1` enables the per-instruction CPU-context hook
so breakpoints are checked every instruction. (NB: a bp keeps `SPCFLAG_BRK` set,
so this also needs the SOCKET_POLL_INTERVAL throttle above to avoid a `select()`
per instruction.)

## What works / what doesn't (macOS arm64, this fork)

Verified working:
- Boots Kickstart 1.2 + the Marble Madness floppy to **Workbench** under the
  debugger (free-run; do not interrupt).
- Connect, halt (raw `0x03`), read/write registers and memory, single-step.
- **Software breakpoints fire during free-run** — arming `Z0,<addr>` while the
  CPU runs delivers an async `%Stop;swbreak` notification when the address
  executes. Reading state at that stop works. This is a usable one-shot capture.

Known open issue — **resume after a halt does not run the CPU**. After any halt
(manual `0x03` *or* a breakpoint hit), `c` (continue) does not advance the
emulation thread (verified: CPU runs at ~50 Hz before the first halt, then a
breakpoint stops it, and subsequent `c` produces zero further hits). Root cause
(via instrumentation): once `remote_debug_()` enters its Tracing spin the
emulation thread never re-enters it after the spin breaks — the thread doesn't
resume the m68k run loop. Not yet fixed. Consequence: **one breakpoint hit per
session, no chaining/stepping-through.** A one-shot capture (boot free-run → arm
one breakpoint at a known address → read state at the hit) is the usable mode;
multi-stage debugging needs this fixed.

The patch only touches `src/include/sysdeps.h` and `src/cpuboard.cpp`; the
narrowing/implicit-decl issues are handled by the `make` flags, and the
pagezero strip is a `sed` on generated files. The GDB stub is compiled in
unconditionally (`REMOTE_DEBUGGER` is `#define`d in `src/remote_debug.h`).

## Run (debug server on tcp/6860)

FS-UAE requires a real OpenGL context (`SDL_VIDEODRIVER=dummy` fails with
`[GLAD] Failed to initialize OpenGL context`), so run it with a real window on
a logged-in GUI session.

```sh
./fs-uae \
  --kickstart_file=/path/to/kick13.rom \
  --floppy_drive_0="/path/to/Marble_Madness.adf" \
  --amiga_model=A500 \
  --remote_debugger=10        `# wait up to 9s for a debugger to connect` \
  --remote_debugger_port=6860 \
  --fullscreen=0
```

`remote_debugger=N` waits `N-1` seconds at startup for a connection; the server
keeps listening afterwards. The ROM and ADF are copyrighted and not committed.

## Drive it (`rsp.py`)

Minimal GDB remote-serial-protocol client. Connects, interrupts (raw `0x03`),
reads registers (`g` → 8×D, 8×A, SR, PC as u32), reads memory (`m<addr>,<len>`
→ hex), sets breakpoints (`Z0,addr,kind`). Example session dumps the exception
vector table `$0..$BF`.

```sh
python3 rsp.py
```
