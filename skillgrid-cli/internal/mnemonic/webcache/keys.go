package webcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

var validSources = map[string]struct{}{
	"context7": {},
	"exa":      {},
	"deepwiki": {},
	"fetch":    {},
	"manual":   {},
}

// KeyInput holds fields used to derive a normalized cache_key per source.
type KeyInput struct {
	URL         string
	Query       string
	SortParams  string
	LibraryID   string
	VersionTag  string
	RepoName    string
	Question    string
	Title       string
	ContentHash string
}

// CacheKey returns a sha256 hex digest for dedup keyed by source-specific rules.
func CacheKey(source string, in KeyInput) (string, error) {
	if _, ok := validSources[source]; !ok {
		return "", fmt.Errorf("unsupported web cache source %q", source)
	}

	var material string
	switch source {
	case "fetch":
		if strings.TrimSpace(in.URL) == "" {
			return "", fmt.Errorf("url is required for fetch cache_key")
		}
		material = normalizeURL(in.URL)
	case "exa":
		if strings.TrimSpace(in.Query) == "" {
			return "", fmt.Errorf("query is required for exa cache_key")
		}
		material = in.Query + "|" + in.SortParams
	case "context7":
		if strings.TrimSpace(in.LibraryID) == "" {
			return "", fmt.Errorf("library_id is required for context7 cache_key")
		}
		if strings.TrimSpace(in.Query) == "" {
			return "", fmt.Errorf("query is required for context7 cache_key")
		}
		material = in.LibraryID + "|" + in.VersionTag + "|" + in.Query
	case "deepwiki":
		if strings.TrimSpace(in.RepoName) == "" {
			return "", fmt.Errorf("repo_name is required for deepwiki cache_key")
		}
		question := in.Question
		if question == "" {
			question = in.Query
		}
		if strings.TrimSpace(question) == "" {
			return "", fmt.Errorf("question is required for deepwiki cache_key")
		}
		material = in.RepoName + "|" + question
	case "manual":
		if strings.TrimSpace(in.Title) == "" {
			return "", fmt.Errorf("title is required for manual cache_key")
		}
		if strings.TrimSpace(in.ContentHash) == "" {
			return "", fmt.Errorf("content_hash is required for manual cache_key")
		}
		material = in.Title + "|" + in.ContentHash
	}

	return sha256Hex(material), nil
}

func keyInputFromSave(in SaveWebInput, contentHash string) KeyInput {
	return KeyInput{
		URL:         in.URL,
		Query:       in.Query,
		SortParams:  in.SortParams,
		LibraryID:   in.LibraryID,
		VersionTag:  in.VersionTag,
		RepoName:    in.RepoName,
		Question:    in.Question,
		Title:       in.Title,
		ContentHash: contentHash,
	}
}

func keyInputFromLookup(in LookupInput) KeyInput {
	return KeyInput{
		URL:         in.URL,
		Query:       in.Query,
		SortParams:  in.SortParams,
		LibraryID:   in.LibraryID,
		VersionTag:  in.VersionTag,
		RepoName:    in.RepoName,
		Question:    in.Question,
		Title:       in.Title,
		ContentHash: in.ContentHash,
	}
}

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(raw)
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	if len(u.Path) > 1 {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	return u.String()
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func contentHash(content string) string {
	return sha256Hex(content)
}
