# StateLedger REST API Documentation

Complete API reference for the StateLedger HTTP server.

## Base URL

```
http://localhost:8080
```

Configure with `--addr` flag:
```bash
stateledger server --db data/ledger.db --addr :8080
```

---

## Authentication

Current version: **No authentication** (intended for trusted internal networks).

For production deployments, use:
- Kubernetes Network Policies
- Service mesh (Istio/Linkerd)
- API Gateway with auth (Kong/Nginx/Traefik)

---

## Response Format

All responses follow this structure:

```json
{
  "success": true,
  "data": { ... },
  "time": "2026-02-01T12:00:00Z"
}
```

Error responses:

```json
{
  "success": false,
  "error": "Error message",
  "time": "2026-02-01T12:00:00Z"
}
```

---

## Endpoints

### Health Check

**GET /health**

Returns server health status.

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "time": "2026-02-01T12:00:00Z"
  },
  "time": "2026-02-01T12:00:00Z"
}
```

**Status Codes:**
- `200 OK` - Server is healthy

---

### List Records

**GET /api/v1/records**

Retrieve paginated list of ledger records.

**Query Parameters:**
- `limit` (optional): Maximum records to return (default: 100, max: 1000)
- `offset` (optional): Number of records to skip (default: 0)

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/records?limit=10&offset=0"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "records": [
      {
        "id": 1,
        "kind": "code",
        "timestamp": "2026-02-01T12:00:00Z",
        "hash": "abc123...",
        "payload": "{\"repo\":\"example\",\"commit\":\"def456\"}"
      },
      {
        "id": 2,
        "kind": "config",
        "timestamp": "2026-02-01T12:05:00Z",
        "hash": "ghi789...",
        "payload": "{\"source\":\"app.yaml\",\"hash\":\"jkl012\"}"
      }
    ],
    "offset": 0,
    "limit": 10,
    "total": 2
  },
  "time": "2026-02-01T12:10:00Z"
}
```

**Status Codes:**
- `200 OK` - Success
- `500 Internal Server Error` - Database error

---

### Get Record by ID

**GET /api/v1/records/{id}**

Retrieve a specific record by its ID.

**Path Parameters:**
- `id` (required): Record ID (integer)

**Example Request:**
```bash
curl http://localhost:8080/api/v1/records/1
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "kind": "code",
    "timestamp": "2026-02-01T12:00:00Z",
    "hash": "abc123...",
    "payload": "{\"repo\":\"example\",\"commit\":\"def456\"}"
  },
  "time": "2026-02-01T12:10:00Z"
}
```

**Status Codes:**
- `200 OK` - Success
- `400 Bad Request` - Invalid record ID
- `404 Not Found` - Record does not exist

---

### Verify Chain Integrity

**GET /api/v1/verify**

Verify the integrity of the entire hash chain.

**Example Request:**
```bash
curl http://localhost:8080/api/v1/verify
```

**Response (Valid Chain):**
```json
{
  "success": true,
  "data": {
    "valid": true,
    "checked": 1000,
    "failed_id": 0,
    "reason": "",
    "time": "2026-02-01T12:10:00Z"
  },
  "time": "2026-02-01T12:10:00Z"
}
```

**Response (Invalid Chain):**
```json
{
  "success": true,
  "data": {
    "valid": false,
    "checked": 245,
    "failed_id": 246,
    "reason": "hash mismatch at record 246",
    "time": "2026-02-01T12:10:00Z"
  },
  "time": "2026-02-01T12:10:00Z"
}
```

**Status Codes:**
- `200 OK` - Success (check `valid` field for result)
- `500 Internal Server Error` - Database error

---

### Reconstruct Snapshot

**GET /api/v1/snapshot**

Reconstruct system state at a specific point in time.

**Query Parameters:**
- `time` (optional): RFC3339 timestamp (default: current time)

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/snapshot?time=2026-02-01T12:00:00Z"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "time": "2026-02-01T12:00:00Z",
    "records": [
      {
        "id": 1,
        "timestamp": 1738411200,
        "type": "code",
        "source": "git",
        "payload": "{\"repo\":\"example\",\"commit\":\"abc123\"}",
        "hash": "def456...",
        "prev_hash": "000000..."
      }
    ],
    "count": 1
  },
  "time": "2026-02-01T12:10:00Z"
}
```

**Status Codes:**
- `200 OK` - Success
- `500 Internal Server Error` - Database error

---

## Error Handling

### Common Error Responses

**400 Bad Request:**
```json
{
  "success": false,
  "error": "Invalid record ID",
  "time": "2026-02-01T12:10:00Z"
}
```

**404 Not Found:**
```json
{
  "success": false,
  "error": "Record not found",
  "time": "2026-02-01T12:10:00Z"
}
```

**500 Internal Server Error:**
```json
{
  "success": false,
  "error": "database locked: unable to open database file",
  "time": "2026-02-01T12:10:00Z"
}
```

**501 Not Implemented:**
```json
{
  "success": false,
  "error": "Record creation via API not implemented yet",
  "time": "2026-02-01T12:10:00Z"
}
```

---

## Rate Limiting

Current version: **No rate limiting**.

For production:
- Implement rate limiting at API Gateway level
- Use Kubernetes NetworkPolicies for traffic control
- Consider Redis-backed rate limiting middleware

---

## CORS

Current version: **No CORS headers**.

For browser-based clients, add CORS middleware or configure at API Gateway.

---

## Performance

API performance characteristics (see [BENCHMARKS.md](../BENCHMARKS.md)):

| Endpoint | Throughput | Latency |
|----------|-----------|---------|
| GET /health | ~445,000 ops/sec | 2.2µs |
| GET /api/v1/records (50 records) | ~4,843 ops/sec | 206µs |
| GET /api/v1/records/{id} | ~18,233 ops/sec | 54µs |
| GET /api/v1/verify (50 records) | ~5,512 ops/sec | 181µs |
| GET /api/v1/snapshot (100 records) | ~3,181 ops/sec | 314µs |

---

## Client Examples

### curl

```bash
# Health check
curl http://localhost:8080/health

# List first 10 records
curl "http://localhost:8080/api/v1/records?limit=10&offset=0"

# Get specific record
curl http://localhost:8080/api/v1/records/42

# Verify chain
curl http://localhost:8080/api/v1/verify

# Snapshot at specific time
curl "http://localhost:8080/api/v1/snapshot?time=2026-02-01T12:00:00Z"
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
    Error   string      `json:"error,omitempty"`
}

func main() {
    resp, err := http.Get("http://localhost:8080/api/v1/records?limit=10")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    
    var result Response
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        panic(err)
    }
    
    fmt.Printf("Success: %v\n", result.Success)
    fmt.Printf("Data: %+v\n", result.Data)
}
```

### Python

```python
import requests

# Health check
response = requests.get("http://localhost:8080/health")
print(response.json())

# List records
response = requests.get("http://localhost:8080/api/v1/records", params={
    "limit": 10,
    "offset": 0
})
data = response.json()
print(f"Success: {data['success']}")
print(f"Records: {data['data']['records']}")

# Verify chain
response = requests.get("http://localhost:8080/api/v1/verify")
result = response.json()
print(f"Chain valid: {result['data']['valid']}")
```

### JavaScript (Node.js)

```javascript
const fetch = require('node-fetch');

async function listRecords() {
    const response = await fetch('http://localhost:8080/api/v1/records?limit=10');
    const data = await response.json();
    
    console.log('Success:', data.success);
    console.log('Records:', data.data.records);
}

async function verifyChain() {
    const response = await fetch('http://localhost:8080/api/v1/verify');
    const data = await response.json();
    
    console.log('Chain valid:', data.data.valid);
    console.log('Records checked:', data.data.checked);
}

listRecords();
verifyChain();
```

---

## Future Enhancements

Planned for future releases:

- **Authentication**: JWT, OAuth2, API keys
- **Filtering**: Query by type, source, time range
- **Websockets**: Real-time ledger updates
- **Batch Operations**: Bulk record creation
- **GraphQL**: Alternative query interface
- **OpenAPI Spec**: Auto-generated documentation

---

## Support

For issues or questions:
- GitHub Issues: [https://github.com/Retr0-XD/StateLedger/issues](https://github.com/Retr0-XD/StateLedger/issues)
- Documentation: [README.md](../README.md)
- Examples: [examples/](../examples/)
