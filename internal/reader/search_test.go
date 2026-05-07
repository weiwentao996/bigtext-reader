package reader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func TestSearchForwardAcrossChunks(t *testing.T) {
	path := writeTempFile(t, "aaaaTARGETbbbb")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 6})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	result, err := r.SearchForward(0, "TARGET")
	if err != nil {
		t.Fatal(err)
	}
	if result.Offset != 4 {
		t.Fatalf("unexpected offset: %d", result.Offset)
	}
	if result.ByteLength != len("TARGET") {
		t.Fatalf("unexpected byte length: %d", result.ByteLength)
	}
}

func TestSearchForwardNotFound(t *testing.T) {
	path := writeTempFile(t, "abcdef")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	_, err = r.SearchForward(0, "xyz")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSearchForwardFromNextHit(t *testing.T) {
	path := writeTempFile(t, "one TARGET two TARGET three")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	first, err := r.SearchForward(0, "TARGET")
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.SearchForward(first.Offset+int64(first.ByteLength), "TARGET")
	if err != nil {
		t.Fatal(err)
	}
	if second.Offset <= first.Offset {
		t.Fatalf("expected second hit after first: first=%#v second=%#v", first, second)
	}
	if second.Offset != int64(len("one TARGET two ")) {
		t.Fatalf("unexpected second offset: %d", second.Offset)
	}
}

func TestLocateHitUTF8Chinese(t *testing.T) {
	path := writeTempFile(t, "第一行\n前缀关键字后缀\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, PageSize: 2, SearchChunk: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	result, err := r.SearchForward(0, "关键字")
	if err != nil {
		t.Fatal(err)
	}
	page, err := r.ReadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	location, err := r.LocateHit(page, result.Offset, result.ByteLength)
	if err != nil {
		t.Fatal(err)
	}
	if location.LineIndex != 1 || location.LineCharStart != 2 || location.LineCharEnd != 5 {
		t.Fatalf("unexpected location: %#v", location)
	}
}

func TestSearchForwardGBKChinese(t *testing.T) {
	encoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte("开始\n前缀关键字后缀\n"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sample-gbk.txt")
	if err := os.WriteFile(path, encoded, 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path, Config{Encoding: EncodingGBK, PageSize: 2, SearchChunk: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	result, err := r.SearchForward(0, "关键字")
	if err != nil {
		t.Fatal(err)
	}
	if result.ByteLength != 6 {
		t.Fatalf("unexpected GBK byte length: %d", result.ByteLength)
	}
	page, err := r.ReadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	location, err := r.LocateHit(page, result.Offset, result.ByteLength)
	if err != nil {
		t.Fatal(err)
	}
	if location.LineIndex != 1 || location.LineCharStart != 2 || location.LineCharEnd != 5 {
		t.Fatalf("unexpected GBK location: %#v", location)
	}
}

func TestSearchAllCountsAndLimitsHits(t *testing.T) {
	path := writeTempFile(t, "hit one\nhit two\nnope\nhit three\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 7})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	summary, err := r.SearchAll("hit", 2)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 3 || len(summary.Hits) != 2 || !summary.Truncated {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.Hits[0].LineNumber != 1 || summary.Hits[1].LineNumber != 2 {
		t.Fatalf("unexpected line numbers: %#v", summary.Hits)
	}
	if summary.Hits[1].LinePreview != "hit two" {
		t.Fatalf("unexpected preview: %q", summary.Hits[1].LinePreview)
	}
}

func TestSearchAllNotFound(t *testing.T) {
	path := writeTempFile(t, "abcdef")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	summary, err := r.SearchAll("xyz", 10)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 0 || len(summary.Hits) != 0 || summary.Truncated {
		t.Fatalf("unexpected not found summary: %#v", summary)
	}
}

func TestSearchAllUTF8ChinesePreview(t *testing.T) {
	path := writeTempFile(t, "第一行\n前缀关键字后缀\n末尾关键字\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	summary, err := r.SearchAll("关键字", 10)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 2 || len(summary.Hits) != 2 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	first := summary.Hits[0]
	if first.LineNumber != 2 || first.LinePreview != "前缀关键字后缀" || first.LineCharStart != 2 || first.LineCharEnd != 5 {
		t.Fatalf("unexpected first hit: %#v", first)
	}
}

func TestBuildSearchIndexCountsAllHits(t *testing.T) {
	path := writeTempFile(t, "hit one\nhit two\nnope\nhit three\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 7})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	refs, err := r.BuildSearchIndex("hit")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 3 {
		t.Fatalf("unexpected refs: %#v", refs)
	}
	for i, ref := range refs {
		if ref.Index != i || ref.ByteLength != len("hit") {
			t.Fatalf("unexpected ref at %d: %#v", i, ref)
		}
	}
}

func TestBuildSearchIndexAcrossChunks(t *testing.T) {
	path := writeTempFile(t, "aaaaTARGETbbbbTARGET")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 6})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	refs, err := r.BuildSearchIndex("TARGET")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 || refs[0].Offset != 4 || refs[1].Offset != int64(len("aaaaTARGETbbbb")) {
		t.Fatalf("unexpected refs: %#v", refs)
	}
}

func TestBuildSearchIndexIncludesLineMetadata(t *testing.T) {
	path := writeTempFile(t, "first line\nsecond TARGET line\nthird line")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 7})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	refs, err := r.BuildSearchIndex("TARGET")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Fatalf("unexpected refs: %#v", refs)
	}
	ref := refs[0]
	if ref.LineStart != int64(len("first line\n")) || ref.LineEnd != int64(len("first line\nsecond TARGET line")) || ref.LineNumber != 2 {
		t.Fatalf("unexpected line metadata: %#v", ref)
	}
}

func TestBuildSearchHitPreviewFromRefFindsHitBeyondMaxLineBytes(t *testing.T) {
	longPrefix := strings.Repeat("a", 80)
	path := writeTempFile(t, longPrefix+"TARGET"+strings.Repeat("b", 20)+"\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 17, MaxLineBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	refs, err := r.BuildSearchIndex("TARGET")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Fatalf("unexpected refs: %#v", refs)
	}
	hit, err := r.BuildSearchHitPreviewFromRef(refs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(hit.LinePreview, "TARGET") || hit.LineCharStart <= 0 || hit.LineCharEnd <= hit.LineCharStart {
		t.Fatalf("unexpected long-line preview: %#v", hit)
	}
}

func TestBuildSearchHitPreviewsWindow(t *testing.T) {
	path := writeTempFile(t, "hit one\nhit two\nhit three\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	refs, err := r.BuildSearchIndex("hit")
	if err != nil {
		t.Fatal(err)
	}
	hits, err := r.BuildSearchHitPreviews(refs, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Index != 1 || hits[0].LinePreview != "hit two" || hits[0].LineNumber != 2 {
		t.Fatalf("unexpected previews: %#v", hits)
	}
}

func TestBuildSearchHitPreviewsFromRefsPreservesIndexes(t *testing.T) {
	path := writeTempFile(t, "hit one\nhit two\nhit three\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	refs, err := r.BuildSearchIndex("hit")
	if err != nil {
		t.Fatal(err)
	}
	hits, err := r.BuildSearchHitPreviewsFromRefs(refs[1:3])
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 || hits[0].Index != 1 || hits[1].Index != 2 {
		t.Fatalf("unexpected preserved indexes: %#v", hits)
	}
	if hits[0].LinePreview != "hit two" || hits[1].LinePreview != "hit three" {
		t.Fatalf("unexpected previews: %#v", hits)
	}
}

func TestSearchAllGBKChinesePreview(t *testing.T) {
	encoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte("开始\n前缀关键字后缀\n"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sample-gbk.txt")
	if err := os.WriteFile(path, encoded, 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path, Config{Encoding: EncodingGBK, SearchChunk: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	summary, err := r.SearchAll("关键字", 10)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 1 || len(summary.Hits) != 1 {
		t.Fatalf("unexpected GBK summary: %#v", summary)
	}
	hit := summary.Hits[0]
	if hit.ByteLength != 6 || hit.LinePreview != "前缀关键字后缀" || hit.LineCharStart != 2 || hit.LineCharEnd != 5 {
		t.Fatalf("unexpected GBK hit: %#v", hit)
	}
}

func TestBuildSearchIndexWithCaseOptions(t *testing.T) {
	path := writeTempFile(t, "Error\nerror\nERROR\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	sensitive, err := r.BuildSearchIndexWithOptions("error", SearchOptions{CaseSensitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(sensitive) != 1 || sensitive[0].LineNumber != 2 {
		t.Fatalf("unexpected case-sensitive refs: %#v", sensitive)
	}

	folded, err := r.BuildSearchIndexWithOptions("error", SearchOptions{CaseSensitive: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(folded) != 3 {
		t.Fatalf("unexpected case-insensitive refs: %#v", folded)
	}
}

func TestBuildSearchIndexWithRegex(t *testing.T) {
	path := writeTempFile(t, "error 100\nwarn 20\nERROR 300\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	refs, err := r.BuildSearchIndexWithOptions(`error\s+\d+`, SearchOptions{Regex: true, CaseSensitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].LineNumber != 1 || refs[0].ByteLength != len("error 100") {
		t.Fatalf("unexpected regex refs: %#v", refs)
	}

	refs, err = r.BuildSearchIndexWithOptions(`error\s+\d+`, SearchOptions{Regex: true, CaseSensitive: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 || refs[1].LineNumber != 3 {
		t.Fatalf("unexpected folded regex refs: %#v", refs)
	}
}

func TestBuildSearchIndexWithRegexErrors(t *testing.T) {
	path := writeTempFile(t, "abc\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if _, err := r.BuildSearchIndexWithOptions("(", SearchOptions{Regex: true, CaseSensitive: true}); err == nil {
		t.Fatal("expected invalid regex error")
	}
	_, err = r.BuildSearchIndexWithOptions(".*", SearchOptions{Regex: true, CaseSensitive: true})
	if !errors.Is(err, ErrEmptyRegexMatch) {
		t.Fatalf("expected ErrEmptyRegexMatch, got %v", err)
	}
}

func TestBuildSearchIndexGB18030ByteOffsets(t *testing.T) {
	encoded, _, err := transform.Bytes(simplifiedchinese.GB18030.NewEncoder(), []byte("开始\n前缀关键字后缀\n"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sample-gb18030.txt")
	if err := os.WriteFile(path, encoded, 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path, Config{Encoding: EncodingGB18030})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	refs, err := r.BuildSearchIndexWithOptions("关键字", SearchOptions{CaseSensitive: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Fatalf("unexpected refs: %#v", refs)
	}
	if refs[0].ByteLength != 6 || refs[0].LineNumber != 2 {
		t.Fatalf("unexpected GB18030 ref: %#v", refs[0])
	}
}

func TestStreamSearchWithOptionsReportsProgress(t *testing.T) {
	content := "hit one\nmiss\nhit two\n"
	path := writeTempFile(t, content)
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	var refs []SearchHitRef
	var progress []int64
	err = r.StreamSearchWithOptions(context.Background(), "hit", SearchOptions{CaseSensitive: true}, func(ref SearchHitRef) error {
		refs = append(refs, ref)
		return nil
	}, func(scannedOffset int64) error {
		progress = append(progress, scannedOffset)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 || refs[0].LineNumber != 1 || refs[1].LineNumber != 3 {
		t.Fatalf("unexpected refs: %#v", refs)
	}
	if len(progress) == 0 || progress[len(progress)-1] != int64(len(content)) {
		t.Fatalf("unexpected progress: %#v", progress)
	}
	for i := 1; i < len(progress); i++ {
		if progress[i] < progress[i-1] {
			t.Fatalf("progress is not monotonic: %#v", progress)
		}
	}
}

func TestStreamSearchWithOptionsStopsOnCancel(t *testing.T) {
	path := writeTempFile(t, "hit\nhit\nhit\n")
	r, err := Open(path, Config{Encoding: EncodingUTF8, SearchChunk: 12})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var refs []SearchHitRef
	err = r.StreamSearchWithOptions(ctx, "hit", SearchOptions{CaseSensitive: true}, func(ref SearchHitRef) error {
		refs = append(refs, ref)
		cancel()
		return nil
	}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected one ref before cancel, got %#v", refs)
	}
}
