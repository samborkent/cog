package component

import "unicode"

// ConvertExport adjusts identifier casing for Go export rules.
func ConvertExport(name string, exported, global bool) string {
	// Get first letter.
	r := rune(name[0])

	if exported && !unicode.IsUpper(r) {
		// If exported but not uppercase, uppercase first letter.
		return string(unicode.ToUpper(r)) + name[1:]
	}

	if !exported && unicode.IsUpper(r) && global {
		// If not exported but uppercase and global, prefix by underscore, to prevent exporting.
		return "_" + name
	}

	return name
}
