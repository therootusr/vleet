package leetcode

import "fmt"

// LangSlug is the LeetCode language language slug (e.g. "cpp", "python3", "golang").
// Consumers should treat this as the primary language identifier.
type LangSlug string

const (
	CPP        LangSlug = "cpp"
	Golang     LangSlug = "golang"
	Python3    LangSlug = "python3"
	JavaScript LangSlug = "javascript"
	TypeScript LangSlug = "typescript"
)

// Info describes a language in terms typical LeetCode tooling needs for workspace generation.
type Info struct {
	LangSlug      LangSlug
	FileExtension string // includes the leading dot (e.g. ".cpp")
}

var infos = map[LangSlug]Info{
	CPP:        {LangSlug: CPP, FileExtension: ".cpp"},
	Golang:     {LangSlug: Golang, FileExtension: ".go"},
	Python3:    {LangSlug: Python3, FileExtension: ".py"},
	JavaScript: {LangSlug: JavaScript, FileExtension: ".js"},
	TypeScript: {LangSlug: TypeScript, FileExtension: ".ts"},
}

// ParseLangSlug returns Info for a LeetCode language slug.
func ParseLangSlug(langSlug string) (Info, error) {
	if langSlug == "" {
		return Info{}, fmt.Errorf("language slug is required")
	}
	info, ok := infos[LangSlug(langSlug)]
	if !ok {
		return Info{}, fmt.Errorf("unsupported language slug: %q", langSlug)
	}
	return info, nil
}

