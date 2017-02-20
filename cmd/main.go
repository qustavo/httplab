package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gchaincl/httplab"
	"github.com/jroimartin/gocui"
)

func NewHandler(ui *httplab.UI, g *gocui.Gui) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ui.Info(g, "New Request from "+req.Host)
		buf, err := httplab.DumpRequest(req)
		if err != nil {
			ui.Info(g, "%v", err)
		}

		ui.Display(g, "request", buf)

		resp := ui.Response()
		time.Sleep(resp.Delay)
		resp.Write(w)

	}
	return http.HandlerFunc(fn)
}

func main() {
	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		log.Fatalln(err)
	}

	ui := httplab.NewUI()
	if err := ui.Init(g); err != nil {
		log.Fatalln(err)
	}

	http.Handle("/", NewHandler(ui, g))
	go func() {
		ui.Info(g, "Listening on :18000")
		if err := http.ListenAndServe(":18000", nil); err != nil {
			log.Fatalln(err)
		}
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}
}
