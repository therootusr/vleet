package leetcode

import (
	"context"
	"net/http"
	"time"

	"vleet/internal/config"
	"vleet/internal/errx"
)

// Client is the internal API contract described in docs/architecture.md.
type Client interface {
	FetchQuestion(ctx context.Context, titleSlug string) (Question, error)
	Submit(ctx context.Context, req SubmitRequest) (SubmissionID, error)
	PollSubmission(ctx context.Context, submissionID SubmissionID, opts PollOptions) (SubmissionResult, error)
}

type SubmitRequest struct {
	TitleSlug  string
	QuestionID string
	Lang       string // LeetCode language slug (e.g. "cpp", "python3")
	TypedCode  string
}

// SubmissionID is the ID returned by LeetCode's submit endpoint.
// It is used in the check endpoint: /submissions/detail/<id>/check/
type SubmissionID int64

// PollOptions describes polling behavior (interval/backoff/timeout).
// See docs/architecture.md "Concurrency / polling".
type PollOptions struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Timeout         time.Duration
}

// SubmissionResult is the summary vleet prints after polling completes.
type SubmissionResult struct {
	State  string
	Status string

	Runtime string
	Memory  string

	// Error details (compile/runtime), when present.
	CompileError string
	RuntimeError string
}

// HTTPClient is a LeetCode client backed by net/http.
//
// Skeleton only: network behavior (GraphQL queries, submit, poll) is not implemented yet.
// It exists to define the API surface and dependency wiring.
type HTTPClient struct {
	BaseURL   string
	UserAgent string

	HTTP *http.Client
	Auth config.LeetCodeAuth
}

type HTTPClientOptions struct {
	BaseURL   string
	UserAgent string
	HTTP      *http.Client
	Auth      config.LeetCodeAuth
}

func NewHttpClient(opts HTTPClientOptions) *HTTPClient {
	c := &HTTPClient{
		BaseURL:   opts.BaseURL,
		UserAgent: opts.UserAgent,
		HTTP:      opts.HTTP,
		Auth:      opts.Auth,
	}
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
	return c
}

func (c *HTTPClient) FetchQuestion(ctx context.Context, titleSlug string) (Question, error) {
	return Question{}, errx.NotImplemented("leetcode.HTTPClient.FetchQuestion")
}

func (c *HTTPClient) Submit(ctx context.Context, req SubmitRequest) (SubmissionID, error) {
	return 0, errx.NotImplemented("leetcode.HTTPClient.Submit")
}

func (c *HTTPClient) PollSubmission(ctx context.Context, submissionID SubmissionID, opts PollOptions) (SubmissionResult, error) {
	return SubmissionResult{}, errx.NotImplemented("leetcode.HTTPClient.PollSubmission")
}
