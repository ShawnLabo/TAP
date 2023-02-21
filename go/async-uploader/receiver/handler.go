package main

import "net/http"

type handler struct {
	mux *http.ServeMux
}

func newHandler(c *controller) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", c.root)
	mux.HandleFunc("/dataSequence", c.postDataSequence)

	return mux
}
