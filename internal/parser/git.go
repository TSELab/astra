package parser

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitParser struct{}

// Parse get repo url, clone it and get git log and normalizes it
func (p *GitParser) Parse(repoURL string) (Normalized, error) {

	tmpDir := filepath.Join(os.TempDir(), "gitrepo-history")
	_ = os.RemoveAll(tmpDir) // clean up from previous runs

	fmt.Println("Cloning:", repoURL)
	fmt.Println("Into   :", tmpDir)

	repo, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		log.Fatal("clone error:", err)
	}

	ref, err := repo.Head()
	if err != nil {
		log.Fatal("head error:", err)
	}

	commits, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		log.Fatal("log error:", err)
	}

	fmt.Println()
	fmt.Println("====================================")
	fmt.Println("        COMMIT HISTORY FOR THE MAIN BRANCE")
	fmt.Println("====================================")

	err = commits.ForEach(func(c *object.Commit) error {
		fmt.Println("------------------------------------")
		fmt.Println("Commit :", c.Hash)
		fmt.Println("Author :", c.Author.Name, "<"+c.Author.Email+">")
		fmt.Println("Date   :", c.Author.When)
		fmt.Println("Message:")
		fmt.Println(c.Message)

		//  Changed files vs parent (diff)
		fmt.Println("Changed files (vs parent):")
		if c.NumParents() == 0 {
			fmt.Println("  (no parent – initial commit)")
		} else {
			parent, err := c.Parent(0)
			if err != nil {
				return err
			}

			patch, err := parent.Patch(c)
			if err != nil {
				return err
			}

			stats := patch.Stats()
			if len(stats) == 0 {
				fmt.Println("  (no changes detected)")
			}
			for _, s := range stats {
				fmt.Printf("  %s ( +%d / -%d )\n", s.Name, s.Addition, s.Deletion)
			}
		}

		//  All files in this commit (tree walk)
		fmt.Println("All files in this commit (tree):")
		tree, err := c.Tree()
		if err != nil {
			return err
		}

		err = tree.Files().ForEach(func(f *object.File) error {
			fmt.Println("  ", f.Name)
			return nil
		})
		if err != nil {
			return err
		}

		fmt.Println()
		return nil
	})

	if err != nil {
		log.Fatal("iteration error:", err)
	}

	n := Normalized{
		Source:       "git_log/go-git",
		NormalizedAt: time.Now().Unix(),
	}

	return n, nil
}
