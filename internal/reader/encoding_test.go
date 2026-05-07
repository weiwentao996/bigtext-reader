package reader

import (
	"testing"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
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

func TestAdditionalEncodings(t *testing.T) {
	tests := []struct {
		name         string
		encodingName string
		text         string
		encoder      transform.Transformer
	}{
		{name: "gb18030", encodingName: EncodingGB18030, text: "中文𠀀", encoder: simplifiedchinese.GB18030.NewEncoder()},
		{name: "big5", encodingName: EncodingBig5, text: "繁體中文", encoder: traditionalchinese.Big5.NewEncoder()},
		{name: "shift_jis", encodingName: EncodingShiftJIS, text: "日本語", encoder: japanese.ShiftJIS.NewEncoder()},
		{name: "euc_kr", encodingName: EncodingEUCKR, text: "한국어", encoder: korean.EUCKR.NewEncoder()},
		{name: "windows1252", encodingName: EncodingWindows1252, text: "café", encoder: charmap.Windows1252.NewEncoder()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, _, err := transform.Bytes(tt.encoder, []byte(tt.text))
			if err != nil {
				t.Fatal(err)
			}
			decoder, err := NewDecoder(tt.encodingName)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := decoder.DecodeLine(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if decoded != tt.text {
				t.Fatalf("unexpected decoded text: %q", decoded)
			}
			keyword, err := EncodeKeyword(tt.text, tt.encodingName)
			if err != nil {
				t.Fatal(err)
			}
			if string(keyword) != string(encoded) {
				t.Fatalf("unexpected encoded keyword: %v", keyword)
			}
		})
	}
}
