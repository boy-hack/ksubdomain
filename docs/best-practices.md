# Best Practices

## Bandwidth selection

ksubdomain sends DNS queries as raw UDP packets.  Choose a bandwidth limit
that matches your uplink without causing router/firewall drops.

| Scenario | Recommended `-b` |
|---|---|
| Home broadband (< 100 Mbit) | `5m` – `20m` |
| Cloud VM, shared NIC | `50m` – `100m` |
| Dedicated server, 1 Gbit NIC | `200m` – `500m` |
| Bare-metal, 10 Gbit NIC | `1000m` – `2000m` |

**Signs you are going too fast:**
- Many retries (progress bar retry count climbs quickly)
- Results drop off near the end of a scan
- Your router / firewall drops UDP packets (check `netstat -s`)

Start at `5m` and double until you see retries climb, then back off 20%.

---

## DNS resolver selection

### Default resolvers

ksubdomain ships a built-in list of reliable public resolvers.  For most
users this is fine.

### Custom resolvers

Use authoritative resolvers of the target TLD for maximum accuracy:

```bash
# Use a specific resolver list
sudo ksubdomain enum -d example.com --resolvers resolvers.txt
```

Good public resolver choices:

| Resolver | IP |
|---|---|
| Cloudflare | `1.1.1.1`, `1.0.0.1` |
| Google | `8.8.8.8`, `8.8.4.4` |
| Quad9 | `9.9.9.9` |

### Avoid overloading a single resolver

Spread queries across at least 3–5 resolvers.  A single resolver hit with
high RPS may start rate-limiting or returning `SERVFAIL`.

---

## Wildcard DNS handling

Some domains return a valid A record for **any** subdomain
(e.g., `*.example.com => 1.2.3.4`).  Without filtering, every single
wordlist entry appears to resolve.

```bash
# Detect and filter wildcard results automatically
sudo ksubdomain enum -d example.com --wild-filter-mode basic

# Stricter: also filter IPs that appear in >1% of responses
sudo ksubdomain enum -d example.com --wild-filter-mode advanced
```

When to use each mode:

| Mode | When |
|---|---|
| `none` | You have already confirmed there is no wildcard |
| `basic` | Default — removes exact wildcard IPs |
| `advanced` | Aggressive wildcard domains; may miss a few real records |

---

## Pipe-friendly output

For integration with `httpx`, `nuclei`, and other tools:

```bash
# One clean domain per line, no control characters
sudo ksubdomain enum -d example.com --silent --only-domain | httpx -silent

# Save and pipe simultaneously
sudo ksubdomain enum -d example.com --silent --only-domain \
    | tee found.txt | httpx -silent -o http-alive.txt
```

**Important**: always use `--silent` together with `--only-domain` when
piping.  Without `--silent`, progress-bar updates are written to stdout
and will corrupt the domain list.

---

## JSONL for downstream analysis

```bash
sudo ksubdomain enum -d example.com -o results.jsonl --ot jsonl

# All A record IPs
jq 'select(.type=="A") | .records[]' results.jsonl

# Unique CNAME targets
jq 'select(.type=="CNAME") | .records[]' results.jsonl | sort -u

# Count by type
jq -r '.type' results.jsonl | sort | uniq -c | sort -rn
```

---

## Large-scale enumeration

For scans with millions of domains (e.g., combined wordlists):

1. **Split the wordlist** into chunks of ~500 K lines and run sequentially
   to avoid memory pressure on the status database.

2. **Use a retry of 2** for speed; increase to 3 only if accuracy is critical.

3. **Monitor memory** — the sharded statusDB holds in-flight queries.
   At 100 K qps with a 3 s window you may have ~300 K in-flight entries.

4. **Deduplicate results** afterwards:
   ```bash
   sort -u results.txt -o results-dedup.txt
   ```

---

## SDK usage tips

### Cancellation

Always pass a `context.WithTimeout` or `context.WithCancel` context so
the scan can be stopped cleanly:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()
scanner.EnumStream(ctx, "example.com", callback)
```

### One scanner at a time

Do **not** run two `Scanner` instances simultaneously on the same network
interface.  They will fight over the same raw socket and corrupt each
other's DNS ID space.

### Thread safety of callbacks

`EnumStream` / `VerifyStream` callbacks are called from multiple goroutines.
Protect shared state with a mutex:

```go
var mu sync.Mutex
var results []sdk.Result
scanner.EnumStream(ctx, domain, func(r sdk.Result) {
    mu.Lock()
    results = append(results, r)
    mu.Unlock()
})
```
