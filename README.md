Glog is a very, very simple static site generator written in Go (golang).

Glog treats the first argument as a path.
It reads all the `.txt` files in that directory, and outputs `.html` files to the same path.
It overwrites existing `.html` files with the same name.
If the `-o` flag is given with the path to a directory, output will be sent there.
Glog also produces a `contents.html` --- an alphabetized list of links to each `.html` file.

	$ glog /home/me/blog/
	$ glog -g '*.md' -o /tmp /home/me/blog/

If the user does not supply a path, Glog looks in the current directory.

For a complete list of command-line options:

	$ glog -h

For files named like `20171231.md` or `20171231235959.md`, Glog creates "forward" and "backward" links between files.
Glog may also produce an RSS feed.

If the file `page.tmpl` exists in the target directory, Glog reads it as a Go template for format output.
If the first line if an input file is a Markdown header, Glog uses that as the title of the page;
otherwise, it uses the file name.

Glog sets the date of the page as the first line containing a date formatted like "Sat Dec 31 09:18:57 EST 2016" or "Sun Jan  1 07:56:01 EST 2017".
This is the default format of the GNU `date` command.
If the file does not contain such a line, Glog sets the page date to the mtime of the file (see INODE(7)).

## Dependencies ##

Glog depends on the Blackfriday Markdown parser.

	$ go get github.com/russross/blackfriday

## Configuration and Publishing ##

Glog does not use a configuration file.
It does not have a "publish" function.

The recommended way to fulfill these rolls is with a simple shell script, like:

```
#!/bin/sh

title="My Blog"
desc="My blog posts awesome stuff!"
url="https://example.com/blog/"
now=$(date +%Y-%m-%d-%H%M%S)
odir=/tmp/"$title"/"$now"

mkdir -p odir

$HOME/bin/glog -o "$odir" \
	-t $HOME/repo/"$title"/templates \
	-g '.md' \
	-t "$title" \
	-d "$desc" \
	-u "$url" \
	$HOME/repo/"$title"

rsync
```

## License (2-Clause BSD License) ##

Copyright 2017 Paul Gorman

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
