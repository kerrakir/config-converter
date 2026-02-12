package main

import (
	"fmt"
	"strings"
	"unicode"

	"converter/model"
)

type interfaceMapping struct {
	from string
	to   string
}

type interfaceTransformOptions struct {
	indexStyle      string
	threePartPrefix string
}

func parseInterfaceMappings(raw string) ([]interfaceMapping, error) {
	parts := strings.Split(raw, ",")
	result := make([]interfaceMapping, 0, len(parts))
	for _, part := range parts {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("invalid -if-map format: %q", part)
		}
		from := normalizeIfaceType(pair[0])
		to := strings.TrimSpace(pair[1])
		if from == "" || to == "" {
			return nil, fmt.Errorf("invalid -if-map entry: %q", part)
		}
		result = append(result, interfaceMapping{
			from: from,
			to:   to,
		})
	}
	return result, nil
}

func applyInterfaceMappings(cfg *model.Config, mappings []interfaceMapping) {
	for i := range cfg.Interfaces {
		cfg.Interfaces[i].Name = mapInterfaceName(cfg.Interfaces[i].Name, mappings, interfaceTransformOptions{indexStyle: "keep", threePartPrefix: "1"})
	}
	for i := range cfg.NAT {
		cfg.NAT[i].Inside = mapInterfaceName(cfg.NAT[i].Inside, mappings, interfaceTransformOptions{indexStyle: "keep", threePartPrefix: "1"})
		cfg.NAT[i].Outside = mapInterfaceName(cfg.NAT[i].Outside, mappings, interfaceTransformOptions{indexStyle: "keep", threePartPrefix: "1"})
	}
	for i := range cfg.NATRule {
		cfg.NATRule[i].Outside = mapInterfaceName(cfg.NATRule[i].Outside, mappings, interfaceTransformOptions{indexStyle: "keep", threePartPrefix: "1"})
	}
}

func applyInterfaceTransformations(cfg *model.Config, mappings []interfaceMapping, opts interfaceTransformOptions) {
	for i := range cfg.Interfaces {
		cfg.Interfaces[i].Name = mapInterfaceName(cfg.Interfaces[i].Name, mappings, opts)
	}
	for i := range cfg.NAT {
		cfg.NAT[i].Inside = mapInterfaceName(cfg.NAT[i].Inside, mappings, opts)
		cfg.NAT[i].Outside = mapInterfaceName(cfg.NAT[i].Outside, mappings, opts)
	}
	for i := range cfg.NATRule {
		cfg.NATRule[i].Outside = mapInterfaceName(cfg.NATRule[i].Outside, mappings, opts)
	}
	for i := range cfg.OSPFNoPassiveIfaces {
		cfg.OSPFNoPassiveIfaces[i] = mapInterfaceName(cfg.OSPFNoPassiveIfaces[i], mappings, opts)
	}
}

func mapInterfaceName(name string, mappings []interfaceMapping, opts interfaceTransformOptions) string {
	typ, suffix := splitInterfaceName(name)
	normalizedType := normalizeIfaceType(typ)
	if normalizedType == "" || strings.HasPrefix(normalizedType, "vlan") {
		return name
	}
	targetType := typ
	for _, mapping := range mappings {
		if normalizedType == mapping.from {
			targetType = mapping.to
			break
		}
	}

	separator := ""
	if strings.HasPrefix(suffix, " ") {
		separator = " "
	}
	transformedSuffix := transformInterfaceSuffix(strings.TrimSpace(suffix), opts)
	if transformedSuffix == "" {
		return targetType
	}
	return targetType + separator + transformedSuffix
}

func splitInterfaceName(name string) (string, string) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ""
	}
	fields := strings.Fields(trimmed)
	if len(fields) >= 2 && startsWithDigit(fields[1]) {
		return fields[0], strings.TrimPrefix(trimmed, fields[0])
	}
	if slash := strings.Index(trimmed, "/"); slash > 0 {
		boundary := slash
		for boundary > 0 && unicode.IsDigit(rune(trimmed[boundary-1])) {
			boundary--
		}
		if boundary > 0 {
			return strings.TrimSpace(trimmed[:boundary]), trimmed[boundary:]
		}
	}
	boundary := len(trimmed)
	for boundary > 0 && unicode.IsDigit(rune(trimmed[boundary-1])) {
		boundary--
	}
	if boundary > 0 && boundary < len(trimmed) {
		return strings.TrimSpace(trimmed[:boundary]), trimmed[boundary:]
	}
	for i, r := range trimmed {
		if unicode.IsDigit(r) && i > 0 {
			return strings.TrimSpace(trimmed[:i]), trimmed[i:]
		}
	}
	return trimmed, ""
}

func startsWithDigit(s string) bool {
	if s == "" {
		return false
	}
	return unicode.IsDigit(rune(s[0]))
}

func normalizeIfaceType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

func transformInterfaceSuffix(suffix string, opts interfaceTransformOptions) string {
	if suffix == "" {
		return suffix
	}
	parts := strings.Split(suffix, "/")
	if !allPartsNumeric(parts) {
		return suffix
	}

	switch opts.indexStyle {
	case "2":
		if len(parts) == 3 {
			return parts[1] + "/" + parts[2]
		}
	case "3":
		if len(parts) == 2 {
			prefix := strings.TrimSpace(opts.threePartPrefix)
			if prefix == "" {
				prefix = "1"
			}
			return prefix + "/" + parts[0] + "/" + parts[1]
		}
	}
	return suffix
}

func allPartsNumeric(parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if !unicode.IsDigit(r) {
				return false
			}
		}
	}
	return true
}
