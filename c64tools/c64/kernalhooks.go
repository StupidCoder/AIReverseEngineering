package c64

// InstallKernalTapeHooks registers Go implementations of the handful of
// KERNAL ROM routines that tape loaders commonly call while ROM is banked
// out. They operate on the standard zero-page tape pointer ($AC/$AD) and end
// address ($AE/$AF).
//
//	$FCDB  increment the tape buffer pointer $AC/$AD
//	$FCD1  compare $AC/$AD with the end address $AE/$AF, returning carry
//	$FCCA  switch the cassette motor off (a no-op here)
//	$FF90  SETMSG: store A into the message/status flag $9D
func (m *Machine) InstallKernalTapeHooks() {
	m.SetHook(0xFCDB, func(m *Machine) bool {
		a := uint16(m.RAM[0xAC]) | uint16(m.RAM[0xAD])<<8
		a++
		m.RAM[0xAC] = byte(a)
		m.RAM[0xAD] = byte(a >> 8)
		m.RTS()
		return true
	})
	m.SetHook(0xFCD1, func(m *Machine) bool {
		cur := uint16(m.RAM[0xAC]) | uint16(m.RAM[0xAD])<<8
		end := uint16(m.RAM[0xAE]) | uint16(m.RAM[0xAF])<<8
		m.CPU.C = cur >= end
		m.CPU.Z = cur == end
		m.RTS()
		return true
	})
	m.SetHook(0xFCCA, func(m *Machine) bool { m.RTS(); return true })
	m.SetHook(0xFF90, func(m *Machine) bool {
		m.RAM[0x9D] = m.CPU.A
		m.RTS()
		return true
	})
}
