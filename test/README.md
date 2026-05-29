# Test data

## Dataset (`dataset/`)

Three Debian source packages from the bookworm release, each with a buildinfo, a
debtrace link file, and a shared Packages index from the same snapshot.

| File | Description |
|---|---|
| `lz4_1.9.4-1_amd64.buildinfo` | Debian build record for lz4 1.9.4-1 |
| `libarchive_3.6.2-1_amd64.buildinfo` | Debian build record for libarchive 3.6.2-1 |
| `xxhash_0.8.1-1_amd64.buildinfo` | Debian build record for xxhash 0.8.1-1 |
| `lz4-build.link.json` | Link file: `github.com/lz4/lz4@v1.9.4` → `lz4_1.9.4.orig.tar.gz` |
| `libarchive-build.link.json` | Link file: `github.com/libarchive/libarchive@v3.6.2` → `libarchive_3.6.2.orig.tar.xz` |
| `xxhash-build.link.json` | Link file: `github.com/Cyan4973/xxHash@v0.8.1` → `xxhash_0.8.1.orig.tar.gz` |
| `Packages.gz` | Binary package index, Debian snapshot `20230614T204442Z`, bookworm/main/amd64 |

**Note on versioning:** link file products use the upstream version (`0.8.1`) while
buildinfos carry the full Debian version (`0.8.1-1`). This is correct — orig tarballs
are always named with the upstream version only. The `-1` Debian revision does not
appear in the tarball filename. The parsers strip the revision before generating
tarball artifact IDs, so both sides resolve to the same node.

Snapshot URL used:
```
https://snapshot.debian.org/archive/debian/20230614T204442Z/dists/bookworm/main/binary-amd64/
```

## Reproducing

From `test/output/`, with an empty `astra.db`:

```bash
astra init
astra ingest debian buildinfo ../dataset/lz4_1.9.4-1_amd64.buildinfo
astra ingest debian buildinfo ../dataset/libarchive_3.6.2-1_amd64.buildinfo
astra ingest debian buildinfo ../dataset/xxhash_0.8.1-1_amd64.buildinfo
astra ingest intoto --linker debtrace ../dataset/lz4-build.link.json
astra ingest intoto --linker debtrace ../dataset/libarchive-build.link.json
astra ingest intoto --linker debtrace ../dataset/xxhash-build.link.json
astra ingest debian packages ../dataset/Packages.gz
astra ingest git --tag v1.9.4 https://github.com/lz4/lz4.git
astra ingest git --tag v3.6.2 https://github.com/libarchive/libarchive.git
astra ingest git --tag v0.8.1 https://github.com/Cyan4973/xxHash.git
astra viz -o full.dot
```

## Output (`output/`)

| File | Description |
|---|---|
| `full.dot` | Complete graph export — gitignored, too large to commit (~36 MB, ~64K nodes) |
| `subset.dot` | Focused supply-chain subset — see below |
| `subset.svg` | Rendered SVG of `subset.dot` |

### subset.dot / subset.svg

`subset.svg` is a **focused subset** of the full graph, not a complete view.
The full graph (`full.dot`) contains ~64K artifact nodes and ~127K edges — the
majority are git file artifacts and archive steps for all 63K packages in the
Packages index. Graphviz cannot render a graph that size to explore `full.dot`.

The subset is hand-filtered to show the meaningful provenance chain for the
three packages:

- Git commit artifact for each upstream tag (from the link file materials)
- Debtrace linker step: git commit → source tarball
- Source tarball artifact (the merge point between the linker step and the build step)
- Build step and its output artifacts (.deb binaries, buildinfo file)
- Archive steps (one per binary package) and the Debian snapshot resource
- `principal:Debian`
- Cross-package build dependencies: `liblz4-1` and `liblz4-dev` are shown as
  resources on the libarchive build step, since libarchive's buildinfo lists the
  exact lz4 version that the lz4 build step produced in this same graph

Edge types: `consumes`, `produces`, `carries_out`, `uses`.
