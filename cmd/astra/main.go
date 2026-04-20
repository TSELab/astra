package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	graph "github.com/TSELab/astra/internal/graph"
	"github.com/TSELab/astra/internal/mapper"
	parser "github.com/TSELab/astra/internal/parser"
	buildinfoparser "github.com/TSELab/astra/internal/parser/buildinfo"
	gitparser "github.com/TSELab/astra/internal/parser/git"
	intotoparser "github.com/TSELab/astra/internal/parser/intoto"
	slsaparser "github.com/TSELab/astra/internal/parser/slsa"
	entstore "github.com/TSELab/astra/internal/store/entstore"
)

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

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

func main() {

	if len(os.Args) < 2 {
		fmt.Println("usage: astra <parse|map|graph|risk|condense> [flags]")
		os.Exit(2)
	}
	ctx := context.Background()

	// 3. open DB
	db, err := entstore.OpenSQLite("astra.db")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	sub := os.Args[1]
	switch sub {
	case "parse":
		var parser parser.Parser
		fs := flag.NewFlagSet("parse", flag.ExitOnError)
		in := fs.String("i", "", "input raw log (JSON)")
		out := fs.String("o", "", "output normalized JSON")
		format := fs.String("f", "git", "format of input (git|intoto|slsa|buildinfo)")
		fs.Parse(os.Args[2:])
		if *in == "" || *out == "" {
			fs.Usage()
			os.Exit(2)
		}
		switch *format {
		case "git": // git logs
			parser = &gitparser.GitParser{}
		case "intoto": // in-toto links
			parser = &intotoparser.InTotoParser{}
		case "slsa": // slsa
			parser = &slsaparser.SlsaParser{}
		case "buildinfo": // debian buildinfo logs
			parser = &buildinfoparser.BuildinfoParser{}

		default:
			fmt.Fprintf(os.Stderr, "unknown format: %s\n", *format)
			os.Exit(1)
		}
		data, err := parser.Parse(*in)
		must(err)
		must(writeJSON(*out, data))
		fmt.Println("[OK] Parsed ->", *out)
	case "map":
		fs := flag.NewFlagSet("map", flag.ExitOnError)
		in := fs.String("i", "", "input parsed JSON (parser.Mapped)")
		out := fs.String("o", "", "output AStRA graph JSON (typed)")

		fs.Parse(os.Args[2:])

		if *in == "" || *out == "" {
			fs.Usage()
			os.Exit(2)
		}

		// Read parsed
		var parsed parser.Mapped
		b, err := os.ReadFile(*in)
		must(err)
		must(json.Unmarshal(b, &parsed))

		// Convert to typed AStRA graph
		astra := mapper.ToAstraGraph(parsed)

		// 4. save graph
		if err := db.SaveGraph(ctx, astra); err != nil {
			log.Fatalf("save graph: %v", err)
		}

		log.Println("Graph saved successfully")
		//  validate schema invariants
		// must(graph.Validate(astra))

		must(writeJSON(*out, astra))
		fmt.Println("[OK] Mapped ->", *out)

		//case "graph":
	// TODO visual graph with cloned resources
	/*
		case "risk":
				fs := flag.NewFlagSet("risk", flag.ExitOnError)
				in := fs.String("i", "", "input graph JSON")
				rep := fs.String("r", "", "output risk report JSON")
				fromT := fs.String("paths-from", "", "optional source node type for shortest paths")
				toT := fs.String("paths-to", "", "optional dest node type for shortest paths")
				fs.Parse(os.Args[2:])
				if *in == "" || *rep == "" {
					fs.Usage()
					os.Exit(2)
				}
				var g graph.AstraGraph
				b, err := os.ReadFile(*in)
				must(err)
				must(json.Unmarshal(b, &g))
				r := risk.ComputeRiskReport(g, *fromT, *toT)
				must(writeJSON(*rep, r))
				fmt.Println("[OK] Risk report ->", *rep)

			case "condense":
				fs := flag.NewFlagSet("condense", flag.ExitOnError)
				in := fs.String("i", "", "input graph JSON")
				out := fs.String("o", "", "output condensed JSON")
				group := fs.String("group-by", "phase", "phase|type")
				fs.Parse(os.Args[2:])
				if *in == "" || *out == "" {
					fs.Usage()
					os.Exit(2)
				}
				var g graph.AstraGraph
				b, err := os.ReadFile(*in)
				must(err)
				must(json.Unmarshal(b, &g))
				cg := condense.Condense(g, *group)
				must(writeJSON(*out, cg))
				fmt.Println("[OK] Condensed ->", *out)
	*/
	case "viz":
		fs := flag.NewFlagSet("viz", flag.ExitOnError)
		in := fs.String("i", "", "input graph JSON")
		out := fs.String("o", "graph.dot", "output DOT file")
		fs.Parse(os.Args[2:])

		if *in == "" {
			fs.Usage()
			os.Exit(2)
		}
		//load graph
		loaded, err := db.LoadGraph(ctx, "")
		if err != nil {
			log.Fatalf("load graph: %v", err)
		}
		fmt.Printf("loaded: %d artifacts, %d steps, %d principals, %d resources, %d edges\n",
			len(loaded.Artifacts),
			len(loaded.Steps),
			len(loaded.Principals),
			len(loaded.Resources),
			len(loaded.Edges),
		)

		dot := graph.ToDOT(loaded)
		if err := os.WriteFile(*out, []byte(dot), 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Println("[OK] DOT graph written to", *out)

	default:
		fmt.Println("unknown subcommand:", sub)
		os.Exit(2)

	}
}
