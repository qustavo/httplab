package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/gchaincl/httplab"
	"github.com/gchaincl/httplab/ui"
	"github.com/jroimartin/gocui"
	"github.com/rs/cors"
	flag "github.com/spf13/pflag"
)

const VERSION = "v0.5.0-dev"

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

type cmdArgs struct {
	autoUpdate  bool
	body        string
	config      string
	corsEnabled bool
	corsDisplay bool
	delay       int
	headers     []string
	port        int
	status      string
	version     bool
}

func main() {
	var args cmdArgs

	flag.Usage = usage

	flag.BoolVarP(&args.autoUpdate, "auto-update", "a", true, "Auto-updates response when fields change.")
	flag.StringVarP(&args.body, "body", "b", "Hello, World", "Specifies the inital response body.")
	flag.StringVarP(&args.config, "config", "c", "", "Specifies custom config path.")
	flag.BoolVar(&args.corsEnabled, "cors", false, "Enable CORS.")
	flag.BoolVar(&args.corsDisplay, "cors-display", true, "Display CORS requests.")
	flag.IntVarP(&args.delay, "delay", "d", 0, "Specifies the initial response delay in ms.")
	flag.StringSliceVarP(&args.headers, "headers", "H", []string{"X-Server:HTTPLab"}, "Specifies the initial response headers.")
	flag.IntVarP(&args.port, "port", "p", 10080, "Specifies the port where HTTPLab will bind to.")
	flag.StringVarP(&args.status, "status", "s", "200", "Specifies the initial response status.")
	flag.BoolVarP(&args.version, "version", "v", false, "Prints current version.")

	flag.Parse()

	if args.version {
		Version()
	}

	// noop
	middleware := func(next http.Handler) http.Handler {
		return next
	}

	if args.corsEnabled {
		middleware = cors.New(cors.Options{
			OptionsPassthrough: args.corsDisplay,
		}).Handler
	}

	if srv, err := run(args, middleware); err != nil {
		if err == gocui.ErrQuit {
			log.Println("HTTPLab is shutting down")
			srv.Shutdown(context.Background())
		} else {
			log.Println(err)
		}
	}
}

func newResponse(args *cmdArgs) (*httplab.Response, error) {
	resp, err := httplab.NewResponse(args.status, strings.Join(args.headers, "\n"), args.body)
	if err != nil {
		return nil, err
	}
	resp.Delay = time.Duration(args.delay) * time.Millisecond
	return resp, nil
}

func run(args cmdArgs, middleware func(next http.Handler) http.Handler) (*http.Server, error) {
	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		return nil, err
	}
	defer g.Close()

	if args.config == "" {
		args.config = defaultConfigPath()
	}

	resp, err := newResponse(&args)
	if err != nil {
		return nil, err
	}

	ui := ui.New(resp, args.config)
	ui.AutoUpdate = args.autoUpdate

	errCh, err := ui.Init(g)
	if err != nil {
		return nil, err
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", args.port),
		Handler: http.Handler(middleware(NewHandler(ui, g))),
	}

	go func() {
		// Make sure gocui has started
		g.Execute(func(g *gocui.Gui) error { return nil })

		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		} else {
			ui.Info(g, "Listening on :%d", args.port)
		}
	}()

	return srv, g.MainLoop()
}
