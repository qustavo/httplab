package httplab

import (
	"bytes"
	"fmt"
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
		fmt.Printf("%s\n", buf)
	})

	t.Run("should be displayed as is", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/invalidJSON", bytes.NewBuffer(
			[]byte(`invalid json`),
		))
		req.Header.Set("Content-Type", "application/json")

		buf, err := DumpRequest(req)
		require.NoError(t, err)
		fmt.Printf("%s\n", buf)
	})
}

func TestDecolorization(t *testing.T) {
	for i, _ := range [107]struct{}{} {
		text := "Some Text"
		nocolor := Decolorize([]byte(withColor(i, text)))
		assert.Equal(t, text, string(nocolor))
	}
}

func TestAlphabetizedHeaders(t *testing.T) {
	// TODO: randomize this
	keys := []string{"Z5", "A1", "M3", "R4", "C2"}
	sortedKeys := []string{"A1", "C2", "M3", "R4", "Z5"}

	req, _ := http.NewRequest("GET", "/", bytes.NewBuffer(nil))
	for _, k := range keys {
		req.Header.Set(k, "")
	}

	buf, err := DumpRequest(req)
	require.NoError(t, err)
	decBuf := Decolorize(buf)
	priorIdx := -1
	for _, val := range sortedKeys {
		foundIdx := strings.Index(string(decBuf), val)
		assert.True(t, foundIdx > priorIdx)
		priorIdx = foundIdx
	}
}
