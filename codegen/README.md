<div align="center">

</div>

# SumUp Java SDK Codegen

Custom Go-based generator that reads the repository’s `openapi.json` and emits the Java SDK in a tag-based structure. Each tag becomes its own client and the runtime is kept intentionally small so we can iterate quickly.

## Java SDK

The `generate` command reads `openapi.json` and generates the Java client, grouped API clients, models, and supporting source files. Generate the SDK from the repository root with:

```bash
just generate
```

### CLI flags

- `--spec` (default `../openapi.json`) – path to the OpenAPI document.
- `--output` (default `../src/main/java`) – directory where Java sources are written.
- `--package` (default `com.sumup.sdk`) – base package for generated classes.

The command is idempotent; rerunning it rewrites the generated clients in-place. Continuous Integration runs the same invocation and fails when the working tree is dirty afterward.

## Java Code Samples

The `samples` command generates a deterministic, versioned JSON catalog of Java examples from the same intermediate representation used to generate the SDK. Each catalog entry contains a complete Java program. Named OpenAPI request examples produce separate entries.

Generate a catalog from the repository root with:

```bash
just generate-codesamples
```

The recipe writes `code-samples.json` in the repository root by default. Pass another path as its argument to use a different destination. Every generated program is compiled in Continuous Integration. When an SDK release is published, the release workflow regenerates the catalog from that tag and opens or updates a pull request in `sumup/sumup-developer`; the generated JSON is not committed to this repository.
