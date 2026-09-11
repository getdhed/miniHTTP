package main

import "net/http"

func myHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello!"))
}
