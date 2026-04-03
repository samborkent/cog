package transpiler

import (
	"strings"
	"testing"
)

func TestPostProcess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		in          string
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:        "no_placeholder",
			in:          "line one\nline two\n",
			wantContain: []string{"line one", "line two"},
		},
		{
			name:        "removes_placeholder_line",
			in:          "// comment\nvar __cogReplaceMe int = 0\nnext line\n",
			wantContain: []string{"// comment", "next line"},
			wantAbsent:  []string{"__cogReplaceMe"},
		},
		{
			name:        "preserves_surrounding_lines",
			in:          "before\nvar __cogReplaceMe int = 0\nafter\n",
			wantContain: []string{"before", "after"},
			wantAbsent:  []string{"__cogReplaceMe"},
		},
		{
			name:        "multiple_placeholders",
			in:          "var __cogReplaceMe int = 0\nmiddle\nvar __cogReplaceMe int = 0\n",
			wantContain: []string{"middle"},
			wantAbsent:  []string{"__cogReplaceMe"},
		},
		{
			name: "empty_input",
			in:   "",
		},
		{
			name:       "only_placeholder",
			in:         "var __cogReplaceMe int = 0\n",
			wantAbsent: []string{"__cogReplaceMe"},
		},
		{
			name:       "indented_placeholder",
			in:         "\tvar __cogReplaceMe int = 0\n",
			wantAbsent: []string{"__cogReplaceMe"},
		},
		{
			name:        "single_line_block_comment",
			in:          "//__cog_block__ block comment \n",
			wantContain: []string{"/* block comment  */"},
			wantAbsent:  []string{"__cog_block__"},
		},
		{
			name:        "multi_line_block_comment",
			in:          "//__cog_block__ multi\n//__cog_block__   line\n//__cog_block__   comment \n",
			wantContain: []string{"/* multi", "  line", "  comment  */"},
			wantAbsent:  []string{"__cog_block__"},
		},
		{
			name:        "indented_block_comment",
			in:          "\t//__cog_block__ inside block \n",
			wantContain: []string{"\t/* inside block  */"},
			wantAbsent:  []string{"__cog_block__"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := string(postProcess([]byte(tt.in)))
			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\ngot: %q", want, got)
				}
			}

			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("output should not contain %q\ngot: %q", absent, got)
				}
			}
		})
	}
}
