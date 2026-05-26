package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	graph "github.com/TSELab/astra/internal/graph"
	"github.com/TSELab/astra/internal/mapper"
	parser "github.com/TSELab/astra/internal/parser"
	buildinfoparser "github.com/TSELab/astra/internal/parser/buildinfo"
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
		// save graph to database
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
		//in, _ := cmd.Flags().GetString("input")
		out, _ := cmd.Flags().GetString("output")

		//to viz from json export
		/*
			graphJSON, err := os.ReadFile(in)
			if err != nil {
				return err
			}

			var eg graph.ExportGraph
			if err := json.Unmarshal(graphJSON, &eg); err != nil {
				return err
			}

			g := graph.FromExport(eg)
			dot := graph.ToDOT(g)
		*/
		//load graph from database
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

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest a provenance document into the AStRA graph",
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

var ingestGitCmd = &cobra.Command{
	Use:   "git <repo-url> <current-tag>",
	Short: "Parse and ingest commits between tags from a git repo",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoURL, currentTag := args[0], args[1]
		prevTag, _ := cmd.Flags().GetString("prev-tag")

		mapped, err := (&gitparser.GitParser{}).ParseTagRange(repoURL, currentTag, prevTag)
		if err != nil {
			return fmt.Errorf("parse git: %w", err)
		}

		g := mapper.ToAstraGraph(mapped)
		if err := db.SaveGraph(ctx, g); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		fmt.Println("[OK] Ingested git", repoURL, "@", currentTag)
		return nil
	},
}

var rootCmd = &cobra.Command{
	Use:   "astra",
	Short: "AStRA provenance graph tool",
}

func init() {
	parseCmd.Flags().StringP("input", "i", "", "input file path or repo URL (git)")
	parseCmd.Flags().StringP("output", "o", "", "output normalized JSON")
	parseCmd.Flags().StringP("format", "f", "git", "format: git|buildinfo|intoto|slsa")
	parseCmd.MarkFlagRequired("input")
	parseCmd.MarkFlagRequired("output")

	mapCmd.Flags().StringP("input", "i", "", "input parsed JSON (parser.Mapped)")
	mapCmd.Flags().StringP("output", "o", "", "output AStRA graph JSON")
	mapCmd.MarkFlagRequired("input")
	mapCmd.MarkFlagRequired("output")

	vizCmd.Flags().StringP("input", "i", "", "input graph JSON")
	vizCmd.Flags().StringP("output", "o", "graph.dot", "output DOT file")
	vizCmd.MarkFlagRequired("input")

	ingestGitCmd.Flags().String("prev-tag", "", "previous release tag (auto-discovered if omitted)")
	ingestCmd.AddCommand(ingestBuildinfoCmd, ingestGitCmd)

	rootCmd.AddCommand(parseCmd, mapCmd, vizCmd, initCmd, ingestCmd)
}

func main() {

	ctx = context.Background()
	// open DB
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
