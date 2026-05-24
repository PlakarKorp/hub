# marketplace

Plugin build recipes for the kloset plugin ecosystem, used by `krossbuild` to cross-compile `.ptar` artifacts for each integration.

Layout:

```
{tier}/{kloset-compat-version}/{plugin}/recipe.yaml
```

- `community/` — recipes pointing at [PlakarKorpAgentic/integrations](https://github.com/PlakarKorpAgentic/integrations) (public plugins).
- `enterprise/` — recipes for the full plugin set; public plugins point at the same public repo, internal-only plugins point at [PlakarKorpAgentic/integrations-private](https://github.com/PlakarKorpAgentic/integrations-private).

Each `recipe.yaml`:

```yaml
name: ftp
version: v1.1.1
repository: https://github.com/PlakarKorpAgentic/integrations
```

The aggregated `integrations` / `integrations-private` repos use prefixed tags of the form `<name>/<version>` (e.g. `ftp/v1.1.1`). The builder is expected to check out `<name>/<version>` rather than `<version>` when cloning these repos.
