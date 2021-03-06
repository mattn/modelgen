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

var gorp = flag.Bool("gorp", false, "use gorp")
var pkg = flag.String("pkg", "", "package Name")
var dbi = flag.String("dbi", "sqlite3", "database driver (sqlite3/pq/mysql)")
var tag = flag.String("tag", "db", "tag name")

var dialectMap = map[string]string{
	"sqlite3": "SqliteDialect",
	"pq":      "PostgresDialect",
	"mysql":   "MySQLDialect",
}

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
	"bool":    "bool",
	"boolean": "bool",
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

	cname := CamelCase(name)
	code := "type " + cname + " struct {\n"
	tags := ""
	for i, t := range strings.Split(*tag, ",") {
		if i > 0 {
			tags += " "
		}
		tags += t + ":\"id\""
	}
	code += "\tId int64 `" + tags + "`\n"
	hasTime := false
	for _, arg := range flag.Args()[1:] {
		token := strings.Split(arg, ":")
		if len(token) == 2 {
			if typ, ok := typeMap[token[1]]; ok {
				field := strings.Title(CamelCase(token[0]))
				tags = ""
				for i, t := range strings.Split(*tag, ",") {
					if i > 0 {
						tags += " "
					}
					tags += t + ":\"" + token[0] + "\""
				}
				code += "\t" + field + "\t" + typ + " `" + tags + "`\n"
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
	if *gorp {
		out += "\n\t\"github.com/coopernurse/gorp\""
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
		out += fmt.Sprintf(`
func main() {
	conn, err := sql.Open("%s", "...")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
`, *dbi)
		if *gorp {
			out += fmt.Sprintf(`
	dbmap := &gorp.DbMap{Db: conn, Dialect: gorp.%s{}}
    dbmap.AddTableWithName(%s{}, "%ss").SetKeys(true, "id")
    err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		log.Fatal(err)
	}
`, dialectMap[*dbi], cname, name)
		}
		out += `}`

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
