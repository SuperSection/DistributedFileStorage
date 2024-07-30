package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "onepiecepicture"
	pathKey := CASPathTransformFunc(key)
	expectedOriginalKey := "eac313584ec0f3e5a5da458ab909f40bc763df5c"
	expectedPathname := "eac31/3584e/c0f3e/5a5da/458ab/909f4/0bc76/3df5c"
	if pathKey.Pathanme != expectedPathname {
		t.Errorf("have %s, expected %s", pathKey.Pathanme, expectedPathname)
	}
	if pathKey.Filename != expectedOriginalKey {
		t.Errorf("have %s, expected %s", pathKey.Filename, expectedOriginalKey)
	}
}

func TestStorage(t *testing.T) {
	options := StorageOptions{
		PathTransformFunc: CASPathTransformFunc,
	}
	storage := NewStorage(options)
	key := "onepiecepicture"
	data := []byte("some jpg bytes")

	if err := storage.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	r, err := storage.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, _ := io.ReadAll(r)

	fmt.Println(string(b))

	if string(b) != string(data) {
		t.Errorf("expected %s have %s", data, b)
	}
}