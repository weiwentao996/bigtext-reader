package state

type Bookmark struct {
	Name      string `json:"name"`
	Offset    int64  `json:"offset"`
	Note      string `json:"note,omitempty"`
	CreatedAt int64  `json:"createdAt"`
}

type FileState struct {
	Path      string     `json:"path"`
	Size      int64      `json:"size"`
	ModTime   int64      `json:"modTime"`
	Offset    int64      `json:"offset"`
	Encoding  string     `json:"encoding"`
	Bookmarks []Bookmark `json:"bookmarks"`
	UpdatedAt int64      `json:"updatedAt"`
}

type State struct {
	Files map[string]FileState `json:"files"`
}
