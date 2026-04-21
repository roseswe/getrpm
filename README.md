# SUSE SCC Package Search Tools

>[!NOTE]
>A collection of lightweight Go utilities designed to interface with the **SUSE Customer Center (SCC) API**. These tools allow you to search for RPM packages and retrieve product information directly from SUSE's repositories without needing a registered or active system.

## 🛠 The Tools

### 1. `getrpm`
The primary tool for querying package details. It fetches the package name, available versions, architecture, release, and the specific repository identifier.

* **Fuzzy Matching:** Can search for partial names.
* **Version Sorting:** Intelligently sorts results by numeric version and release string.
* **Product Context:** Filters results by specific SUSE Product IDs (e.g., SLES 15 SP6, Liberty Linux 9).

### 2. `listprodids`
A helper utility that fetches the live list of all available SUSE products and their unique IDs directly from the SCC.

* **Live Data:** Unlike the hardcoded list in `getrpm`, this queries the API in real-time.
* **Formatting:** Supports table output, one-line strings (for code updates), and JSON-like formats for integration with other scripts like `vercheck.py`.

🛠️TODO🛠️ Merge both tools into a single one

---

## 🚀 Use Cases

* **Version Verification:** Quickly check if a specific security patch or version of `glibc`, `kernel`, or `openssl` is available for a specific SUSE distribution.
* **Repository Discovery:** Find the exact `identifier` (e.g., `sle-module-basesystem/15.6/x86_64`) needed for automation or RMT (Repository Mirroring Tool) configuration.
* **System Agnostic Checks:** Perform lookups from any machine (macOS, Windows, or non-SUSE Linux) without needing `zypper` or a registered subscription.
* **CI/CD Integration:** Use the structured output and exit codes to validate package availability before triggering build pipelines.

---

## 📦 Installation & Building

Since these are written in Go, you can compile them into single binaries:

```bash
# Build getrpm
go build -o getrpm getrpm.go

# Build listprodids
go build -o listprodids listprodids.go
```
Currently no makefile provided...

Hosted at [getrpm@github](https://github.com/roseswe/getrpm)
---

## 📖 Usage Examples

### Searching for a Package
To find `bash` versions for **SLES 15 SP5 x86_64** (Product ID 2465):
```bash
./getrpm -r bash -p 2465
```

### Fuzzy Search
If you aren't sure of the exact name, use the `-f` flag:
```bash
./getrpm -r python3-base -f -p 2793
```

### Listing Products
To see which Product IDs are currently available on the SCC:
```bash
./listprodids
./listprodids | grep "15 SP7"
```

To get a condensed list for script processing:
```bash
./listprodids --one
```

---

## ⌨️ Command Line Options

### `getrpm`
| Option | Long Flag | Description |
| :--- | :--- | :--- |
| `-r` | `--rpm` | Name of the RPM package (default: `glibc`) |
| `-p` | `--product` | SUSE Product ID (default: `2795`) |
| `-f` | `--fuzzy` | Enable fuzzy search (matches partial names) |
| `-l` | `--list` | Show the internal list of common Product IDs |
| `-v` | `--verbose` | Debug mode; shows API URLs and raw JSON response |
| `-V` | `--version` | Show build version and exit |

### `listprodids`
| Option | Long Flag | Description |
| :--- | :--- | :--- |
| `-1` | `--one` | Print IDs and Names in a single-line format |
| `--vercheck` | | Format output for `vercheck.py` compatibility |
| `-V` | `--version` | Show build version and exit |

---

## 🚦 Exit Codes (`getrpm`)
The tool returns specific exit codes for easier automation:
* **0**: Success.
* **64**: Invalid or missing parameters.
* **65**: API request failed (network or 404).
* **66**: Decoding failed (invalid JSON or no packages found).

---

## ⚖️ Disclaimer
These tools use the public-facing SUSE SCC API. While highly useful, detailed documentation for specific endpoints is limited. For mission-critical automation, consider the [SUSE Manager API](https://documentation.suse.com/suma/5.0/api/suse-manager/index.html).

**(c) ROSE SWE, Ralph Roth**

<!--
vim:set fileencoding=utf8 fileformat=unix filetype=gfm tabstop=2 expandtab:
@(#)  $Id: README.md,v 1.4 2026/04/21 12:03:01 ralph Exp $
-->
