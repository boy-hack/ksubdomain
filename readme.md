# KSubdomain: Ultra-Fast Stateless Subdomain Enumeration Tool

[![Release](https://img.shields.io/github/release/boy-hack/ksubdomain.svg)](https://github.com/boy-hack/ksubdomain/releases) [![Go Report Card](https://goreportcard.com/badge/github.com/boy-hack/ksubdomain)](https://goreportcard.com/report/github.com/boy-hack/ksubdomain) [![License](https://img.shields.io/github/license/boy-hack/ksubdomain)](https://github.com/boy-hack/ksubdomain/blob/main/LICENSE)

**KSubdomain is a stateless subdomain enumeration tool that delivers unprecedented scanning speed with extremely low memory consumption.** Say goodbye to traditional tool bottlenecks and experience lightning-fast DNS queries, backed by a reliable state-table retransmission mechanism that ensures result completeness. KSubdomain supports Windows, Linux, and macOS, making it the ideal choice for large-scale DNS asset discovery.

![](image.gif)

## 🚀 Core Advantages

*   **Lightning-Fast Speed:** Utilizes stateless scanning technology to directly operate network adapters for raw socket packet sending, bypassing the system kernel's network protocol stack to achieve astonishing packet rates. Use the `test` command to probe your local network adapter's maximum sending speed.
*   **Extremely Low Resource Consumption:** Innovative memory management mechanisms—including object pools and global memory pools—significantly reduce memory allocation and GC pressure, maintaining a low memory footprint even when processing massive domain lists.
*   **Stateless Design:** Similar to Masscan's stateless scanning, it does not maintain a state table in the system. Instead it builds a lightweight state table of its own, fundamentally solving traditional scanning tools' memory bottlenecks and performance limitations, as well as the packet-loss problem inherent in stateless scanning.
*   **Reliable Retransmission:** A built-in intelligent retransmission mechanism effectively handles network jitter and packet loss, ensuring result accuracy and completeness.
*   **Cross-Platform Support:** Perfect compatibility with Windows, Linux, and macOS.
*   **Easy to Use:** A clean command-line interface with verify and enum modes, plus built-in common dictionaries.

## ⚡ Performance Highlights

KSubdomain far exceeds similar tools in speed and efficiency. The following comparison was conducted on a 4-core CPU with 5 M bandwidth using a 100k dictionary:

| Tool         | Mode   | Method        | Command                                                                       | Time           | Success | Notes                           |
| ------------ | ------ | ------------- | ----------------------------------------------------------------------------- | -------------- | ------- | ------------------------------- |
| **KSubdomain** | Verify | pcap NIC send | `time ./ksubdomain v -b 5m -f d2.txt -o k.txt -r dns.txt --retry 3 --np`     | **~30 sec**    | 1397    | `--np` disables real-time print |
| massdns      | Verify | pcap/socket   | `time ./massdns -r dns.txt -t A -w m.txt d2.txt --root -o L`                  | ~3 min 29 sec  | 1396    |                                 |
| dnsx         | Verify | socket        | `time ./dnsx -a -o d.txt -r dns.txt -l d2.txt -retry 3 -t 5000`              | ~5 min 26 sec  | 1396    | `-t 5000` sets 5000 concurrency |

**Conclusion:** KSubdomain is **7× faster** than massdns and **10× faster** than dnsx!

## 🛠️ Technical Innovations (v2.0)

KSubdomain 2.0 introduces multiple low-level optimizations to squeeze out even more performance:

1.  **State Table Optimization:**
    *   **Sharded Locks:** Replace the global lock, dramatically reducing lock contention and improving concurrent write throughput.
    *   **Efficient Hashing:** Optimized key-value storage with even domain distribution for faster lookups.
2.  **Packet-Sending Optimization:**
    *   **Object Pools:** Reuse DNS packet structs to reduce memory allocation and GC overhead.
    *   **Template Caching:** Reuse Ethernet/IP/UDP layer data for the same DNS server, eliminating redundant construction.
    *   **Parallel Sending:** Multiple goroutines send packets concurrently, fully leveraging multi-core CPUs.
    *   **Batch Processing:** Send domain requests in batches to reduce system calls and context switches.
3.  **Receive Optimization:**
    *   **Object Pools:** Reuse parsers and buffers to lower memory consumption.
    *   **Parallel Processing Pipeline:** Receive → Parse → Process three-stage parallelism improves pipeline throughput.
    *   **Buffer Optimization:** Larger internal Channel buffers prevent processing blockage.
    *   **Efficient Filtering:** Optimized BPF filter rules and packet-processing logic quickly discard invalid packets.
4.  **Memory Management Optimization:**
    *   **Global Memory Pool:** `sync.Pool` manages common data structures to reduce allocation and fragmentation.
    *   **Structure Reuse:** Reuse DNS query structures and serialization buffers.
5.  **Architecture and Concurrency Optimization:**
    *   **Dynamic Concurrency:** Automatically adjusts goroutine count based on CPU core count.
    *   **Efficient Random Numbers:** Uses a higher-performance random-number generator.
    *   **Adaptive Rate:** Dynamically adjusts packet-sending rate based on network conditions and system load.
    *   **Batch Loading:** Loads and processes domains in batches to reduce per-domain fixed overhead.

## 📦 Installation

1.  **Download Pre-compiled Binary:** Visit the [Releases](https://github.com/boy-hack/ksubdomain/releases) page and download the latest version for your platform.
2.  **Install `libpcap` Dependency:**
    *   **Windows:** Download and install the [Npcap](https://npcap.com/) driver (WinPcap may not work).
    *   **Linux:** `libpcap` is statically linked in the release binary; no extra steps are usually needed. If you encounter issues, try installing the `libpcap-dev` or `libcap-devel` package.
    *   **macOS:** `libpcap` ships with the OS; no installation required.
3.  **Grant Execute Permission (Linux/macOS):** `chmod +x ksubdomain`
4.  **Run!**

### Build from Source (Optional)

Ensure Go 1.23 and `libpcap` are installed.

```bash
go install -v github.com/boy-hack/ksubdomain/v2/cmd/ksubdomain@latest
# The binary is usually placed in $GOPATH/bin or $HOME/go/bin
```

## 📖 Usage

```bash
KSubdomain - Ultra-Fast Stateless Subdomain Enumeration Tool

Usage:
  ksubdomain [global options] command [command options] [arguments...]

Version:
  Check version: ksubdomain --version

Commands:
  enum, e    Enumeration mode: provide a root domain for brute-forcing
  verify, v  Verification mode: provide a domain list to verify
  test       Test the maximum packet-sending speed of the local NIC
  help, h    Show the command list or help for a specific command

Global Options:
  --help, -h     Show help (default: false)
  --version, -v  Print version information (default: false)
```

### Verification Mode (Verify)

Verification mode quickly checks the alive status of a provided domain list.

```bash
./ksubdomain verify -h # View verify-mode help; can be abbreviated as ksubdomain v

USAGE:
   ksubdomain verify [command options] [arguments...]

OPTIONS:
   --filename value, -f value       Path to the file containing domains to verify
   --domain value, -d value         Domain name
   --band value, -b value           Downstream bandwidth, e.g. 5M, 5K, 5G (default: "3m")
   --resolvers value, -r value      DNS servers; uses built-in DNS by default
   --output value, -o value         Output filename
   --output-type value, --oy value  Output file type: json, txt, csv (default: "txt")
   --silent                         Only print domain names to screen (default: false)
   --retry value                    Retry count; -1 to retry indefinitely (default: 3)
   --timeout value                  Timeout in seconds (default: 6)
   --stdin                          Accept input from stdin (default: false)
   --not-print, --np                Do not print domain results (default: false)
   --eth value, -e value            Specify network adapter name
   --wild-filter-mode value         Wildcard filter mode [filter wildcard domains from final results]: basic, advanced, none (default: "none")
   --predict                        Enable domain prediction mode (default: false)
   --help, -h                       show help (default: false)

# Examples:
# Verify multiple domains
./ksubdomain v -d xx1.example.com -d xx2example.com

# Read domains from a file and save results to output.txt
./ksubdomain v -f domains.txt -o output.txt

# Read domains from stdin with a bandwidth limit of 10 M
cat domains.txt | ./ksubdomain v --stdin -b 10M

# Enable prediction mode with advanced wildcard filtering; save as CSV
./ksubdomain v -f domains.txt --predict --wild-filter-mode advanced --oy csv -o output.csv
```

### Enumeration Mode (Enum)

Enumeration mode brute-forces subdomains under a given domain using a dictionary and a prediction algorithm.

```bash
./ksubdomain enum -h # View enum-mode help; can be abbreviated as ksubdomain e

USAGE:
   ksubdomain enum [command options] [arguments...]

OPTIONS:
   --domain value, -d value         Domain name
   --band value, -b value           Downstream bandwidth, e.g. 5M, 5K, 5G (default: "3m")
   --resolvers value, -r value      DNS servers; uses built-in DNS by default
   --output value, -o value         Output filename
   --output-type value, --oy value  Output file type: json, txt, csv (default: "txt")
   --silent                         Only print domain names to screen (default: false)
   --retry value                    Retry count; -1 to retry indefinitely (default: 3)
   --timeout value                  Timeout in seconds (default: 6)
   --stdin                          Accept input from stdin (default: false)
   --not-print, --np                Do not print domain results (default: false)
   --eth value, -e value            Specify network adapter name
   --wild-filter-mode value         Wildcard filter mode [filter wildcard domains from final results]: basic, advanced, none (default: "none")
   --predict                        Enable domain prediction mode (default: false)
   --filename value, -f value       Dictionary file path
   --ns                             Read the domain's NS records and add them to the resolver list (default: false)
   --help, -h                       show help (default: false)

# Examples:
# Enumerate multiple domains
./ksubdomain e -d example.com -d hacker.com

# Use a dictionary file and save results to output.txt
./ksubdomain e -f sub.dict -o output.txt

# Read domains from stdin with a bandwidth limit of 10 M
cat domains.txt | ./ksubdomain e --stdin -b 10M

# Enable prediction mode with advanced wildcard filtering; save as CSV
./ksubdomain e -d example.com --predict --wild-filter-mode advanced --oy csv -o output.csv
```

## ✨ Features & Tips

*   **Automatic Bandwidth Adaptation:** Simply set your public downstream bandwidth with `-b` (e.g., `-b 10m`) and KSubdomain will automatically optimize the packet-sending rate.
*   **Test Maximum Rate:** Run `./ksubdomain test` to measure the maximum theoretical packet rate in your current environment.
*   **Automatic NIC Detection:** KSubdomain auto-detects available network adapters.
*   **Progress Display:** A real-time progress bar shows Success / Sent / Queue Length / Received / Failed / Elapsed Time.
*   **Parameter Tuning:** Adjust `--retry` and `--timeout` based on network quality and the number of target domains to get the best results. When `--retry` is -1, retries are unlimited until all requests succeed or time out.
*   **Multiple Output Formats:** Supports `txt` (real-time output), `json` (output after completion), and `csv` (output after completion). Specify the format via the `-o` file extension (e.g., `result.json`).
*   **Environment Variable Configuration:**
    *   `KSubdomainConfig`: Specifies the path to a configuration file.

## 💡 References

*   Original KSubdomain project: [https://github.com/knownsec/ksubdomain](https://github.com/knownsec/ksubdomain)
*   From Masscan/Zmap Source Analysis to Development Practice: [https://paper.seebug.org/1052/](https://paper.seebug.org/1052/)
*   KSubdomain Stateless Subdomain Enumeration Tool Introduction: [https://paper.seebug.org/1325/](https://paper.seebug.org/1325/)
*   KSubdomain vs massdns Comparison: [WeChat article](https://mp.weixin.qq.com/s?__biz=MzU2NzcwNTY3Mg==&mid=2247484471&idx=1&sn=322d5db2d11363cd2392d7bd29c679f1&chksm=fc986d10cbefe406f4bda22f62a16f08c71f31c241024fc82ecbb8e41c9c7188cfbd71276b81&token=76024279&lang=zh_CN#rd)
