# Official A2UI v0.9.1 schemas (pinned)

Verbatim copies of the official A2UI JSON schemas used by the conformance
test (`conformance_test.go`). They are pinned so the test never depends on
network access or upstream churn.

| File                              | Upstream path (repo `a2ui-project/a2ui`)                                  |
| --------------------------------- | ------------------------------------------------------------------------- |
| `server_to_client.json`           | `specification/v0_9_1/json/server_to_client.json`                         |
| `common_types.json`               | `specification/v0_9_1/json/common_types.json`                             |
| `catalog.json`                    | `specification/v0_9_1/catalogs/basic/catalog.json`                        |
| `example_interactive-button.json` | `specification/v0_9_1/catalogs/basic/examples/00_interactive-button.json` |

- Pinned from commit `29b715fa89fc5bb8351d2ea0116f03d4f2e212f2` (2026-08-17).
- Upstream license: Apache 2.0 (project by Google and contributors,
  <https://a2ui.org/>).
- The files are byte-identical to upstream; the `$id` values intentionally
  keep their upstream `v0_9` URLs even inside the `v0_9_1` directory. The
  conformance test works around the resulting sibling-URL mismatch by
  aliasing the catalog under `.../v0_9/catalog.json` (what
  `server_to_client.json`'s relative `$ref`s resolve to).
- Refresh procedure: re-download from the paths above at a new commit,
  update the commit hash here, and run `go test ./pkg/vision/a2ui/...`.
  The `catalogSignatures` pin test will fail loudly if the component set
  changed.
