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
	statusIdentical = "IDENTICAL"
	statusChanged   = "CHANGED"
	statusMoved     = "MOVED"
	statusAdded     = "ADDED"
	statusDeleted   = "DELETED"
)

type TreeDiff struct {
	Files []*FileDiff
}

type FileDiff struct {
	Status      string
	left, right *FileDsc
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
	t1.RemovePrefix(dir1)
	t2.RemovePrefix(dir2)
	return DiffTrees(t1, t2), nil
}

func DiffTrees(t1, t2 *FileTree) *TreeDiff {
	res := &TreeDiff{}
	dir1ByHash := map[string][]*FileDsc{}
	dir1ByPath := map[string]*FileDsc{}
	matched := map[string]int{}
	for dsc1 := range t1.AllFiles() {
		if dsc1.Hash != "" {
			dir1ByHash[dsc1.Hash] = append(dir1ByHash[dsc1.Hash], dsc1)
		}
		dir1ByPath[dsc1.Path] = dsc1
	}

t2loop:
	for dsc2 := range t2.AllFiles() {
		if dsc2.IsDir {
			continue
		}
		if dsc1, ok := dir1ByPath[dsc2.Path]; ok {
			d := &FileDiff{Status: statusIdentical, left: dsc1, right: dsc2}
			if dsc1.Hash != dsc2.Hash || dsc1.Size != dsc2.Size {
				d.Status = statusChanged
			}
			res.Files = append(res.Files, d)
			matched[dsc1.Hash]++
			continue
		}
		// Let's try to find the file somewhere else:
		for _, dsc1 := range dir1ByHash[dsc2.Hash] {
			// Ignore accidentally equal hashes if the files are of different sizes
			if dsc1.Size != dsc2.Size {
				continue
			}
			r := &FileDiff{Status: statusMoved, left: dsc1, right: dsc2}
			res.Files = append(res.Files, r)
			matched[dsc2.Hash]++
			continue t2loop // Once we found one match we continue with the outer loop.
		}
		res.Files = append(res.Files, &FileDiff{Status: statusAdded, right: dsc2})
	}
	for dsc1 := range t1.AllFiles() {
		if dsc1.IsDir {
			continue
		}
		num := matched[dsc1.Hash]
		if num == 0 {
			res.Files = append(res.Files, &FileDiff{Status: statusDeleted, left: dsc1})
		} else {
			matched[dsc1.Hash]--
		}
	}

	slices.SortStableFunc(res.Files, func(a, b *FileDiff) int {
		return cmp.Compare(a.GetPath(), b.GetPath())
	})

	return res
}

func (d *FileDiff) GetPath() string {
	if d.left != nil {
		return d.left.Path
	}
	return d.right.Path
}

func (d *TreeDiff) String() string {
	var sb strings.Builder
	counts := map[string]int{}
	for _, f := range d.Files {
		counts[f.Status]++
		if f.Status == statusIdentical {
			continue
		}
		sb.WriteString(f.String())
	}
	sb.WriteString(colored(fmt.Sprintf("\nTotal     : %d \n", len(d.Files)), colorWhite, modeFont, styleOverline))

	sb.WriteString(fmt.Sprintf("%s     : %d\n", green(statusAdded), counts[statusAdded]))
	sb.WriteString(fmt.Sprintf("%s   : %d\n", red(statusDeleted), counts[statusDeleted]))
	sb.WriteString(fmt.Sprintf("%s     : %d\n", blue(statusMoved), counts[statusMoved]))
	sb.WriteString(fmt.Sprintf("%s   : %d\n", yellow(statusChanged), counts[statusChanged]))
	sb.WriteString(fmt.Sprintf("%s : %d\n", statusIdentical, counts[statusIdentical]))

	return sb.String()
}

func (d *FileDiff) String() string {
	path := d.GetPath()
	if d.Status == statusMoved {
		path += blue(" -> ") + d.right.Path
	}
	return fmt.Sprintf("%s: %s\n", d.FormatStatus(), path)
}

func (d *FileDiff) FormatStatus() string {
	st := fmt.Sprintf("%7s ", d.Status)
	switch d.Status {
	case statusIdentical:
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
