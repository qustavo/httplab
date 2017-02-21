package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

type viewEditor struct {
	app           *UI
	g             *gocui.Gui
	orig          gocui.Editor
	backTabEscape bool
}

func (e *viewEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	if ch == '[' && mod == gocui.ModAlt {
		e.backTabEscape = true
		return
	}

	if e.backTabEscape {
		if ch == 'Z' {
			e.app.prevView(e.g, nil)
			e.backTabEscape = false
			return
		}
	}

	e.orig.Edit(v, key, ch, mod)
}

type readOnlyEditor struct {
	editor gocui.Editor
}

func (e *readOnlyEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch {
	case key == gocui.KeyArrowDown:
		v.MoveCursor(0, 1, true)
	case key == gocui.KeyArrowUp:
		v.MoveCursor(0, -1, false)
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		v.MoveCursor(1, 0, false)
	}

	e.editor.Edit(v, key, ch, mod)
}

type statusEditor struct {
	editor gocui.Editor
}

func (e *statusEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch {
	case ch >= 48 && ch <= 57:
		if len(v.Buffer()) > 4 {
			return
		}
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		v.MoveCursor(1, 0, false)
	}

	e.editor.Edit(v, key, ch, mod)
}

var defaultEditor gocui.Editor

type UI struct {
	resp      *Response
	infoTimer *time.Timer
	viewIndex int
}

func NewUI() *UI {
	return &UI{
		resp: &Response{
			Status: 200,
			Headers: http.Header{
				"X-Server": []string{"HTTPLab"},
			},
			Body: []byte("Hello, World"),
		}}
}

func (ui *UI) Init(g *gocui.Gui) error {
	g.Cursor = true
	g.Highlight = true
	g.SelFgColor = gocui.ColorGreen

	defaultEditor = &viewEditor{ui, g, gocui.DefaultEditor, false}

	ui.Layout(g)
	ui.bindKeys(g)
	return nil
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
		v.Editor = &readOnlyEditor{defaultEditor}
	}

	if err := ui.setResponseView(g, splitX.Current(), 0, maxX-1, splitY.Current()); err != nil {
		return err
	}

	if _, err := g.SetView("info", 0, splitY.Current()+1, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}

	ui.setView(g, "status")

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
		v.Editor = &statusEditor{defaultEditor}
		fmt.Fprintf(v, "%d", ui.resp.Status)
	}

	if v, err := g.SetView("delay", x0, split.Current()+1, x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Delay (ms) "
		v.Editable = true
		v.Editor = defaultEditor
		fmt.Fprintf(v, "%d", ui.resp.Delay/time.Millisecond)
	}

	if v, err := g.SetView("headers", x0, split.Current()+1, x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
		v.Editor = defaultEditor
		v.Title = "Headers"
		for key, _ := range ui.resp.Headers {
			fmt.Fprintf(v, "%s: %s\n", key, ui.resp.Headers.Get(key))
		}
	}

	if v, err := g.SetView("body", x0, split.Current()+1, x1, y1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Body"
		v.Editable = true
		v.Editor = defaultEditor
		fmt.Fprintf(v, "%s", string(ui.resp.Body))
	}

	return nil
}

func (ui *UI) setView(g *gocui.Gui, view string) error {
	_, err := g.SetCurrentView(view)
	return err
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

func (ui *UI) bindKeys(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlS, gocui.ModNone, ui.saveResponse); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, ui.nextView); err != nil {
		return err
	}

	return nil
}

func (ui *UI) nextView(g *gocui.Gui, _ *gocui.View) error {
	ui.viewIndex = (ui.viewIndex + 1) % len(cicleable)
	return ui.setView(g, cicleable[ui.viewIndex])
}

func (ui *UI) prevView(g *gocui.Gui, cur *gocui.View) error {
	ui.viewIndex = (ui.viewIndex - 1 + len(cicleable)) % len(cicleable)
	return ui.setView(g, cicleable[ui.viewIndex])
}

func getViewBuffer(g *gocui.Gui, view string) string {
	v, err := g.View(view)
	if err != nil {
		return ""
	}
	return v.Buffer()
}

func (ui *UI) saveResponse(g *gocui.Gui, v *gocui.View) error {
	status := getViewBuffer(g, "status")
	headers := getViewBuffer(g, "headers")
	body := getViewBuffer(g, "body")

	resp, err := NewResponse(status, headers, body)
	if err != nil {
		ui.Info(g, "%v", err)
		return nil
	}

	delay := getViewBuffer(g, "delay")
	delay = strings.Trim(delay, " \n")
	intDelay, err := strconv.Atoi(delay)
	if err != nil {
		ui.Info(g, "Can't parse '%s' as number", delay)
		return nil
	}
	resp.Delay = time.Duration(intDelay) * time.Millisecond

	ui.resp = resp
	ui.Info(g, "Response saved!")
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
