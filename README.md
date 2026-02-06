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

### Quick Start
Run the automated runner which builds binaries, generates keys, and starts the node:
```bash
go run ./cmd/vdcs-runner
```

### Manual Setup
If you prefer to run steps manually:

### 1. Build
```bash
go build -o ./bin/vdcs-node ./cmd/vdcs-node
go build -o ./bin/vdcs-cli ./cmd/vdcs-cli
go build -o ./bin/key-gen ./cmd/key-gen
```

### 2. Generate Identity
VDCS uses Ed25519 keys for signing.
```bash
go run ./cmd/key-gen
# Output:
# Private Key (Hex): <PRIV_KEY>
# Public Key (Hex):  <PUB_KEY>
```

### 3. Start the Node
By default, VDCS uses SQLite for storage. Storage is persistent in `./data`.
```bash
# Start with your Public Key as the trusted admin
./bin/vdcs-node -trusted-keys <PUB_KEY>
```

**Options:**
- `-storage file`: Use the legacy flat-file storage instead of SQLite.
- `-port 9091`: Change the gRPC listening port.

### 4. Write Data
```bash
./bin/vdcs-cli set -key "database/host" -value "10.0.0.5" -author "admin" -priv-key <PRIV_KEY>
```

### 5. Verify Data (Client Side)
The client fetches the `RootHash` and verifies the inclusion proof locally.
```bash
./bin/vdcs-cli get -key "database/host"
# Output:
# Trusted Root (Version 1): <ROOT_HASH>
# Verified Value Hash: <VAL_HASH>
```

### 6. Monitor (Optional)
To detect split-view attacks, run a monitor that reports the state to an external service.
```bash
./bin/vdcs-cli monitor -target https://monitor.example.com -interval 1m
```

## Consensus
*Current Version (v1)*: Single trusted log authority.
*Future*: Raft-based consensus for high availability.

## License
MIT
