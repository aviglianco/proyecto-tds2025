package main

import (
	ir "compilador/ir"
	"fmt"
	"slices"
	"strconv"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Builder struct {
	symbolTable     Env
	src             []byte
	currentOffset   int
	currentLabelNum int
}

// builderErrorf creates an error message annotated with the line and column
func builderErrorf(n *sitter.Node, format string, a ...any) error {
	if n != nil {
		return fmt.Errorf("line %d, col %d: "+format, append([]any{nodeLine(n), nodeCol(n)}, a...)...)
	}
	return fmt.Errorf(format, a...)
}

// BuildAST takes a CST node (root of a parsed source file) and returns our AST.
func BuildAST(root *sitter.Node, src []byte) (*Program, error) {
	if root.Kind() != "source_file" {
		return nil, builderErrorf(root, "expected root to be source_file, got %s", root.Kind())
	}

	// source_file -> program
	if root.ChildCount() == 0 {
		return nil, builderErrorf(root, "empty source file")
	}

	symbolTable := Env{Table: make(map[Identifier]Symbol)}

	builder := Builder{
		src:         src,
		symbolTable: symbolTable,
	}

	return builder.buildProgram(root.Child(0))
}

// ----------------------------------------------------------------------
// Helpers for extracting fields
// ----------------------------------------------------------------------

func text(node *sitter.Node, src []byte) string {
	if node == nil {
		return ""
	}
	return string(src[node.StartByte():node.EndByte()])
}

// nodeLine returns the 1-based start line for a CST node.
func nodeLine(n *sitter.Node) int {
	if n == nil {
		return 0
	}
	return int(n.Range().StartPoint.Row) + 1
}

// nodeCol returns the 1-based start column for a CST node.
func nodeCol(n *sitter.Node) int {
	if n == nil {
		return 0
	}
	return int(n.Range().StartPoint.Column) + 1
}

// ----------------------------------------------------------------------
// Builders
// ----------------------------------------------------------------------

func (builder *Builder) buildProgram(n *sitter.Node) (*Program, error) {
	if n.Kind() != "program" {
		return nil, builderErrorf(n, "expected program node, got %s", n.Kind())
	}

	p := &Program{NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)}}
	var ir_code ir.Code

	for i := uint(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c == nil {
			continue
		}
		switch c.Kind() {
		case "declaration_statement":
			decl, err := builder.buildGlobalVarDecl(c)
			if err != nil {
				return nil, err
			}
			p.Declarations = append(p.Declarations, decl)
			ir_code = slices.Concat(ir_code, decl.getCode())
		case "method_declaration_statement":
			m, err := builder.buildMethodDecl(c)
			if err != nil {
				return nil, err
			}
			p.Methods = append(p.Methods, m)
			// ir_code = slices.Concat(ir_code, m.getCode())
		}
	}

	// expose built symbol table at program level
	p.Symbols = builder.symbolTable
	return p, nil
}

func (builder *Builder) getNewOffset() int {
	builder.currentOffset += 1
	return builder.currentOffset
}

func (builder *Builder) getNewLabel() string {
	builder.currentLabelNum += 1
	return "L" + strconv.Itoa(builder.currentLabelNum)
}

func (builder *Builder) resetOffset() {
	builder.currentOffset = 0
}

func (builder *Builder) buildLocalVarDecl(n *sitter.Node) (*VarDecl, error) {
	typNode := n.ChildByFieldName("type")
	idNode := n.ChildByFieldName("identifier")
	valNode := n.ChildByFieldName("value")

	t, err := builder.buildType(typNode)
	if err != nil {
		return nil, err
	}
	name := Identifier(text(idNode, builder.src))
	val, err := builder.buildExpr(valNode)

	_, ok := builder.symbolTable.Table[name]
	if ok {
		return nil, builderErrorf(n, "cannot double declare :%s", name)
	} else {
		builder.symbolTable.Insert(
			name,
			Symbol{
				Type:    t,
				VarKind: LocalVar,
				Address: ir.Addr{
					Kind:  ir.Offset,
					Value: builder.getNewOffset(),
				},
			},
		)
	}

	if err != nil {
		return nil, err
	}
	return &VarDecl{NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)}, Type: t, Name: name, Value: val}, nil
}

func (builder *Builder) buildGlobalVarDecl(n *sitter.Node) (*VarDecl, error) {
	typNode := n.ChildByFieldName("type")
	idNode := n.ChildByFieldName("identifier")
	valNode := n.ChildByFieldName("value")

	t, err := builder.buildType(typNode)
	if err != nil {
		return nil, err
	}
	name := Identifier(text(idNode, builder.src))
	val, err := builder.buildExpr(valNode)

	_, ok := builder.symbolTable.Table[name]
	if ok {
		return nil, builderErrorf(n, "cannot double declare :%s", name)
	} else {
		builder.symbolTable.Insert(
			name,
			Symbol{
				Type:    t,
				VarKind: GlobalVar,
				Address: ir.Addr{
					Kind:  ir.Global,
					Value: builder.getNewOffset(),
				},
			},
		)
	}

	if err != nil {
		return nil, err
	}
	return &VarDecl{NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)}, Type: t, Name: name, Value: val}, nil
}

func (builder *Builder) buildType(n *sitter.Node) (TypeKind, error) {
	if n == nil {
		return 0, builderErrorf(n, "nil type node")
	}
	switch n.Kind() {
	case "void":
		return TypeVoid, nil
	case "bool":
		return TypeBool, nil
	case "integer":
		return TypeInteger, nil
	default:
		return 0, builderErrorf(n, "unknown type node: %s", n.Kind())
	}
}

func (builder *Builder) buildMethodDecl(n *sitter.Node) (*MethodDecl, error) {
	retNode := n.ChildByFieldName("type")
	idNode := n.ChildByFieldName("identifier")

	t, err := builder.buildType(retNode)
	if err != nil {
		return nil, err
	}
	name := Identifier(text(idNode, builder.src))

	_, ok := builder.symbolTable.Table[name]
	if ok {
		return nil, builderErrorf(n, "cannot redefine:%s", name)
	}

	// parameters
	var params []*Parameter
	for i := uint(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c.Kind() == "parameter" {
			p, err := builder.buildParameter(c)
			if err != nil {
				return nil, err
			}
			params = append(params, p)
		}
	}

	paramInfos := make([]ParamInfo, 0, len(params))
	for _, p := range params {
		paramInfos = append(paramInfos, ParamInfo{Name: p.Name, Type: p.Type})
	}
	builder.symbolTable.Insert(name, Symbol{Type: t, VarKind: Method, Func: &FuncInfo{Return: t, Params: paramInfos, Arity: len(paramInfos), DeclLine: nodeLine(n)}})
	prevEnv := builder.symbolTable

	if len(params) > 0 {
		paramNames := make(map[Identifier]struct{})
		for _, p := range params {
			if _, clash := paramNames[p.Name]; clash {
				return nil, builderErrorf(n, "duplicate parameter name: %s", p.Name)
			}
			paramNames[p.Name] = struct{}{}
		}

		funcEnv := Env{Prev: &prevEnv, Table: make(Table)}
		for i, p := range params {
			funcEnv.Insert(
				p.Name,
				Symbol{Type: p.Type, VarKind: LocalVar,
					Address: ir.Addr{
						Kind:  ir.Offset,
						Value: -(i + 1),
					},
				})
		}
		builder.symbolTable = funcEnv
	}

	// extern or block
	var body *Block
	extern := false
	for i := uint(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c.Kind() == "extern" {
			extern = true
		}
		if c.Kind() == "block" {
			b, err := builder.buildBlock(c)
			if err != nil {
				return nil, err
			}
			body = b
		}
	}

	builder.symbolTable = prevEnv

	builder.resetOffset()

	return &MethodDecl{
		NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)},
		Return:   t,
		Name:     name,
		Params:   params,
		Body:     body,
		Extern:   extern,
	}, nil
}

func (builder *Builder) buildParameter(n *sitter.Node) (*Parameter, error) {
	tNode := n.ChildByFieldName("type")
	idNode := n.ChildByFieldName("identifier")

	t, err := builder.buildType(tNode)
	if err != nil {
		return nil, err
	}
	return &Parameter{NodeBase: NodeBase{Line: nodeLine(n)}, Type: t, Name: Identifier(text(idNode, builder.src))}, nil
}

// ----------------------------------------------------------------------
// Blocks & Statements
// ----------------------------------------------------------------------

func (builder *Builder) buildBlock(n *sitter.Node) (*Block, error) {
	b := &Block{NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)}}
	prevEnv := builder.symbolTable
	builder.symbolTable = Env{Prev: &prevEnv, Table: make(Table)}

	for i := uint(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		switch c.Kind() {
		case "declaration_statement":
			d, err := builder.buildLocalVarDecl(c)
			if err != nil {
				return nil, err
			}
			b.Declarations = append(b.Declarations, d)
		case "assignment_statement":
			as, err := builder.buildAssignment(c)
			if err != nil {
				return nil, err
			}
			b.Stmts = append(b.Stmts, as)
		case "return_statement":
			rs, err := builder.buildReturnStmt(c)
			if err != nil {
				return nil, err
			}
			b.Stmts = append(b.Stmts, rs)
		case "if_statement":
			is, err := builder.buildIfStmt(c)
			if err != nil {
				return nil, err
			}
			b.Stmts = append(b.Stmts, is)
		case "while_statement":
			ws, err := builder.buildWhileStmt(c)
			if err != nil {
				return nil, err
			}
			b.Stmts = append(b.Stmts, ws)
		case "method_call":
			e, err := builder.buildExpr(c)
			if err != nil {
				return nil, err
			}
			b.Stmts = append(b.Stmts, &ExprStmt{NodeBase: NodeBase{Line: nodeLine(c), Col: nodeCol(c)}, Expr: e})
		}
	}

	builder.symbolTable = prevEnv
	return b, nil
}

func (builder *Builder) buildAssignment(n *sitter.Node) (*Assignment, error) {
	idNode := n.ChildByFieldName("identifier")
	valNode := n.ChildByFieldName("value")
	val, err := builder.buildExpr(valNode)
	if err != nil {
		return nil, err
	}

	name := Identifier(text(idNode, builder.src))
	symbol, ok := builder.symbolTable.Lookup(name)

	if !ok {
		return nil, fmt.Errorf("undefined identifier %s", name)
	}

	ir_code := slices.Concat(
		val.getCode(),
		[]ir.Instr{
			{
				Op: ir.OpCopy,
				D:  symbol.Address,
				A:  val.getAddress(),
			},
		},
	)

	return &Assignment{NodeBase: NodeBase{
		Code: ir_code,
		Line: nodeLine(n), Col: nodeCol(n)}, Target: Identifier(text(idNode, builder.src)), Value: val}, nil
}

func (builder *Builder) buildReturnStmt(n *sitter.Node) (*ReturnStmt, error) {
	valNode := n.ChildByFieldName("value")
	if valNode == nil {
		return &ReturnStmt{NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)}}, nil
	}
	val, err := builder.buildExpr(valNode)
	if err != nil {
		return nil, err
	}
	return &ReturnStmt{NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)}, Value: val}, nil
}

func (builder *Builder) buildIfStmt(n *sitter.Node) (*IfStmt, error) {
	condNode := n.ChildByFieldName("condition")
	if condNode == nil {
		// fallback: in your grammar it's field-less, just the first child
		condNode = n.NamedChild(0)
	}
	cond, err := builder.buildExpr(condNode)
	if err != nil {
		return nil, err
	}

	var thenBlk, elseBlk *Block
	// second block is then, optional third is else
	blocks := []*sitter.Node{}
	for i := uint(0); i < n.NamedChildCount(); i++ {
		if n.NamedChild(i).Kind() == "block" {
			blocks = append(blocks, n.NamedChild(i))
		}
	}
	if len(blocks) > 0 {
		thenBlk, _ = builder.buildBlock(blocks[0])
	}
	if len(blocks) > 1 {
		elseBlk, _ = builder.buildBlock(blocks[1])
	}

	return &IfStmt{NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)}, Cond: cond, Then: thenBlk, Else: elseBlk}, nil
}

func (builder *Builder) buildWhileStmt(n *sitter.Node) (*WhileStmt, error) {
	condNode := n.NamedChild(0)
	cond, err := builder.buildExpr(condNode)
	if err != nil {
		return nil, err
	}
	bodyNode := n.NamedChild(n.NamedChildCount() - 1)
	body, err := builder.buildBlock(bodyNode)
	if err != nil {
		return nil, err
	}
	return &WhileStmt{NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)}, Cond: cond, Body: body}, nil
}

// ----------------------------------------------------------------------
// Expressions
// ----------------------------------------------------------------------

func (builder *Builder) buildExpr(n *sitter.Node) (Expr, error) {
	if n == nil {
		return nil, builderErrorf(n, "nil expression node")
	}
	addr := builder.getNewOffsetAddress()

	switch n.Kind() {
	case "num":
		// parse int
		var v int
		fmt.Sscanf(text(n, builder.src), "%d", &v)
		return &IntLiteral{NodeBase: NodeBase{Code: []ir.Instr{{Op: ir.OpCopy, D: addr, A: builder.getLiteralAddress(v)}}, Line: nodeLine(n), Col: nodeCol(n)}, Value: v, Type: TypeInteger, ExprBase: ExprBase{Address: addr}}, nil
	case "true":
		return &BoolLiteral{NodeBase: NodeBase{Code: []ir.Instr{{Op: ir.OpCopy, D: addr, A: builder.getLiteralAddress(1)}}, Line: nodeLine(n), Col: nodeCol(n)}, Value: true, Type: TypeBool, ExprBase: ExprBase{Address: addr}}, nil
	case "false":
		return &BoolLiteral{NodeBase: NodeBase{Code: []ir.Instr{{Op: ir.OpCopy, D: addr, A: builder.getLiteralAddress(0)}}, Line: nodeLine(n), Col: nodeCol(n)}, Value: true, Type: TypeBool, ExprBase: ExprBase{Address: addr}}, nil
	case "identifier":
		name := Identifier(text(n, builder.src))
		symbol, ok := builder.symbolTable.Lookup(name)
		if !ok {
			return nil, builderErrorf(n, "could not resolve type of %s", name)
		}
		return &IdentExpr{NodeBase: NodeBase{Line: nodeLine(n), Col: nodeCol(n)}, Name: name, Type: symbol.Type, ExprBase: ExprBase{Address: symbol.Address}}, nil
	case "method_call":
		return builder.buildCallExpr(n)
	case "int_sum", "int_sub", "int_prod", "int_div",
		"rel_eq", "rel_lt", "rel_gt",
		"bool_conjunction", "bool_disjunction", "int_rem":
		return builder.buildBinaryExpr(n)
	case "minus", "bool_not":
		return builder.buildUnaryExpr(n)
	case "paren_expr":
		inner := n.NamedChild(0)
		e, err := builder.buildExpr(inner)
		if err != nil {
			return nil, err
		}
		return e, nil
	}
	return nil, builderErrorf(n, "unhandled expression node type: %s", n.Kind())
}

func (builder *Builder) buildCallExpr(n *sitter.Node) (Expr, error) {
	idNode := n.Child(0)
	args := []Expr{}
	var ir_argument_code ir.Code
	var ir_param_calls_code ir.Code
	functionName := text(idNode, builder.src)
	for i := uint(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c.Kind() == "identifier" && i == 0 {
			continue
		}
		e, err := builder.buildExpr(c)
		ir_argument_code = slices.Concat(ir_argument_code, e.getCode())
		ir_param_calls_code = slices.Concat(ir_param_calls_code, ir.Code{{Op: ir.OpParam, A: e.getAddress()}})
		if err != nil {
			return nil, err
		}
		args = append(args, e)
	}

	addr := builder.getNewOffsetAddress()
	ir_code := slices.Concat(ir_argument_code, ir_param_calls_code, ir.Code{{Op: ir.OpCall, D: addr, S: functionName, K: len(args)}})
	return &CallExpr{NodeBase: NodeBase{Code: ir_code, Line: nodeLine(n), Col: nodeCol(n)}, Callee: Identifier(functionName), Args: args, ExprBase: ExprBase{Address: addr}}, nil
}

func (builder *Builder) getNewOffsetAddress() ir.Addr {

	addr := ir.Addr{
		Kind:  ir.Offset,
		Value: builder.getNewOffset(),
	}

	return addr
}

func (builder *Builder) getLiteralAddress(value int) ir.Addr {
	addr := ir.Addr{
		Kind:  ir.Literal,
		Value: value,
	}

	return addr
}

func (builder *Builder) buildBinaryExpr(n *sitter.Node) (Expr, error) {
	left := n.NamedChild(0)
	right := n.NamedChild(1)
	l, err := builder.buildExpr(left)
	if err != nil {
		return nil, err
	}
	r, err := builder.buildExpr(right)
	if err != nil {
		return nil, err
	}
	var op BinOp
	var t TypeKind

	// IR stuff
	var ir_op ir.Op
	addr := builder.getNewOffsetAddress()
	var ir_code []ir.Instr

	expIsBoolean := false

	switch n.Kind() {
	case "int_sum":
		op = BinAdd
		ir_op = ir.OpAdd
		t = TypeInteger
	case "int_sub":
		op = BinSub
		ir_op = ir.OpSub
		t = TypeInteger
	case "int_prod":
		op = BinMul
		ir_op = ir.OpMul
		t = TypeInteger
	case "int_div":
		op = BinDiv
		ir_op = ir.OpDiv
		t = TypeInteger
	case "int_rem":
		op = BinRem
		ir_op = ir.OpRem
		t = TypeInteger
	case "rel_eq":
		op = BinEq
		ir_op = ir.OpEQ
		t = TypeBool
	case "rel_lt":
		op = BinLT
		ir_op = ir.OpLT
		t = TypeBool
	case "rel_gt":
		op = BinGT
		ir_op = ir.OpGT
		t = TypeBool
	case "bool_conjunction":
		expIsBoolean = true
		// TODO: fixear este error haciendo codigo de saltos y COPY
		op = BinAnd
		t = TypeBool

		setTrueLabel := builder.getNewLabel()
		setFalseLabel := builder.getNewLabel()
		exitLabel := builder.getNewLabel()

		ir_code = slices.Concat(
			l.getCode(),
			r.getCode(),
			[]ir.Instr{
				{
					Op: ir.OpIfZ, S: setFalseLabel, A: l.getAddress(),
				},
				{
					Op: ir.OpIfZ, S: setFalseLabel, A: r.getAddress(),
				},
				{
					Op: ir.OpGoto, S: setTrueLabel,
				},
				{
					Op: ir.OpLabel, S: setFalseLabel,
				},
				{
					Op: ir.OpCopy, D: addr, A: builder.getLiteralAddress(0),
				},
				{
					Op: ir.OpGoto, S: exitLabel,
				},
				{
					Op: ir.OpLabel, S: setTrueLabel,
				},
				{
					Op: ir.OpCopy, D: addr, A: builder.getLiteralAddress(1),
				},
				{
					Op: ir.OpLabel, S: exitLabel,
				},
			},
		)

	case "bool_disjunction":
		expIsBoolean = true
		op = BinOr
		t = TypeBool

		setTrueLabel := builder.getNewLabel()
		setFalseLabel := builder.getNewLabel()
		checkSecondLabel := builder.getNewLabel()
		exitLabel := builder.getNewLabel()

		ir_code = slices.Concat(
			l.getCode(),
			r.getCode(),
			[]ir.Instr{
				{
					Op: ir.OpIfZ, A: l.getAddress(), S: checkSecondLabel,
				},
				{
					Op: ir.OpGoto, A: l.getAddress(), S: setTrueLabel,
				},
				{
					Op: ir.OpLabel, S: checkSecondLabel,
				},
				{
					Op: ir.OpIfZ, A: r.getAddress(), S: setFalseLabel,
				},
				{
					Op: ir.OpLabel, S: setTrueLabel,
				},
				{
					Op: ir.OpCopy, D: addr, A: builder.getLiteralAddress(1),
				},
				{
					Op: ir.OpGoto, S: exitLabel,
				},
				{
					Op: ir.OpLabel, S: setFalseLabel,
				},
				{
					Op: ir.OpCopy, D: addr, A: builder.getLiteralAddress(0),
				},
				{
					Op: ir.OpLabel, S: exitLabel,
				},
			},
		)
	}

	if n.Kind() == "int_rem" {
		fmt.Println(t)
	}

	if !expIsBoolean {
		ir_code = slices.Concat(
			l.getCode(),
			r.getCode(),
			[]ir.Instr{
				{
					Op: ir_op, D: addr, A: l.getAddress(), B: r.getAddress(),
				},
			},
		)

	}

	return &BinaryExpr{NodeBase: NodeBase{Code: ir_code, Line: nodeLine(n), Col: nodeCol(n)}, Left: l, Op: op, Right: r, Type: t, ExprBase: ExprBase{Address: addr}}, nil
}

func (builder *Builder) buildUnaryExpr(n *sitter.Node) (Expr, error) {
	opNode := n.Child(0)
	exprNode := n.Child(1)
	expr, err := builder.buildExpr(exprNode)
	if err != nil {
		return nil, err
	}
	var op UnaryOp
	var t TypeKind
	var ir_code []ir.Instr

	addr := builder.getNewOffsetAddress()

	switch text(opNode, builder.src) {
	case "-":
		op = UnaryNeg
		t = TypeInteger

		// si la produccion es E -> - E1
		// generamos la operacion binaria de resta #0 - M[lv(E1)]
		ir_code = slices.Concat(
			expr.getCode(),
			[]ir.Instr{
				{Op: ir.OpSub, D: addr, A: builder.getLiteralAddress(0), B: expr.getAddress()},
			},
		)
	case "!":
		op = UnaryNot
		t = TypeBool
		ir_code = slices.Concat(
			expr.getCode(),
			[]ir.Instr{
				{Op: ir.OpNot, D: addr, A: expr.getAddress()},
			},
		)

	default:
		return nil, builderErrorf(n, "unknown unary op: %s", text(opNode, builder.src))
	}

	return &UnaryExpr{NodeBase: NodeBase{Code: ir_code, Line: nodeLine(n), Col: nodeCol(n)}, Op: op, Expr: expr, Type: t, ExprBase: ExprBase{Address: addr}}, nil
}
