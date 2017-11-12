package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
	"path/filepath"
	"strconv"
	"bytes"

	"io/ioutil"
	"text/template"

	jp "github.com/AndiBrunner/gte/jsonpath"
	"encoding/json"
	jme "github.com/AndiBrunner/go-jmespath"
)

type sliceVar []string
type hostFlagsVar []string

type Context struct {
}

type HttpHeader struct {
	name  string
	value string
}

func (c *Context) Env() map[string]string {
	env := make(map[string]string)
	for _, i := range os.Environ() {
		sep := strings.Index(i, "=")
		env[i[0:sep]] = i[sep+1:]
	}
	return env
}

var (
	buildVersion string
	version      bool
	poll         bool
	wg           sync.WaitGroup

	templatesFlag     sliceVar
	templateDirsFlag  sliceVar
	stdoutTailFlag    sliceVar
	stderrTailFlag    sliceVar
	headersFlag       sliceVar
	delimsFlag        string
	delims            []string
	headers           []HttpHeader
	urls              []url.URL
	waitFlag          hostFlagsVar
	waitRetryInterval time.Duration
	waitTimeoutFlag   time.Duration
	dependencyChan    chan struct{}
	noOverwriteFlag   bool

	ctx    context.Context
	cancel context.CancelFunc
)

func (i *hostFlagsVar) String() string {
	return fmt.Sprint(*i)
}

func (i *hostFlagsVar) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (s *sliceVar) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *sliceVar) String() string {
	return strings.Join(*s, ",")
}

func usage() {
	println(`Usage: gte [options] [command]

Utility to simplify running applications in docker containers

Options:`)
	flag.PrintDefaults()

	println(`
Arguments:
  command - command to be executed
  `)

	println(`Examples:
`)
	println(`   Generate /etc/nginx/nginx.conf using nginx.tmpl as a template.`)
	println(`
   gte -template nginx.tmpl:/etc/nginx/nginx.conf
	`)

	println(`For more information, see https://github.com/andibrunner/gte`)
}

func main() {

	flag.BoolVar(&version, "version", false, "show version")

	flag.Var(&templatesFlag, "template", "Template (/template:/dest). Can be passed multiple times. Does also support directories")
	flag.BoolVar(&noOverwriteFlag, "no-overwrite", false, "Do not overwrite destination file if it already exists.")
	flag.StringVar(&delimsFlag, "delims", "", `template tag delimiters. default "{{":"}}" `)

	flag.Usage = usage
	flag.Parse()

	if version {
		fmt.Println(buildVersion)
		return
	}

	if flag.NArg() == 0 && flag.NFlag() == 0 {
		usage()
		os.Exit(1)
	}

	if delimsFlag != "" {
		delims = strings.Split(delimsFlag, ":")
		if len(delims) != 2 {
			log.Fatalf("bad delimiters argument: %s. expected \"left:right\"", delimsFlag)
		}
	}

	for _, host := range waitFlag {
		u, err := url.Parse(host)
		if err != nil {
			log.Fatalf("bad hostname provided: %s. %s", host, err.Error())
		}
		urls = append(urls, *u)
	}

	for _, h := range headersFlag {
		//validate headers need -wait options
		if len(waitFlag) == 0 {
			log.Fatalf("-wait-http-header \"%s\" provided with no -wait option", h)
		}

		const errMsg = "bad HTTP Headers argument: %s. expected \"headerName: headerValue\""
		if strings.Contains(h, ":") {
			parts := strings.Split(h, ":")
			if len(parts) != 2 {
				log.Fatalf(errMsg, headersFlag)
			}
			headers = append(headers, HttpHeader{name: strings.TrimSpace(parts[0]), value: strings.TrimSpace(parts[1])})
		} else {
			log.Fatalf(errMsg, headersFlag)
		}

	}

	for _, t := range templatesFlag {
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

func jsonQuery2(jsonString string, jsonQuery string) (interface{}, error) {

	var jsonData interface{}
	var jsonBytes = []byte(jsonString)
	err := json.Unmarshal(jsonBytes, &jsonData)

	j := jp.New("JQ")
	j.AllowMissingKeys(false) //allowMissingKeys)

	err = j.Parse(jsonQuery)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = j.Execute(buf, jsonData)
	if err != nil {
		return nil, err
	}
	println(buf.String())
	return buf, nil
}

func jsonQuery(jsonString string, jsonQuery string) (interface{}, error) {

	var jsonData interface{}
	var jsonBytes = []byte(jsonString)

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

	toJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	println(string(toJSON))
	return string(toJSON), nil
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
		"contains":  contains,
		"exists":    exists,
		"split":     strings.Split,
		"replace":   strings.Replace,
		"default":   defaultValue,
		"parseUrl":  parseUrl,
		"atoi":      strconv.Atoi,
		"add":       add,
		"isTrue":    isTrue,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"jsonQuery": jsonQuery,
		"loop":      loop,
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
	if _, err := os.Stat(destPath); err == nil && noOverwriteFlag {
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

/*	if fi, err := os.Stat(destPath); err == nil {
		if err := dest.Chmod(fi.Mode()); err != nil {
			log.Fatalf("unable to chmod temp file: %s\n", err)
		}
		//		if err := dest.Chown(int(fi.Sys().(*syscall.Stat_t).Uid), int(fi.Sys().(*syscall.Stat_t).Gid)); err != nil {
		//			log.Fatalf("unable to chown temp file: %s\n", err)
		//		}
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

