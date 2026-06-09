# Contract change checklist

Use this when adding or changing Podbay contract behavior. Touch only the layers that need to change for your change type.

See [architecture.md](architecture.md) for why `composefile`, `spec`, and `composeimport/emit_types` are separate.

---

## Change types

| Change type | `spec` | `composefile` | `composeimport` translate | `composeimport` emit | `validate` | `deploy` / runtime | `clijson` / CLI |
| --- | --- | --- | --- | --- | --- | --- | --- |
| New `spec.Service` field (runtime only) | yes | — | — | — | if validated | if used at deploy | if JSON exposes it |
| New field with Compose import round-trip | yes | maybe | yes | yes | if validated | if used | if JSON exposes it |
| Compose dialect / parse only | — | yes | — | — | — | — | import errors only |
| Import emit presentation (YAML shape) | — | — | — | yes | — | — | import JSON only |
| Graph / partial deploy semantics | yes | — | — | — | yes | yes | observability cmds |
| Host env expansion field | yes | — | — | — | yes | yes | explain if probed |
| Health model change | yes | maybe | maybe | — | yes | yes | deploy/explain JSON |
| New CLI command or flag | — | — | — | — | maybe | maybe | yes (`cmd/podbay`) |

**Legend:** yes = usually required; maybe = required when import or Compose parity is in scope; — = not typically touched.

---

## Layer responsibilities (quick reference)

### `internal/spec`

- Authoritative `podbay.yaml` model and `spec.Load`
- Dependency graph, profiles, partial selection (`ObservabilityActiveServices`)
- Validation rules consumed by `internal/validate`

### `internal/composefile`

- Parsing Docker Compose YAML (include, extends, profiles, Compose healthcheck, etc.)
- Never used by validate/deploy/diff at runtime

### `internal/composeimport`

- **translate.go** — `composefile.File` → `spec.Contract`
- **emit.go / emit_types.go** — `spec.Contract` → first-pass migration YAML

### `internal/validate` / `internal/deploy` / observability

- Operate on `spec.Contract` after load
- Use `expand.ExpandService` and `spec.ObservabilityActiveServices` for consistent agent-loop semantics

---

## Verification after a contract change

```bash
go test ./internal/spec/... ./internal/composeimport/... ./internal/validate/...
go test ./cmd/podbay/...
```

For import round-trips, run `internal/composeimport/*_test.go` and any relevant `examples/ci-*-demo.sh` scripts.
