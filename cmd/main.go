package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gchaincl/httplab"
)

func NewHandler(ui *httplab.UI) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		buf, err := httplab.DumpRequest(req)
		if err != nil {
			fmtErr := fmt.Sprintf("%+v", err)
			ui.Display(httplab.InfoView, []byte(fmtErr))
		}

		ui.Display(httplab.RequestView, buf)

		resp := ui.Response()
		time.Sleep(resp.Delay)
		resp.Write(w)

	}
	return http.HandlerFunc(fn)
}

func main() {
	ui, err := httplab.NewUI()
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		http.Handle("/", NewHandler(ui))
		ui.Display(httplab.InfoView, []byte("Listening on :8000"))
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatalln(err)
		}
	}()

	if err := ui.Loop(); err != nil {
		log.Fatalln(err)
	}

}
