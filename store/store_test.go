package store

import (
	"bytes"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CasPathTransformFunc,
	}

	s := NewStore(opts)
	return s
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}

func TestStore(t *testing.T) {
	s := newStore()
	defer teardown(t, s)

	data := []byte("hello there")
	if err := s.writeSream("dir", bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	r, err := s.Read("dir")
	if err != nil {
		t.Error(err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}

	if string(b) != string(data) {
		t.Error("data mismatch")
	}

	if !s.Has("dir") {
		t.Error("dir should exist")
	}

	if err := s.Delete("dir"); err != nil {
		t.Error(err)
	}

	if s.Has("dir") {
		t.Error("dir should not exist")
	}

}

func TestCasPathTransformFunc(t *testing.T) {
	path := "my/path"
	transformed := CasPathTransformFunc(path)
	log.Println(transformed)
	assert.IsType(t, "", transformed.Filename)
}
