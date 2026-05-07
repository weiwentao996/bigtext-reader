package reader

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

type Decoder struct {
	encoding string
}

func DetectEncoding(sample []byte) string {
	if utf8.Valid(sample) {
		return EncodingUTF8
	}
	return EncodingGB18030
}

func NewDecoder(encoding string) (Decoder, error) {
	encoding = normalizeEncodingName(encoding)
	if encoding == "" {
		encoding = EncodingAuto
	}
	if isSupportedEncoding(encoding) {
		return Decoder{encoding: encoding}, nil
	}
	return Decoder{}, fmt.Errorf("unsupported encoding: %s", encoding)
}

func (d Decoder) Encoding() string {
	return d.encoding
}

func (d Decoder) WithEncoding(encoding string) Decoder {
	return Decoder{encoding: normalizeEncodingName(encoding)}
}

func (d Decoder) DecodeLine(data []byte) (string, error) {
	data = bytes.TrimSuffix(data, []byte{'\r'})
	if len(data) == 0 {
		return "", nil
	}
	return decodeBytes(data, d.encoding)
}

func EncodeKeyword(keyword string, encodingName string) ([]byte, error) {
	encodingName = normalizeEncodingName(encodingName)
	switch encodingName {
	case "", EncodingUTF8, EncodingAuto:
		return []byte(keyword), nil
	}
	enc, ok := textEncoding(encodingName)
	if !ok {
		return nil, fmt.Errorf("unsupported encoding: %s", encodingName)
	}
	out, _, err := transform.Bytes(enc.NewEncoder(), []byte(keyword))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func decodeBytes(data []byte, encodingName string) (string, error) {
	encodingName = normalizeEncodingName(encodingName)
	switch encodingName {
	case "", EncodingUTF8, EncodingAuto:
		return string(data), nil
	}
	enc, ok := textEncoding(encodingName)
	if !ok {
		return "", fmt.Errorf("unsupported encoding: %s", encodingName)
	}
	out, _, err := transform.Bytes(enc.NewDecoder(), data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func normalizeEncodingName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "-", "_")
	switch name {
	case "utf_8":
		return EncodingUTF8
	case "gb2312":
		return EncodingGBK
	case "gb_18030":
		return EncodingGB18030
	case "big_5":
		return EncodingBig5
	case "shift_jis", "shiftjis", "sjis":
		return EncodingShiftJIS
	case "euc_kr", "euckr":
		return EncodingEUCKR
	case "windows_1252", "cp1252":
		return EncodingWindows1252
	default:
		return name
	}
}

func isSupportedEncoding(name string) bool {
	if name == EncodingAuto || name == EncodingUTF8 {
		return true
	}
	_, ok := textEncoding(name)
	return ok
}

func textEncoding(name string) (encoding.Encoding, bool) {
	switch name {
	case EncodingGBK:
		return simplifiedchinese.GBK, true
	case EncodingGB18030:
		return simplifiedchinese.GB18030, true
	case EncodingBig5:
		return traditionalchinese.Big5, true
	case EncodingShiftJIS:
		return japanese.ShiftJIS, true
	case EncodingEUCKR:
		return korean.EUCKR, true
	case EncodingWindows1252:
		return charmap.Windows1252, true
	default:
		return nil, false
	}
}
