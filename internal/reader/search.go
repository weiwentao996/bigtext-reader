package reader

import (
	"bytes"
	"errors"
	"io"
	"unicode/utf8"
)

var ErrNotFound = errors.New("keyword not found")

func (r *Reader) SearchForward(startOffset int64, keyword string) (SearchResult, error) {
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
	if limit < 0 {
		limit = 0
	}
	summary := SearchSummary{Keyword: keyword, Hits: []SearchHit{}, Limit: limit, FileSize: r.meta.Size, Encoding: r.meta.Encoding}
	refs, err := r.BuildSearchIndex(keyword)
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
	hits := make([]SearchHit, 0, end-offset)
	for i := offset; i < end; i++ {
		hit, err := r.buildSearchHitFromRef(refs[i])
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
