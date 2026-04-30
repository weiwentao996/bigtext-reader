package reader

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type Decoder struct {
	encoding string
}

func DetectEncoding(sample []byte) string {
	if utf8.Valid(sample) {
		return EncodingUTF8
	}
	return EncodingGBK
}

func NewDecoder(encoding string) (Decoder, error) {
	encoding = strings.ToLower(strings.TrimSpace(encoding))
	switch encoding {
	case "", EncodingAuto:
		return Decoder{encoding: EncodingAuto}, nil
	case EncodingUTF8, "utf-8":
		return Decoder{encoding: EncodingUTF8}, nil
	case EncodingGBK, "gb2312":
		return Decoder{encoding: EncodingGBK}, nil
	default:
		return Decoder{}, fmt.Errorf("unsupported encoding: %s", encoding)
	}
}

func (d Decoder) Encoding() string {
	return d.encoding
}

func (d Decoder) WithEncoding(encoding string) Decoder {
	return Decoder{encoding: encoding}
}

func (d Decoder) DecodeLine(data []byte) (string, error) {
	data = bytes.TrimSuffix(data, []byte{'\r'})
	if len(data) == 0 {
		return "", nil
	}

	switch d.encoding {
	case EncodingUTF8, EncodingAuto:
		return string(data), nil
	case EncodingGBK:
		out, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data)
		if err != nil {
			return "", err
		}
		return string(out), nil
	default:
		return "", fmt.Errorf("unsupported encoding: %s", d.encoding)
	}
}

func EncodeKeyword(keyword string, encoding string) ([]byte, error) {
	switch encoding {
	case EncodingUTF8, EncodingAuto, "":
		return []byte(keyword), nil
	case EncodingGBK:
		out, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte(keyword))
		if err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encoding)
	}
}
