package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bigtext-reader/internal/reader"
	"bigtext-reader/internal/state"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	reader        *reader.Reader
	store         *state.Store
	currentKey    string
	currentFile   state.FileState
	searchID      string
	searchKeyword string
	searchHits    []reader.SearchHitRef
	language      string
}

type OpenResult struct {
	Meta      reader.FileMeta  `json:"meta"`
	Page      reader.Page      `json:"page"`
	Bookmarks []state.Bookmark `json:"bookmarks"`
	Resumed   bool             `json:"resumed"`
}

type SearchPageResult struct {
	Page            reader.Page `json:"page"`
	HitOffset       int64       `json:"hitOffset"`
	HitByteLength   int         `json:"hitByteLength"`
	LineStartOffset int64       `json:"lineStartOffset"`
	LineIndex       int         `json:"lineIndex"`
	LineCharStart   int         `json:"lineCharStart"`
	LineCharEnd     int         `json:"lineCharEnd"`
	Keyword         string      `json:"keyword"`
	Wrapped         bool        `json:"wrapped"`
}

type FolderFile struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
}

func NewApp() *App {
	store, _ := state.NewStore("bigtext-reader", "bf-reader")
	return &App{store: store, language: "zh"}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) SetLanguage(lang string) error {
	a.language = normalizeLanguage(lang)
	return nil
}

func (a *App) SelectFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: a.msg("dialog.openFile"),
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt;*.log)", Pattern: "*.txt;*.log"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

func (a *App) SelectFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: a.msg("dialog.openFolder")})
}

func (a *App) ListFolderFiles(folder string) ([]FolderFile, error) {
	if strings.TrimSpace(folder) == "" {
		return []FolderFile{}, nil
	}
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	files := make([]FolderFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(folder, entry.Name())
		files = append(files, FolderFile{Path: path, Name: entry.Name(), Size: info.Size(), ModTime: info.ModTime().Unix()})
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	return files, nil
}

func (a *App) OpenFile(path string, encoding string, pageSize int) (OpenResult, error) {
	if path == "" {
		return OpenResult{}, errors.New(a.msg("error.selectFile"))
	}
	if a.reader != nil {
		_ = a.reader.Close()
		a.reader = nil
	}

	r, err := reader.Open(path, reader.Config{Encoding: encoding, PageSize: pageSize})
	if err != nil {
		return OpenResult{}, err
	}
	a.reader = r
	a.clearSearchSession()
	meta := r.Meta()
	a.currentKey = state.FileKey(meta.Path, meta.Size, meta.ModTime.Unix())

	fileState, ok, err := a.store.Get(a.currentKey)
	if err != nil {
		return OpenResult{}, err
	}
	resumed := ok && fileState.Offset > 0
	if !ok {
		fileState = state.FileState{
			Path:      meta.Path,
			Size:      meta.Size,
			ModTime:   meta.ModTime.Unix(),
			Offset:    0,
			Encoding:  meta.Encoding,
			Bookmarks: []state.Bookmark{},
		}
	}
	if fileState.Encoding == "" || encoding != reader.EncodingAuto && encoding != "" {
		fileState.Encoding = meta.Encoding
	}
	a.currentFile = fileState

	offset := fileState.Offset
	if offset < 0 || offset > meta.Size {
		offset = 0
	}
	page, err := r.ReadPage(offset)
	if err != nil {
		return OpenResult{}, err
	}
	a.currentFile.Offset = page.StartOffset
	a.currentFile.Encoding = meta.Encoding
	_ = a.persist()

	return OpenResult{Meta: meta, Page: page, Bookmarks: a.currentFile.Bookmarks, Resumed: resumed}, nil
}

func (a *App) ReadPage(offset int64) (reader.Page, error) {
	if a.reader == nil {
		return reader.Page{}, errors.New(a.msg("error.noFileOpen"))
	}
	page, err := a.reader.ReadPage(offset)
	if err != nil {
		return page, err
	}
	a.currentFile.Offset = page.StartOffset
	a.currentFile.Encoding = page.Encoding
	return page, a.persist()
}

func (a *App) NextPage(offset int64) (reader.Page, error) {
	return a.ReadPage(offset)
}

func (a *App) ReadNextPage(afterOffset int64) (reader.Page, error) {
	return a.readPageNoPersist(afterOffset)
}

func (a *App) ReadPreviousPage(beforeOffset int64) (reader.Page, error) {
	if a.reader == nil {
		return reader.Page{}, errors.New(a.msg("error.noFileOpen"))
	}
	return a.reader.ReadPreviousPage(beforeOffset)
}

func (a *App) ReadAroundOffset(offset int64, beforePages int, afterPages int) (reader.PageWindow, error) {
	if a.reader == nil {
		return reader.PageWindow{}, errors.New(a.msg("error.noFileOpen"))
	}
	return a.reader.ReadWindowAround(offset, beforePages, afterPages)
}

func (a *App) readPageNoPersist(offset int64) (reader.Page, error) {
	if a.reader == nil {
		return reader.Page{}, errors.New(a.msg("error.noFileOpen"))
	}
	return a.reader.ReadPage(offset)
}

func (a *App) JumpToPercent(percent float64) (reader.Page, error) {
	if a.reader == nil {
		return reader.Page{}, errors.New(a.msg("error.noFileOpen"))
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	meta := a.reader.Meta()
	offset := int64(float64(meta.Size) * percent / 100)
	normalized, err := a.reader.NormalizeOffsetToNextLine(offset)
	if err != nil {
		return reader.Page{}, err
	}
	return a.ReadPage(normalized)
}

func (a *App) JumpToOffset(offset int64) (reader.Page, error) {
	if a.reader == nil {
		return reader.Page{}, errors.New(a.msg("error.noFileOpen"))
	}
	normalized, err := a.reader.NormalizeOffsetToNextLine(offset)
	if err != nil {
		return reader.Page{}, err
	}
	return a.ReadPage(normalized)
}

func (a *App) SearchForward(startOffset int64, keyword string) (SearchPageResult, error) {
	if a.reader == nil {
		return SearchPageResult{}, errors.New(a.msg("error.noFileOpen"))
	}
	result, err := a.reader.SearchForward(startOffset, keyword)
	wrapped := false
	if errors.Is(err, reader.ErrNotFound) && startOffset > 0 {
		result, err = a.reader.SearchForward(0, keyword)
		wrapped = true
	}
	if err != nil {
		if errors.Is(err, reader.ErrNotFound) {
			return SearchPageResult{}, fmt.Errorf(a.msg("error.notFound"), keyword)
		}
		return SearchPageResult{}, err
	}
	return a.searchHitPage(result.Offset, result.ByteLength, keyword, wrapped)
}

func (a *App) SearchStats(keyword string, limit int) (reader.SearchSummary, error) {
	if a.reader == nil {
		return reader.SearchSummary{}, errors.New(a.msg("error.noFileOpen"))
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return reader.SearchSummary{}, errors.New(a.msg("error.enterKeyword"))
	}
	if limit <= 0 {
		limit = 1000
	}
	if limit > 5000 {
		limit = 5000
	}
	return a.reader.SearchAll(keyword, limit)
}

func (a *App) StartSearch(keyword string) (reader.SearchSessionSummary, error) {
	if a.reader == nil {
		return reader.SearchSessionSummary{}, errors.New(a.msg("error.noFileOpen"))
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return reader.SearchSessionSummary{}, errors.New(a.msg("error.enterKeyword"))
	}
	hits, err := a.reader.BuildSearchIndex(keyword)
	if err != nil {
		return reader.SearchSessionSummary{}, err
	}
	meta := a.reader.Meta()
	a.searchID = fmt.Sprintf("%s:%s:%d", a.currentKey, keyword, time.Now().UnixNano())
	a.searchKeyword = keyword
	a.searchHits = hits
	return reader.SearchSessionSummary{SearchID: a.searchID, Keyword: keyword, Total: len(hits), FileSize: meta.Size, Encoding: meta.Encoding}, nil
}

func (a *App) SearchHitPreviews(searchID string, offset int, limit int) (reader.SearchHitPreviewPage, error) {
	if a.reader == nil {
		return reader.SearchHitPreviewPage{}, errors.New(a.msg("error.noFileOpen"))
	}
	if err := a.validateSearchSession(searchID); err != nil {
		return reader.SearchHitPreviewPage{}, err
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	hits, err := a.reader.BuildSearchHitPreviews(a.searchHits, offset, limit)
	if err != nil {
		return reader.SearchHitPreviewPage{}, err
	}
	return reader.SearchHitPreviewPage{SearchID: a.searchID, Keyword: a.searchKeyword, Offset: offset, Limit: limit, Total: len(a.searchHits), Hits: hits}, nil
}

func (a *App) SearchHitPageByIndex(searchID string, index int) (SearchPageResult, error) {
	if a.reader == nil {
		return SearchPageResult{}, errors.New(a.msg("error.noFileOpen"))
	}
	if err := a.validateSearchSession(searchID); err != nil {
		return SearchPageResult{}, err
	}
	if index < 0 || index >= len(a.searchHits) {
		return SearchPageResult{}, errors.New(a.msg("error.searchResultMissing"))
	}
	hit := a.searchHits[index]
	return a.searchHitPageAtLine(hit.Offset, hit.ByteLength, hit.LineStart, a.searchKeyword, false)
}

func (a *App) SearchHitPage(hitOffset int64, hitByteLength int) (SearchPageResult, error) {
	if a.reader == nil {
		return SearchPageResult{}, errors.New(a.msg("error.noFileOpen"))
	}
	return a.searchHitPage(hitOffset, hitByteLength, "", false)
}

func (a *App) searchHitPage(hitOffset int64, hitByteLength int, keyword string, wrapped bool) (SearchPageResult, error) {
	lineStart, err := a.reader.FindLineStartNear(hitOffset)
	if err != nil {
		return SearchPageResult{}, err
	}
	return a.searchHitPageAtLine(hitOffset, hitByteLength, lineStart, keyword, wrapped)
}

func (a *App) searchHitPageAtLine(hitOffset int64, hitByteLength int, lineStart int64, keyword string, wrapped bool) (SearchPageResult, error) {
	pageStart := lineStart
	if hitOffset-lineStart > int64(a.reader.Config().MaxLineBytes) {
		pageStart = hitOffset - int64(a.reader.Config().MaxLineBytes/3)
	}
	page, err := a.ReadPage(pageStart)
	if err != nil {
		return SearchPageResult{}, err
	}
	location, err := a.reader.LocateHit(page, hitOffset, hitByteLength)
	if err != nil {
		return SearchPageResult{}, err
	}
	return SearchPageResult{
		Page:            page,
		HitOffset:       hitOffset,
		HitByteLength:   hitByteLength,
		LineStartOffset: lineStart,
		LineIndex:       location.LineIndex,
		LineCharStart:   location.LineCharStart,
		LineCharEnd:     location.LineCharEnd,
		Keyword:         keyword,
		Wrapped:         wrapped,
	}, nil
}

func (a *App) AddBookmark(name string, offset int64) error {
	if a.reader == nil {
		return errors.New(a.msg("error.noFileOpen"))
	}
	if name == "" {
		name = fmt.Sprintf("%s @ %d", time.Now().Format("2006-01-02 15:04"), offset)
	}
	a.currentFile.Bookmarks = append(a.currentFile.Bookmarks, state.Bookmark{
		Name:      name,
		Offset:    offset,
		CreatedAt: time.Now().Unix(),
	})
	return a.persist()
}

func (a *App) ListBookmarks() ([]state.Bookmark, error) {
	if a.reader == nil {
		return nil, errors.New(a.msg("error.noFileOpen"))
	}
	return a.currentFile.Bookmarks, nil
}

func (a *App) DeleteBookmark(index int) error {
	if a.reader == nil {
		return errors.New(a.msg("error.noFileOpen"))
	}
	if index < 0 || index >= len(a.currentFile.Bookmarks) {
		return errors.New(a.msg("error.bookmarkMissing"))
	}
	a.currentFile.Bookmarks = append(a.currentFile.Bookmarks[:index], a.currentFile.Bookmarks[index+1:]...)
	return a.persist()
}

func (a *App) GoToBookmark(index int) (reader.Page, error) {
	if a.reader == nil {
		return reader.Page{}, errors.New(a.msg("error.noFileOpen"))
	}
	if index < 0 || index >= len(a.currentFile.Bookmarks) {
		return reader.Page{}, errors.New(a.msg("error.bookmarkMissing"))
	}
	return a.ReadPage(a.currentFile.Bookmarks[index].Offset)
}

func (a *App) SaveProgress(offset int64) error {
	if a.reader == nil {
		return errors.New(a.msg("error.noFileOpen"))
	}
	a.currentFile.Offset = offset
	return a.persist()
}

func (a *App) clearSearchSession() {
	a.searchID = ""
	a.searchKeyword = ""
	a.searchHits = nil
}

func (a *App) validateSearchSession(searchID string) error {
	if a.searchID == "" || searchID == "" || searchID != a.searchID {
		return errors.New(a.msg("error.searchExpired"))
	}
	return nil
}

func (a *App) persist() error {
	if a.store == nil || a.currentKey == "" {
		return nil
	}
	meta := a.reader.Meta()
	a.currentFile.Path = meta.Path
	a.currentFile.Size = meta.Size
	a.currentFile.ModTime = meta.ModTime.Unix()
	if a.currentFile.Bookmarks == nil {
		a.currentFile.Bookmarks = []state.Bookmark{}
	}
	return a.store.Upsert(a.currentKey, a.currentFile)
}

func normalizeLanguage(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "en", "en-us", "en_us":
		return "en"
	default:
		return "zh"
	}
}

var localizedMessages = map[string]map[string]string{
	"zh": {
		"dialog.openFile":           "选择要阅读的文本文件",
		"dialog.openFolder":         "选择文件夹",
		"error.selectFile":          "请选择文件",
		"error.noFileOpen":          "尚未打开文件",
		"error.enterKeyword":        "请输入搜索关键字",
		"error.notFound":            "没有找到：%s",
		"error.searchResultMissing": "搜索结果不存在",
		"error.bookmarkMissing":     "书签不存在",
		"error.searchExpired":       "搜索结果已过期，请重新搜索",
	},
	"en": {
		"dialog.openFile":           "Select a text file to read",
		"dialog.openFolder":         "Select folder",
		"error.selectFile":          "Please select a file",
		"error.noFileOpen":          "No file is open",
		"error.enterKeyword":        "Enter a search keyword",
		"error.notFound":            "Not found: %s",
		"error.searchResultMissing": "Search result does not exist",
		"error.bookmarkMissing":     "Bookmark does not exist",
		"error.searchExpired":       "Search results expired. Search again",
	},
}

func (a *App) msg(key string) string {
	lang := normalizeLanguage(a.language)
	if messages, ok := localizedMessages[lang]; ok {
		if value, ok := messages[key]; ok {
			return value
		}
	}
	if value, ok := localizedMessages["zh"][key]; ok {
		return value
	}
	return key
}

func cleanPath(path string) string {
	if path == "" {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}
