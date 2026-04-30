package reader

import "time"

const (
	EncodingAuto = "auto"
	EncodingUTF8 = "utf8"
	EncodingGBK  = "gbk"
)

type Config struct {
	Encoding      string
	PageSize      int
	ChunkSize     int
	MaxLineBytes  int
	SearchChunk   int
	LineStartScan int
}

type FileMeta struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"modTime"`
	Encoding string    `json:"encoding"`
}

type Page struct {
	StartOffset      int64    `json:"startOffset"`
	EndOffset        int64    `json:"endOffset"`
	Lines            []string `json:"lines"`
	LineStartOffsets []int64  `json:"lineStartOffsets"`
	LineEndOffsets   []int64  `json:"lineEndOffsets"`
	EOF              bool     `json:"eof"`
	BOF              bool     `json:"bof"`
	HasPrevious      bool     `json:"hasPrevious"`
	Truncated        bool     `json:"truncated"`
	Encoding         string   `json:"encoding"`
	FileSize         int64    `json:"fileSize"`
}

type SearchHitLocation struct {
	LineIndex     int `json:"lineIndex"`
	LineCharStart int `json:"lineCharStart"`
	LineCharEnd   int `json:"lineCharEnd"`
}

type SearchHitRef struct {
	Index      int   `json:"index"`
	Offset     int64 `json:"offset"`
	ByteLength int   `json:"byteLength"`
	LineStart  int64 `json:"lineStart"`
	LineEnd    int64 `json:"lineEnd"`
	LineNumber int64 `json:"lineNumber"`
}

type SearchHit struct {
	Index         int    `json:"index"`
	Offset        int64  `json:"offset"`
	ByteLength    int    `json:"byteLength"`
	LineStart     int64  `json:"lineStart"`
	LineEnd       int64  `json:"lineEnd"`
	LineNumber    int64  `json:"lineNumber"`
	LinePreview   string `json:"linePreview"`
	LineCharStart int    `json:"lineCharStart"`
	LineCharEnd   int    `json:"lineCharEnd"`
}

type SearchSessionSummary struct {
	SearchID string `json:"searchId"`
	Keyword  string `json:"keyword"`
	Total    int    `json:"total"`
	FileSize int64  `json:"fileSize"`
	Encoding string `json:"encoding"`
}

type SearchHitPreviewPage struct {
	SearchID string      `json:"searchId"`
	Keyword  string      `json:"keyword"`
	Offset   int         `json:"offset"`
	Limit    int         `json:"limit"`
	Total    int         `json:"total"`
	Hits     []SearchHit `json:"hits"`
}

type SearchSummary struct {
	Keyword   string      `json:"keyword"`
	Total     int         `json:"total"`
	Hits      []SearchHit `json:"hits"`
	Truncated bool        `json:"truncated"`
	Limit     int         `json:"limit"`
	FileSize  int64       `json:"fileSize"`
	Encoding  string      `json:"encoding"`
}

type PageWindow struct {
	Pages    []Page `json:"pages"`
	Anchor   Page   `json:"anchor"`
	FileSize int64  `json:"fileSize"`
	Encoding string `json:"encoding"`
}

type SearchResult struct {
	Offset     int64 `json:"offset"`
	ByteLength int   `json:"byteLength"`
}

func NormalizeConfig(config Config) Config {
	if config.Encoding == "" {
		config.Encoding = EncodingAuto
	}
	if config.PageSize <= 0 {
		config.PageSize = 60
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = 256 * 1024
	}
	if config.MaxLineBytes <= 0 {
		config.MaxLineBytes = 256 * 1024
	}
	if config.SearchChunk <= 0 {
		config.SearchChunk = 1024 * 1024
	}
	if config.LineStartScan <= 0 {
		config.LineStartScan = 1024 * 1024
	}
	return config
}
