# Verifiable Distributed Configuration System (VDCS)

**VDCS** is a cryptographically verifiable, append-only configuration store. It allows applications to consume configuration (like feature flags or database hosts) without blindly trusting the control plane.

## Key Features

*   **Trustless**: Clients verify Merkle proofs for every value.
*   **Immutable**: All changes are signed and appended to a hash-linked cryptographic log.
*   **Auditable**: The entire history is tamper-evident and can be audited by any 3rd party.
*   **Fast**: Uses Ed25519 signatures and SHA-256 Merkle Trees.

## Architecture

*   **Node**: The gRPC server that manages the Log and State.
*   **Client**: The CLI (and future SDK) that proposes signed changes and verifies proofs.
*   **Protocol**: A custom Protobuf-based protocol ensuring strict verification.

## Getting Started

### 1. Build

```bash
go build -o bin/vdcs-node ./cmd/vdcs-node
go build -o bin/vdcs-cli ./cmd/vdcs-cli
```

### 2. Run Node

```bash
# Start with a trusted admin public key
./bin/vdcs-node -port 9090 -data ./data -trusted-keys "<YOUR_PUBLIC_KEY_HEX>"
```

### 3. Use CLI

```bash
# Set a value (requires private key)
./bin/vdcs-cli set -key "my-config" -value "true" -author "admin" -priv-key "<YOUR_PRIVATE_KEY_HEX>"

# Get and Verify (verifies Merkle proof automatically)
./bin/vdcs-cli get -key "my-config"
```

## Consensus
*Current Version (v1)*: Single trusted log authority.
*Future*: Raft-based consensus for high availability.

## License
MIT
