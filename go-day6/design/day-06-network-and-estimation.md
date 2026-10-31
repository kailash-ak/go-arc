# Day 6 — URL Shortener: Capacity and Storage Estimation

## Problem statement

Estimate the basic capacity requirements for a URL-shortening service:

- New URLs: **100 million per month**
- Read-to-write ratio: **100 reads per write**
- Record size: **500 bytes**
- Retention period: **5 years**
- Peak traffic assumption: **3× average read traffic**

These are planning estimates, not a final production capacity plan.

## 1. Average writes per second

Assume 30 days per month.

```text
30 × 24 × 60 × 60 = 2,592,000 seconds/month
100,000,000 ÷ 2,592,000 = 38.58 writes/second
≈ 40 writes/second
```

## 2. Average reads per second

There are 100 reads for every write:

```text
40 × 100 = 4,000 reads/second
```

Approximate average workload:
- Writes: **40 requests/second**
- Reads: **4,000 requests/second**

The workload is read-heavy, so read optimization matters.

## 3. Peak read traffic

Assuming peak traffic is three times the average:

```text
4,000 × 3 = 12,000 reads/second
```

The service should be evaluated for approximately **12,000 reads/second at peak** under this assumption. A real design should use measured traffic patterns and a defined peak window.

## 4. Number of records after five years

```text
5 × 12 = 60 months
100,000,000 × 60 = 6,000,000,000 records
```

That is **6 billion records**, assuming the service creates 100 million records every month and retains each for five years.

## 5. Raw storage requirement

At 500 bytes per record:

```text
6,000,000,000 × 500
= 3,000,000,000,000 bytes
≈ 3 TB (decimal)
```

Therefore, estimated **raw record storage is approximately 3 TB**.

This excludes:
- Primary and secondary indexes
- Database metadata and row/document overhead
- Replicas
- Backups and snapshots
- Write-ahead or transaction logs
- Temporary space for compaction and maintenance
- Growth headroom

Actual provisioned storage will be higher than the raw estimate.

## 6. Can one database server handle this?

The estimated average write rate is about 40 writes/second, and the assumed peak read rate is about 12,000 reads/second.

A suitably configured database server might handle this workload, but the estimates alone cannot prove it. Performance depends on:
- Database engine and data model
- Index design and query shape
- Hardware, memory, disk and network
- Cache hit rate and key distribution
- Replication and consistency requirements
- Traffic bursts and latency targets

Choose a database, define the lookup query and benchmark it with realistic data and traffic before making a capacity decision.

## 7. Caching for a read-heavy workload

An in-memory cache such as Redis can reduce repeated database lookups.

A common pattern is **cache-aside**:

1. The application receives a short-code lookup.
2. It checks Redis.
3. If the value is present, return the cached destination URL.
4. If it is absent, read the database.
5. Store the result in Redis with a TTL.
6. Return the destination URL.

For example, if 95% of one million lookup requests are cache hits:

```text
Cache hits: 1,000,000 × 0.95 = 950,000
DB lookups: 1,000,000 × 0.05 =  50,000
```

This is an illustration, not a guaranteed hit rate. The actual result depends on access patterns, cache size, key popularity and TTL.

### Cache invalidation

If a destination URL is updated:

1. Update the database successfully.
2. Invalidate the corresponding Redis key.
3. The next lookup misses the cache, reads the updated value from the database and repopulates the cache.

TTL is a fallback expiry mechanism. If invalidation fails, a stale value may remain until TTL expiry. Systems needing stronger freshness guarantees require a deliberate consistency strategy.

## 8. High availability

A database replica can help keep the service available if the primary database fails.

A simplified setup:
- The primary database handles writes.
- One or more replicas receive replicated data.
- Reads may be served from replicas if consistency requirements allow.
- If the primary fails, an eligible replica can be promoted.

Replication is not a substitute for backups. Replicas can reproduce accidental updates or deletions, so backups and recovery testing are still necessary.

## Summary

| Metric | Estimate |
|---|---:|
| New URLs per month | 100 million |
| Average writes/second | ~40 |
| Average reads/second | ~4,000 |
| Assumed peak reads/second | ~12,000 |
| Retention period | 5 years |
| Total records | 6 billion |
| Raw storage | ~3 TB |

## Assumptions and limitations

- A month is treated as 30 days.
- Traffic is averaged across the full month.
- The read-to-write ratio remains 100:1.
- Each record is treated as exactly 500 bytes for the raw-storage calculation.
- All records are retained for five years.
- Peak read traffic is assumed to be 3× average.
- Indexes, replication, backups and other storage overhead are excluded from the 3 TB raw estimate.
- No benchmark has been run; database capacity must be validated experimentally.

## Reflection questions

1. Why is this service considered read-heavy?
2. Why might Redis reduce database load?
3. What can happen if a database update succeeds but cache invalidation fails?
4. Why is 3 TB not the same as total production storage?
5. What measurements would you collect before deciding to shard the database?
[url shortener design - medium article](https://medium.com/%40manishnalumachu/designing-a-url-shortener-like-bitly-c0e214370634 )





