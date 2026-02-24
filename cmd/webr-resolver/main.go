package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/noatgnu/cauldron-go/internal/webrpkg"
)

func main() {
	webrVersion := flag.String("webr-version", "0.5.0", "WebR version to use")
	outputFormat := flag.String("format", "json", "Output format: json, list, or order")
	flag.Parse()

	packages := flag.Args()
	if len(packages) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: webr-resolver [--webr-version VERSION] [--format json|list|order] PACKAGE...")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Initializing resolver for WebR %s...\n", *webrVersion)
	resolver, err := webrpkg.NewResolver(*webrVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Detected R version: %s (major.minor: %s)\n", resolver.GetRVersion(), resolver.GetRMajorMinor())
	fmt.Fprintf(os.Stderr, "Resolving dependencies for: %v\n", packages)

	result, err := resolver.Resolve(packages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving: %v\n", err)
		os.Exit(1)
	}

	switch *outputFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)

	case "list":
		for _, pkg := range result.ResolvedPkgs {
			status := "available"
			if !pkg.Available {
				status = "unavailable"
			}
			fmt.Printf("%s\t%s\t%s\n", pkg.Name, pkg.Repository, status)
		}

	case "order":
		for _, pkg := range result.InstallOrder {
			fmt.Println(pkg)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", *outputFormat)
		os.Exit(1)
	}

	if len(result.Unavailable) > 0 {
		fmt.Fprintf(os.Stderr, "\nWarning: %d packages unavailable: %v\n", len(result.Unavailable), result.Unavailable)
	}

	fmt.Fprintf(os.Stderr, "\nResolved %d packages total, install order has %d packages\n",
		len(result.ResolvedPkgs), len(result.InstallOrder))
}
