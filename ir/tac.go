package ir

type Op int

const (
	// Arithmetic operations
	OpAdd Op = iota
	OpSub
	OpMul
	OpDiv
	OpRem

	// Relational operations
	OpLT
	OpGT
	OpEQ

	// Unary operations
	OpNot
	OpCopy

	// Control flow
	OpIfZ
	OpGoto
	OpLabel

	// Calls and return
	OpParam
	OpCall
	OpRet
	OpRetV
)

type Instr struct {
	Op Op
	D  string // Destination
	A  string // Operand A
	B  string // Operand B
	S  string // Name
	K  int    // Arity (CALL)
}

type Block struct {
	Label  string
	Instrs []Instr
}

type Function struct {
	Name   string
	Params []string
	Locals []string
	Blocks []Block
	Extern bool
}

type Module struct {
	Functions []Function
}

// --- helpers for constants ---
func ConstInt(n int) string { return "#" + itoa(n) }

func ConstBool(b bool) string {
	if b {
		return "#1"
	}
	return "#0"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + (n % 10))
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
