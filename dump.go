package httplab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func valueOrDefault(value, def string) string {
	if value == "" {
		return def
	}
	return value
}

func withColor(color int, text string) string {
	return fmt.Sprintf("\x1b[0;%dm%s\x1b[0;0m", color, text)
}

func DumpRequest(req *http.Request) []byte {
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

	for key, _ := range req.Header {
		val := req.Header.Get(key)
		fmt.Fprintf(buf, "%s: %s\n", withColor(31, key), withColor(32, val))
	}
	buf.WriteRune('\n')

	if req.Body == nil {
		return buf.Bytes()
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Fprintf(buf, "Error reading body: %+v", err)
		return buf.Bytes()
	}

	if strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		json.Indent(buf, body, "", "  ")

	}

	return buf.Bytes()
}
