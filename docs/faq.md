# FAQ

## Permissions

### Why do I need sudo?

ksubdomain uses **raw packet capture** (libpcap / npcap) to send and
receive DNS queries at wire speed.  This requires either root access or
the `CAP_NET_RAW` capability.

```bash
# Option 1: run with sudo
sudo ksubdomain enum -d example.com

# Option 2: grant capability to the binary (Linux only)
sudo setcap cap_net_raw+ep /usr/local/bin/ksubdomain
ksubdomain enum -d example.com   # no sudo needed
```

---

## Network interface

### Error: `network device not found`

Your interface name is wrong.  List available interfaces:

```bash
# Linux / WSL
ip link show

# macOS
ifconfig -a
```

Then pass the correct name:

```bash
sudo ksubdomain enum -d example.com --eth eth0
```

### Error: `network device not active`

The interface exists but is not up:

```bash
# Bring it up
sudo ip link set eth0 up
```

### Which interface does ksubdomain pick by default?

It reads the system routing table and picks the interface on the default
route.  If your machine has multiple NICs (e.g., VPN + physical), you may
need to specify `--eth` explicitly.

---

## macOS

### `ENOBUFS` / packet drops at high bandwidth

macOS has a conservative BPF buffer size.  Keep bandwidth below 50 Mbit:

```bash
sudo ksubdomain enum -d example.com -b 10m
```

Alternatively, increase the BPF buffer:

```bash
sudo sysctl -w debug.bpf_maxbufsize=8388608
```

---

## WSL2

### No results / wrong interface

WSL2 uses a virtual NIC named `eth0`.  Always specify it:

```bash
sudo ksubdomain enum -d example.com --eth eth0
```

### libpcap not found in WSL2

```bash
sudo apt-get install libpcap-dev
```

---

## DNS / results

### Getting zero results — the domain has wildcard DNS

If `*.example.com` resolves to a real IP, every query appears successful.
Enable wildcard filtering:

```bash
sudo ksubdomain enum -d example.com --wild-filter-mode basic
```

If that's too aggressive, check with:

```bash
dig $(openssl rand -hex 8).example.com
```

If that resolves, the domain uses wildcard DNS.

### Results look incomplete / many retries

- Lower your bandwidth (`-b 5m` to start)
- Add more resolvers (`--resolvers` with a list file)
- Increase retry count (`-r 5`)

### SERVFAIL / REFUSED responses

Some resolvers rate-limit aggressive queries.  Use more resolvers or
switch to dedicated recursive resolvers.

---

## Piping / output

### httpx sees garbled input / extra characters

Make sure you use **both** `--silent` and `--only-domain` (or `--od`):

```bash
sudo ksubdomain enum -d example.com --silent --od | httpx -silent
```

`--silent` suppresses the progress bar output on stdout.
`--od` ensures only the bare domain name is printed, with no IP or CNAME suffix.

### jq can't parse JSONL output

Confirm the file has one valid JSON object per line:

```bash
head -1 results.jsonl | jq .
```

If you see parse errors, the file may have been written while the scan
was still running and the last line is incomplete.  Always wait for the
scan to finish before processing the file (or use `EnumStream` via the
SDK for real-time processing).

---

## Build / Go

### `go build` fails with missing libpcap

```bash
# Debian / Ubuntu
sudo apt-get install libpcap-dev

# RHEL / CentOS
sudo yum install libpcap-devel

# macOS
brew install libpcap
```

### Cross-compilation

Use the `build.sh` script which sets the correct `CGO_ENABLED` and
`CC` for each target platform.

---

## Exit codes

| Code | Meaning |
|---|---|
| 0 | At least one subdomain was resolved |
| 1 | No subdomains found (empty result set) |
| non-zero (from framework) | CLI usage error or fatal initialisation failure |

This lets you use `&&` in shell pipelines:

```bash
sudo ksubdomain enum -d example.com --od --silent | httpx -silent \
  && echo "httpx ran because at least one domain was found"
```
