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
	if pathKey.Pathname != expectedPathname {
		t.Errorf("have %s, expected %s", pathKey.Pathname, expectedPathname)
	}
	if pathKey.Filename != expectedOriginalKey {
		t.Errorf("have %s, expected %s", pathKey.Filename, expectedOriginalKey)
	}
}

func TestStorage(t *testing.T) {
	s := newStorage()
	defer teardown(t, s)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("foo_%d", i)
		data := []byte("some jpg bytes")

		if _, err := s.writeStream(key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); !ok {
			t.Errorf("expected to have have %s", key)
		}

		_, r, err := s.Read(key)
		if err != nil {
			t.Error(err)
		}

		b, _ := io.ReadAll(r)
		if string(b) != string(data) {
			t.Errorf("expected %s have %s", data, b)
		}

		if err := s.Delete(key); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); ok {
			t.Errorf("expected to NOT have key %s", key)
		}
	}
}

func newStorage() *Storage {
	opts := StorageOptions{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStorage(opts)
}

func teardown(t *testing.T, s *Storage) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}
