package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jroimartin/gocui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestUI(t *testing.T) (*gocui.Gui, *UI) {
	g, err := gocui.NewGui(gocui.Output256)
	require.NoError(t, err)

	ui := NewUI("")
	require.NoError(t, ui.Layout(g))

	return g, ui
}

func TestUIAddRequestSavesInOrder(t *testing.T) {
	g, ui := newTestUI(t)
	defer g.Close()

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/%d", i), &bytes.Buffer{})
		require.NoError(t, ui.AddRequest(g, req))
	}

	assert.Len(t, ui.requests, 10)
	for i := 0; i < 10; i++ {
		req := ui.requests[i]
		split := strings.Split(string(req), " ")
		path := split[1]
		assert.Equal(t, fmt.Sprintf("/%d", i), path)
	}

	assert.Equal(t, 9, ui.currentRequest)
}

func TestUIScrollRequests(t *testing.T) {
	g, ui := newTestUI(t)
	defer g.Close()

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/%d", i), &bytes.Buffer{})
		require.NoError(t, ui.AddRequest(g, req))
	}

	cur := ui.currentRequest
	ui.prevRequest(g, nil)
	assert.Equal(t, ui.currentRequest, cur-1)

	ui.nextRequest(g, nil)
	assert.Equal(t, ui.currentRequest, cur)

	t.Run("Doesn't autoscroll", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", &bytes.Buffer{})
		ui.prevRequest(g, nil)
		cur := ui.currentRequest

		ui.AddRequest(g, req)
		assert.Equal(t, cur, ui.currentRequest)
	})
}
