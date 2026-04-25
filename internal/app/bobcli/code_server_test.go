package bobcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"bob/internal/protocol"
)

func TestRunCodeServerBuildsURLFromCWD(t *testing.T) {
	ctx := newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s"}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"code-server"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}
	ctx.assertOpened(t, "http", "127.0.0.1:8080", ctx.cwd)
}

func TestRunCodeServerResolvesAbsolutePath(t *testing.T) {
	ctx := newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s"}`)
	if err := os.MkdirAll(filepath.Join(ctx.cwd, "subdir"), 0o700); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"code-server", "subdir"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}
	ctx.assertOpened(t, "http", "127.0.0.1:8080", filepath.Join(ctx.cwd, "subdir"))
}

func TestRunCodeServerEncodesFolderQuery(t *testing.T) {
	ctx := newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s"}`)
	path := filepath.Join(ctx.cwd, "space dir", "#hash")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"code-server", path}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}
	ctx.assertOpened(t, "http", "127.0.0.1:8080", path)
	if !strings.Contains(ctx.request.URL, "folder="+url.QueryEscape(path)) {
		t.Fatalf("opened URL = %q, want URL-encoded folder query", ctx.request.URL)
	}
}

func TestRunCodeServerPortFlagOverridesConfig(t *testing.T) {
	ctx := newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s","codeServer":{"port":65508}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"code-server", "--port", "9090", "."}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}
	ctx.assertOpened(t, "http", "127.0.0.1:9090", ctx.cwd)
}

func TestRunCodeServerUsesConfigPort(t *testing.T) {
	ctx := newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s","codeServer":{"port":65508}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"code-server", "."}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}
	ctx.assertOpened(t, "http", "127.0.0.1:65508", ctx.cwd)
}

func TestRunCodeServerEnvPortOverridesConfig(t *testing.T) {
	ctx := newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s","codeServer":{"port":65508}}`)
	t.Setenv("BOB_CODE_SERVER_PORT", "9090")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"code-server", "."}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}
	ctx.assertOpened(t, "http", "127.0.0.1:9090", ctx.cwd)
}

func TestRunCodeServerInvalidPortFailsClearly(t *testing.T) {
	newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s"}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"code-server", "--port", "70000", "."}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	got := strings.ToLower(stderr.String())
	if !strings.Contains(got, "port") || !strings.Contains(got, "65535") {
		t.Fatalf("stderr = %q, want clear invalid port message", stderr.String())
	}
}

func TestRunCodeServerInvalidConfigPortFailsClearly(t *testing.T) {
	newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s","codeServer":{"port":0}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"code-server", "."}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	got := strings.ToLower(stderr.String())
	if !strings.Contains(got, "port") || !strings.Contains(got, "65535") {
		t.Fatalf("stderr = %q, want clear invalid port message", stderr.String())
	}
}

func TestRunCodeServerPortFlagBypassesInvalidConfigPort(t *testing.T) {
	ctx := newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s","codeServer":{"port":0}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"code-server", "--port", "9090", "."}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}
	ctx.assertOpened(t, "http", "127.0.0.1:9090", ctx.cwd)
}

func TestRunOpenBehaviorRemainsUnchanged(t *testing.T) {
	ctx := newCodeServerTestContext(t, `{"endpoint":%q,"token":"cli-token","session":"devbox","timeout":"5s"}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run([]string{"open", "https://example.com/path?q=1"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}
	if ctx.request.URL != "https://example.com/path?q=1" {
		t.Fatalf("opened URL = %q, want unchanged open URL", ctx.request.URL)
	}
	ctx.assertNoHandlerError(t)
}

type codeServerTestContext struct {
	cwd        string
	request    protocol.OpenRequestV2
	handlerMu  sync.Mutex
	handlerErr string
}

func newCodeServerTestContext(t *testing.T, configTemplate string) *codeServerTestContext {
	t.Helper()
	ctx := &codeServerTestContext{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/open" {
			ctx.recordHandlerError("path = %q, want /v2/open", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer cli-token" {
			ctx.recordHandlerError("Authorization = %q, want bearer token", got)
			http.Error(w, "unexpected authorization", http.StatusUnauthorized)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&ctx.request); err != nil {
			ctx.recordHandlerError("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(protocol.OpenResponse{OK: true, Status: protocol.StatusOK})
	}))
	t.Cleanup(server.Close)

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("BOB_ENDPOINT", "")
	t.Setenv("BOB_TOKEN", "")
	t.Setenv("BOB_SESSION", "")
	t.Setenv("BOB_TIMEOUT", "")
	t.Setenv("BOB_CODE_SERVER_PORT", "")

	configPath := filepath.Join(configHome, "bob", "bob.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(fmtConfig(configTemplate, server.URL)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx.cwd = t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(ctx.cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	return ctx
}

func (ctx *codeServerTestContext) assertOpened(t *testing.T, scheme, host, folder string) {
	t.Helper()
	ctx.assertNoHandlerError(t)
	opened, err := url.Parse(ctx.request.URL)
	if err != nil {
		t.Fatalf("parse opened URL %q: %v", ctx.request.URL, err)
	}
	if opened.Scheme != scheme || opened.Host != host {
		t.Fatalf("opened URL = %q, want %s://%s", ctx.request.URL, scheme, host)
	}
	if got := opened.Query().Get("folder"); got != folder {
		t.Fatalf("folder = %q, want %q (url %q)", got, folder, ctx.request.URL)
	}
}

func (ctx *codeServerTestContext) recordHandlerError(format string, args ...any) {
	ctx.handlerMu.Lock()
	defer ctx.handlerMu.Unlock()
	ctx.handlerErr = fmt.Sprintf(format, args...)
}

func (ctx *codeServerTestContext) assertNoHandlerError(t *testing.T) {
	t.Helper()
	ctx.handlerMu.Lock()
	defer ctx.handlerMu.Unlock()
	if ctx.handlerErr != "" {
		t.Fatal(ctx.handlerErr)
	}
}

func fmtConfig(format, endpoint string) string {
	return strings.Replace(format, "%q", strconvQuote(endpoint), 1)
}

func strconvQuote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
