package glerp

import (
	"fmt"
	"maps"
	"sync/atomic"
)

// gensymCounter generates unique names for hygienic macro expansion.
var gensymCounter atomic.Int64

// gensym returns a fresh name derived from base that is guaranteed not to
// collide with any user-written identifier (which cannot contain ".").
func gensym(base string) string {
	n := gensymCounter.Add(1)
	return fmt.Sprintf("%s.%d", base, n)
}

// SyntaxRulesExpr is a hygienic macro transformer produced by syntax-rules.
// It holds an ordered list of pattern/template rules and the set of literal
// keywords that must match verbatim rather than acting as pattern variables.
//
// When a macro call is evaluated, the transformer matches the unevaluated
// call form against each rule's pattern in order and rewrites the call to the
// corresponding template. The result is then evaluated in place of the
// original call.
type SyntaxRulesExpr struct {
	name     string          // set by define-syntax; used only for display
	literals map[string]bool // keywords that match verbatim in patterns
	rules    []macroRule
}

type macroRule struct {
	pattern  Expr
	template Expr
}

func (e *SyntaxRulesExpr) Eval(_ *Environment) (Expr, error) { return e, nil }
func (e *SyntaxRulesExpr) Token() Token                      { return Token{} }
func (e *SyntaxRulesExpr) String() string {
	return fmt.Sprintf("#<syntax-transformer %s>", e.name)
}

// expand tries each rule against the full unevaluated call form and returns
// the rewritten AST ready for evaluation.
func (e *SyntaxRulesExpr) expand(form *ListExpr) (Expr, error) {
	for _, rule := range e.rules {
		patList, ok := rule.pattern.(*ListExpr)
		if !ok {
			continue
		}

		b := newMacroBindings()

		// The first element of each pattern is the macro keyword (or _).
		// Skip it in both the pattern and the call form.
		if matchList(patList.elements[1:], form.elements[1:], e.literals, b) {
			return expandTemplate(rule.template, b, e.literals)
		}
	}

	return nil, fmt.Errorf("syntax: no matching pattern for %s", form.String())
}

// macroBindings holds the variables captured during a successful pattern match.
type macroBindings struct {
	// vars maps a simple pattern variable to its single matched value.
	vars map[string]Expr
	// ellipsis maps an ellipsis-bound pattern variable to the slice of values
	// matched across all repetitions, in order.
	ellipsis map[string][]Expr
}

func newMacroBindings() *macroBindings {
	return &macroBindings{
		vars:     make(map[string]Expr),
		ellipsis: make(map[string][]Expr),
	}
}

func (b *macroBindings) isPatternVar(name string) bool {
	_, inVars := b.vars[name]
	_, inEllipsis := b.ellipsis[name]
	return inVars || inEllipsis
}

// parseLiterals extracts symbol names from a literal list expression,
// returning a set. Used by syntax-rules and syntax-case.
func parseLiterals(name string, list *ListExpr) (map[string]bool, error) {
	literals := make(map[string]bool, len(list.elements))

	for _, el := range list.elements {
		sym, ok := el.(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("%s: literal must be a symbol, got %s", name, el.String())
		}

		literals[sym.val] = true
	}

	return literals, nil
}

// mergeSyntaxEnv creates a new syntaxEnvExpr by merging an existing syntax
// environment (may be nil) with additional bindings and literals.
func mergeSyntaxEnv(existing *syntaxEnvExpr, b *macroBindings, literals map[string]bool) *syntaxEnvExpr {
	merged := newMacroBindings()
	mergedLiterals := make(map[string]bool)

	if existing != nil {
		maps.Copy(merged.vars, existing.bindings.vars)
		maps.Copy(merged.ellipsis, existing.bindings.ellipsis)
		maps.Copy(mergedLiterals, existing.literals)
	}

	maps.Copy(merged.vars, b.vars)
	maps.Copy(merged.ellipsis, b.ellipsis)
	maps.Copy(mergedLiterals, literals)

	return &syntaxEnvExpr{bindings: merged, literals: mergedLiterals}
}

// matchPattern matches a single pattern element against a form element,
// recording any captured bindings. Returns false if the match fails.
//
//   - _ matches anything without binding.
//   - A symbol in the literals list matches only the identical symbol.
//   - Any other symbol is a pattern variable and captures the form.
//   - A list pattern matches a list form structurally, including ... repetition.
//   - Number, string, and bool patterns match only their equal counterpart.
func matchPattern(pat, form Expr, literals map[string]bool, b *macroBindings) bool {
	switch p := pat.(type) {
	case *SymbolExpr:
		if p.val == "_" {
			return true
		}
		if literals[p.val] {
			sym, ok := form.(*SymbolExpr)
			return ok && sym.val == p.val
		}
		b.vars[p.val] = form
		return true
	case *ListExpr:
		formList, ok := form.(*ListExpr)
		if !ok {
			return false
		}
		return matchList(p.elements, formList.elements, literals, b)
	case *NumberExpr:
		fn, ok := form.(*NumberExpr)
		return ok && fn.val == p.val
	case *BoolExpr:
		fb, ok := form.(*BoolExpr)
		return ok && fb.val == p.val
	case *StringExpr:
		fs, ok := form.(*StringExpr)
		return ok && fs.val == p.val
	}
	return false
}

func isEllipsis(e Expr) bool {
	sym, ok := e.(*SymbolExpr)
	return ok && sym.val == "..."
}

// matchList matches a sequence of pattern elements against a sequence of form
// elements. An element followed by ... matches zero or more repetitions.
func matchList(patElems, formElems []Expr, literals map[string]bool, b *macroBindings) bool {
	pi, fi := 0, 0

	for pi < len(patElems) {
		if pi+1 < len(patElems) && isEllipsis(patElems[pi+1]) {
			subPat := patElems[pi]
			patVars := collectPatVars(subPat, literals)

			// Initialise slots so zero-match ellipsis variables still exist.
			for _, v := range patVars {
				if _, exists := b.ellipsis[v]; !exists {
					b.ellipsis[v] = []Expr{}
				}
			}

			// Consume as many form elements as possible while leaving enough
			// for the remaining (non-ellipsis) pattern elements.
			remaining := len(patElems) - (pi + 2)
			canConsume := len(formElems) - fi - remaining
			if canConsume < 0 {
				return false
			}

			for j := range canConsume {
				sub := newMacroBindings()
				if !matchPattern(subPat, formElems[fi+j], literals, sub) {
					return false
				}
				for _, v := range patVars {
					b.ellipsis[v] = append(b.ellipsis[v], sub.vars[v])
				}
			}

			fi += canConsume
			pi += 2 // advance past subPat and "..."
			continue
		}

		if fi >= len(formElems) {
			return false
		}
		if !matchPattern(patElems[pi], formElems[fi], literals, b) {
			return false
		}
		pi++
		fi++
	}

	return fi == len(formElems)
}

// collectPatVars returns the pattern variable names reachable from pattern.
func collectPatVars(pat Expr, literals map[string]bool) []string {
	var vars []string
	switch p := pat.(type) {
	case *SymbolExpr:
		if p.val != "_" && p.val != "..." && !literals[p.val] {
			vars = append(vars, p.val)
		}
	case *ListExpr:
		for _, el := range p.elements {
			vars = append(vars, collectPatVars(el, literals)...)
		}
	}
	return vars
}

// expandTemplate instantiates template using bindings captured from the
// pattern match. It is hygienic: variables introduced by the template in
// let, let*, and lambda binding positions are renamed to fresh gensym names
// so they cannot accidentally capture or be captured by use-site variables.
func expandTemplate(template Expr, b *macroBindings, literals map[string]bool) (Expr, error) {
	renames := make(map[string]string)
	collectBindingRenames(template, b, renames)

	return doExpand(template, b, literals, renames)
}

// collectBindingRenames walks template and records a gensym rename for every
// variable introduced in a binding position (let, let*, lambda, define) that
// is not already a pattern variable. Only binding-position variables are
// renamed; free references such as recursive macro calls are left alone.
func collectBindingRenames(template Expr, b *macroBindings, renames map[string]string) {
	lst, ok := template.(*ListExpr)
	if !ok || len(lst.elements) == 0 {
		return
	}
	head, ok := lst.elements[0].(*SymbolExpr)
	if !ok {
		for _, el := range lst.elements {
			collectBindingRenames(el, b, renames)
		}
		return
	}

	rename := func(sym *SymbolExpr) {
		if !b.isPatternVar(sym.val) {
			if _, exists := renames[sym.val]; !exists {
				renames[sym.val] = gensym(sym.val)
			}
		}
	}

	switch head.val {
	case "let", "let*":
		// (let ((var val) ...) body ...)
		if len(lst.elements) >= 2 {
			if bindings, ok := lst.elements[1].(*ListExpr); ok {
				for _, binding := range bindings.elements {
					if pair, ok := binding.(*ListExpr); ok && len(pair.elements) >= 1 {
						if sym, ok := pair.elements[0].(*SymbolExpr); ok {
							rename(sym)
						}
					}
				}
			}
		}
	case "lambda":
		// (lambda (param ...) body ...)
		if len(lst.elements) >= 2 {
			if params, ok := lst.elements[1].(*ListExpr); ok {
				for _, p := range params.elements {
					if sym, ok := p.(*SymbolExpr); ok {
						rename(sym)
					}
				}
			}
		}
	case "define":
		// (define name val) or (define (name params...) body ...)
		if len(lst.elements) >= 2 {
			switch target := lst.elements[1].(type) {
			case *SymbolExpr:
				rename(target)
			case *ListExpr:
				if len(target.elements) > 0 {
					if sym, ok := target.elements[0].(*SymbolExpr); ok {
						rename(sym)
					}
				}
			}
		}
	}

	for _, el := range lst.elements[1:] {
		collectBindingRenames(el, b, renames)
	}
}

func doExpand(template Expr, b *macroBindings, literals map[string]bool, renames map[string]string) (Expr, error) {
	switch t := template.(type) {
	case *SymbolExpr:
		// Pattern variable: substitute the matched value.
		if val, ok := b.vars[t.val]; ok {
			return val, nil
		}
		// Ellipsis variable used outside an ellipsis context is a macro error.
		if _, ok := b.ellipsis[t.val]; ok {
			return nil, fmt.Errorf("syntax: ellipsis variable %q used outside ellipsis template", t.val)
		}
		// Template-introduced binding: apply hygiene rename.
		if renamed, ok := renames[t.val]; ok {
			return &SymbolExpr{val: renamed}, nil
		}
		return t, nil

	case *ListExpr:
		result := make([]Expr, 0, len(t.elements))

		for i := 0; i < len(t.elements); i++ {
			if i+1 < len(t.elements) && isEllipsis(t.elements[i+1]) {
				// Ellipsis expansion: repeat the sub-template once per match.
				evars := findEllipsisVars(t.elements[i], b)
				if len(evars) == 0 {
					return nil, fmt.Errorf("syntax: no ellipsis variable in ellipsis template position")
				}

				count := len(b.ellipsis[evars[0]])
				for _, v := range evars[1:] {
					if len(b.ellipsis[v]) != count {
						return nil, fmt.Errorf("syntax: ellipsis variables %q and %q have different match counts", evars[0], v)
					}
				}

				for j := range count {
					sub := &macroBindings{
						vars:     copyBindings(b.vars),
						ellipsis: b.ellipsis,
					}
					for _, v := range evars {
						sub.vars[v] = b.ellipsis[v][j]
					}
					expanded, err := doExpand(t.elements[i], sub, literals, renames)
					if err != nil {
						return nil, err
					}
					result = append(result, expanded)
				}
				i++ // skip "..."
				continue
			}
			expanded, err := doExpand(t.elements[i], b, literals, renames)
			if err != nil {
				return nil, err
			}
			result = append(result, expanded)
		}
		return &ListExpr{elements: result}, nil

	default:
		// Numbers, strings, and booleans in templates are self-quoting.
		return template, nil
	}
}

// findEllipsisVars returns the names of ellipsis-bound pattern variables
// reachable from template.
func findEllipsisVars(template Expr, b *macroBindings) []string {
	var vars []string
	switch t := template.(type) {
	case *SymbolExpr:
		if _, ok := b.ellipsis[t.val]; ok {
			vars = append(vars, t.val)
		}
	case *ListExpr:
		for _, el := range t.elements {
			vars = append(vars, findEllipsisVars(el, b)...)
		}
	}
	return vars
}

func copyBindings(m map[string]Expr) map[string]Expr {
	cp := make(map[string]Expr, len(m))
	maps.Copy(cp, m)
	return cp
}

// evalDefineSyntax implements (define-syntax name transformer).
// The transformer is typically a (syntax-rules ...) expression.
func evalDefineSyntax(args []Expr, env *Environment) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("define-syntax: expected (define-syntax name transformer), got %d args", len(args))
	}

	nameSym, ok := args[0].(*SymbolExpr)
	if !ok {
		return nil, fmt.Errorf("define-syntax: name must be a symbol, got %s", args[0].String())
	}

	transformer, err := args[1].Eval(env)
	if err != nil {
		return nil, err
	}

	switch t := transformer.(type) {
	case *SyntaxRulesExpr:
		t.name = nameSym.val
		env.Bind(nameSym.val, t)
	case *LambdaExpr, *BuiltinExpr:
		env.Bind(nameSym.val, &TransformerExpr{proc: transformer})
	default:
		return nil, fmt.Errorf("define-syntax: expected syntax-rules or procedure, got %s", transformer.String())
	}

	return Void(), nil
}

// evalSyntaxRules implements (syntax-rules (literal ...) (pattern template) ...).
// Returns a SyntaxRulesExpr ready to be bound via define-syntax.
func evalSyntaxRules(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("syntax-rules: expected (syntax-rules (literals ...) rule ...)")
	}

	litList, ok := args[0].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("syntax-rules: first argument must be a list of literals, got %s", args[0].String())
	}

	literals, err := parseLiterals("syntax-rules", litList)
	if err != nil {
		return nil, err
	}

	rules := make([]macroRule, 0, len(args)-1)
	for _, arg := range args[1:] {
		pair, ok := arg.(*ListExpr)
		if !ok || len(pair.elements) != 2 {
			return nil, fmt.Errorf("syntax-rules: each rule must be (pattern template), got %s", arg.String())
		}
		rules = append(rules, macroRule{
			pattern:  pair.elements[0],
			template: pair.elements[1],
		})
	}

	return &SyntaxRulesExpr{literals: literals, rules: rules}, nil
}
