# Security Policy

## Supported Versions

Podbay is currently in public preview. The first public release is `v2026.5.0`; the latest public preview is `v2026.7.0`.

Security fixes are prioritized for the latest public preview release. Until a future `v2026.x-stable` or `v1.0` compatibility commitment, the `podbay.yaml` contract, CLI behavior, and receipt formats may still evolve between releases.

## Reporting a Vulnerability

Please report suspected security vulnerabilities privately by emailing security@leveluplabs.ai.

Include:

- affected Podbay version or commit
- operating system and Podman version
- the relevant `podbay.yaml` shape, with secrets removed
- steps to reproduce
- impact assessment, if known

Do not open a public issue for vulnerabilities involving secret exposure, command execution, host filesystem access, or container escape concerns.

## Trust Boundary

Podbay shells out to `podman` as the invoking user. It is not a sandbox for untrusted contracts.

A `podbay.yaml` contract can describe builds, images, commands, environment, ports, mounts, volumes, networks, and host paths. Review contracts before running `podbay validate`, `podbay deploy`, `podbay down`, or related commands on a real host.

Operators remain responsible for:

- securing the host and Podman installation
- reviewing container images and build contexts
- protecting secrets and mounted files
- enforcing network and firewall policy
- testing workloads before production use

Podbay receipts and JSON output are evidence artifacts. They do not provide cryptographic attestation, SBOM provenance, automatic rollback, or isolation from malicious workloads.
