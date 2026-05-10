package dto

import "time"

// TreeEntry represents one item in a repository tree listing.
type TreeEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	Size int64  `json:"size"`
	SHA  string `json:"sha"`
	Mode string `json:"mode"`
}

// TreeResponse is the response for GET /repos/:owner/:repo/tree/:ref.
type TreeResponse struct {
	SHA     string      `json:"sha"`
	Path    string      `json:"path"`
	Entries []TreeEntry `json:"entries"`
}

// BlobResponse is the response for GET /repos/:owner/:repo/blob/:ref/*path.
type BlobResponse struct {
	Path     string `json:"path"`
	Content  string `json:"content"`  // base64 encoded
	Encoding string `json:"encoding"` // always "base64"
	Size     int64  `json:"size"`
	SHA      string `json:"sha"`
}

// CommitAuthor carries the author or committer identity for a commit.
type CommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// CommitResponse is the representation of a single git commit.
type CommitResponse struct {
	SHA       string       `json:"sha"`
	Message   string       `json:"message"`
	Author    CommitAuthor `json:"author"`
	Committer CommitAuthor `json:"committer"`
	Parents   []string     `json:"parents"`
	HTMLURL   string       `json:"html_url"`
}

// BranchResponse is the representation of a single git branch.
type BranchResponse struct {
	Name      string `json:"name"`
	CommitSHA string `json:"commit_sha"`
	IsDefault bool   `json:"is_default"`
}
