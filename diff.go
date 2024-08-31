package main

import (
	"cmp"
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"slices"
	"strings"
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
		dir1ByHash[dsc1.Hash] = append(dir1ByHash[dsc1.Hash], dsc1)
		dir1ByPath[dsc1.Path] = dsc1
	}

	for _, dsc2 := range t2.Files {
		if dsc2.IsDir {
			continue
		}
		if dsc1, ok := dir1ByPath[dsc2.Path]; ok {
			if dsc1.Hash != dsc2.Hash {
				res.Files = append(res.Files, &FileDiff{Status: "CHANGED", FileDsc: dsc1})
			} else {
				res.Files = append(res.Files, &FileDiff{Status: "UNCHANGED", FileDsc: dsc1})
			}
			matched[dsc1.Hash]++
			continue
		}
		// Let's try to find the file somewhere else:
		if dsc1, ok := dir1ByHash[dsc2.Hash]; ok {
			r := &FileDiff{Status: "MOVED", FileDsc: dsc2}
			r.Path = dsc1[0].Path + " -> " + dsc2.Path
			res.Files = append(res.Files, r)
			matched[dsc2.Hash]++
			continue
		}
		res.Files = append(res.Files, &FileDiff{Status: "ADDED", FileDsc: dsc2})
	}
	for _, dsc1 := range t1.Files {
		if dsc1.IsDir {
			continue
		}
		num := matched[dsc1.Hash]
		if num == 0 {
			res.Files = append(res.Files, &FileDiff{Status: "DELETED", FileDsc: dsc1})
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
		if f.Status == "UNCHANGED" {
			unchanged++
			continue
		}
		sb.WriteString(fmt.Sprintf("%s:\t%s\n", f.Status, f.Path))
	}
	sb.WriteString(fmt.Sprintf("\n%d identical files\n", unchanged))
	return sb.String()
}
