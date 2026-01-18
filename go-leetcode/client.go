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
)

const (
	kDefaultBaseURL = "https://leetcode.com"
	kGraphQLPath    = "/graphql"

	kHeaderAccept      = "Accept"
	kHeaderContentType = "Content-Type"
	kHeaderOrigin      = "Origin"
	kHeaderReferer     = "Referer"
	kHeaderUserAgent   = "User-Agent"
	kHeaderXCSRFTOKEN  = "x-csrftoken"

	kContentTypeApplicationJSON = "application/json"
	kContentTypeTextHTML        = "text/html"

	kCookieLeetCodeSession = "LEETCODE_SESSION"
	kCookieCSRFTOKEN       = "csrftoken"

	kMaxErrorBodyBytes = 8 << 10

	kProblemPathFormat          = "/problems/%s/"
	kSubmitPathFormat           = "/problems/%s/submit/"
	kSubmissionDetailPathFormat = "/submissions/detail/%d/"
	kSubmissionCheckPathFormat  = "/submissions/detail/%d/check/"
	kDefaultPollInitialInterval = 1 * time.Second
	kDefaultPollMaxInterval     = 5 * time.Second
	kDefaultPollTimeout         = 2 * time.Minute
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

// Client is the public client contract.
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
type PollOptions struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Timeout         time.Duration
}

// SubmissionResult is the summary callers can print after polling completes.
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
type HttpClient struct {
	BaseURL   string
	UserAgent string

	Http *http.Client
	Auth Auth
}

type HttpClientOptions struct {
	BaseURL   string
	UserAgent string
	Http      *http.Client
	Auth      Auth
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
	if c.Auth.CsrfToken != "" {
		req.AddCookie(&http.Cookie{Name: kCookieCSRFTOKEN, Value: c.Auth.CsrfToken})
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
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	titleSlug := strings.TrimSpace(req.TitleSlug)
	if titleSlug == "" {
		return 0, fmt.Errorf("titleSlug is required")
	}
	questionID := strings.TrimSpace(req.QuestionID)
	if questionID == "" {
		return 0, fmt.Errorf("questionID is required")
	}
	lang := strings.TrimSpace(req.Lang)
	if lang == "" {
		return 0, fmt.Errorf("lang is required")
	}
	if strings.TrimSpace(req.TypedCode) == "" {
		return 0, fmt.Errorf("typed_code is required")
	}
	if strings.TrimSpace(c.Auth.Session) == "" {
		return 0, fmt.Errorf("leetcode session cookie is required")
	}

	base := normalizedBaseURL(c.BaseURL)
	endpoint := base + fmt.Sprintf(kSubmitPathFormat, titleSlug)
	referer := base + fmt.Sprintf(kProblemPathFormat, titleSlug)

	payload := map[string]any{
		"lang":        lang,
		"question_id": questionID,
		"typed_code":  req.TypedCode,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("encode submit payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return 0, fmt.Errorf("create submit request: %w", err)
	}
	httpReq.Header.Set(kHeaderContentType, kContentTypeApplicationJSON)
	httpReq.Header.Set(kHeaderAccept, kContentTypeApplicationJSON)
	httpReq.Header.Set(kHeaderReferer, referer)
	httpReq.Header.Set(kHeaderOrigin, base)
	if c.UserAgent != "" {
		httpReq.Header.Set(kHeaderUserAgent, c.UserAgent)
	}
	if c.Auth.CsrfToken != "" {
		httpReq.Header.Set(kHeaderXCSRFTOKEN, c.Auth.CsrfToken)
	}

	// Attach cookies. These are secrets; never log them.
	httpReq.AddCookie(&http.Cookie{Name: kCookieLeetCodeSession, Value: c.Auth.Session})
	if c.Auth.CsrfToken != "" {
		httpReq.AddCookie(&http.Cookie{Name: kCookieCSRFTOKEN, Value: c.Auth.CsrfToken})
	}

	resp, err := c.Http.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("leetcode submit request failed: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get(kHeaderContentType)
	if strings.Contains(strings.ToLower(contentType), kContentTypeTextHTML) {
		return 0, fmt.Errorf(
			"leetcode submit: unexpected html response (status %d); leetcode may be blocking requests",
			resp.StatusCode,
		)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, kMaxErrorBodyBytes))
		msg := strings.TrimSpace(string(snippet))
		if msg == "" {
			msg = resp.Status
		}
		return 0, fmt.Errorf("leetcode submit: status %d: %s", resp.StatusCode, msg)
	}

	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return 0, fmt.Errorf("decode leetcode submit response: %w", err)
	}

	if errMsg := extractString(m, "error"); errMsg != "" {
		return 0, fmt.Errorf("leetcode submit: %s", errMsg)
	}

	sid, ok := extractInt64(m, "submission_id")
	if !ok || sid <= 0 {
		// Include a tiny hint if present.
		if msg := extractString(m, "message"); msg != "" {
			return 0, fmt.Errorf("leetcode submit: %s", msg)
		}
		return 0, fmt.Errorf("leetcode submit: missing submission_id")
	}
	return SubmissionID(sid), nil
}

func (c *HttpClient) PollSubmission(ctx context.Context, submissionID SubmissionID, opts PollOptions) (SubmissionResult, error) {
	if err := ctx.Err(); err != nil {
		return SubmissionResult{}, err
	}
	if submissionID <= 0 {
		return SubmissionResult{}, fmt.Errorf("submissionID is required")
	}
	if strings.TrimSpace(c.Auth.Session) == "" {
		return SubmissionResult{}, fmt.Errorf("leetcode session cookie is required")
	}

	base := normalizedBaseURL(c.BaseURL)
	endpoint := base + fmt.Sprintf(kSubmissionCheckPathFormat, submissionID)
	referer := base + fmt.Sprintf(kSubmissionDetailPathFormat, submissionID)

	initial := opts.InitialInterval
	if initial <= 0 {
		initial = kDefaultPollInitialInterval
	}
	maxInterval := opts.MaxInterval
	if maxInterval <= 0 {
		maxInterval = kDefaultPollMaxInterval
	}
	if maxInterval < initial {
		maxInterval = initial
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = kDefaultPollTimeout
	}

	pollCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		pollCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	interval := initial
	for {
		if err := pollCtx.Err(); err != nil {
			return SubmissionResult{}, err
		}

		httpReq, err := http.NewRequestWithContext(pollCtx, http.MethodGet, endpoint, nil)
		if err != nil {
			return SubmissionResult{}, fmt.Errorf("create submission check request: %w", err)
		}
		httpReq.Header.Set(kHeaderAccept, kContentTypeApplicationJSON)
		httpReq.Header.Set(kHeaderReferer, referer)
		if c.UserAgent != "" {
			httpReq.Header.Set(kHeaderUserAgent, c.UserAgent)
		}
		if c.Auth.CsrfToken != "" {
			httpReq.Header.Set(kHeaderXCSRFTOKEN, c.Auth.CsrfToken)
		}
		httpReq.AddCookie(&http.Cookie{Name: kCookieLeetCodeSession, Value: c.Auth.Session})
		if c.Auth.CsrfToken != "" {
			httpReq.AddCookie(&http.Cookie{Name: kCookieCSRFTOKEN, Value: c.Auth.CsrfToken})
		}

		resp, err := c.Http.Do(httpReq)
		if err != nil {
			return SubmissionResult{}, fmt.Errorf("leetcode submission check request failed: %w", err)
		}
		contentType := resp.Header.Get(kHeaderContentType)
		if strings.Contains(strings.ToLower(contentType), kContentTypeTextHTML) {
			resp.Body.Close()
			return SubmissionResult{}, fmt.Errorf(
				"leetcode submission check: unexpected html response (status %d); leetcode may be blocking requests",
				resp.StatusCode,
			)
		}
		if resp.StatusCode != http.StatusOK {
			snippet, _ := io.ReadAll(io.LimitReader(resp.Body, kMaxErrorBodyBytes))
			resp.Body.Close()
			msg := strings.TrimSpace(string(snippet))
			if msg == "" {
				msg = resp.Status
			}
			return SubmissionResult{}, fmt.Errorf("leetcode submission check: status %d: %s", resp.StatusCode, msg)
		}

		dec := json.NewDecoder(io.LimitReader(resp.Body, 2<<20))
		dec.UseNumber()
		var m map[string]any
		err = dec.Decode(&m)
		resp.Body.Close()
		if err != nil {
			return SubmissionResult{}, fmt.Errorf("decode leetcode submission check response: %w", err)
		}

		state := extractString(m, "state")
		status := extractString(m, "status_msg")
		runtime := extractString(m, "runtime")
		memory := extractString(m, "memory")
		compileErr := extractString(m, "compile_error")
		runtimeErr := extractString(m, "runtime_error")

		// Terminal states are typically SUCCESS (and sometimes FAILURE).
		if state == "SUCCESS" || state == "FAILURE" {
			return SubmissionResult{
				State:        state,
				Status:       status,
				Runtime:      runtime,
				Memory:       memory,
				CompileError: compileErr,
				RuntimeError: runtimeErr,
			}, nil
		}

		// If state is missing, avoid looping forever on an unexpected payload.
		if strings.TrimSpace(state) == "" {
			return SubmissionResult{}, fmt.Errorf("leetcode submission check: missing state")
		}

		// Sleep with backoff, respecting cancellation.
		timer := time.NewTimer(interval)
		select {
		case <-pollCtx.Done():
			timer.Stop()
			return SubmissionResult{}, pollCtx.Err()
		case <-timer.C:
		}

		interval *= 2
		if interval > maxInterval {
			interval = maxInterval
		}
	}
}

func normalizedBaseURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return kDefaultBaseURL
	}
	return base
}

func extractString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func extractInt64(m map[string]any, key string) (int64, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case float64:
		return int64(t), true
	case int64:
		return t, true
	case int:
		return int64(t), true
	case string:
		n, err := json.Number(strings.TrimSpace(t)).Int64()
		return n, err == nil
	default:
		return 0, false
	}
}

