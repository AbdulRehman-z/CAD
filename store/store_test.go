package store

import (
	"bytes"
	"io"
	"log"
	"testing"

	"github.com/1500-bytes/CAD/crypto"
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
	id := crypto.NewID()
	if _, err := s.writeSream(id, "dir", bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	_, r, err := s.Read(id, "dir")
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

	if !s.Has(id, "dir") {
		t.Error("dir should exist")
	}

	if err := s.Delete(id, "dir"); err != nil {
		t.Error(err)
	}

	if s.Has(id, "dir") {
		t.Error("dir should not exist")
	}

}

func TestCasPathTransformFunc(t *testing.T) {
	path := "my/path"
	transformed := CasPathTransformFunc(path)
	log.Println(transformed)
	assert.IsType(t, "", transformed.Filename)
}
