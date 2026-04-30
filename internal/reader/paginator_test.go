package reader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPageUTF8(t *testing.T) {
	path := writeTempFile(t, "line1\nline2\nline3\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, PageSize: 2, ChunkSize: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	page, err := r.ReadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Lines) != 2 || page.Lines[0] != "line1" || page.Lines[1] != "line2" {
		t.Fatalf("unexpected page: %#v", page.Lines)
	}
	if page.EndOffset != int64(len("line1\nline2\n")) {
		t.Fatalf("unexpected end offset: %d", page.EndOffset)
	}
	if len(page.LineStartOffsets) != len(page.Lines) || len(page.LineEndOffsets) != len(page.Lines) {
		t.Fatalf("line offset metadata length mismatch: %#v", page)
	}
	if page.LineStartOffsets[0] != 0 || page.LineEndOffsets[0] != int64(len("line1")) {
		t.Fatalf("unexpected first line offsets: %#v %#v", page.LineStartOffsets, page.LineEndOffsets)
	}
	if page.LineStartOffsets[1] != int64(len("line1\n")) || page.LineEndOffsets[1] != int64(len("line1\nline2")) {
		t.Fatalf("unexpected second line offsets: %#v %#v", page.LineStartOffsets, page.LineEndOffsets)
	}

	next, err := r.ReadPage(page.EndOffset)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Lines) != 1 || next.Lines[0] != "line3" || !next.EOF {
		t.Fatalf("unexpected next page: %#v", next)
	}
}

func TestNormalizeOffsetToNextLine(t *testing.T) {
	path := writeTempFile(t, "alpha\nbeta\ngamma\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	offset, err := r.NormalizeOffsetToNextLine(2)
	if err != nil {
		t.Fatal(err)
	}
	if offset != int64(len("alpha\n")) {
		t.Fatalf("unexpected offset: %d", offset)
	}
}

func TestReadPreviousPageFromBoundary(t *testing.T) {
	path := writeTempFile(t, "line1\nline2\nline3\nline4\nline5\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, PageSize: 2, ChunkSize: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	first, err := r.ReadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.ReadPage(first.EndOffset)
	if err != nil {
		t.Fatal(err)
	}
	prev, err := r.ReadPreviousPage(second.StartOffset)
	if err != nil {
		t.Fatal(err)
	}
	if prev.StartOffset != first.StartOffset || prev.EndOffset != first.EndOffset {
		t.Fatalf("unexpected previous offsets: %#v want %#v", prev, first)
	}
	if len(prev.Lines) != 2 || prev.Lines[0] != "line1" || prev.Lines[1] != "line2" {
		t.Fatalf("unexpected previous lines: %#v", prev.Lines)
	}
}

func TestReadPreviousPageFromEOFWithoutTrailingNewline(t *testing.T) {
	path := writeTempFile(t, "a\nb\nc\nd\ne")
	r, err := Open(path, Config{Encoding: EncodingUTF8, PageSize: 2, ChunkSize: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	prev, err := r.ReadPreviousPage(r.Meta().Size)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Lines) != 2 || prev.Lines[0] != "d" || prev.Lines[1] != "e" {
		t.Fatalf("unexpected previous lines: %#v", prev.Lines)
	}
}

func TestReadPreviousPageInsideLine(t *testing.T) {
	path := writeTempFile(t, "line1\nline2\nline3\nline4\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, PageSize: 2, ChunkSize: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	insideLine3 := int64(len("line1\nline2\nli"))
	prev, err := r.ReadPreviousPage(insideLine3)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Lines) != 2 || prev.Lines[0] != "line1" || prev.Lines[1] != "line2" {
		t.Fatalf("unexpected previous lines: %#v", prev.Lines)
	}
}

func TestReadPreviousPageBOF(t *testing.T) {
	path := writeTempFile(t, "a\nb\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	prev, err := r.ReadPreviousPage(0)
	if err != nil {
		t.Fatal(err)
	}
	if !prev.BOF || prev.HasPrevious || prev.StartOffset != 0 || prev.EndOffset != 0 {
		t.Fatalf("unexpected BOF page: %#v", prev)
	}
}

func TestReadPreviousPageCRLF(t *testing.T) {
	path := writeTempFile(t, "a\r\nb\r\nc\r\nd\r\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, PageSize: 2, ChunkSize: 4})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	prev, err := r.ReadPreviousPage(r.Meta().Size)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Lines) != 2 || prev.Lines[0] != "c" || prev.Lines[1] != "d" {
		t.Fatalf("unexpected CRLF lines: %#v", prev.Lines)
	}
}

func TestReadWindowAround(t *testing.T) {
	path := writeTempFile(t, "1\n2\n3\n4\n5\n6\n7\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, PageSize: 2, ChunkSize: 4})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	anchorOffset := int64(len("1\n2\n"))
	window, err := r.ReadWindowAround(anchorOffset, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(window.Pages) != 3 {
		t.Fatalf("unexpected window size: %d", len(window.Pages))
	}
	for i := 1; i < len(window.Pages); i++ {
		if window.Pages[i-1].StartOffset >= window.Pages[i].StartOffset {
			t.Fatalf("pages not ordered: %#v", window.Pages)
		}
	}
	if window.Anchor.StartOffset != anchorOffset {
		t.Fatalf("unexpected anchor: %#v", window.Anchor)
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
