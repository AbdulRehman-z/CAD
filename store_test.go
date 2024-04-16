package main

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CasPathTransformFunc,
	}

	s := NewStore(opts)
	bytes := bytes.NewReader([]byte("hello there"))
	if err := s.writeSream("dir", bytes); err != nil {
		t.Error(err)
	}

}

func TestCasPathTransformFunc(t *testing.T) {
	path := "my/path"
	transformed := CasPathTransformFunc(path)
	log.Println(transformed)
	assert.IsType(t, "", transformed)
}
