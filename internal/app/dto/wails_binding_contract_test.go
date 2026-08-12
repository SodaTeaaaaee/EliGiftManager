package dto

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Wails cannot infer a TypeScript representation for time.Time. Every time
// field in an exposed DTO must declare its wire type explicitly so generated
// bindings remain string-typed and binding generation stays warning-free.
func TestWailsTimeFieldsDeclareStringType(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(fileSet, ".", func(info os.FileInfo) bool {
		return strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse DTO package: %v", err)
	}

	pkg, ok := packages["dto"]
	if !ok {
		t.Fatal("DTO package was not found")
	}

	var violations []string
	for _, file := range pkg.Files {
		timeAliases := importedTimeAliases(file)
		if len(timeAliases) == 0 {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			structure, ok := node.(*ast.StructType)
			if !ok {
				return true
			}

			for _, field := range structure.Fields.List {
				if !containsTimeType(field.Type, timeAliases) {
					continue
				}
				if field.Tag == nil || structTagValue(field.Tag.Value, "ts_type") != "string" {
					position := fileSet.Position(field.Pos())
					violations = append(violations, position.String())
				}
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("time.Time DTO fields must include ts_type:\"string\":\n%s", strings.Join(violations, "\n"))
	}
}

func importedTimeAliases(file *ast.File) map[string]struct{} {
	aliases := make(map[string]struct{})
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != "time" {
			continue
		}

		alias := "time"
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		aliases[alias] = struct{}{}
	}
	return aliases
}

func containsTimeType(expression ast.Expr, aliases map[string]struct{}) bool {
	switch value := expression.(type) {
	case *ast.SelectorExpr:
		identifier, ok := value.X.(*ast.Ident)
		if !ok || value.Sel.Name != "Time" {
			return false
		}
		_, ok = aliases[identifier.Name]
		return ok
	case *ast.StarExpr:
		return containsTimeType(value.X, aliases)
	case *ast.ParenExpr:
		return containsTimeType(value.X, aliases)
	}
	return false
}

func structTagValue(literal string, key string) string {
	raw, err := strconv.Unquote(literal)
	if err != nil {
		return ""
	}
	return reflect.StructTag(raw).Get(key)
}
