package main

import (
	"flag"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	var match, ignore string
	var iolang string
	flag.StringVar(&match, "match", ".", "include only functions matching this regular expression")
	flag.StringVar(&ignore, "ignore", "$^", "exclude functions matching this regular expression")
	flag.StringVar(&iolang, "iolang", "github.com/zephyrtronium/iolang", "import path for package iolang source code")
	flag.Parse()
	mre, err := regexp.Compile(match)
	if err != nil {
		fail("error compiling match:", err)
	}
	ire, err := regexp.Compile(ignore)
	if err != nil {
		fail("error compiling ignore:", err)
	}

	fset := token.NewFileSet()
	config := packages.Config{Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedImports, Fset: fset}
	pkgs, err := packages.Load(&config, append([]string{iolang}, flag.Args()...)...)
	if err != nil {
		fail("error loading packages:", err)
	}
	fn, pkgs := getFn(pkgs)
	results := []string{}
	for _, pkg := range pkgs {
		for res := range find(pkg.Types.Scope(), fn, mre, ire) {
			results = append(results, res)
		}
	}
	sort.Strings(results)
	for _, name := range results {
		fmt.Printf("      %s: {fn: %s}\n", trimMatch(name, mre), name)
	}
}

func fail(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func getFn(pkgs []*packages.Package) (types.Type, []*packages.Package) {
	pkg := pkgs[0].Types
	r := pkg.Scope().Lookup("Fn")
	if r == nil {
		fail(pkg.Name(), "has no definition of Fn")
	}
	t, ok := r.(*types.TypeName)
	if !ok {
		fail(pkg.Name(), "has incorrect definition of Fn:", r)
	}
	fn := t.Type().Underlying()
	return fn, pkgs[1:]
}

func find(pkg *types.Scope, fn types.Type, mre, ire *regexp.Regexp) chan string {
	ch := make(chan string, 8)
	go func() {
		defer close(ch)
		for _, name := range pkg.Names() {
			if mre.MatchString(name) && !ire.MatchString(name) {
				t := pkg.Lookup(name).Type()
				if types.AssignableTo(t, fn) {
					ch <- name
				}
			}
		}
	}()
	return ch
}

func trimMatch(name string, mre *regexp.Regexp) string {
	if mre.String() != "." {
		k := mre.FindStringIndex(name)
		name = name[k[1]:]
	}
	return strings.ToLower(name[:1]) + name[1:]
}
