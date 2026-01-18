# go-leetcode

`go-leetcode` is a small Go library that implements the practical LeetCode
endpoints used by tooling:

- GraphQL `questionData` fetch
- REST submit
- REST polling of submission status

## Security note

LeetCode authentication relies on session cookies (and an optional CSRF token).
Treat these values as secrets: do not log them or print them.

