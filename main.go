package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"net/http"
	"strings"
)

func fetchSource() (*ast.File, error) {
	resp, err := http.Get("https://raw.githubusercontent.com/go-redis/redis/master/commands.go")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, "", body, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func main() {
	node, err := fetchSource()
	if err != nil {
		panic(err)
	}

	for _, f := range node.Decls {
		fn, ok := f.(*ast.FuncDecl)
		if !ok {
			continue
		}
		r := fn.Type.Results
		if r != nil {
			if len(r.List) == 1 {
				ret := r.List[0].Type
				var rn string
				ast.Inspect(ret, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if ok {
						rn = id.Name
						return false
					}
					return true
				})
				if strings.HasSuffix(rn, "Cmd") {
					fmt.Println(fn.Name.Name, rn)
				}
			}
		}
	}

}
