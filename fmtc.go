package main

import (
	"context"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const indexHTML = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<title>C Formatter</title>
		<style>
			#src {
				margin-bottom: 1rem;
				font-family: monospace;
				white-space: pre;
			}
		</style>
	</head>
	<body>
		<form action="/" method="POST">
			<textarea id="src" cols="80" rows="20" name="src"></textarea>
			<br>
			<input type="submit" value="Submit">
		</form>
	</body>
</html>`

func format(src string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "indent")

	cmd.Stdin = strings.NewReader(src)

	return cmd.Output()
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || r.ParseForm() != nil {
		w.Write([]byte(indexHTML))
		return
	}

	src := r.PostFormValue("src")

	out, err := format(src)

	if err != nil {
		w.Write([]byte(indexHTML))
		return
	}

	w.Write(out)
}

func main() {
	http.HandleFunc("/", index)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
