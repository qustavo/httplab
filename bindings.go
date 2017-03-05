package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/jroimartin/gocui"
)

type ActionFn func(*gocui.Gui, *gocui.View) error

type binding struct {
	keyCode interface{}
	keyName string
	help    string
	views   []string
	action  func(*UI) ActionFn
}

type bindings []binding

func (bs bindings) Apply(ui *UI, g *gocui.Gui) error {
	for _, b := range bs {
		if b.action == nil {
			continue
		}

		views := b.views
		if len(views) == 0 {
			views = []string{""}
		}

		for _, v := range views {
			err := g.SetKeybinding(v, b.keyCode, gocui.ModNone, b.action(ui))
			if err != nil {
				return err
			}
		}
	}

	return g.SetKeybinding("", gocui.KeyCtrlH, gocui.ModNone, func(g *gocui.Gui, _ *gocui.View) error {
		return ui.toggleHelp(g, bs.Help())
	})
}

func (bs bindings) Help() string {
	buf := &bytes.Buffer{}
	w := tabwriter.NewWriter(buf, 0, 0, 3, ' ', tabwriter.DiscardEmptyColumns)
	for _, b := range bs {
		if b.keyName == "" || b.help == "" {
			continue
		}
		fmt.Fprintf(w, "  %s\t: %s\n", b.keyName, b.help)
	}

	fmt.Fprintf(w, "  %s\t: %s\n", "Ctrl+h", "Toggle Help")
	w.Flush()
	return buf.String()
}

var Bindings = &bindings{
	{gocui.KeyTab, "Tab", "Next Input", nil, onNextView},
	{0xFF, "Shift+Tab", "Previous Input", nil, nil}, // only to display on help
	{gocui.KeyCtrlA, "Ctrl+a", "Update Response", nil, onUpdateResponse},
	{gocui.KeyCtrlS, "Ctrl+s", "Save Response as", nil, onSaveResponseAs},
	{gocui.KeyCtrlL, "Ctrl+l", "Toggle Responses list", nil, onToggleResponsesList},
	{'q', "q", "Close Popup", []string{"bindings", "responses"}, onClosePopup},
	{gocui.KeyPgup, "PgUp", "Previous Request", nil, onPrevRequest},
	{gocui.KeyPgdn, "PgDown", "Next Request", nil, onNextRequest},
	{gocui.KeyCtrlC, "Ctrl+c", "Quit", nil, onQuit},

	// Bindings not displayed by help
	{gocui.KeyArrowUp, "", "", []string{"responses"}, onCursorUp},
	{gocui.KeyArrowDown, "", "", []string{"responses"}, onCursorDown},
	{gocui.KeyEnter, "", "", []string{"responses"}, onSelectResponse},
	{gocui.KeyEnter, "", "", []string{"save"}, onSaveResponseName},
}

func onNextView(ui *UI) ActionFn {
	return func(g *gocui.Gui, _ *gocui.View) error {
		return ui.nextView(g)
	}
}

func onUpdateResponse(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		ui.updateResponse(g)
		return nil
	}
}

func onSaveResponseAs(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		return ui.saveResponsePopup(g)
	}
}

func onToggleResponsesList(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		if err := ui.toggleResponsesLoader(g); err != nil {
			ui.Info(g, err.Error())
		}
		return nil
	}
}

func onPrevRequest(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		return ui.prevRequest(g)
	}
}

func onNextRequest(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		return ui.nextRequest(g)
	}
}

func onClosePopup(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		return ui.closePopup(g, v.Name())
	}
}

func onQuit(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		return gocui.ErrQuit
	}
}

func onCursorUp(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		cx, cy := v.Cursor()
		v.SetCursor(cx, cy-1)
		return nil
	}
}

func onCursorDown(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		cx, cy := v.Cursor()
		v.SetCursor(cx, cy+1)
		return nil
	}
}

func onSelectResponse(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		_, y := v.Cursor()
		line, err := v.Line(y)
		if err != nil {
			return err
		}

		ui.selectResponse(g, line)
		return nil
	}
}

func onSaveResponseName(ui *UI) ActionFn {
	return func(g *gocui.Gui, v *gocui.View) error {
		defer func() {
			ui.closePopup(g, "save")
		}()

		name := strings.Trim(v.Buffer(), " \n")
		if err := ui.saveResponseAs(g, name); err != nil {
			ui.Info(g, "%v", err)
		}
		return nil
	}
}
