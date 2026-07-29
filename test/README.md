# XZ-Utils Attack Dataset

Study of the xz-utils backdoor (CVE-2024-3094) using AStRAFy.

## Dataset (`dataset/`)

| File | Description |
|---|---|
| `xz-utils_5.6.0-0.2_amd64.buildinfo` | Debian build record for the compromised xz-utils 5.6.0-0.2 release |
| `openssh_9.6p1-5_amd64.buildinfo` | Debian build record for openssh (runtime and build dependency on liblzma) |
| `systemd_255.4-1_amd64.buildinfo` | Debian build record for systemd (runtime and build dependency on liblzma) |
| `Packages.gz` | Binary package index, Debian unstable snapshot `20240315T085855Z` |

The snapshot is from March 15, 2024 — within the window when `5.6.0-0.2` was the
current version in Debian unstable (Feb 26 – March 28, 2024). `liblzma5@5.6.0-0.2`
appears as both the build output and the runtime/build dependency of downstream
packages, with no emergency replacement noise.

## Reproducing

From `output/`, with an empty `astra.db`:

```bash
astra init
astra ingest debian buildinfo ../dataset/xz-utils_5.6.0-0.2_amd64.buildinfo
astra ingest debian buildinfo ../dataset/openssh_9.6p1-5_amd64.buildinfo
astra ingest debian buildinfo ../dataset/systemd_255.4-1_amd64.buildinfo
astra ingest debian packages ../dataset/Packages.gz \
  --archive-url "https://snapshot.debian.org/archive/debian/20240315T085855Z"
astra ingest git --tag v5.6.0 https://github.com/tukaani-project/xz.git
astra viz -o full.dot
python3 gen_subset.py
dot -Tsvg subset.dot -o subset.svg
```

## Output (`output/`)

| File | Description |
|---|---|
| `full.dot` | Complete graph — gitignored (~73K artifacts, 73K steps, ~391K edges) |
| `subset.dot` | Focused attack chain subset |
| `subset.svg` | Rendered SVG of `subset.dot` |
| `gen_subset.py` | Script that extracts the focused subset from `full.dot` |

## What the Graph Shows

`subset.svg` is a **focused subset** showing the attack-relevant supply chain.
The full graph contains all 72K packages from the Debian unstable snapshot — too
large to render directly.

### Supply chain gap

```
artifact:gitcommit:github.com/tukaani-project/xz@2d7d862e...  [no outgoing edges — disconnected]

artifact:tarball:deb/xz-utils@5.6.0  (completeness: incomplete)
  --consumes-->
step:build:deb/xz-utils@5.6.0-0.2
  --produces-->
liblzma5_5.6.0-0.2 / liblzma-dev_5.6.0-0.2 (compromised artifacts)
```

The git commit and the tarball are structurally disconnected: the tarball is
referenced by the buildinfo but has no attested producer. This is the provenance
gap the attack exploited. The malicious content was injected into the upstream
tarball outside of any verifiable build pipeline.

### Runtime blast radius

```
openssh-server --depends--> libsystemd0 --depends--> liblzma5
```

Runtime dependencies (from `Packages Depends:`) are modelled as direct
`depends` edges between artifact nodes. Starting from `liblzma5@5.6.0-0.2`
and traversing reverse `depends` edges gives the set of packages affected at
runtime.

### Build-time compromise

```
pkg:liblzma5@5.6.0-0.2  --carries_out-->  step:build:deb/openssh@1:9.6p1-5
pkg:liblzma5@5.6.0-0.1  --carries_out-->  step:build:deb/systemd@255.4-1
```

The compromised library was also in the build environment of both downstream
packages. Build dependencies (from `Installed-Build-Depends:`) are modelled as
resource nodes with `carries_out` edges to the build step.
