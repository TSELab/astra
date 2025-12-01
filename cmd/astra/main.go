package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	parse "github.com/abuishgair/astra/internal/parser"
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
	sub := os.Args[1]
	switch sub {
	case "parse":
		var parser parse.Parser

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
			parser = &parse.GitParser{}
		case "intoto": // in-toto links
			parser = &parse.InTotoParser{}
		case "slsa": // slsa
			parser = &parse.SlsaParser{}
		case "buildinfo": // debian buildinfo logs
			parser = &parse.BuildinfoParser{}
		default:
			fmt.Fprintf(os.Stderr, "unknown format: %s\n", *format)
			os.Exit(1)
		}
		data, err := parser.Parse(*in)
		must(err)
		must(writeJSON(*out, data))
		fmt.Println("[OK] Parsed ->", *out)
		/*
			case "map":

				fs := flag.NewFlagSet("map", flag.ExitOnError)
				in := fs.String("i", "", "input normalized JSON")
				m := fs.String("m", "", "mapping YAML")
				out := fs.String("o", "", "output mapped JSON")
				fs.Parse(os.Args[2:])
				_ = m
				if *in == "" || *out == "" || *m == "" {
					fs.Usage()
					os.Exit(2)
				}
				var norm parse.Normalized
				b, err := os.ReadFile(*in)
				must(err)
				must(json.Unmarshal(b, &norm))
				rules, err := mapping.LoadMapping(*m)
				must(err)
				mapped := mapping.MapEvents(norm, rules)
				must(writeJSON(*out, mapped))
				fmt.Println("[OK] Mapped ->", *out)

			case "graph":
				fs := flag.NewFlagSet("graph", flag.ExitOnError)
				in := fs.String("i", "", "input mapped JSON")
				out := fs.String("o", "", "output graph JSON")
				fs.Parse(os.Args[2:])
				if *in == "" || *out == "" {
					fs.Usage()
					os.Exit(2)
				}
				var mapped mapping.Mapped
				b, err := os.ReadFile(*in)
				must(err)
				must(json.Unmarshal(b, &mapped))
				// canonical graph
				canonical := graphing.BuildGraph(mapped, false)
				graphing.SaveGraphJSON(*out, canonical)

				// visual graph with cloned resources
				visual := graphing.BuildGraph(mapped, true)
				graphing.SaveGraphJSON("graph.visual.json", visual)

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
				var g graphing.AstraGraph
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
				var g graphing.AstraGraph
				b, err := os.ReadFile(*in)
				must(err)
				must(json.Unmarshal(b, &g))
				cg := condense.Condense(g, *group)
				must(writeJSON(*out, cg))
				fmt.Println("[OK] Condensed ->", *out)

			case "viz":
				fs := flag.NewFlagSet("viz", flag.ExitOnError)
				in := fs.String("i", "", "input graph JSON")
				out := fs.String("o", "graph.dot", "output DOT file")
				fs.Parse(os.Args[2:])

				if *in == "" {
					fs.Usage()
					os.Exit(2)
				}

				graphJSON, err := os.ReadFile(*in)
				if err != nil {
					log.Fatal(err)
				}

				var g graphing.AstraGraph
				if err := json.Unmarshal(graphJSON, &g); err != nil {
					log.Fatal(err)
				}

				dot := graphing.ToDOT(g)
				if err := os.WriteFile(*out, []byte(dot), 0644); err != nil {
					log.Fatal(err)
				}
				fmt.Println("[OK] DOT graph written to", *out)

			default:
				fmt.Println("unknown subcommand:", sub)
				os.Exit(2)
		*/
	}
}
