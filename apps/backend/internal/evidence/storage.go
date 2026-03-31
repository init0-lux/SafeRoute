package evidence

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type Storage interface {
	Put(ctx context.Context, key string, body io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

type LocalStorage struct {
	root string
}

func NewLocalStorage(root string) *LocalStorage {
	return &LocalStorage{root: root}
}

func (s *LocalStorage) Put(_ context.Context, key string, body io.Reader) error {
	fullPath := filepath.Join(s.root, key)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, body)
	return err
}

func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.root, key))
}
