package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

type Value struct {
	Type  string
	Value string
}

func getTypeName(ty ast.Expr, currentPackage string, pkgs map[string]string) string {
	if typeName, ok := ty.(*ast.Ident); ok {
		return currentPackage + "." + typeName.String()
	}

	if typeName, ok := ty.(*ast.SelectorExpr); ok {
		tn := fmt.Sprintf("%v", typeName.X)

		// First check if the type is an alias for a known package in the
		// imports.
		if fqn, ok := pkgs[tn]; ok {
			tn = fqn
		}

		return tn + "." + typeName.Sel.String()
	}

	return ""
}

func resolveValue(value ast.Expr, ty ast.Expr, found map[string]Value, currentPackage string, pkgs map[string]string) Value {
	switch v := value.(type) {
	case *ast.BasicLit:
		return Value{Type: getTypeName(ty, currentPackage, pkgs), Value: v.Value}

	case *ast.CallExpr:
		// If there are no args this means it must be a call to a function.
		// Which we don't support.
		if len(v.Args) != 1 {
			return Value{}
		}

		v2 := resolveValue(v.Args[0], ty, found, currentPackage, pkgs)
		return Value{Type: getTypeName(v.Fun, currentPackage, pkgs), Value: v2.Value}

	case *ast.Ident:
		f, ok := found[v.Name]
		if ok {
			return f
		}

		return Value{Type: getTypeName(ty, currentPackage, pkgs), Value: "0"}

	case *ast.UnaryExpr:
		v2 := resolveValue(v.X, ty, found, currentPackage, pkgs)
		return Value{Type: getTypeName(ty, currentPackage, pkgs), Value: "-" + v2.Value}
	}

	// Any other case we encounter we will assume it's too complicated to
	// understand.
	return Value{}
}

var inBuiltTypes = map[string]struct{}{
	"byte":       {},
	"complex128": {},
	"complex64":  {},
	"float32":    {},
	"float64":    {},
	"int":        {},
	"int16":      {},
	"int32":      {},
	"int64":      {},
	"int8":       {},
	"rune":       {},
	"string":     {},
	"uint":       {},
	"uint16":     {},
	"uint32":     {},
	"uint64":     {},
	"uint8":      {},
	"uintptr":    {},
}

func isCastToInBuiltType(t ast.Expr) bool {
	if i, ok := t.(*ast.CallExpr); ok {
		_, ok := inBuiltTypes[fmt.Sprintf("%v", i.Fun)]

		return ok
	}

	return false
}

func getIotaType(ty ast.Expr, values []ast.Expr) ast.Expr {
	if len(values) == 0 {
		return nil
	}

	// "iota" as the raw keyword
	if v, ok := values[0].(*ast.Ident); ok && v.Name == "iota" {
		return ty
	}

	// iota wrapped in a type cast, like: Foo(iota)
	if v, ok := values[0].(*ast.CallExpr); ok && len(v.Args) == 1 {
		return getIotaType(v.Fun, v.Args)
	}

	return nil
}

func appendEnumValues(in map[string]Value, decl *ast.GenDecl, currentPackage string, pkgs map[string]string) {
	iotaValue := 1
	var iotaType ast.Expr

	for _, spec := range decl.Specs {
		if valueSpec, ok := spec.(*ast.ValueSpec); ok {
			// Catch iota so we can give future values to it. However, be
			// careful that not all empty types mean an iota. Without a previous
			// iota it would just be no type (the Go compiler would resolve the
			// basic lit of whatever type in this case).
			if iotaType == nil {
				iotaType = getIotaType(valueSpec.Type, valueSpec.Values)
			}

			// Not all names will have a value. An iota is an example of when
			// there will only be one value for multiple names. However, there
			// may be any number of values up to the number of names.
			if len(valueSpec.Names) > len(valueSpec.Values) {
				valueSpec.Values = append(valueSpec.Values, &ast.BasicLit{
					Kind:  token.INT,
					Value: strconv.Itoa(iotaValue),
				})
				iotaValue++
			}

			for i, name := range valueSpec.Names {
				// Try to use the explicit type if there is one, or fallback to
				// the iotaType if not. There also may not be an iota type
				// either.
				ty := valueSpec.Type
				if ty == nil {
					ty = iotaType
				}

				if !isCastToInBuiltType(valueSpec.Values[i]) {
					fqn := currentPackage + "." + name.String()
					in[fqn] = resolveValue(valueSpec.Values[i], ty, in, currentPackage, pkgs)
				}
			}
		}
	}
}

func valuesToEnums(values map[string]Value) map[string][]string {
	r := map[string][]string{}
	for name, value := range values {
		// An empty type means that its a basic literal, or perhaps something
		// more complex. Either way we could not resolve it's custom type so we
		// will ignore it as an enum value.
		if value.Type != "" {
			r[value.Type] = append(r[value.Type], name)
		}
	}

	// Keep enum values sorted by name.
	for ty := range r {
		sort.Strings(r[ty])
	}

	return r
}

func getEnumValuesFromFile(node *ast.File, basePackage string, pkgs map[string]string) map[string]Value {
	found := map[string]Value{}

	ast.Inspect(node, func(n ast.Node) bool {
		// GenDecl can appear anywhere, make sure we do not traverse into
		// functions, otherwise local variables would be included in the enum
		// values.
		if _, ok := n.(*ast.FuncDecl); ok {
			return false
		}

		if n, ok := n.(*ast.GenDecl); ok {
			appendEnumValues(found, n, basePackage, pkgs)
		}

		return true
	})

	return found
}

func main() {
	var (
		flagShowEnums bool
		flagVerbose   bool
	)

	flag.BoolVar(&flagShowEnums, "show-enums", false,
		"Show all enums. Useful for debugging.")

	flag.BoolVar(&flagVerbose, "verbose", false,
		"Show all files processed.")

	flag.Parse()

	out, exitStatus := run(flagVerbose, flagShowEnums, flag.Args())
	fmt.Printf("%s", out)
	os.Exit(exitStatus)
}

func run(flagVerbose bool, flagShowEnums bool, args []string) (string, int) {
	if len(args) == 0 {
		args = []string{"."}
	}

	var out string
	allValues := map[string]Value{}
	allSwitches := map[string][]string{}
	basePackage := getBasePackageName()
	for _, p := range args {
		out += runPath(p, allValues, allSwitches, flagVerbose, basePackage)
	}

	if flagShowEnums {
		allEnums := valuesToEnums(allValues)

		// Show sorted by name.
		var tys []string
		for ty := range allEnums {
			tys = append(tys, ty)
		}

		sort.Strings(tys)

		for _, ty := range tys {
			out += fmt.Sprintln(ty)
			for _, enum := range allEnums[ty] {
				// The enum will be in the form of:
				// <package location>.<name>
				//
				// This is confusing because it looks like <package location> is
				// the type, which is impossible since we define the type above.
				// To make it less ambiguous and verbose we only need to print
				// the package it was found in if it was a different package
				// from where the type is defined.
				parts := strings.Split(enum, ".")
				pkgName := strings.Join(parts[:len(parts)-1], ".")

				if pkgName == pkgNameFromType(ty) {
					out += fmt.Sprintf("  %s\n", parts[len(parts)-1])
				} else {
					out += fmt.Sprintf("  %s (in %s)\n", parts[len(parts)-1], pkgName)
				}
			}
		}
	}

	var switchKeys []string
	for switchPos := range allSwitches {
		switchKeys = append(switchKeys, switchPos)
	}

	sort.Strings(switchKeys)

	exitStatus := 0
	for _, switchPos := range switchKeys {
		switchStmt := allSwitches[switchPos]
		missingValues := findMissingValues(allValues, switchStmt)
		if len(missingValues) > 0 {
			out += fmt.Sprintln(switchPos, "switch is missing cases for:",
				strings.Join(missingValues, ", "))
			exitStatus = 1
		}
	}

	return out, exitStatus
}

func pkgNameFromType(ty string) string {
	parts := strings.Split(ty, ".")

	return strings.Join(parts[:len(parts)-1], ".")
}

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Fatalln(err)
		return false
	}

	return fileInfo.IsDir()
}

func runPath(p string, allValues map[string]Value, allSwitches map[string][]string, verbose bool, basePackage string) (out string) {
	if isDirectory(p) {
		files, err := ioutil.ReadDir(p)
		if err != nil {
			log.Fatalln(err)
		}

		for _, file := range files {
			subPath := p + "/" + file.Name()
			if isDirectory(subPath) || strings.HasSuffix(subPath, ".go") {
				subBasePackage := path.Join(basePackage, path.Base(p))
				out += runPath(subPath, allValues, allSwitches, verbose, subBasePackage)
			}
		}

		return
	}

	if verbose {
		out += fmt.Sprintln("#", p)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, p, nil, 0)
	if err != nil {
		log.Panic(err)
	}

	pkgs := map[string]string{}
	for _, imp := range node.Imports {
		value := strings.Trim(imp.Path.Value, `"`)
		key := value
		if imp.Name != nil {
			key = imp.Name.String()
		}
		onlyKey := path.Base(key)
		pkgs[onlyKey] = value

		// This is tricky. Go lets you specify a different package name in the
		// source files from the directory they reside in. Right now I haven't
		// built a way to fetch all the real package names from the directories
		// so I'll just assume that people would only remove punctuation (like a
		// dash or underscore).
		onlyKey = strings.Replace(onlyKey, "-", "", -1)
		onlyKey = strings.Replace(onlyKey, "_", "", -1)
		pkgs[onlyKey] = value
	}

	values := getEnumValuesFromFile(node, basePackage, pkgs)
	for a, b := range values {
		allValues[a] = b
	}

	switches := getSwitchesFromFile(fset, node, basePackage, pkgs)
	for a, b := range switches {
		allSwitches[a] = append(allSwitches[a], b...)
	}

	return
}

func getBasePackageName() string {
	goMod, err := ioutil.ReadFile("go.mod")
	if err == nil {
		lines := strings.Split(string(goMod), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "module ") {
				return line[7:]
			}
		}
	}

	// If there is no go.mod (or at least we couldn't pull the information we
	// need from it) we have to just use the directory name.
	//
	// TODO(elliotchance): We could compare the $GOPATH with the current
	//  directory maybe?
	cwd, err := os.Getwd()
	if err == nil {
		return path.Base(cwd)
	}

	panic("could not determine base package (no go.mod in current directory)")
}
