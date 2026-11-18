# Trustwright handover

Last updated: 18 Nov 2026 · Stage: **design done, no code yet** · Next: **M1 (model and map)**

Read this first in a new chat. It says what Trustwright is, what has been decided, where everything lives, the rules to follow, and what to build next.

## 1. What Trustwright is

A standalone tool that maps a federated or agent-trust system, finds broken trust rules, proposes fixes, proves each fix, and lets the user apply the one they pick.

**The one design rule:** anything may *propose* a fix (built-in templates, the user's own AI, an outside agent), but only a deterministic checker written in Go *decides*. A fix is accepted only if it passes four gates:

1. the broken rule now holds;
2. every other rule still holds and the project's own tests pass;
3. the new test fails on the old code and passes on the new code;
4. every change in allowed or denied behaviour is listed and expected.

## 2. Links

| What | Where |
| --- | --- |
| Design doc (full spec, the source of truth) | https://claude.ai/artifact/NFRjP27EbnXxMhUP4B8ok2 |
| Trademark sheet | https://claude.ai/artifact/77er9inrAN5MDshcDwRq9R |
| Product page (featured, "In development") | https://altru.dev/apps/trustwright |
| Brand files | `brand/` in this repo, rules in `brand/TRADEMARK.md` |

A new chat can read the design doc with the Claude Docs tools; it is not copied into this repo.

## 3. Decisions made (18 Nov 2026)

| Topic | Decision |
| --- | --- |
| Name | **Trustwright**; learning engine **Svitovyd**. ™ until registered. No domain for now. |
| Licence | Open-core: AGPL-3.0-or-later plus a commercial licence (`LICENSING.md`). Svitovyd, template packs, team features and hosted runs are proprietary. A contributor licence agreement is required before outside code is merged (text not written yet). |
| Hosting | Product page, accounts and payments on altru.dev (WHC). Hosted runs sandboxed **on the existing VPS**, under the rules in section 7. |
| Stack | Go standard library + one vanilla JS page. No frameworks, no npm, no build step. Linux first. |
| AI | Optional. Fixes come first from the no-AI template catalogue (45 templates, 7 families). Bring-your-own model through an OpenAI-compatible adapter (works with local Ollama / llama.cpp), or MCP server mode for outside agents. No `apply_fix` tool: only the user applies. |
| Repo flow | Each candidate fix in its own git worktree on a throwaway branch; sandboxed build and test; side-by-side preview; the chosen fix becomes a branch + pull request with a signed receipt. Main is never touched directly. |
| Business | Free self-hosted engine; paid Pro key (Svitovyd, team rules, history) via Gumroad license keys; template packs; hosted runs; GitHub App / CI check; "fix and prove" audits; commercial licence; free for humanitarian, education and open-source use. |

## 4. Reuse from Altru's other projects

| Need | Reuse |
| --- | --- |
| Code readers and the map | Ripple (`altrudev/Ripple`, `index.html`, CORE section: 21 languages, engine model export of nodes, edges, levers, effects) |
| Before/after comparison | MCP DriftGuard (`altrudev/MCP-DriftGuard`) |
| Signed receipts | Signet (ECDSA P-256 signing from the Zero-Dep PWA Kit) |
| Checker starting point | Code Constructor's Go verification gate |
| Agent permission test system | Lullwire Link house rules (`altrudev/Lullwire-Link`, private) |

## 5. Test systems

**1. Scientific Interoperability Kernel** (`altrudev/scientific-interoperability-kernel`, private, main at `20a51ac`): C, Rust and Python workers exchanging binary frames through a Python coordinator. 54 files; `python3 -m unittest discover -s tests` runs 27 tests in about 14 s, all passing on 18 Nov. Its rules are written in `docs/CONTRACT-V1.md`, `wire/SPEC-V2.md` and `docs/ACCEPTANCE.md`.

Findings Trustwright must reproduce (all checked by hand against the code):

| Finding | Evidence | Template family |
| --- | --- | --- |
| Workers accept unauthenticated frames | `experiments/binary-process/README.md`: no authentication; CRC32 only catches accidental corruption | A: checksum used as authentication |
| Frames from an earlier run can be replayed | `coordinator_v2.py` requires sequence = step number, which restarts every run | A: no replay protection |
| Spec says V2 is Python-only, but C and Rust decoders exist | `wire/SPEC-V2.md` vs `wire/decode_v2.c`, `wire/decode_v2.rs`, `tests/test_native_wire_v2.py` | G: spec says X, code does Y |
| "V2 frames are never silently accepted by the v1 decoder" is untested | `tests/test_wire_v2.py` only tests the reverse | G: claim without a test |

Bugs to plant for break-it mode: remove the NaN check; accept a non-zero padding byte; commit after one reply instead of two; skip the CRC compare.

**2. Lullwire Link house rules** for agent permissions (template family C): wildcard grants, risky tools without approval, unbounded delegation, narrowing scopes from recorded runs.

## 6. Milestones

1. **M1 Model and map:** read the kernel into the graph, draw it with Ripple's renderer, import its three docs as draft rules. *Gate:* every worker, frame type and check is on the map with a source link.
2. **M2 Checker:** rule format, exhaustive graph checks, generated cases run in the sandbox. *Gate:* all four kernel findings reported with a counterexample.
3. **M3 Templates:** families A, D, E, G for C, Rust, Python. *Gate:* the unauthenticated-frames fix passes all four gates; break-it mode catches every planted bug.
4. **M4 Repo flow:** worktrees, preview, branch + PR with receipt. *Gate:* a real PR on a kernel fork, re-checked by its CI.
5. **M5 Agent trust:** family C on Lullwire Link.
6. **M6 AI harness:** only after M3, so AI is measured against templates.
7. **M7 Svitovyd:** shape signatures, shadow → promoted → rollback lifecycle, glowing unknowns. *Gate:* finds a planted problem no rule covers; promotes nothing without approval.
8. **M8 Paid launch:** Pro keys, first template pack, hosted runs on the VPS sandbox. *Gate:* a stranger's repo runs end to end with nothing of Altru's reachable from the sandbox.

## 7. VPS sandbox rules (hosted runs)

The VPS also holds SSH access to the WHC sites, so a kernel-level escape is the main risk. Build hosted runs only like this:

- dedicated unprivileged user (e.g. `twrun`): no login, no sudo, no SSH keys; Altru's home and `~/.ssh` unreadable to it;
- rootless Podman with the gVisor runtime, one throwaway container per run;
- no network inside runs; about 2 CPUs, 4 GB memory, disk quota and time limit per run;
- at most two runs at once through a queue;
- uploaded code deleted after the run;
- a kill switch in the altru.dev admin;
- revisit moving the sandbox to its own host once paid usage grows.

## 8. Where things live

| Thing | Location |
| --- | --- |
| VPS | Desktop Commander device `vps-377a113a`, user `ubuntu`. 8 CPUs, 22 GB RAM, 111 GB free disk after the 18 Nov cleanup |
| altru.dev on WHC | `ssh whc` from the VPS (host alias in `~/.ssh/config`), site root `~/altru.dev` |
| Trustwright product row | `products` table, id **47**, slug `trustwright`, type `app`, featured, sort order 2, status "In development". Icon at `assets/products/trustwright/icon.svg` |
| Products backup taken before adding it | WHC `~/altru-backups/products-before-trustwright-20261118-175210.json` |
| Scripts used | VPS `~/claude-scratch/trustwright/` (`add-product.php`, `shots.py`, `clean.sh`, `save-loose.sh`) |
| Note on the site's PHP | `includes/bootstrap.php` throws under the command line (header code). For read-only DB queries from the shell, load `includes/config.local.php` and open PDO directly, as `add-product.php` does |

## 9. VPS cleanup done on 18 Nov 2026

Freed 39 GB (disk went from 72 GB to 111 GB free).

- Rebuildable caches cleared: uv, pip, Go build cache, three Rust build folders under `~/.cache`, unused Podman images.
- Removed with Altru's approval: `~/federation-radial-20261028`, `~/ddc-radial-final-20261029`, `~/federation-radial-20261029-full`, `~/.gradle`, `~/.cache/huggingface`, `~/.local/opt/Bforartists-5.2.0-Linux` (Blender kept), `~/lumen-convert`, `~/.cache/gitai-go-cache`, `~/.cache/mergetober-venv-161`, `~/.cache/puppeteer`.
- **Saved first** in `~/archive/vps-cleanup-20261118/` (576 KB):
  - `MCP-DriftGuard-audit-ddc-radial-repair-20261028.bundle`: one commit (`e880e2a`) never pushed. Restore with `git fetch <bundle> audit/ddc-radial-repair-20261028` inside an MCP-DriftGuard clone.
  - `Frequency-Conduit-fix-ddc-conduit-expiry-receipt-20261029.bundle`: one commit (`ffd72b2`) never pushed. Same method.
  - `*-loose.tar.gz`: logs, receipts and evidence files that sat outside the repos. Every repo copy itself was clean and already on GitHub.
- **Still open, needs Altru:**
  - System logs (~1.2 GB) need root: `sudo journalctl --vacuum-size=200M`.
  - The VPS GitHub CLI login has expired (`gh auth status` fails), so nothing can be pushed from the VPS until `gh auth login` is run again.
  - Two services were already failing and are unrelated to the cleanup: `mcp-driftguard-monitor` (user service) reports real drift ("verifier exit 2") since at least 17:00; `ddcal-review-notify` exits 1 every minute (logs need root to read).

## 10. Rules for the next session

- Commits are authored as **Val Rukhaylo**, with no AI attribution.
- No Mac or Apple-style UI anywhere.
- Zero dependencies: Go standard library and vanilla JS; vendored files only when unavoidable.
- AgentTrust (`chenglin1112/AgentTrust`) is AGPL-3.0 with a paid commercial licence: use its published ideas (rule floor, shadow → promote → rollback, learning only adds protection, ordered chains), never its code, rule files or test scenarios.
- Never claim a feature works on the product page until it ships; status stays "In development".
- Deploys to altru.dev: back up first, verify live (HTTP 200, one h1, no JS errors, no sideways scroll at 390 px).
- Never run customers' code anywhere except the sandbox in section 7.

## 11. Start here (next steps)

1. Write the rule format and the graph schema as Go types, extending Ripple's engine model JSON (`nodes`, `edges`, `levers`, `effects`).
2. Port the Ripple readers needed for the kernel (C, Rust, Python, Markdown) into a Go reader package, or run Ripple's CORE through a small adapter first.
3. Load the kernel, draw the map, and import the three spec docs as draft rules (M1).
4. Draft the contributor licence agreement text before accepting any outside code.
