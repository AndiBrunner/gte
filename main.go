package main

import (
	"encoding/json"
	"fmt"
	jme "github.com/AndiBrunner/go-jmespath"
	yml "github.com/AndiBrunner/yaml"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

type Context struct {
}

func (c *Context) Env() map[string]string {
	env := make(map[string]string)
	for _, i := range os.Environ() {
		sep := strings.Index(i, "=")
		env[i[0:sep]] = i[sep+1:]
	}
	return env
}

type argFlags struct {
	delims      string
	noOverwrite bool
}

var (
	Version   = "release-1.0.0"
	BuildTime = "2017-11-13 UTC"
	ArgFlags  = argFlags{delims: "", noOverwrite: false}
	delims    []string
)

func usage() {
	println(`Usage: gte [options] template:dest

Go template engine

Options:`)
	println(`  -n --no-overwrite
        Do not overwrite destination file if it already exists.`)
	println(`  -d --delims
        template tag delimiters. default "{{":"}}" `)

	println(`
Arguments:
  template:dest - Template (/template:/dest). Can be passed multiple times. Does also support directories.
  `)

	println(`Examples:
`)
	println(`   Generate /etc/nginx/nginx.conf using nginx.tmpl as a template.`)
	println(`
   gte nginx.tmpl:/etc/nginx/nginx.conf
	`)

	println(`For more information, see https://github.com/andibrunner/gte`)
}

func main() {

	var parsedArgs []string

	//eval args
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--no-overwrite":
			ArgFlags.noOverwrite = true
		case "-d", "--delims":
			if (i + 1) < len(args) {
				i++
				ArgFlags.delims = args[i]
			} else {
				usage()
				os.Exit(0)
			}
		case "-v", "--version":
			fmt.Println("Version:", Version)
			os.Exit(0)
		case "":
			usage()
			os.Exit(0)
		default:
			d := args[i]
			if d[0:1] == "-" {
				usage()
				os.Exit(0)
			}
			parsedArgs = append(parsedArgs, args[i])
		}
	}

	if len(parsedArgs) == 0 {
		usage()
		os.Exit(0)
	}

	if ArgFlags.delims != "" {
		delims = strings.Split(ArgFlags.delims, ":")
		if len(delims) != 2 {
			log.Fatalf("bad delimiters argument: %s. expected \"left:right\"", ArgFlags.delims)
		}
	}

	for _, t := range parsedArgs {
		template, dest := t, ""
		if strings.Contains(t, ":") {
			parts := strings.Split(t, ":")
			if len(parts) != 2 {
				log.Fatalf("bad template argument: %s. expected \"/template:/dest\"", t)
			}
			template, dest = parts[0], parts[1]
		}

		fi, err := os.Stat(template)
		if err != nil {
			log.Fatalf("unable to stat %s, error: %s", template, err)
		}
		if fi.IsDir() {
			generateDir(template, dest)
		} else {
			generateFile(template, dest)
		}
	}

}
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func contains(item map[string]string, key string) bool {
	if _, ok := item[key]; ok {
		return true
	}
	return false
}

func defaultValue(args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("default called with no values!")
	}

	if len(args) > 0 {
		if args[0] != nil {
			return args[0].(string), nil
		}
	}

	if len(args) > 1 {
		if args[1] == nil {
			return "", fmt.Errorf("default called with nil default value!")
		}

		if _, ok := args[1].(string); !ok {
			return "", fmt.Errorf("default is not a string value. hint: surround it w/ double quotes.")
		}

		return args[1].(string), nil
	}

	return "", fmt.Errorf("default called with no default value")
}

func parseUrl(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		log.Fatalf("unable to parse url %s: %s", rawurl, err)
	}
	return u
}

func add(arg1, arg2 int) int {
	return arg1 + arg2
}

func isTrue(s string) bool {
	b, err := strconv.ParseBool(strings.ToLower(s))
	if err == nil {
		return b
	}
	return false
}

func jsonQuery(jsonString string, jsonQueryFull string) (interface{}, error) {

	var jsonData interface{}
	var jsonBytes = []byte(jsonString)
	var jsonQuery string
	var formatIndent bool = false
	var formatYaml bool = false
	var jsonParts []string
	var jsonParameter string
	var jsonIndent int

	if jsonQueryFull[0:1] == "-" {
		jsonParts = strings.SplitN(jsonQueryFull, " ", 2)
		if len(jsonParts) == 2 {
			jsonQuery = jsonParts[1]
			jsonParameter = jsonParts[0][1:]

			switch jsonParameter[0:1] {
			case "i":
				var err error
				formatIndent = true
				jsonIndent, err = strconv.Atoi(jsonParameter[1:])
				if err != nil {
					jsonIndent = 2
				}
			case "y":
				formatYaml = true
			default:

			}
		}
	} else {
		jsonQuery = jsonQueryFull
	}

	parser := jme.NewParser()

	_, err := parser.Parse(jsonQuery)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonBytes, &jsonData)
	if err != nil {
		return nil, err
	}

	result, err := jme.Search(jsonQuery, jsonData)
	if err != nil {
		return nil, err
	}

	var output []byte
	if formatIndent {
		output, err = json.MarshalIndent(result, "", strings.Repeat(" ", jsonIndent))
	} else if formatYaml {
		t, _ := json.Marshal(result)
		output, err = yml.JSONToYAML(t)
	} else {
		output, err = json.Marshal(result)
	}
	if err != nil {
		return nil, err
	}

	return string(output), nil
}

func loop(args ...int) (<-chan int, error) {
	var start, stop, step int
	switch len(args) {
	case 1:
		start, stop, step = 0, args[0], 1
	case 2:
		start, stop, step = args[0], args[1], 1
	case 3:
		start, stop, step = args[0], args[1], args[2]
	default:
		return nil, fmt.Errorf("wrong number of arguments, expected 1-3"+
			", but got %d", len(args))
	}

	c := make(chan int)
	go func() {
		for i := start; i < stop; i += step {
			c <- i
		}
		close(c)
	}()
	return c, nil
}

func generateFile(templatePath, destPath string) bool {
	tmpl := template.New(filepath.Base(templatePath)).Funcs(template.FuncMap{
		"contains":   contains,
		"exists":     exists,
		"split":      strings.Split,
		"replace":    strings.Replace,
		"default":    defaultValue,
		"parseUrl":   parseUrl,
		"atoi":       strconv.Atoi,
		"add":        add,
		"isTrue":     isTrue,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"jsonQuery":  jsonQuery,
		"loop":       loop,
		"trimSuffix": strings.TrimSuffix,
	})

	if len(delims) > 0 {
		tmpl = tmpl.Delims(delims[0], delims[1])
	}
	tmpl, err := tmpl.ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("unable to parse template: %s", err)
	}

	// Don't overwrite destination file if it exists and no-overwrite flag passed
	if _, err := os.Stat(destPath); err == nil && ArgFlags.noOverwrite {
		return false
	}

	dest := os.Stdout
	if destPath != "" {
		dest, err = os.Create(destPath)
		if err != nil {
			log.Fatalf("unable to create %s", err)
		}
		defer dest.Close()
	}

	err = tmpl.ExecuteTemplate(dest, filepath.Base(templatePath), &Context{})
	if err != nil {
		log.Fatalf("template error: %s\n", err)
	}

	/*
		if fi, err := os.Stat(destPath); err == nil {
			if err := dest.Chmod(fi.Mode()); err != nil {
				log.Fatalf("unable to chmod temp file: %s\n", err)
			}
			if err := dest.Chown(int(fi.Sys().(*syscall.Stat_t).Uid), int(fi.Sys().(*syscall.Stat_t).Gid)); err != nil {
				log.Fatalf("unable to chown temp file: %s\n", err)
			}
		}
	*/
	return true
}

func generateDir(templateDir, destDir string) bool {
	if destDir != "" {
		fiDest, err := os.Stat(destDir)
		if err != nil {
			log.Fatalf("unable to stat %s, error: %s", destDir, err)
		}
		if !fiDest.IsDir() {
			log.Fatalf("if template is a directory, dest must also be a directory (or stdout)")
		}
	}

	files, err := ioutil.ReadDir(templateDir)
	if err != nil {
		log.Fatalf("bad directory: %s, error: %s", templateDir, err)
	}

	for _, file := range files {
		if destDir == "" {
			generateFile(filepath.Join(templateDir, file.Name()), "")
		} else {
			generateFile(filepath.Join(templateDir, file.Name()), filepath.Join(destDir, file.Name()))
		}
	}

	return true
}
