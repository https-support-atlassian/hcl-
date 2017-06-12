package zclsyntax

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-zcl/zcl"
)

type Operation struct {
	Impl function.Function
	Type cty.Type
}

var (
	OpLogicalOr = &Operation{
		Impl: stdlib.OrFunc,
		Type: cty.Bool,
	}
	OpLogicalAnd = &Operation{
		Impl: stdlib.AndFunc,
		Type: cty.Bool,
	}
	OpLogicalNot = &Operation{
		Impl: stdlib.NotFunc,
		Type: cty.Bool,
	}

	OpEqual = &Operation{
		Impl: stdlib.EqualFunc,
		Type: cty.Bool,
	}
	OpNotEqual = &Operation{
		Impl: stdlib.NotEqualFunc,
		Type: cty.Bool,
	}

	OpGreaterThan = &Operation{
		Impl: stdlib.GreaterThanFunc,
		Type: cty.Bool,
	}
	OpGreaterThanOrEqual = &Operation{
		Impl: stdlib.GreaterThanOrEqualToFunc,
		Type: cty.Bool,
	}
	OpLessThan = &Operation{
		Impl: stdlib.LessThanFunc,
		Type: cty.Bool,
	}
	OpLessThanOrEqual = &Operation{
		Impl: stdlib.LessThanOrEqualToFunc,
		Type: cty.Bool,
	}

	OpAdd = &Operation{
		Impl: stdlib.AddFunc,
		Type: cty.Number,
	}
	OpSubtract = &Operation{
		Impl: stdlib.SubtractFunc,
		Type: cty.Number,
	}
	OpMultiply = &Operation{
		Impl: stdlib.MultiplyFunc,
		Type: cty.Number,
	}
	OpDivide = &Operation{
		Impl: stdlib.DivideFunc,
		Type: cty.Number,
	}
	OpModulo = &Operation{
		Impl: stdlib.ModuloFunc,
		Type: cty.Number,
	}
	OpNegate = &Operation{
		Impl: stdlib.NegateFunc,
		Type: cty.Number,
	}
)

var binaryOps []map[TokenType]*Operation

func init() {
	// This operation table maps from the operator's token type
	// to the AST operation type. All expressions produced from
	// binary operators are BinaryOp nodes.
	//
	// Binary operator groups are listed in order of precedence, with
	// the *lowest* precedence first. Operators within the same group
	// have left-to-right associativity.
	binaryOps = []map[TokenType]*Operation{
		{
			TokenOr: OpLogicalOr,
		},
		{
			TokenAnd: OpLogicalAnd,
		},
		{
			TokenEqualOp:  OpEqual,
			TokenNotEqual: OpNotEqual,
		},
		{
			TokenGreaterThan:   OpGreaterThan,
			TokenGreaterThanEq: OpGreaterThanOrEqual,
			TokenLessThan:      OpLessThan,
			TokenLessThanEq:    OpLessThanOrEqual,
		},
		{
			TokenPlus:  OpAdd,
			TokenMinus: OpSubtract,
		},
		{
			TokenStar:    OpMultiply,
			TokenSlash:   OpDivide,
			TokenPercent: OpModulo,
		},
	}
}

type BinaryOpExpr struct {
	LHS Expression
	Op  *Operation
	RHS Expression

	SrcRange zcl.Range
}

func (e *BinaryOpExpr) walkChildNodes(w internalWalkFunc) {
	e.LHS = w(e.LHS).(Expression)
	e.RHS = w(e.LHS).(Expression)
}

func (e *BinaryOpExpr) Value(ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	impl := e.Op.Impl // assumed to be a function taking exactly two arguments
	params := impl.Params()
	lhsParam := params[0]
	rhsParam := params[1]

	var diags zcl.Diagnostics

	givenLHSVal, lhsDiags := e.LHS.Value(ctx)
	givenRHSVal, rhsDiags := e.RHS.Value(ctx)
	diags = append(diags, lhsDiags...)
	diags = append(diags, rhsDiags...)

	lhsVal, err := convert.Convert(givenLHSVal, lhsParam.Type)
	if err != nil {
		diags = append(diags, &zcl.Diagnostic{
			Severity: zcl.DiagError,
			Summary:  "Invalid operand",
			Detail:   fmt.Sprintf("Unsuitable value for left operand: %s.", err),
			Subject:  e.LHS.Range().Ptr(),
			Context:  &e.SrcRange,
		})
	}
	rhsVal, err := convert.Convert(givenRHSVal, rhsParam.Type)
	if err != nil {
		diags = append(diags, &zcl.Diagnostic{
			Severity: zcl.DiagError,
			Summary:  "Invalid operand",
			Detail:   fmt.Sprintf("Unsuitable value for right operand: %s.", err),
			Subject:  e.RHS.Range().Ptr(),
			Context:  &e.SrcRange,
		})
	}

	if diags.HasErrors() {
		// Don't actually try the call if we have errors already, since the
		// this will probably just produce a confusing duplicative diagnostic.
		return cty.UnknownVal(e.Op.Type), diags
	}

	args := []cty.Value{lhsVal, rhsVal}
	result, err := impl.Call(args)
	if err != nil {
		diags = append(diags, &zcl.Diagnostic{
			// FIXME: This diagnostic is useless.
			Severity: zcl.DiagError,
			Summary:  "Operation failed",
			Detail:   fmt.Sprintf("Error during operation: %s.", err),
			Subject:  e.RHS.Range().Ptr(),
			Context:  &e.SrcRange,
		})
		return cty.UnknownVal(e.Op.Type), diags
	}

	return result, diags
}

func (e *BinaryOpExpr) Range() zcl.Range {
	return e.SrcRange
}

func (e *BinaryOpExpr) StartRange() zcl.Range {
	return e.LHS.StartRange()
}

type UnaryOpExpr struct {
	Op  *Operation
	Val Expression

	SrcRange    zcl.Range
	SymbolRange zcl.Range
}

func (e *UnaryOpExpr) walkChildNodes(w internalWalkFunc) {
	e.Val = w(e.Val).(Expression)
}

func (e *UnaryOpExpr) Value(ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	panic("UnaryOpExpr.Value not yet implemented")
}

func (e *UnaryOpExpr) Range() zcl.Range {
	return e.SrcRange
}

func (e *UnaryOpExpr) StartRange() zcl.Range {
	return e.SymbolRange
}
