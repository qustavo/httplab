package httplab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

var decolorizeRegex = regexp.MustCompile("\x1b\\[0;\\d+m")

func Decolorize(s []byte) []byte {
	return decolorizeRegex.ReplaceAll(s, nil)
}

func valueOrDefault(value, def string) string {
	if value == "" {
		return def
	}
	return value
}

func withColor(color int, text string) string {
	return fmt.Sprintf("\x1b[0;%dm%s\x1b[0;0m", color, text)
}

func writeBody(buf *bytes.Buffer, req *http.Request) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	if len(body) > 0 {
		buf.WriteRune('\n')
	}

	if strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		if err := json.Indent(buf, body, "", "  "); err == nil {
			return nil
		}
	}

	_, err = buf.Write(body)
	return err
}

func DumpRequest(req *http.Request) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	reqURI := req.RequestURI
	if reqURI == "" {
		reqURI = req.URL.RequestURI()
	}

	fmt.Fprintf(buf, "%s %s %s/%d.%d\n",
		withColor(35, valueOrDefault(req.Method, "GET")),
		reqURI,
		withColor(35, "HTTP"),
		req.ProtoMajor,
		req.ProtoMinor,
	)

	var keys []string
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := req.Header.Get(key)
		fmt.Fprintf(buf, "%s: %s\n", withColor(31, key), withColor(32, val))
	}

	err := writeBody(buf, req)
	return buf.Bytes(), err
}
