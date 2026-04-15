# AStRA Toolchain (Go)

Go implementation of the AStRA pipeline:

- `astra parse`    → normalize raw logs
- `astra map`      → map normalized events to AStRA schema
- `astra graph`    → build a DAG and export JSON
- `astra risk`     → compute risk metrics (centrality, articulation, topo)
- `astra condense` → group nodes for simpler views


## Database Setup (Ent)
This project uses Ent as the persistence layer for storing AStRA graphs.

**1. Install dependencies**
Run this once after cloning the repo:
```bash
cd astra-go
go get entgo.io/ent@v0.14.6
go get entgo.io/ent/cmd/ent@v0.14.6
go mod tidy
```

**Defined schemas**
Schemas are located in:
internal/store/ent/schema/
Each file defines one entity:
artifact.go
step.go
principal.go
resource.go
edge.go

**2. Generate Ent code**

After creating or modifying schemas, run:
```bash
go run -mod=mod entgo.io/ent/cmd/ent generate ./internal/store/ent/schema
```

## Quickstart

```bash
cd astra-go
go mod tidy
go build ./cmd/astra
./astra parse   -f git -i "git repo URL" -o out/parsed.json
./astra map     -i out/parsed.json  -o out/graph.json
./astra graph   -i out/graph.json #TO DO 
./astra risk    -i out/graph.json -r out/risk.json --paths-from Principal --paths-to Artifact #TO DO 
./astra condense -i out/graph.json -o out/condensed.json --group-by phase #TO DO 
./astra viz -i out/graph.json -o out/graph.dot  
dot -Tsvg out/graph.dot -o out/graph.svg  
```

