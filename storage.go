package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

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
		Pathanme: strings.Join(paths, "/"),
		Filename: hashedStr,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	Pathanme string
	Filename string
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.Pathanme, p.Filename)
}

var DefaultPathTransformFunc = func(key string) string { return key }

type StorageOptions struct {
	PathTransformFunc PathTransformFunc
}

type Storage struct {
	StorageOptions
}

func NewStorage(options StorageOptions) *Storage {
	return &Storage{
		StorageOptions: options,
	}
}

func (store *Storage) Read(key string) (io.Reader, error) {
	file, err := store.readStream(key)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, file)

	return buf, err
}

func (store *Storage) readStream(key string) (io.ReadCloser, error) {
	pathKey := store.PathTransformFunc(key)
	return os.Open(pathKey.FullPath())
}


func (store *Storage) writeStream(key string, r io.Reader) error {
	pathKey := store.PathTransformFunc(key)

	if err := os.MkdirAll(pathKey.Pathanme, os.ModePerm); err != nil {
		return err
	}

	fullPath := pathKey.FullPath()

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}

	n, err := io.Copy(file, r)
	if err != nil {
		return err
	}

	log.Printf("written (%d) bytes to disk: %s", n, fullPath)

	return nil
}