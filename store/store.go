package store

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var defaultRootPathname = "store"

type PathTransformFunc func(string) Path

type Path struct {
	Pathname string
	Filename string
}

func (s Path) FullPath() string {
	return fmt.Sprintf("%s/%s", s.Pathname, s.Filename)
}

func (s Path) FirstPathname() string {
	paths := strings.Split(s.Pathname, "/")
	if len(paths) == 0 {
		return ""
	}

	return paths[0]
}

var DefaultPathTransformFunc = func(path string) Path {
	return Path{
		Pathname: path,
		Filename: path,
	}
}

func CasPathTransformFunc(path string) Path {
	hash := sha1.Sum([]byte(path))     // 20 bytes
	hashStr := fmt.Sprintf("%x", hash) // 40 bytes

	blocksize := 5
	sliceLen := len(hashStr) / blocksize // 40 / 5 = 8
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := (i * blocksize), (i*blocksize)+blocksize // 0, 5
		paths[i] = hashStr[from:to]                          // 0:5, 5:10, 10:15, 15:20, 20:25, 25:30, 30:35, 35:40
	}

	return Path{
		Pathname: strings.Join(paths, "/"),
		Filename: hashStr,
	}

}

type StoreOpts struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootPathname
	}

	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
	_, err := os.Stat(fullPathWithRoot)
	return err == nil
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	firstPathnameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FirstPathname())

	return os.RemoveAll(firstPathnameWithRoot)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Read(key string) (int64, io.Reader, error) {
	return s.readStream(key)
}

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeSream(key, r)
}

func (s *Store) readStream(key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, fmt.Errorf("err opening file: %s", err.Error())
	}

	info, err := file.Stat()
	if err != nil {
		return 0, nil, fmt.Errorf("err getting file info: %s", err.Error())
	}

	return info.Size(), file, nil
}

func (s *Store) writeSream(key string, r io.Reader) (int64, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.Pathname)
	// create a directory
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return 0, fmt.Errorf("err in write stream: %s", err)
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
	// create a file
	f, err := os.Create(fullPathWithRoot)
	if err != nil {
		return 0, fmt.Errorf("err in creating file: %s", err)
	}

	defer func() {
		log.Println("write finished to disc")
		f.Close()
	}()

	n, err := io.Copy(f, r)
	if err != nil {
		return 0, fmt.Errorf("err in copying number of bytes: %s", err)
	}

	return n, nil
}
