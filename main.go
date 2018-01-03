package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
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

type Command struct {
	Name          string
	Arity         int
	Sflags        string
	FirstKeyIndex uint
	Return        string
}

func commandsFromCSource() (map[string]*Command, error) {
	resp, err := http.Get("https://github.com/antirez/redis/raw/unstable/src/server.c")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// {"get",getCommand,2,"rF",0,NULL,1,1,1,0,0},
	command := regexp.MustCompile(`\s+\{"(\w+)",\w+,(-?\d+),"(\w+)",\d+,\w+,(\d+),[0-9,]*\},?`)
	scanner := bufio.NewScanner(resp.Body)
	cmds := make(map[string]*Command, 0)
	for scanner.Scan() {
		s := command.FindStringSubmatch(scanner.Text())
		if len(s) > 0 {
			arity, err := strconv.Atoi(s[2])
			if err != nil {
				return nil, err
			}
			first, err := strconv.Atoi(s[4])
			if err != nil {
				return nil, err
			}
			cmd := &Command{
				Name:          s[1],
				Arity:         arity,
				Sflags:        s[3],
				FirstKeyIndex: uint(first),
			}
			cmds[cmd.Name] = cmd
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cmds, nil
}

func main() {
	commands, err := commandsFromCSource()
	if err != nil {
		panic(err)
	}
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
					cmd := strings.ToLower(fn.Name.Name)
					if c, ok := commands[cmd]; ok {
						c.Return = rn
					}
				}
			}
		}
	}

	for _, cmd := range commands {
		fmt.Println(cmd)
	}
}
