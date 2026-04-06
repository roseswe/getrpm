package main

// gets the RPM package information from the SUSE SCC API
// prints the package name, available versions, architecture, release, and repository
// supports fuzzy search and verbose output
// supports listing available product IDs

// curl -s "https://scc.suse.com/api/package_search/packages?product_id=2611&query=glibc"

// While the SCC offers a set of APIs to facilitate access to available products and their repositories, detailed public documentation for this specific API endpoint is limited. The SCC's APIs are primarily intended to assist with entitlement management, subscription overviews, and access to updates and patches. These APIs enable integration with management tools such as SUSE Manager or Repository Mirroring Tool (RMT). ([scc.suse.com](https://scc.suse.com/docs/help?id=help&locale=en&utm_source))

// For more comprehensive API interactions, SUSE Manager provides a well-documented API that allows for extensive automation of various tasks, including package management and system configuration. The SUSE Manager API documentation is available at: SUSE Manager API Version 27: ([documentation.suse.com](https://documentation.suse.com/suma/5.0/api/suse-manager/index.html))

// (c) by ROSE SWE, Ralph Roth
// @(#) $Id: getrpm.go,v 1.27 2026/04/06 09:51:48 ralph Exp $

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const (
	// go fix: ./listprodids.go:29:2: cStrAPIURL redeclared in this block
	cStrAPIURL    = "https://scc.suse.com/api/package_search/packages?product_id="
	cIntErrUsage  = 64 // Fehlender oder falscher Parameter
	cIntErrAPI    = 65 // Fehlerhafte API-Anfrage
	cIntErrDecode = 66 // Fehler beim Dekodieren der Antwort
)

/* HTTP API response: https://scc.suse.com/api/package_search/packages?product_id=2611&query=glibc-extra

data
	0
	id	37704504
	name	"glibc-extra"
	arch	"x86_64"
	version	"2.38"
	release	"150600.12.1"
	products
	0
	id	2618
	name	"Basesystem Module"
	identifier	"sle-module-basesystem/15.6/x86_64"
	type	"module"
	free	true
	architecture	"x86_64"
	version	"15 SP6"

The API documented at https://scc.suse.com/api/products/v4/documentation is the SUSE Customer Center (SCC) Products API, Version 4.
This is a RESTful API that uses standard HTTP methods (GET, POST, PUT/PATCH, DELETE) and returns data in JSON format.

*/

var cMapProductIDs = map[int]string{
	// Supported products/out in the wild -- 2025
	// SLES Kernel version = https://www.suse.com/support/kb/doc/?id=000019587
	// SUSE Linux Micro Kernel versions = https://www.suse.com/support/kb/doc/?id=000021672

	2538: "SUSE Liberty Linux 9 x86_64",
	2539: "SUSE Liberty Linux High Availability Extension 9 x86_64", // non-200 response from API
	1921: "SUSE Liberty Linux 8 x86_64",
	1922: "SUSE Liberty Linux High Availability Extension 8 x86_64", // non-200 response from API
	1251: "SUSE Liberty Linux 7 x86_64",
	1252: "SUSE Liberty Linux High Availability Extension 7 x86_64", // non-200 response from API
	2702: "SUSE Liberty Linux LTSS 7 x86_64",
	1138: "SUSE Liberty Linux 6 x86_64", // non-200 response from API

	2470: "SUSE Linux Enterprise High Performance Computing 15 SP5 x86_64",
	2469: "SUSE Linux Enterprise High Performance Computing 15 SP5 aarch64",
	2353: "SUSE Linux Enterprise High Performance Computing 15 SP4 aarch64",
	2354: "SUSE Linux Enterprise High Performance Computing 15 SP4 x86_64",
	2132: "SUSE Linux Enterprise High Performance Computing 15 SP3 aarch64",
	2133: "SUSE Linux Enterprise High Performance Computing 15 SP3 x86_64",
	1934: "SUSE Linux Enterprise High Performance Computing 15 SP2 x86_64",
	1933: "SUSE Linux Enterprise High Performance Computing 15 SP2 aarch64",
	1768: "SUSE Linux Enterprise High Performance Computing 15 SP1 x86_64",
	1767: "SUSE Linux Enterprise High Performance Computing 15 SP1 aarch64",
	1731: "SUSE Linux Enterprise High Performance Computing 15 aarch64",
	1732: "SUSE Linux Enterprise High Performance Computing 15 x86_64",
	1872: "SUSE Linux Enterprise High Performance Computing 12 SP5 aarch64",
	1873: "SUSE Linux Enterprise High Performance Computing 12 SP5 x86_64",
	1759: "SUSE Linux Enterprise High Performance Computing 12 SP4 x86_64",
	1758: "SUSE Linux Enterprise High Performance Computing 12 SP4 aarch64",
	1750: "SUSE Linux Enterprise High Performance Computing 12 SP3 aarch64",
	1751: "SUSE Linux Enterprise High Performance Computing 12 SP3 x86_64",
	1749: "SUSE Linux Enterprise High Performance Computing 12 SP2 x86_64",

	2926: "SUSE Linux Enterprise Real Time 15 SP7 x86_64",
	2735: "SUSE Linux Enterprise Real Time 15 SP6 x86_64",
	2582: "SUSE Linux Enterprise Real Time 15 SP5 x86_64",
	2421: "SUSE Linux Enterprise Real Time 15 SP4 x86_64",
	2285: "SUSE Linux Enterprise Real Time 15 SP3 x86_64",
	2003: "SUSE Linux Enterprise Real Time 15 SP2 x86_64",
	1861: "SUSE Linux Enterprise Real Time 15 SP1 x86_64",

	1875: "SUSE Linux Enterprise Server 12 SP5 aarch64",
	1876: "SUSE Linux Enterprise Server 12 SP5 ppc64le",
	1877: "SUSE Linux Enterprise Server 12 SP5 s390x",
	1878: "SUSE Linux Enterprise Server 12 SP5 x86_64",

	2292: "SUSE Linux Enterprise Server 15 SP4 x86_64",
	2140: "SUSE Linux Enterprise Server 15 SP3 x86_64",
	2138: "SUSE Linux Enterprise Server 15 SP3 ppc64le",
	1939: "SUSE Linux Enterprise Server 15 SP2 x86_64",
	1763: "SUSE Linux Enterprise Server 15 SP1 x86_64",
	1575: "SUSE Linux Enterprise Server 15 x86_64",
	2290: "SUSE Linux Enterprise Server 15 SP4 ppc64le",

	2462: "SUSE Linux Enterprise Server 15 SP5 aarch64",
	2463: "SUSE Linux Enterprise Server 15 SP5 ppc64le",
	2464: "SUSE Linux Enterprise Server 15 SP5 s390x",
	2465: "SUSE Linux Enterprise Server 15 SP5 x86_64",

	2606: "SUSE Linux Enterprise Server 15 SP6 aarch64",
	2607: "SUSE Linux Enterprise Server 15 SP6 ppc64le",
	2608: "SUSE Linux Enterprise Server 15 SP6 s390x",
	2609: "SUSE Linux Enterprise Server 15 SP6 x86_64",

	2468: "SUSE Linux Enterprise Desktop 15 SP5 x86_64",
	2612: "SUSE Linux Enterprise Desktop 15 SP6 x86_64",
	2796: "SUSE Linux Enterprise Desktop 15 SP7 x86_64",

	1879: "SUSE Linux Enterprise Server for SAP Applications 12 SP5 ppc64le",
	1880: "SUSE Linux Enterprise Server for SAP Applications 12 SP5 x86_64",
	1755: "SUSE Linux Enterprise Server for SAP Applications 12 SP4 x86_64",
	1754: "SUSE Linux Enterprise Server for SAP Applications 12 SP4 ppc64le",
	1572: "SUSE Linux Enterprise Server for SAP Applications 12 SP3 ppc64le",
	1426: "SUSE Linux Enterprise Server for SAP Applications 12 SP3 x86_64",
	1414: "SUSE Linux Enterprise Server for SAP Applications 12 SP2 x86_64",
	1521: "SUSE Linux Enterprise Server for SAP Applications 12 SP2 ppc64le",
	1346: "SUSE Linux Enterprise Server for SAP Applications 12 SP1 x86_64",
	1437: "SUSE Linux Enterprise Server for SAP Applications 12 SP1 ppc64le",
	1319: "SUSE Linux Enterprise Server for SAP Applications 12 x86_64",

	1612: "SUSE Linux Enterprise Server for SAP Applications 15 x86_64",
	1613: "SUSE Linux Enterprise Server for SAP Applications 15 ppc64le",
	1765: "SUSE Linux Enterprise Server for SAP Applications 15 SP1 ppc64le",
	1766: "SUSE Linux Enterprise Server for SAP Applications 15 SP1 x86_64",
	1940: "SUSE Linux Enterprise Server for SAP Applications 15 SP2 ppc64le",
	1941: "SUSE Linux Enterprise Server for SAP Applications 15 SP2 x86_64",
	2135: "SUSE Linux Enterprise Server for SAP Applications 15 SP3 ppc64le",
	2136: "SUSE Linux Enterprise Server for SAP Applications 15 SP3 x86_64",
	2293: "SUSE Linux Enterprise Server for SAP Applications 15 SP4 ppc64le",
	2294: "SUSE Linux Enterprise Server for SAP Applications 15 SP4 x86_64",
	2467: "SUSE Linux Enterprise Server for SAP Applications 15 SP5 x86_64",
	2611: "SUSE Linux Enterprise Server for SAP Applications 15 SP6 x86_64",

	2603: "SUSE Linux Enterprise Micro 5.5 aarch64",
	2605: "SUSE Linux Enterprise Micro 5.5 x86_64",
	2780: "SUSE Linux Enterprise Micro 5.5 ppc64le",
	2604: "SUSE Linux Enterprise Micro 5.5 s390x",
	2573: "SUSE Linux Enterprise Micro 5.4 s390x",
	2574: "SUSE Linux Enterprise Micro 5.4 x86_64",
	2572: "SUSE Linux Enterprise Micro 5.4 aarch64",
	2428: "SUSE Linux Enterprise Micro 5.3 x86_64",
	2426: "SUSE Linux Enterprise Micro 5.3 aarch64",
	2427: "SUSE Linux Enterprise Micro 5.3 s390x",
	2401: "SUSE Linux Enterprise Micro 5.2 x86_64",
	2400: "SUSE Linux Enterprise Micro 5.2 s390x",
	2399: "SUSE Linux Enterprise Micro 5.2 aarch64",
	2283: "SUSE Linux Enterprise Micro 5.1 x86_64",
	2282: "SUSE Linux Enterprise Micro 5.1 aarch64",
	2287: "SUSE Linux Enterprise Micro 5.1 s390x",
	2202: "SUSE Linux Enterprise Micro 5.0 x86_64",
	2201: "SUSE Linux Enterprise Micro 5.0 aarch64",

	2697: "SUSE Linux Micro 6.0 aarch64",
	2698: "SUSE Linux Micro 6.0 s390x",
	2699: "SUSE Linux Micro 6.0 x86_64",

	2775: "SUSE Linux Micro 6.1 aarch64",
	2776: "SUSE Linux Micro 6.1 s390x",
	2774: "SUSE Linux Micro 6.1 x86_64",
	2777: "SUSE Linux Micro 6.1 ppc64le",

	2916: "SUSE Linux Micro 6.2 ppc64le",
	2915: "SUSE Linux Micro 6.2 s390x",
	2914: "SUSE Linux Micro 6.2 aarch64",
	2913: "SUSE Linux Micro 6.2 x86_64",

	2379: "SUSE Manager Proxy 4.3 x86_64",
	2223: "SUSE Manager Proxy 4.2 x86_64",
	2009: "SUSE Manager Proxy 4.1 x86_64",
	1900: "SUSE Manager Proxy 4.0 x86_64",
	2376: "SUSE Manager Server 4.3 ppc64le",
	2378: "SUSE Manager Server 4.3 x86_64",
	2377: "SUSE Manager Server 4.3 s390x",
	2221: "SUSE Manager Server 4.2 s390x",
	2220: "SUSE Manager Server 4.2 ppc64le",
	2222: "SUSE Manager Server 4.2 x86_64",
	2011: "SUSE Manager Server 4.1 s390x",
	2012: "SUSE Manager Server 4.1 x86_64",
	2010: "SUSE Manager Server 4.1 ppc64le",
	1899: "SUSE Manager Server 4.0 x86_64",
	1898: "SUSE Manager Server 4.0 s390x",
	1897: "SUSE Manager Server 4.0 ppc64le",

	// new hot stuff Autumn 2025 / Winter 2025
	2790: "SUSE Linux Enterprise Server 15 SP7 aarch64",
	2791: "SUSE Linux Enterprise Server 15 SP7 ppc64le",
	2793: "SUSE Linux Enterprise Server 15 SP7 x86_64",
	2792: "SUSE Linux Enterprise Server 15 SP7 s390x",
	2794: "SUSE Linux Enterprise Server for SAP Applications 15 SP7 ppc64le",
	2795: "SUSE Linux Enterprise Server for SAP Applications 15 SP7 x86_64",

	// released 05.11.25 --  https://scc.suse.com/admin/products/2930
	2933: "SUSE Linux Enterprise Server 16.0 ppc64le",
	2930: "SUSE Linux Enterprise Server 16.0 x86_64", // Trial product code 113-002667-001
	2931: "SUSE Linux Enterprise Server 16.0 aarch64",
	2932: "SUSE Linux Enterprise Server 16.0 s390x",
	2985: "SUSE Linux Enterprise Server for SAP Applications 16.0 x86_64",
	2986: "SUSE Linux Enterprise Server for SAP Applications 16.0 ppc64le",

	// 05.02.2026
	3236: "SUSE Linux Enterprise Server for SAP applications 16.1 ppc64le",
	3235: "SUSE Linux Enterprise Server for SAP applications 16.1 x86_64",
	3234: "SUSE Linux Enterprise Server 16.1 ppc64le",
	3231: "SUSE Linux Enterprise Server 16.1 x86_64",
	3233: "SUSE Linux Enterprise Server 16.1 s390x",
	3232: "SUSE Linux Enterprise Server 16.1 aarch64",
}

var compileDate string

type tStructProduct struct {
	Identifier string `json:"identifier"`
}

type tStructPackage struct {
	Name     string           `json:"name"`
	Version  string           `json:"version"`
	Arch     string           `json:"arch"`
	Release  string           `json:"release"`
	Products []tStructProduct `json:"products"`
	Repo     string           `json:"-"`
}

type tStructAPIResponse struct {
	Data []tStructPackage `json:"data"`
}

func fUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
	fmt.Println("Search for a RPM package using the SUSE SCC Public API. (c) by ROSE SWE, Ralph Roth")
	fmt.Println("\n-=| Options:")
	fmt.Println("  -?, -h, --help       Show this help message")

	fmt.Println("  -v, --verbose        Enable verbose output (debug mode)")
	fmt.Println("  -V, --version        Show version and exit")
	fmt.Println("  -r, --rpm <name>     Specify the RPM package name (default: glibc)")
	fmt.Println("  -p, --product <id>   Specify the product ID (default: 2795)")
	fmt.Println("  -l, --list           List available product IDs for option -p")
	fmt.Println("  -f, --fuzzy          Enable fuzzy output (ignore exact name match)")
	fmt.Println("\n-=| Exit Codes:")
	fmt.Println("  64 - Invalid or missing parameter")
	fmt.Println("  65 - API request failed")
	fmt.Println("  66 - Response decoding failed")
	fmt.Println("\n-=| Examples:")
	fmt.Println("  ./getrpm --rpm glibc")
	fmt.Println("  ./getrpm -r bash -v -p 2465 -f")
	fmt.Println("  ./getrpm --list")
	os.Exit(cIntErrUsage)
}

func fGetPackages(pStrRPM string, pStrProduct string, pBooVerbose bool) ([]tStructPackage, error) {
	lStrURL := cStrAPIURL + pStrProduct + "&query=" + pStrRPM
	if pBooVerbose {
		fmt.Println("Fetching data from API...", lStrURL)
	}

	req, lErr := http.NewRequest("GET", lStrURL, nil)
	if lErr != nil {
		return nil, fmt.Errorf("failed to create request: %w", lErr)
	}

	// not sure if we need this.... -> https://scc.suse.com/api/products/v4/documentation#/product_trials/get_downloads
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept", "application/vnd.scc.suse.com.v4+json")
	req.Header.Set("Connection", "close")

	client := &http.Client{}
	lResp, lErr := client.Do(req)
	if lErr != nil {
		return nil, fmt.Errorf("SCC Public API request failed: %w", lErr)
	}
	defer lResp.Body.Close()

	if pBooVerbose {
		fmt.Print("Response status code: ", lResp.StatusCode, "\n")
	}

	if lResp.StatusCode != http.StatusOK {
		return nil, errors.New("non-200 response from API")
	}

	lBytBody, lErr := io.ReadAll(lResp.Body)
	if lErr != nil {
		return nil, fmt.Errorf("failed to read API response: %w", lErr)
	}

	if pBooVerbose {
		fmt.Println("Response body dump:", string(lBytBody))
	}

	var lStructResp tStructAPIResponse
	lErr = json.Unmarshal(lBytBody, &lStructResp) // the big magic happens here
	if lErr != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", lErr)
	}

	for i, pkg := range lStructResp.Data {
		if len(pkg.Products) > 0 {
			lStructResp.Data[i].Repo = pkg.Products[0].Identifier
		}
	}

	if pBooVerbose && len(lStructResp.Data) == 0 {
		fmt.Println("No packages found. (Empty response)")
	}

	return lStructResp.Data, nil
}

// parseVersionFloat extracts digit and dot characters from a version string
// and attempts to parse them as a float64 for numeric comparison.
// Non-numeric/qualifier parts are ignored.
func parseVersionFloat(s string) float64 {
	filtered := strings.Map(func(r rune) rune {
		if r == '.' || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, s)
	if filtered == "" {
		return 0.0
	}
	f, err := strconv.ParseFloat(filtered, 64)
	if err != nil {
		return 0.0
	}
	return f
}

func fPrintTable(pArrPackages []tStructPackage, rpmName string, productID string, fuzzy bool) {
	// sort packages by name (case-insensitive), then numeric version, then release
	sort.Slice(pArrPackages, func(i, j int) bool {
		a := pArrPackages[i]
		b := pArrPackages[j]
		ai := strings.ToLower(a.Name)
		bi := strings.ToLower(b.Name)
		if ai != bi {
			return ai < bi
		}
		af := parseVersionFloat(a.Version)
		bf := parseVersionFloat(b.Version)
		if af != bf {
			return af < bf
		}
		// fallback to textual compare of version if numeric equal
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return a.Release < b.Release
	})

	productIDInt, err := strconv.Atoi(productID)
	if err != nil {
		fmt.Println("[!!] Invalid product ID, request to update this program! https://scc.suse.com/api/package_search/products")
		return
	}
	productName := cMapProductIDs[productIDInt]
	fmt.Printf(">> Product ID: %s (%s), RPM Name: <%s>\n", productID, productName, rpmName)

	maxNameLength := len("Package Name")
	maxVersionLength := len("Version")
	maxArchLength := len("Arch") + 2
	maxReleaseLength := len("Release") + 2
	maxRepoLength := len("Repository") + 2

	for _, pkg := range pArrPackages {
		if fuzzy || strings.EqualFold(pkg.Name, rpmName) {
			if len(pkg.Name) > maxNameLength {
				maxNameLength = len(pkg.Name)
			}
			if len(pkg.Version) > maxVersionLength {
				maxVersionLength = len(pkg.Version)
			}
			if len(pkg.Arch) > maxArchLength {
				maxArchLength = len(pkg.Arch)
			}
			if len(pkg.Release) > maxReleaseLength {
				maxReleaseLength = len(pkg.Release)
			}
			if len(pkg.Repo) > maxRepoLength {
				maxRepoLength = len(pkg.Repo)
			}
		}
	}

	fmt.Printf("+-%s-+-%s-+-%s-+-%s-+-%s-+\n", strings.Repeat("-", maxNameLength), strings.Repeat("-", maxVersionLength), strings.Repeat("-", maxArchLength), strings.Repeat("-", maxReleaseLength), strings.Repeat("-", maxRepoLength))
	fmt.Printf("| %-*s | %-*s | %-*s | %-*s | %-*s |\n", maxNameLength, "Package Name", maxVersionLength, "Version", maxArchLength, "Arch", maxReleaseLength, "Release", maxRepoLength, "Repository")
	fmt.Printf("+-%s-+-%s-+-%s-+-%s-+-%s-+\n", strings.Repeat("-", maxNameLength), strings.Repeat("-", maxVersionLength), strings.Repeat("-", maxArchLength), strings.Repeat("-", maxReleaseLength), strings.Repeat("-", maxRepoLength))

	lLines := 0
	for _, pkg := range pArrPackages {
		if fuzzy || strings.EqualFold(pkg.Name, rpmName) {
			fmt.Printf("| %-*s | %-*s | %-*s | %-*s | %-*s |\n",
				maxNameLength, pkg.Name, maxVersionLength, pkg.Version, maxArchLength, pkg.Arch, maxReleaseLength, pkg.Release, maxRepoLength, pkg.Repo)
			lLines++
		}
	}
	fmt.Printf("+-%s-+-%s-+-%s-+-%s-+-%s-+\n", strings.Repeat("-", maxNameLength), strings.Repeat("-", maxVersionLength), strings.Repeat("-", maxArchLength), strings.Repeat("-", maxReleaseLength), strings.Repeat("-", maxRepoLength))
	fmt.Println("Total packages found:", lLines)
}

func fListProductIDs() {
	keys := make([]int, 0, len(cMapProductIDs))
	for k := range cMapProductIDs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return cMapProductIDs[keys[i]] < cMapProductIDs[keys[j]]
	})

	fmt.Println("+------+=-----------------------------------------------------------------=+")
	fmt.Println("|  ID  | Product Name (most popular/supported repos), sorted by name       |")
	fmt.Println("+------+=-----------------------------------------------------------------=+")
	for _, k := range keys {
		fmt.Printf("| %4d | %-65s |\n", k, cMapProductIDs[k])
	}
	fmt.Println("+------+=-----------------------------------------------------------------=+")
}

// ####################################################################################
// main function to parse command line arguments and call the API
// and print the results in a table format, also supports listing available product IDs
// ####################################################################################

func main() {
	var rpmName string
	var productID string
	var verbose bool
	var list bool
	var fuzzy bool
	var version bool

	flag.StringVar(&rpmName, "r", "glibc", "Specify the RPM package name (default: glibc)")
	flag.StringVar(&rpmName, "rpm", "glibc", "Specify the RPM package name (default: glibc)")
	flag.StringVar(&productID, "p", "2795", "Specify the product ID (default: 2795)")
	flag.StringVar(&productID, "product", "2795", "Specify the product ID (default: 2795)")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&list, "l", false, "List available hardcoded product IDs")
	flag.BoolVar(&list, "list", false, "List available hardcoded product IDs")
	flag.BoolVar(&fuzzy, "f", false, "Enable fuzzy output (ignore exact name match)")
	flag.BoolVar(&fuzzy, "fuzzy", false, "Enable fuzzy output (ignore exact name match)")
	flag.BoolVar(&version, "V", false, "Show version and exit")
	flag.BoolVar(&version, "version", false, "Show version and exit")
	flag.Usage = fUsage
	flag.Parse()

	if version {
		if compileDate == "" {
			compileDate = "unknown"
		}
		// [%s.%s, %s.%s, %s (DD.MMY.YYYY)]\n", runtime.GOOS, runtime.GOARCH, runtime.Compiler, runtime.Version(), compileDate)
		fmt.Printf("\ngetrpm version [%s.%s,%s.%s,%s (DD.MMY.YYYY)]\n@(#) $Id: getrpm.go,v 1.27 2026/04/06 09:51:48 ralph Exp $\n", runtime.GOOS, runtime.GOARCH, runtime.Compiler, runtime.Version(), compileDate)
		os.Exit(0)
	}

	if list {
		fListProductIDs()
		return
	}

	packages, err := fGetPackages(rpmName, productID, verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cIntErrAPI)
	}

	if len(packages) == 0 {
		fmt.Println("No packages found.")
		os.Exit(cIntErrDecode)
	}

	fPrintTable(packages, rpmName, productID, fuzzy)
}
