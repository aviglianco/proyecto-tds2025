package ir

import "fmt"

type Op int

const (
	// Arithmetic operations
	OpAdd Op = iota
	OpSub
	OpMul
	OpDiv
	OpRem

	// Logical operations
	OpAnd
	OpOr

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

var opNames = []string{
	"OpAdd",
	"OpSub",
	"OpMul",
	"OpDiv",
	"OpRem",
	"OpAnd",
	"OpOr",
	"OpLT",
	"OpGT",
	"OpEQ",
	"OpNot",
	"OpCopy",
	"OpIfZ",
	"OpGoto",
	"OpLabel",
	"OpParam",
	"OpCall",
	"OpRet",
	"OpRetV",
}

func (o Op) String() string {
	if int(o) < len(opNames) {
		return opNames[o]
	}
	return fmt.Sprintf("Op(%d)", int(o))
}

func (op Op) isUnary() bool {
	return op == OpNot || op == OpCopy
}

type AddrKind uint

const (
	Literal = iota // prefijo #
	Global         // prefijo @
	Offset         // sin prefijo
)

type Addr struct {
	Kind  AddrKind
	Value int // literal or offset in frame or offset in data section
}

func (a Addr) String() string {
	switch a.Kind {
	case Literal:
		return "#" + fmt.Sprint(a.Value)
	case Global:
		return "@" + fmt.Sprint(a.Value)
	case Offset:
		return fmt.Sprint(a.Value)
	default:
		panic(fmt.Sprintf("invalid address kind %d", int(a.Kind)))
	}
}

type Instr struct {
	Op Op
	D  Addr // Destination
	A  Addr // Operand A
	B  Addr // Operand B
	S  Addr // Name
	K  int  // Arity (CALL)
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
