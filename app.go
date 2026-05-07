package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"bigtext-reader/internal/reader"
	"bigtext-reader/internal/state"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	reader      *reader.Reader
	store       *state.Store
	currentKey  string
	currentFile state.FileState
	searchMu    sync.Mutex
	search      *searchSession
	language    string
}

type searchSession struct {
	mu            sync.RWMutex
	id            string
	keyword       string
	options       reader.SearchOptions
	source        *reader.Reader
	store         *searchHitStore
	scannedOffset int64
	fileSize      int64
	encoding      string
	done          bool
	canceled      bool
	errText       string
	cancel        context.CancelFunc
}

type searchSessionSnapshot struct {
	id            string
	keyword       string
	options       reader.SearchOptions
	total         int
	scannedOffset int64
	fileSize      int64
	encoding      string
	done          bool
	canceled      bool
	errText       string
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

func (a *App) domReady(ctx context.Context) {
	a.openFirstPath(os.Args[1:])
}

func (a *App) openFirstPath(paths []string) {
	if a.ctx == nil {
		return
	}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
		runtime.EventsEmit(a.ctx, "app:open-file", cleanPath(path))
		return
	}
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
	a.clearSearchSession()
	if a.reader != nil {
		_ = a.reader.Close()
		a.reader = nil
	}

	r, err := reader.Open(path, reader.Config{Encoding: encoding, PageSize: pageSize})
	if err != nil {
		return OpenResult{}, err
	}
	a.reader = r
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
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return SearchPageResult{}, errors.New(a.msg("error.enterKeyword"))
	}
	options := reader.SearchOptions{CaseSensitive: true}
	result, err := a.reader.SearchForwardWithOptions(startOffset, keyword, options)
	wrapped := false
	if errors.Is(err, reader.ErrNotFound) && startOffset > 0 {
		result, err = a.reader.SearchForwardWithOptions(0, keyword, options)
		wrapped = true
	}
	if err != nil {
		if errors.Is(err, reader.ErrNotFound) {
			return SearchPageResult{}, fmt.Errorf(a.msg("error.notFound"), keyword)
		}
		return SearchPageResult{}, a.searchError(err)
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
	return a.reader.SearchAllWithOptions(keyword, limit, reader.SearchOptions{CaseSensitive: true})
}

func (a *App) StartSearch(keyword string, regex bool, caseSensitive bool) (reader.SearchSessionSummary, error) {
	if a.reader == nil {
		return reader.SearchSessionSummary{}, errors.New(a.msg("error.noFileOpen"))
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return reader.SearchSessionSummary{}, errors.New(a.msg("error.enterKeyword"))
	}
	options := reader.SearchOptions{Regex: regex, CaseSensitive: caseSensitive}
	if err := a.reader.ValidateSearchOptions(keyword, options); err != nil {
		return reader.SearchSessionSummary{}, a.searchError(err)
	}
	meta := a.reader.Meta()
	searchID := fmt.Sprintf("%s:%s:%t:%t:%d", a.currentKey, keyword, regex, caseSensitive, time.Now().UnixNano())
	store, err := newSearchHitStore()
	if err != nil {
		return reader.SearchSessionSummary{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	session := &searchSession{id: searchID, keyword: keyword, options: options, source: a.reader, store: store, fileSize: meta.Size, encoding: meta.Encoding, cancel: cancel}
	a.setActiveSearchSession(session)
	go a.runSearchSession(ctx, session)
	return session.summary(), nil
}

func (a *App) SearchSessionStatus(searchID string) (reader.SearchSessionSummary, error) {
	session, err := a.activeSearchSession(searchID)
	if err != nil {
		return reader.SearchSessionSummary{}, err
	}
	return session.summary(), nil
}

func (a *App) StopSearch(searchID string) (reader.SearchSessionSummary, error) {
	session, err := a.activeSearchSession(searchID)
	if err != nil {
		return reader.SearchSessionSummary{}, err
	}
	session.mu.Lock()
	if !session.done {
		session.canceled = true
		session.done = true
	}
	session.mu.Unlock()
	if session.cancel != nil {
		session.cancel()
	}
	summary := session.summary()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "search:progress", summary)
	}
	return summary, nil
}

func (a *App) SearchHitPreviews(searchID string, offset int, limit int) (reader.SearchHitPreviewPage, error) {
	if a.reader == nil {
		return reader.SearchHitPreviewPage{}, errors.New(a.msg("error.noFileOpen"))
	}
	session, err := a.activeSearchSession(searchID)
	if err != nil {
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
	snapshot := session.snapshot()
	refs, err := session.store.Window(offset, limit)
	if err != nil {
		return reader.SearchHitPreviewPage{}, err
	}
	hits, err := a.reader.BuildSearchHitPreviewsFromRefs(refs)
	if err != nil {
		return reader.SearchHitPreviewPage{}, err
	}
	return reader.SearchHitPreviewPage{SearchID: snapshot.id, Keyword: snapshot.keyword, Offset: offset, Limit: limit, Total: snapshot.total, ScannedOffset: snapshot.scannedOffset, FileSize: snapshot.fileSize, Done: snapshot.done, Canceled: snapshot.canceled, Error: snapshot.errText, Regex: snapshot.options.Regex, CaseSensitive: snapshot.options.CaseSensitive, Hits: hits}, nil
}

func (a *App) SearchHitPageByIndex(searchID string, index int) (SearchPageResult, error) {
	if a.reader == nil {
		return SearchPageResult{}, errors.New(a.msg("error.noFileOpen"))
	}
	session, err := a.activeSearchSession(searchID)
	if err != nil {
		return SearchPageResult{}, err
	}
	snapshot := session.snapshot()
	if index < 0 || index >= snapshot.total {
		return SearchPageResult{}, errors.New(a.msg("error.searchResultMissing"))
	}
	hit, err := session.store.Get(index)
	if err != nil {
		return SearchPageResult{}, err
	}
	return a.searchHitPageAtLine(hit.Offset, hit.ByteLength, hit.LineStart, snapshot.keyword, false)
}

func (a *App) SearchHitPage(hitOffset int64, hitByteLength int) (SearchPageResult, error) {
	if a.reader == nil {
		return SearchPageResult{}, errors.New(a.msg("error.noFileOpen"))
	}
	return a.searchHitPage(hitOffset, hitByteLength, "", false)
}

func (a *App) ExportSearchResults(searchID string) (string, error) {
	if a.reader == nil {
		return "", errors.New(a.msg("error.noFileOpen"))
	}
	session, err := a.activeSearchSession(searchID)
	if err != nil {
		return "", err
	}
	snapshot := session.snapshot()
	if snapshot.total == 0 {
		return "", errors.New(a.msg("error.noSearchResults"))
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           a.msg("dialog.exportSearchResults"),
		DefaultFilename: "search-results.tsv",
		Filters: []runtime.FileFilter{
			{DisplayName: "TSV Files (*.tsv)", Pattern: "*.tsv"},
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	meta := a.reader.Meta()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	status := "InProgress"
	if snapshot.canceled {
		status = "Canceled"
	} else if snapshot.done {
		status = "Complete"
	}
	_, _ = fmt.Fprintf(writer, "Source\t%s\n", sanitizeTSV(meta.Path))
	_, _ = fmt.Fprintf(writer, "Encoding\t%s\n", sanitizeTSV(snapshot.encoding))
	_, _ = fmt.Fprintf(writer, "Keyword\t%s\n", sanitizeTSV(snapshot.keyword))
	_, _ = fmt.Fprintf(writer, "Regex\t%t\n", snapshot.options.Regex)
	_, _ = fmt.Fprintf(writer, "CaseSensitive\t%t\n", snapshot.options.CaseSensitive)
	_, _ = fmt.Fprintf(writer, "Status\t%s\n", status)
	_, _ = fmt.Fprintf(writer, "ScannedOffset\t%d\n", snapshot.scannedOffset)
	_, _ = fmt.Fprintf(writer, "FileSize\t%d\n", snapshot.fileSize)
	_, _ = fmt.Fprintf(writer, "TotalDiscovered\t%d\n\n", snapshot.total)
	_, _ = writer.WriteString("Index\tLine\tOffset\tByteLength\tPreview\n")

	const batchSize = 200
	for offset := 0; offset < snapshot.total; offset += batchSize {
		refs, err := session.store.Window(offset, batchSize)
		if err != nil {
			return "", err
		}
		hits, err := a.reader.BuildSearchHitPreviewsFromRefs(refs)
		if err != nil {
			return "", err
		}
		for _, hit := range hits {
			_, _ = fmt.Fprintf(writer, "%d\t%d\t%d\t%d\t%s\n", hit.Index+1, hit.LineNumber, hit.Offset, hit.ByteLength, sanitizeTSV(hit.LinePreview))
		}
	}
	if err := writer.Flush(); err != nil {
		return "", err
	}
	return path, nil
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
	a.searchMu.Lock()
	session := a.search
	a.search = nil
	a.searchMu.Unlock()
	cleanupSearchSession(session)
}

func (a *App) setActiveSearchSession(session *searchSession) {
	a.searchMu.Lock()
	old := a.search
	a.search = session
	a.searchMu.Unlock()
	cleanupSearchSession(old)
}

func cleanupSearchSession(session *searchSession) {
	if session == nil {
		return
	}
	if session.cancel != nil {
		session.cancel()
	}
	if session.store != nil {
		_ = session.store.Remove()
	}
}

func (a *App) activeSearchSession(searchID string) (*searchSession, error) {
	a.searchMu.Lock()
	session := a.search
	a.searchMu.Unlock()
	if session == nil || searchID == "" || session.id != searchID {
		return nil, errors.New(a.msg("error.searchExpired"))
	}
	return session, nil
}

func (s *searchSession) summary() reader.SearchSessionSummary {
	snapshot := s.snapshot()
	return reader.SearchSessionSummary{SearchID: snapshot.id, Keyword: snapshot.keyword, Total: snapshot.total, ScannedOffset: snapshot.scannedOffset, FileSize: snapshot.fileSize, Encoding: snapshot.encoding, Done: snapshot.done, Canceled: snapshot.canceled, Error: snapshot.errText, Regex: snapshot.options.Regex, CaseSensitive: snapshot.options.CaseSensitive}
}

func (s *searchSession) snapshot() searchSessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := 0
	if s.store != nil {
		total = s.store.Count()
	}
	return searchSessionSnapshot{id: s.id, keyword: s.keyword, options: s.options, total: total, scannedOffset: s.scannedOffset, fileSize: s.fileSize, encoding: s.encoding, done: s.done, canceled: s.canceled, errText: s.errText}
}

func (a *App) runSearchSession(ctx context.Context, session *searchSession) {
	lastEmit := time.Now()
	emit := func(force bool) {
		if a.ctx == nil || (!force && time.Since(lastEmit) < 200*time.Millisecond) {
			return
		}
		lastEmit = time.Now()
		runtime.EventsEmit(a.ctx, "search:progress", session.summary())
	}
	err := session.source.StreamSearchWithOptions(ctx, session.keyword, session.options, func(ref reader.SearchHitRef) error {
		if err := session.store.Append(ref); err != nil {
			return err
		}
		emit(false)
		return nil
	}, func(scannedOffset int64) error {
		session.mu.Lock()
		if scannedOffset > session.scannedOffset {
			session.scannedOffset = scannedOffset
		}
		session.mu.Unlock()
		emit(false)
		return nil
	})
	session.mu.Lock()
	if err != nil && !errors.Is(err, context.Canceled) {
		session.errText = a.searchError(err).Error()
	}
	if session.scannedOffset < session.fileSize && err == nil {
		session.scannedOffset = session.fileSize
	}
	if errors.Is(err, context.Canceled) {
		session.canceled = true
	}
	session.done = true
	session.mu.Unlock()
	emit(true)
}

func (a *App) searchError(err error) error {
	if errors.Is(err, reader.ErrEmptyRegexMatch) {
		return errors.New(a.msg("error.regexEmptyMatch"))
	}
	return err
}

func (a *App) validateSearchSession(searchID string) error {
	_, err := a.activeSearchSession(searchID)
	return err
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

func sanitizeTSV(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.ReplaceAll(value, "\n", " ")
}

var localizedMessages = map[string]map[string]string{
	"zh": {
		"dialog.openFile":            "选择要阅读的文本文件",
		"dialog.openFolder":          "选择文件夹",
		"dialog.exportSearchResults": "导出搜索结果",
		"error.selectFile":           "请选择文件",
		"error.noFileOpen":           "尚未打开文件",
		"error.enterKeyword":         "请输入搜索关键字",
		"error.notFound":             "没有找到：%s",
		"error.searchResultMissing":  "搜索结果不存在",
		"error.bookmarkMissing":      "书签不存在",
		"error.searchExpired":        "搜索结果已过期，请重新搜索",
		"error.noSearchResults":      "没有可导出的搜索结果",
		"error.regexEmptyMatch":      "正则表达式不能匹配空文本",
	},
	"en": {
		"dialog.openFile":            "Select a text file to read",
		"dialog.openFolder":          "Select folder",
		"dialog.exportSearchResults": "Export search results",
		"error.selectFile":           "Please select a file",
		"error.noFileOpen":           "No file is open",
		"error.enterKeyword":         "Enter a search keyword",
		"error.notFound":             "Not found: %s",
		"error.searchResultMissing":  "Search result does not exist",
		"error.bookmarkMissing":      "Bookmark does not exist",
		"error.searchExpired":        "Search results expired. Search again",
		"error.noSearchResults":      "No search results to export",
		"error.regexEmptyMatch":      "Regex must not match empty text",
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
