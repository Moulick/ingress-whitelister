# Ingress Whitelister

Kubernetes operator (Kubebuilder) that manages IP whitelist annotations on Ingress resources. Single Go binary, no external services required.

## Cursor Cloud specific instructions

### Prerequisites

- **Go 1.26** must be installed at `/usr/local/go` and on `PATH` (`/usr/local/go/bin`).
- All dev tool binaries (controller-gen, ginkgo, setup-envtest, etc.) are installed into `./bin/` by the Makefile automatically.

### Key commands

All commands are defined in the `Makefile`. Key ones:

| Action | Command |
|---|---|
| Lint | `make fmt` then `make vet` |
| Test | `make test` (runs code-gen, lint, envtest-based Ginkgo suite — 4 tests) |
| Build | `make build` (produces `bin/manager`) |
| Run locally | `make run` (requires a kubeconfig / real or kind cluster) |

### Gotchas

- `make test` automatically downloads envtest Kubernetes API-server + etcd binaries to `./bin/k8s/` on first run; this can take ~60s but is cached afterward.
- `go vet ./...` takes ~30s due to compilation of the `controller-runtime` dependency tree on first invocation; subsequent runs are fast.
- The operator binary (`bin/manager`) requires a Kubernetes cluster to run. Tests use `envtest` (local API server) so no cluster is needed for `make test`.
- The `bin/` directory is gitignored and holds all generated tool binaries plus the built `manager` binary. It is recreated by the Makefile targets.
