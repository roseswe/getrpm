package main

// small helper tool to list all available SUSE products and their IDs for getrpm.go
// This tool uses the SUSE Customer Center API to fetch the product list.
// The output is a list of product IDs and their names.   -->   rmt-cli products list --all
// The output can be used as input for the getrpm.go tool.
// The output can be formatted in one line or in a table.
// (c) by ROSE SWE, Ralph Roth -- https://github.com/roseswe/getrpm

/* TODO:

Activating SLL-HA 7 x86_64 ...
Error: Registration server returned 'No product found on SCC for: SLL-HA 7 x86_64 json api/connect/v4/systems/products activate' (422)

*/

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

const (
	// go fix: ./listprodids.go:29:2: cStrAPIURL redeclared in this block
	cStrAPIURL    = "https://scc.suse.com/api/package_search/products" // see also https://scc.suse.com/api/products/downloads
	cIntErrAPI    = 65                                                 // Fehlerhafte API-Anfrage
	cIntErrDecode = 66                                                 // Fehler beim Dekodieren der Antwort
)

// https://scc.suse.com/api/package_search/v4/documentation

// curl https://scc.suse.com/api/package_search/products | jq

// $ curl -sL -X GET -H 'accept: application/json' -H 'Accept: application/vnd.scc.suse.com.v4+json' 'https://scc.suse.com/api/package_search/packages?product_id=2136&query=kernel-default' | jq -e -r '.data[]' | grep -B 3 204.1

// $ curl -s "https://scc.suse.com/api/package_search/products"
//{"data":[{"id":2538,"name":"SUSE Liberty Linux","identifier":"SLL/9/x86_64","type":"base","free":false,"architecture":"x86_64","version":"9"},
//'{"id":1921,"name":"SUSE Liberty Linux","identifier":"RES/8/x86_64","type":"base","free":false,"architecture":"x86_64","version":"8"}

type tStructProduct struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	Identifier   string `json:"identifier"`
}

type tStructAPIResponse struct {
	Data []tStructProduct `json:"data"`
}

func main() {
	var lBooHelp bool
	var lBooVersion bool
	var lBooOneLine bool
	var lVerCheck bool
	currentDate := time.Now().Format("02.01.2006")

	flag.BoolVar(&lBooHelp, "h", false, "Show help message")
	flag.BoolVar(&lBooHelp, "help", false, "Show help message")
	flag.BoolVar(&lBooVersion, "V", false, "Show version information")
	flag.BoolVar(&lBooVersion, "version", false, "Show version information")
	flag.BoolVar(&lBooOneLine, "1", false, "Print information in one line")
	flag.BoolVar(&lBooOneLine, "one", false, "Print information in one line")
	flag.BoolVar(&lVerCheck, "vercheck", false, "Print information for vercheck.py (needs rework)")
	flag.Parse()

	if lBooHelp {
		fmt.Println("(c) by ROSE SWE, Ralph Roth -- https://github.com/roseswe/getrpm")
		fmt.Println("\nUsage: ListProdIDs [options]")
		fmt.Println("Fetch and displays SUSE product IDs from the SCC API (to be used with getrpm).")
		fmt.Println("\n== Options:")
		fmt.Println("  -?, -h, --help   Show this help message")
		fmt.Println("  -V, --version    Show version information")
		fmt.Println("  -1, --one        Print information in one line")
		fmt.Println("  --vercheck       Print information (almost) suitable for vercheck.py")
		fmt.Println("\n== Hint")
		fmt.Println("  ./listprodis -1 | sort > listprodis2.txt; sdiff -s listprodis.txt listprodis2.txt")
		os.Exit(0)
	}

	if lBooVersion {
		// go build -ldflags "-X main.compileDate=$(date +%d.%m.%Y)" -o listprodids listprodids.go
		fmt.Printf("\nListProdIDs version [%s,%s]: @(#) $Id: listprodids.go,v 1.17 2026/04/21 12:03:01 ralph Exp $\n", runtime.GOOS, runtime.Version())
		os.Exit(0)
	}

	resp, err := http.Get(cStrAPIURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: API request failed: %v\n", err)
		os.Exit(cIntErrAPI)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to read API response: %v\n", err)
		os.Exit(cIntErrDecode)
	}

	var lStructResponse tStructAPIResponse
	if err := json.Unmarshal(body, &lStructResponse); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to decode API response: %v\n", err)
		os.Exit(cIntErrDecode)
	}

	//fmt.Println("Currently supported SUSE Product IDs:")
	if lBooOneLine {
		for _, product := range lStructResponse.Data {
			fmt.Printf("%d: \"%s %s %s\",\n", product.ID, product.Name, product.Version, product.Architecture)
		}
	} else if lVerCheck {
		for _, product := range lStructResponse.Data {
			fmt.Printf("%d: {'name': '%s %s %s', 'arch': '%s', 'identifier': '%s'},\n", product.ID, product.Name, product.Version, product.Architecture, product.Architecture, product.Identifier)
		}
		// TODO: vercheck.py - 1319: {'name': 'SUSE Linux Enterprise Server for SAP Applications 12 x86_64', 'arch': 'x86_64', 'identifier': 'cpe:/o:suse:sles_sap:12'},
		// see also issue #38 - https://github.com/doccaz/scc-tools/issues/38#issuecomment-2686898724
	} else {
		fmt.Println("+------+=------------------------------------------------------------------=+------------------+--------------+")
		fmt.Printf("|  ID  | Product Name  (Currently supported SUSE Product IDs) %s    | Version          | Architecture |\n", currentDate)
		fmt.Println("+------+=------------------------------------------------------------------=+------------------+--------------+")
		for _, product := range lStructResponse.Data {
			fmt.Printf("| %-4d | %-66s | %-16s | %-12s |\n", product.ID, product.Name, product.Version, product.Architecture)
		}
		fmt.Println("+------+=------------------------------------------------------------------=+------------------+--------------+")
	}
}

// EOF, (c) by ROSE SWE, Ralph Roth
