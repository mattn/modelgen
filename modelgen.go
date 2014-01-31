package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"regexp"
	"strings"
)

const doNotEdit = "//-- DO_NOT_EDIT --\n"

var pkg = flag.String("pkg", "", "Package Name")
var dbi = flag.String("dbi", "sqlite3", "Database Driver (sqlite3/pq/mysql)")
var tag = flag.String("tag", "db", "Database Tag")

var dbiMap = map[string]string{
	"sqlite3": "github.com/mattn/go-sqlite3",
	"pq":      "github.com/lib/pq",
	"mysql":   "github.com/go-sql-driver/mysql",
}

var typeMap = map[string]string{
	"int":     "int64",
	"integer": "int64",
	"number":  "int64",
	"float":   "float64",
	"double":  "float64",
	"float64": "float64",
	"string":  "string",
	"text":    "string",
	"date":    "time.Time",
	"time":    "time.Time",
}

var re = regexp.MustCompile("[0-9A-Za-z]+")

func CamelCase(s string) string {
	b := re.FindAll([]byte(s), -1)
	for i, c := range b {
		b[i] = bytes.Title(c)
	}
	return string(bytes.Join(b, nil))
}

func main() {
	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}
	if _, ok := dbiMap[*dbi]; !ok {
		flag.Usage()
		os.Exit(1)
	}
	name := flag.Arg(0)

	code := "type " + CamelCase(name) + " struct {\n"
	code += "\tId int64 `" + *tag + ":\"id\"`\n"
	hasTime := false
	for _, arg := range flag.Args()[1:] {
		token := strings.Split(arg, ":")
		if len(token) == 2 {
			if typ, ok := typeMap[token[1]]; ok {
				field := strings.Title(CamelCase(token[0]))
				code += "\t" + field + "\t" + typ + " `" + *tag + ":\"" + token[0] + "\"`\n"
				if typ == "time.Time" {
					hasTime = true
				}
			}
		}
	}
	code += "}"

	hasMain := false
	p := "main"
	if *pkg != "" {
		p = *pkg
		if p == "main" {
			hasMain = true
		}
	}
	out := "package " + p + "\n\nimport (\n\t_ \"" + dbiMap[*dbi] + "\""
	if hasMain {
		out += "\n\t\"database/sql\""
		out += "\n\t\"log\""
	}
	if hasTime {
		out += "\n\t\"time\""
	}
	out += "\n)\n\n"
	if *pkg == "" {
		out += doNotEdit
	}
	out += code

	if hasMain {
		out += "\n\nfunc main() {\n"
		out += "\tconn, err := sql.Open(\"" + *dbi + "\", \"...\")\n"
		out += "\tif err != nil {\n"
		out += "\t\tlog.Fatal(err)\n"
		out += "\t}\n"
		out += "\tdefer conn.Close()\n"
		out += "}\n"
	}

	var buf bytes.Buffer
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name+".go", out, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	err = (&printer.Config{
		Mode:     printer.UseSpaces | printer.TabIndent,
		Tabwidth: 8,
	}).Fprint(&buf, fset, file)
	if err != nil {
		log.Fatal(err)
	}
	out = buf.String()
	if *pkg == "" {
		if pos := strings.Index(out, doNotEdit); pos > 0 {
			out = out[pos+len(doNotEdit):]
		}
	}
	fmt.Print(out)
}
