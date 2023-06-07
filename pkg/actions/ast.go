package actions

import (
	"fmt"
	"strings"

	"github.com/aweris/ghx/pkg/expression"
)

// The original source code is from https://github.com/rhysd/actionlint/blob/5656337c1ab1c7022a74181428f6ebb4504d2d25/ast.go
// We modified it to make it compilable with our codebase.

// String represents generic string value with Github Actions expression support.
type String struct {
	Value  string // Value is a raw value of the string.
	Quoted bool   // Quoted represents the string is quoted with ' or " in the YAML source.
}

// Eval evaluates the expression and returns the string value.
func (s *String) Eval(ctx *Context) (string, error) {
	if s.Quoted {
		return s.Value, nil
	}

	exprs, err := expression.ParseExpressions(s.Value)
	if err != nil {
		return "", err
	}

	if len(exprs) == 0 {
		return s.Value, nil
	}

	str := s.Value

	for _, expr := range exprs {
		val, err := expr.Evaluate(ctx)
		if err != nil {
			return "", err
		}

		if v, ok := val.(string); ok {
			str = strings.Replace(str, expr.Value, v, 1)
		}
	}

	return str, nil
}

// GenericValue represents generic value with Github Actions expression support.
type GenericValue[T any] struct {
	Value      T       // Value is a raw value of the T.
	Expression *String // Expression is a string when expression syntax ${{ }} is used for this section.
}

// Bool represents generic boolean value with Github Actions expression support.
//
// The value could be a raw boolean value or an expression. If Expression field is not nil, the value is considered
// as an expression, and it should be evaluated to get the boolean value. Otherwise, the value is a raw boolean value.
type Bool GenericValue[bool]

// Eval evaluates the expression and returns the boolean value.
func (b *Bool) Eval(ctx *Context) (bool, error) {
	if b.Expression == nil {
		return b.Value, nil
	}

	expr, err := expression.NewExpression(b.Expression.Value)
	if err != nil {
		return false, err
	}

	val, err := expr.Evaluate(ctx)
	if err != nil {
		return false, err
	}

	if v, ok := val.(bool); ok {
		return v, nil
	}

	return false, fmt.Errorf("cannot evaluate expression: %s as bool", b.Expression.Value)
}

// Int represents generic integer value with Github Actions expression support.
//
// The value could be a raw integer value or an expression. If Expression field is not nil, the value is considered
// as an expression, and it should be evaluated to get the integer value. Otherwise, the value is a raw integer value.
type Int GenericValue[int]

// Eval evaluates the expression and returns the integer value.
func (i *Int) Eval(ctx *Context) (int, error) {
	if i.Expression == nil {
		return i.Value, nil
	}

	expr, err := expression.NewExpression(i.Expression.Value)
	if err != nil {
		return 0, err
	}

	val, err := expr.Evaluate(ctx)
	if err != nil {
		return 0, err
	}

	if v, ok := val.(int); ok {
		return v, nil
	}

	return 0, fmt.Errorf("cannot evaluate expression: %s as int", i.Expression.Value)
}

// Float represents generic float value with Github Actions expression support.
//
// The value could be a raw float value or an expression. If Expression field is not nil, the value is considered
// as an expression, and it should be evaluated to get the float value. Otherwise, the value is a raw float value.
type Float GenericValue[float64]

// Eval evaluates the expression and returns the float value.
func (f *Float) Eval(ctx *Context) (float64, error) {
	if f.Expression == nil {
		return f.Value, nil
	}

	expr, err := expression.NewExpression(f.Expression.Value)
	if err != nil {
		return 0, err
	}

	val, err := expr.Evaluate(ctx)
	if err != nil {
		return 0, err
	}

	if v, ok := val.(float64); ok {
		return v, nil
	}

	return 0, fmt.Errorf("cannot evaluate expression: %s as float", f.Expression.Value)
}
