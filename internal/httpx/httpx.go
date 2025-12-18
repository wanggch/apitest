package httpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"apitest/internal/config"
	"apitest/internal/templ"
)

const (
	// MaxResponseBodySize defines the maximum bytes to read from response body.
	MaxResponseBodySize = 2 * 1024 * 1024
)

// RequestInfo represents prepared request used in reporting.
type RequestInfo struct {
	Method  string
	URL     string
	Headers map[string]string
	Query   map[string]string
	Body    string
}

// ResponseInfo represents HTTP response data used in assertions and reporting.
type ResponseInfo struct {
	StatusCode    int
	Headers       http.Header
	Body          string
	BodyTruncated bool
	Duration      time.Duration
}

// BuildClient returns http.Client configured with insecure flag and timeout.
func BuildClient(timeout time.Duration, insecure bool) *http.Client {
	transport := &http.Transport{}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// DoRequest builds and executes HTTP request based on config.
func DoRequest(ctx context.Context, client *http.Client, baseURL string, req config.Request, vars map[string]string) (RequestInfo, ResponseInfo, error) {
	ri := RequestInfo{Headers: map[string]string{}, Query: map[string]string{}}

	// method
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	// url
	resolvedURL, err := templ.ApplyString(req.URL, vars)
	if err != nil {
		return ri, ResponseInfo{}, fmt.Errorf("url template: %w", err)
	}
	if baseURL != "" {
		if strings.HasPrefix(resolvedURL, "http://") || strings.HasPrefix(resolvedURL, "https://") {
			// use as is
		} else {
			resolvedURL = strings.TrimRight(baseURL, "/") + resolvedURL
		}
	}
	// query
	query := url.Values{}
	for k, v := range req.Query {
		strVal := fmt.Sprint(v)
		repl, err := templ.ApplyString(strVal, vars)
		if err != nil {
			return ri, ResponseInfo{}, fmt.Errorf("query %s: %w", k, err)
		}
		query.Set(k, repl)
		ri.Query[k] = repl
	}
	if len(query) > 0 {
		if strings.Contains(resolvedURL, "?") {
			resolvedURL += "&" + query.Encode()
		} else {
			resolvedURL += "?" + query.Encode()
		}
	}

	// headers
	hdr := http.Header{}
	for k, v := range req.Headers {
		repl, err := templ.ApplyString(v, vars)
		if err != nil {
			return ri, ResponseInfo{}, fmt.Errorf("header %s: %w", k, err)
		}
		hdr.Set(k, repl)
		ri.Headers[k] = repl
	}

	var body io.Reader
	var bodyText string
	if req.Body != nil {
		used := 0
		if req.Body.Raw != "" {
			used++
		}
		if req.Body.JSON != nil {
			used++
		}
		if req.Body.Form != nil {
			used++
		}
		if used > 1 {
			return ri, ResponseInfo{}, errors.New("only one of raw/json/form is allowed in body")
		}
		switch {
		case req.Body.Raw != "":
			var err error
			bodyText, err = templ.ApplyString(req.Body.Raw, vars)
			if err != nil {
				return ri, ResponseInfo{}, fmt.Errorf("body raw: %w", err)
			}
			body = strings.NewReader(bodyText)
		case req.Body.JSON != nil:
			replaced, err := templ.ApplyInterface(req.Body.JSON, vars)
			if err != nil {
				return ri, ResponseInfo{}, fmt.Errorf("body json: %w", err)
			}
			data, err := json.Marshal(replaced)
			if err != nil {
				return ri, ResponseInfo{}, fmt.Errorf("body json marshal: %w", err)
			}
			bodyText = string(data)
			body = bytes.NewReader(data)
			if hdr.Get("Content-Type") == "" {
				hdr.Set("Content-Type", "application/json")
			}
		case req.Body.Form != nil:
			vals := url.Values{}
			for k, v := range req.Body.Form {
				s := fmt.Sprint(v)
				repl, err := templ.ApplyString(s, vars)
				if err != nil {
					return ri, ResponseInfo{}, fmt.Errorf("body form %s: %w", k, err)
				}
				vals.Set(k, repl)
			}
			encoded := vals.Encode()
			bodyText = encoded
			body = strings.NewReader(encoded)
			if hdr.Get("Content-Type") == "" {
				hdr.Set("Content-Type", "application/x-www-form-urlencoded")
			}
		}
	}

	reqObj, err := http.NewRequestWithContext(ctx, method, resolvedURL, body)
	if err != nil {
		return ri, ResponseInfo{}, fmt.Errorf("build request: %w", err)
	}
	reqObj.Header = hdr
	ri.Method = method
	ri.URL = reqObj.URL.String()
	ri.Body = bodyText

	start := time.Now()
	resp, err := client.Do(reqObj)
	duration := time.Since(start)
	if err != nil {
		return ri, ResponseInfo{Duration: duration}, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	bodyLimit := MaxResponseBodySize + 1
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(bodyLimit)))
	if err != nil {
		return ri, ResponseInfo{Duration: duration}, fmt.Errorf("read body: %w", err)
	}
	truncated := len(data) > MaxResponseBodySize
	if truncated {
		data = data[:MaxResponseBodySize]
	}
	bodyStr := string(data)

	// For gzip or content enc specify? net/http handles decoding automatically unless disabled.
	// Ensure charset decoding? keep raw bytes to string.

	// Try to honor charset in Content-Type (basic)
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		if mediatype, params, err := mime.ParseMediaType(ct); err == nil {
			_ = mediatype
			if cs := strings.ToLower(params["charset"]); cs == "utf-8" || cs == "" {
				// nothing
			}
		}
	}

	return ri, ResponseInfo{
		StatusCode:    resp.StatusCode,
		Headers:       resp.Header.Clone(),
		Body:          bodyStr,
		BodyTruncated: truncated,
		Duration:      duration,
	}, nil
}
