package main

import (
	"app/bcrypt"
	"bufio"
	"bytes"
	"fmt"
	"github.com/xwb1989/sqlparser"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
)

var version string
var commit string
var config = kingpin.Flag("config", "Path to YAML masking rule config file.").Required().Short('c').String()
var cost = kingpin.Flag("cost", fmt.Sprintf("bcrypt cost. Min: %d, Max: %d", bcrypt.MinCost, bcrypt.MaxCost)).Default(fmt.Sprintf("%d", bcrypt.DefaultCost)).Int()

type TemplateValue struct {
	Raw  string
	Salt string
}

func (v TemplateValue) Hashed() string {
	result, err := bcrypt.Bcrypt([]byte(v.Raw), *cost, []byte(v.Salt))
	if err != nil {
		log.Fatal(err)
	}
	return string(result)
}

func (v TemplateValue) First(n int) string {
	rawRune := []rune(v.Raw)

	if len(rawRune) < n {
		return v.Raw
	}

	return string(rawRune[0:n])
}

func (v TemplateValue) Last(n int) string {
	rawRune := []rune(v.Raw)

	if len(rawRune) < n {
		return v.Raw
	}

	return string(rawRune[len(rawRune)-n:])
}

func loadConfig(config string) map[string]map[string]*template.Template {
	data, err := ioutil.ReadFile(config)
	if err != nil {
		log.Fatal(err)
	}

	configData := make(map[string]map[string]string)
	err = yaml.Unmarshal([]byte(data), &configData)
	if err != nil {
		log.Fatal(err)
	}

	templates := make(map[string]map[string]*template.Template)

	for tableName, colNames := range configData {
		templates[tableName] = make(map[string]*template.Template)
		for colName, colTemplate := range colNames {
			tmpl, err := template.New(fmt.Sprintf("%s-%s", tableName, colName)).Parse(colTemplate)
			if err != nil {
				log.Fatal(err)
			}
			templates[tableName][colName] = tmpl
		}
	}

	return templates
}

func handleLine(line string, configData map[string]map[string]*template.Template, salt []byte) string {
	if !strings.HasPrefix(line, "INSERT ") {
		return line
	}

	tree, err := sqlparser.Parse(strings.TrimSuffix(line, ";"))

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return line
	}

	insert := tree.(*sqlparser.Insert)
	tableName := string(insert.Table.Name)
	fmt.Fprintf(os.Stderr, "Masking `%s`...\n", tableName)
	_, present := configData[tableName]

	if !present {
		return line
	}

	columnNames := make([]string, len(insert.Columns))
	for i, column := range insert.Columns {
		columnNames[i] = string(column.(*sqlparser.NonStarExpr).Expr.(*sqlparser.ColName).Name)
	}

	for i, row := range insert.Rows.(sqlparser.Values) {
		for j, col := range row.(sqlparser.ValTuple) {
			tmpl, present := configData[tableName][columnNames[j]]
			if !present {
				continue
			}
			var buf bytes.Buffer

			switch val := col.(type) {
			case sqlparser.StrVal:
				err = tmpl.Execute(&buf, &TemplateValue{Raw: string(val), Salt: string(salt)})
				if err != nil {
					log.Fatal(err)
				}
				insert.Rows.(sqlparser.Values)[i].(sqlparser.ValTuple)[j] = sqlparser.StrVal(buf.Bytes())
			case sqlparser.NumVal:
				err = tmpl.Execute(&buf, &TemplateValue{Raw: string(val), Salt: string(salt)})
				if err != nil {
					log.Fatal(err)
				}
				insert.Rows.(sqlparser.Values)[i].(sqlparser.ValTuple)[j] = sqlparser.NumVal(buf.Bytes())
			}
		}
	}
	return fmt.Sprintf("%s;", sqlparser.String(insert))
}

func main() {
	kingpin.Version(fmt.Sprintf("%s (%s)", version, commit))
	kingpin.Parse()

	configData := loadConfig(*config)

	salt, err := bcrypt.GenerateSalt()
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	// depends on max_allowed_packet
	// maximum is 1G
	// https://dev.mysql.com/doc/refman/5.7/en/mysqldump.html
	const maxCapacity = 1000 * 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		fmt.Println(handleLine(strings.TrimSuffix(scanner.Text(), "\n"), configData, salt))
	}

	err = scanner.Err()
	if err != nil {
		log.Fatal(err)
	}
}
