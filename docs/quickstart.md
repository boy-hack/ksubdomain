# Quick Start Guide

## Prerequisites

| Requirement | Notes |
|---|---|
| OS | Linux, macOS, Windows (WSL2 recommended) |
| Privileges | **root** or `CAP_NET_RAW` — raw packet capture requires elevated access |
| libpcap / npcap | Linux: `libpcap-dev`; macOS: built-in; Windows: [Npcap](https://npcap.com) |

---

## Installation

### Download pre-built binary (recommended)

```bash
# Linux x86_64
curl -L https://github.com/boy-hack/ksubdomain/releases/latest/download/ksubdomain_linux_amd64 \
     -o /usr/local/bin/ksubdomain
chmod +x /usr/local/bin/ksubdomain
```

### Build from source

```bash
git clone https://github.com/boy-hack/ksubdomain.git
cd ksubdomain
go build -o ksubdomain ./cmd/ksubdomain
# or use the build script (cross-compile all platforms):
./build.sh
```

---

## Your first scan

### 1 — Enumerate subdomains (built-in wordlist)

```bash
sudo ksubdomain enum -d example.com
```

Sample output:

```
www.example.com => 93.184.216.34
mail.example.com => 93.184.216.50
api.example.com => 93.184.216.51
```

### 2 — Enumerate with a custom wordlist

```bash
sudo ksubdomain enum -d example.com -f /path/to/wordlist.txt
```

### 3 — Verify a list of known subdomains

```bash
cat domains.txt | sudo ksubdomain verify
# or
sudo ksubdomain verify -f domains.txt
```

### 4 — Pipe into httpx

```bash
sudo ksubdomain enum -d example.com --silent --only-domain | httpx -silent
```

`--only-domain` prints one clean domain per line with no extra characters,
making the output safe to pipe into any line-oriented tool.

---

## Common flags

| Flag | Short | Description |
|---|---|---|
| `--domain` | `-d` | Target domain (enum mode) |
| `--file` | `-f` | Input file (wordlist for enum, domain list for verify) |
| `--band` | `-b` | Bandwidth limit, e.g. `5m`, `100m` (default: `5m`) |
| `--retry` | `-r` | Max retries per domain (default: `3`) |
| `--resolvers` | | Custom DNS resolver IPs, comma-separated |
| `--output` | `-o` | Output file path |
| `--output-type` | `--ot` | Output format: `txt`, `json`, `csv`, `jsonl` |
| `--only-domain` | `--od` | Print only the domain name, no record values |
| `--silent` | | Suppress progress bar and informational logs |
| `--wild-filter-mode` | | Wildcard filter: `none` (default), `basic`, `advanced` |
| `--predict` | | Enable AI subdomain prediction |

---

## Output formats

```bash
# Plain text (default)
sudo ksubdomain enum -d example.com -o results.txt

# JSON Lines (jq-compatible, one object per line)
sudo ksubdomain enum -d example.com -o results.jsonl --ot jsonl

# Parse with jq
jq '.domain' results.jsonl
jq 'select(.type=="A") | .records[]' results.jsonl
```

See [docs/OUTPUT_FORMATS.md](./OUTPUT_FORMATS.md) for full format details.

---

## Bandwidth tuning

ksubdomain operates at the raw packet level and sends DNS queries at the
rate you specify. Start conservatively and increase:

```bash
# ~60 Mbit bandwidth cap (safe for most home connections)
sudo ksubdomain enum -d example.com -b 60m

# Max out a Gigabit interface
sudo ksubdomain enum -d example.com -b 1000m
```

See [docs/best-practices.md](./best-practices.md) for bandwidth and resolver advice.

---

## Troubleshooting quick reference

| Symptom | Fix |
|---|---|
| `permission denied` | Run with `sudo` or grant `CAP_NET_RAW` |
| `network device not found` | Specify interface with `--eth <name>`; list with `ip link show` |
| `network device not active` | Bring interface up: `sudo ip link set <name> up` |
| No results, no errors | Try `--wild-filter-mode none`; target domain may have wildcard DNS |
| macOS `ENOBUFS` | Lower bandwidth: `-b 5m` |
| WSL2 wrong interface | Add `--eth eth0` |

For more, see [docs/faq.md](./faq.md).
