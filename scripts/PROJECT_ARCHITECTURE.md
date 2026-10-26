# QuorumDB Living Project Guide

> Status: current implementation reference
>
> Last reviewed: 2026-10-26
>
> Scope: the source tree as it exists today, including uncommitted work in progress
> Maintenance rule: update this document whenever behavior, architecture, configuration, APIs, guarantees, dependencies, tests, or roadmap status changes.

## 1. Purpose of this document

This is the private, living technical reference for QuorumDB. It explains what the project is, how every current package participates in the system, how requests move through a node, what guarantees the implementation does and does not provide, what is incomplete, and how the project can evolve toward an etcd-inspired distributed database.

This file describes actual behavior rather than intended behavior. A feature should not be marked implemented merely because a type, placeholder, comment, or README entry exists.

## 2. Project summary

QuorumDB is an educational distributed key-value database written in Go. The current design is inspired primarily by Dynamo-style databases:

- A consistent hash ring partitions keys.
- Each key is assigned to a primary and intended replicas.
- writes are coordinated by the primary.
- replicas are contacted through internal HTTP endpoints.
- reads query multiple owners and choose the record with the greatest version.
- read and write quorum thresholds are both set to two.
- BadgerDB provides local persistence for each node.
- periodic HTTP health checks maintain a local view of node availability.

The project is not currently an etcd-style database. etcd uses Raft to replicate a single ordered state machine and provides strongly consistent, globally ordered mutations. QuorumDB currently partitions keys and reconciles replica values using local version numbers, which is an eventual-consistency-oriented approach.

## 3. Current status at a glance

### Implemented

- Go server executable.
- Static three-node cluster definition.
- One BadgerDB database directory per node ID.
- Public PUT, GET, and DELETE HTTP operations.
- Internal replica PUT, GET, and DELETE operations.
- Request forwarding for public PUT and DELETE operations.
- Consistent hash ring with 64 virtual positions per configured node.
- Concurrent replica reads and writes.
- Hard-coded read quorum of two.
- Hard-coded write quorum of two.
- Process-local atomic version counter.
- Selection of the greatest observed record version during reads.
- Initial read-repair implementation.
- Periodic node health probes.
- Storage interface and BadgerDB implementation.
- Graceful use of goroutines and wait groups for parallel peer requests.

### Partially implemented

- Read repair exists, but currently sends the stale replica record instead of the selected newest record.
- Hinted handoff has data types only; it has no constructor, enqueue, persistence, delivery, retry, or cleanup behavior.
- In-memory storage exists, but does not implement the `Store` interface and is not concurrency-safe.
- Health response types exist, but the registered health endpoint only returns plain text.
- Versioned records exist, but versions are only process-local and are not durable or globally ordered.
- Quorum logic exists, but replica selection can contain duplicate physical nodes and therefore does not yet represent a valid distinct-node quorum.

### Not implemented

- Raft consensus.
- Leader election.
- Replicated log.
- Write-ahead log owned by QuorumDB.
- Consensus-backed membership changes.
- Cluster-wide revisions.
- Linearizable reads.
- MVCC and historical reads.
- Transactions or compare-and-swap.
- Watches.
- Leases and TTL expiration.
- Tombstone-based distributed deletion.
- Snapshots and restore.
- Automatic rebalancing when nodes join or leave.
- Anti-entropy/Merkle-tree synchronization.
- Completed hinted handoff.
- Authentication, authorization, or TLS.
- Separate client and peer listeners.
- Metrics, tracing, and structured logging.
- Rate limiting or admission control.
- Request-size limits.
- Production-grade timeout configuration.
- Graceful server shutdown and BadgerDB close.
- Automated unit, integration, race, failure-injection, or benchmark tests.
- Docker or container-based cluster environment.

## 4. Repository layout

```text
quorumDb/
|-- cmd/
|   `-- server/
|       `-- main.go               Process entry point and dependency wiring
|-- internal/
|   |-- api/
|   |   |-- handlers.go           Public and internal HTTP operations
|   |   `-- health.go             Health endpoint and unused response types
|   |-- cluster/
|   |   |-- node.go               Node model
|   |   `-- manager.go            Membership view and periodic health checks
|   |-- handoff/
|   |   `-- handoff.go            Hinted-handoff placeholders
|   |-- hash/
|   |   `-- ring.go               Consistent hash ring
|   |-- model/
|   |   `-- models.go             Request, record, and replica response models
|   |-- storage/
|   |   |-- storage.go            Storage interface
|   |   |-- badger.go             Persistent BadgerDB implementation
|   |   `-- memory.go             Incomplete in-memory implementation
|   `-- version/
|       `-- version.go            Process-local atomic counter
|-- data/                          Runtime BadgerDB node data; ignored by Git
|-- scripts/
|   `-- PROJECT_ARCHITECTURE.md    This ignored living reference
|-- README.md                      Public project overview
|-- Taskfile.yml                   Local node commands; currently stale
|-- go.mod                         Module and dependency declarations
|-- go.sum                         Dependency checksums
`-- .gitignore                     Generated/local-file exclusions
```

Runtime directories and compiled `.exe` files may exist locally but are not part of the source architecture.

## 5. Component architecture

Each process is a complete QuorumDB node:

```text
                         QuorumDB node process
+------------------------------------------------------------------+
|                                                                  |
|  net/http routes                                                 |
|      |                                                           |
|      v                                                           |
|  api.Handler ------------------------------------------------+   |
|      |                 |                  |                  |   |
|      v                 v                  v                  v   |
|  HashRing       Cluster Manager     Version Manager     Store   |
|                      |                    |                |     |
|                      v                    v                v     |
|                Health probes       atomic counter      BadgerDB  |
|                                                                  |
+------------------------------------------------------------------+
              | peer HTTP requests to other node processes
              v
       /key/{key}, /replica/{key}, /health
```

There is no separately deployed coordinator. Any node may receive a client request. The receiving node calculates ownership. For writes and deletes, it forwards to the calculated primary when necessary. The primary coordinates storage and replication.

## 6. Process startup

The executable is `cmd/server/main.go`.

Expected invocation:

```powershell
go run ./cmd/server <nodeID> <port>
```

Example:

```powershell
go run ./cmd/server node1 8080
```

Startup sequence:

1. Require two positional arguments: node ID and port.
2. Construct the local `cluster.Node` with `Alive=true`.
3. Construct a hard-coded list containing node1:8080, node2:8081, and node3:8082.
4. Open BadgerDB at `./data/<nodeID>`.
5. Create a `cluster.Manager`.
6. Create a `hash.HashRing`.
7. Create a `version.Manager`.
8. Add all three hard-coded nodes to the manager and ring.
9. Create the API handler with the store, ring, local identity, manager, and version manager.
10. Create and start the cluster health manager.
11. Register routes on Go's default HTTP serve mux.
12. Start `http.ListenAndServe` on the local node port.

Important startup limitations:

- The supplied node ID and port are not validated against the hard-coded node list.
- All peer URLs assume `localhost`, so nodes cannot currently run across machines or containers without changes.
- There is no configuration file or environment-driven cluster configuration.
- There is no explicit `http.Server` with read, write, idle, or header timeouts.
- There is no signal handling or graceful shutdown.
- The BadgerDB handle is never closed explicitly.
- Startup does not restore any process-local version counter.
- Cluster IDs and member IDs do not exist.

## 7. Domain and wire models

### `cluster.Node`

```go
type Node struct {
    ID    string
    PORT  string
    Alive bool
}
```

- `ID` identifies a configured physical node.
- `PORT` is used to construct localhost peer URLs.
- `Alive` is a locally observed health flag.
- Fields have no JSON tags because this type is not currently a public API model.
- `PORT` should eventually become idiomatic `Port` or, preferably, a complete advertised peer URL.

### `model.PutRequest`

```go
type PutRequest struct {
    Value   string `json:"value"`
    Version uint64 `json:"version,omitempty"`
}
```

- Clients provide `value`.
- The primary overwrites `Version` with its next local version.
- Internal replica requests carry the primary-assigned version.
- Empty string values are rejected, so an empty byte sequence cannot currently be stored.

### `model.Record`

```go
type Record struct {
    Value   string `json:"value"`
    Version uint64 `json:"version"`
}
```

This is both the stored value and internal replica-read response. It lacks creation metadata, modification metadata, tombstone state, checksum, timestamp, lease, and origin/coordinator information.

### `model.ReplicaResponse`

Associates a responding node with its returned record so that the read path can identify stale replicas.

### `model.Resposnse`

This type is misspelled, unused, and should be removed or replaced by well-defined API response types.

## 8. Storage subsystem

### Store contract

`internal/storage/storage.go` defines:

```go
type Store interface {
    Put(key string, record model.Record) error
    Get(key string) (model.Record, error)
    Delete(key string) error
}
```

The contract does not currently include:

- `Close`.
- typed not-found errors.
- transactions.
- batches.
- iteration/range scans.
- snapshots.
- version preconditions.
- atomic compare-and-swap.
- context cancellation.

### BadgerStore

`BadgerStore` wraps `*badger.DB`.

Put flow:

1. JSON-marshal a `model.Record`.
2. Start a Badger update transaction.
3. store the encoded record using the raw key bytes.

Get flow:

1. Start a Badger read transaction.
2. retrieve the raw value by key.
3. copy and JSON-decode it inside `item.Value`.
4. return the record or raw Badger error.

Delete flow:

1. Start a Badger update transaction.
2. call `txn.Delete` for the raw key.

Important behavior:

- Badger deletion is idempotent and does not prove that a key previously existed.
- Storage errors leak implementation semantics because they are not translated.
- JSON is readable but adds encoding cost and does not define a durable schema migration strategy.
- No tombstone is stored, so deletion cannot safely reconcile against stale replicas.
- There is no explicit database close method in the wrapper.

### MemoryStorage

The in-memory type currently stores `map[string]string`. Its method signatures differ from `Store`, and it has no mutex. It should eventually become a deterministic, concurrency-safe test double implementing the exact storage interface.

## 9. Consistent hash ring

The ring uses SHA-1 and takes the first eight digest bytes as a big-endian `uint64`.

Adding a node:

1. Generate 64 strings of the form `<nodeID>#<virtual-index>`.
2. Hash each virtual-node string.
3. map the hash to the physical node.
4. append each hash to `sortedHashes`.
5. sort all ring positions.

Looking up a key:

1. Hash the key.
2. scan sorted hashes from the beginning.
3. append ring entries whose hash is greater than or equal to the key hash.
4. stop after three entries.

Intended owner semantics:

```text
owners[0] = primary
owners[1] = replica 1
owners[2] = replica 2
```

Known correctness gaps:

- Lookup does not wrap from the end of the ring to the beginning.
- It selects virtual positions, not distinct physical nodes.
- Adjacent positions can therefore produce the same node more than once.
- It can return zero, one, or two owners, while handlers assume at least one owner.
- Calling `owner[0]` can panic.
- Repeatedly adding the same node duplicates sorted positions.
- Nodes cannot be removed.
- There is no thread safety for runtime membership changes.
- There is no rebalancing or transfer of already stored keys.
- Replication factor is hard-coded to three inside lookup behavior.
- Ring configuration is not versioned, so different processes could route the same key differently.

## 10. Cluster manager and failure detection

The cluster manager stores nodes in `map[string]Node` and supports:

- `AddNode`
- `GetNode`
- `GetAllNodes`
- `IsAlive`
- `SetAlive`

The health manager creates a five-second ticker in a background goroutine. On each tick, it queries every other configured node at `http://localhost:<port>/health`.

Observed behavior:

- An HTTP 200 response marks the node alive.
- A request failure or non-200 response marks it dead.
- The local node is not probed.
- Status changes are printed to standard output every cycle.

Limitations:

- The map is read and written concurrently without a mutex, creating a data race and potential concurrent-map failure.
- `http.Get` has no timeout or request context.
- A slow peer can stall an entire health-check cycle.
- The ticker is never stopped.
- The goroutine cannot be canceled.
- A single failed probe immediately marks a node down.
- There is no failure threshold, backoff, jitter, phi accrual, or gossip.
- Liveness observations are local and may differ between nodes.
- Health state is not membership state and provides no split-brain protection.

## 11. HTTP API

### Route table

| Route | Method | Role | Current success response |
|---|---|---|---|
| `/health` | GET in practice | Liveness | `200 alive` plain text |
| `/key/{key}` | PUT | Public write | `201 ok` |
| `/key/{key}` | GET | Public read | `200 <value>` plain text |
| `/key/{key}` | DELETE | Public delete | `200 Deleted` |
| `/replica/{key}` | PUT | Internal replication/repair | `200 ok` |
| `/replica/{key}` | GET | Internal replica read | JSON `Record` |
| `/replica/{key}` | DELETE | Internal replica delete | `200 deleted` |

General API limitations:

- Keys are taken from the remaining URL path without a proper router.
- Keys inserted into peer URLs are not URL-escaped.
- A slash inside a key is treated as part of the remaining path and has ambiguous semantics.
- Client responses are inconsistent: some plain text and some JSON.
- Error responses use `http.Error` plain text.
- There is no request ID, cluster metadata, member metadata, revision, or trace context.
- No content negotiation or explicit API version exists.
- The health handler does not restrict the HTTP method.
- Internal replica handler has no default method-not-allowed branch.
- Internal endpoints have no authentication and are reachable on the client listener.

## 12. PUT request flow

```text
Client
  |
  | PUT /key/{key} {"value":"..."}
  v
Receiving node
  |
  | hash-ring lookup
  v
Calculated owners
  |
  +-- receiver is not owners[0] --> forward request to primary
  |
  `-- receiver is owners[0]
          |
          | validate body and value
          | allocate local version
          | store locally
          | send concurrent replica PUTs
          | wait for every attempted request
          v
      ack >= 2 ? 201 Created : 503 Service Unavailable
```

Detailed behavior:

1. `HandleKey` strips `/key/` and rejects an empty key.
2. The ring returns intended owners.
3. If the first owner is another node, the original method, body, and headers are forwarded to that node's public key endpoint.
4. The primary reads the complete request body with no size limit.
5. JSON is decoded into `PutRequest`.
6. Missing and empty values are both rejected.
7. The primary calls `version.Next()`.
8. The local record is persisted before remote replication.
9. The version is placed into the peer request body.
10. Alive replicas in `owners[1:]` are contacted concurrently.
11. The primary waits for all contacted replicas rather than returning as soon as quorum is reached.
12. Local persistence counts as one acknowledgement.
13. A total of two acknowledgements returns HTTP 201.

Write semantics and edge cases:

- A returned 503 does not roll back the already committed local write.
- Client retry after a 503 may create another version.
- Duplicate physical owners can incorrectly count as separate replicas.
- A node marked dead is skipped even if it has recovered before the next health probe.
- There is no hinted handoff for skipped or failed replicas.
- There is no idempotency key.
- There is no conditional write or expected-version field.
- Replica PUT blindly overwrites local data even if its supplied version is older.
- Internal callers can supply arbitrary versions.

## 13. GET request flow

```text
Client
  |
  | GET /key/{key}
  v
Receiving node
  |
  | hash-ring lookup
  v
Read all available owners concurrently
  |
  +-- local owner  -> Store.Get
  `-- remote owner -> GET /replica/{key}
  |
  | choose largest Version
  | attempt repair of lower versions
  v
successful reads >= 2 ? return value : 503
```

Unlike PUT and DELETE, GET is not first forwarded to the primary. Any receiving node acts as a read coordinator and contacts the calculated owners.

Detailed behavior:

1. Skip a non-local owner when the local cluster manager marks it dead.
2. Launch one goroutine per remaining owner.
3. Use the local store for self and internal HTTP for other owners.
4. Count only reads that return without error.
5. Select the record with the greatest numeric version.
6. collect successful node/record pairs.
7. After all reads complete, inspect successful lower-version records for repair.
8. Require at least two successful reads.
9. Return only the selected value as plain text.

Known read issues:

- Read repair passes each stale `replica.Record` back to that same replica rather than passing the selected newest record. It therefore does not repair the replica.
- Repair runs before confirming that a read quorum was obtained.
- Repair is synchronous in the client request path.
- Equal versions with different values have no deterministic resolution.
- A process-local version is not a valid cluster-wide ordering mechanism.
- Missing keys are treated as failed reads, not as a versioned absence.
- A key missing everywhere tends to produce 503 instead of 404.
- Internal missing-key errors are returned as 500.
- An empty owner list can lead to incorrect behavior or a panic earlier in routing.
- No read deadline exists.
- A read waits for all launched requests even after quorum is satisfied.

## 14. DELETE request flow

Public DELETE is forwarded to the calculated primary if necessary.

Primary behavior:

1. Delete the local key.
2. Count the local operation as one acknowledgement.
3. Concurrently send DELETE to alive replicas.
4. Wait for every attempted replica operation.
5. Return success when at least two acknowledgements exist.

Known delete issues:

- Badger delete generally succeeds even if the key did not exist, so the intended not-found branch is unreliable.
- Physical deletion has no version.
- There are no tombstones.
- A stale replica or later read repair can resurrect deleted data once repair is corrected.
- Failed deletion replication is not retried.
- Hinted handoff does not record missed deletes.
- A failed quorum does not undo the local deletion.

## 15. Request forwarding

`forwardRequest` sends a request to:

```text
http://localhost:<primary-port>/key/<key>
```

It preserves the method, reuses the incoming request body, clones request headers, and copies upstream response headers, status, and body to the client.

Limitations:

- It does not propagate the incoming request context to the outgoing request.
- It uses `http.DefaultClient` without timeouts.
- It does not prevent forwarding loops caused by divergent ring configurations.
- It does not add a hop count or coordinator header.
- It does not URL-escape the key.
- It exposes upstream response details without a stable error schema.

## 16. Replica protocol

The replica API is an internal HTTP convention, not a formally versioned protocol.

### Replica PUT

- Decodes a `PutRequest`.
- rejects empty values.
- creates a `Record` using the supplied version.
- unconditionally stores it.

It should eventually reject an older record, authenticate the peer, distinguish normal replication from repair, and use an explicit protocol schema.

### Replica GET

- Reads a local record.
- Returns JSON when found.
- Returns HTTP 500 for all errors, including not found.

### Replica DELETE

- Physically deletes the key.
- Does not preserve deletion version information.

## 17. Versioning and conflict resolution

`version.Manager` contains a `uint64` incremented with `atomic.AddUint64`.

Properties it does provide:

- Thread-safe unique increasing numbers within one running process.

Properties it does not provide:

- Persistence across restart.
- Uniqueness across nodes.
- Causal ordering across coordinators.
- Wall-clock ordering.
- Cluster-wide monotonicity.
- Protection against counter reset.
- Conflict detection for concurrent writes.

Consequently, `Version=10` from one process is not necessarily newer than `Version=9` from another, and two nodes can generate the same version for different values.

Possible Dynamo-style solutions include vector clocks, dotted version vectors, hybrid logical clocks with node tie-breaking, or a single authoritative coordinator per partition with durable epochs. The etcd-style solution is to order every mutation through Raft and assign a cluster-wide revision when the committed operation is applied.

## 18. Current consistency model

The intended replication parameters are:

```text
N = 3 replicas
R = 2 successful reads
W = 2 successful writes
```

The usual quorum intersection rule is `R + W > N`, which holds numerically here. The current implementation still cannot claim strong or reliable quorum consistency because:

- Owner selection may contain duplicate physical nodes.
- Versions are not globally ordered.
- Membership/ring views can diverge.
- Internal replica writes accept older records.
- Deletes have no tombstones.
- Failures after local application are not rolled back or durably queued.
- There is no consensus on the identity of the primary.
- Health detection is not proof that a node is safely absent.

Current safest description:

> QuorumDB is an educational, partially implemented Dynamo-style replicated key-value store. It does not yet provide a formally validated consistency or availability guarantee.

It should not currently claim linearizability, strict serializability, conflict-free eventual consistency, production readiness, or split-brain safety.

## 19. Concurrency model

Concurrency currently comes from:

- Go's HTTP server starting a goroutine for each request.
- one background health-check goroutine.
- per-request goroutines for replica reads.
- per-request goroutines for replica writes and deletes.
- mutexes protecting per-request response slices and acknowledgement counters.
- an atomic version counter.

Unsafe shared state:

- `cluster.Manager.nodes` is accessed without synchronization.
- `MemoryStorage.data` is accessed without synchronization.
- `HashRing` has no synchronization if membership is ever changed at runtime.

There is no goroutine limit, peer-request semaphore, backpressure, or cancellation of work once a request is no longer needed.

## 20. Error handling and status semantics

Current notable mappings:

- Empty key: 400.
- Invalid JSON: 400.
- Empty value: 400.
- Local storage PUT failure: 500.
- Forwarding connection failure: 502.
- Failed read quorum: 503.
- Failed write/delete quorum: 503.
- Unsupported public key method: 405.
- Missing replica record: currently 500.

Required future error taxonomy:

- invalid request.
- key not found.
- conflict / failed precondition.
- not leader, with leader hint.
- no quorum / unavailable.
- deadline exceeded.
- request too large.
- unauthorized / forbidden.
- storage corruption.
- compacted revision.
- cluster/member mismatch.

Responses should use a stable JSON or protobuf schema and include a request ID and relevant revision/member metadata.

## 21. Security posture

Current system assumptions:

- All nodes and clients are trusted.
- All communication is localhost HTTP.
- The replica API is not isolated.

Current risks:

- Any caller can write directly to replica endpoints.
- A caller can submit arbitrary replica versions.
- Traffic is unencrypted.
- Nodes do not authenticate peers.
- Clients do not authenticate.
- No authorization policy exists.
- Request bodies are unbounded.
- No rate limiting exists.
- No audit logging exists.

Future direction:

- Separate peer and client URLs/listeners.
- Mutual TLS for peer traffic.
- TLS for client traffic.
- user authentication and role-based permissions.
- bounded request sizes and timeouts.
- validation of key/value sizes.
- audit events for administrative and authentication activity.

## 22. Observability and operations

Present observability consists of `log.Println`, `fmt.Println`, and `fmt.Printf` messages. There are no levels, structured fields, metrics, traces, or health detail.

Recommended future signals:

- Request count, latency, size, and status by operation.
- Replication request latency and failures by peer.
- Quorum successes and failures.
- Current leader, term, commit index, and applied index for Raft.
- Badger/backend size and operation latency.
- WAL fsync duration.
- Snapshot count, size, and duration.
- Watchers, leases, and expired keys.
- Peer health and leader changes.
- Goroutine count and process/runtime metrics.

Health endpoints should eventually distinguish:

- process liveness.
- readiness to serve reads.
- readiness to accept writes.
- cluster quorum health.
- storage corruption or quota alarms.

## 23. Build, dependencies, and local tooling

The Go module is:

```text
github.com/yenuganti/quorumdb
```

BadgerDB v4 is the only direct library used by project source, although it is currently marked indirect in `go.mod`. Transitive Badger dependencies are also listed.

The README's Go badge and `go.mod` Go directive currently disagree and must be reconciled with the actually supported toolchain.

Useful checks:

```powershell
go test ./...
go test -race ./...
go vet ./...
gofmt -w ./cmd ./internal
```

At the time of this review, `go test ./...` and `go vet ./...` pass, but every package reports no test files. Passing currently demonstrates compilation and basic vet checks, not behavioral correctness.

The Taskfile is stale:

- It calls `cmd/node/main.go`, but the executable is under `cmd/server`.
- It defines five node commands while the server configures only three members.

## 24. Testing status and required coverage

### Existing automated coverage

None.

### Unit tests to add first

#### Hash ring

- empty-ring behavior.
- one-node behavior.
- three distinct owners.
- wraparound at the end.
- stable mapping for unchanged membership.
- redistribution bounds after adding/removing a node.
- duplicate AddNode handling.
- replication factor greater than member count.

#### Storage contract

- put/get round trip.
- overwrite.
- missing key.
- deletion.
- empty and binary values after model upgrade.
- concurrent access.
- close behavior.
- corrupted stored data.

Run the same contract suite against memory and Badger implementations.

#### API and coordination

- request validation.
- primary forwarding.
- successful and failed write quorums.
- successful and failed read quorums.
- selection of newest record.
- repair sends the selected newest record.
- peer timeout.
- missing key returns 404.
- delete tombstone propagation.
- context cancellation.

### Integration tests

- Start three nodes on dynamically allocated ports.
- write through every node and read through every node.
- stop one replica and verify expected behavior.
- restart a replica and verify repair/catch-up.
- introduce slow peers.
- introduce dropped and reordered peer messages.
- verify persistence after full restart.

### Consensus tests for the future

- leader election.
- follower and leader failure.
- minority and majority network partitions.
- stale leader rejection.
- proposal before/after commit crash boundaries.
- WAL truncation/corruption.
- snapshot installation and slow follower recovery.
- safe membership changes.

## 25. Immediate correctness backlog

Priority zero means the current prototype can return wrong results, panic, hang, or misrepresent its guarantees.

### P0

- Make ring lookup wrap around and return distinct physical nodes.
- Return explicit errors when no nodes or too few nodes exist.
- prevent `owner[0]` panics.
- Fix read repair to send the newest selected record.
- Add synchronization to cluster manager state.
- Add peer HTTP clients with timeouts and request contexts.
- Implement versioned tombstones for delete.
- Prevent an older replica PUT from overwriting a newer record.
- Define correct not-found behavior.
- Add foundational unit and integration tests.

### P1

- Make the memory store implement `Store`.
- Add `Close` to storage lifecycle.
- Add graceful HTTP and background-worker shutdown.
- Replace hard-coded cluster values and quorum constants with validated configuration.
- Correct Taskfile and README claims.
- Add request-size limits and key URL escaping.
- Add stable response/error models.
- Complete or remove hinted-handoff placeholders.
- Stop waiting for unnecessary peer operations after quorum, while safely managing remaining work.

### P2

- Separate API decoding from distributed coordination logic.
- Separate client and peer transport.
- Add structured logging and metrics.
- Add retry policies with bounded exponential backoff and jitter.
- Add anti-entropy or durable hinted handoff if retaining Dynamo architecture.
- Add authentication and TLS.

## 26. Recommended package evolution

Before Raft, separate transport from core behavior:

```text
internal/
|-- api/             Client protocol adapters only
|-- kv/              Put/Get/Delete application service
|-- consensus/       Proposal, commit, and read-barrier abstraction
|-- state/           Deterministic state-machine commands and results
|-- storage/         Durable backend and MVCC implementation
|-- transport/       Peer and client networking
|-- cluster/         Member configuration and lifecycle
|-- config/          Parsing and validation
|-- observability/   Logging and metrics setup
`-- testutil/        Cluster harness and fault controls
```

Desired dependency direction:

```text
HTTP/gRPC adapter
      |
      v
KV application service
      |
      v
Consensus abstraction
      |
      v
Deterministic state machine
      |
      v
Storage interfaces
```

Networking and storage implementations should depend on core interfaces; the core should not depend on HTTP handlers.

## 27. etcd-inspired target architecture

If the project follows etcd rather than Dynamo, consistent hashing and per-key replica ownership should leave the main data path. Every voting member will maintain the same logical keyspace.

```text
Client
  |
  v
KV API
  |
  +-- linearizable read --> Raft ReadIndex/read barrier
  |
  `-- mutation ----------> leader proposal
                              |
                              v
                         append/persist WAL
                              |
                              v
                      replicate Raft entry
                              |
                              v
                       majority commits entry
                              |
                              v
                    apply deterministic command
                              |
                              v
                        MVCC backend update
                              |
                       watchers receive event
```

Required consensus concepts:

- follower, candidate, leader roles.
- terms and votes.
- election and heartbeat timeouts.
- replicated log indexes.
- majority commit rule.
- leader completeness and log matching.
- persistent hard state.
- proposal forwarding/rejection.
- linearizable read barriers.
- snapshots and log compaction.
- joint/safe membership changes or library-supported equivalents.

Recommended implementation strategy:

- Use `go.etcd.io/raft/v3` for the production-inspired path.
- Implement the transport, WAL integration, application loop, state machine, snapshots, restart recovery, membership API, and testing harness in QuorumDB.
- Optionally build a separate small educational Raft implementation to learn the algorithm without presenting it as production-safe.

## 28. MVCC target data model

A future etcd-style key-value record should conceptually include:

```text
key
value bytes
create revision
modify revision
version within current key generation
lease ID
deletion/tombstone information
```

Cluster revision properties:

- increased once per committed mutating transaction.
- globally ordered within the cluster.
- persisted and restored.
- returned in response metadata.
- used by watches and historical reads.

MVCC features:

- read at latest revision.
- read at a historical revision.
- range and prefix reads.
- atomic multi-operation transactions.
- comparisons against value, version, create revision, or modify revision.
- watch from a revision.
- compaction of old history.
- explicit compacted-revision errors.

## 29. Feature roadmap

### Milestone 1: trustworthy current prototype

- Complete all P0 corrections.
- Add deterministic tests and a three-node integration harness.
- Document exact current guarantees.
- Make local startup reproducible.

### Milestone 2: clean single-node database core

- Refactor service boundaries.
- Implement binary-safe values.
- Add typed errors and stable API responses.
- Add batches/transactions at the storage boundary.
- Add clean lifecycle and observability.

### Milestone 3: Raft replication

- Static three-member bootstrap.
- leader election.
- persistent WAL/hard state.
- command proposal and application.
- restart recovery.
- linearizable writes.
- linearizable reads using ReadIndex or an equivalent safe barrier.

### Milestone 4: snapshot and membership operations

- state-machine snapshot generation.
- Raft snapshot persistence and installation.
- slow follower catch-up.
- member add/remove flow.
- reject reconfiguration that would destroy quorum.

### Milestone 5: MVCC and coordination APIs

- cluster-wide revisions.
- range API.
- transactions and compare-and-swap.
- watch streams.
- compaction.
- leases, keep-alive, and expiration.

### Milestone 6: production-inspired hardening

- peer/client TLS.
- authentication and role-based authorization.
- quotas and alarms.
- backup, restore, and integrity verification.
- Prometheus metrics and structured logs.
- chaos/fault tests.
- performance benchmarks and profiling.
- versioned upgrades and storage migrations.

## 30. Interview narrative

The project should be presented honestly:

1. The first version explored Dynamo concepts: consistent hashing, replicas, quorums, health checks, and read repair.
2. Testing exposed subtle correctness issues such as duplicate physical owners, local-only versions, unsafe deletion, and ambiguous failures.
3. The architecture then moved toward Raft to obtain a single committed order and clear safety guarantees.
4. MVCC revisions enabled compare-and-swap, transactions, reliable watches, and leases.
5. Failure-injection tests demonstrate the guarantees instead of relying on feature claims.

Questions the repository should be able to answer with documentation and tests:

- When is a write considered committed?
- Can an acknowledged write be lost?
- What does a read return during a partition?
- What happens when the leader fails?
- How are duplicate requests handled?
- How does a restarted node recover?
- Why is an odd number of voting members preferred?
- How are stale leaders prevented from committing?
- How are deletes represented and replicated?
- How do watches resume without losing events?
- How are membership changes performed safely?
- Which guarantees are deliberately not provided?

## 31. Definition of done for a highly polished project

The project is interview-ready when:

- Architecture and consistency guarantees are precise and accurate.
- All exported/core behavior has tests.
- `go test -race ./...` passes.
- Static analysis and formatting checks pass in CI.
- A reproducible local cluster starts with one command.
- A leader can fail and the cluster recovers automatically.
- Completed writes survive process and cluster restart according to documented guarantees.
- Linearizable versus serializable/stale reads are explicit.
- Snapshots and restore are demonstrated.
- Network partition tests distinguish majority and minority behavior.
- Metrics make cluster state observable.
- Security assumptions and known limitations are documented.
- Benchmarks state hardware, workload, value size, durability settings, and percentiles.
- README feature claims match executable behavior.

No distributed database is “perfect.” The goal is a small system with explicit invariants, well-defined failure behavior, thorough tests, and evidence supporting every guarantee.

## 32. Document update checklist

Whenever code changes, review the following:

- [ ] Update the “Last reviewed” date.
- [ ] Update implemented, partial, and missing feature lists.
- [ ] Update repository layout for added, moved, or removed files.
- [ ] Update startup/configuration behavior.
- [ ] Update route and wire-protocol documentation.
- [ ] Update PUT, GET, and DELETE flows.
- [ ] Update consistency and durability guarantees.
- [ ] Update concurrency and lifecycle details.
- [ ] Update known limitations and security assumptions.
- [ ] Update test status and commands.
- [ ] Mark completed backlog items and add newly discovered work.
- [ ] Update roadmap milestone status.
- [ ] Check the public README for matching claims.

## 33. Current source-of-truth warning

This guide is intentionally ignored by Git at the user's request. It exists only in this working copy unless backed up separately. Because it is ignored, it will not be included in commits, pushes, clones, pull requests, or normal repository backups. Future development sessions must explicitly inspect and update this file; Git cannot detect when its contents become stale.
