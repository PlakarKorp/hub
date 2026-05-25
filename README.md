# hub

Plugin build recipes for the kloset plugin ecosystem, used by `krossbuild` to cross-compile `.ptar` artifacts for each integration.

## Layout

```
{tier}/{kloset-compat-version}/{plugin}/recipe.yaml
```

- `community/` — recipes for the open-source plugin set.
- `enterprise/` — recipes for the full plugin set, including internal-only plugins.

Each `recipe.yaml`:

```yaml
name: ftp
version: v1.1.1
repository: https://github.com/PlakarKorp/integrations
```

## Where recipes point

The `repository` field depends on the plakar compatibility version of the recipe:

- **`{tier}/v1.1.0/`** — recipes point at the aggregated mono-repos:
  - Public plugins → [PlakarKorp/integrations](https://github.com/PlakarKorp/integrations)
  - Internal-only plugins (enterprise tier) → [PlakarKorp/integrations-private](https://github.com/PlakarKorp/integrations-private)

  Both mono-repos use prefixed tags of the form `<name>/<version>` (e.g. `ftp/v1.1.1`). The builder is expected to resolve `<name>/<version>` rather than `<version>` when cloning these repos.

- **`{tier}/v1.0.0/`** — recipes point at the legacy per-plugin repositories under [PlakarKorpAttic](https://github.com/PlakarKorpAttic) (e.g. `PlakarKorpAttic/integration-ftp`), which retain flat `<version>` tags. This preserves compatibility with plakar v1.0.x, whose `pkg` does not understand prefixed tags.
