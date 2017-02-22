package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
)

func defaultConfigPath() string {
	u, err := user.Current()
	if err != nil {
		return ".httplab"
	}

	path := u.HomeDir + "/.httplab"
	return path
}

type Response struct {
	Status  int
	Headers http.Header
	Body    []byte
	Delay   time.Duration
}

func (r *Response) UnmarshalJSON(data []byte) error {
	type alias Response
	v := struct {
		alias
		Body    string
		Headers map[string]string
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	r.Status = v.Status
	r.Delay = v.Delay
	r.Body = []byte(v.Body)
	if r.Headers == nil {
		r.Headers = http.Header{}
	}
	for key, _ := range v.Headers {
		r.Headers.Set(key, v.Headers[key])
	}

	return nil
}

func (r *Response) MarshalJSON() ([]byte, error) {
	type alias Response
	v := struct {
		alias
		Body    string
		Headers map[string]string
	}{
		Headers: make(map[string]string),
	}

	v.Status = r.Status
	v.Body = string(r.Body)
	for key, _ := range r.Headers {
		v.Headers[key] = r.Headers.Get(key)
	}

	return json.MarshalIndent(v, "", "  ")
}

func NewResponse(status, headers, body string) (*Response, error) {
	// Parse Status
	status = strings.Trim(status, " \r\n")
	if status == "" {
		status = "200"
	}
	code, err := strconv.Atoi(status)
	if err != nil {
		return nil, fmt.Errorf("Status: %v", err)
	}

	if code < 100 || code > 599 {
		return nil, fmt.Errorf("Status should be between 100 and 599")
	}

	// Parse Headers
	hdr := http.Header{}
	lines := strings.Split(headers, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		kv := strings.Split(line, ":")
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		hdr.Set(key, val)
	}

	return &Response{
		Status:  code,
		Headers: hdr,
		Body:    []byte(body),
	}, nil
}

func (r *Response) Write(w http.ResponseWriter) error {
	for key, _ := range r.Headers {
		w.Header().Set(key, r.Headers.Get(key))
	}
	w.WriteHeader(r.Status)
	_, err := w.Write(r.Body)

	return err
}

func openConfigFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
}

func LoadResponses() (map[string]*Response, error) {
	return LoadResponsesFromPath(defaultConfigPath())
}

func LoadResponsesFromPath(path string) (map[string]*Response, error) {
	f, err := openConfigFile(path)
	if err != nil {
		return nil, err
	}

	return LoadResponsesFromFile(f)
}

func LoadResponsesFromFile(f *os.File) (map[string]*Response, error) {
	r := struct {
		Responses map[string]*Response
	}{}

	if err := json.NewDecoder(f).Decode(&r); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}

	return r.Responses, nil
}

func SaveResponses(rs map[string]*Response) error {
	return SaveResponsesToPath(rs, defaultConfigPath())
}

func SaveResponsesToPath(rs map[string]*Response, path string) error {
	f, err := openConfigFile(path)
	if err != nil {
		return err
	}

	return SaveResponsesToFile(rs, f)
}

func SaveResponsesToFile(rs map[string]*Response, f *os.File) error {
	return json.NewEncoder(f).Encode(struct {
		Responses map[string]*Response
	}{rs})
}
