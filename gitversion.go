package main

import (
	"fmt"
	"strings"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

type gitVersion struct {
	r       *git.Repository
	visited map[plumbing.Hash]struct{}

	head      *plumbing.Reference
	tags      map[plumbing.Hash]*plumbing.Reference
	latestTag *plumbing.Reference
	distance  int
}

func (gt *gitVersion) getTag(hash string) string {
	h := plumbing.NewHash(hash)
	v, ok := gt.tags[h]
	if !ok {
		return ""
	}
	return v.Name().Short()
}

func newGitVersion(path string) (*gitVersion, error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return &gitVersion{}, err
	}

	gt := &gitVersion{r: r}
	gt.initVersion()
	return gt, nil
}

func (gt *gitVersion) Commit() string {
	if gt.head == nil {
		return ""
	}
	return gt.head.String()[:7]
}

func (gt *gitVersion) TagVersion() string {
	if gt.latestTag == nil {
		return "0.0.0"
	}
	return strings.TrimPrefix(gt.latestTag.Name().Short(), "v")
}

func (gt *gitVersion) Version() string {
	cmt := gt.Commit()
	tv := gt.TagVersion()
	if cmt == "" || gt.distance == 0 {
		return tv
	}
	return fmt.Sprintf("%s-%d-%s", tv, gt.distance, cmt)
}

func (gt *gitVersion) initVersion() {
	var err error
	if gt.head, err = gt.r.Head(); err != nil {
		return
	}

	gt.getTags()

	err = gt.getLatestTag()
	if err != nil {
		return
	}

	gt.visited = map[plumbing.Hash]struct{}{}
}

func (gt *gitVersion) getTags() error {
	gt.tags = map[plumbing.Hash]*plumbing.Reference{}
	iter, err := gt.r.Tags()
	if err != nil {
		return err
	}

	iter.ForEach(func(arg1 *plumbing.Reference) error {

		if obj, err := gt.r.Object(plumbing.AnyObject, arg1.Hash()); err == nil {
			switch obj.(type) {
			case *object.Tag:
				tag := obj.(*object.Tag)
				if c, e := tag.Commit(); e == nil {
					if oldTagRef, ok := gt.tags[c.Hash]; ok {
						oldObj, err := gt.r.Object(plumbing.AnyObject, oldTagRef.Hash())
						if err != nil {
							return nil
						}

						switch oldObj.(type) {
						case *object.Tag:
							oldTag := oldObj.(*object.Tag)
							if tag.Tagger.When.After(oldTag.Tagger.When) {
								gt.tags[c.Hash] = arg1
							}
						case *object.Commit:
							gt.tags[c.Hash] = arg1
						}
					} else {
						gt.tags[c.Hash] = arg1
					}
				}

			case *object.Commit:
				if _, ok := gt.tags[obj.ID()]; !ok {
					gt.tags[obj.ID()] = arg1
				}
			}
		}

		return nil
	})
	return nil
}

func (gt *gitVersion) getLatestTag() error {
	if gt.head.IsTag() {
		gt.latestTag = gt.head
		gt.distance = 0
		return nil
	}
	hhash := gt.head.Hash()
	var thash plumbing.Hash
	cIter, err := gt.r.Log(&git.LogOptions{From: hhash})
	if err != nil {
		return err
	}
	err = cIter.ForEach(func(c *object.Commit) error {
		if tag, ok := gt.tags[c.Hash]; ok {
			gt.latestTag = tag
			thash = c.Hash
			return storer.ErrStop
		}
		return nil
	})
	if err != nil {
		return err
	}

	d, err := gt.diffCommits(hhash, thash)
	if err != nil {
		return err
	}
	gt.distance = d
	return nil
}

func (gt *gitVersion) countCommits(h plumbing.Hash) (int, error) {
	if h.IsZero() {
		return 0, nil
	}
	cIter, err := gt.r.Log(&git.LogOptions{From: h})
	if err != nil {
		return 0, err
	}
	n := 0
	err = cIter.ForEach(func(c *object.Commit) error {
		n++
		return nil
	})
	return n, err
}

func (gt *gitVersion) diffCommits(from, to plumbing.Hash) (int, error) {
	if from == to {
		return 0, nil
	}
	n1, err := gt.countCommits(from)
	if err != nil {
		return 0, err
	}
	n2, err := gt.countCommits(to)
	if err != nil {
		return 0, err
	}
	return n1 - n2, nil
}
