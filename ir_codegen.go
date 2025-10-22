package main

import (
	"fmt"

	ir "compilador/ir"
)

func GenerateIR(p *Program) (*ir.Module, error) {
	if p == nil {
		return nil, fmt.Errorf("codegen: nil Program")
	}
	g := &gen{mod: &ir.Module{}}
	g.initGlobals()
	g.emitProgram(p)

	if g.buildInitFunction(p) {
		prependInitCallToMain(g.mod)
	}

	injectImplicitRetVoid(g.mod, p)

	return g.mod, nil
}

// ----------------------------------------------------------------------------
// Internal generator state
// ----------------------------------------------------------------------------

type gen struct {
	mod     *ir.Module
	fun     *ir.Function
	blk     *ir.Block
	globals map[Identifier]bool

	tcnt int
	lcnt int

	env []map[Identifier]string
}

func (g *gen) initGlobals() { g.globals = make(map[Identifier]bool) }

// ----------------------------------------------------------------------------
// Helpers: temporaries & labels
// ----------------------------------------------------------------------------

// newTemp returns a fresh temporary name (e.g., %t0, %t1, ...).
func (g *gen) newTemp() string {
	name := fmt.Sprintf("%%t%d", g.tcnt)
	g.tcnt++
	return name
}

// newLabel returns a fresh label name with an optional human-readable prefix.
func (g *gen) newLabel(prefix string) string {
	if prefix == "" {
		prefix = "L"
	}
	name := fmt.Sprintf(".%s%d", prefix, g.lcnt)
	g.lcnt++
	return name
}

// ----------------------------------------------------------------------------
// Helpers: block & instruction plumbing
// ----------------------------------------------------------------------------

// startBlock finalizes the current block (if any) and starts a new one.
func (g *gen) startBlock(label string) {
	if g.fun == nil {
		panic("startBlock called with no current function")
	}
	b := &ir.Block{Label: label}
	g.fun.Blocks = append(g.fun.Blocks, *b)
	// Keep blk pointing to the last item in the slice
	g.blk = &g.fun.Blocks[len(g.fun.Blocks)-1]
}

// emit appends an instruction to the current block.
func (g *gen) emit(i ir.Instr) {
	if g.blk == nil {
		panic("emit called with no current block")
	}
	g.blk.Instrs = append(g.blk.Instrs, i)
}

// ----------------------------------------------------------------------------
// Helpers: lexical scopes
// ----------------------------------------------------------------------------

func (g *gen) pushScope() { g.env = append(g.env, make(map[Identifier]string)) }

func (g *gen) popScope() {
	if len(g.env) == 0 {
		panic("popScope on empty env stack")
	}
	g.env = g.env[:len(g.env)-1]
}

// bind associates an identifier with an IR location in the current scope.
func (g *gen) bind(id Identifier, irName string) {
	if len(g.env) == 0 {
		g.pushScope()
	}
	g.env[len(g.env)-1][id] = irName
}

// lookup finds the nearest IR name bound to an identifier.
func (g *gen) lookup(id Identifier) (string, bool) {
	for i := len(g.env) - 1; i >= 0; i-- {
		if v, ok := g.env[i][id]; ok {
			return v, true
		}
	}
	return "", false
}

// ----------------------------------------------------------------------------

// emitBlock walks a block, allocating locals, binding them, and emitting
// initializers and statements in sequence.
func (g *gen) emitBlock(b *Block) {
	if b == nil {
		return
	}
	g.pushScope()
	for _, d := range b.Declarations {
		if g.fun != nil {
			g.fun.Locals = append(g.fun.Locals, string(d.Name))
		}
		g.bind(d.Name, string(d.Name))
		if d.Value != nil {
			val := g.emitExpr(d.Value)
			g.emit(ir.Instr{Op: ir.OpCopy, D: string(d.Name), A: val})
		}
	}
	for _, s := range b.Stmts {
		g.emitStmt(s)
	}
	g.popScope()
}

func (g *gen) emitStmt(s Stmt) {
	switch st := s.(type) {
	case *Assignment:
		val := g.emitExpr(st.Value)
		name, ok := g.lookup(st.Target)
		if !ok {
			name = string(st.Target)
		}
		g.emit(ir.Instr{Op: ir.OpCopy, D: name, A: val})
	case *ReturnStmt:
		if st.Value == nil {
			g.emit(ir.Instr{Op: ir.OpRet})
			return
		}
		v := g.emitExpr(st.Value)
		g.emit(ir.Instr{Op: ir.OpRetV, D: v})
	case *ExprStmt:
		_ = g.emitExpr(st.Expr)
	case *Block:
		g.emitBlock(st)
	case *IfStmt:
		cond := g.emitExpr(st.Cond)
		var labelElse string
		labelEnd := g.newLabel("if_end")
		if st.Else != nil {
			labelElse = g.newLabel("if_else")
			g.emit(ir.Instr{Op: ir.OpIfZ, A: cond, S: labelElse})
			g.emitBlock(st.Then)
			g.emit(ir.Instr{Op: ir.OpGoto, S: labelEnd})
			g.startBlock(labelElse)
			g.emitBlock(st.Else)
		} else {
			g.emit(ir.Instr{Op: ir.OpIfZ, A: cond, S: labelEnd})
			g.emitBlock(st.Then)
		}
		g.startBlock(labelEnd)
	case *WhileStmt:
		lCond := g.newLabel("while_cond")
		lBody := g.newLabel("while_body")
		lEnd := g.newLabel("while_end")
		g.emit(ir.Instr{Op: ir.OpGoto, S: lCond})
		g.startBlock(lBody)
		g.emitBlock(st.Body)
		g.emit(ir.Instr{Op: ir.OpGoto, S: lCond})
		g.startBlock(lCond)
		c := g.emitExpr(st.Cond)
		g.emit(ir.Instr{Op: ir.OpIfZ, A: c, S: lEnd})
		g.emit(ir.Instr{Op: ir.OpGoto, S: lBody})
		g.startBlock(lEnd)
	default:
		panic("unhandled statement kind")
	}
}

// emitExpr lowers an expression and returns the IR name/constant holding its value.
func (g *gen) emitExpr(e Expr) string {
	switch ex := e.(type) {
	case *IntLiteral:
		return ir.ConstInt(ex.Value)
	case *BoolLiteral:
		return ir.ConstBool(ex.Value)
	case *IdentExpr:
		if n, ok := g.lookup(ex.Name); ok {
			return n
		}
		return string(ex.Name)
	case *ParenExpr:
		return g.emitExpr(ex.Inner)
	case *CallExpr:
		for _, a := range ex.Args {
			v := g.emitExpr(a)
			g.emit(ir.Instr{Op: ir.OpParam, A: v})
		}
		if ex.Type != TypeVoid {
			t := g.newTemp()
			g.emit(ir.Instr{Op: ir.OpCall, D: t, S: string(ex.Callee), K: len(ex.Args)})
			return t
		}
		g.emit(ir.Instr{Op: ir.OpCall, S: string(ex.Callee), K: len(ex.Args)})
		return ""
	case *UnaryExpr:
		v := g.emitExpr(ex.Expr)
		switch ex.Op {
		case UnaryNeg:
			// -(x) == 0 - x
			t := g.newTemp()
			g.emit(ir.Instr{Op: ir.OpSub, D: t, A: ir.ConstInt(0), B: v})
			return t
		case UnaryNot:
			t := g.newTemp()
			g.emit(ir.Instr{Op: ir.OpNot, D: t, A: v})
			return t
		default:
			return v
		}
	case *BinaryExpr:
		left := g.emitExpr(ex.Left)
		right := g.emitExpr(ex.Right)
		t := g.newTemp()
		switch ex.Op {
		case BinAdd:
			g.emit(ir.Instr{Op: ir.OpAdd, D: t, A: left, B: right})
		case BinSub:
			g.emit(ir.Instr{Op: ir.OpSub, D: t, A: left, B: right})
		case BinMul:
			g.emit(ir.Instr{Op: ir.OpMul, D: t, A: left, B: right})
		case BinDiv:
			g.emit(ir.Instr{Op: ir.OpDiv, D: t, A: left, B: right})
		case BinRem:
			g.emit(ir.Instr{Op: ir.OpRem, D: t, A: left, B: right})
		case BinLT:
			g.emit(ir.Instr{Op: ir.OpLT, D: t, A: left, B: right})
		case BinGT:
			g.emit(ir.Instr{Op: ir.OpGT, D: t, A: left, B: right})
		case BinEq:
			g.emit(ir.Instr{Op: ir.OpEQ, D: t, A: left, B: right})
		case BinAnd:
			// Short-circuit AND using CFG
			// t = (left && right)
			lFalse := g.newLabel("and_false")
			lEnd := g.newLabel("and_end")
			// evaluate left
			a := left
			// ifz a -> false
			g.emit(ir.Instr{Op: ir.OpIfZ, A: a, S: lFalse})
			// evaluate right
			b := g.emitExpr(ex.Right)
			g.emit(ir.Instr{Op: ir.OpCopy, D: t, A: b})
			g.emit(ir.Instr{Op: ir.OpGoto, S: lEnd})
			// false path
			g.startBlock(lFalse)
			g.emit(ir.Instr{Op: ir.OpCopy, D: t, A: ir.ConstBool(false)})
			// end
			g.startBlock(lEnd)
		case BinOr:
			// Short-circuit OR using CFG
			// t = (left || right)
			lAfter := g.newLabel("or_after")
			lEnd := g.newLabel("or_end")
			// evaluate left
			a := left
			// ifz a -> after (evaluate right); else set true
			g.emit(ir.Instr{Op: ir.OpIfZ, A: a, S: lAfter})
			g.emit(ir.Instr{Op: ir.OpCopy, D: t, A: ir.ConstBool(true)})
			g.emit(ir.Instr{Op: ir.OpGoto, S: lEnd})
			// after: evaluate right
			g.startBlock(lAfter)
			b := g.emitExpr(ex.Right)
			g.emit(ir.Instr{Op: ir.OpCopy, D: t, A: b})
			// end
			g.startBlock(lEnd)
		}
		return t
	default:
		return g.newTemp()
	}
}

func (g *gen) emitProgram(p *Program) {
	if p == nil {
		return
	}

	for _, d := range p.Declarations {
		g.globals[d.Name] = true
	}

	for _, m := range p.Methods {
		g.emitMethod(m)
	}
}

func (g *gen) emitMethod(m *MethodDecl) {
	if m == nil {
		return
	}
	if m.Extern || m.Body == nil {
		fn := ir.Function{Name: string(m.Name), Extern: true}
		g.mod.Functions = append(g.mod.Functions, fn)
		return
	}

	fn := ir.Function{Name: string(m.Name)}
	for _, prm := range m.Params {
		fn.Params = append(fn.Params, string(prm.Name))
	}
	g.mod.Functions = append(g.mod.Functions, fn)
	g.fun = &g.mod.Functions[len(g.mod.Functions)-1]

	// Reset per-function state
	g.tcnt, g.lcnt = 0, 0
	g.env = nil
	g.pushScope()
	for _, prm := range m.Params {
		g.bind(prm.Name, string(prm.Name))
	}

	// Entry block and body
	g.startBlock("entry")
	g.emitBlock(m.Body)
}

func hasTerminator(b *ir.Block) bool {
	if b == nil || len(b.Instrs) == 0 {
		return false
	}
	last := b.Instrs[len(b.Instrs)-1]
	switch last.Op {
	case ir.OpRet, ir.OpRetV, ir.OpGoto, ir.OpIfZ:
		return true
	default:
		return false
	}
}

func (g *gen) buildInitFunction(p *Program) bool {
	hasInits := false
	for _, d := range p.Declarations {
		if d != nil && d.Value != nil {
			hasInits = true
			break
		}
	}
	if !hasInits {
		return false
	}

	initFn := ir.Function{Name: "__init"}
	g.mod.Functions = append(g.mod.Functions, initFn)
	g.fun = &g.mod.Functions[len(g.mod.Functions)-1]
	g.tcnt, g.lcnt = 0, 0
	g.startBlock("entry")

	for _, d := range p.Declarations {
		if d == nil || d.Value == nil {
			continue
		}
		val := g.emitExpr(d.Value)
		g.emit(ir.Instr{Op: ir.OpCopy, D: string(d.Name), A: val})
	}
	g.emit(ir.Instr{Op: ir.OpRet})
	g.fun = nil
	return true
}

func prependInitCallToMain(m *ir.Module) {
	for i := range m.Functions {
		fn := &m.Functions[i]
		if fn.Extern || fn.Name != "main" || len(fn.Blocks) == 0 {
			continue
		}
		b := &fn.Blocks[0]
		call := ir.Instr{Op: ir.OpCall, S: "__init", K: 0}
		b.Instrs = append([]ir.Instr{call}, b.Instrs...)
		break
	}
}

func injectImplicitRetVoid(m *ir.Module, p *Program) {
	retByName := map[string]TypeKind{}
	for _, md := range p.Methods {
		if md != nil {
			retByName[string(md.Name)] = md.Return
		}
	}
	for i := range m.Functions {
		fn := &m.Functions[i]
		if fn.Extern {
			continue
		}
		if retByName[fn.Name] != TypeVoid {
			continue
		}
		if len(fn.Blocks) == 0 {
			fn.Blocks = append(fn.Blocks, ir.Block{Label: "entry"})
		}
		last := &fn.Blocks[len(fn.Blocks)-1]
		if !hasTerminator(last) {
			last.Instrs = append(last.Instrs, ir.Instr{Op: ir.OpRet})
		}
	}
}
