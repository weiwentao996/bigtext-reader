package reader

import (
	"io"
	"os"
	"path/filepath"
)

type Reader struct {
	path    string
	file    *os.File
	meta    FileMeta
	config  Config
	decoder Decoder
}

func Open(path string, config Config) (*Reader, error) {
	config = NormalizeConfig(config)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	decoder, err := NewDecoder(config.Encoding)
	if err != nil {
		file.Close()
		return nil, err
	}

	encoding := decoder.Encoding()
	if encoding == EncodingAuto {
		sampleSize := int64(64 * 1024)
		if stat.Size() < sampleSize {
			sampleSize = stat.Size()
		}
		sample := make([]byte, sampleSize)
		n, err := file.ReadAt(sample, 0)
		if err != nil && err != io.EOF {
			file.Close()
			return nil, err
		}
		encoding = DetectEncoding(sample[:n])
		decoder = decoder.WithEncoding(encoding)
	}

	return &Reader{
		path:    path,
		file:    file,
		config:  config,
		decoder: decoder,
		meta: FileMeta{
			Path:     path,
			Name:     filepath.Base(path),
			Size:     stat.Size(),
			ModTime:  stat.ModTime(),
			Encoding: encoding,
		},
	}, nil
}

func (r *Reader) Close() error {
	if r == nil || r.file == nil {
		return nil
	}
	return r.file.Close()
}

func (r *Reader) Meta() FileMeta {
	return r.meta
}

func (r *Reader) Config() Config {
	return r.config
}

func (r *Reader) readAt(offset int64, size int) ([]byte, error) {
	if offset < 0 {
		offset = 0
	}
	if offset >= r.meta.Size {
		return nil, io.EOF
	}
	remaining := r.meta.Size - offset
	if int64(size) > remaining {
		size = int(remaining)
	}
	buf := make([]byte, size)
	n, err := r.file.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:n], err
}
