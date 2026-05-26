/*
Package git implements the GitParser for AStRA.

GitParser parses Git repositories and maps commits and files
into AStRA's DAG-based artifact graph.

For each commit:
- The commit is represented as a step.
- Authors are represented as principals.
- The Git repo is treated as a resource (keyed by its URL).
- Files are tracked as input/output artifacts.
- Parent commits are mapped as input artifacts.
- The commit itself and changed files are mapped as output artifacts.

ParseTagRange walks commits between two release tags (prevTag exclusive,
currentTag inclusive). The commit at prevTag is emitted as an incomplete
boundary artifact so downstream queries know where exploration stopped.
PrevTag auto-discovers the immediately preceding tag by commit timestamp.
*/
package git

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	parser "github.com/TSELab/astra/internal/parser"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

// GitParser implements parser.Parser for Git repositories.
type GitParser struct{}

// TODO IDs should be moved out from git parser
// MakeFileArtifactID returns a namespaced AStRA artifact ID for a file
// at a specific commit in the given repository.
// Artifact ID format: artifact:gitfile:<host>/<owner>/<repo>@<commit-hash>:<filePath>
func MakeFileArtifactID(repoURL string, commitHash, filePath string) string {
	repoSlug := getRepoSlug(repoURL)
	return fmt.Sprintf("artifact:gitfile:%s@%s:%s", repoSlug, commitHash, filePath)
}

/*
MakeCommitStepID returns namespaced AStRA Step ID for a Git commit.
Step ID format: step:gitcommit:<host>/<owner>/<repo>@<commit-hash>
*/
func MakeCommitStepID(repoURL string, commitHash string) string {
	repoSlug := getRepoSlug(repoURL)
	return fmt.Sprintf("step:gitcommit:%s@%s", repoSlug, commitHash)
}

// MakeCommitArtifactID returns namespaced AStRA artifact ID
// for a Git commit.
// Commit artifact ID format: artifact:gitcommit:<host>/<owner>/<repo>@<commit-hash>
func MakeCommitArtifactID(repoURL string, commitHash string) string {
	repoSlug := getRepoSlug(repoURL)
	return fmt.Sprintf("artifact:gitcommit:%s@%s", repoSlug, commitHash)
}

// getRepoSlug normalizes a Git repository URL into a host/owner/repo slug.
// Supported URL formats include:
// https://github.com/owner/repo.git
// https://github.com/owner/repo
// git@github.com:owner/repo.git
func getRepoSlug(raw string) string {
	// Handle SSH scp-style: git@github.com:owner/repo.git
	if strings.HasPrefix(raw, "git@") {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) == 2 {
			host := strings.TrimPrefix(parts[0], "git@")
			path := strings.TrimSuffix(parts[1], ".git")
			return host + "/" + path
		}
		return raw
	}

	// Handle real URLs: https://..., ssh://...
	u, err := url.Parse(raw)
	if err == nil && u.Host != "" {
		path := strings.TrimPrefix(u.Path, "/")
		path = strings.TrimSuffix(path, ".git")
		return u.Host + "/" + path
	}

	return raw
}

// GetCommitIO computes the input and output files for a commit in a Git repository.
// - Inputs: files from parent commits that are modified or deleted.
// - Outputs: files added or modified in this commit.
// - Returns slices of *object.File for inputs and outputs.
func GetCommitIO(repo *git.Repository, hash string) (inputs []*object.File, outputs []*object.File, err error) {
	commit, err := repo.CommitObject(plumbing.NewHash(hash))
	if err != nil {
		return nil, nil, err
	}

	currTree, err := commit.Tree()
	if err != nil {
		return nil, nil, err
	}

	var parentTree *object.Tree
	var parentHash plumbing.Hash
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return nil, nil, err
		}
		parentHash = parent.Hash
		parentTree, err = parent.Tree()
		if err != nil {
			return nil, nil, err
		}
	}

	seenIn := map[string]bool{}
	seenOut := map[string]bool{}

	// if this is a root commit: everything is output
	if parentTree == nil {
		if err := currTree.Files().ForEach(func(f *object.File) error {
			if seenOut[f.Name] {
				return nil
			}
			seenOut[f.Name] = true
			outputs = append(outputs, f)
			return nil
		}); err != nil {
			return nil, nil, err
		}
		return inputs, outputs, nil
	}

	changes, err := object.DiffTree(parentTree, currTree)
	if err != nil {
		return nil, nil, err
	}

	for _, ch := range changes {
		action, _ := ch.Action()

		switch action {
		case merkletrie.Insert:
			path := ch.To.Name
			if seenOut[path] {
				continue
			}
			f, err := currTree.File(path)
			if err != nil {
				// submodule/symlink/rename edge-case; can't fetch file blob
				// skip metadata but continue
				continue
			}
			seenOut[path] = true
			outputs = append(outputs, f)

		case merkletrie.Delete:
			path := ch.From.Name
			if seenIn[path] {
				continue
			}
			f, err := parentTree.File(path)
			if err != nil {
				// submodule/symlink edge-case; skip metadata but continue
				continue
			}
			seenIn[path] = true
			inputs = append(inputs, f)

		case merkletrie.Modify:
			inPath := ch.From.Name
			outPath := ch.To.Name

			if !seenIn[inPath] {
				fBefore, err := parentTree.File(inPath)
				if err != nil {
					// submodule/symlink edge-case; skip metadata but continue
					continue
				}
				seenIn[inPath] = true
				inputs = append(inputs, fBefore)
			}

			if !seenOut[outPath] {
				fAfter, err := currTree.File(outPath)
				if err != nil {
					return nil, nil, err
				}
				seenOut[outPath] = true
				outputs = append(outputs, fAfter)
			}
		}
	}

	_ = parentHash // keep to later build artifact IDs for inputs at parent commit
	return inputs, outputs, nil
}

// commitToRecord maps a single git commit into a parser.Record.
// It computes file-level IO, parent commit artifacts, and attaches the git resource.
func commitToRecord(c *object.Commit, repo *git.Repository, remoteURL string) (parser.Record, error) {
	rec := parser.Record{
		Step: parser.StepItem{
			ID:           MakeCommitStepID(remoteURL, c.Hash.String()),
			Label:        "Commit",
			Kind:         "step",
			Completeness: "complete",
			Attrs: map[string]string{
				"phase":   "source",
				"message": strings.TrimSpace(c.Message),
			},
		},
		Principal: parser.PrincipalItem{
			ID:           fmt.Sprintf("principal:%s", c.Author.Email),
			Label:        c.Author.Name,
			Kind:         "principal",
			Completeness: "complete",
			Attrs:        map[string]string{"email": c.Author.Email},
		},
	}

	inputs, outputs, err := GetCommitIO(repo, c.Hash.String())
	if err != nil {
		return parser.Record{}, err
	}

	// Parent hash is needed to version the "before" (input) artifacts.
	parentHash := ""
	if c.NumParents() > 0 {
		for i := 0; i < c.NumParents(); i++ {
			parent, err := c.Parent(i)
			if err != nil {
				return parser.Record{}, err
			}
			parentHash = parent.Hash.String()
			rec.ArtifactsIn = append(rec.ArtifactsIn, parser.ArtifactItem{
				ID:           MakeCommitArtifactID(remoteURL, parentHash),
				Label:        parentHash,
				Kind:         "git-commit",
				Completeness: "complete",
				Attrs: map[string]string{
					"role":  "parent",
					"index": strconv.Itoa(i),
				},
			})
		}
	}

	// ArtifactsIn (before versions).
	// Root commit => parentHash == "" and inputs should be empty.
	// Input files are connected to parent 0 only.
	for _, f := range inputs {
		if f == nil || parentHash == "" {
			continue
		}
		rec.ArtifactsIn = append(rec.ArtifactsIn, parser.ArtifactItem{
			ID:           MakeFileArtifactID(remoteURL, parentHash, f.Name),
			Label:        f.Name,
			Kind:         "git-file",
			Completeness: "complete",
			Attrs: map[string]string{
				"hash": f.Hash.String(),
				"size": strconv.FormatInt(f.Size, 10),
				"mode": f.Mode.String(),
			},
		})
	}

	// Add the commit itself as an output artifact.
	rec.ArtifactsOut = append(rec.ArtifactsOut, parser.ArtifactItem{
		ID:           MakeCommitArtifactID(remoteURL, c.Hash.String()),
		Label:        c.Hash.String(),
		Kind:         "git-commit",
		Completeness: "complete",
		Attrs: map[string]string{
			"message": strings.TrimSpace(c.Message),
			"author":  c.Author.Email,
			"time":    strconv.FormatInt(c.Author.When.Unix(), 10),
		},
	})

	// ArtifactsOut (after versions).
	for _, f := range outputs {
		if f == nil {
			continue
		}
		rec.ArtifactsOut = append(rec.ArtifactsOut, parser.ArtifactItem{
			ID:           MakeFileArtifactID(remoteURL, c.Hash.String(), f.Name),
			Label:        f.Name,
			Kind:         "git-file",
			Completeness: "complete",
			Attrs: map[string]string{
				"content-hash": f.Hash.String(),
				"size":         strconv.FormatInt(f.Size, 10),
				"mode":         f.Mode.String(),
			},
		})
	}

	rec.Resources = append(rec.Resources, parser.ResourceItem{
		ID:           "vcs:" + remoteURL,
		Label:        remoteURL,
		Kind:         "vcs",
		Completeness: "complete",
		Attrs:        map[string]string{"uri": remoteURL},
	})

	return rec, nil
}

// resolveTagCommit resolves a tag name to its underlying commit.
// Handles both lightweight tags (direct commit reference) and annotated tags
// (tag object that points to a commit).
func resolveTagCommit(repo *git.Repository, tagName string) (*object.Commit, error) {
	tagRef, err := repo.Tag(tagName)
	if err != nil {
		return nil, fmt.Errorf("tag %q not found: %w", tagName, err)
	}

	// Annotated tag: the reference points to a tag object
	if tagObj, err := repo.TagObject(tagRef.Hash()); err == nil {
		return repo.CommitObject(tagObj.Target)
	}

	// Lightweight tag: the reference points directly to a commit
	return repo.CommitObject(tagRef.Hash())
}

// PrevTag finds the tag immediately preceding currentTag by commit timestamp.
// It clones from the same already-cloned repo (no network call).
// Returns ("", nil) if currentTag is the oldest tag (walk to root).
func PrevTag(repo *git.Repository, currentTag string) (string, error) {
	currentCommit, err := resolveTagCommit(repo, currentTag)
	if err != nil {
		return "", fmt.Errorf("resolve current tag: %w", err)
	}
	currentTime := currentCommit.Committer.When

	tagsIter, err := repo.Tags()
	if err != nil {
		return "", fmt.Errorf("list tags: %w", err)
	}
	defer tagsIter.Close()

	var bestTag string
	var bestTime time.Time

	err = tagsIter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		if name == currentTag {
			return nil
		}
		commit, err := resolveTagCommit(repo, name)
		if err != nil {
			return nil // skip tags that can't be resolved
		}
		t := commit.Committer.When
		if t.Before(currentTime) && (bestTag == "" || t.After(bestTime)) {
			bestTag = name
			bestTime = t
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("iterate tags: %w", err)
	}
	return bestTag, nil
}

// TagHash returns the commit hash string for the given tag name.
func TagHash(repo *git.Repository, tagName string) (string, error) {
	c, err := resolveTagCommit(repo, tagName)
	if err != nil {
		return "", err
	}
	return c.Hash.String(), nil
}

// LatestTag finds the tag whose commit has the highest committer timestamp.
// Returns ("", nil) if the repo has no tags.
func LatestTag(repo *git.Repository) (string, error) {
	tagsIter, err := repo.Tags()
	if err != nil {
		return "", fmt.Errorf("list tags: %w", err)
	}
	defer tagsIter.Close()

	var bestTag string
	var bestTime time.Time

	_ = tagsIter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		commit, err := resolveTagCommit(repo, name)
		if err != nil {
			return nil
		}
		t := commit.Committer.When
		if bestTag == "" || t.After(bestTime) {
			bestTag = name
			bestTime = t
		}
		return nil
	})
	return bestTag, nil
}

// CloneRepo clones repoURL into memory with all tags fetched.
func CloneRepo(repoURL string) (*git.Repository, error) {
	fmt.Println("Cloning:", repoURL)
	return git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:      repoURL,
		Tags:     git.AllTags,
		Progress: os.Stdout,
	})
}

// ParseTagRangeFromRepo walks commits from currentTag back to (but not including)
// prevTag using an already-cloned repo. If currentTag is empty, HEAD is used and
// prevTag defaults to the latest tag. If prevTag is empty it is auto-discovered
// via PrevTag.
//
// Each walked commit is emitted as a complete Step + Artifacts. The commit at
// prevTag (the boundary) is emitted as a single incomplete Artifact so the
// graph records where exploration stopped.
func ParseTagRangeFromRepo(repo *git.Repository, repoURL, currentTag, prevTag string) (parser.Mapped, error) {
	var headCommit *object.Commit
	var err error

	if currentTag == "" {
		ref, err := repo.Head()
		if err != nil {
			return parser.Mapped{}, fmt.Errorf("head: %w", err)
		}
		headCommit, err = repo.CommitObject(ref.Hash())
		if err != nil {
			return parser.Mapped{}, fmt.Errorf("resolve HEAD commit: %w", err)
		}
		if prevTag == "" {
			prevTag, err = LatestTag(repo)
			if err != nil {
				return parser.Mapped{}, fmt.Errorf("find latest tag: %w", err)
			}
		}
	} else {
		headCommit, err = resolveTagCommit(repo, currentTag)
		if err != nil {
			return parser.Mapped{}, fmt.Errorf("resolve tag %q: %w", currentTag, err)
		}
		if prevTag == "" {
			prevTag, err = PrevTag(repo, currentTag)
			if err != nil {
				return parser.Mapped{}, fmt.Errorf("find prev tag: %w", err)
			}
		}
	}

	stopHash := ""
	if prevTag != "" {
		stopCommit, err := resolveTagCommit(repo, prevTag)
		if err != nil {
			return parser.Mapped{}, fmt.Errorf("resolve prev tag %q: %w", prevTag, err)
		}
		stopHash = stopCommit.Hash.String()
	}

	commits, err := repo.Log(&git.LogOptions{From: headCommit.Hash})
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("log: %w", err)
	}
	defer commits.Close()

	out := parser.Mapped{
		Source:       "go-git",
		NormalizedAt: time.Now().Unix(),
	}

	for {
		c, err := commits.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return parser.Mapped{}, err
		}

		if stopHash != "" && c.Hash.String() == stopHash {
			out.Mapped = append(out.Mapped, parser.Record{
				ArtifactsOut: []parser.ArtifactItem{
					{
						ID:           MakeCommitArtifactID(repoURL, c.Hash.String()),
						Label:        c.Hash.String(),
						Kind:         "git-commit",
						Completeness: "incomplete",
						Attrs: map[string]string{
							"boundary": "true",
							"tag":      prevTag,
						},
					},
				},
			})
			break
		}

		rec, err := commitToRecord(c, repo, repoURL)
		if err != nil {
			return parser.Mapped{}, err
		}
		out.Mapped = append(out.Mapped, rec)
	}

	return out, nil
}

// ParseTagRange clones repoURL and delegates to ParseTagRangeFromRepo.
// If currentTag is empty, HEAD is used. If prevTag is empty it is auto-discovered.
func (p *GitParser) ParseTagRange(repoURL, currentTag, prevTag string) (parser.Mapped, error) {
	repo, err := CloneRepo(repoURL)
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("clone %s: %w", repoURL, err)
	}
	return ParseTagRangeFromRepo(repo, repoURL, currentTag, prevTag)
}

// Parse clones the Git repository from the given URL into memory, extracts commits,
// and maps them into a parser.Mapped structure for AStRA.
//
// This method implements the parser.Parser interface. For tag-range parsing
// (the recommended approach), use ParseTagRange instead.
func (p *GitParser) Parse(r io.Reader) (parser.Mapped, error) {
	urlBytes, err := io.ReadAll(r)
	if err != nil {
		return parser.Mapped{}, err
	}
	repoURL := strings.TrimSpace(string(urlBytes))
	fmt.Println("Cloning:", repoURL)

	repo, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("clone error: %w", err)
	}

	rem, err := repo.Remote("origin")
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("remote error: %w", err)
	}
	remoteURLs := rem.Config().URLs
	if len(remoteURLs) == 0 {
		return parser.Mapped{}, fmt.Errorf("origin remote has no URLs")
	}
	remoteURL := remoteURLs[0]

	ref, err := repo.Head()
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("head error: %w", err)
	}

	commits, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("log error: %w", err)
	}

	out := parser.Mapped{
		Source:       "go-git",
		NormalizedAt: time.Now().Unix(),
	}

	err = commits.ForEach(func(c *object.Commit) error {
		rec, err := commitToRecord(c, repo, remoteURL)
		if err != nil {
			return err
		}
		out.Mapped = append(out.Mapped, rec)
		return nil
	})
	if err != nil {
		return parser.Mapped{}, err
	}

	return out, nil
}
