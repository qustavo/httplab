package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/gchaincl/httplab"
)

func NewHandler(ui *httplab.UI) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ui.Info("New Request from " + req.Host)
		buf, err := httplab.DumpRequest(req)
		if err != nil {
			ui.Info(fmt.Sprintf("%+v", err))
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

		// Let the UI allocate the views before use them
		runtime.Gosched()
		ui.Info("Listening on :8000")

		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatalln(err)
		}
	}()

	if err := ui.Loop(); err != nil {
		log.Fatalln(err)
	}

}
