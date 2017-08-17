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
	Hashtags string // TODO Change to a slice?
}

func main() {
	outdir := flag.String("o", "", "Specify an output directory for .html files (i.e., not the input directory).")
	fglob := flag.String("g", "*.txt", "Specify the file glob pattern of input files.")
	utc := flag.Bool("u", false, "When unspecified, assume UTC for dates rather than local time.")
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
	if _, err := os.Stat(path.Join(inDir, "template.html")); os.IsNotExist(err) {
		tmpl, err = template.New("").Parse(`<!DOCTYPE html>
		<html lang="en-us">
		<head>
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
		<title>{{.Title}}</title>
		</head>
		<body>{{.Body}}</body>
		</html>`)
	} else {
		tmpl, err = template.ParseFiles(path.Join(inDir, "template.html"))
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
	for _, f := range inFiles {
		var p Page
		input, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		p.File = strings.TrimSuffix(path.Base(f), path.Ext(f))
		p.Body = string(blackfriday.MarkdownCommon(input))
		// TODO File output.
		// err = tmpl.Execute(os.Stdout, p)
		err = tmpl.Execute(ioutil.Discard, p)

		newlines := func(c rune) bool {
			return strings.ContainsRune("\u000A\u000B\u000C\u000D\u0085\u2028\u2029", c)
		}

		// Process each line in this input file, looking for date, title, and hashtags:
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
			}
		}

		// If we didn't find good a good date or title in the file contents, guess:
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
		}
		if p.Date.IsZero() {
			st, err := os.Stat(f)
			if err != nil {
				log.Println(err, f)
			}
			p.Date = st.ModTime()
		}

		fmt.Println("FILE", p.File)
		fmt.Println("TITLE", p.Title)
		fmt.Println("DATE", p.Date)
		fmt.Println("TAGS", p.Hashtags, "\n")
	}

	fmt.Println(outDir)
}
