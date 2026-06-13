#!/usr/bin/env python3
"""Minimal GDB remote-serial-protocol client for the FS-UAE m68k debug stub."""
import socket, sys, time

class RSP:
    def __init__(self, host="127.0.0.1", port=6860):
        self.s = socket.create_connection((host, port), timeout=10)
        self.s.settimeout(10)
        self.buf = b""

    def _recvbyte(self):
        while not self.buf:
            d = self.s.recv(4096)
            if not d:
                raise EOFError("connection closed")
            self.buf += d
        b, self.buf = self.buf[:1], self.buf[1:]
        return b

    def send(self, data):
        cks = sum(data.encode()) & 0xFF
        pkt = b"$" + data.encode() + b"#" + ("%02x" % cks).encode()
        self.s.sendall(pkt)
        # wait for ack '+'
        while True:
            b = self._recvbyte()
            if b == b"+":
                return
            if b == b"-":
                self.s.sendall(pkt)  # resend

    def recv(self):
        # skip until '$'
        while True:
            b = self._recvbyte()
            if b == b"$":
                break
        data = b""
        while True:
            b = self._recvbyte()
            if b == b"#":
                break
            data += b
        c1 = self._recvbyte(); c2 = self._recvbyte()
        self.s.sendall(b"+")
        return data.decode(errors="replace")

    def cmd(self, data):
        self.send(data)
        return self.recv()

    def cont(self):
        # 'c' replies OK immediately, then the CPU runs until a later stop.
        self.send("c")
        return self.recv()   # consume the OK reply so the stream stays in sync

    def interrupt(self):
        self.s.sendall(b"\x03")
        return self.recv()

def hexmem(r):
    return bytes(int(r[i:i+2],16) for i in range(0,len(r),2))

if __name__ == "__main__":
    c = RSP()
    print("connected")
    # try to halt
    try:
        r = c.interrupt()
        print("interrupt ->", r)
    except Exception as e:
        print("interrupt failed:", e)
    print("? ->", c.cmd("?"))
    g = c.cmd("g")
    print("g ->", g)
    if len(g) >= 18*8:
        vals = [int(g[i:i+8],16) for i in range(0,18*8,8)]
        for i in range(8): print("  D%d=%08X" % (i, vals[i]))
        for i in range(8): print("  A%d=%08X" % (i, vals[8+i]))
        print("  SR=%08X PC=%08X" % (vals[16], vals[17]))
    # read exception vectors $0-$BF
    m = c.cmd("m0,c0")
    print("mem $0..$BF:", m)
