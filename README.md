<div align="center">

# 🚀 QuorumDB

### A Production-Inspired Distributed Key-Value Database built in Go

*Inspired by Amazon DynamoDB's architecture*

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=for-the-badge&logo=go)
![BadgerDB](https://img.shields.io/badge/Storage-BadgerDB-blue?style=for-the-badge)
![Distributed Systems](https://img.shields.io/badge/System-Distributed-success?style=for-the-badge)
![Status](https://img.shields.io/badge/Status-In%20Development-orange?style=for-the-badge)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)

</div>

---

## 📖 Overview

QuorumDB is a **distributed key-value database** built completely from scratch in **Go**, designed to understand how modern distributed databases such as **Amazon DynamoDB**, **Apache Cassandra**, and **Riak** work internally.

Instead of relying on existing distributed database frameworks, QuorumDB implements core distributed systems concepts manually including:

- Consistent Hashing
- Data Replication
- Cluster Coordination
- Persistent Storage
- Graceful Shutdown
- Production-grade API design

The primary objective of this project is to learn **Distributed Systems Engineering** by implementing real-world database architecture from first principles.

---

# ✨ Current Features

✅ REST API

✅ Persistent Storage using BadgerDB

✅ Consistent Hash Ring

✅ Primary Node Selection

✅ Replica Nodes

✅ Coordinator Architecture

✅ Internal Replication API

✅ Graceful Shutdown

✅ Health Endpoint

✅ Modular Go Project Structure

---

# 🏗 Architecture

```
                 Client
                    │
             HTTP Request
                    │
                    ▼
          ┌──────────────────┐
          │  Coordinator Node │
          └──────────────────┘
                    │
         Hash Key using Ring
                    │
                    ▼
          Find Primary Node
                    │
      ┌─────────────┴─────────────┐
      │                           │
      ▼                           ▼
Store Locally             Replicate Data
                              │
                 ┌────────────┴────────────┐
                 ▼                         ▼
           Replica Node 1            Replica Node 2
```

---

# 🔄 Request Flow

## PUT

```
Client
   │
   ▼
/key
   │
Coordinator
   │
Hash Ring
   │
Primary Node
   │
Store
   │
───────────────
│             │
▼             ▼
Replica1   Replica2
```

---

## GET

```
Client
   │
   ▼
/key
   │
Coordinator
   │
Hash Ring
   │
Read
   │
Return Value
```

---

## DELETE

```
Client
   │
   ▼
/key
   │
Coordinator
   │
Delete Local
   │
───────────────
│             │
▼             ▼
Replica1   Replica2
```

---

# 📂 Project Structure

```
quorumdb/

├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── api/
│   │   └── handlers.go
│   │
│   ├── cluster/
│   │   └── manager.go
│   │
│   ├── hash/
│   │   └── ring.go
│   │
│   ├── storage/
│   │   ├── badger.go
│   │   └── memory.go
│   │
│   ├── replication/
│   │
│   └── config/
│
├── data/
│
├── go.mod
└── README.md
```

---

# ⚙ Tech Stack

| Layer | Technology |
|--------|------------|
| Language | Go |
| Database | BadgerDB |
| Communication | HTTP REST |
| Hashing | Consistent Hashing |
| Storage | Embedded KV Store |
| Architecture | Distributed Systems |

---

# 🌐 API

## PUT

```
PUT /key/{key}
```

Example

```
PUT /key/user1
```

Body

```json
{
    "value":"Rahul"
}
```

---

## GET

```
GET /key/{key}
```

---

## DELETE

```
DELETE /key/{key}
```

---

## Health Check

```
GET /health
```

---

# 🧠 Core Concepts Implemented

- Consistent Hashing
- Key Partitioning
- Replication
- Coordinator Pattern
- Internal Replica API
- Persistent Storage
- HTTP API
- Graceful Shutdown
- Modular Architecture

---


# 💡 Design Principles

- High Availability
- Horizontal Scalability
- Data Replication
- Fault Tolerance
- Modular Components
- Separation of Responsibilities
- Production-grade Code Structure

---

# 📚 Inspired By

- Amazon DynamoDB
- Google Bigtable
- Apache Cassandra
- Riak KV
- MIT 6.824 Distributed Systems

---


<div align="center">

### ⭐ If you found this project interesting, consider giving it a star!

**Building Distributed Systems • One Commit at a Time**

</div>
