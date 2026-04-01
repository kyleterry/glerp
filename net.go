package glerp

import (
	"fmt"
	"net/http"
	"net/url"
)

func netHTTPBuiltins() map[string]BuiltinFn {
	return map[string]BuiltinFn{
		"http-request": builtinNetHTTPRequest,
	}
}

func builtinNetHTTPRequest(args []Expr) (Expr, error) {
	if err := checkArity("http-request", args, 2); err != nil {
		return nil, err
	}

	urlString, ok := args[0].(*StringExpr)
	if !ok {
		return nil, fmt.Errorf("http-request: expected a string url, got %s", args[0].String())
	}

	u, err := url.Parse(urlString.Value())
	if err != nil {
		return nil, fmt.Errorf("http-request: malformed url: %w", err)
	}

	opts, ok := args[1].(*HashTableExpr)
	if !ok {
		return nil, fmt.Errorf("http-request: expected a hash table of options, got %s", args[1].String())
	}

	_, _ = opts.Get(str("method"))

	req, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("http-request: request failed: %w", err)
	}

	return num(float64(req.StatusCode)), nil
}
