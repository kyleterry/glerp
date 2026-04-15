package glerp

// TailCall represents a deferred evaluation in tail position. Instead of
// recursing in Go, tail-position forms return a TailCall that the trampoline
// loop in EvalFull resolves iteratively. This prevents stack overflow for
// deep tail recursion.
type TailCall struct {
	expr Expr
	env  *Environment
}

func (tc *TailCall) Eval(_ *Environment) (Expr, error) {
	return EvalFull(tc.expr, tc.env)
}

func (tc *TailCall) Token() Token   { return tc.expr.Token() }
func (tc *TailCall) String() string { return tc.expr.String() }

// EvalFull evaluates an expression and trampolines any TailCall results
// until a final value is produced. All non-tail-position .Eval() calls
// should use this instead of calling .Eval() directly.
func EvalFull(expr Expr, env *Environment) (Expr, error) {
	return trampoline(expr.Eval(env))
}

// trampoline resolves a (result, error) pair that may contain a TailCall.
// Used when the result comes from apply() or another function that doesn't
// take an environment parameter.
func trampoline(result Expr, err error) (Expr, error) {
	for err == nil {
		tc, ok := result.(*TailCall)
		if !ok {
			return result, nil
		}

		result, err = tc.expr.Eval(tc.env)
	}

	return nil, err
}
