package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/esote/graceful"
	"golang.org/x/sys/unix"
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
	// force init of lazy sysctls
	if l, err := net.Listen("tcp", "localhost:0"); err != nil {
		log.Fatal(err)
	} else {
		l.Close()
	}

	// letsencrypt key and cert, /etc/letsencrypt/live symlinks to archive/
	if err := unix.Unveil("/etc/letsencrypt/archive/", "r"); err != nil {
		log.Fatal(err)
	}

	if err := unix.Unveil("./indent.out", "x"); err != nil {
		log.Fatal(err)
	}

	if err := unix.Pledge("stdio inet proc exec rpath",
		"stdio"); err != nil {
		log.Fatal(err)
	}

	var (
		cert string
		key  string
	)

	flag.StringVar(&cert, "cert", "server.crt", "TLS certificate file")
	flag.StringVar(&key, "key", "server.key", "TLS key file")

	flag.Parse()

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.X25519,
		},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", index)
	mux.HandleFunc("/format", format)

	server := &http.Server{
		Addr:         ":8443",
		Handler:      mux,
		TLSConfig:    cfg,
		TLSNextProto: nil,
	}

	graceful.Graceful(server, func() {
		err := server.ListenAndServeTLS(cert, key)

		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}, os.Interrupt)
}
