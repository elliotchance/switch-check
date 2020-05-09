package main

import (
	"go/ast"
	"go/token"
	"sort"
)

func getSwitchesFromFile(fset *token.FileSet, node *ast.File, currentPackage string, pkgs map[string]string) map[string][]string {
	found := map[string][]string{}

	ast.Inspect(node, func(n ast.Node) bool {
		if n, ok := n.(*ast.SwitchStmt); ok {
			pos := fset.Position(n.Pos()).String()

			hasDefault := false
			for _, stmt := range n.Body.List {
				if caseClauseValues, ok := stmt.(*ast.CaseClause); ok {
					if len(caseClauseValues.List) == 0 {
						hasDefault = true
					}
					for _, caseValueValue := range caseClauseValues.List {
						found[pos] = append(found[pos], getTypeName(caseValueValue, currentPackage, pkgs))
					}
				}
			}

			// If there is a default case we should not include this switch
			// statement.
			if hasDefault {
				delete(found, pos)
			}
		}

		return true
	})

	return found
}

func findMissingValues(allValues map[string]Value, values []string) []string {
	found := false
	for _, value := range values {
		_, found = allValues[value]
		if found {
			break
		}
	}

	// If none of the input values (which were the case expressions) match any
	// of the known enum values this entire switch statement can be ignored.
	if !found {
		return nil
	}

	// Otherwise we assume that all values should appear.
	var missing []string
	ty := allValues[values[0]]

	// If Type is empty then the enum type is not known. We shouldn't add any
	// missing values in this case. Otherwise it would add every constant that
	// was unresolved.
	if ty.Type != "" {
		for name, value := range allValues {
			if value.Type == ty.Type {
				missing = append(missing, name)
			}
		}
	}

next:
	for i, have := range missing {
		for _, want := range values {
			if have == want {
				missing = append(missing[:i], missing[i+1:]...)
				goto next
			}
		}
	}

	sort.Strings(missing)

	return missing
}
