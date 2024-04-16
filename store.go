package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type PathTransformFunc func(string) string

var DefaultPathTransformFunc = func(path string) string {
	return path
}

func CasPathTransformFunc(path string) string {
	hash := sha1.Sum([]byte(path))     // 20 bytes
	hashStr := fmt.Sprintf("%x", hash) // 40 bytes

	blocksize := 5
	sliceLen := len(hashStr) / blocksize // 40 / 5 = 8
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := (i * blocksize), (i*blocksize)+blocksize // 0, 5
		paths[i] = hashStr[from:to]                          // 0:5, 5:10, 10:15, 15:20, 20:25, 25:30, 30:35, 35:40
	}

	return strings.Join(paths, "/")
}

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) writeSream(key string, r io.Reader) error {
	pathName := s.PathTransformFunc(key)
	// create a directory
	if err := os.MkdirAll(pathName, os.ModePerm); err != nil {
		return fmt.Errorf("err in write stream: %s", err)
	}
	buf := new(bytes.Buffer) // 0 bytes
	io.Copy(buf, r)          // copy the reader to the buffer

	filenameBytes := md5.Sum(buf.Bytes())        // 16 bytes
	filename := fmt.Sprintf("%x", filenameBytes) // 32 bytes
	fileAndPathName := pathName + "/" + filename

	// create a file
	f, err := os.Create(fileAndPathName)
	if err != nil {
		return fmt.Errorf("err in creating file: %s", err)
	}

	n, err := io.Copy(f, buf)
	if err != nil {
		return fmt.Errorf("err in copying number of bytes: %s", err)
	}

	log.Println("Copied bytes: ", n)
	return nil
}
