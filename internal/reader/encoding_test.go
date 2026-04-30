package reader

import (
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func TestDecodeGBKLine(t *testing.T) {
	gbk, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte("中文"))
	if err != nil {
		t.Fatal(err)
	}
	decoder, err := NewDecoder(EncodingGBK)
	if err != nil {
		t.Fatal(err)
	}
	line, err := decoder.DecodeLine(gbk)
	if err != nil {
		t.Fatal(err)
	}
	if line != "中文" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestEncodeKeywordGBK(t *testing.T) {
	keyword, err := EncodeKeyword("中文", EncodingGBK)
	if err != nil {
		t.Fatal(err)
	}
	if len(keyword) != 4 {
		t.Fatalf("unexpected gbk keyword length: %d", len(keyword))
	}
}
