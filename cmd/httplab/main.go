package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/gchaincl/httplab/ui"
	"github.com/jroimartin/gocui"
	"github.com/rs/cors"
)

const VERSION = "v0.4.0-dev"

func NewHandler(ui *ui.UI, g *gocui.Gui) http.Handler {
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

func main() {
	var (
		config   = kingpin.Flag("config", "Specifies custom config path.").Short('c').String()
		cors     = kingpin.Flag("cors", "Enable CORS").Default("false").Bool()
		port     = kingpin.Flag("port", "Specifies the port where HTTPLab will bind to.").Short('p').Default("10080").Int()
		bindings = kingpin.Flag("bindings", "Show keyboard bindings").Bool()
	)

	kingpin.Version(VERSION)
	kingpin.CommandLine.VersionFlag.Short('v')
	kingpin.CommandLine.HelpFlag.Short('h')

	kingpin.Parse()

	if *bindings {
		fmt.Fprintf(os.Stderr, "\nBindings:\n%s", ui.Bindings.Help())
		os.Exit(0)
	}

	if err := run(*config, *port, *cors); err != nil && err != gocui.ErrQuit {
		log.Println(err)
	}
}

func run(config string, port int, _cors bool) error {
	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	if config == "" {
		config = defaultConfigPath()
	}

	ui := ui.New(config)
	errCh, err := ui.Init(g)
	if err != nil {
		return err
	}

	handler := NewHandler(ui, g)
	if _cors == true {
		log.Printf("With CORS")
		handler = cors.Default().Handler(handler)
	}

	http.Handle("/", handler)
	go func() {
		// Make sure gocui has started
		g.Execute(func(g *gocui.Gui) error { return nil })

		ui.Info(g, "Listening on :%d", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			errCh <- err
		}
	}()

	return g.MainLoop()
}
