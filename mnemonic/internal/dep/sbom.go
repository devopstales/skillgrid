package dep

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Pkg is one parsed SBOM component (an ingestable dependency node).
type Pkg struct {
	Purl      string `json:"purl"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Ecosystem string `json:"ecosystem,omitempty"`
}

// Edge is one dependency relation: From depends on To.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type cdxComponent struct {
	Type    string `json:"type"`
	BomRef  string `json:"bom-ref"`
	Purl    string `json:"purl"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type cdxDependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

type cdxBOM struct {
	BomFormat  string          `json:"bomFormat"`
	Components []cdxComponent  `json:"components"`
	Deps       []cdxDependency `json:"dependencies"`
}

type spdxDocument struct {
	SPDXVersion string `json:"spdxVersion"`
}

// ParseSBOM parses a CycloneDX 1.5 SBOM (components + dependencies) into the
// Pkg set and the dependency edges. CycloneDX is auto-detected by the
// bomFormat field; SPDX is a declared fallback that is not yet implemented.
func ParseSBOM(data []byte) ([]Pkg, []Edge, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil, errors.New("sbom is empty")
	}
	var probe struct {
		BomFormat   string `json:"bomFormat"`
		SpdxVersion string `json:"spdxVersion"`
	}
	if err := json.Unmarshal(trimmed, &probe); err != nil {
		return nil, nil, fmt.Errorf("parse sbom json: %w", err)
	}
	switch {
	case strings.EqualFold(probe.BomFormat, "CycloneDX"):
		return parseCycloneDX(trimmed)
	case probe.BomFormat == "" && probe.SpdxVersion != "":
		return nil, nil, errors.New("SPDX not yet supported")
	case strings.EqualFold(probe.BomFormat, "SPDX"):
		return nil, nil, errors.New("SPDX not yet supported")
	default:
		return nil, nil, fmt.Errorf("unrecognized SBOM format %q", probe.BomFormat)
	}
}

func parseCycloneDX(data []byte) ([]Pkg, []Edge, error) {
	var bom cdxBOM
	if err := json.Unmarshal(data, &bom); err != nil {
		return nil, nil, fmt.Errorf("parse cyclonedx: %w", err)
	}
	seen := map[string]bool{}
	var pkgs []Pkg
	for _, c := range bom.Components {
		purl := c.BomRef
		if purl == "" {
			purl = c.Purl
		}
		if purl == "" {
			purl = buildPurl(c.Name, c.Version)
		}
		if purl == "" || seen[purl] {
			continue
		}
		seen[purl] = true
		pkgs = append(pkgs, Pkg{
			Purl:      purl,
			Name:      c.Name,
			Version:   c.Version,
			Ecosystem: ecosystemOf(c.Type, purl),
		})
	}
	edgeSeen := map[string]bool{}
	var edges []Edge
	for _, d := range bom.Deps {
		for _, dep := range d.DependsOn {
			key := d.Ref + "->" + dep
			if edgeSeen[key] {
				continue
			}
			edgeSeen[key] = true
			edges = append(edges, Edge{From: d.Ref, To: dep})
		}
	}
	if len(pkgs) == 0 {
		return nil, nil, errors.New("sbom has no components")
	}
	return pkgs, edges, nil
}

// buildPurl assembles a purl from a bare name/version when the SBOM provides
// no bom-ref/purl. The type segment is "generic" (no ecosystem signal).
func buildPurl(name, version string) string {
	if name == "" {
		return ""
	}
	if version == "" {
		return "pkg:generic/" + name
	}
	return "pkg:generic/" + name + "@" + version
}

// ecosystemOf derives the ecosystem column value: an explicit purl type wins,
// otherwise the component type is used as a best-effort label.
func ecosystemOf(componentType, purl string) string {
	if idx := strings.LastIndex(purl, "/"); idx > 0 {
		seg := purl[:idx]
		if parts := strings.SplitN(seg, ":", 2); len(parts) == 2 {
			return parts[1]
		}
	}
	if componentType != "" {
		return componentType
	}
	return ""
}
