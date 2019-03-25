package core

import (
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
)

const (
	EmptyTreeCommitId = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
)

type File interface {
	Name() string
	Content() string
}

type FileList struct {
	Files map[string]File
}

type GitFile struct {
	fo *object.File
}

func (f GitFile) Name() string {
	return f.fo.Name
}
func (f GitFile) Content() string {

	content, err := f.fo.Contents()

	if err != nil {
		return ""
	}

	return content
}

func CloneRepository(url *string, branch *string, depth int) (*git.Repository, string, error) {
	urlVal := *url
	branchVal := *branch
	dir, err := ioutil.TempDir("", "gitrob")
	if err != nil {
		return nil, "", err
	}
	repository, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:           urlVal,
		Depth:         depth,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchVal)),
		SingleBranch:  true,
		Tags:          git.NoTags,
	})
	if err != nil {
		return nil, dir, err
	}
	return repository, dir, nil
}

func FetchFiles(gc *git.Repository) (*FileList, error) {

	ref, err := gc.Head()
	if err != nil {
		return nil, err
	}

	// ... retrieving the commit object
	commit, err := gc.CommitObject(ref.Hash())

	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()

	if err != nil {
		return nil, err
	}
	ff := tree.Files()
	repoFiles := map[string]File{}
	for {
		fo, err := ff.Next()

		if err != nil {
			break
		}
		repoFile := GitFile{
			fo: fo,
		}
		repoFiles[fo.Name] = repoFile
	}
	return &FileList{repoFiles}, nil
}

func GetRepositoryFiles(repository *git.Repository) []io.ReadCloser {
	blobIterOutput, err := repository.BlobObjects()
	blobIter := blobIterOutput.EncodedObjectIter
	fileReaders := make([]io.ReadCloser, 0)
	if err != nil {
		return fileReaders
	}
	for {
		blobEncoded, err := blobIter.Next()
		if err != nil {
			return fileReaders
		}
		if blobEncoded == nil {
			return fileReaders
		}
		blobObjectReader, err := blobEncoded.Reader()
		fileReaders = append(fileReaders, blobObjectReader)
		// buf := new(bytes.Buffer)
		// buf.ReadFrom(blobObjectReader)
		// s := buf.String() // Does a complete copy of the bytes in the buffer.
	}
	if err != nil {
		return fileReaders
	}
	return fileReaders
}

func GetRepositoryHistory(repository *git.Repository) ([]*object.Commit, error) {
	var commits []*object.Commit
	ref, err := repository.Head()
	if err != nil {
		return nil, err
	}
	cIter, err := repository.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, err
	}
	cIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})
	return commits, nil
}

func GetChanges(commit *object.Commit, repo *git.Repository) (object.Changes, error) {
	parentCommit, err := GetParentCommit(commit, repo)
	if err != nil {
		return nil, err
	}

	commitTree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	parentCommitTree, err := parentCommit.Tree()
	if err != nil {
		return nil, err
	}

	changes, err := object.DiffTree(parentCommitTree, commitTree)
	if err != nil {
		return nil, err
	}
	return changes, nil
}

func GetParentCommit(commit *object.Commit, repo *git.Repository) (*object.Commit, error) {
	if commit.NumParents() == 0 {
		parentCommit, err := repo.CommitObject(plumbing.NewHash(EmptyTreeCommitId))
		if err != nil {
			return nil, err
		}
		return parentCommit, nil
	}
	parentCommit, err := commit.Parents().Next()
	if err != nil {
		return nil, err
	}
	return parentCommit, nil
}

func GetChangeAction(change *object.Change) string {
	action, err := change.Action()
	if err != nil {
		return "Unknown"
	}
	switch action {
	case merkletrie.Insert:
		return "Insert"
	case merkletrie.Modify:
		return "Modify"
	case merkletrie.Delete:
		return "Delete"
	default:
		return "Unknown"
	}
}

func GetChangePath(change *object.Change) string {
	action, err := change.Action()
	if err != nil {
		return change.To.Name
	}

	if action == merkletrie.Delete {
		return change.From.Name
	} else {
		return change.To.Name
	}
}
