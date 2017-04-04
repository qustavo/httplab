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
)

const VERSION = "v0.2.0"

func NewHandler(ui *UI, g *gocui.Gui) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if err := ui.AddRequest(g, req); err != nil {
			ui.Info(g, "%v", err)
		}

		resp := ui.Response()
		time.Sleep(resp.Delay)
		resp.Write(w)

	}
	return http.HandlerFunc(fn)
}

func defaultConfigPath() string {
	var path = ".httplab"

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path
	}

	u, err := user.Current()
	if err != nil {
		return path
	}

	return u.HomeDir + "/" + path
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nBindings:\n%s", Bindings.Help())
}

func Version() {
	fmt.Fprintf(os.Stdout, "%s\n", VERSION)
	os.Exit(0)
}

func main() {
	var port int
	var config string
	var version bool

	flag.Usage = usage
	flag.IntVar(&port, "port", 10080, "Specifies the port where HTTPLab will bind to.")
	flag.StringVar(&config, "config", "", "Specifies custom config path.")
	flag.BoolVar(&version, "version", false, "Prints current version.")

	flag.Parse()

	if version {
		Version()
	}

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
		log.Println(err)
		return
	}

	http.Handle("/", NewHandler(ui, g))
	go func() {
		// Make sure gocui has started
		g.Execute(func(g *gocui.Gui) error { return nil })

		ui.Info(g, "Listening on :%d", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Println(err)
			return
		}
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Println(err)
		return
	}
}
