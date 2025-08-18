package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// This script scans Go source files for methods like (c *Config) Get<Type>E
// and generates:
//   - global Get<Type>E wrappers
//   - Get<Type>Must wrappers
//   - Get<Type> wrappers
// into generated.go

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <project-root>")
		os.Exit(1)
	}
	root := os.Args[1]

	outFile, err := os.Create("generated.go")
	if err != nil {
		fmt.Println("Error creating generated.go:", err)
		os.Exit(1)
	}
	defer outFile.Close()

	// write package header
	fmt.Fprintln(outFile, `// This file is auto generated; DO NOT EDIT IT.`)
	fmt.Fprintln(outFile, "package config")
	fmt.Fprintln(outFile)
	fmt.Fprintln(outFile, `
import "reflect"
    `)

	fset := token.NewFileSet()
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(path) == "generated.go" {
			return nil
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Recv == nil {
				continue
			}

			if !ast.IsExported(fn.Name.Name) {
				continue
			}

			name := fn.Name.Name
			if strings.HasPrefix(name, "Get") && strings.HasSuffix(name, "E") {
				if fn.Type.Results != nil && len(fn.Type.Results.List) >= 1 {
					// take the first result type (before error)
					retExpr := fn.Type.Results.List[0].Type
					retType := exprToString(retExpr)
					baseType := strings.TrimSuffix(strings.TrimPrefix(name, "Get"), "E")
					generateFunctions(outFile, baseType, retType)
				}
				continue
			}

			// Generate default wrapper for non-Getter function
			params := []string{}
			args := []string{}
			for _, p := range fn.Type.Params.List {
				for _, n := range p.Names {
					params = append(params, n.Name+" "+exprToString(p.Type))
					args = append(args, n.Name)
				}
			}
			retTypes := []string{}
			if fn.Type.Results != nil {
				for _, r := range fn.Type.Results.List {
					retTypes = append(retTypes, exprToString(r.Type))
				}
			}

			if fn.Doc != nil {
				for _, c := range fn.Doc.List {
					fmt.Fprintln(outFile, c.Text)
				}
			}
			if len(retTypes) != 0 {
				fmt.Fprintf(outFile, "func %s(%s) (%s) { return Default().%s(%s) }\n\n",
					name, strings.Join(params, ", "), strings.Join(retTypes, ", "), name, strings.Join(args, ", "))
				continue
			}

			fmt.Fprintf(outFile, "func %s(%s) { Default().%s(%s) }\n\n",
				name, strings.Join(params, ", "), name, strings.Join(args, ", "))
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// exprToString returns a string representation of an ast.Expr
func exprToString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	default:
		return "interface{}"
	}
}

func generateFunctions(f *os.File, typeName, retType string) {
	lower := strings.ToLower(typeName)

	fmt.Fprintf(f, "// Get%sE returns the %s value for the key, or error if missing/invalid.\n", typeName, lower)
	fmt.Fprintf(f, "func Get%sE(key string) (%s, error) { return Default().Get%sE(key) }\n\n", typeName, retType, typeName)

	fmt.Fprintf(f, "// Get%sMust returns the %s value for the key. Panics if missing/invalid.\n", typeName, lower)
	fmt.Fprintf(f, "func Get%sMust(key string) %s { return Default().Get%sMust(key) }\n\n", typeName, retType, typeName)

	fmt.Fprintf(f, "// Get%sMust returns the %s value for the key. Panics if missing/invalid.\n", typeName, lower)
	fmt.Fprintf(f, "func (c *Config) Get%sMust(key string) %s {\n", typeName, retType)
	fmt.Fprintf(f, "\treturn Must(c.Get%sE(key))\n", typeName)
	fmt.Fprintf(f, "}\n\n")

	fmt.Fprintf(f, "// Get%s returns the %s value for the key. Returns default if missing/invalid.\n", typeName, lower)
	fmt.Fprintf(f, "func Get%s(key string) %s { return Default().Get%s(key) }\n\n", typeName, retType, typeName)

	fmt.Fprintf(f, "// Get%s returns the %s value for the key. Returns default if missing/invalid.\n", typeName, lower)
	fmt.Fprintf(f, "func (c *Config) Get%s(key string) %s {\n", typeName, retType)
	fmt.Fprintf(f, "\tv, _ := c.Get%sE(key)\n", typeName)
	fmt.Fprintf(f, "\treturn v\n")
	fmt.Fprintf(f, "}\n\n")
}
