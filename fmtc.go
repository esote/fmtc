package main

import (
	"context"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/esote/openshim"
)

const indexHTML = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<title>C Formatter</title>
		<meta name="author" content="Esote">
		<meta name="description"
			content="Format C code using the indent(1) command.">
		<style>
			textarea {
				font-family: monospace;
				white-space: pre;
			}
		</style>
	</head>
	<body>
		<h1>C Formatter</h1>
		<p>Format C code using the <code>
			<a href="https://man.openbsd.org/indent.1"
			target="_blank">indent(1)</a></code> command. Author:
			<a href="https://esote.net" target="_blank">Esote</a>.
			<a href="https://github.com/esote/fmtc"
			target="_blank">View source</a>.</p>
		<form action="/format" method="POST">
			<p><input type="submit" value="Format"></p>
			<textarea cols="80" rows="20" name="src"></textarea>
		</form>
	</body>
</html>`

func format(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/", http.StatusBadRequest)
		return
	}

	src := r.PostFormValue("src")

	src = strings.Replace(src, "\r", "", -1)
	src += "\n"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "./indent.out")

	cmd.Stdin = strings.NewReader(src)

	out, err := cmd.CombinedOutput()

	if err != nil {
		if err == context.DeadlineExceeded {
			http.Redirect(w, r, "/", http.StatusRequestTimeout)
		} else {
			http.Redirect(w, r, "/", http.StatusBadRequest)
		}
		return
	}

	w.Write(out)
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Redirect(w, r, "/", http.StatusMethodNotAllowed)
		return
	}

	w.Write([]byte(indexHTML))
}

func main() {
	if _, err := openshim.Pledge("stdio inet proc exec unveil",
		"stdio rpath"); err != nil {
		log.Fatal(err)
	}

	if _, err := openshim.Unveil("/usr/lib/", "r"); err != nil {
		log.Fatal(err)
	}

	if _, err := openshim.Unveil("/usr/libexec/ld.so", "r"); err != nil {
		log.Fatal(err)
	}

	if _, err := openshim.Unveil("/root/indent/indent.out",
		"x"); err != nil {
		log.Fatal(err)
	}

	if _, err := openshim.Pledge("stdio inet proc exec",
		"stdio rpath"); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/format", format)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
