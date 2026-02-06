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

## Use Cases

### 1. AI Agent Governance
Store system prompts and safety constraints in VDCS. Agents can cryptographically verify that their instructions haven't been tampered with by a compromised middleman or "jailbroken" via prompt injection in the delivery pipeline.
- *Key*: `agents/finance-bot/v1/system-prompt`
- *Value*: "You are a helpful assistant. Do not output financial advice..."

### 2. Secure Feature Flags
Traditional feature flag services require blind trust. If the flag provider is compromised, they can disable security features or enable backdoors. VDCS ensures that your application only accepts flag states signed by your offline keys.

### 3. Supply Chain Security
Store the SHA-256 hashes of your build artifacts (binaries, docker images). Deployment agents verify the artifact hash against the VDCS record before deploying.
- *Key*: `releases/ios/v1.2.0`
- *Value*: `sha256:8f43...`

### 4. Decentralized Configuration (D-Config)
For distributed systems that need a shared source of truth without a centralized database that everyone trusts blindly.

## Integration & Tailoring

VDCS is built as a modular "Root of Trust".

### Customizing Storage
The `internal/storage` package defines a `Store` interface. You can swap the default SQLite/File storage for:
- **Redis/Etcd**: For higher write throughput.
- **S3/GCS**: for infinite archive storage of the Merkle Log.

### Client Integration
The core protocol is defined in `proto/vdcs.proto`. You can generate clients for any language:
1.  **Generate Code**: `protoc --python_out=. --grpc_python_out=. proto/vdcs.proto`
2.  **Verify Proofs**: Implement the Merkle Path verification logic (see `internal/crypto/merkle.go`) in your target language. We recommend porting the `VerifyInclusion` function for strict client-side validation.

### Public Auditing
For high-stakes environments, you should publish the "Root Hash" to a public ledger (e.g., Ethereum smart contract or Twitter/X bot).
- Use the `monitor` command logic to periodically fetch the latest root and publish it.
- This ensures that history cannot be rewritten even if the VDCS server itself is compromised.

## Limitations & Future Work

### 1. Missing: Fine-Grained Access Control (RBAC)
Currently, **VDCS is binary: You are either a Trusted Admin or you are not.**
Anyone with a trusted private key can modify *any* key in the system.
*   *Impact*: Suitable for single-team projects or microservices, but not for multi-tenant enterprise environments where different teams need write access to only their specific namespaces.

### 2. Achilles' Heel: High Availability
The current implementation runs as a **Single Primary Node**.
*   *The Risk*: If the node goes down, the configuration is read-only (clients can verify cached data) but no new updates can be made until the node is restored.
*   *Mitigation*: The `store` interface allows for distributed backends (like etcd), but the Merkle Tree is currently maintained in-memory on the single leader.

### 3. Secret Management
VDCS optimizes for verification, not confidentiality. Values are stored as plain bytes (or hashes). It does **not** natively encrypt secrets at rest or hide them from read-access clients. Do not store raw API keys unless you encrypt them client-side before sending.

## Consensus
*Current Version (v1)*: Single trusted log authority.
*Future*: Raft-based consensus for high availability.

## License
MIT
