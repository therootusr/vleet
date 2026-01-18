package leetcode

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHttpClient_FetchQuestion_RejectsHTMLResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(kHeaderContentType, "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>captcha</html>"))
	}))
	t.Cleanup(ts.Close)

	lc := NewHttpClient(HttpClientOptions{BaseURL: ts.URL, Http: ts.Client()})
	_, err := lc.FetchQuestion(context.Background(), "two-sum")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unexpected html response") {
		t.Fatalf("error = %q, want mention of unexpected html response", err.Error())
	}
}

func TestHttpClient_Submit_RejectsHTMLResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/problems/two-sum/submit/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set(kHeaderContentType, "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>captcha</html>"))
	}))
	t.Cleanup(ts.Close)

	lc := NewHttpClient(HttpClientOptions{
		BaseURL: ts.URL,
		Http:    ts.Client(),
		Auth: Auth{
			Session:   "sess",
			CsrfToken: "csrf",
		},
	})

	_, err := lc.Submit(context.Background(), SubmitRequest{
		TitleSlug:  "two-sum",
		QuestionID: "1",
		Lang:       "cpp",
		TypedCode:  "CODE\n",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unexpected html response") {
		t.Fatalf("error = %q, want mention of unexpected html response", err.Error())
	}
}

func TestHttpClient_Submit_Non2xx_ReturnsError(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/problems/two-sum/submit/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set(kHeaderContentType, kContentTypeApplicationJSON)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	t.Cleanup(ts.Close)

	lc := NewHttpClient(HttpClientOptions{
		BaseURL: ts.URL,
		Http:    ts.Client(),
		Auth: Auth{
			Session:   "sess",
			CsrfToken: "csrf",
		},
	})

	_, err := lc.Submit(context.Background(), SubmitRequest{
		TitleSlug:  "two-sum",
		QuestionID: "1",
		Lang:       "cpp",
		TypedCode:  "CODE\n",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("error = %q, want status 403", err.Error())
	}
}

func TestHttpClient_PollSubmission_LoopsUntilSuccess(t *testing.T) {
	t.Parallel()

	var n atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/submissions/detail/123/check/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set(kHeaderContentType, kContentTypeApplicationJSON)
		if n.Add(1) == 1 {
			_, _ = w.Write([]byte(`{"state":"STARTED","status_msg":"Pending"}`))
			return
		}
		_, _ = w.Write([]byte(`{"state":"SUCCESS","status_msg":"Accepted","runtime":"1 ms","memory":"2 MB"}`))
	}))
	t.Cleanup(ts.Close)

	lc := NewHttpClient(HttpClientOptions{
		BaseURL: ts.URL,
		Http:    ts.Client(),
		Auth: Auth{
			Session:   "sess",
			CsrfToken: "csrf",
		},
	})

	got, err := lc.PollSubmission(context.Background(), 123, PollOptions{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     5 * time.Millisecond,
		Timeout:         200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("PollSubmission() error = %v", err)
	}
	if got.Status != "Accepted" {
		t.Fatalf("Status = %q, want %q", got.Status, "Accepted")
	}
	if n.Load() < 2 {
		t.Fatalf("expected multiple polls, got %d", n.Load())
	}
}

func TestHttpClient_PollSubmission_TimesOut(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/submissions/detail/123/check/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set(kHeaderContentType, kContentTypeApplicationJSON)
		_, _ = w.Write([]byte(`{"state":"STARTED","status_msg":"Pending"}`))
	}))
	t.Cleanup(ts.Close)

	lc := NewHttpClient(HttpClientOptions{
		BaseURL: ts.URL,
		Http:    ts.Client(),
		Auth: Auth{
			Session:   "sess",
			CsrfToken: "csrf",
		},
	})

	_, err := lc.PollSubmission(context.Background(), 123, PollOptions{
		InitialInterval: 5 * time.Millisecond,
		MaxInterval:     5 * time.Millisecond,
		Timeout:         30 * time.Millisecond,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
}

