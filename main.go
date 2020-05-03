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
	"sort"
	"strconv"
	"strings"
)

type Value struct {
	Type  string
	Value string
}

func resolveValue(value ast.Expr, ty ast.Expr, found map[string]Value) Value {
	switch v := value.(type) {
	case *ast.BasicLit:
		return Value{Type: fmt.Sprintf("%v", ty), Value: v.Value}

	case *ast.CallExpr:
		// If there are no args this means it must be a call to a function.
		// Which we don't support.
		if len(v.Args) != 1 {
			return Value{Type: "<nil>"}
		}

		v2 := resolveValue(v.Args[0], ty, found)
		return Value{Type: fmt.Sprintf("%v", v.Fun), Value: v2.Value}

	case *ast.Ident:
		f, ok := found[v.Name]
		if ok {
			return f
		}

		return Value{Type: fmt.Sprintf("%v", ty), Value: "0"}

	case *ast.UnaryExpr:
		v2 := resolveValue(v.X, ty, found)
		return Value{Type: fmt.Sprintf("%v", ty), Value: "-" + v2.Value}
	}

	// Any other case we encounter we will assume it's too complicated to
	// understand.
	return Value{Type: "<nil>"}
}

func appendEnumValues(in map[string]Value, decl *ast.GenDecl) {
	iotaValue := 1
	var iotaType ast.Expr

	for _, spec := range decl.Specs {
		if valueSpec, ok := spec.(*ast.ValueSpec); ok {
			// Catch iota so we can give future values to it. However, be
			// careful that not all empty types mean an iota. Without a previous
			// iota it would just be no type (the Go compiler would resolve the
			// basic lit of whatever type in this case).
			if len(valueSpec.Values) > 0 {
				if v, ok := valueSpec.Values[0].(*ast.Ident); valueSpec.Type != nil && ok && v.Name == "iota" {
					iotaType = valueSpec.Type
				}
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

				in[name.String()] = resolveValue(valueSpec.Values[i], ty, in)
			}
		}
	}
}

func valuesToEnums(values map[string]Value) map[string][]string {
	r := map[string][]string{}
	for name, value := range values {
		// A nil type means that its a basic literal, or perhaps something more
		// complex. Either way we could not resolve it's custom type so we will
		// ignore it as an enum value.
		if value.Type != "<nil>" {
			r[value.Type] = append(r[value.Type], name)
		}
	}

	// Keep enum values sorted by name.
	for ty := range r {
		sort.Strings(r[ty])
	}

	return r
}

func getEnumValuesFromFile(path string) map[string]Value {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		log.Panic(err)
	}

	found := map[string]Value{}

	ast.Inspect(node, func(n ast.Node) bool {
		// GenDecl can appear anywhere, make sure we do not traverse into
		// functions, otherwise local variables would be included in the enum
		// values.
		if _, ok := n.(*ast.FuncDecl); ok {
			return false
		}

		if n, ok := n.(*ast.GenDecl); ok {
			appendEnumValues(found, n)
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
	for _, path := range args {
		out += runPath(path, allValues, allSwitches, flagVerbose)
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
			out += fmt.Sprintln(ty, allEnums[ty])
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

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Fatalln(err)
		return false
	}

	return fileInfo.IsDir()
}

func runPath(path string, allValues map[string]Value, allSwitches map[string][]string, verbose bool) (out string) {
	if isDirectory(path) {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			log.Fatalln(err)
		}

		for _, file := range files {
			subpath := path + "/" + file.Name()
			if isDirectory(subpath) || strings.HasSuffix(subpath, ".go") {
				out += runPath(subpath, allValues, allSwitches, verbose)
			}
		}

		return
	}

	if verbose {
		out += fmt.Sprintln("#", path)
	}

	values := getEnumValuesFromFile(path)
	for a, b := range values {
		allValues[a] = b
	}

	switches := getSwitchesFromFile(path)
	for a, b := range switches {
		allSwitches[a] = append(allSwitches[a], b...)
	}

	return
}
