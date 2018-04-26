package ui

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gchaincl/httplab"
	"github.com/jroimartin/gocui"
)

const (
	// StatusView widget sets the response staus code
	StatusView = "status"
	// DelayView widget sets the response delay time
	DelayView = "delay"
	// HeaderView widget sets the response headers
	HeaderView = "headers"
	// BodyView widget sets the response body content
	BodyView = "body"
	// BodyFileView widget shows the file content to be set on the response body
	BodyFileView = "bodyfile"
	// RequestView widget displays the request
	RequestView = "request"
	// InfoView widget displays the bottom info bar
	InfoView = "info"
	// SaveView widget displays the saving location
	SaveView = "save"
	// ResponsesView widget displays the saved responses
	ResponsesView = "responses"
	// BindingsView widget displays binding help
	BindingsView = "bindings"
	// FileDialogView widget displays the popup to choose the response body file
	FileDialogView = "file-dialog"
)

var cicleable = []string{
	StatusView,
	DelayView,
	HeaderView,
	BodyView,
	RequestView,
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Cursors stores the cursor position for a specific view
// this is used to restore mouse position when click is detected.
type Cursors map[string]struct{ x, y int }

// NewCursors returns a new Cursor.
func NewCursors() Cursors {
	return make(Cursors)
}

// Restore restores the cursor position.
func (c Cursors) Restore(view *gocui.View) error {
	return view.SetCursor(c.Get(view.Name()))
}

// Get gets the cursor position.
func (c Cursors) Get(view string) (int, int) {
	if v, ok := c[view]; ok {
		return v.x, v.y
	}
	return 0, 0
}

// Set sets the cursor position.
func (c Cursors) Set(view string, x, y int) {
	c[view] = struct{ x, y int }{x, y}
}

// UI represent the state of the ui.
type UI struct {
	resp                *httplab.Response
	responses           *httplab.ResponsesList
	infoTimer           *time.Timer
	viewIndex           int
	currentPopup        string
	configPath          string
	hideResponseBuilder bool
	cursors             Cursors

	reqLock        sync.Mutex
	requests       [][]byte
	currentRequest int

	AutoUpdate bool
	hasChanged bool
}

// New returns a new UI with default values specified on the Response.
func New(resp *httplab.Response, configPath string) *UI {
	return &UI{
		resp:       resp,
		responses:  httplab.NewResponsesList(),
		configPath: configPath,
		cursors:    NewCursors(),
	}
}

// Init initializes the UI.
func (ui *UI) Init(g *gocui.Gui) (chan<- error, error) {
	g.Cursor = true
	g.Highlight = true
	g.SelFgColor = gocui.ColorGreen
	g.Mouse = true

	g.SetManager(ui)
	if err := Bindings.Apply(ui, g); err != nil {
		return nil, err
	}

	errCh := make(chan error)
	go func() {
		err := <-errCh
		g.Execute(func(g *gocui.Gui) error {
			return err
		})
	}()

	for _, view := range cicleable {
		fn := func(g *gocui.Gui, v *gocui.View) error {
			cx, cy := v.Cursor()
			line, err := v.Line(cy)
			if err != nil {
				ui.cursors.Restore(v)
				ui.setView(g, v.Name())
				return nil
			}

			if cx > len(line) {
				v.SetCursor(len(line), cy)
				ui.cursors.Set(v.Name(), len(line), cy)
			}

			ui.setView(g, v.Name())
			return nil
		}

		if err := g.SetKeybinding(view, gocui.MouseLeft, gocui.ModNone, fn); err != nil {
			return nil, err
		}

		if err := g.SetKeybinding(view, gocui.MouseRelease, gocui.ModNone, fn); err != nil {
			return nil, err
		}
	}

	return errCh, nil
}

// AddRequest adds a new request to the UI.
func (ui *UI) AddRequest(g *gocui.Gui, req *http.Request) error {
	ui.reqLock.Lock()
	defer ui.reqLock.Unlock()

	ui.Info(g, "New Request from "+req.Host)
	buf, err := httplab.DumpRequest(req)
	if err != nil {
		return err
	}

	if ui.currentRequest == len(ui.requests)-1 {
		ui.currentRequest = ui.currentRequest + 1
	}

	ui.requests = append(ui.requests, buf)
	return ui.updateRequest(g)
}

func (ui *UI) updateRequest(g *gocui.Gui) error {
	req := ui.requests[ui.currentRequest]

	view, err := g.View(RequestView)
	if err != nil {
		return err
	}

	view.Title = fmt.Sprintf("Request (%d/%d)", ui.currentRequest+1, len(ui.requests))
	return ui.Display(g, RequestView, req)
}

func (ui *UI) resetRequests(g *gocui.Gui) error {
	ui.reqLock.Lock()
	defer ui.reqLock.Unlock()
	ui.requests = nil
	ui.currentRequest = 0

	v, err := g.View(RequestView)
	if err != nil {
		return err
	}

	v.Title = "Request"
	v.Clear()
	ui.Info(g, "Requests cleared")
	return nil
}

// Layout sets the layout
func (ui *UI) Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	var splitX, splitY *Split
	if ui.hideResponseBuilder {
		splitX = NewSplit(maxX).Fixed(maxX - 1)
	} else {
		splitX = NewSplit(maxX).Relative(70)
	}
	splitY = NewSplit(maxY).Fixed(maxY - 2)

	if v, err := g.SetView(RequestView, 0, 0, splitX.Next(), splitY.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Request"
		v.Editable = true
		v.Editor = newEditor(ui, g, &motionEditor{})
	}

	if err := ui.setResponseView(g, splitX.Current(), 0, maxX-1, splitY.Current()); err != nil {
		return err
	}

	if v, err := g.SetView(InfoView, -1, splitY.Current(), maxX-1, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
	}

	if v := g.CurrentView(); v == nil {
		_, err := g.SetCurrentView(StatusView)
		if err != gocui.ErrUnknownView {
			return err
		}
	}

	return nil
}

func (ui *UI) setResponseView(g *gocui.Gui, x0, y0, x1, y1 int) error {
	if ui.hideResponseBuilder {
		g.DeleteView(StatusView)
		g.DeleteView(DelayView)
		g.DeleteView(HeaderView)
		g.DeleteView(BodyView)
		return nil
	}

	split := NewSplit(y1).Fixed(2, 2).Relative(40)
	if v, err := g.SetView(StatusView, x0, y0, x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Status"
		v.Editable = true
		v.Editor = newEditor(ui, g, &numberEditor{3})
		fmt.Fprintf(v, "%d", ui.resp.Status)
	}

	if v, err := g.SetView(DelayView, x0, split.Current(), x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Delay (ms) "
		v.Editable = true
		v.Editor = newEditor(ui, g, &numberEditor{9})
		fmt.Fprintf(v, "%d", ui.resp.Delay/time.Millisecond)
	}

	if v, err := g.SetView(HeaderView, x0, split.Current(), x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
		v.Editor = newEditor(ui, g, nil)
		v.Title = "Headers"
		var headers []string
		for key := range ui.resp.Headers {
			headers = append(headers, key+": "+ui.resp.Headers.Get(key))
		}
		fmt.Fprint(v, strings.Join(headers, "\n"))
	}

	if v, err := g.SetView(BodyView, x0, split.Current(), x1, y1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
		v.Editor = newEditor(ui, g, nil)
		ui.renderBody(g)
	}

	return nil
}

// Info prints information on the InfoView.
func (ui *UI) Info(g *gocui.Gui, format string, args ...interface{}) {
	v, err := g.View(InfoView)
	if v == nil || err != nil {
		return
	}

	g.Execute(func(g *gocui.Gui) error {
		v.Clear()
		_, err := fmt.Fprintf(v, format, args...)
		return err
	})

	if ui.infoTimer != nil {
		ui.infoTimer.Stop()
	}
	ui.infoTimer = time.AfterFunc(3*time.Second, func() {
		g.Execute(func(g *gocui.Gui) error {
			v.Clear()
			return nil
		})
	})
}

// Display displays arbitraty info into a given view.
func (ui *UI) Display(g *gocui.Gui, view string, bytes []byte) error {
	v, err := g.View(view)
	if err != nil {
		return err
	}

	g.Execute(func(g *gocui.Gui) error {
		v.Clear()
		_, err := v.Write(bytes)
		return err
	})

	return nil
}

// Response returns the current response setting.
func (ui *UI) Response() *httplab.Response {
	return ui.resp
}

func (ui *UI) nextView(g *gocui.Gui) error {
	if ui.hideResponseBuilder {
		return nil
	}
	ui.viewIndex = (ui.viewIndex + 1) % len(cicleable)
	return ui.setView(g, cicleable[ui.viewIndex])
}

func (ui *UI) prevView(g *gocui.Gui) error {
	if ui.hideResponseBuilder {
		return nil
	}
	ui.viewIndex = (ui.viewIndex - 1 + len(cicleable)) % len(cicleable)
	return ui.setView(g, cicleable[ui.viewIndex])
}

func (ui *UI) prevRequest(g *gocui.Gui) error {
	ui.reqLock.Lock()
	defer ui.reqLock.Unlock()

	if ui.currentRequest == 0 {
		return nil
	}

	ui.currentRequest = ui.currentRequest - 1
	return ui.updateRequest(g)
}

func (ui *UI) nextRequest(g *gocui.Gui) error {
	ui.reqLock.Lock()
	defer ui.reqLock.Unlock()

	if ui.currentRequest >= len(ui.requests)-1 {
		return nil
	}

	ui.currentRequest = ui.currentRequest + 1
	return ui.updateRequest(g)
}

func getViewBuffer(g *gocui.Gui, view string) string {
	v, err := g.View(view)
	if err != nil {
		return ""
	}
	return v.Buffer()
}

func (ui *UI) currentResponse(g *gocui.Gui) (*httplab.Response, error) {
	status := getViewBuffer(g, StatusView)
	headers := getViewBuffer(g, HeaderView)

	resp, err := httplab.NewResponse(status, headers, "")
	if err != nil {
		return nil, err
	}

	resp.Body = ui.resp.Body
	if ui.Response().Body.Mode == httplab.BodyInput {
		resp.Body.Input = []byte(getViewBuffer(g, BodyView))
	}

	delay := getViewBuffer(g, DelayView)
	delay = strings.Trim(delay, " \n")
	intDelay, err := strconv.Atoi(delay)
	if err != nil {
		return nil, fmt.Errorf("Can't parse '%s' as number", delay)
	}
	resp.Delay = time.Duration(intDelay) * time.Millisecond

	return resp, nil
}

func (ui *UI) updateResponse(g *gocui.Gui) error {
	resp, err := ui.currentResponse(g)
	if err != nil {
		ui.Info(g, err.Error())
		return err
	}

	ui.resp = resp
	ui.Info(g, "Response updated!")
	return nil
}

func (ui *UI) restoreResponse(g *gocui.Gui, r *httplab.Response) {
	ui.resp = r

	var v *gocui.View
	v, _ = g.View(StatusView)
	v.Clear()
	fmt.Fprintf(v, "%d", r.Status)

	v, _ = g.View(DelayView)
	v.Clear()
	fmt.Fprintf(v, "%d", r.Delay)

	v, _ = g.View(HeaderView)
	v.Clear()
	for key := range r.Headers {
		fmt.Fprintf(v, "%s: %s", key, r.Headers.Get(key))
	}

	ui.renderBody(g)

	ui.Info(g, "Response loaded!")
}

func (ui *UI) setView(g *gocui.Gui, view string) error {
	if err := ui.closePopup(g, ui.currentPopup); err != nil {
		return err
	}

	// Save cursor position before switch view
	cur := g.CurrentView()
	x, y := cur.Cursor()
	ui.cursors.Set(cur.Name(), x, y)

	if _, err := g.SetCurrentView(view); err != nil {
		return err
	}

	if ui.AutoUpdate && ui.hasChanged {
		ui.hasChanged = false
		return ui.updateResponse(g)
	}

	return nil
}

func (ui *UI) createPopupView(g *gocui.Gui, viewname string, w, h int) (*gocui.View, error) {
	maxX, maxY := g.Size()
	x := maxX/2 - w/2
	y := maxY/2 - h/2
	view, err := g.SetView(viewname, x, y, x+w, y+h)
	if err != nil && err != gocui.ErrUnknownView {
		return nil, err
	}

	return view, nil
}

func (ui *UI) closePopup(g *gocui.Gui, viewname string) error {
	if _, err := g.View(viewname); err != nil {
		if err == gocui.ErrUnknownView {
			return nil
		}
		return err
	}

	g.DeleteView(viewname)
	g.DeleteKeybindings(viewname)
	g.Cursor = true
	ui.currentPopup = ""
	return ui.setView(g, cicleable[ui.viewIndex])
}

func (ui *UI) openPopup(g *gocui.Gui, viewname string, x, y int) (*gocui.View, error) {
	view, err := ui.createPopupView(g, viewname, x, y)
	if err != nil {
		return nil, err
	}

	if err := ui.setView(g, view.Name()); err != nil {
		return nil, err
	}
	ui.currentPopup = viewname
	g.Cursor = false

	return view, nil
}

func (ui *UI) toggleHelp(g *gocui.Gui, help string) error {
	if ui.currentPopup == BindingsView {
		return ui.closePopup(g, BindingsView)
	}

	view, err := ui.openPopup(g, BindingsView, 40, strings.Count(help, "\n"))
	if err != nil {
		return err
	}

	view.Title = "Bindings"
	fmt.Fprint(view, help)

	return nil
}

func (ui *UI) toggleResponsesLoader(g *gocui.Gui) error {
	if ui.currentPopup == ResponsesView {
		return ui.closePopup(g, ResponsesView)
	}

	if err := ui.responses.Load(ui.configPath); err != nil {
		return err
	}

	if ui.responses.Len() == 0 {
		return errors.New("No responses has been saved")
	}

	popup, err := ui.openPopup(g, ResponsesView, 30, ui.responses.Len()+1)
	if err != nil {
		return err
	}

	cx, _ := g.CurrentView().Cursor()
	onUp := func(g *gocui.Gui, v *gocui.View) error {
		ui.responses.Prev()
		v.SetCursor(cx, ui.responses.Index())
		return nil
	}

	onDown := func(g *gocui.Gui, v *gocui.View) error {
		ui.responses.Next()
		v.SetCursor(cx, ui.responses.Index())
		return nil
	}

	onDelete := func(g *gocui.Gui, v *gocui.View) error {
		key := ui.responses.Keys()[ui.responses.Index()]
		ui.responses.Del(key)
		if err := ui.responses.Save(ui.configPath); err != nil {
			return err
		}

		if err := ui.closePopup(g, ResponsesView); err != nil {
			return err
		}

		if err := ui.toggleResponsesLoader(g); err != nil {
			return nil
		}

		return nil
	}

	onEnter := func(g *gocui.Gui, v *gocui.View) error {
		ui.restoreResponse(g, ui.responses.Cur())
		return nil
	}

	onQuit := func(g *gocui.Gui, v *gocui.View) error {
		return ui.closePopup(g, ResponsesView)
	}

	view := []string{popup.Name()}
	(&bindings{
		{gocui.KeyArrowUp, "", "", view, func(*UI) ActionFn { return onUp }},
		{gocui.KeyArrowDown, "", "", view, func(*UI) ActionFn { return onDown }},
		{gocui.KeyEnter, "", "", view, func(*UI) ActionFn { return onEnter }},
		{'d', "", "", view, func(*UI) ActionFn { return onDelete }},
		{'q', "", "", view, func(*UI) ActionFn { return onQuit }},
	}).Apply(ui, g)

	for _, key := range ui.responses.Keys() {
		fmt.Fprintf(popup, "%s > %d\n", key, ui.responses.Get(key).Status)
	}

	popup.Title = "Responses"
	popup.Highlight = true
	return nil
}

func (ui *UI) toggleResponseBuilder(g *gocui.Gui) error {
	ui.hideResponseBuilder = !ui.hideResponseBuilder
	if ui.hideResponseBuilder {
		_, err := g.SetCurrentView(RequestView)
		return err
	}
	return nil
}

func (ui *UI) openSavePopup(g *gocui.Gui, title string, fn func(*gocui.Gui, string) error) error {
	if err := ui.closePopup(g, ui.currentPopup); err != nil {
		return err
	}

	popup, err := ui.openPopup(g, SaveView, max(20, len(title)+3), 2)
	if err != nil {
		return err
	}

	onEnter := func(g *gocui.Gui, v *gocui.View) error {
		value := strings.Trim(v.Buffer(), " \n")
		if err := fn(g, value); err != nil {
			ui.Info(g, err.Error())
		}
		return ui.closePopup(g, SaveView)
	}

	if err := g.SetKeybinding(popup.Name(), gocui.KeyEnter, gocui.ModNone, onEnter); err != nil {
		return err
	}

	popup.Title = title
	popup.Editable = true
	g.Cursor = true
	return nil
}

func (ui *UI) saveResponsePopup(g *gocui.Gui) error {
	fn := func(g *gocui.Gui, name string) error {
		return ui.saveResponseAs(g, name)
	}
	return ui.openSavePopup(g, "Save Response as...", fn)
}

func (ui *UI) saveResponseAs(g *gocui.Gui, name string) error {
	resp, err := ui.currentResponse(g)
	if err != nil {
		return err
	}

	ui.responses.Add(name, resp)
	if err := ui.responses.Save(ui.configPath); err != nil {
		return err
	}

	ui.Info(g, "Response applied and saved as '%s'", name)
	return nil
}

func (ui *UI) saveRequestPopup(g *gocui.Gui) error {
	// Only open the popup if there's requests
	if len(ui.requests) == 0 {
		ui.Info(g, "No Requests to save")
		return nil
	}

	fn := func(g *gocui.Gui, name string) error {
		return ui.saveRequestAs(g, name)
	}

	return ui.openSavePopup(g, "Save Request as...", fn)
}

func (ui *UI) saveRequestAs(g *gocui.Gui, name string) error {
	ui.reqLock.Lock()
	defer ui.reqLock.Unlock()
	if len(ui.requests) == 0 {
		return nil
	}
	req := ui.requests[ui.currentRequest]

	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(httplab.Decolorize(req)); err != nil {
		return err
	}

	ui.Info(g, "Request saved as '%s'", name)
	return nil
}

func (ui *UI) renderBody(g *gocui.Gui) error {
	v, err := g.View(BodyView)
	if err != nil {
		return err
	}

	body := ui.resp.Body

	v.Title = fmt.Sprintf("Body (%s)", body.Mode)
	v.Clear()
	v.Write(body.Info())
	return nil
}

func (ui *UI) openBodyFilePopup(g *gocui.Gui) error {
	if err := ui.closePopup(g, ui.currentPopup); err != nil {
		return err
	}

	popup, err := ui.openPopup(g, FileDialogView, 20, 2)
	if err != nil {
		return err
	}

	g.Cursor = true
	popup.Title = "Open Body File"
	popup.Editable = true

	onEnter := func(g *gocui.Gui, v *gocui.View) error {
		path := strings.Trim(v.Buffer(), " \n")
		if path == "" {
			return ui.closePopup(g, popup.Name())
		}

		if err := ui.resp.Body.SetFile(path); err != nil {
			ui.Info(g, "%+v", err)
		} else {
			if err := ui.renderBody(g); err != nil {
				return err
			}
		}
		return ui.closePopup(g, popup.Name())
	}

	return g.SetKeybinding(popup.Name(), gocui.KeyEnter, gocui.ModNone, onEnter)
}

func (ui *UI) nextBodyMode(g *gocui.Gui) error {
	modes := []httplab.BodyMode{
		httplab.BodyInput,
		httplab.BodyFile,
	}
	body := &ui.resp.Body
	body.Mode = body.Mode%httplab.BodyMode(len(modes)) + 1
	return ui.renderBody(g)
}

func (ui *UI) toggleLineWrap(g *gocui.Gui) error {
	view, err := g.View(RequestView)
	if err != nil {
		return err
	}

	view.Wrap = !view.Wrap
	return nil
}
