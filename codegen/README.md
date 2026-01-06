<div align="center">

</div>

# SumUp Java SDK Codegen

Custom Go-based generator that reads the repository’s `openapi.json` and emits the Java SDK in a tag-based structure. Each tag becomes its own client and the runtime is kept intentionally small so we can iterate quickly.

## Quickstart

```bash
go -C codegen run . generate
```

### CLI flags

- `--spec` (default `../openapi.json`) – path to the OpenAPI document.
- `--output` (default `../src/main/java`) – directory where Java sources are written.
- `--package` (default `com.sumup.sdk`) – base package for generated classes.

The command is idempotent; rerunning it rewrites the generated clients in-place. Continuous Integration runs the same invocation and fails when the working tree is dirty afterward.
