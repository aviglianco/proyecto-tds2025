package ir

import (
	"fmt"
	"strings"
)

// PrettyPrint formats a Code (slice of Instr) into a human-readable string.
func PrettyPrint(code Code) string {
	var sb strings.Builder
	indent := ""
	needsBlankLineBefore := false

	isControlLabel := func(name string) bool {
		// Control-flow labels are auto-generated like L1, L2, ...
		return strings.HasPrefix(name, "L")
	}

	formatAddr := func(a Addr) string {
		return a.String()
	}

	for i, instr := range code {
		// Insert a blank line before function labels (not control labels),
		// except before the very first emitted line.
		if instr.Op == OpLabel && instr.S != "" && !isControlLabel(instr.S) {
			if needsBlankLineBefore {
				sb.WriteByte('\n')
			}
		}

		// Do not print numeric indices to keep an assembly-like look
		_ = i

		switch instr.Op {

		// Label
		case OpLabel:
			if instr.S != "" {
				// Print both function and control-flow labels uniformly
				sb.WriteString(fmt.Sprintf("%s:\n", instr.S))
				indent = "  "
				needsBlankLineBefore = true
				continue
			}
			// Anonymous label (unlikely)
			sb.WriteString(":\n")
			indent = "  "
			needsBlankLineBefore = true
			continue

		// Calls (arity K)
		case OpCall:
			callee := instr.S
			if callee == "" {
				callee = "<call>"
			}
			dest := formatAddr(instr.D)
			// OpCall name, argc -> D
			sb.WriteString(fmt.Sprintf("%sOpCall %s, %d -> %s", indent, callee, instr.K, dest))

		case OpRetV:
			// OpRetV A
			sb.WriteString(fmt.Sprintf("%sOpRetV %s", indent, formatAddr(instr.A)))

		case OpRet:
			// OpRet
			sb.WriteString(fmt.Sprintf("%sOpRet", indent))

		// Unary ops
		case OpNot, OpCopy:
			// OpNot D, A  |  OpCopy D, A
			sb.WriteString(fmt.Sprintf("%s%s %s, %s", indent, instr.Op.String(), formatAddr(instr.D), formatAddr(instr.A)))

		// Jumps
		case OpGoto:
			sb.WriteString(fmt.Sprintf("%sOpGoto %s", indent, instr.S))

		case OpIfZ:
			sb.WriteString(fmt.Sprintf("%sOpIfZ %s, %s", indent, formatAddr(instr.A), instr.S))

		case OpParam:
			// OpParam A
			sb.WriteString(fmt.Sprintf("%sOpParam %s", indent, formatAddr(instr.A)))

		// Binary ops (A, B -> D)
		default:
			// Ternary form: OpX D, A, B  (e.g., OpAdd D, A, B)
			sb.WriteString(fmt.Sprintf("%s%s %s, %s, %s", indent, instr.Op.String(), formatAddr(instr.D), formatAddr(instr.A), formatAddr(instr.B)))
		}

		sb.WriteByte('\n')
	}

	return sb.String()
}
