package main

import (
	"compilador/ir"
	"strconv"
)

// Node is the common interface implemented by all AST nodes.
type Node interface {
	// optionally add Pos/Span methods here later
	NodeType() string
	getCode() []ir.Instr
}

// ===== Program / Top-level =====

type Program struct {
	NodeBase
	// e.g. "program { ... }"
	Declarations []*VarDecl    // top-level variable declarations
	Methods      []*MethodDecl // top-level method (function) declarations
	Symbols      Env           // top-level symbol table after building
}

func (p *Program) NodeType() string { return "Program" }

// ===== Types =====

type TypeKind int

const (
	TypeVoid TypeKind = iota
	TypeBool
	TypeInteger
)

func (t TypeKind) String() string {
	switch t {
	case TypeVoid:
		return "void"
	case TypeBool:
		return "bool"
	case TypeInteger:
		return "integer"
	default:
		return "unknown"
	}
}

// ===== Identifiers =====

type Identifier string

func (id Identifier) NodeType() string { return "Identifier" }
func (id Identifier) String() string   { return string(id) }

// ===== Declarations & Parameters =====

// VarDecl corresponds to `declaration_statement` in your grammar:
//
//	<type> <identifier> = <expression> ;
type VarDecl struct {
	NodeBase
	Type  TypeKind
	Name  Identifier
	Value Expr
}

func (n *VarDecl) getCode() []ir.Instr { return n.Code }

func (d *VarDecl) NodeType() string { return "VarDecl" }

// Parameter corresponds to `parameter` (type + identifier)
type Parameter struct {
	NodeBase
	Type TypeKind
	Name Identifier
}

func (p *Parameter) NodeType() string { return "Parameter" }

// ===== Method Declarations =====
//
// method_declaration_statement:
//   <type_or_void> <identifier> "(" commaSeparatedOptional(parameter) ")" ( block | "extern" ";" )

type MethodDecl struct {
	NodeBase
	Return TypeKind
	Name   Identifier
	Params []*Parameter
	Body   *Block // nil if extern or if you want to represent "extern" via Extern=true
	Extern bool
}

func (m *MethodDecl) NodeType() string    { return "MethodDecl" }
func (m *MethodDecl) getCode() []ir.Instr { return m.Code }

// ===== Statements =====

type Stmt interface {
	Node
	isStmt()
}

type Block struct {
	NodeBase
	Declarations []*VarDecl // declarations local to the block (corresponds to repeat(field("declaration", ...)))
	Stmts        []Stmt
}

func (n *Block) getCode() []ir.Instr { return n.Code }
func (b *Block) NodeType() string    { return "Block" }
func (b *Block) isStmt()             {}

type Assignment struct {
	NodeBase
	Target Identifier // field("identifier", $.identifier)
	Value  Expr       // field("value", $._expression)
}

func (n *Assignment) getCode() []ir.Instr { return n.Code }
func (a *Assignment) NodeType() string    { return "Assignment" }
func (a *Assignment) isStmt()             {}

// base struct for expressions
type ExprBase struct {
	Address ir.Addr
}

type ExprStmt struct {
	NodeBase
	Expr Expr // used for method_call followed by ';' or any expression statement
	ExprBase
}

func (n *ExprStmt) getCode() []ir.Instr { return n.Code }
func (e *ExprStmt) NodeType() string    { return "ExprStmt" }
func (e *ExprStmt) isStmt()             {}

// ReturnStmt corresponds to `return` optional expression + ';'
type ReturnStmt struct {
	NodeBase
	Value Expr // nil if no value
}

func (n *ReturnStmt) getCode() []ir.Instr { return n.Code }
func (r *ReturnStmt) NodeType() string    { return "ReturnStmt" }
func (r *ReturnStmt) isStmt()             {}

type IfStmt struct {
	NodeBase
	Cond Expr
	Then *Block
	Else *Block // nil if absent
}

func (n *IfStmt) getCode() []ir.Instr { return n.Code }
func (i *IfStmt) NodeType() string    { return "IfStmt" }
func (i *IfStmt) isStmt()             {}

type WhileStmt struct {
	NodeBase
	Cond Expr
	Body *Block
}

func (n *WhileStmt) getCode() []ir.Instr { return n.Code }
func (w *WhileStmt) NodeType() string    { return "WhileStmt" }
func (w *WhileStmt) isStmt()             {}

// ===== Expressions =====

type Expr interface {
	Node
	isExpr()
	getAddress() ir.Addr
}

type IntLiteral struct {
	NodeBase
	Value int
	Type  TypeKind
	ExprBase
}

func (n *IntLiteral) getAddress() ir.Addr { return n.Address }
func (n *IntLiteral) getCode() []ir.Instr { return n.Code }
func (n *IntLiteral) NodeType() string    { return "IntLiteral" }
func (n *IntLiteral) isExpr()             {}

type BoolLiteral struct {
	NodeBase
	Value bool
	Type  TypeKind
	ExprBase
}

func (n *BoolLiteral) getAddress() ir.Addr { return n.Address }
func (n *BoolLiteral) getCode() []ir.Instr { return n.Code }
func (n *BoolLiteral) NodeType() string    { return "BoolLiteral" }
func (n *BoolLiteral) isExpr()             {}

type IdentExpr struct {
	NodeBase
	Name Identifier
	Type TypeKind
	ExprBase
}

func (n *IdentExpr) getAddress() ir.Addr { return n.Address }
func (n *IdentExpr) getCode() []ir.Instr { return n.Code }
func (n *IdentExpr) NodeType() string    { return "IdentExpr" }
func (n *IdentExpr) isExpr()             {}

// Unary operator kinds (for '-' and '!')
type UnaryOp int

const (
	UnaryNeg UnaryOp = iota // "-"
	UnaryNot                // "!"
)

func (op UnaryOp) String() string {
	switch op {
	case UnaryNeg:
		return "-"
	case UnaryNot:
		return "!"
	default:
		return "unknown_unary"
	}
}

type UnaryExpr struct {
	NodeBase
	Op   UnaryOp
	Expr Expr
	Type TypeKind
	ExprBase
}

func (n *UnaryExpr) getAddress() ir.Addr { return n.Address }
func (n *UnaryExpr) getCode() []ir.Instr { return n.Code }
func (n *UnaryExpr) NodeType() string    { return "UnaryExpr" }
func (n *UnaryExpr) isExpr()             {}

// Binary operators (covers int ops, relational ops, boolean ops)
type BinOp int

const (
	// arithmetic
	BinAdd BinOp = iota
	BinSub
	BinMul
	BinDiv
	BinRem

	// relational
	BinEq
	BinLT
	BinGT

	// boolean
	BinAnd
	BinOr
)

func (op BinOp) String() string {
	switch op {
	case BinAdd:
		return "+"
	case BinSub:
		return "-"
	case BinMul:
		return "*"
	case BinDiv:
		return "/"
	case BinRem:
		return "%"
	case BinEq:
		return "=="
	case BinLT:
		return "<"
	case BinGT:
		return ">"
	case BinAnd:
		return "&&"
	case BinOr:
		return "||"
	default:
		return "unknown_binop"
	}
}

type BinaryExpr struct {
	NodeBase
	Left  Expr
	Op    BinOp
	Right Expr
	Type  TypeKind
	ExprBase
}

func (n *BinaryExpr) getAddress() ir.Addr { return n.Address }
func (n *BinaryExpr) getCode() []ir.Instr { return n.Code }
func (n *BinaryExpr) NodeType() string    { return "BinaryExpr" }
func (n *BinaryExpr) isExpr()             {}

// CallExpr / Method call: identifier "(" args... ")"
type CallExpr struct {
	NodeBase
	Callee Identifier
	Args   []Expr
	Type   TypeKind
	ExprBase
}

func (n *CallExpr) getAddress() ir.Addr { return n.Address }
func (n *CallExpr) getCode() []ir.Instr { return n.Code }
func (n *CallExpr) NodeType() string    { return "CallExpr" }
func (n *CallExpr) isExpr()             {}

// Parenthesized expression (explicit in grammar as "(" _expression ")")
type ParenExpr struct {
	NodeBase
	Inner Expr
	ExprBase
}

func (n *ParenExpr) getAddress() ir.Addr { return n.Address }
func (n *ParenExpr) getCode() []ir.Instr { return n.Code }
func (n *ParenExpr) NodeType() string    { return "ParenExpr" }
func (n *ParenExpr) isExpr()             {}

// ===== Helpers (optional) =====

// NodeBase contains fields common to all AST nodes.
type NodeBase struct {
	Line int // 1-based line number of the starting token for this node
	Col  int // 1-based column number of the starting token for this node
	Code []ir.Instr
}

// LineNumber exposes the line for nodes embedding NodeBase.
func (n NodeBase) LineNumber() int { return n.Line }

// Convenience constructors (not required but often handy)
func NewIntLit(v int) *IntLiteral     { return &IntLiteral{Value: v} }
func NewBoolLit(v bool) *BoolLiteral  { return &BoolLiteral{Value: v} }
func NewIdent(name string) *IdentExpr { return &IdentExpr{Name: Identifier(name)} }

func (p *Program) String() string {
	s := "program {\n"
	for _, d := range p.Declarations {
		s += "  var " + d.Type.String() + " " + string(d.Name) + " = <expr>\n"
	}
	for _, m := range p.Methods {
		ret := m.Return.String()
		params := ""
		for i, pr := range m.Params {
			if i > 0 {
				params += ", "
			}
			params += pr.Type.String() + " " + string(pr.Name)
		}
		body := "{ ... }"
		if m.Extern {
			body = "extern;"
		}
		s += "  " + ret + " " + string(m.Name) + "(" + params + ") " + body + "\n"
	}
	s += "}\n"
	return s
}

// simple debug helpers
func (i *IntLiteral) GoString() string  { return strconv.Itoa(i.Value) }
func (b *BoolLiteral) GoString() string { return strconv.FormatBool(b.Value) }
func (id *IdentExpr) GoString() string  { return string(id.Name) }
func (p *ParenExpr) GoString() string   { return "(" + p.Inner.NodeType() + ")" }
func (c *CallExpr) GoString() string    { return string(c.Callee) + "(...)" }
func (b *BinaryExpr) GoString() string {
	return "(" + b.Left.NodeType() + " " + b.Op.String() + " " + b.Right.NodeType() + ")"
}
