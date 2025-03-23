package main

import (
	"fmt"
	"log"
	"net/http"

	_ "embed"
)

//go:embed web/index.html
var page []byte

//go:embed wasm/zig-out/bin/pastesec.wasm
var wasm_bin []byte

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write(page)
			return
		}

		if r.URL.Path == "/wasm" {
			w.Header().Set("Content-Type", "application/wasm")
			w.Write(wasm_bin)
			return
		}

		w.WriteHeader(404)
		w.Write([]byte("404 Page Not Found"))
	})

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
