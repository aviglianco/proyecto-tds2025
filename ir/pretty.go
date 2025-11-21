package ir

import (
	"fmt"
	"strings"
)

func (m *Module) String() string {
	var b strings.Builder
	for i, fn := range m.Functions {
		b.WriteString(fn.String())
		if i+1 < len(m.Functions) {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (f *Function) String() string {
	var b strings.Builder
	if f.Extern {
		b.WriteString("extern func ")
		b.WriteString(f.Name)
		b.WriteString("()\n")
		return b.String()
	}
	b.WriteString("func ")
	b.WriteString(f.Name)
	b.WriteString("():\n")

	for _, blk := range f.Blocks {
		if blk.Label != "" {
			b.WriteString(blk.Label)
			b.WriteString(":\n")
		}
		for _, ins := range blk.Instrs {
			line := ins.Format()
			if line == "" {
				continue
			}
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (i Instr) FormatUnary() string {

	switch i.Op {
	case OpNot:
		return i.D.String() + " = !" + i.A.String()
	case OpCopy:
		return "copy " + i.D.String() + ", " + i.A.String()
	default:
		panic(fmt.Sprintf("not a unary operation %s", i.Op.String()))
	}

}

func (i Instr) Format() string {

	if i.Op.isUnary() {
		return i.FormatUnary()
	}

	switch i.Op {
	case OpAdd:
		return i.D.String() + " = (" + i.A.String() + " + " + i.B.String() + ")"
	case OpSub:
		return i.D.String() + " = (" + i.A.String() + " - " + i.B.String() + ")"
	case OpMul:
		return i.D.String() + " = (" + i.A.String() + " * " + i.B.String() + ")"
	case OpDiv:
		return i.D.String() + " = (" + i.A.String() + " / " + i.B.String() + ")"
	case OpRem:
		return i.D.String() + " = (" + i.A.String() + " % " + i.B.String() + ")"

	case OpLT:
		return i.D.String() + " = (" + i.A.String() + " < " + i.B.String() + ")"
	case OpGT:
		return i.D.String() + " = (" + i.A.String() + " > " + i.B.String() + ")"
	case OpEQ:
		return i.D.String() + " = (" + i.A.String() + " == " + i.B.String() + ")"

	case OpIfZ:
		return "ifz " + i.A.String() + " -> " + i.S
	case OpGoto:
		return "goto " + i.S
	case OpLabel:
		return i.S + ":"

	case OpParam:
		return "param " + i.A.String()
	case OpCall:
		if i.D.String() != "" {
			return i.D.String() + " = call " + i.S + ", " + itoa(i.K)
		}
		return "call " + i.S + ", " + itoa(i.K)

	case OpRet:
		return "ret"
	case OpRetV:
		return "ret " + i.D.String()
	default:
		return ""
	}
}
