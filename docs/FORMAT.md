# Trustwright formats

Two JSON files describe a project: the **graph** (what exists and how it connects) and the **rules** (what must hold). Both are read with unknown keys treated as errors, and both report every problem at once.

## Graph (`model` package, engine `trustwright/0.1`)

The graph extends Ripple's engine model export, so a model copied from Ripple (engine `ripple/0.1`) loads unchanged. Writing a graph always marks it `trustwright/0.1`.

```json
{
  "engine": "trustwright/0.1",
  "source": { "type": "local", "name": "kernel", "url": null, "commit": "20a51ac" },
  "nodes": [
    { "id": "worker:c", "kind": "worker", "type": "c", "label": "C worker",
      "src": [ { "path": "workers/worker.c", "line": 1, "end": 120 } ],
      "attrs": { "lang": "c" } }
  ],
  "edges": [ { "from": "worker:c", "to": "frame:step", "kind": "accepts" } ],
  "levers": [], "effects": []
}
```

| Part | From Ripple | Added by Trustwright |
| --- | --- | --- |
| node | `id`, `kind`, `type`, `group`, `health` | `label`, `src`, `attrs` |
| edge | `from`, `to`, `kind`, `why` | `src`, `attrs` |
| lever, effect, snapshot, notes | all fields | nothing |

**Node kinds:** Ripple's `file`, `package`; Trustwright's `worker`, `coordinator`, `frame`, `check`, `agent`, `tool`, `server`, `key`, `spec`, `test`.

**Edge kinds:** Ripple's `imports`, `uses`, `runs`, `loads`, `links`, `copies`, `suggested`; Trustwright's `sends`, `accepts`, `performs`, `checks`, `grants`, `delegates`, `signs`, `implements`, `documents`, `tests`.

Other kinds are allowed; these are the ones the readers and templates use.

**Source links** (`src`) are `path` (relative, forward slashes, never `..`), optional `line` and `end` (1-based, inclusive), optional `commit` (defaults to the graph's) and an optional short `quote`. The M1 gate is that `Graph.Unsourced("worker", "frame", "check")` is empty.

## Rules (`rules` package, format `trustwright.rules/1`)

```json
{
  "format": "trustwright.rules/1",
  "rules": [
    {
      "id": "frames.authenticated",
      "title": "Workers authenticate frames",
      "claim": "Every worker that accepts frames checks who sent them before using them.",
      "status": "draft",
      "severity": "high",
      "family": "A",
      "src": [ { "path": "experiments/binary-process/README.md" } ],
      "check": {
        "op": "each-has-edge",
        "subject": { "kind": "worker" },
        "edge": "performs",
        "object": { "kind": "check", "attrs": { "purpose": "authenticate" } }
      }
    }
  ]
}
```

- `id`: lower-case words joined by `-`, `_` and `.`.
- `status`: `draft` (imported or proposed, reported but never blocks), `active` (confirmed, a break is a finding), `off`.
- `severity`: `high`, `medium`, `low`.
- `family`: the template family (`A`–`G`) that can fix a break. Optional.

**Selectors** pick nodes by `id` (a pattern such as `worker:*` is allowed), `kind`, `type`, `group` and `attrs`. Every field that is set must match; an attribute value of `*` means "present, any value". A rule's subject may not be empty.

**Check operations:**

| `op` | Holds when | Uses |
| --- | --- | --- |
| `each-has-edge` | every subject has an `edge` to an object (`direction: "in"`: from an object) | `edge`, `object`, `direction` |
| `no-edge` | no subject has an `edge` to (or from) an object | `edge`, `object`, `direction` |
| `each-attr` | every subject has every attribute in `want` (`*` = any value) | `want` |
| `all-paths-via` | every path from a subject to an object, along `edge` (any kind if empty), passes through a `via` node | `object`, `via`, `edge` |
| `same-set` | the values of attribute `key` on subjects equal those on objects | `object`, `key` |

Fields an operation does not use are rejected. Examples of every operation are in `testdata/rules-example.json`.
