package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
)

var cicleable = []string{
	"status",
	"delay",
	"headers",
	"body",
	"request",
}

type editor struct {
	ui            *UI
	g             *gocui.Gui
	handler       gocui.Editor
	backTabEscape bool
}

func newEditor(ui *UI, g *gocui.Gui, handler gocui.Editor) *editor {
	if handler == nil {
		handler = gocui.DefaultEditor
	}

	return &editor{ui, g, handler, false}
}

func (e *editor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	if ch == '[' && mod == gocui.ModAlt {
		e.backTabEscape = true
		return
	}

	if e.backTabEscape {
		if ch == 'Z' {
			e.ui.prevView(e.g)
			e.backTabEscape = false
			return
		}
	}

	e.handler.Edit(v, key, ch, mod)
}

type motionEditor struct{}

func (e *motionEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	_, y := v.Cursor()
	maxY := strings.Count(v.Buffer(), "\n")
	switch {
	case key == gocui.KeyArrowDown:
		if y < maxY {
			v.MoveCursor(0, 1, true)
		}
	case key == gocui.KeyArrowUp:
		v.MoveCursor(0, -1, false)
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		v.MoveCursor(1, 0, false)
	}
}

type numberEditor struct {
	maxLength int
}

func (e *numberEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	x, _ := v.Cursor()
	switch {
	case ch >= 48 && ch <= 57:
		if len(v.Buffer()) > e.maxLength+1 {
			return
		}
		gocui.DefaultEditor.Edit(v, key, ch, mod)
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		if x < len(v.Buffer())-1 {
			v.MoveCursor(1, 0, false)
		}
	}
}

type UI struct {
	resp         *Response
	responses    Responses
	infoTimer    *time.Timer
	viewIndex    int
	currentPopup string
	configPath   string

	reqLock        sync.Mutex
	requests       [][]byte
	currentRequest int
}

func NewUI(configPath string) *UI {
	return &UI{
		resp: &Response{
			Status: 200,
			Headers: http.Header{
				"X-Server": []string{"HTTPLab"},
			},
			Body: Body{
				Mode:  BodyInput,
				Input: []byte("Hello, World"),
			},
		},
		configPath: configPath,
	}
}

func (ui *UI) Init(g *gocui.Gui) error {
	g.Cursor = true
	g.Highlight = true
	g.SelFgColor = gocui.ColorGreen

	g.SetManager(ui)
	return Bindings.Apply(ui, g)
}

func (ui *UI) AddRequest(g *gocui.Gui, req *http.Request) error {
	ui.reqLock.Lock()
	defer ui.reqLock.Unlock()

	ui.Info(g, "New Request from "+req.Host)
	buf, err := DumpRequest(req)
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

	view, err := g.View("request")
	if err != nil {
		return err
	}

	view.Title = fmt.Sprintf("Request (%d/%d)", ui.currentRequest+1, len(ui.requests))
	return ui.Display(g, "request", req)
}

func (ui *UI) Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	splitX := NewSplit(maxX).Relative(70)
	splitY := NewSplit(maxY).Fixed(maxY - 4)

	if v, err := g.SetView("request", 0, 0, splitX.Next(), splitY.Next()); err != nil {
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

	if _, err := g.SetView("info", 0, splitY.Current()+1, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}

	if v := g.CurrentView(); v == nil {
		_, err := g.SetCurrentView("status")
		return err
	}

	return nil
}

func (ui *UI) setResponseView(g *gocui.Gui, x0, y0, x1, y1 int) error {
	split := NewSplit(y1).Fixed(2, 3).Relative(40)
	if v, err := g.SetView("status", x0, y0, x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Status"
		v.Editable = true
		v.Editor = newEditor(ui, g, &numberEditor{3})
		fmt.Fprintf(v, "%d", ui.resp.Status)
	}

	if v, err := g.SetView("delay", x0, split.Current()+1, x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Delay (ms) "
		v.Editable = true
		v.Editor = newEditor(ui, g, &numberEditor{9})
		fmt.Fprintf(v, "%d", ui.resp.Delay/time.Millisecond)
	}

	if v, err := g.SetView("headers", x0, split.Current()+1, x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
		v.Editor = newEditor(ui, g, nil)
		v.Title = "Headers"
		for key := range ui.resp.Headers {
			fmt.Fprintf(v, "%s: %s\n", key, ui.resp.Headers.Get(key))
		}
	}

	if v, err := g.SetView("body", x0, split.Current()+1, x1, y1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
		v.Editor = newEditor(ui, g, nil)
		ui.renderBody(g)
	}

	return nil
}

func (ui *UI) Info(g *gocui.Gui, format string, args ...interface{}) {
	v, err := g.View("info")
	if v == nil || err != nil {
		return
	}

	v.Clear()
	fmt.Fprintf(v, format, args...)

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

func (ui *UI) Response() *Response {
	return ui.resp
}

func (ui *UI) nextView(g *gocui.Gui) error {
	ui.viewIndex = (ui.viewIndex + 1) % len(cicleable)
	return ui.setView(g, cicleable[ui.viewIndex])
}

func (ui *UI) prevView(g *gocui.Gui) error {
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

func (ui *UI) currentResponse(g *gocui.Gui) (*Response, error) {
	status := getViewBuffer(g, "status")
	headers := getViewBuffer(g, "headers")

	resp, err := NewResponse(status, headers, "")
	if err != nil {
		return nil, err
	}

	resp.Body = ui.resp.Body

	delay := getViewBuffer(g, "delay")
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
		return err
	}

	ui.resp = resp
	return nil
}

func (ui *UI) restoreResponse(g *gocui.Gui, r *Response) {
	ui.resp = r

	var v *gocui.View
	v, _ = g.View("status")
	v.Clear()
	fmt.Fprintf(v, "%d", r.Status)

	v, _ = g.View("delay")
	v.Clear()
	fmt.Fprintf(v, "%d", r.Delay)

	v, _ = g.View("headers")
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

	_, err := g.SetCurrentView(view)
	return err
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
	if ui.currentPopup == "bindings" {
		return ui.closePopup(g, "bindings")
	}

	view, err := ui.openPopup(g, "bindings", 40, strings.Count(help, "\n"))
	if err != nil {
		return err
	}

	view.Title = "Bindings"
	fmt.Fprint(view, help)

	return nil
}

func (ui *UI) toggleResponsesLoader(g *gocui.Gui) error {
	if ui.currentPopup == "responses" {
		return ui.closePopup(g, "responses")
	}

	rs, err := LoadResponsesFromPath(ui.configPath)
	if err != nil {
		return err
	}

	if len(rs) == 0 {
		return errors.New("No responses has been saved")
	}

	popup, err := ui.openPopup(g, "responses", 30, len(rs)+1)
	if err != nil {
		return err
	}

	onEnter := func(g *gocui.Gui, v *gocui.View) error {
		_, y := v.Cursor()
		line, err := v.Line(y)
		if err != nil {
			return err
		}

		ui.selectResponse(g, line)
		return nil
	}

	if err := g.SetKeybinding(popup.Name(), gocui.KeyEnter, gocui.ModNone, onEnter); err != nil {
		return err
	}

	for key := range rs {
		fmt.Fprintf(popup, "%s\n", rs.String(key))
	}

	popup.Title = "Responses"
	popup.Highlight = true

	ui.responses = rs
	return nil
}

func (ui *UI) selectResponse(g *gocui.Gui, s string) {
	if r := ui.responses.FromString(s); r != nil {
		ui.restoreResponse(g, r)
	}
}

func (ui *UI) saveResponsePopup(g *gocui.Gui) error {
	if err := ui.closePopup(g, ui.currentPopup); err != nil {
		return err
	}

	popup, err := ui.openPopup(g, "save", 20, 2)
	if err != nil {
		return err
	}

	onEnter := func(g *gocui.Gui, v *gocui.View) error {
		name := strings.Trim(v.Buffer(), " \n")
		if err := ui.saveResponseAs(g, name); err != nil {
			ui.Info(g, "%v", err)
		}
		return ui.closePopup(g, "save")
	}

	if err := g.SetKeybinding(popup.Name(), gocui.KeyEnter, gocui.ModNone, onEnter); err != nil {
		return err
	}

	popup.Title = "Save as..."
	popup.Editable = true
	return nil
}

func (ui *UI) saveResponseAs(g *gocui.Gui, name string) error {
	rs, err := LoadResponsesFromPath(ui.configPath)
	if err != nil {
		return err
	}
	if rs == nil {
		rs = make(Responses)
	}

	resp, err := ui.currentResponse(g)
	if err != nil {
		return err
	}

	rs[name] = resp
	if err := rs.SaveResponsesToPath(ui.configPath); err != nil {
		return err
	}

	ui.Info(g, "Response applied and saved as '%s'", name)
	return nil
}

func (ui *UI) renderBody(g *gocui.Gui) error {
	v, err := g.View("body")
	if err != nil {
		return err
	}

	body := ui.resp.Body

	v.Title = fmt.Sprintf("Body (%s)", body.Mode)
	v.Clear()
	v.Write(body.Info())
	return nil
}

func (ui *UI) nextBodyMode(g *gocui.Gui) error {
	modes := []BodyMode{BodyInput, BodyFile}
	body := &ui.resp.Body
	body.Mode = body.Mode%BodyMode(len(modes)) + 1
	return ui.renderBody(g)
}
