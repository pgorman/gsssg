// Gsssg is a very simple static site generator.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/russross/blackfriday"
)

type Page struct {
	File     string
	Body     string
	Title    string
	Date     time.Time
	firmDate bool
	Link     string
	Hashtags []string
	Next     string
	Prev     string
}

type Feed struct {
	Title string
	Desc  string
	URL   string
	Items []*Page
}

func main() {
	siteDesc := flag.String("d", "", "Description of the site, like 'All the news that's fit to print'. Required to produce RSS feed.")
	debug := flag.Bool("debug", false, "Write debug info to STDERR.")
	fglob := flag.String("g", "*.txt", "Set the file glob pattern of input files.")
	tmpldir := flag.String("l", "", "Set the directory for template files. (default to input directory).")
	outdir := flag.String("o", "", "Set the output directory. (default to the current working directory).")
	pre := flag.Bool("p", false, "Leave input as pre-formatted text; don't process it like Markdown.")
	siteTitle := flag.String("t", "", "Title of site, like 'My Blog'. Required to produce RSS feed.")
	siteURL := flag.String("u", "", "URL of site, like 'https://example.com/blog/'. Required to produce RSS feed.")
	utc := flag.Bool("z", false, "For dates with unknown time zones, assume UTC rather than local time.")
	flag.Parse()

	var err error
	var inDir, outDir string
	var tmpl *template.Template

	switch len(flag.Args()) {
	case 0:
		inDir, err = os.Getwd()
		if err != nil {
			log.Fatal("error using the current directory for input: ", err)
		}
		outDir = inDir
	case 1:
		inDir = flag.Args()[0]
		if _, err := os.Stat(inDir); os.IsNotExist(err) {
			log.Fatal("error using input directory: ", err)
		}
		outDir = inDir
	default:
		log.Fatalf("argument error: one or zero arguments required, but %d supplied", len(flag.Args()))
	}
	if *outdir != "" {
		if _, err := os.Stat(*outdir); os.IsNotExist(err) {
			log.Fatal("output directory error: ", err)
		}
		outDir = *outdir
	}
	if *tmpldir == "" {
		*tmpldir = inDir
	}

	inFiles, err := filepath.Glob(path.Join(inDir, *fglob))
	if err != nil {
		log.Fatal("error globbing input files: ", err)
	}

	Pages := make([]*Page, len(inFiles))

	reTitle := regexp.MustCompile(`\s*#*\s+\w+\s*#*\s*`)
	reHashtags := regexp.MustCompile(`(?:^|\s|,)(#\w+)`)
	// "Sat Dec 31 09:18:57 EST 2016" or "Sun Jan  1 07:56:01 EST 2017" i.e. time.UnixDate
	reDate1 := regexp.MustCompile(`\s*[MTWFS][ouehrau][neduitn] [JFMASOND][aepuco][nbrylgptvc]\s{1,2}\d{1,2} [0-2]\d:[0-5][0-9]:[0-5][0-9] [A-Z]{3} \d{4}\s*`)
	// "20161231" or "20161231091857" or "20170101" or "20170101075601"
	reDate2 := regexp.MustCompile(`\d{4}[01]\d[0-3]\d([01]\d[0-5]\d[0-5]\d)?`)

	//////////////// Process each input file ////////////////
	for i, f := range inFiles {
		var p Page
		input, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		p.File = strings.TrimSuffix(path.Base(f), path.Ext(f))
		if *pre {
			p.Body = string(input)
		} else {
			p.Body = string(blackfriday.MarkdownCommon(input))
		}

		newlines := func(c rune) bool {
			return strings.ContainsRune("\u000A\u000B\u000C\u000D\u0085\u2028\u2029", c)
		}

		// Process each line in this input file, looking for date, title, and hashtags.
		for _, l := range bytes.FieldsFunc(input, newlines) {
			if reTitle.Match(l) && p.Title == "" {
				p.Title = string(bytes.Trim(l, " #"))
			}
			if reHashtags.Match(l) {
				p.Hashtags = reHashtags.FindAllString(string(l), -1)
				for i, t := range p.Hashtags {
					p.Hashtags[i] = strings.TrimSpace(t)
				}
			}
			if reDate1.Match(l) && p.Date.IsZero() {
				p.Date, err = time.Parse(time.UnixDate, string(l))
				if err != nil {
					log.Fatal(err, string(l))
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
				log.Fatal(err)
			}
			p.firmDate = true
		}
		if p.Date.IsZero() {
			st, err := os.Stat(f)
			if err != nil {
				log.Fatal(err, f)
			}
			p.Date = st.ModTime()
		}

		Pages[i] = &p

		if *debug {
			fmt.Fprintf(os.Stderr, "\nProcessing input file '%v'...\n", f)
			fmt.Fprintf(os.Stderr, "FILE\t%v\n", p.File)
			fmt.Fprintf(os.Stderr, "TITLE\t%v\n", p.Title)
			fmt.Fprintf(os.Stderr, "DATE\t%v\n", p.Date)
			fmt.Fprintf(os.Stderr, "TAGS\t%v\n", p.Hashtags)
			fmt.Fprintf(os.Stderr, "OUT\t%v\n", path.Join(outDir, strings.Join([]string{p.File, ".html"}, "")))
		}
	}

	sort.Slice(Pages, func(i, j int) bool { return Pages[i].Date.After(Pages[j].Date) })

	for i, p := range Pages {
		p.Link = strings.Join([]string{p.File, ".html"}, "")
		if len(Pages) == 1 {
			p.Prev = p.Link
			p.Next = p.Link
			break
		}
		switch i {
		case 0:
			p.Prev = strings.Join([]string{Pages[i+1].File, ".html"}, "")
		case len(Pages) - 1:
			p.Next = strings.Join([]string{Pages[i-1].File, ".html"}, "")
		default:
			p.Prev = strings.Join([]string{Pages[i+1].File, ".html"}, "")
			p.Next = strings.Join([]string{Pages[i-1].File, ".html"}, "")
		}
	}

	//////////////// Output individual pages ////////////////
	if _, err := os.Stat(path.Join(*tmpldir, "page.tmpl")); os.IsNotExist(err) {
		tmpl, err = template.New("").Parse(`<!DOCTYPE html>
		<html lang="en-us">
		<head>
		<meta charset="utf-8" />
		<link rel="stylesheet" href="default.css" />
		<title>{{.Title}}</title>
		</head>
		<body>{{.Body}}</body>
		</html>`)
		if err != nil {
			log.Fatal(err)
		}
		if *debug {
			fmt.Fprintf(os.Stderr, "\nPage template not found; using minmal fallback template.\n")
		}
	} else {
		tmpl, err = template.ParseFiles(path.Join(*tmpldir, "page.tmpl"))
		if err != nil {
			log.Fatal(err)
		}
	}
	for _, p := range Pages {
		f, err := os.Create(path.Join(outDir, strings.Join([]string{p.File, ".html"}, "")))
		if err != nil {
			log.Fatal(err)
		}
		err = tmpl.Execute(f, p)
		if err != nil {
			log.Fatal(err)
		}
	}

	postListTmpl := `<!DOCTYPE html>
		<html lang="en-us">
		<head>
		<meta charset="utf-8" />
		<link rel="stylesheet" href="default.css" />
		<title>{{.Title}}</title>
		</head><body>
		<ul>{{range .Pages}}
		<li><a href="{{.Link}}">{{.Title}}</a></li>{{end}}
		</ul></body>
		</html>`

	//////////////// Generate chronological Archive page ////////////////
	if _, err := os.Stat(path.Join(*tmpldir, "archive.tmpl")); os.IsNotExist(err) {
		tmpl, err = template.New("").Parse(postListTmpl)
		if err != nil {
			log.Fatal(err)
		}
		if *debug {
			fmt.Fprintf(os.Stderr, "\nArchive template not found; using minmal fallback template.\n")
		}
	} else {
		tmpl, err = template.ParseFiles(path.Join(*tmpldir, "archive.tmpl"))
		if err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.Create(path.Join(outDir, "archive.html"))
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.Execute(f, struct {
		Title string
		Pages []*Page
	}{
		"Archive",
		Pages,
	})
	if err != nil {
		log.Fatal(err)
	}
	if *debug {
		fmt.Fprintf(os.Stderr, "\nGenerated Archive file '%v'.\n", f.Name())
	}

	//////////////// Generate alphabetically sorted Contents page ////////////////
	sort.Slice(Pages, func(i, j int) bool { return Pages[i].Title < Pages[j].Title })
	if _, err := os.Stat(path.Join(*tmpldir, "contents.tmpl")); os.IsNotExist(err) {
		tmpl, err = template.New("").Parse(postListTmpl)
		if err != nil {
			log.Fatal(err)
		}
		if *debug {
			fmt.Fprintf(os.Stderr, "\nContents template not found; using minmal fallback template.\n")
		}
	} else {
		tmpl, err = template.ParseFiles(path.Join(*tmpldir, "contents.tmpl"))
		if err != nil {
			log.Fatal(err)
		}
	}
	f, err = os.Create(path.Join(outDir, "contents.html"))
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.Execute(f, struct {
		Title string
		Pages []*Page
	}{
		"Contents",
		Pages,
	})
	if err != nil {
		log.Fatal(err)
	}
	if *debug {
		fmt.Fprintf(os.Stderr, "\nGenerated Contents file '%v'.\n", f.Name())
	}

	//////////////// Generate RSS feed ////////////////
	sort.Slice(Pages, func(i, j int) bool { return Pages[i].Date.After(Pages[j].Date) })
	if *siteTitle != "" && *siteURL != "" && *siteDesc != "" {
		if !strings.HasSuffix(*siteURL, "/") {
			*siteURL += "/"
		}
		feed := Feed{
			Title: html.EscapeString(*siteTitle),
			URL:   html.EscapeString(*siteURL),
			Desc:  html.EscapeString(*siteDesc),
			Items: make([]*Page, 0, len(Pages)),
		}
		for i := 0; i < 25 && i < len(Pages); i++ {
			if Pages[i].firmDate {
				feed.Items = append(feed.Items, Pages[i])
			}
		}
		for _, item := range feed.Items {
			item.Link = strings.Join([]string{feed.URL, item.File, ".html"}, "")
			item.Title = html.EscapeString(item.Title)
			item.Link = html.EscapeString(item.Link)
		}
		if _, err := os.Stat(path.Join(*tmpldir, "rss.tmpl")); os.IsNotExist(err) {
			tmpl, err = template.New("").Parse(`<?xml version="1.0" encoding="utf-8"?>
			<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
			<channel>
			<atom:link href="{{.URL}}rss.xml" rel="self" type="application/rss+xml" />
			<title>{{.Title}}</title>
			<link>{{.URL}}</link>
			<description>{{.Desc}}</description>
			{{range .Items}}<item>
			<title>{{.Title}}</title>
			<link>{{.Link}}</link>
			<guid isPermaLink="true">{{.Link}}</guid>
			</item>{{end}}
			</channel>
			</rss>`)
			if err != nil {
				log.Fatal(err)
			}
			if *debug {
				fmt.Fprintf(os.Stderr, "\nRSS template not found; using minmal fallback template.\n")
			}
		} else {
			tmpl, err = template.ParseFiles(path.Join(*tmpldir, "rss.tmpl"))
			if err != nil {
				log.Fatal(err)
			}
		}
		f, err := os.Create(path.Join(outDir, "rss.xml"))
		if err != nil {
			log.Fatal(err)
		}
		err = tmpl.Execute(f, feed)
		if err != nil {
			log.Fatal(err)
		}
		if *debug {
			fmt.Fprintf(os.Stderr, "\nAdded RSS feed items:\n")
			for _, item := range feed.Items {
				fmt.Fprintf(os.Stderr, "%v\t%v\t%v\n", item.Date, item.Link, item.Title)
			}
		}
	}

	//////////////// Generate "lastest" posts HTML snippet ////////////////
	Latest := make([]*Page, 0, 10)
	for i := 0; i < 10 && i < len(Pages); i++ {
		Latest = append(Latest, Pages[i])
	}
	if _, err := os.Stat(path.Join(*tmpldir, "latest.tmpl")); os.IsNotExist(err) {
		tmpl, err = template.New("").Parse(`<div id="latest">
		<ul>{{range .}}
		<li><a href="{{.Link}}">{{.Title}}</a></li>{{end}}
		</ul>
		</div>`)
		if err != nil {
			log.Fatal(err)
		}
		if *debug {
			fmt.Fprintf(os.Stderr, "\n\"Latest posts\" template not found; using minmal fallback template.\n")
		}
	} else {
		tmpl, err = template.ParseFiles(path.Join(*tmpldir, "latest.tmpl"))
		if err != nil {
			log.Fatal(err)
		}
	}
	f, err = os.Create(path.Join(outDir, "latest.html"))
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.Execute(f, Latest)
	if err != nil {
		log.Fatal(err)
	}
	if *debug {
		fmt.Fprintf(os.Stderr, "\nGenerated \"latest posts\" HTML snippet.\n")
	}

	//////////////// Generate Hashtags index page ////////////////
	TagIndex := make(map[string][]*Page)
	for _, p := range Pages {
		for _, t := range p.Hashtags {
			TagIndex[t] = append(TagIndex[t], p)
		}
	}
	if _, err := os.Stat(path.Join(*tmpldir, "hashtags.tmpl")); os.IsNotExist(err) {
		tmpl, err = template.New("").Parse(`<!DOCTYPE html>
		<html lang="en-us">
		<head>
		<meta charset="utf-8" />
		<link rel="stylesheet" href="default.css" />
		<title>Hashtags</title>
		</head>
		<body>
		{{range $k, $v := .}}<h2 id="{{ $k }}">{{ $k }}</h2>
		<ul>{{ range $v }}
		<li><a href="{{ .Link }}">{{ .Title }}</a></li>{{end}}
		</ul>{{ end }}
		</body>
		</html>`)
		if err != nil {
			log.Fatal(err)
		}
		if *debug {
			fmt.Fprintf(os.Stderr, "\nHashtags template not found; using minmal fallback template.\n")
		}
	} else {
		tmpl, err = template.ParseFiles(path.Join(*tmpldir, "hashtags.tmpl"))
		if err != nil {
			log.Fatal(err)
		}
	}
	f, err = os.Create(path.Join(outDir, "hashtags.html"))
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.Execute(f, TagIndex)
	if err != nil {
		log.Fatal(err)
	}
	if *debug {
		fmt.Fprintf(os.Stderr, "\nTag Index:\n")
		for tag, pages := range TagIndex {
			for _, p := range pages {
				fmt.Fprintf(os.Stderr, "%v\t%v\n", tag, p.Title)
			}
		}
	}
}
