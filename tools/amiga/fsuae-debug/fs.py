# FS-UAE helper: robust cont-to-breakpoint that handles the %Stop:T05 notification.
import socket, re, sys
sys.path.insert(0,"/Users/dennis/Development/AIReverseEngineering/tools/amiga/fsuae-debug")
from dbg import Dbg
class FS(Dbg):
    def setreg(self, **kw):
        g=self.cmd("g"); v=[int(g[i:i+8],16) for i in range(0,18*8,8)]
        idx={'d0':0,'d1':1,'d2':2,'a4':12,'a5':13,'a6':14,'sp':15,'pc':17}
        for k,val in kw.items(): v[idx[k]]=val&0xFFFFFFFF
        self.cmd("G"+"".join("%08x"%x for x in v))
    def poke(self, addr, hexstr):
        self.cmd("M%x,%x:%s"%(addr,len(hexstr)//2,hexstr))
    def run_until(self, *addrs, timeout=15):
        for a in addrs: self.bp(a)
        self._send("c"); self.s.settimeout(timeout); buf=b""
        try:
            while b"T05" not in buf: buf+=self.s.recv(4096)
        except socket.timeout:
            for a in addrs: self.rmbp(a)
            return None
        self.s.sendall(b"+")
        try: self.cmd("vStopped")
        except Exception: pass
        self.drain(0.1)
        for a in addrs: self.rmbp(a)
        regs={int(x,16):int(y,16) for x,y in re.findall(r"([0-9a-f]{2}):([0-9a-f]{8})",buf.decode('latin1','replace'))}
        return regs.get(17)  # PC
