package ir

import "strings"

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
				continue}
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (i Instr) Format() string {
	switch i.Op {
	case OpAdd:
		return i.D + " = (" + i.A + " + " + i.B + ")"
	case OpSub:
		return i.D + " = (" + i.A + " - " + i.B + ")"
	case OpMul:
		return i.D + " = (" + i.A + " * " + i.B + ")"
	case OpDiv:
		return i.D + " = (" + i.A + " / " + i.B + ")"
	case OpRem:
		return i.D + " = (" + i.A + " % " + i.B + ")"

	case OpLT:
		return i.D + " = (" + i.A + " < " + i.B + ")"
	case OpGT:
		return i.D + " = (" + i.A + " > " + i.B + ")"
	case OpEQ:
		return i.D + " = (" + i.A + " == " + i.B + ")"

	case OpNot:
		return i.D + " = !" + i.A
	case OpCopy:
		return "copy " + i.D + ", " + i.A

	case OpIfZ:
		return "ifz " + i.A + " -> " + i.S
	case OpGoto:
		return "goto " + i.S
	case OpLabel:
		return i.S + ":"

	case OpParam:
		return "param " + i.A
	case OpCall:
		if i.D != "" {
			return i.D + " = call " + i.S + ", " + itoa(i.K)
		}
		return "call " + i.S + ", " + itoa(i.K)

	case OpRet:
		return "ret"
	case OpRetV:
		return "ret " + i.D
	default:
		return ""
	}
}
