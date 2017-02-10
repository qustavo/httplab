package httplab

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
)

const (
	RequestView = "request"
	StatusView  = "status"
	DelayView   = "delay"
	HeadersView = "headers"
	BodyView    = "body"
	InfoView    = "info"
)

var cicleable = []string{
	StatusView,
	DelayView,
	HeadersView,
	BodyView,
	RequestView,
}

type UI struct {
	*gocui.Gui
	resp *Response
}

func NewUI() (*UI, error) {
	gui, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return nil, err
	}

	gui.Cursor = true
	gui.Highlight = true
	gui.SelFgColor = gocui.ColorGreen

	ui := &UI{Gui: gui, resp: &Response{
		Status: 200,
		Headers: http.Header{
			"X-Server": []string{"HTTPLab"},
		},
		Body: []byte("Hello, World"),
	}}

	gui.SetManager(ui)
	return ui, nil
}

func (ui *UI) Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	splitX := NewSplit(maxX).Relative(70)
	splitY := NewSplit(maxY).Fixed(maxY - 4)

	if v, err := g.SetView(RequestView, 0, 0, splitX.Next(), splitY.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Request"
		v.Editable = true
		v.Editor = gocui.EditorFunc(readOnlyEditor)
	}

	if err := ui.setResponseView(splitX.Current(), 0, maxX-1, splitY.Current()); err != nil {
		return err
	}

	if _, err := ui.SetView(InfoView, 0, splitY.Current()+1, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}

	return nil
}

func (ui *UI) setResponseView(x0, y0, x1, y1 int) error {
	split := NewSplit(y1).Fixed(2, 3).Relative(40)
	if v, err := ui.SetView(StatusView, x0, y0, x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Status"
		v.Editable = true
		v.Editor = gocui.EditorFunc(statusEditor)
		fmt.Fprintf(v, "%d", ui.resp.Status)
	}

	if v, err := ui.SetView(DelayView, x0, split.Current()+1, x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Delay (ms) "
		v.Editable = true
		fmt.Fprintf(v, "%d", ui.resp.Delay/time.Millisecond)
	}

	if v, err := ui.SetView(HeadersView, x0, split.Current()+1, x1, split.Next()); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
		v.Title = "Headers"
		for key, _ := range ui.resp.Headers {
			fmt.Fprintf(v, "%s: %s\n", key, ui.resp.Headers.Get(key))
		}
	}

	if v, err := ui.SetView(BodyView, x0, split.Current()+1, x1, y1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
		v.Title = "Body"
		fmt.Fprintf(v, "%s", string(ui.resp.Body))
	}

	return nil
}

func (ui *UI) Display(view string, bytes []byte) error {
	v, err := ui.View(view)
	if err != nil {
		return err
	}

	ui.Execute(func(g *gocui.Gui) error {
		v.Clear()
		_, err := v.Write(bytes)
		return err
	})

	return nil
}

func (ui *UI) Response() *Response {
	return ui.resp
}

func (ui *UI) Loop() error {
	defer ui.Close()

	if err := ui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := ui.SetKeybinding("", gocui.KeyCtrlS, gocui.ModNone, saveResponse(ui)); err != nil {
		return err
	}

	if err := ui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, cicleViews); err != nil {
		return err
	}

	if err := ui.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}

func readOnlyEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
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
}

func statusEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch {
	case ch >= 48 && ch <= 57:
		if len(v.Buffer()) > 4 {
			return
		}
		v.EditWrite(ch)
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		v.MoveCursor(1, 0, false)
	}
}

func cicleViews(g *gocui.Gui, cur *gocui.View) error {
	next := cicleable[0]
	if cur == nil {
		_, err := g.SetCurrentView(next)
		return err
	}

	for i, view := range cicleable {
		if view == cur.Name() {
			next = cicleable[(i+1)%len(cicleable)]
		}
	}

	_, err := g.SetCurrentView(next)
	return err
}

func getViewBuffer(g *gocui.Gui, view string) string {
	v, err := g.View(view)
	if err != nil {
		return ""
	}
	return v.Buffer()
}

func saveResponse(ui *UI) func(g *gocui.Gui, v *gocui.View) error {
	fn := func(g *gocui.Gui, v *gocui.View) error {
		status := getViewBuffer(g, StatusView)
		headers := getViewBuffer(g, HeadersView)
		body := getViewBuffer(g, BodyView)

		bar, _ := g.View(InfoView)
		bar.Clear()
		resp, err := NewResponse(status, headers, body)
		if err != nil {
			bar.Write([]byte(fmt.Sprintf("%+v", err)))
			return nil
		}

		delay := getViewBuffer(g, DelayView)
		delay = strings.Trim(delay, " \n")
		intDelay, err := strconv.Atoi(delay)
		if err != nil {
			return fmt.Errorf("Invalid delay format: %+v", err)
		}

		resp.Delay = time.Duration(intDelay) * time.Millisecond

		bar.Write([]byte("Response saved!"))

		go func() {
			time.Sleep(time.Second * 3)
			g.Execute(func(_ *gocui.Gui) error {
				bar.Clear()
				return nil
			})
		}()

		ui.resp = resp
		return nil
	}

	return fn
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
