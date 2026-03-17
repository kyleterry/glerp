package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"go.e64ec.com/glerp"
)

func main() {
	if len(os.Args) > 1 {
		runFile(os.Args[1])
		return
	}
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		runFile("-")
		return
	}
	runREPL()
}

func runFile(path string) {
	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer f.Close() //nolint:errcheck
		r = f
	}

	lexer, err := glerp.NewLexer(r)
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

	env := glerp.NewEnvironment(glerp.DefaultConfig())
	for _, expr := range exprs {
		if _, err := expr.Eval(env); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

// replCompleter implements readline.AutoCompleter for the REPL.
// It completes Scheme symbol names bound in env.
type replCompleter struct {
	env *glerp.Environment
}

// isSymbolRune reports whether r can appear in a Scheme symbol name.
func isSymbolRune(r rune) bool {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	switch r {
	case '!', '$', '%', '&', '*', '+', '-', '.', '/', ':', '<', '=', '>', '?', '^', '_', '~':
		return true
	}
	return false
}

// Do implements readline.AutoCompleter. It finds the symbol prefix before the
// cursor and returns all matching completions.
func (c *replCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	// Find the start of the current symbol token.
	start := pos
	for start > 0 && isSymbolRune(line[start-1]) {
		start--
	}
	prefix := string(line[start:pos])

	for _, name := range c.env.AllNames() {
		if strings.HasPrefix(name, prefix) {
			newLine = append(newLine, []rune(name[len(prefix):]))
		}
	}
	return newLine, len(prefix)
}

func runREPL() {
	env := glerp.NewEnvironment(glerp.DefaultConfig())

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     os.ExpandEnv("$HOME/.glerp_history"),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    &replCompleter{env: env},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "readline error: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close() //nolint:errcheck

	fmt.Println("glerp 0.1  --  Ctrl-D to exit")
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

		lexer, err := glerp.NewLexer(strings.NewReader(src))
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
		case r == '(' || r == '[':
			depth++
		case r == ')' || r == ']':
			depth--
		}
	}
	return depth == 0
}
