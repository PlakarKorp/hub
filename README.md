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

- **`{tier}/v1.0.0/`** — recipes point at the legacy per-plugin repositories under [PlakarKorp](https://github.com/PlakarKorp) (e.g. `PlakarKorp/integration-ftp`), which retain flat `<version>` tags. This preserves compatibility with plakar v1.0.x, whose `pkg` does not understand prefixed tags.

- **`{tier}/v1.1.0/`** — recipes point at the aggregated mono-repos:
  - Public plugins → [PlakarKorp/integrations](https://github.com/PlakarKorp/integrations)
  - Internal-only plugins (enterprise tier) → [PlakarKorp/integrations-private](https://github.com/PlakarKorp/integrations-private)

  Both mono-repos use prefixed tags of the form `<name>/<version>` (e.g. `ftp/v1.1.1`). The builder is expected to resolve `<name>/<version>` rather than `<version>` when cloning these repos.

---

## Feature and Bug Bounty Program

Plakar operates a dedicated Feature and Bug Bounty Program to safely collaborate with independent security researchers and open-source developers. This program governs the submission of critical software vulnerabilities and specific roadmap development contributions. The complete program policy, including terms of participation, validation workflows, financial reward grids, and legal safe harbor provisions, can be accessed directly at https://plakar.io/legal-notice/bounty-policy/.

### Integration bounties

Plakar offers paid bounties for high-quality integrations that expand the ecosystem. Each open bounty issue describes exactly what needs to be built, the acceptance criteria, and the payout.

#### Tiers

| Label | Bounty | Scope |
|-------|--------|-------|
| `bounty:tier1` | $1,500 | Advanced integrations — distributed systems, remote orchestration, complex restore flows |
| `bounty:tier2` | $750 | Standard integrations — well-documented tooling, auth support, filter options |
| `bounty:tier3` | $500 | Basic integrations — straightforward dump/restore with a single data source |

#### Priority contributors

We give priority consideration to companies and teams that can commit to both building **and maintaining** integrations over time. A great example is [FactorFX](https://factorfx.com), the team behind our Proxmox integration, who have demonstrated exactly the kind of long-term ownership we value.

If your organization relies on one of the listed technologies and wants to own that integration, reach out — we want to work with you.

#### How to claim a bounty

1. Browse open bounty issues — all tagged `status:open`.
2. **Ask questions before you start.** Comment on the issue to clarify scope, edge cases, or implementation approach. The Plakar core team will respond. Do not submit a PR without aligning on scope first.
3. Once scope is confirmed, comment to claim the issue. We allow one active claim at a time with a 2-week completion window.
4. Submit a PR to [PlakarKorp/integrations](https://github.com/PlakarKorp/integrations) referencing the issue.
5. First merged PR that meets the acceptance criteria wins the bounty.

#### Rules

- **Only the Plakar core team creates bounty issues.** Do not open issues to propose integrations — use [Discussions](https://github.com/PlakarKorp/hub/discussions) instead (see below).
- Bounties are paid upon merge, after a brief review period.
- The core team reserves the right to reject submissions that do not meet the stated requirements.

### Propose a new integration

Have an integration idea that isn't listed? Open a thread in [Discussions](https://github.com/PlakarKorp/hub/discussions). The core team reviews community proposals regularly and may convert popular ones into funded bounty issues.

**👍 Upvote issues and discussions** you want to see prioritized — thumbs-up reactions directly influence which bounties we fund next.
