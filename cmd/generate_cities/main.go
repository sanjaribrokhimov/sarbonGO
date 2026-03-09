// generate_cities downloads world cities (lutangar/cities.json), assigns codes (TAS, SAM, DXB style), and writes internal/reference/data/cities.json.gz for embedding.
// Run: go run ./cmd/generate_cities
package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const lutangarURL = "https://raw.githubusercontent.com/lutangar/cities.json/master/cities.json"

type lutangarCity struct {
	Name   string `json:"name"`
	Lat    string `json:"lat"`
	Lng    string `json:"lng"`
	Country string `json:"country"`
}

type cityOut struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	CountryCode string  `json:"country_code"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

func main() {
	outDir := flag.String("out", "internal/reference/data", "output directory")
	flag.Parse()

	// Download
	fmt.Println("Downloading cities.json...")
	resp, err := http.Get(lutangarURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "download: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "status %d\n", resp.StatusCode)
		os.Exit(1)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}

	var raw []lutangarCity
	if err := json.Unmarshal(body, &raw); err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d cities, generating codes...\n", len(raw))
	seen := make(map[string]int) // countryCode+"\t"+code -> count for dedup
	var out []cityOut
	alpha := regexp.MustCompile(`[a-zA-Z]+`)

	for _, c := range raw {
		lat, _ := parseFloat(c.Lat)
		lng, _ := parseFloat(c.Lng)
		country := strings.TrimSpace(c.Country)
		if len(country) != 2 {
			continue
		}
		code := codeFromName(c.Name, alpha)
		if code == "" {
			code = "XXX"
		}
		key := country + "\t" + code
		n := seen[key]
		seen[key] = n + 1
		if n > 0 {
			code = code + fmt.Sprintf("%d", n)
			if len(code) > 8 {
				code = code[:8]
			}
		}
		out = append(out, cityOut{
			Code:        code,
			Name:        strings.TrimSpace(c.Name),
			CountryCode: country,
			Lat:         lat,
			Lng:         lng,
		})
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}
	gzPath := filepath.Join(*outDir, "cities.json.gz")
	f, err := os.Create(gzPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	enc := json.NewEncoder(gz)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(1)
	}
	if err := gz.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "gzip close: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s (%d cities)\n", gzPath, len(out))
}

func codeFromName(name string, alpha *regexp.Regexp) string {
	s := alpha.FindAllString(name, -1)
	var b strings.Builder
	for _, part := range s {
		for _, r := range part {
			if unicode.IsLetter(r) {
				b.WriteRune(unicode.ToUpper(r))
				if b.Len() >= 3 {
					return b.String()
				}
			}
		}
	}
	return b.String()
}

func parseFloat(s string) (float64, bool) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err == nil
}
