package httplab

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDumpRequestWithJSON(t *testing.T) {
	t.Run("should be indented", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/withJSON", bytes.NewBuffer(
			[]byte(`{"foo": "bar", "a": [1,2,3]}`),
		))
		req.Header.Set("Content-Type", "application/json")

		buf, err := DumpRequest(req)
		require.NoError(t, err)
		t.Logf("%s\n", buf)
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

func TestDecolorization(t *testing.T) {
	for i, _ := range [107]struct{}{} {
		text := "Some Text"
		nocolor := Decolorize([]byte(withColor(i, text)))
		assert.Equal(t, text, string(nocolor))
	}
}
