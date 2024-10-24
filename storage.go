package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "supernetwork"

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashedStr := hex.EncodeToString(hash[:])

	blocksize := 5
	slicelen := len(hashedStr) / blocksize

	paths := make([]string, slicelen)
	for i := range slicelen {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashedStr[from:to]
	}

	return PathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: hashedStr,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	Pathname string
	Filename string
}

func (p PathKey) FirstPathname() string {
	paths := strings.Split(p.Pathname, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.Pathname, p.Filename)
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		Pathname: key,
		Filename: key,
	}
}

type StorageOptions struct {
	/*
		Root is the folder name of the root,
		containing all the folders / files of the system
	*/
	Root              string
	PathTransformFunc PathTransformFunc
}

type Storage struct {
	StorageOptions
}

func NewStorage(options StorageOptions) *Storage {
	if options.PathTransformFunc == nil {
		options.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(options.Root) == 0 {
		options.Root = defaultRootFolderName
	}

	return &Storage{
		StorageOptions: options,
	}
}

func (store *Storage) Has(id string, key string) bool {
	pathKey := store.PathTransformFunc(key)

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", store.Root, id, pathKey.FullPath())
	_, err := os.Stat(fullPathWithRoot)

	return !errors.Is(err, os.ErrNotExist)
}

func (s *Storage) Clear() error {
	return os.RemoveAll(s.Root)
}

func (store *Storage) Delete(id string, key string) error {
	pathKey := store.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.Filename)
	}()

	firstPathnameWithRoot := fmt.Sprintf("%s/%s/%s", store.Root, id, pathKey.FirstPathname())

	return os.RemoveAll(firstPathnameWithRoot)
}

func (store *Storage) Write(id string, key string, r io.Reader) (int64, error) {
	return store.writeStream(id, key, r)
}

func (store *Storage) writeDecrypt(encKye []byte, id string, key string, r io.Reader) (int64, error) {
	file, err := store.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	n, err := copyDecrypt(encKye, r, file)
	return int64(n), err
}

func (store *Storage) writeStream(id string, key string, r io.Reader) (int64, error) {
	file, err := store.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	return io.Copy(file, r)
}

func (store *Storage) openFileForWriting(id string, key string) (*os.File, error) {
	pathKey := store.PathTransformFunc(key)

	pathnameWithRoot := fmt.Sprintf("%s/%s/%s", store.Root, id, pathKey.Pathname)
	if err := os.MkdirAll(pathnameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", store.Root, id, pathKey.FullPath())

	return os.Create(fullPathWithRoot)
}

func (store *Storage) Read(id string, key string) (int64, io.Reader, error) {
	return store.readStream(id, key)
}

func (store *Storage) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := store.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", store.Root, id, pathKey.FullPath())

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}
