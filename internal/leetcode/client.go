package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"vleet/internal/config"
	"vleet/internal/errx"
)

const (
	kDefaultBaseURL = "https://leetcode.com"
	kGraphQLPath    = "/graphql"

	kHeaderAccept      = "Accept"
	kHeaderContentType = "Content-Type"
	kHeaderUserAgent   = "User-Agent"

	kContentTypeApplicationJSON = "application/json"
	kContentTypeTextHTML        = "text/html"

	kCookieLeetCodeSession = "LEETCODE_SESSION"
	kCookieCSRFTOKEN       = "csrftoken"

	kMaxErrorBodyBytes = 8 << 10
)

const kQuestionDataQuery = `
query questionData($titleSlug: String!) {
  question(titleSlug: $titleSlug) {
    questionId
    questionFrontendId
    title
    titleSlug
    difficulty
    content
    exampleTestcases
    sampleTestCase
    hints
    topicTags { name slug }
    codeSnippets { lang langSlug code }
  }
}
`

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
	if err := ctx.Err(); err != nil {
		return Question{}, err
	}
	if strings.TrimSpace(titleSlug) == "" {
		return Question{}, fmt.Errorf("titleSlug is required")
	}

	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		base = kDefaultBaseURL
	}
	endpoint := base + kGraphQLPath

	type vars struct {
		TitleSlug string `json:"titleSlug"`
	}
	reqBody := struct {
		Query     string `json:"query"`
		Variables vars   `json:"variables"`
	}{
		Query:     kQuestionDataQuery,
		Variables: vars{TitleSlug: titleSlug},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return Question{}, fmt.Errorf("encode graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return Question{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set(kHeaderContentType, kContentTypeApplicationJSON)
	req.Header.Set(kHeaderAccept, kContentTypeApplicationJSON)
	if c.UserAgent != "" {
		req.Header.Set(kHeaderUserAgent, c.UserAgent)
	}
	// Attach cookies if configured. These are secrets; never log them.
	if c.Auth.Session != "" {
		req.AddCookie(&http.Cookie{Name: kCookieLeetCodeSession, Value: c.Auth.Session})
	}
	if c.Auth.CSRFTOKEN != "" {
		req.AddCookie(&http.Cookie{Name: kCookieCSRFTOKEN, Value: c.Auth.CSRFTOKEN})
	}

	resp, err := c.Http.Do(req)
	if err != nil {
		return Question{}, fmt.Errorf("leetcode graphql request failed: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get(kHeaderContentType)
	if strings.Contains(strings.ToLower(contentType), kContentTypeTextHTML) {
		// LeetCode may be blocking automated requests; the response is often HTML.
		return Question{}, fmt.Errorf(
			"leetcode graphql: unexpected html response (status %d); leetcode may be blocking requests",
			resp.StatusCode,
		)
	}

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, kMaxErrorBodyBytes))
		msg := strings.TrimSpace(string(snippet))
		if msg == "" {
			msg = resp.Status
		}
		return Question{}, fmt.Errorf("leetcode graphql: status %d: %s", resp.StatusCode, msg)
	}

	type gqlErr struct {
		Message string `json:"message"`
	}
	type gqlQuestion struct {
		QuestionID         string        `json:"questionId"`
		QuestionFrontendID string        `json:"questionFrontendId"`
		Title              string        `json:"title"`
		TitleSlug          string        `json:"titleSlug"`
		Difficulty         string        `json:"difficulty"`
		Content            string        `json:"content"`
		ExampleTestcases   string        `json:"exampleTestcases"`
		SampleTestCase     string        `json:"sampleTestCase"`
		Hints              []string      `json:"hints"`
		TopicTags          []TopicTag    `json:"topicTags"`
		CodeSnippets       []CodeSnippet `json:"codeSnippets"`
	}

	var gqlResp struct {
		Data struct {
			Question *gqlQuestion `json:"question"`
		} `json:"data"`
		Errors []gqlErr `json:"errors"`
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&gqlResp); err != nil {
		// If LeetCode returns HTML while claiming JSON, include a tiny snippet for diagnosis.
		return Question{}, fmt.Errorf("decode leetcode graphql response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		msgs := make([]string, 0, len(gqlResp.Errors))
		for _, e := range gqlResp.Errors {
			if strings.TrimSpace(e.Message) != "" {
				msgs = append(msgs, e.Message)
			}
		}
		if len(msgs) == 0 {
			return Question{}, fmt.Errorf("leetcode graphql: unknown graphql error")
		}
		return Question{}, fmt.Errorf("leetcode graphql: %s", strings.Join(msgs, "; "))
	}
	if gqlResp.Data.Question == nil {
		return Question{}, fmt.Errorf("problem not found: %s", titleSlug)
	}

	q := Question{
		QuestionID:       gqlResp.Data.Question.QuestionID,
		FrontendID:       gqlResp.Data.Question.QuestionFrontendID,
		Title:            gqlResp.Data.Question.Title,
		TitleSlug:        gqlResp.Data.Question.TitleSlug,
		Difficulty:       gqlResp.Data.Question.Difficulty,
		ContentHTML:      gqlResp.Data.Question.Content,
		ExampleTestcases: gqlResp.Data.Question.ExampleTestcases,
		SampleTestCase:   gqlResp.Data.Question.SampleTestCase,
		Hints:            gqlResp.Data.Question.Hints,
		TopicTags:        gqlResp.Data.Question.TopicTags,
		CodeSnippets:     gqlResp.Data.Question.CodeSnippets,
	}
	return q, nil
}

func (c *HttpClient) Submit(ctx context.Context, req SubmitRequest) (SubmissionID, error) {
	return 0, errx.NotImplemented("leetcode.HttpClient.Submit")
}

func (c *HttpClient) PollSubmission(ctx context.Context, submissionID SubmissionID, opts PollOptions) (SubmissionResult, error) {
	return SubmissionResult{}, errx.NotImplemented("leetcode.HttpClient.PollSubmission")
}
