package main

import "compilador/ir"

type VarKind uint

const (
	LocalVar VarKind = iota
	Param
	GlobalVar
	Method
)

type Env struct {
	Table Table
	Prev  *Env
}

type ParamInfo struct {
	Name Identifier
	Type TypeKind
}

type FuncInfo struct {
	Return   TypeKind
	Params   []ParamInfo
	Arity    int
	DeclLine int
}

type Symbol struct {
	Type    TypeKind
	VarKind VarKind
	Func    *FuncInfo
	Address ir.Addr
}

type Table map[Identifier]Symbol

func (env Env) Lookup(name Identifier) (Symbol, bool) {
	var currentEnv Env = env
	for {

		// lookup in current frame
		symbol, ok := currentEnv.Table[name]
		if ok {
			return symbol, true
		} else if currentEnv.Prev != nil {
			// lookup in previous frame
			currentEnv = *currentEnv.Prev
		} else {
			break
		}
	}
	return Symbol{}, false
}

func (env Env) Insert(name Identifier, symbol Symbol) {
	env.Table[name] = symbol
}
