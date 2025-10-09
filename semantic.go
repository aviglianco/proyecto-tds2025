package main

import (
	"fmt"
)

// Analyzer performs semantic checks over the AST.
type Analyzer struct {
	env        Env
	errors     []error
	currentFun *FuncInfo
}

// errorf records a semantic error with line information when available.
func (an *Analyzer) errorf(n interface{}, format string, a ...interface{}) {
	line := 0
	switch node := n.(type) {
	case *Program:
		line = node.Line
	case *VarDecl:
		line = node.Line
	case *MethodDecl:
		line = node.Line
	case *Block:
		line = node.Line
	case *Assignment:
		line = node.Line
	case *ReturnStmt:
		line = node.Line
	case *IfStmt:
		line = node.Line
	case *WhileStmt:
		line = node.Line
	case *ExprStmt:
		line = node.Line
	case *IntLiteral:
		line = node.Line
	case *BoolLiteral:
		line = node.Line
	case *IdentExpr:
		line = node.Line
	case *UnaryExpr:
		line = node.Line
	case *BinaryExpr:
		line = node.Line
	case *CallExpr:
		line = node.Line
	case *ParenExpr:
		line = node.Line
	}
	if line > 0 {
		an.errors = append(an.errors, fmt.Errorf("line %d: "+format, append([]interface{}{line}, a...)...))
	} else {
		an.errors = append(an.errors, fmt.Errorf(format, a...))
	}
}

func Analyze(p *Program) error {
	// Start from the symbol table produced during building
	an := &Analyzer{env: p.Symbols}
	// Functions should already be registered in the builder's symbol table.
	// We still detect duplicates by scanning names in the top frame.
	seen := make(map[Identifier]struct{})
	for _, m := range p.Methods {
		if _, ok := seen[m.Name]; ok {
			an.errorf(m, "duplicate method declaration in same scope: %s", m.Name)
		}
		seen[m.Name] = struct{}{}
	}

	// Check there is a main method declared
	if _, ok := an.env.Table[Identifier("main")]; !ok {
		an.errorf(p, "program must declare a main method")
	}

	// Top-level variable declarations and their initializers are already in table from builder.
	// Re-validate duplicates vs current frame and check initializers.
	seenVars := make(map[Identifier]struct{})
	for _, d := range p.Declarations {
		if _, ok := seenVars[d.Name]; ok {
			an.errorf(d, "duplicate declaration in same scope: %s", d.Name)
		}
		seenVars[d.Name] = struct{}{}
		if d.Value != nil {
			_, _ = an.checkExpr(d.Value, false)
		}
	}

	// Analyze methods
	for _, m := range p.Methods {
		an.analyzeMethod(m)
	}

	if len(an.errors) > 0 {
		// Return first error for simplicity; could aggregate if desired
		return an.errors[0]
	}
	return nil
}

func (an *Analyzer) analyzeMethod(m *MethodDecl) {
	// New scope for parameters and locals
	prev := an.env
	an.env = Env{Prev: &prev, Table: make(Table)}
	an.currentFun = an.lookupFunc(string(m.Name))

	// Insert parameters into scope, checking duplicates
	for _, prm := range m.Params {
		if _, exists := an.env.Table[prm.Name]; exists {
			an.errors = append(an.errors, fmt.Errorf("duplicate parameter name: %s", prm.Name))
			continue
		}
		an.env.Insert(prm.Name, Symbol{Type: prm.Type, isVar: true})
	}

	if m.Body != nil {
		an.analyzeBlock(m.Body)
	}

	// restore
	an.env = prev
}

func (an *Analyzer) analyzeBlock(b *Block) {
	prev := an.env
	an.env = Env{Prev: &prev, Table: make(Table)}

	for _, d := range b.Declarations {
		if _, exists := an.env.Table[d.Name]; exists {
			an.errorf(d, "duplicate declaration in same scope: %s", d.Name)
		} else {
			an.env.Insert(d.Name, Symbol{Type: d.Type, isVar: true})
		}
		if d.Value != nil {
			t, ok := an.checkExpr(d.Value, false)
			if ok && t != d.Type {
				an.errorf(d, "initializer type mismatch for %s: expected %s, got %s", d.Name, d.Type.String(), t.String())
			}
		}
	}

	for _, s := range b.Stmts {
		switch st := s.(type) {
		case *Assignment:
			an.checkAssignment(st)
		case *ReturnStmt:
			an.checkReturn(st)
		case *IfStmt:
			an.checkIf(st)
		case *WhileStmt:
			an.checkWhile(st)
		case *ExprStmt:
			an.checkExprStmt(st)
		}
	}

	an.env = prev
}

func (an *Analyzer) checkAssignment(a *Assignment) {
	sym, ok := an.env.Lookup(a.Target)
	if !ok || !sym.isVar {
		an.errorf(a, "assignment to undeclared identifier: %s", a.Target)
		return
	}
	t, _ := an.checkExpr(a.Value, false)
	if t != sym.Type {
		an.errorf(a, "assignment type mismatch for %s: expected %s, got %s", a.Target, sym.Type.String(), t.String())
	}
}

func (an *Analyzer) checkReturn(r *ReturnStmt) {
	if an.currentFun == nil {
		return
	}
	if an.currentFun.Return == TypeVoid {
		if r.Value != nil {
			an.errorf(r, "void function should not return a value")
		}
		return
	}
	if r.Value == nil {
		an.errorf(r, "non-void function must return a value")
		return
	}
	t, _ := an.checkExpr(r.Value, false)
	if t != an.currentFun.Return {
		an.errorf(r, "return type mismatch: expected %s, got %s", an.currentFun.Return.String(), t.String())
	}
}

func (an *Analyzer) checkIf(i *IfStmt) {
	t, _ := an.checkExpr(i.Cond, false)
	if t != TypeBool {
		an.errorf(i, "if condition must be bool")
	}
	if i.Then != nil {
		an.analyzeBlock(i.Then)
	}
	if i.Else != nil {
		an.analyzeBlock(i.Else)
	}
}

func (an *Analyzer) checkWhile(w *WhileStmt) {
	t, _ := an.checkExpr(w.Cond, false)
	if t != TypeBool {
		an.errorf(w, "while condition must be bool")
	}
	an.analyzeBlock(w.Body)
}

func (an *Analyzer) checkExprStmt(e *ExprStmt) {
	// Allow void function calls in statement position
	_, _ = an.checkExpr(e.Expr, true)
}

// checkExpr returns (type, ok) where ok indicates whether the type could be inferred despite errors recorded
// TODO: tomar el tipo del arbol, no hacer switch
func (an *Analyzer) checkExpr(e Expr, allowVoidCall bool) (TypeKind, bool) {
	switch ex := e.(type) {
	case *IntLiteral:
		return TypeInteger, true
	case *BoolLiteral:
		return TypeBool, true
	case *IdentExpr:
		sym, ok := an.env.Lookup(ex.Name)
		if !ok {
			an.errorf(ex, "identifier used before declaration: %s", ex.Name)
			return 0, false
		}
		return sym.Type, true
	case *ParenExpr:
		return an.checkExpr(ex.Inner, allowVoidCall)
	case *CallExpr:
		return an.checkCallExpr(ex, allowVoidCall)
	case *UnaryExpr:
		return an.checkUnary(ex)
	case *BinaryExpr:
		return an.checkBinary(ex)
	default:
		an.errors = append(an.errors, fmt.Errorf("unknown expression node: %T", e))
	}
	return 0, false
}

func (an *Analyzer) checkCallExpr(c *CallExpr, allowVoidCall bool) (TypeKind, bool) {
	sym, ok := an.env.Lookup(c.Callee)
	if !ok || sym.Func == nil {
		an.errorf(c, "call to undeclared method: %s", c.Callee)
		return 0, false
	}
	fi := sym.Func
	if fi.Arity != len(fi.Params) { // internal consistency
		fi.Arity = len(fi.Params)
	}
	if len(c.Args) != fi.Arity {
		an.errorf(c, "wrong number of arguments in call to %s: expected %d, got %d", c.Callee, fi.Arity, len(c.Args))
	}
	// type-check args
	max := len(c.Args)
	if fi.Arity < max {
		max = fi.Arity
	}
	for i := 0; i < max; i++ {
		argT, _ := an.checkExpr(c.Args[i], false)
		if argT != fi.Params[i].Type {
			an.errorf(c, "argument %d type mismatch in call to %s: expected %s, got %s", i+1, c.Callee, fi.Params[i].Type.String(), argT.String())
		}
	}
	if fi.Return == TypeVoid && !allowVoidCall {
		an.errorf(c, "void method call used as expression")
	}
	return fi.Return, true
}

func (an *Analyzer) checkUnary(u *UnaryExpr) (TypeKind, bool) {
	// Determine by operator
	switch u.Op {
	case UnaryNeg:
		t, _ := an.checkExpr(u.Expr, false)
		if t != TypeInteger {
			an.errorf(u, "unary - requires integer operand")
		}
		return TypeInteger, true
	case UnaryNot:
		t, _ := an.checkExpr(u.Expr, false)
		if t != TypeBool {
			an.errorf(u, "! requires bool operand")
		}
		return TypeBool, true
	default:
		return 0, false
	}
}

func (an *Analyzer) checkBinary(b *BinaryExpr) (TypeKind, bool) {
	lt, _ := an.checkExpr(b.Left, false)
	rt, _ := an.checkExpr(b.Right, false)
	switch b.Op {
	case BinAdd, BinSub, BinMul, BinDiv:
		if lt != TypeInteger || rt != TypeInteger {
			an.errorf(b, "arithmetic operands must be integer")
		}
		return TypeInteger, true
	case BinLT, BinGT:
		if lt != TypeInteger || rt != TypeInteger {
			an.errorf(b, "relational operands must be integer")
		}
		return TypeBool, true
	case BinEq:
		if lt != rt {
			an.errorf(b, "== operands must be of the same type")
		}
		return TypeBool, true
	case BinAnd, BinOr:
		if lt != TypeBool || rt != TypeBool {
			an.errorf(b, "conditional operands must be bool")
		}
		return TypeBool, true
	default:
		return 0, false
	}
}

func (an *Analyzer) lookupFunc(name string) *FuncInfo {
	sym, ok := an.env.Lookup(Identifier(name))
	if !ok || sym.Func == nil {
		return nil
	}
	return sym.Func
}
