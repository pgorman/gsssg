// Glog is a very, very simple static site generator.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

type Page struct {
	File     string
	Body     string
	Title    string
	Date     time.Time
	firmDate bool
	Hashtags string // TODO Change to a slice?
}

func main() {
	debug := flag.Bool("d", false, "Write debug info to STDOUT.")
	outdir := flag.String("o", "", "Specify an output directory for .html files (i.e., instead of the input directory).")
	fglob := flag.String("g", "*.txt", "Specify the file glob pattern of input files.")
	utc := flag.Bool("u", false, "For dates with unknown time zones, assume UTC rather than local time.")
	tmpldir := flag.String("t", "", "Specify the directory that contains template files (defaults to input directory).")
	flag.Parse()

	var inDir, outDir string
	var err error
	switch len(flag.Args()) {
	case 0:
		inDir, err = os.Getwd()
		if err != nil {
			log.Fatal("Couldn't get the current directory for input: ", err)
		}
		outDir = inDir
	case 1:
		inDir = flag.Args()[0]
		if _, err := os.Stat(inDir); os.IsNotExist(err) {
			log.Fatal("Input directory error: ", err)
		}
		outDir = inDir
	default:
		log.Fatalf("One or zero arguments required, but %d supplied.", len(flag.Args()))
	}
	if *outdir != "" {
		if _, err := os.Stat(*outdir); os.IsNotExist(err) {
			log.Fatal("Output directory error: ", err)
		}
		outDir = *outdir
	}

	var tmpl *template.Template
	if *tmpldir == "" {
		*tmpldir = inDir
	}
	if _, err := os.Stat(path.Join(*tmpldir, "page.tmpl")); os.IsNotExist(err) {
		tmpl, err = template.New("").Parse(`<!DOCTYPE html>
		<html lang="en-us">
		<head>
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
		<title>{{.Title}}</title>
		</head>
		<body>{{.Body}}</body>
		</html>`)
	} else {
		tmpl, err = template.ParseFiles(path.Join(*tmpldir, "page.tmpl"))
		if err != nil {
			log.Fatal(err)
		}
	}

	inFiles, err := filepath.Glob(path.Join(inDir, *fglob))
	if err != nil {
		log.Fatal(err)
	}

	reTitle := regexp.MustCompile(`\s*#*\s+\w+\s*#*\s*`)
	reHashtags := regexp.MustCompile(`(\s*#\w+,?\s*)+`)
	// "Sat Dec 31 09:18:57 EST 2016" or "Sun Jan  1 07:56:01 EST 2017" i.e. time.UnixDate
	reDate1 := regexp.MustCompile(`\s*[MTWFS][ouehrau][neduitn] [JFMASOND][aepuco][nbrylgptvc]\s{1,2}\d{1,2} [0-2]\d:[0-5][0-9]:[0-5][0-9] [A-Z]{3} \d{4}\s*`)
	// "20161231" or "20161231091857" or "20170101" or "20170101075601"
	reDate2 := regexp.MustCompile(`\d{4}[01]\d[0-3]\d([01]\d[0-5]\d[0-5]\d)?`)

	// Process each input file:
	for _, fi := range inFiles {
		var p Page
		input, err := ioutil.ReadFile(fi)
		if err != nil {
			log.Fatal(err)
		}
		p.File = strings.TrimSuffix(path.Base(fi), path.Ext(fi))
		p.Body = string(blackfriday.MarkdownCommon(input))

		newlines := func(c rune) bool {
			return strings.ContainsRune("\u000A\u000B\u000C\u000D\u0085\u2028\u2029", c)
		}

		// Process each line in this input file, looking for date, title, and hashtags.
		for _, v := range bytes.FieldsFunc(input, newlines) {
			if reTitle.Match(v) && p.Title == "" {
				p.Title = string(bytes.Trim(v, " #"))
			}
			if reHashtags.Match(v) {
				p.Hashtags = string(v)
			}
			if reDate1.Match(v) && p.Date.IsZero() {
				p.Date, err = time.Parse(time.UnixDate, string(v))
				if err != nil {
					log.Println(err, string(v))
				}
				p.firmDate = true
			}
		}

		// If we didn't find a good date or title in the file contents, guess:
		if p.Title == "" {
			p.Title = p.File
		}
		if p.Date.IsZero() && reDate2.MatchString(p.File) {
			d := []byte(p.File)
			for len(d) < 14 {
				d = append(d, "0"...)
			}
			if *utc {
				p.Date, err = time.Parse("20060102150405", string(d))
			} else {
				p.Date, err = time.ParseInLocation("20060102150405", string(d), time.Now().Location())
			}
			if err != nil {
				log.Println(err)
			}
			p.firmDate = true
		}
		if p.Date.IsZero() {
			st, err := os.Stat(fi)
			if err != nil {
				log.Println(err, fi)
			}
			p.Date = st.ModTime()
		}

		if *debug {
			fmt.Fprintf(os.Stderr, "FILE\t%v\n", p.File)
			fmt.Fprintf(os.Stderr, "TITLE\t%v\n", p.Title)
			fmt.Fprintf(os.Stderr, "DATE\t%v\n", p.Date)
			fmt.Fprintf(os.Stderr, "TAGS\t%v\n", p.Hashtags)
			fmt.Fprintf(os.Stderr, "OUT\t%v\n\n", path.Join(outDir, strings.Join([]string{p.File, ".html"}, "")))
			// err = tmpl.Execute(os.Stdout, p)
			// err = tmpl.Execute(ioutil.Discard, p)
		}

		fo, err := os.Create(path.Join(outDir, strings.Join([]string{p.File, ".html"}, "")))
		if err != nil {
			log.Println(err)
		}
		err = tmpl.Execute(fo, p)
	}
}
