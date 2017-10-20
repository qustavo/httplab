package httplab

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sort"
	"strings"
)

func TestDumpRequestWithJSON(t *testing.T) {
	t.Run("should be indented", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/withJSON", bytes.NewBuffer(
			[]byte(`{"foo": "bar", "a": [1,2,3]}`),
		))
		req.Header.Set("Content-Type", "application/json")

		buf, err := DumpRequest(req)
		require.NoError(t, err)
		fmt.Printf("%s\n", buf)
	})

	t.Run("should be displayed as is", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/invalidJSON", bytes.NewBuffer(
			[]byte(`invalid json`),
		))
		req.Header.Set("Content-Type", "application/json")

		buf, err := DumpRequest(req)
		require.NoError(t, err)
		t.Logf("%s\n", buf)
	})

	t.Run("Gzip", func(t *testing.T) {
		var gzbuf bytes.Buffer
		gz := gzip.NewWriter(&gzbuf)
		gz.Write([]byte(`{"foo": "bar", "a": [1,2,3]}`))
		gz.Close()

		req, _ := http.NewRequest("GET", "/withJSON", &gzbuf)
		req.Header.Set("Content-Type", "gzip")

		buf, err := DumpRequest(req)
		require.NoError(t, err)
		require.True(t, strings.Contains(string(buf), `"foo": "bar"`))
		t.Logf("%s\n", buf)
	})

	t.Run("Invalid Gzip", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/withJSON", bytes.NewBuffer(
			[]byte(`This is not a gzip`),
		))
		req.Header.Set("Content-Type", "gzip")

		buf, err := DumpRequest(req)
		require.Error(t, err)
		t.Logf("%s\n", buf)
	})
}

func TestDumpRequestHeaders(t *testing.T) {
	t.Run("request headers should be dumped in sorted order", func(t *testing.T) {

		keys := []string{"B", "A", "C", "D", "E", "F", "H", "G", "I"}
		req, _ := http.NewRequest("GET", "/", bytes.NewBuffer(nil))
		for _, k := range keys {
			req.Header.Set(k, "")
		}

		buf, err := DumpRequest(req)
		require.NoError(t, err)
		sort.Strings(keys)

		startLine := "GET / HTTP/1.1\n"
		response := startLine + strings.Join(keys, ": \n") + ": \n"

		assert.Contains(t, response, string(Decolorize(buf)))
	})
}

func TestDecolorization(t *testing.T) {
	for i, _ := range [107]struct{}{} {
		text := "Some Text"
		nocolor := Decolorize([]byte(withColor(i, text)))
		assert.Equal(t, text, string(nocolor))
	}
}
