// Package osexitanalyzer содержит анализатор, который запрещает использовать os.Exit
// внутри функции main() пакета main. Это сделано для повышения управляемости ошибок.
//
// Пример нарушения:
//
//	package main
//
//	import "os"
//
//	func main() {
//		os.Exit(1) // запрещено
//	}
//
// Для использования добавьте Analyzer в multichecker.
package osexitanalyzer

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "osexitcheck",
	Doc:  "Запрещает использование os.Exit в функции main пакета main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Не анализируем, если не main-пакет
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	// Пропускаем анализатор, если файл не из проекта
	skip := true
	for _, f := range pass.Files {
		filename := pass.Fset.File(f.Pos()).Name()
		// Убедись, что анализируем только файлы из твоего проекта
		if strings.Contains(filename, "practicum-shortener-url/") &&
			!strings.Contains(filename, "cmd/staticlint") {
			skip = false
			break
		}
	}
	if skip {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				continue
			}
			// Внутри main-функции ищем вызовы os.Exit
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Exit" {
					return true
				}

				// Проверим, что это os.Exit
				if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == "os" {
					obj := pass.TypesInfo.Uses[ident]
					if pkgName, ok := obj.(*types.PkgName); ok && pkgName.Imported().Path() == "os" {
						pass.Reportf(call.Pos(), "нельзя использовать os.Exit в main.main")
					}
				}

				return true
			})
		}
	}
	return nil, nil
}
