package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileTree struct {
	Dir   string
	Files []*FileDsc
}

type FileDsc struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	Level   int
	ModTime time.Time
	Hash    string
}

func Tree(dir string) (*FileTree, error) {
	tree := &FileTree{Dir: dir}
	return tree, tree.calculate("", 0)
}

func (t *FileTree) calculate(path string, level int) error {
	entries, err := os.ReadDir(filepath.Join(t.Dir, path))
	if err != nil {
		return err
	}
	for _, file := range entries {
		dsc, err := NewFileDsc(filepath.Join(t.Dir, path, file.Name()), level)
		if err != nil {
			return err
		}
		t.Files = append(t.Files, dsc)
		if file.IsDir() {
			err := t.calculate(filepath.Join(path, file.Name()), level+1)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func NewFileDsc(path string, level int) (*FileDsc, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	dsc := &FileDsc{
		Name:    stat.Name(),
		Path:    path,
		IsDir:   stat.IsDir(),
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		Level:   level,
	}

	if !stat.IsDir() {
		dsc.Hash, err = fastHash(path, stat.Size())
		if err != nil {
			return nil, err
		}
	}
	return dsc, nil
}

func fastHash(path string, size int64) (string, error) {
	hash := sha256.New()
	const minSize = 1024 * 32 // 32KB
	const chunkSize = 1024 * 4
	if size <= minSize {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		hash.Write(b)
		return hex.EncodeToString(hash.Sum(nil)), nil
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	b := make([]byte, chunkSize)
	for _, off := range []int64{0, size / 2, size - chunkSize} {
		_, err = f.ReadAt(b, off)
		if err != nil {
			return "", err
		}
		hash.Write(b)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (t *FileTree) RemovePrefix() {
	for _, dsc := range t.Files {
		dsc.Path = strings.TrimPrefix(dsc.Path, t.Dir)
		dsc.Path = strings.TrimPrefix(dsc.Path, "/")
	}
}

func (t *FileTree) String() string {
	sb := strings.Builder{}
	for _, dsc := range t.Files {
		if dsc.IsDir {
			sb.WriteString(fmt.Sprintf("%s+ %s %s\n", strings.Repeat("  ", dsc.Level), dsc.Name, dsc.ModTime.Format(time.RFC3339)))
		} else {
			sb.WriteString(fmt.Sprintf("%s└ %s %d %s %s\n", strings.Repeat("  ", dsc.Level), dsc.Name, dsc.Size, dsc.ModTime.Format(time.RFC3339), dsc.Hash))
		}
	}

	return sb.String()
}
