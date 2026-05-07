package main

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"

	"bigtext-reader/internal/reader"
)

const searchHitRecordSize = 48

var errSearchHitStoreClosed = errors.New("search hit store is closed")

type searchHitStore struct {
	mu     sync.Mutex
	file   *os.File
	path   string
	count  int
	closed bool
}

func newSearchHitStore() (*searchHitStore, error) {
	file, err := os.CreateTemp("", "bigtext-reader-search-*.idx")
	if err != nil {
		return nil, err
	}
	return &searchHitStore{file: file, path: file.Name()}, nil
}

func (s *searchHitStore) Append(ref reader.SearchHitRef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errSearchHitStoreClosed
	}
	var record [searchHitRecordSize]byte
	binary.LittleEndian.PutUint64(record[0:8], uint64(ref.Index))
	binary.LittleEndian.PutUint64(record[8:16], uint64(ref.Offset))
	binary.LittleEndian.PutUint64(record[16:24], uint64(ref.ByteLength))
	binary.LittleEndian.PutUint64(record[24:32], uint64(ref.LineStart))
	binary.LittleEndian.PutUint64(record[32:40], uint64(ref.LineEnd))
	binary.LittleEndian.PutUint64(record[40:48], uint64(ref.LineNumber))
	if _, err := s.file.WriteAt(record[:], int64(s.count*searchHitRecordSize)); err != nil {
		return err
	}
	s.count++
	return nil
}

func (s *searchHitStore) Get(index int) (reader.SearchHitRef, error) {
	refs, err := s.Window(index, 1)
	if err != nil {
		return reader.SearchHitRef{}, err
	}
	if len(refs) == 0 {
		return reader.SearchHitRef{}, io.EOF
	}
	return refs[0], nil
}

func (s *searchHitStore) Window(offset int, limit int) ([]reader.SearchHitRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errSearchHitStoreClosed
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || offset >= s.count {
		return []reader.SearchHitRef{}, nil
	}
	end := offset + limit
	if end > s.count {
		end = s.count
	}
	refs := make([]reader.SearchHitRef, 0, end-offset)
	var record [searchHitRecordSize]byte
	for index := offset; index < end; index++ {
		if _, err := s.file.ReadAt(record[:], int64(index*searchHitRecordSize)); err != nil {
			return nil, err
		}
		refs = append(refs, reader.SearchHitRef{
			Index:      int(binary.LittleEndian.Uint64(record[0:8])),
			Offset:     int64(binary.LittleEndian.Uint64(record[8:16])),
			ByteLength: int(binary.LittleEndian.Uint64(record[16:24])),
			LineStart:  int64(binary.LittleEndian.Uint64(record[24:32])),
			LineEnd:    int64(binary.LittleEndian.Uint64(record[32:40])),
			LineNumber: int64(binary.LittleEndian.Uint64(record[40:48])),
		})
	}
	return refs, nil
}

func (s *searchHitStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

func (s *searchHitStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.file.Close()
}

func (s *searchHitStore) Remove() error {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		_ = s.file.Close()
	}
	path := s.path
	s.mu.Unlock()
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
