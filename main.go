package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"time"

	"github.com/jroimartin/gocui"
    "github.com/zchee/go-xdgbasedir"
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

func defaultConfigPath() string {
    var configFilename = "httplab"
    var dotConfigFilename = "." + configFilename

    if _, err := os.Stat(xdgbasedir.ConfigHome()); !os.IsNotExist(err) {
        return xdgbasedir.ConfigHome() + "/" + configFilename
    }

	if _, err := os.Stat(dotConfigFilename); !os.IsNotExist(err) {
		return dotConfigFilename
	}

	u, err := user.Current()
	if err != nil {
		return dotConfigFilename
	}

	return u.HomeDir + "/" + dotConfigFilename
}

func main() {
	var port int
	var config string

	flag.IntVar(&port, "port", 10080, "Specifies the port where HTTPLab will bind to")
	flag.StringVar(&config, "config", "", "Specifies custom config path.")

	flag.Parse()

	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	if config == "" {
		config = defaultConfigPath()
	}

	ui := NewUI(config)
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
