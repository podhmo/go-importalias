package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis/unitchecker"

	importalias "github.com/podhmo/go-importalias"
)

func main() {
	if looksLikeVetToolInvocation(os.Args) {
		unitchecker.Main(importalias.Analyzer)
		return
	}
	os.Exit(runCLI(os.Args[1:]))
}

func looksLikeVetToolInvocation(args []string) bool {
	if len(args) == 2 {
		arg := args[1]
		return arg == "-V=full" || arg == "-flags" || strings.HasSuffix(arg, ".cfg")
	}
	return len(args) == 3 && args[1] == "-json" && strings.HasSuffix(args[2], ".cfg")
}

func runCLI(args []string) int {
	fmt.Fprintln(os.Stderr, "goimportalias: CLI mode is not implemented yet")
	return 2
}
