package ui

import (
	"strings"

	"github.com/jroimartin/gocui"
)

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

	// prevent infinite scrolling
	if (key == gocui.KeyArrowDown || key == gocui.KeyArrowRight) && mod == gocui.ModNone {
		_, cy := v.Cursor()
		if _, err := v.Line(cy); err != nil {
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
