package reader

import (
	"bytes"
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

var ErrNotFound = errors.New("keyword not found")
var ErrEmptyRegexMatch = errors.New("regex must not match empty text")

func (r *Reader) SearchForward(startOffset int64, keyword string) (SearchResult, error) {
	return r.SearchForwardWithOptions(startOffset, keyword, SearchOptions{CaseSensitive: true})
}

func (r *Reader) SearchForwardWithOptions(startOffset int64, keyword string, options SearchOptions) (SearchResult, error) {
	if options.Regex || !options.CaseSensitive {
		refs, err := r.BuildSearchIndexWithOptions(keyword, options)
		if err != nil {
			return SearchResult{}, err
		}
		for _, ref := range refs {
			if ref.Offset >= startOffset {
				return SearchResult{Offset: ref.Offset, ByteLength: ref.ByteLength}, nil
			}
		}
		return SearchResult{}, ErrNotFound
	}
	return r.searchForwardLiteralBytes(startOffset, keyword)
}

func (r *Reader) searchForwardLiteralBytes(startOffset int64, keyword string) (SearchResult, error) {
	needle, err := EncodeKeyword(keyword, r.meta.Encoding)
	if err != nil {
		return SearchResult{}, err
	}
	if len(needle) == 0 {
		return SearchResult{}, ErrNotFound
	}
	if startOffset < 0 {
		startOffset = 0
	}
	if startOffset >= r.meta.Size {
		return SearchResult{}, ErrNotFound
	}

	current := startOffset
	var tail []byte
	for current < r.meta.Size {
		chunk, err := r.readAt(current, r.config.SearchChunk)
		if err != nil && err != io.EOF {
			return SearchResult{}, err
		}
		if len(chunk) == 0 {
			break
		}

		baseOffset := current
		data := chunk
		if len(tail) > 0 {
			data = make([]byte, len(tail)+len(chunk))
			copy(data, tail)
			copy(data[len(tail):], chunk)
			baseOffset = current - int64(len(tail))
		}

		idx := bytes.Index(data, needle)
		if idx >= 0 {
			return SearchResult{Offset: baseOffset + int64(idx), ByteLength: len(needle)}, nil
		}

		keep := len(needle) - 1
		if keep > len(data) {
			keep = len(data)
		}
		if keep > 0 {
			if cap(tail) < keep {
				tail = make([]byte, keep)
			} else {
				tail = tail[:keep]
			}
			copy(tail, data[len(data)-keep:])
		} else {
			tail = nil
		}

		current += int64(len(chunk))
		if err == io.EOF {
			break
		}
	}
	return SearchResult{}, ErrNotFound
}

func (r *Reader) SearchAll(keyword string, limit int) (SearchSummary, error) {
	return r.SearchAllWithOptions(keyword, limit, SearchOptions{CaseSensitive: true})
}

func (r *Reader) SearchAllWithOptions(keyword string, limit int, options SearchOptions) (SearchSummary, error) {
	if limit < 0 {
		limit = 0
	}
	summary := SearchSummary{Keyword: keyword, Hits: []SearchHit{}, Limit: limit, FileSize: r.meta.Size, Encoding: r.meta.Encoding, Regex: options.Regex, CaseSensitive: options.CaseSensitive}
	refs, err := r.BuildSearchIndexWithOptions(keyword, options)
	if err != nil {
		return SearchSummary{}, err
	}
	summary.Total = len(refs)
	for i := 0; i < len(refs) && i < limit; i++ {
		ref := refs[i]
		hit, err := r.BuildSearchHitPreview(ref.Index, ref.Offset, ref.ByteLength)
		if err != nil {
			return SearchSummary{}, err
		}
		summary.Hits = append(summary.Hits, hit)
	}
	summary.Truncated = summary.Total > len(summary.Hits)
	return summary, nil
}

func (r *Reader) BuildSearchIndex(keyword string) ([]SearchHitRef, error) {
	return r.BuildSearchIndexWithOptions(keyword, SearchOptions{CaseSensitive: true})
}

func (r *Reader) BuildSearchIndexWithOptions(keyword string, options SearchOptions) ([]SearchHitRef, error) {
	refs := []SearchHitRef{}
	err := r.StreamSearchWithOptions(context.Background(), keyword, options, func(ref SearchHitRef) error {
		refs = append(refs, ref)
		return nil
	}, nil)
	if err != nil {
		return nil, err
	}
	return refs, nil
}

type SearchHitCallback func(SearchHitRef) error

type SearchProgressCallback func(scannedOffset int64) error

func (r *Reader) ValidateSearchOptions(keyword string, options SearchOptions) error {
	_, err := r.newStreamingSearchMatcher(keyword, options)
	return err
}

func (r *Reader) StreamSearchWithOptions(ctx context.Context, keyword string, options SearchOptions, onHit SearchHitCallback, onProgress SearchProgressCallback) error {
	if strings.TrimSpace(keyword) == "" {
		return nil
	}
	matcher, err := r.newStreamingSearchMatcher(keyword, options)
	if err != nil {
		return err
	}
	hitIndex := 0
	lineStart := int64(0)
	lineNumber := int64(1)
	current := int64(0)
	lineBytes := []byte{}
	for current < r.meta.Size {
		if err := ctx.Err(); err != nil {
			return err
		}
		chunk, err := r.readAt(current, r.config.SearchChunk)
		if err != nil && err != io.EOF {
			return err
		}
		if len(chunk) == 0 {
			break
		}
		for i, b := range chunk {
			lineBytes = append(lineBytes, b)
			if b != '\n' {
				continue
			}
			lineEnd := current + int64(i)
			if err := r.emitSearchLine(ctx, matcher, lineBytes[:len(lineBytes)-1], lineStart, lineEnd, lineNumber, &hitIndex, onHit); err != nil {
				return err
			}
			lineBytes = lineBytes[:0]
			lineStart = lineEnd + 1
			lineNumber++
		}
		current += int64(len(chunk))
		if onProgress != nil {
			if err := onProgress(current); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
	}
	if len(lineBytes) > 0 || lineStart < r.meta.Size || r.meta.Size == 0 {
		if err := r.emitSearchLine(ctx, matcher, lineBytes, lineStart, r.meta.Size, lineNumber, &hitIndex, onHit); err != nil {
			return err
		}
	}
	if onProgress != nil {
		return onProgress(r.meta.Size)
	}
	return nil
}

type streamingSearchMatcher struct {
	keyword string
	options SearchOptions
	needle  []byte
	pattern *regexp.Regexp
}

func (r *Reader) newStreamingSearchMatcher(keyword string, options SearchOptions) (streamingSearchMatcher, error) {
	matcher := streamingSearchMatcher{keyword: keyword, options: options}
	if !options.Regex && options.CaseSensitive {
		needle, err := EncodeKeyword(keyword, r.meta.Encoding)
		if err != nil {
			return streamingSearchMatcher{}, err
		}
		matcher.needle = needle
		return matcher, nil
	}
	if options.Regex {
		patternText := keyword
		if !options.CaseSensitive {
			patternText = "(?i)" + patternText
		}
		pattern, err := regexp.Compile(patternText)
		if err != nil {
			return streamingSearchMatcher{}, err
		}
		if pattern.MatchString("") {
			return streamingSearchMatcher{}, ErrEmptyRegexMatch
		}
		matcher.pattern = pattern
	}
	return matcher, nil
}

func (r *Reader) emitSearchLine(ctx context.Context, matcher streamingSearchMatcher, lineBytes []byte, lineStart int64, lineEnd int64, lineNumber int64, hitIndex *int, onHit SearchHitCallback) error {
	if onHit == nil || lineEnd < lineStart {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !matcher.options.Regex && matcher.options.CaseSensitive {
		return emitLiteralByteMatches(ctx, matcher.needle, lineBytes, lineStart, lineEnd, lineNumber, hitIndex, onHit)
	}
	decoded, err := r.decodeLineWithByteMap(lineBytes)
	if err != nil {
		return err
	}
	matches := decoded.searchMatches(matcher.keyword, matcher.pattern, matcher.options)
	for _, match := range matches {
		if err := ctx.Err(); err != nil {
			return err
		}
		if match.startRune < 0 || match.endRune <= match.startRune || match.endRune > len(decoded.runeByteEnds) {
			continue
		}
		startByte := decoded.runeByteStarts[match.startRune]
		endByte := decoded.runeByteEnds[match.endRune-1]
		ref := SearchHitRef{Index: *hitIndex, Offset: lineStart + int64(startByte), ByteLength: endByte - startByte, LineStart: lineStart, LineEnd: lineEnd, LineNumber: lineNumber}
		*hitIndex = *hitIndex + 1
		if err := onHit(ref); err != nil {
			return err
		}
	}
	return nil
}

func emitLiteralByteMatches(ctx context.Context, needle []byte, lineBytes []byte, lineStart int64, lineEnd int64, lineNumber int64, hitIndex *int, onHit SearchHitCallback) error {
	if len(needle) == 0 || len(lineBytes) < len(needle) {
		return nil
	}
	searchFrom := 0
	for searchFrom <= len(lineBytes) {
		if err := ctx.Err(); err != nil {
			return err
		}
		idx := bytes.Index(lineBytes[searchFrom:], needle)
		if idx < 0 {
			break
		}
		hitStart := searchFrom + idx
		ref := SearchHitRef{Index: *hitIndex, Offset: lineStart + int64(hitStart), ByteLength: len(needle), LineStart: lineStart, LineEnd: lineEnd, LineNumber: lineNumber}
		*hitIndex = *hitIndex + 1
		if err := onHit(ref); err != nil {
			return err
		}
		searchFrom += idx + maxInt(1, len(needle))
	}
	return nil
}

func (r *Reader) buildLiteralByteSearchIndex(keyword string) ([]SearchHitRef, error) {
	needle, err := EncodeKeyword(keyword, r.meta.Encoding)
	if err != nil {
		return nil, err
	}
	if len(needle) == 0 {
		return []SearchHitRef{}, nil
	}

	lineStarts, lineEnds, lineNumbers, err := r.collectLineBoundaries()
	if err != nil {
		return nil, err
	}

	refs := []SearchHitRef{}
	current := int64(0)
	var tail []byte
	lastRecordedOffset := int64(-1)
	for current < r.meta.Size {
		chunk, err := r.readAt(current, r.config.SearchChunk)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if len(chunk) == 0 {
			break
		}

		baseOffset := current
		data := chunk
		if len(tail) > 0 {
			data = make([]byte, len(tail)+len(chunk))
			copy(data, tail)
			copy(data[len(tail):], chunk)
			baseOffset = current - int64(len(tail))
		}

		searchFrom := 0
		for searchFrom <= len(data) {
			idx := bytes.Index(data[searchFrom:], needle)
			if idx < 0 {
				break
			}
			hitOffset := baseOffset + int64(searchFrom+idx)
			if hitOffset+int64(len(needle)) > current && hitOffset != lastRecordedOffset {
				lineIndex := findLineBoundaryIndex(lineStarts, lineEnds, hitOffset)
				if lineIndex < 0 {
					return nil, ErrNotFound
				}
				refs = append(refs, SearchHitRef{
					Index:      len(refs),
					Offset:     hitOffset,
					ByteLength: len(needle),
					LineStart:  lineStarts[lineIndex],
					LineEnd:    lineEnds[lineIndex],
					LineNumber: lineNumbers[lineIndex],
				})
				lastRecordedOffset = hitOffset
			}
			searchFrom += idx + maxInt(1, len(needle))
		}

		keep := len(needle) - 1
		if keep > len(data) {
			keep = len(data)
		}
		if keep > 0 {
			if cap(tail) < keep {
				tail = make([]byte, keep)
			} else {
				tail = tail[:keep]
			}
			copy(tail, data[len(data)-keep:])
		} else {
			tail = nil
		}

		current += int64(len(chunk))
		if err == io.EOF {
			break
		}
	}
	return refs, nil
}

func (r *Reader) buildDecodedSearchIndex(keyword string, options SearchOptions) ([]SearchHitRef, error) {
	if strings.TrimSpace(keyword) == "" {
		return []SearchHitRef{}, nil
	}

	lineStarts, lineEnds, lineNumbers, err := r.collectLineBoundaries()
	if err != nil {
		return nil, err
	}

	var pattern *regexp.Regexp
	if options.Regex {
		patternText := keyword
		if !options.CaseSensitive {
			patternText = "(?i)" + patternText
		}
		pattern, err = regexp.Compile(patternText)
		if err != nil {
			return nil, err
		}
		if pattern.MatchString("") {
			return nil, ErrEmptyRegexMatch
		}
	}

	refs := []SearchHitRef{}
	for i := range lineStarts {
		lineStart := lineStarts[i]
		lineEnd := lineEnds[i]
		if lineEnd < lineStart {
			continue
		}
		lineBytes, err := r.readAt(lineStart, int(lineEnd-lineStart))
		if err != nil && err != io.EOF {
			return nil, err
		}
		decoded, err := r.decodeLineWithByteMap(lineBytes)
		if err != nil {
			return nil, err
		}
		matches := decoded.searchMatches(keyword, pattern, options)
		for _, match := range matches {
			if match.startRune < 0 || match.endRune <= match.startRune || match.endRune > len(decoded.runeByteEnds) {
				continue
			}
			startByte := decoded.runeByteStarts[match.startRune]
			endByte := decoded.runeByteEnds[match.endRune-1]
			refs = append(refs, SearchHitRef{
				Index:      len(refs),
				Offset:     lineStart + int64(startByte),
				ByteLength: endByte - startByte,
				LineStart:  lineStart,
				LineEnd:    lineEnd,
				LineNumber: lineNumbers[i],
			})
		}
	}
	return refs, nil
}

type decodedSearchLine struct {
	text           string
	runes          []rune
	runeByteStarts []int
	runeByteEnds   []int
}

type decodedMatch struct {
	startRune int
	endRune   int
}

func (r *Reader) decodeLineWithByteMap(data []byte) (decodedSearchLine, error) {
	data = bytes.TrimSuffix(data, []byte{'\r'})
	result := decodedSearchLine{
		runes:          []rune{},
		runeByteStarts: []int{},
		runeByteEnds:   []int{},
	}
	for pos := 0; pos < len(data); {
		width := encodedRuneWidth(data[pos:], r.meta.Encoding)
		if width <= 0 {
			width = 1
		}
		if pos+width > len(data) {
			width = len(data) - pos
		}
		part, err := decodeBytes(data[pos:pos+width], r.meta.Encoding)
		if err != nil {
			return decodedSearchLine{}, err
		}
		partRunes := []rune(part)
		if len(partRunes) == 0 {
			pos += width
			continue
		}
		for _, rn := range partRunes {
			result.runes = append(result.runes, rn)
			result.runeByteStarts = append(result.runeByteStarts, pos)
			result.runeByteEnds = append(result.runeByteEnds, pos+width)
		}
		pos += width
	}
	result.text = string(result.runes)
	return result, nil
}

func encodedRuneWidth(data []byte, encodingName string) int {
	if len(data) == 0 {
		return 0
	}
	encodingName = normalizeEncodingName(encodingName)
	b := data[0]
	switch encodingName {
	case "", EncodingAuto, EncodingUTF8:
		_, width := utf8.DecodeRune(data)
		if width <= 0 {
			return 1
		}
		return width
	case EncodingWindows1252:
		return 1
	case EncodingGB18030:
		if len(data) >= 4 && b >= 0x81 && b <= 0xfe && data[1] >= 0x30 && data[1] <= 0x39 && data[2] >= 0x81 && data[2] <= 0xfe && data[3] >= 0x30 && data[3] <= 0x39 {
			return 4
		}
		if b < 0x80 {
			return 1
		}
		return 2
	case EncodingGBK, EncodingBig5, EncodingEUCKR:
		if b < 0x80 {
			return 1
		}
		return 2
	case EncodingShiftJIS:
		if b < 0x80 || b >= 0xa1 && b <= 0xdf {
			return 1
		}
		return 2
	default:
		if b < 0x80 {
			return 1
		}
		return 2
	}
}

func (line decodedSearchLine) searchMatches(keyword string, pattern *regexp.Regexp, options SearchOptions) []decodedMatch {
	if options.Regex {
		return line.regexMatches(pattern)
	}
	if options.CaseSensitive {
		return line.literalMatches([]rune(keyword))
	}
	return line.literalFoldMatches([]rune(keyword))
}

func (line decodedSearchLine) regexMatches(pattern *regexp.Regexp) []decodedMatch {
	if pattern == nil {
		return []decodedMatch{}
	}
	indexes := pattern.FindAllStringIndex(line.text, -1)
	matches := make([]decodedMatch, 0, len(indexes))
	for _, index := range indexes {
		if index[0] == index[1] {
			continue
		}
		matches = append(matches, decodedMatch{startRune: stringByteIndexToRuneIndex(line.text, index[0]), endRune: stringByteIndexToRuneIndex(line.text, index[1])})
	}
	return matches
}

func (line decodedSearchLine) literalMatches(needle []rune) []decodedMatch {
	if len(needle) == 0 || len(needle) > len(line.runes) {
		return []decodedMatch{}
	}
	matches := []decodedMatch{}
	for i := 0; i <= len(line.runes)-len(needle); {
		matched := true
		for j := range needle {
			if line.runes[i+j] != needle[j] {
				matched = false
				break
			}
		}
		if matched {
			matches = append(matches, decodedMatch{startRune: i, endRune: i + len(needle)})
			i += maxInt(1, len(needle))
			continue
		}
		i++
	}
	return matches
}

func (line decodedSearchLine) literalFoldMatches(needle []rune) []decodedMatch {
	if len(needle) == 0 || len(needle) > len(line.runes) {
		return []decodedMatch{}
	}
	needleText := string(needle)
	matches := []decodedMatch{}
	for i := 0; i <= len(line.runes)-len(needle); {
		candidate := string(line.runes[i : i+len(needle)])
		if strings.EqualFold(candidate, needleText) {
			matches = append(matches, decodedMatch{startRune: i, endRune: i + len(needle)})
			i += maxInt(1, len(needle))
			continue
		}
		i++
	}
	return matches
}

func stringByteIndexToRuneIndex(text string, byteIndex int) int {
	if byteIndex <= 0 {
		return 0
	}
	if byteIndex >= len(text) {
		return utf8.RuneCountInString(text)
	}
	return utf8.RuneCountInString(text[:byteIndex])
}

func (r *Reader) collectLineBoundaries() ([]int64, []int64, []int64, error) {
	starts := []int64{0}
	ends := []int64{}
	numbers := []int64{1}
	current := int64(0)
	lineNumber := int64(1)
	for current < r.meta.Size {
		chunk, err := r.readAt(current, r.config.SearchChunk)
		if err != nil && err != io.EOF {
			return nil, nil, nil, err
		}
		for i, b := range chunk {
			if b != '\n' {
				continue
			}
			ends = append(ends, current+int64(i))
			nextStart := current + int64(i) + 1
			if nextStart < r.meta.Size {
				lineNumber++
				starts = append(starts, nextStart)
				numbers = append(numbers, lineNumber)
			}
		}
		current += int64(len(chunk))
		if err == io.EOF || len(chunk) == 0 {
			break
		}
	}
	if len(ends) < len(starts) {
		ends = append(ends, r.meta.Size)
	}
	return starts, ends, numbers, nil
}

func (r *Reader) BuildSearchHitPreview(index int, offset int64, byteLength int) (SearchHit, error) {
	lineStarts, lineEnds, lineNumbers, err := r.collectLineBoundaries()
	if err != nil {
		return SearchHit{}, err
	}
	return r.buildSearchHit(index, offset, byteLength, lineStarts, lineEnds, lineNumbers)
}

func (r *Reader) BuildSearchHitPreviewFromRef(ref SearchHitRef) (SearchHit, error) {
	return r.buildSearchHitFromRef(ref)
}

func (r *Reader) BuildSearchHitPreviews(refs []SearchHitRef, offset int, limit int) ([]SearchHit, error) {
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}
	if offset >= len(refs) || limit == 0 {
		return []SearchHit{}, nil
	}
	end := offset + limit
	if end > len(refs) {
		end = len(refs)
	}
	return r.BuildSearchHitPreviewsFromRefs(refs[offset:end])
}

func (r *Reader) BuildSearchHitPreviewsFromRefs(refs []SearchHitRef) ([]SearchHit, error) {
	hits := make([]SearchHit, 0, len(refs))
	for _, ref := range refs {
		hit, err := r.buildSearchHitFromRef(ref)
		if err != nil {
			return nil, err
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

func (r *Reader) buildSearchHitFromRef(ref SearchHitRef) (SearchHit, error) {
	return r.buildSearchHitAtLine(ref.Index, ref.Offset, ref.ByteLength, ref.LineStart, ref.LineEnd, ref.LineNumber)
}

func (r *Reader) buildSearchHit(index int, offset int64, byteLength int, lineStarts []int64, lineEnds []int64, lineNumbers []int64) (SearchHit, error) {
	lineIndex := findLineBoundaryIndex(lineStarts, lineEnds, offset)
	if lineIndex < 0 {
		return SearchHit{}, ErrNotFound
	}
	return r.buildSearchHitAtLine(index, offset, byteLength, lineStarts[lineIndex], lineEnds[lineIndex], lineNumbers[lineIndex])
}

func (r *Reader) buildSearchHitAtLine(index int, offset int64, byteLength int, lineStart int64, lineEnd int64, lineNumber int64) (SearchHit, error) {
	hitEnd := offset + int64(byteLength)
	if hitEnd > lineEnd {
		hitEnd = lineEnd
	}
	previewStart := lineStart
	if offset-previewStart > int64(r.config.MaxLineBytes) {
		previewStart = offset - int64(r.config.MaxLineBytes/3)
		if previewStart < lineStart {
			previewStart = lineStart
		}
	}
	previewEnd := lineEnd
	if previewEnd-previewStart > int64(r.config.MaxLineBytes) {
		previewEnd = previewStart + int64(r.config.MaxLineBytes)
	}
	previewBytes, err := r.readAt(previewStart, int(previewEnd-previewStart))
	if err != nil && err != io.EOF {
		return SearchHit{}, err
	}
	prefixBytes, err := r.readAt(previewStart, int(offset-previewStart))
	if err != nil && err != io.EOF {
		return SearchHit{}, err
	}
	hitBytes, err := r.readAt(offset, int(hitEnd-offset))
	if err != nil && err != io.EOF {
		return SearchHit{}, err
	}
	preview, err := r.decoder.DecodeLine(previewBytes)
	if err != nil {
		return SearchHit{}, err
	}
	if previewStart > lineStart {
		preview = "… " + preview
	}
	if previewEnd < lineEnd {
		preview += " …[line truncated]"
	}
	prefix, err := r.decoder.DecodeLine(prefixBytes)
	if err != nil {
		return SearchHit{}, err
	}
	hit, err := r.decoder.DecodeLine(hitBytes)
	if err != nil {
		return SearchHit{}, err
	}
	start := utf8.RuneCountInString(prefix)
	if previewStart > lineStart {
		start += 2
	}
	return SearchHit{
		Index:         index,
		Offset:        offset,
		ByteLength:    byteLength,
		LineStart:     lineStart,
		LineEnd:       lineEnd,
		LineNumber:    lineNumber,
		LinePreview:   preview,
		LineCharStart: start,
		LineCharEnd:   start + utf8.RuneCountInString(hit),
	}, nil
}

func findLineBoundaryIndex(starts []int64, ends []int64, offset int64) int {
	lo := 0
	hi := len(starts) - 1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if offset < starts[mid] {
			hi = mid - 1
			continue
		}
		if offset > ends[mid] {
			lo = mid + 1
			continue
		}
		return mid
	}
	return -1
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func (r *Reader) LocateHit(page Page, hitOffset int64, hitByteLength int) (SearchHitLocation, error) {
	for i := range page.LineStartOffsets {
		lineStart := page.LineStartOffsets[i]
		lineEnd := page.LineEndOffsets[i]
		if hitOffset < lineStart || hitOffset > lineEnd {
			continue
		}

		hitEnd := hitOffset + int64(hitByteLength)
		if hitEnd > lineEnd {
			hitEnd = lineEnd
		}
		prefixBytes, err := r.readAt(lineStart, int(hitOffset-lineStart))
		if err != nil && err != io.EOF {
			return SearchHitLocation{}, err
		}
		hitBytes, err := r.readAt(hitOffset, int(hitEnd-hitOffset))
		if err != nil && err != io.EOF {
			return SearchHitLocation{}, err
		}
		prefix, err := r.decoder.DecodeLine(prefixBytes)
		if err != nil {
			return SearchHitLocation{}, err
		}
		hit, err := r.decoder.DecodeLine(hitBytes)
		if err != nil {
			return SearchHitLocation{}, err
		}
		start := utf8.RuneCountInString(prefix)
		return SearchHitLocation{LineIndex: i, LineCharStart: start, LineCharEnd: start + utf8.RuneCountInString(hit)}, nil
	}
	return SearchHitLocation{}, ErrNotFound
}
