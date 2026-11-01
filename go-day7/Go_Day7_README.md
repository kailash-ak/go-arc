# Go Day 7 — URL Shortener System Design

This document contains the Day 7 design, the reasoning behind each decision, and learning notes for the system-design concepts introduced during the exercise.

## Learning objectives

By the end of this day, you should be able to explain:

- How to estimate request rates and storage.
- How to design create and redirect APIs.
- Why a database uniqueness constraint is needed for generated short codes.
- How random Base62 code generation and collision retries work.
- How cache-aside, negative caching and request coalescing work.
- Why cache stampedes happen and how to reduce them.
- How replicas, replication lag and read-after-write consistency affect correctness.
- How to count clicks without turning every redirect into a database write.
- How a denylist/revocation set can prevent disabled links from redirecting.
- Why this design uses 302 with `Cache-Control: no-store`.

---

# Part A — System design

## 1. Estimation

### Requirements

- New URLs: 100 million per month.
- Read-to-write ratio: 100:1.
- Average record size: 500 bytes.
- Retention: 5 years.
- Assumed peak read traffic: 3× average.
- Assume 30 days per month.

### Total URLs over five years

```text
5 × 12 = 60 months
100 million × 60 = 6 billion URLs
```

### Average writes per second

```text
Seconds/month = 30 × 24 × 60 × 60
              = 2,592,000

Writes/sec = 100,000,000 ÷ 2,592,000
           = 38.58
           ≈ 39 writes/sec
```

### Average reads per second

```text
Reads/sec = 39 × 100
          = 3,900 reads/sec
```

Using the unrounded write rate gives approximately 3,858 reads/sec. The rounded estimate is sufficient for this high-level exercise.

### Peak reads per second

Assume peak traffic is three times average:

```text
Peak reads/sec = 3,900 × 3
               = 11,700 reads/sec
```

### Raw storage

```text
6,000,000,000 records × 500 bytes
= 3,000,000,000,000 bytes
≈ 3 TB (decimal)
```

This is raw record storage only. Indexes, metadata, replicas, backups, logs and growth headroom increase actual provisioned storage.

| Metric | Estimate |
|---|---:|
| URLs created/month | 100 million |
| URLs over 5 years | 6 billion |
| Average writes/sec | ~39 |
| Average reads/sec | ~3,900 |
| Assumed peak reads/sec | ~11,700 |
| Raw storage | ~3 TB |

### Notes: estimation

Capacity estimation is a way to get an approximate scale before choosing infrastructure. It is not a benchmark or a guarantee that a particular database can handle the workload.

Keep these distinctions clear:

- **Average traffic** is the total number of operations divided by the time period.
- **Peak traffic** is a higher rate during busy periods. Here, 3× is an explicit assumption.
- **Raw storage** is just record bytes. Production storage also includes indexes, metadata, replication, backups and operational headroom.
- Reads and writes are different workloads. A read-heavy service may benefit from caching and read replicas, but those components should be justified by latency, availability and measured capacity.

## 2. API

### Create a short URL

`POST /shorten`

Request:

```json
{
  "longUrl": "https://example.com/articles/some-long-article"
}
```

Success: **201 Created**

```json
{
  "shortCode": "a8K2pQz",
  "shortUrl": "https://sho.rt/a8K2pQz"
}
```

`201 Created` is appropriate because this endpoint creates a resource.

Input validation should:

- Require a syntactically valid URL.
- Allow only supported schemes, typically `http` and `https`.
- Reject malformed URLs and unsupported schemes such as `javascript:`.
- Apply a maximum URL length.
- Apply per-client rate limits and abuse controls.

If the service fetches or previews submitted URLs, it must additionally defend against SSRF by restricting private, loopback, link-local and internal destinations and validating DNS resolution. A redirect-only service should not fetch the destination itself.

Possible errors:

- `400 Bad Request` — malformed or unsupported URL.
- `429 Too Many Requests` — client rate limit exceeded.
- `503 Service Unavailable` — mapping cannot be persisted.

### Redirect to the original URL

`GET /{shortCode}`

Example: `GET /a8K2pQz`

Success: **302 Found**

```http
HTTP/1.1 302 Found
Location: https://example.com/articles/some-long-article
Cache-Control: no-store
```

Possible errors:

- `404 Not Found` — code never existed or is not disclosed.
- `410 Gone` — code existed but has intentionally been disabled or permanently removed.
- `429 Too Many Requests` — rate limit exceeded.
- `503 Service Unavailable` — destination cannot be looked up reliably.

### Notes: HTTP status codes

- `201 Created` means the server successfully created a resource.
- `302 Found` tells the client to make a temporary redirect to the URL in the `Location` header.
- `404 Not Found` means the requested resource was not found.
- `410 Gone` communicates that the resource existed but has been intentionally and permanently removed.
- `429 Too Many Requests` indicates rate limiting.
- `503 Service Unavailable` indicates that the service cannot currently handle the request.

The create endpoint and redirect endpoint have different jobs. Creating a short link is not itself a redirect, so `POST /shorten` returns `201`, not `301` or `302`.

## 3. Data model

Use a relational table named `urls`.

| Column | Type | Purpose |
|---|---|---|
| `short_code` | `VARCHAR(7)` | Public short code; primary key |
| `long_url` | `TEXT` | Original destination |
| `created_at` | `TIMESTAMP` | Creation time |
| `expires_at` | `TIMESTAMP NULL` | Optional expiry |
| `is_active` | `BOOLEAN` | Whether the link is enabled |

**Primary key: `short_code`.** Redirects look up records by this value, so it is a natural unique indexed key. The database enforces uniqueness.

Short codes are immutable: once created, a code always maps to its original destination. If a user wants a different destination, create a new short code. A link can be disabled or expire, but its destination mapping is not reassigned to another URL.

### Click analytics

Do not synchronously update a database `click_count` column for every redirect. At roughly 3,900 average reads/sec, that could create thousands of additional database writes per second. Popular links could also create hot rows.

Instead:

- Increment a Redis counter using `INCR`, or append click events to an event log/queue.
- Periodically flush aggregated counts to the database in batches.
- Treat analytics as eventually consistent if a small reporting delay is acceptable.

### Notes: primary keys and unique constraints

A primary key uniquely identifies a row. A unique constraint prevents duplicate values in a column or combination of columns.

For this design, `short_code` is the primary key because the redirect operation looks up the record by short code.

Do not rely on application code that first checks whether a code exists and then inserts it. Two servers can both check at the same time, both see that the code is unused, and both attempt the insert. This is a race condition.

The database's unique constraint is enforced atomically. One insert succeeds; the other receives a duplicate-key error. The application can then generate another code and retry.

## 4. Code generation and collision handling

### Selected strategy: seven random Base62 characters

Alphabet:

```text
0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
```

There are 62 possible characters per position.

```text
62^6 = 56,800,235,584
62^7 = 3,521,614,606,208
```

Seven characters provide approximately 3.52 trillion possible codes.

Generate seven characters using a cryptographically secure random generator. Attempt to insert the code. The database primary-key constraint is the final protection against duplicates. If insertion fails specifically because of a duplicate-key conflict, generate another code and retry. Other database errors must be handled as actual failures, not as collisions.

### Collision probability at 6 billion existing codes

For a randomly generated candidate, approximate collision probability:

```text
p = existing codes / total code space
```

For seven characters:

```text
p = 6,000,000,000 / 3,521,614,606,208
  ≈ 0.001704
  ≈ 0.17%

Expected attempts ≈ 1 / (1 - p)
                 ≈ 1.002 attempts
```

For six characters:

```text
p = 6,000,000,000 / 56,800,235,584
  ≈ 0.1056
  ≈ 10.6%

Expected attempts ≈ 1 / (1 - p)
                 ≈ 1.12 attempts
```

Six characters have enough theoretical combinations for six billion URLs, but random generation would encounter a collision on roughly 10.6% of candidate attempts at that occupancy. Seven characters reduce the retry probability substantially, so seven is the chosen length.

These are approximate per-candidate collision calculations at 6 billion occupied codes, not a guarantee that collisions never occur. The unique constraint and retry are still required.

### Notes: Base62 and random generation

Base62 is a number representation using 62 symbols. It is compact because each character can represent one of 62 values.

With random generation, each character is selected from the alphabet using a secure random source. The generated string is a candidate, not a guarantee of uniqueness.

A collision occurs when the candidate code is already assigned to another URL. The safe process is:

1. Generate a candidate.
2. Attempt the database insert.
3. If the database reports a duplicate-key conflict, generate a new candidate.
4. Retry with a reasonable retry limit and proper error handling.
5. If another database error occurs, stop and report that error.

A cryptographically secure random generator makes codes difficult to predict. The database uniqueness constraint guarantees that two records cannot own the same code.

## 5. Read path, caching and abuse protection

For `GET /a8K2pQz`:

1. The request reaches the URL-shortener service/load balancer.
2. Apply per-IP or per-client rate limiting.
3. Extract the short code.
4. Check the Redis revocation set for disabled codes.
5. If disabled, return `410 Gone` and do not redirect.
6. Check the positive URL cache.
7. **Positive cache hit:** retrieve the destination and return the redirect only if the code has not been revoked and the cached entry is within its validity period.
8. **Cache miss:** use request coalescing (for example, Go's `singleflight`) so concurrent requests for the same code share one in-flight database lookup per application process.
9. Query the authoritative database/primary when required.
10. If a valid active mapping exists, populate Redis with a bounded TTL and return `302 Found` with the destination in `Location`.
11. If the code does not exist, cache a negative result briefly and return `404 Not Found`.
12. Record click analytics asynchronously using Redis `INCR` or an event log; do not write the URL row on every redirect.

### Positive caching

A positive cache stores a known mapping:

```text
Key:   url:a8K2pQz
Value: https://example.com/article
TTL:   configured bounded duration
```

Redis sits between the Go application and the database. The browser does not connect directly to Redis. The Go backend checks Redis first, and queries the database on a cache miss.

### Negative caching

Cache a missing-code result for a short TTL, such as 30–60 seconds:

```text
Key:   missing:xyz9999
Value: NOT_FOUND
TTL:   30–60 seconds
```

This prevents repeated database lookups for the same nonexistent code. When `POST /shorten` successfully creates a code, invalidate any negative-cache entry for that code. Keep the TTL short because a code might be created after an earlier miss.

### Request coalescing and cache stampede protection

If many requests for the same uncached code arrive together, use Go's `singleflight` (`golang.org/x/sync/singleflight`) to let one request perform the database lookup while concurrent requests for that key wait for and share its result.

Example:

```go
var group singleflight.Group

func lookup(code string) (any, error) {
    result, err, shared := group.Do(code, func() (any, error) {
        // Query the database once for this in-flight key.
        return databaseLookup(code)
    })

    _ = shared // true when this caller shared another call's result
    return result, err
}
```

This is process-local. With multiple application instances, each instance can still perform one lookup for the same key. A distributed coordination mechanism may be considered if this becomes a measured bottleneck.

A **cache stampede** happens when many requests miss the cache at once and all query the database for the same item. Request coalescing reduces duplicate work for the same key.

### Many different random missing codes

Negative caching and singleflight help when requests repeat the same key. They do not help much if an attacker sends a million different random codes.

Mitigations:

- Apply per-IP/per-client rate limiting.
- Consider a Bloom filter containing valid short codes before querying the database.
- A Bloom filter can say a code is definitely absent, avoiding a database lookup.
- A positive Bloom-filter result means “possibly present” and must still be checked in the database because Bloom filters can have false positives.
- Keep the Bloom filter updated when codes are created and rebuild/recover it safely after data changes or failures.

### Notes: Redis and HTTP caching are different

**HTTP caching** is controlled by HTTP headers and browser/proxy/CDN caching rules. It can happen in the browser or in intermediary HTTP caches.

**Redis caching** is application-level caching. Redis is a separate in-memory data store that the Go backend calls. It sits between the Go service and the database.

For this service:

```text
Browser
   |
   v
Go URL-shortener backend
   |
   v
Redis cache
   |
   | cache miss
   v
Database
```

The browser never reads the Redis cache directly. `Cache-Control: no-store` controls HTTP caches; it does not delete or disable entries in Redis.

## 6. Failure handling, revocation and consistency

### Database unavailable

- If the mapping is already in Redis, the service may still redirect using the cached value, provided it is within its allowed validity period and the revocation state can be checked.
- On a cache miss, the service cannot reliably retrieve the mapping; return `503 Service Unavailable`, not a misleading `404`.
- New `POST /shorten` requests cannot safely create mappings while persistence is unavailable; return a service error.
- Use database replication and a tested failover strategy to reduce downtime. Backups support recovery but do not replace live failover.

### Cache unavailable

- Bypass the positive URL cache and read from the database if the database is healthy.
- For security-sensitive revocation checks, do not blindly redirect if the revocation state cannot be verified. Fail closed or use a trusted durable fallback.
- Do not treat a cache error as “short code not found.”
- Resume cache operations when Redis recovers.

### Replica lag and newly created codes

A code may be committed to the primary, then immediately requested while a read replica has not yet received the write. The replica can temporarily return no row, causing a false 404.

Possible protections:

- Route a newly created code's immediate reads to the primary for a short read-after-write window.
- Use session/sticky routing or a consistency token so the creating client reads from the primary until replication catches up.
- Wait for the required replication acknowledgement before reporting creation success, if the database supports the required durability/consistency level.
- Do not negative-cache a replica miss as a definitive 404 when replication lag is possible. Retry against the primary or return a temporary error if the authoritative result cannot be confirmed.

### Revocation set / denylist for disabled links

A positive Redis cache may still contain a URL after an administrator disables it. Deleting the positive cache key is useful, but by itself it has a race:

1. Request A reads an active row from the database.
2. Admin disables the link in the database.
3. Admin deletes the Redis URL cache key.
4. Request A finishes and writes its old active result into Redis.
5. A later request reads the stale cache entry.

A shared Redis Set of disabled short codes provides an additional revocation check.

Example Redis commands:

```redis
SADD disabled_codes abc1234
SISMEMBER disabled_codes abc1234
SREM disabled_codes abc1234
```

- `SADD` adds a code to the denylist.
- `SISMEMBER` checks whether a code is disabled.
- `SREM` removes a code from the denylist when it is re-enabled.

Disable workflow:

1. Persist `is_active = false` in the authoritative database.
2. Add the short code to `disabled_codes`.
3. Delete its positive URL cache key.
4. Confirm the revocation has been published before reporting the disable operation as complete.
5. Redirect requests check the denylist before using a positive cache entry. If present, return `410 Gone`.

The denylist check prevents a stale positive cache write from reviving a disabled link, as long as the revocation state is reliably available and checked. A request that already passed its revocation check immediately before the admin disables the link can still race; strict instantaneous revocation across in-flight requests requires stronger coordination.

### Revocation scenarios to test

| Scenario | Expected behaviour |
|---|---|
| Link is cached; admin disables it | The next redirect checks the set and returns `410 Gone`, not a redirect |
| Redis restarts and the set is empty | Rebuild the set from `is_active = false` rows before serving traffic that relies on revocation checks |
| Admin re-enables the link | Remove it from the set; the next redirect can proceed after checking the database/link state |
| Redis is unavailable | Do not blindly trust a positive cache entry for a security-sensitive redirect; fail closed or use a trusted fallback |

Redis persistence and recovery should be configured, but the database remains the source of truth. After Redis restart, rebuild the denylist from disabled rows and verify completion before relying on it.

### Notes: Go map versus Redis Set

A Go map is local to one process:

```go
var disabledCodes = map[string]bool{
    "abc1234": true,
}
```

If there are three Go backend instances, each has its own independent map. Updating the map in one process does not update the others.

A Redis Set is shared external state. All backend instances can check the same set. Redis also provides set operations such as `SADD`, `SISMEMBER` and `SREM`.

A Go map can be very fast and useful for local state, but it is not a shared source of truth across multiple backend servers. Redis adds a network dependency, so the application must define what happens if Redis is unavailable.

## 7. Redirect code: 302 Found

Choose **302 Found** for this shortener.

The reason is not that a code's destination can be updated: short codes are immutable in this design.

A 301 can be cached by browsers and intermediary caches, potentially bypassing the shortener on later visits. That can:

- Prevent the service from counting every redirect request.
- Make it harder to stop future visits to a deleted, disabled or expired link, because a cached redirect may still be used.

For this design, return:

```http
HTTP/1.1 302 Found
Location: https://example.com/new-destination
Cache-Control: no-store
```

`no-store` tells HTTP caches not to store the redirect response. This gives the service an opportunity to count each visit and enforce link disablement or expiration on future requests.

The cost is that every visit must contact the shortener service, increasing request load and potentially latency.

**Important:** `no-store` controls browser and intermediary HTTP caching. It does not invalidate Redis. Redis must be handled separately through TTLs, explicit invalidation and the revocation set.

### Notes: 301 versus 302

- `301 Moved Permanently` communicates a permanent redirect. Browsers and intermediary caches may retain it according to caching rules.
- `302 Found` communicates a temporary redirect. It is a better fit when the shortener wants to remain in the redirect path and make future decisions.
- A 302 can also be cached if caching headers permit it. Therefore, this design explicitly sends `Cache-Control: no-store`.
- `no-store` does not erase a redirect that a browser cached before receiving this policy.

## First-principles questions

### 1. How many Base62 characters are needed for 6 billion URLs?

```text
62^6 = 56,800,235,584       (~56.8 billion)
62^7 = 3,521,614,606,208    (~3.52 trillion)
```

Six characters have enough theoretical capacity for six billion codes. But at six billion occupied codes, random generation has about a 10.6% chance of a collision on each candidate attempt. Seven characters reduce that to about 0.17%, so seven is the better choice.

### 2. What does the Day 6 301 with `Cache-Control: max-age=2592000` imply?

2,592,000 seconds is 30 days. A cache may reuse that redirect while it is fresh, potentially bypassing the shortener for later visits. That can reduce click counting and delay enforcement of a disabled or expired link. This design uses 302 with `Cache-Control: no-store`.

## Architecture overview

```text
POST /shorten
Client ---> URL Shortener ---> Generate random Base62 code
                    |
                    v
              Primary DB
          (unique short_code)
                    |
                    v
              201 Created


GET /{shortCode}
Client ---> URL Shortener ---> Rate limiter
                    |
                    v
            Redis revocation Set
                    |
                    v
              Positive cache
                    |
             cache miss only
                    v
          singleflight per key
                    |
                    v
           Authoritative DB
                    |
              populate cache
                    |
                    v
             302 + Location
                    |
                    v
              Original URL

Redirect analytics ---> Redis INCR / event log ---> batched DB aggregation
```

## Final design decisions

| Concern | Decision |
|---|---|
| Code generation | Seven random Base62 characters |
| Collision protection | Primary-key uniqueness; retry on duplicate-key conflict |
| Primary key | `short_code` |
| Mapping immutability | A code always maps to its original destination |
| Positive cache | Redis, cache-aside |
| Negative cache | Short TTL; invalidate when a code is created |
| Stampede protection | Go `singleflight`, process-local |
| Many random misses | Rate limiting; consider Bloom filter |
| Click analytics | Redis counter or event log, batch persistence |
| Disabled links | Shared Redis Set denylist, backed by database state |
| Source of truth | Database |
| Create success | `201 Created` |
| Redirect status | `302 Found` |
| Redirect HTTP cache policy | `Cache-Control: no-store` |
| Disabled-link response | `410 Gone` |
| Expiration | Optional `expires_at`; no expiry by default |
| Replica consistency | Primary/read-after-write handling for recent creates |
