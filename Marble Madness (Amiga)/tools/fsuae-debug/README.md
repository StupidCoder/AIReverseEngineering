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

With this, `c` resumes the emulation thread, the chipset runs, and the floppy
boots through under debugger control. Verified: after the fix, `LoadSeg`'d
`HUNK_HEADER`/`HUNK_CODE` blocks appear in chip RAM and disk I/O proceeds.

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
