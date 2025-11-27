package ir

import (
	"fmt"
	"strings"
)

// PrettyPrint formats a Code (slice of Instr) into a human-readable string.
func PrettyPrint(code Code) string {
	var sb strings.Builder

	for i, instr := range code {
		sb.WriteString(fmt.Sprintf("%3d: %s", i, instr.Op))

		switch instr.Op {

		// Label
		case OpLabel:
			if instr.S != "" {
				sb.WriteString(fmt.Sprintf(" %s", instr.S))
			}

		// Calls (arity K)
		case OpCall:
			sb.WriteString(fmt.Sprintf(" %s(%d args)", instr.A, instr.K))
			if instr.D.Kind != Offset || instr.D.Value != 0 {
				// Destination only printed if meaningful
				sb.WriteString(fmt.Sprintf(" -> %s", instr.D))
			}

		case OpRetV:
			sb.WriteString(fmt.Sprintf(" %s", instr.A))

		case OpRet:
			// no operands, just return

		// Unary ops
		case OpNot, OpCopy:
			sb.WriteString(fmt.Sprintf(" %s -> %s", instr.A, instr.D))

		// Jumps
		case OpGoto:
			sb.WriteString(fmt.Sprintf(" %s", instr.S))

		case OpIfZ:
			sb.WriteString(fmt.Sprintf(" %s -> %s", instr.A, instr.S))

		// Binary ops (A, B -> D)
		default:
			// Many ops have 3 addresses D, A, B
			// Only print those that are meaningfully used
			hasA := instr.A.Kind == Literal || instr.A.Kind == Global || (instr.A.Kind == Offset)
			hasB := instr.B.Kind == Literal || instr.B.Kind == Global || (instr.B.Kind == Offset)
			hasD := instr.D.Kind == Literal || instr.D.Kind == Global || (instr.D.Kind == Offset)

			if hasD {
				sb.WriteString(fmt.Sprintf(" %s", instr.D))
			}
			if hasA {
				sb.WriteString(fmt.Sprintf(", %s", instr.A))
			}
			if hasB {
				sb.WriteString(fmt.Sprintf(", %s", instr.B))
			}
		}

		sb.WriteByte('\n')
	}

	return sb.String()
}
