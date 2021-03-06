package main

import (
	"context"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/esote/openshim2"
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
				font-family: monospace, monospace;
				white-space: pre;
			}
		</style>
	</head>
	<body>
		<h1>C Formatter</h1>
		<p>Format C code according to the OpenBSD kernel normal form.
		</p>
		<p>Author: <a href="https://github.com/esote/"
			target="_blank">Esote</a>.

			<a href="https://github.com/esote/fmtc"
			target="_blank">Webserver source</a>.

			<a href="https://github.com/esote/indent"
			target="_blank">Formatter source</a>.</p>
		<form action="/format" method="POST">
			<p><input type="submit" value="Format"></p>
			<textarea cols="80" rows="20" name="src"></textarea>
		</form>
	</body>
</html>`

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000;"+
		"includeSubDomains;preload")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "deny")
	w.Header().Set("X-XSS-Protection", "1")
}

func format(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Security-Policy", "default-src 'none';")

	if r.Method != "POST" {
		http.Error(w, "bad http verb", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "form invalid", http.StatusBadRequest)
		return
	}

	src := r.PostFormValue("src")
	src = strings.Replace(src, "\r", "", -1)

	if src == "" {
		return
	}

	src += "\n"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "./indent.out")
	cmd.Stdin = strings.NewReader(src)
	out, err := cmd.CombinedOutput()

	if err != nil {
		if err == context.DeadlineExceeded {
			http.Error(w, "execution time deadline exceeded",
				http.StatusRequestTimeout)
		} else {
			http.Error(w, "error parsing input",
				http.StatusInternalServerError)
		}
		return
	}

	_, _ = w.Write(out)
}

func index(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)
	w.Header().Set("Content-Security-Policy", "default-src 'none';"+
		"style-src 'unsafe-inline'")

	if r.Method != "GET" {
		http.Error(w, "bad http verb", http.StatusMethodNotAllowed)
		return
	}

	_, _ = w.Write([]byte(indexHTML))
}

func main() {
	if err := openshim2.Unveil("./indent.out", "x"); err != nil {
		log.Fatal(err)
	}
	if err := openshim2.Pledge("stdio inet proc exec rpath", "stdio"); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", index)
	mux.HandleFunc("/format", format)

	srv := &http.Server{
		Addr:    ":8443",
		Handler: mux,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
