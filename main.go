package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jroimartin/gocui"
)

func NewHandler(ui *UI, g *gocui.Gui) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ui.Info(g, "New Request from "+req.Host)
		buf, err := DumpRequest(req)
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
	var port int
	flag.IntVar(&port, "port", 10080, "Specifies the port where HTTPLab will bind to")
	flag.Parse()

	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	ui := NewUI()
	if err := ui.Init(g); err != nil {
		log.Fatalln(err)
	}

	http.Handle("/", NewHandler(ui, g))
	go func() {
		// Make sure gocui has started
		g.Execute(func(g *gocui.Gui) error { return nil })

		ui.Info(g, "Listening on :%d", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Fatalln(err)
		}
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}
}
