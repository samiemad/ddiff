package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileTree struct {
	Files   []*FileDsc
	SubDirs []*FileTree
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
	return calculate(dir, 0)
}

func calculate(path string, level int) (*FileTree, error) {
	tree := &FileTree{}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range entries {
		dsc, err := NewFileDsc(filepath.Join(path, file.Name()), level)
		if err != nil {
			return nil, err
		}
		tree.Files = append(tree.Files, dsc)
		if file.IsDir() {
			sd, err := calculate(filepath.Join(path, file.Name()), level+1)
			if err != nil {
				return nil, err
			}
			tree.SubDirs = append(tree.SubDirs, sd)
		}
	}
	return tree, nil
}

func NewFileDsc(path string, level int) (*FileDsc, error) {
	stat, err := os.Lstat(path)
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

	// No need to generate a hash of directories, symlinks or other irregular files.
	mode := stat.Mode()
	if mode.IsRegular() {
		dsc.Hash, err = dashHash(path, stat.Size())
		if err != nil {
			return nil, err
		}
	}
	if mode&os.ModeSymlink != 0 {
		// for a symlink simply use the link directly. No need to hash it since it's also limited in length.
		dsc.Hash, err = os.Readlink(path)
		if err != nil {
			return nil, err
		}
	}
	return dsc, nil
}

func dashHash(path string, size int64) (string, error) {
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

func (t *FileTree) RemovePrefix(dir string) {
	for dsc := range t.AllFiles() {
		dsc.Path = strings.TrimPrefix(dsc.Path, dir)
		dsc.Path = strings.TrimPrefix(dsc.Path, "/")
	}
}

func (t *FileTree) String() string {
	sb := strings.Builder{}
	for dsc := range t.AllFiles() {
		if dsc.IsDir {
			sb.WriteString(fmt.Sprintf("%s+ %s %s\n", strings.Repeat("  ", dsc.Level), dsc.Name, dsc.ModTime.Format(time.RFC3339)))
		} else {
			sb.WriteString(fmt.Sprintf("%s└ %s %d %s %s\n", strings.Repeat("  ", dsc.Level), dsc.Name, dsc.Size, dsc.ModTime.Format(time.RFC3339), dsc.Hash))
		}
	}

	return sb.String()
}

// AllFiles returns an iterator that yields the slice elements in order.
func (t *FileTree) AllFiles() iter.Seq[*FileDsc] {
	return func(yield func(dsc *FileDsc) bool) {
		// Loop over all dirs:
		for _, dir := range t.SubDirs {
			for v := range dir.AllFiles() {
				if !yield(v) {
					return
				}
			}
		}
		for _, v := range t.Files {
			if !yield(v) {
				return
			}
		}
	}
}
