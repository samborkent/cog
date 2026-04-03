package transpiler

import (
	"bytes"
	"fmt"
	goast "go/ast"
	goprinter "go/printer"
	"io"
)

// Print renders the Go AST file to w, with post-processing to:
//   - strip placeholder declarations used to carry comments
//   - reconstruct block comments from marker lines back to /* */ form
func (t *Transpiler) Print(w io.Writer, gofile *goast.File) error {
	var buf bytes.Buffer

	if err := goprinter.Fprint(&buf, t.fset, gofile); err != nil {
		return fmt.Errorf("printing Go AST: %w", err)
	}

	out := postProcess(buf.Bytes())

	if _, err := w.Write(out); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	return nil
}

var (
	placeholderSentinel = []byte("__cogReplaceMe")
	blockMarkerPrefix   = []byte("//" + blockCommentMarker)
)

// postProcess removes placeholder lines and reconstructs block comments.
func postProcess(data []byte) []byte {
	lines := bytes.Split(data, []byte("\n"))
	result := make([][]byte, 0, len(lines))

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := bytes.TrimLeft(line, " \t")

		// Reconstruct block comments from consecutive marker lines.
		if bytes.HasPrefix(trimmed, blockMarkerPrefix) {
			indent := line[:len(line)-len(trimmed)]

			var blockLines [][]byte

			for i < len(lines) {
				tr := bytes.TrimLeft(lines[i], " \t")
				if !bytes.HasPrefix(tr, blockMarkerPrefix) {
					break
				}

				content := bytes.TrimPrefix(tr, blockMarkerPrefix)
				blockLines = append(blockLines, content)
				i++
			}

			if len(blockLines) == 1 {
				var out []byte

				out = append(out, indent...)
				out = append(out, "/*"...)
				out = append(out, blockLines[0]...)
				out = append(out, " */"...)
				result = append(result, out)
			} else {
				// First line: /*content
				first := append(append([]byte{}, indent...), "/*"...)
				first = append(first, blockLines[0]...)
				result = append(result, first)

				// Middle lines: preserve original indentation
				for _, bl := range blockLines[1 : len(blockLines)-1] {
					mid := append([]byte{}, indent...)
					mid = append(mid, bl...)
					result = append(result, mid)
				}

				// Last line: content */
				last := append([]byte{}, indent...)
				last = append(last, blockLines[len(blockLines)-1]...)
				last = append(last, " */"...)
				result = append(result, last)
			}

			continue
		}

		// Strip placeholder lines.
		if !bytes.Contains(line, placeholderSentinel) {
			result = append(result, line)
		}

		i++
	}

	return bytes.Join(result, []byte("\n"))
}
