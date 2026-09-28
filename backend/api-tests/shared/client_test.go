package shared

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingTest struct {
	errors []string
}

func (r *recordingTest) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

type responseBody struct {
	io.Reader
	closeErr error
	closes   int
}

func (b *responseBody) Close() error {
	b.closes++
	return b.closeErr
}

type responseTransport func(*http.Request) (*http.Response, error)

func (f responseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCloseResponseBody(t *testing.T) {
	for _, closeErr := range []error{nil, errors.New("body close failed")} {
		t.Run(fmt.Sprint(closeErr), func(t *testing.T) {
			recorder := &recordingTest{}
			body := &responseBody{closeErr: closeErr}
			continued := false
			func() {
				defer func() { continued = true }()
				defer closeResponseBody(recorder, body, http.MethodPost, "/api/v1/project/get")
			}()
			assert.Equal(t, 1, body.closes)
			assert.True(t, continued, "other deferred cleanup must run")
			if closeErr == nil {
				assert.Empty(t, recorder.errors)
			} else {
				require.Len(t, recorder.errors, 1)
				assert.Contains(t, recorder.errors[0], closeErr.Error())
				assert.Contains(t, recorder.errors[0], "POST /api/v1/project/get")
			}
		})
	}
}

func TestClientResponseCleanup(t *testing.T) {
	for _, method := range []string{"health", http.MethodGet, http.MethodPost} {
		for _, decode := range []bool{false, true} {
			if method == "health" && decode {
				continue
			}
			t.Run(fmt.Sprintf("%s/decode=%t", method, decode), func(t *testing.T) {
				body := &responseBody{Reader: strings.NewReader(`{"id":42}`)}
				transport := responseTransport(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "http://backend.test", req.URL.Scheme+"://"+req.URL.Host)
					if method == "health" {
						assert.Equal(t, http.MethodGet, req.Method)
						assert.Equal(t, "/api/v1/health", req.URL.Path)
					} else {
						assert.Equal(t, method, req.Method)
						assert.Equal(t, "/resource", req.URL.Path)
					}
					if method == http.MethodPost {
						assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
						payload, err := io.ReadAll(req.Body)
						require.NoError(t, err)
						assert.JSONEq(t, `{"name":"fixture"}`, string(payload))
					}
					return &http.Response{StatusCode: http.StatusCreated, Body: body}, nil
				})
				client := &Client{baseURL: "http://backend.test", http: &http.Client{Transport: transport}}
				var result struct{ ID int }
				var out any
				if decode {
					out = &result
				}
				switch method {
				case "health":
					previous := http.DefaultTransport
					t.Cleanup(func() { http.DefaultTransport = previous })
					http.DefaultTransport = transport
					t.Setenv("API_TEST_BASE_URL", "http://backend.test")
					client = NewClient(t)
					assert.Equal(t, "http://backend.test", client.BaseURL())
					assert.Equal(t, 5*time.Second, client.http.Timeout)
				case http.MethodGet:
					assert.Equal(t, http.StatusCreated, client.Get(t, "/resource", out))
				case http.MethodPost:
					assert.Equal(t, http.StatusCreated, client.Post(t, "/resource", map[string]string{"name": "fixture"}, out))
				}
				assert.Equal(t, 1, body.closes)
				if decode && method != "health" {
					assert.Equal(t, 42, result.ID)
				}
			})
		}
	}
}

// Run real fatal assertions in a child test process so their failures stay observable.
func TestClientCleanupFailures(t *testing.T) {
	for _, scenario := range []struct{ mode, primary, context string }{
		{"health", "", "GET /api/v1/health"},
		{"get-decode", "invalid character", "GET /resource"},
		{"post-decode", "response body: invalid", "POST /resource"},
		{"post-read", "unexpected EOF", "POST /resource"},
	} {
		t.Run(scenario.mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestClientCleanupFailureProcess$", "-test.v")
			cmd.Env = append(os.Environ(), "MCH_CLIENT_FAILURE="+scenario.mode)
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr, "%s", output)
			assert.Equal(t, 1, exitErr.ExitCode(), "%s", output)
			text := string(output)
			assert.Contains(t, text, "body close failed")
			assert.Contains(t, text, scenario.context)
			assert.Contains(t, text, "response closes=1")
			assert.Contains(t, text, "other cleanup ran")
			if scenario.primary != "" {
				assert.Contains(t, text, scenario.primary)
				assert.NotContains(t, text, "request unexpectedly returned")
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestClientCleanupFailureProcess(t *testing.T) {
	mode := os.Getenv("MCH_CLIENT_FAILURE")
	if mode == "" {
		return // Only the parent test selects these deliberately failing scenarios.
	}
	body := &responseBody{Reader: strings.NewReader("invalid"), closeErr: errors.New("body close failed")}
	if mode == "post-read" {
		body.Reader = failingReader{}
	}
	t.Cleanup(func() { t.Log("other cleanup ran") })
	t.Cleanup(func() { t.Logf("response closes=%d", body.closes) })
	transport := responseTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	})
	client := &Client{baseURL: "http://backend.test", http: &http.Client{Transport: transport}}
	var out any
	switch mode {
	case "health":
		previous := http.DefaultTransport
		t.Cleanup(func() { http.DefaultTransport = previous })
		http.DefaultTransport = transport
		t.Setenv("API_TEST_BASE_URL", client.baseURL)
		NewClient(t)
	case "get-decode":
		client.Get(t, "/resource", &out)
	case "post-decode", "post-read":
		client.Post(t, "/resource", nil, &out)
	default:
		t.Fatalf("unknown failure mode %q", mode)
	}
	t.Log("request unexpectedly returned")
}
