package transpiler

import (
	goast "go/ast"
	gotoken "go/token"
	"strings"
)

// blockCommentMarker is a sentinel prefix used to mark lines that originated
// from a /* */ block comment. Post-processing reconstructs them back to /* */.
const blockCommentMarker = "__cog_block__"

// commentDecl produces a Go declaration that carries the given comment text.
// The declaration is a dummy variable (`var __cogReplaceMe int = 0`)
// that will be stripped during post-processing, leaving only the comment in the output.
// Block comments (/* */) are marked for reconstruction during post-processing.
func (t *Transpiler) commentDecl(text string) []goast.Decl {
	comments := toLineComments(text)

	return []goast.Decl{&goast.GenDecl{
		Doc: &goast.CommentGroup{
			List: comments,
		},
		Tok: gotoken.VAR,
		Specs: []goast.Spec{
			&goast.ValueSpec{
				Names:  []*goast.Ident{{Name: "__cogReplaceMe"}},
				Type:   &goast.Ident{Name: "int"},
				Values: []goast.Expr{&goast.BasicLit{Kind: gotoken.INT, Value: "0"}},
			},
		},
	}}
}

// toLineComments converts comment text into goast.Comment entries.
// Line comments (//) pass through directly.
// Block comments (/* */) are split and each line is marked with blockCommentMarker
// so post-processing can reconstruct the original /* */ form.
func toLineComments(text string) []*goast.Comment {
	if strings.HasPrefix(text, "//") {
		return []*goast.Comment{{Text: text}}
	}

	// Strip /* and */ delimiters.
	body := strings.TrimPrefix(text, "/*")
	body = strings.TrimSuffix(body, "*/")

	lines := strings.Split(body, "\n")
	comments := make([]*goast.Comment, 0, len(lines))

	for _, line := range lines {
		comments = append(comments, &goast.Comment{
			Text: "//" + blockCommentMarker + line,
		})
	}

	return comments
}
