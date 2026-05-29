package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	graph "github.com/TSELab/astra/internal/graph"
	"github.com/TSELab/astra/internal/mapper"
	parser "github.com/TSELab/astra/internal/parser"
	buildinfoparser "github.com/TSELab/astra/internal/parser/debian/buildinfo"
	packagesparser "github.com/TSELab/astra/internal/parser/debian/packages"
	gitparser "github.com/TSELab/astra/internal/parser/git"
	intotoparser "github.com/TSELab/astra/internal/parser/intoto"
	slsaparser "github.com/TSELab/astra/internal/parser/slsa"
	entstore "github.com/TSELab/astra/internal/store/entstore"
)

var (
	ctx context.Context
	db  *entstore.Store
)

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse a provenance source into normalized records",
	RunE: func(cmd *cobra.Command, args []string) error {
		in, _ := cmd.Flags().GetString("input")
		out, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")

		var p parser.Parser
		var r io.Reader

		switch format {
		case "git":
			p = &gitparser.GitParser{}
			r = strings.NewReader(in)
		case "buildinfo":
			p = &buildinfoparser.BuildinfoParser{}
			f, err := os.Open(in)
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		case "packages":
			archiveURL, _ := cmd.Flags().GetString("archive-url")
			p = &packagesparser.PackagesParser{ArchiveURL: archiveURL}
			f, err := os.Open(in)
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		case "intoto":
			p = &intotoparser.InTotoParser{}
			f, err := os.Open(in)
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		case "slsa":
			p = &slsaparser.SlsaParser{}
			f, err := os.Open(in)
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		default:
			return fmt.Errorf("unknown format: %s", format)
		}

		data, err := p.Parse(r)
		if err != nil {
			return err
		}
		if err := writeJSON(out, data); err != nil {
			return err
		}
		fmt.Println("[OK] Parsed ->", out)
		return nil
	},
}

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Map normalized records to an AStRA graph",
	RunE: func(cmd *cobra.Command, args []string) error {
		in, _ := cmd.Flags().GetString("input")
		out, _ := cmd.Flags().GetString("output")

		b, err := os.ReadFile(in)
		if err != nil {
			return err
		}
		var parsed parser.Mapped
		if err := json.Unmarshal(b, &parsed); err != nil {
			return err
		}

		astra := mapper.ToAstraGraph(parsed)
		exported_graph := graph.ToExport(astra)
		if err := db.SaveGraph(ctx, astra); err != nil {
			log.Fatalf("save graph: %v", err)
		}
		log.Println("Graph saved successfully to the database")

		if err := writeJSON(out, exported_graph); err != nil {
			return err
		}
		log.Println("Graph saved successfully to the disk")

		fmt.Println("[OK] Mapped ->", out)
		return nil
	},
}

var vizCmd = &cobra.Command{
	Use:   "viz",
	Short: "Render an AStRA graph as DOT",
	RunE: func(cmd *cobra.Command, args []string) error {
		out, _ := cmd.Flags().GetString("output")

		loaded, err := db.LoadGraph(ctx, "")
		if err != nil {
			log.Fatalf("load graph: %v", err)
		}
		fmt.Printf("loaded: %d artifacts, %d steps, %d principals, %d resources, %d edges\n",
			len(loaded.Artifacts),
			len(loaded.Steps),
			len(loaded.Principals),
			len(loaded.Resources),
			len(loaded.Edges))

		dot := graph.ToDOT(loaded)
		if err := os.WriteFile(out, []byte(dot), 0o644); err != nil {
			return err
		}
		fmt.Println("[OK] DOT graph written to", out)
		return nil
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the AStRA database",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("[OK] AStRA database ready at astra.db")
		return nil
	},
}

// ── ingest ────────────────────────────────────────────────────────────────────

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest a provenance document into the AStRA graph",
}

var ingestDebianCmd = &cobra.Command{
	Use:   "debian",
	Short: "Ingest Debian provenance documents",
}

var ingestBuildinfoCmd = &cobra.Command{
	Use:   "buildinfo <file>",
	Short: "Parse and ingest a single .buildinfo file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		mapped, err := (&buildinfoparser.BuildinfoParser{}).Parse(f)
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}

		g := mapper.ToAstraGraph(mapped)
		if err := db.SaveGraph(ctx, g); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		fmt.Println("[OK] Ingested", path)
		return nil
	},
}

var ingestPackagesCmd = &cobra.Command{
	Use:   "packages <file-or-url>",
	Short: "Parse and ingest a Debian Packages index file (local path or snapshot URL)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]
		archiveFlagURL, _ := cmd.Flags().GetString("archive-url")

		var reader io.Reader
		var archiveURL string

		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			fetchURL := arg
			if strings.HasSuffix(arg, "/") {
				fetchURL = arg + "Packages.gz"
			}
			// Base directory URL (without the Packages.gz filename) becomes the archive URL.
			archiveURL = strings.TrimSuffix(fetchURL, "Packages.gz")

			resp, err := http.Get(fetchURL) //nolint:noctx
			if err != nil {
				return fmt.Errorf("fetch %s: %w", fetchURL, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("fetch %s: HTTP %d", fetchURL, resp.StatusCode)
			}

			if strings.HasSuffix(fetchURL, ".gz") {
				gzr, err := gzip.NewReader(resp.Body)
				if err != nil {
					return fmt.Errorf("decompress: %w", err)
				}
				defer gzr.Close()
				reader = gzr
			} else {
				reader = resp.Body
			}
		} else {
			f, err := os.Open(arg)
			if err != nil {
				return err
			}
			defer f.Close()
			reader = f
			archiveURL = archiveFlagURL
		}

		mapped, err := (&packagesparser.PackagesParser{ArchiveURL: archiveURL}).Parse(reader)
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}

		g := mapper.ToAstraGraph(mapped)
		if err := db.SaveGraph(ctx, g); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		fmt.Printf("[OK] Ingested %d package entries from %s\n", len(mapped.Mapped), arg)
		return nil
	},
}

var ingestGitCmd = &cobra.Command{
	Use:   "git <repo-url>",
	Short: "Parse and ingest a tag range from a git repo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoURL := args[0]
		currentTag, _ := cmd.Flags().GetString("tag")
		prevTag, _ := cmd.Flags().GetString("prev-tag")

		mapped, err := (&gitparser.GitParser{}).ParseTagRange(repoURL, currentTag, prevTag)
		if err != nil {
			return fmt.Errorf("parse git: %w", err)
		}

		g := mapper.ToAstraGraph(mapped)
		if err := db.SaveGraph(ctx, g); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		fmt.Println("[OK] Ingested git", repoURL)
		return nil
	},
}

var ingestIntotoCmd = &cobra.Command{
	Use:   "intoto <file>",
	Short: "Parse and ingest an in-toto link file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		linker, _ := cmd.Flags().GetString("linker")

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		mapped, err := (&intotoparser.InTotoParser{Linker: linker}).Parse(f)
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}

		g := mapper.ToAstraGraph(mapped)
		if err := db.SaveGraph(ctx, g); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		fmt.Println("[OK] Ingested", path)
		return nil
	},
}

// ── root ──────────────────────────────────────────────────────────────────────

var rootCmd = &cobra.Command{
	Use:   "astra",
	Short: "AStRA provenance graph tool",
}

func init() {
	parseCmd.Flags().StringP("input", "i", "", "input file path or repo URL (git)")
	parseCmd.Flags().StringP("output", "o", "", "output normalized JSON")
	parseCmd.Flags().StringP("format", "f", "git", "format: git|buildinfo|packages|intoto|slsa")
	parseCmd.Flags().StringP("archive-url", "u", "https://deb.debian.org/debian", "base URL of the Debian archive (packages format only)")
	parseCmd.MarkFlagRequired("input")
	parseCmd.MarkFlagRequired("output")

	mapCmd.Flags().StringP("input", "i", "", "input parsed JSON (parser.Mapped)")
	mapCmd.Flags().StringP("output", "o", "", "output AStRA graph JSON")
	mapCmd.MarkFlagRequired("input")
	mapCmd.MarkFlagRequired("output")

	vizCmd.Flags().StringP("input", "i", "", "input graph JSON (unused; graph is loaded from DB)")
	vizCmd.Flags().StringP("output", "o", "graph.dot", "output DOT file")

	ingestPackagesCmd.Flags().StringP("archive-url", "u", "https://deb.debian.org/debian", "base URL of the Debian archive")

	ingestGitCmd.Flags().StringP("tag", "t", "", "current tag to walk from (default: latest tag in repo)")
	ingestGitCmd.Flags().String("prev-tag", "", "previous tag to stop at (default: auto-detected)")

	ingestIntotoCmd.Flags().String("linker", "debtrace", "name of the provenance database that produced the link file")

	ingestDebianCmd.AddCommand(ingestBuildinfoCmd, ingestPackagesCmd)
	ingestCmd.AddCommand(ingestGitCmd, ingestDebianCmd, ingestIntotoCmd)

	rootCmd.AddCommand(parseCmd, mapCmd, vizCmd, initCmd, ingestCmd)
}

func main() {
	ctx = context.Background()
	var err error
	db, err = entstore.OpenSQLite("astra.db")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
