package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"go.e64ec.com/glerp"
	"go.e64ec.com/glerp/token"
)

func main() {
	if len(os.Args) > 1 {
		runFile(os.Args[1])
		return
	}
	runREPL()
}

func runFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close() //nolint:errcheck

	lexer, err := token.NewLexer(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lexer error: %v\n", err)
		os.Exit(1)
	}

	parser := glerp.NewParser(lexer)
	exprs, err := parser.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	env := glerp.NewEnvironment()
	for _, expr := range exprs {
		if _, err := expr.Eval(env); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

func runREPL() {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     os.ExpandEnv("$HOME/.glerp_history"),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "readline error: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close() //nolint:errcheck

	fmt.Println("glerp 0.1  —  Ctrl-D to exit")

	env := glerp.NewEnvironment()
	var buf strings.Builder

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			// Ctrl-C: discard current buffer and start fresh.
			buf.Reset()
			rl.SetPrompt("> ")
			continue
		}
		if err == io.EOF {
			fmt.Println()
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			break
		}

		buf.WriteString(line)
		buf.WriteByte('\n')

		src := buf.String()

		if !balanced(src) {
			if strings.TrimSpace(src) == "" {
				buf.Reset()
				rl.SetPrompt("> ")
			} else {
				rl.SetPrompt("  ")
			}
			continue
		}

		buf.Reset()
		rl.SetPrompt("> ")

		lexer, err := token.NewLexer(strings.NewReader(src))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}

		parser := glerp.NewParser(lexer)
		exprs, err := parser.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
			continue
		}

		for _, expr := range exprs {
			result, err := expr.Eval(env)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				continue
			}
			if _, isVoid := result.(*glerp.VoidExpr); !isVoid {
				fmt.Println(result.String())
			}
		}
	}
}

// balanced reports whether s has balanced parentheses and contains
// non-whitespace content. Returns true for bare atoms and numbers (depth 0).
func balanced(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	depth := 0
	inStr := false
	for _, r := range s {
		switch {
		case r == '"':
			inStr = !inStr
		case inStr:
			// skip string contents
		case r == '(':
			depth++
		case r == ')':
			depth--
		}
	}
	return depth == 0
}
