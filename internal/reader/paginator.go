package reader

import (
	"bytes"
	"io"
)

func (r *Reader) ReadPage(offset int64) (Page, error) {
	if offset < 0 {
		offset = 0
	}
	if offset > r.meta.Size {
		offset = r.meta.Size
	}

	page := Page{
		StartOffset:      offset,
		EndOffset:        offset,
		Lines:            make([]string, 0, r.config.PageSize),
		LineStartOffsets: make([]int64, 0, r.config.PageSize),
		LineEndOffsets:   make([]int64, 0, r.config.PageSize),
		EOF:              offset >= r.meta.Size,
		BOF:              offset <= 0,
		HasPrevious:      offset > 0,
		Encoding:         r.meta.Encoding,
		FileSize:         r.meta.Size,
	}
	if page.EOF {
		return page, nil
	}

	current := offset
	var pending []byte
	for len(page.Lines) < r.config.PageSize && current < r.meta.Size {
		chunk, err := r.readAt(current, r.config.ChunkSize)
		if err != nil && err != io.EOF {
			return page, err
		}
		if len(chunk) == 0 {
			break
		}

		data := append(pending, chunk...)
		baseOffset := current - int64(len(pending))
		start := 0

		for len(page.Lines) < r.config.PageSize {
			idx := bytes.IndexByte(data[start:], '\n')
			if idx < 0 {
				break
			}
			lineEnd := start + idx
			lineBytes := data[start:lineEnd]
			lineStartOffset := baseOffset + int64(start)
			lineTextEndOffset := baseOffset + int64(lineEnd)
			lineOffsetEnd := lineTextEndOffset + 1
			line, truncated, err := r.decodeDisplayLine(lineBytes)
			if err != nil {
				return page, err
			}
			page.Lines = append(page.Lines, line)
			page.LineStartOffsets = append(page.LineStartOffsets, lineStartOffset)
			page.LineEndOffsets = append(page.LineEndOffsets, lineTextEndOffset)
			if truncated {
				page.Truncated = true
			}
			page.EndOffset = lineOffsetEnd
			start = lineEnd + 1
		}

		if len(page.Lines) >= r.config.PageSize {
			break
		}

		pending = data[start:]
		if len(pending) > r.config.MaxLineBytes {
			line, truncated, err := r.decodeDisplayLine(pending[:r.config.MaxLineBytes])
			if err != nil {
				return page, err
			}
			lineStartOffset := baseOffset + int64(start)
			lineEndOffset := lineStartOffset + int64(r.config.MaxLineBytes)
			page.Lines = append(page.Lines, line)
			page.LineStartOffsets = append(page.LineStartOffsets, lineStartOffset)
			page.LineEndOffsets = append(page.LineEndOffsets, lineEndOffset)
			page.Truncated = page.Truncated || truncated || true
			page.EndOffset = baseOffset + int64(len(pending))
			pending = nil
		}

		current += int64(len(chunk))
		if err == io.EOF {
			break
		}
	}

	if len(page.Lines) < r.config.PageSize && len(pending) > 0 {
		line, truncated, err := r.decodeDisplayLine(pending)
		if err != nil {
			return page, err
		}
		lineStartOffset := r.meta.Size - int64(len(pending))
		page.Lines = append(page.Lines, line)
		page.LineStartOffsets = append(page.LineStartOffsets, lineStartOffset)
		page.LineEndOffsets = append(page.LineEndOffsets, r.meta.Size)
		page.Truncated = page.Truncated || truncated
		page.EndOffset = r.meta.Size
	}

	if page.EndOffset >= r.meta.Size {
		page.EndOffset = r.meta.Size
		page.EOF = true
	}
	if page.EndOffset == page.StartOffset && !page.EOF {
		page.EndOffset = current
	}
	return page, nil
}

func (r *Reader) decodeDisplayLine(lineBytes []byte) (string, bool, error) {
	truncated := false
	if len(lineBytes) > r.config.MaxLineBytes {
		lineBytes = lineBytes[:r.config.MaxLineBytes]
		truncated = true
	}
	line, err := r.decoder.DecodeLine(lineBytes)
	if err != nil {
		return "", truncated, err
	}
	if truncated {
		line += " …[line truncated]"
	}
	return line, truncated, nil
}

func (r *Reader) NormalizeOffsetToNextLine(offset int64) (int64, error) {
	if offset <= 0 {
		return 0, nil
	}
	if offset >= r.meta.Size {
		return r.meta.Size, nil
	}

	current := offset
	for current < r.meta.Size {
		chunk, err := r.readAt(current, r.config.ChunkSize)
		if err != nil && err != io.EOF {
			return 0, err
		}
		idx := bytes.IndexByte(chunk, '\n')
		if idx >= 0 {
			return current + int64(idx) + 1, nil
		}
		current += int64(len(chunk))
		if err == io.EOF || len(chunk) == 0 {
			break
		}
	}
	return r.meta.Size, nil
}

func (r *Reader) FindLineStartNear(offset int64) (int64, error) {
	if offset <= 0 {
		return 0, nil
	}
	if offset > r.meta.Size {
		offset = r.meta.Size
	}

	start := offset - int64(r.config.LineStartScan)
	if start < 0 {
		start = 0
	}
	length := int(offset - start)
	chunk, err := r.readAt(start, length)
	if err != nil && err != io.EOF {
		return 0, err
	}
	idx := bytes.LastIndexByte(chunk, '\n')
	if idx < 0 {
		return start, nil
	}
	return start + int64(idx) + 1, nil
}

func (r *Reader) FindLineStartAtOrBefore(offset int64) (int64, error) {
	if offset <= 0 {
		return 0, nil
	}
	if offset >= r.meta.Size {
		return r.meta.Size, nil
	}

	previous, err := r.readAt(offset-1, 1)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if len(previous) == 1 && previous[0] == '\n' {
		return offset, nil
	}

	currentEnd := offset
	for currentEnd > 0 {
		start := currentEnd - int64(r.config.ChunkSize)
		if start < 0 {
			start = 0
		}
		chunk, err := r.readAt(start, int(currentEnd-start))
		if err != nil && err != io.EOF {
			return 0, err
		}
		idx := bytes.LastIndexByte(chunk, '\n')
		if idx >= 0 {
			lineStart := start + int64(idx) + 1
			if lineStart < offset {
				return lineStart, nil
			}
			if idx > 0 {
				prev := bytes.LastIndexByte(chunk[:idx], '\n')
				if prev >= 0 {
					return start + int64(prev) + 1, nil
				}
			}
		}
		currentEnd = start
	}
	return 0, nil
}

func (r *Reader) FindPreviousPageStart(beforeOffset int64) (int64, error) {
	beforeOffset, err := r.FindLineStartAtOrBefore(beforeOffset)
	if err != nil {
		return 0, err
	}
	if beforeOffset <= 0 {
		return 0, nil
	}

	boundaries := []int64{beforeOffset}
	currentEnd := beforeOffset
	for currentEnd > 0 && len(boundaries) <= r.config.PageSize {
		start := currentEnd - int64(r.config.ChunkSize)
		if start < 0 {
			start = 0
		}
		chunk, err := r.readAt(start, int(currentEnd-start))
		if err != nil && err != io.EOF {
			return 0, err
		}
		for i := len(chunk) - 1; i >= 0; i-- {
			if chunk[i] != '\n' {
				continue
			}
			lineStart := start + int64(i) + 1
			if lineStart < boundaries[len(boundaries)-1] {
				boundaries = append(boundaries, lineStart)
			}
			if len(boundaries) > r.config.PageSize {
				break
			}
		}
		currentEnd = start
	}

	if len(boundaries) <= r.config.PageSize {
		return 0, nil
	}
	return boundaries[r.config.PageSize], nil
}

func (r *Reader) ReadPreviousPage(beforeOffset int64) (Page, error) {
	if beforeOffset <= 0 {
		return Page{
			StartOffset: 0,
			EndOffset:   0,
			Lines:       []string{},
			EOF:         r.meta.Size == 0,
			BOF:         true,
			HasPrevious: false,
			Encoding:    r.meta.Encoding,
			FileSize:    r.meta.Size,
		}, nil
	}
	if beforeOffset > r.meta.Size {
		beforeOffset = r.meta.Size
	}

	endOffset, err := r.FindLineStartAtOrBefore(beforeOffset)
	if err != nil {
		return Page{}, err
	}
	if endOffset <= 0 && beforeOffset > 0 {
		endOffset = beforeOffset
	}
	startOffset, err := r.FindPreviousPageStart(endOffset)
	if err != nil {
		return Page{}, err
	}
	page, err := r.ReadPageUntil(startOffset, endOffset)
	if err != nil {
		return page, err
	}
	page.BOF = page.StartOffset == 0
	page.HasPrevious = page.StartOffset > 0
	return page, nil
}

func (r *Reader) ReadPageUntil(offset int64, maxEnd int64) (Page, error) {
	if maxEnd <= 0 || maxEnd > r.meta.Size {
		maxEnd = r.meta.Size
	}
	if offset < 0 {
		offset = 0
	}
	if offset > maxEnd {
		offset = maxEnd
	}

	page := Page{
		StartOffset:      offset,
		EndOffset:        offset,
		Lines:            make([]string, 0, r.config.PageSize),
		LineStartOffsets: make([]int64, 0, r.config.PageSize),
		LineEndOffsets:   make([]int64, 0, r.config.PageSize),
		EOF:              offset >= r.meta.Size,
		BOF:              offset <= 0,
		HasPrevious:      offset > 0,
		Encoding:         r.meta.Encoding,
		FileSize:         r.meta.Size,
	}
	if offset >= maxEnd {
		return page, nil
	}

	current := offset
	var pending []byte
	for len(page.Lines) < r.config.PageSize && current < maxEnd {
		readSize := r.config.ChunkSize
		if current+int64(readSize) > maxEnd {
			readSize = int(maxEnd - current)
		}
		chunk, err := r.readAt(current, readSize)
		if err != nil && err != io.EOF {
			return page, err
		}
		if len(chunk) == 0 {
			break
		}

		data := append(pending, chunk...)
		baseOffset := current - int64(len(pending))
		start := 0
		for len(page.Lines) < r.config.PageSize {
			idx := bytes.IndexByte(data[start:], '\n')
			if idx < 0 {
				break
			}
			lineEnd := start + idx
			lineStartOffset := baseOffset + int64(start)
			lineTextEndOffset := baseOffset + int64(lineEnd)
			lineOffsetEnd := lineTextEndOffset + 1
			if lineOffsetEnd > maxEnd {
				break
			}
			line, truncated, err := r.decodeDisplayLine(data[start:lineEnd])
			if err != nil {
				return page, err
			}
			page.Lines = append(page.Lines, line)
			page.LineStartOffsets = append(page.LineStartOffsets, lineStartOffset)
			page.LineEndOffsets = append(page.LineEndOffsets, lineTextEndOffset)
			page.Truncated = page.Truncated || truncated
			page.EndOffset = lineOffsetEnd
			start = lineEnd + 1
		}

		if len(page.Lines) >= r.config.PageSize {
			break
		}

		pending = data[start:]
		if len(pending) > r.config.MaxLineBytes {
			line, truncated, err := r.decodeDisplayLine(pending[:r.config.MaxLineBytes])
			if err != nil {
				return page, err
			}
			lineStartOffset := baseOffset + int64(start)
			lineEndOffset := lineStartOffset + int64(r.config.MaxLineBytes)
			page.Lines = append(page.Lines, line)
			page.LineStartOffsets = append(page.LineStartOffsets, lineStartOffset)
			page.LineEndOffsets = append(page.LineEndOffsets, lineEndOffset)
			page.Truncated = page.Truncated || truncated || true
			page.EndOffset = baseOffset + int64(len(pending))
			pending = nil
		}

		current += int64(len(chunk))
		if err == io.EOF {
			break
		}
	}

	if len(page.Lines) < r.config.PageSize && len(pending) > 0 && current >= maxEnd {
		line, truncated, err := r.decodeDisplayLine(pending)
		if err != nil {
			return page, err
		}
		lineStartOffset := maxEnd - int64(len(pending))
		page.Lines = append(page.Lines, line)
		page.LineStartOffsets = append(page.LineStartOffsets, lineStartOffset)
		page.LineEndOffsets = append(page.LineEndOffsets, maxEnd)
		page.Truncated = page.Truncated || truncated
		page.EndOffset = maxEnd
	}
	page.EOF = page.EndOffset >= r.meta.Size
	return page, nil
}

func (r *Reader) ReadWindowAround(offset int64, beforePages int, afterPages int) (PageWindow, error) {
	if beforePages < 0 {
		beforePages = 0
	}
	if afterPages < 0 {
		afterPages = 0
	}
	start, err := r.FindLineStartAtOrBefore(offset)
	if err != nil {
		return PageWindow{}, err
	}
	anchor, err := r.ReadPage(start)
	if err != nil {
		return PageWindow{}, err
	}
	pages := []Page{anchor}

	cursor := anchor
	for i := 0; i < beforePages && cursor.StartOffset > 0; i++ {
		prev, err := r.ReadPreviousPage(cursor.StartOffset)
		if err != nil {
			return PageWindow{}, err
		}
		if prev.EndOffset <= prev.StartOffset && prev.StartOffset == 0 {
			break
		}
		pages = append([]Page{prev}, pages...)
		cursor = prev
	}

	cursor = anchor
	for i := 0; i < afterPages && !cursor.EOF; i++ {
		next, err := r.ReadPage(cursor.EndOffset)
		if err != nil {
			return PageWindow{}, err
		}
		if next.StartOffset == cursor.StartOffset && next.EndOffset == cursor.EndOffset {
			break
		}
		pages = append(pages, next)
		cursor = next
	}

	return PageWindow{Pages: pages, Anchor: anchor, FileSize: r.meta.Size, Encoding: r.meta.Encoding}, nil
}
