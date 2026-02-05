# Quick Start Guide

This guide walks you through setting up a verifiable configuration loop.

## 1. Build the Tools
First, compile the node, the CLI, and the key generator.
```bash
go build -o bin/vdcs-node ./cmd/vdcs-node
go build -o bin/vdcs-cli ./cmd/vdcs-cli
go build -o bin/key-gen ./cmd/key-gen
```

## 2. Generate Identity
You need a Keypair. The Private Key stays with you (the admin). The Public Key goes to the Node (the authority).

```bash
./bin/key-gen
```
*Copy the output. We will use it below.*

## 3. Start the Node (The Authority)
Start the server, trusting your public key. All changes must be signed by the corresponding private key.

```bash
# Replace <PUB_KEY> with your generated Public Key
./bin/vdcs-node -port 9090 -trusted-keys <PUB_KEY>
```

## 4. Propose a Change (The Admin)
As an admin, you propose a change. You sign it with your Private Key.

```bash
# Replace <PRIV_KEY> with your generated Private Key
./bin/vdcs-cli set \
  -key "database.host" \
  -value "production-db.internal" \
  -author "admin" \
  -priv-key <PRIV_KEY>
```

## 5. Verify & Read (The Application)
As a client application transparently verify the data. The CLI fetches the Merkle Proof and checks it against the Root.

```bash
./bin/vdcs-cli get -key "database.host"
```
You should see: `Verified Value Hash: <HASH>`
