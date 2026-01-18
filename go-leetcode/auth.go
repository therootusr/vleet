package leetcode

// Auth contains the minimal authentication material needed to submit and poll.
//
// Treat these fields as secrets: do not log or print them.
type Auth struct {
	// Session is the value of the LEETCODE_SESSION cookie.
	Session string

	// CsrfToken is the value of the csrftoken cookie (optional but recommended).
	CsrfToken string
}

