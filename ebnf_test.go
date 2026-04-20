package cog_test

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestEBNFConsistency parses cog.ebnf and validates that every production
// referenced on the right-hand side is defined, every defined production is
// referenced (except entry points), and there are no duplicate definitions.
func TestEBNFConsistency(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("./cog.ebnf")
	if err != nil {
		t.Fatalf("reading cog.ebnf: %v", err)
	}

	src := string(raw)

	defined, referenced := parseEBNF(src)

	// Entry points are allowed to be unreferenced.
	entryPoints := map[string]bool{
		"file":        true,
		"script_file": true,
	}

	// Terminal symbols are defined in the Terminals section
	// and may appear on the RHS without a regular production.
	terminals := map[string]bool{
		"IDENTIFIER": true,
		"INT":        true,
		"FLOAT":      true,
		"STRING":     true,
		"COMMENT":    true,
		"NEWLINE":    true,
		"EOF":        true,
		"letter":     true,
		"digit":      true,
		"character":  true,
		"comment":    true,
	}

	// Check for duplicate definitions.
	seen := make(map[string]bool, len(defined))
	for _, name := range defined {
		if seen[name] {
			t.Errorf("duplicate production: %q", name)
		}

		seen[name] = true
	}

	definedSet := seen

	// Every referenced production must be defined (or be a terminal).
	for name := range referenced {
		if definedSet[name] || terminals[name] {
			continue
		}

		t.Errorf("referenced but not defined: %q", name)
	}

	// Every defined production must be referenced (except entry points and terminals).
	for _, name := range defined {
		if entryPoints[name] || terminals[name] {
			continue
		}

		if !referenced[name] {
			t.Errorf("defined but never referenced: %q", name)
		}
	}
}

// parseEBNF extracts production names (defined) and referenced production names
// from an EBNF grammar string. Returns defined as an ordered slice (to detect
// duplicates in order) and referenced as a set.
func parseEBNF(src string) (defined []string, referenced map[string]bool) {
	referenced = make(map[string]bool)

	nameRe := regexp.MustCompile(`^([a-z][a-z_0-9]*)$`)
	rhsIdentRe := regexp.MustCompile(`[a-z][a-z_0-9]*`)

	lines := strings.Split(src, "\n")

	var currentProduction string
	var pendingName string // name on a solo line, waiting for = on the next
	insideComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle multi-line block comments.
		if insideComment {
			if strings.Contains(trimmed, "*)") {
				insideComment = false
			}

			continue
		}

		if strings.HasPrefix(trimmed, "(*") {
			if !strings.Contains(trimmed[2:], "*)") {
				insideComment = true
			}

			continue
		}

		// Strip inline comments from the line.
		cleaned := stripInlineComments(line)
		trimmedCleaned := strings.TrimSpace(cleaned)

		// Empty line resets state.
		if trimmedCleaned == "" {
			pendingName = ""
			currentProduction = ""

			continue
		}

		// Check for a pending production name: the previous line had just
		// a name, and this line should start with "=".
		if pendingName != "" {
			if strings.HasPrefix(trimmedCleaned, "=") {
				currentProduction = pendingName
				defined = append(defined, pendingName)

				rhs := trimmedCleaned[1:] // everything after =
				collectReferences(rhs, currentProduction, rhsIdentRe, referenced)

				pendingName = ""

				continue
			}

			// Not followed by =, discard pending name.
			pendingName = ""
		}

		// Check if this line is a bare production name (name alone on a line).
		if match := nameRe.FindStringSubmatch(trimmedCleaned); match != nil {
			pendingName = match[1]

			continue
		}

		// Check if this line defines a production inline: name = ...
		if len(cleaned) > 0 && cleaned[0] != ' ' && cleaned[0] != '\t' && cleaned[0] != '|' {
			if idx := strings.Index(cleaned, "="); idx > 0 {
				name := strings.TrimSpace(cleaned[:idx])
				if nameRe.MatchString(name) {
					currentProduction = name
					defined = append(defined, name)

					rhs := cleaned[idx+1:]
					collectReferences(rhs, currentProduction, rhsIdentRe, referenced)

					continue
				}
			}
		}

		// Continuation line for current production.
		if currentProduction != "" {
			collectReferences(cleaned, currentProduction, rhsIdentRe, referenced)
		}
	}

	return defined, referenced
}

// collectReferences finds all lowercase identifiers on an RHS line and adds
// them to the referenced set, excluding the current production name (to avoid
// self-reference counting issues), string literals, and EBNF keywords.
func collectReferences(line, currentProd string, re *regexp.Regexp, refs map[string]bool) {
	// EBNF keywords and meta-symbols that look like identifiers.
	ebnfKeywords := map[string]bool{
		"true": true, "false": true,
		"e": true, // from FLOAT definition: "e" | "E"
	}

	// Remove quoted strings to avoid matching identifiers inside them.
	cleaned := removeQuotedStrings(line)

	for _, m := range re.FindAllString(cleaned, -1) {
		if m == currentProd || ebnfKeywords[m] {
			continue
		}

		refs[m] = true
	}
}

// stripInlineComments removes (* ... *) comments from a single line.
func stripInlineComments(s string) string {
	for {
		start := strings.Index(s, "(*")
		if start < 0 {
			return s
		}

		end := strings.Index(s[start:], "*)")
		if end < 0 {
			return s[:start]
		}

		s = s[:start] + s[start+end+2:]
	}
}

// removeQuotedStrings strips both single- and double-quoted strings.
func removeQuotedStrings(s string) string {
	var b strings.Builder

	inDouble := false
	inSingle := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		switch {
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case !inDouble && !inSingle:
			b.WriteByte(ch)
		}
	}

	return b.String()
}

// TestEBNFProductionCoverage checks that key language constructs each have
// a named production in the grammar.
func TestEBNFProductionCoverage(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("./cog.ebnf")
	if err != nil {
		t.Fatalf("reading cog.ebnf: %v", err)
	}

	defined, _ := parseEBNF(string(raw))

	expected := []string{
		"file",
		"package",
		"statement",
		"expression",
		"block",
		"if_statement",
		"for_statement",
		"switch_statement",
		"match_statement",
		"typed_declaration",
		"combined_type",
		"type",
		"basic_type",
		"struct_type",
		"enum_type",
		"error_type",
		"interface_type",
		"procedure_type",
		"parameter",
		"method_declaration",
		"builtin_expression",
		"call_arguments",
		"type_arguments",
	}

	for _, name := range expected {
		if !slices.Contains(defined, name) {
			t.Errorf("expected production %q not found in grammar", name)
		}
	}
}
