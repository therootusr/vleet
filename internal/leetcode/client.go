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

// HttpClient is a LeetCode client backed by net/http.
//
// Skeleton only: network behavior (GraphQL queries, submit, poll) is not implemented yet.
// It exists to define the API surface and dependency wiring.
type HttpClient struct {
	BaseURL   string
	UserAgent string

	Http *http.Client
	Auth config.LeetCodeAuth
}

type HttpClientOptions struct {
	BaseURL   string
	UserAgent string
	Http      *http.Client
	Auth      config.LeetCodeAuth
}

func NewHttpClient(opts HttpClientOptions) *HttpClient {
	c := &HttpClient{
		BaseURL:   opts.BaseURL,
		UserAgent: opts.UserAgent,
		Http:      opts.Http,
		Auth:      opts.Auth,
	}
	if c.Http == nil {
		c.Http = http.DefaultClient
	}
	return c
}

func (c *HttpClient) FetchQuestion(ctx context.Context, titleSlug string) (Question, error) {
	return Question{}, errx.NotImplemented("leetcode.HttpClient.FetchQuestion")
}

func (c *HttpClient) Submit(ctx context.Context, req SubmitRequest) (SubmissionID, error) {
	return 0, errx.NotImplemented("leetcode.HttpClient.Submit")
}

func (c *HttpClient) PollSubmission(ctx context.Context, submissionID SubmissionID, opts PollOptions) (SubmissionResult, error) {
	return SubmissionResult{}, errx.NotImplemented("leetcode.HttpClient.PollSubmission")
}
