package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"time"

	"github.com/gchaincl/httplab/ui"
	"github.com/jroimartin/gocui"
	"github.com/rs/cors"
	flag "github.com/spf13/pflag"
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

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nBindings:\n%s", ui.Bindings.Help())
}

func Version() {
	fmt.Fprintf(os.Stdout, "%s\n", VERSION)
	os.Exit(0)
}

func main() {
	var (
		port        int
		config      string
		version     bool
		corsEnabled bool
		corsDisplay bool
	)

	flag.Usage = usage

	flag.IntVarP(&port, "port", "p", 10080, "Specifies the port where HTTPLab will bind to.")
	flag.StringVarP(&config, "config", "c", "", "Specifies custom config path.")
	flag.BoolVarP(&version, "version", "v", false, "Prints current version.")
	flag.BoolVar(&corsEnabled, "cors", false, "Enable CORS.")
	flag.BoolVar(&corsDisplay, "cors-display", true, "Display CORS requests")


	flag.Parse()

	if version {
		Version()
	}

	// noop
	middleware := func(next http.Handler) http.Handler {
		return next
	}

	if corsEnabled {
		middleware = cors.New(cors.Options{
			OptionsPassthrough: corsDisplay,
		}).Handler
	}

	if err := run(config, port, middleware); err != nil && err != gocui.ErrQuit {
		log.Println(err)
	}
}

func run(config string, port int, middleware func(next http.Handler) http.Handler) error {
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

	http.Handle("/", middleware(NewHandler(ui, g)))
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
