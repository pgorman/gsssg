// Glog is a very, very simple static site generator.
package main

import (
	"flag"
	"fmt"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type Page struct {
	File  string
	Body  string
	Title string
}

func main() {
	outdir := flag.String("o", "", "Specify an output directory for .html files (i.e., not the input directory).")
	fglob := flag.String("g", "*.txt", "Specify the file glob pattern of input files. Defaults to '*.txt'.")
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
			panic(err)
		}
	}

	inFiles, err := filepath.Glob(path.Join(inDir, *fglob))
	if err != nil {
		log.Fatal(err)
	}
	var p Page
	for _, f := range inFiles {
		input, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		p.File = path.Base(f)
		p.Body = string(blackfriday.MarkdownCommon(input))
		firstLine := strings.Trim(strings.Split(string(input), "\n")[0], " #")
		if firstLine != "" {
			p.Title = firstLine
		} else {
			p.Title = p.File
		}
		err = tmpl.Execute(os.Stdout, p)
	}
	fmt.Println(outDir)
}
