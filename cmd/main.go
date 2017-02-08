package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/gchaincl/httplab"
)

func NewHandler(ui *httplab.UI) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		buf, err := httputil.DumpRequest(req, false)
		if err != nil {
			log.Fatalln(err)
		}

		ui.Display(
			httplab.RequestView,
			[]byte(strings.Replace(string(buf), "\r", "", -1)),
		)

		ui.Response().Write(w)

	}
	return http.HandlerFunc(fn)
}

func main() {
	ui, err := httplab.NewUI()
	if err != nil {
		panic(err)
	}

	go func() {
		http.Handle("/", NewHandler(ui))
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatalln(err)
		}
	}()

	if err := ui.Loop(); err != nil {
		log.Fatalln(err)
	}

}
