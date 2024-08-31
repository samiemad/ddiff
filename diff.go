package main

import (
	"cmp"
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"slices"
	"strings"
)

const (
	statusUnchanged = "UNCHANGED"
	statusChanged   = "CHANGED"
	statusMoved     = "MOVED"
	statusAdded     = "ADDED"
	statusDeleted   = "DELETED"
)

type TreeDiff struct {
	Files []*FileDiff
}

type FileDiff struct {
	Status string
	*FileDsc
}

func DiffDirs(dir1, dir2 string) (*TreeDiff, error) {
	g, _ := errgroup.WithContext(context.Background())
	var t1, t2 *FileTree
	g.Go(func() error {
		var err error
		t1, err = Tree(dir1)
		return err
	})
	g.Go(func() error {
		var err error
		t2, err = Tree(dir2)
		return err
	})
	err := g.Wait()
	if err != nil {
		return nil, err
	}
	t1.RemovePrefix()
	t2.RemovePrefix()
	return DiffTrees(t1, t2), nil
}

func DiffTrees(t1, t2 *FileTree) *TreeDiff {
	res := &TreeDiff{}
	dir1ByHash := map[string][]*FileDsc{}
	dir1ByPath := map[string]*FileDsc{}
	matched := map[string]int{}
	for _, dsc1 := range t1.Files {
		if dsc1.Hash != "" {
			dir1ByHash[dsc1.Hash] = append(dir1ByHash[dsc1.Hash], dsc1)
		}
		dir1ByPath[dsc1.Path] = dsc1
	}

t2loop:
	for _, dsc2 := range t2.Files {
		if dsc2.IsDir {
			continue
		}
		if dsc1, ok := dir1ByPath[dsc2.Path]; ok {
			if dsc1.Hash != dsc2.Hash || dsc1.Size != dsc2.Size {
				res.Files = append(res.Files, &FileDiff{Status: statusChanged, FileDsc: dsc1})
			} else {
				res.Files = append(res.Files, &FileDiff{Status: statusUnchanged, FileDsc: dsc1})
			}
			matched[dsc1.Hash]++
			continue
		}
		// Let's try to find the file somewhere else:
		for _, dsc1 := range dir1ByHash[dsc2.Hash] {
			// Ignore accidentally equal hashes if the files are of different sizes
			if dsc1.Size != dsc2.Size {
				continue
			}
			r := &FileDiff{Status: statusMoved, FileDsc: dsc2}
			r.Path = dsc1.Path + " -> " + dsc2.Path
			res.Files = append(res.Files, r)
			matched[dsc2.Hash]++
			continue t2loop // Once we found one match we continue with the outer loop.
		}
		res.Files = append(res.Files, &FileDiff{Status: statusAdded, FileDsc: dsc2})
	}
	for _, dsc1 := range t1.Files {
		if dsc1.IsDir {
			continue
		}
		num := matched[dsc1.Hash]
		if num == 0 {
			res.Files = append(res.Files, &FileDiff{Status: statusDeleted, FileDsc: dsc1})
		} else {
			matched[dsc1.Hash]--
		}
	}

	slices.SortStableFunc(res.Files, func(a, b *FileDiff) int {
		return cmp.Compare(a.Path, b.Path)
	})

	return res
}

func (d *TreeDiff) String() string {
	var sb strings.Builder
	unchanged := 0
	for _, f := range d.Files {
		if f.Status == statusUnchanged {
			unchanged++
			continue
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", f.FormatStatus(), f.Path))
	}
	sb.WriteString(fmt.Sprintf("\n%d total, %d identical files\n", len(d.Files), unchanged))
	return sb.String()
}

func (d *FileDiff) FormatStatus() string {
	st := fmt.Sprintf("%7s ", d.Status)
	switch d.Status {
	case statusUnchanged:
		return st
	case statusChanged:
		return yellow(st)
	case statusMoved:
		return blue(st)
	case statusAdded:
		return green(st)
	case statusDeleted:
		return red(st)
	}
	return st
}
