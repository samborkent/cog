package tokens

type Type uint8

const (
	// Single-character tokens.
	LParen    Type = iota + 1 // (
	RParen                    // )
	LBrace                    // {
	RBrace                    // }
	LBracket                  // [
	RBracket                  // ]
	Comma                     // ,
	Dot                       // .
	Colon                     // :
	Semicolon                 // ;
	Plus                      // +
	Minus                     // -
	Asterisk                  // *
	Divide                    // -
	Question                  // ?
	Tilde                     // ~
	Pipe                      // |

	// 1 or 2 character tokens.
	Not         // !
	NotEqual    // !=
	Assign      // =
	Equal       // ==
	GT          // >
	GTEqual     // >=
	LT          // <
	LTEqual     // <=
	Declaration // :=
	BitAnd      // &
	BitXor      // ^
	And         // &&
	Or          // ||

	// Literals
	Identifier
	Bool
	StringLiteral
	IntLiteral
	FloatLiteral

	// Types
	ASCII
	UTF8
	Uint8
	Uint16
	Uint32
	Uint64
	Uint128
	Int8
	Int16
	Int32
	Int64
	Int128
	Float16
	Float32
	Float64
	Complex32
	Complex64
	Complex128

	// Boolean keywords
	True
	False

	// Control-flow keywords
	If
	Else
	For
	Switch
	Select
	Case
	Default
	Return
	Break
	Continue
	Async

	// Function keywords
	Function  // func: pure function, return value manditory, no side-effects
	Procedure // proc: procedure, return value optional, can have side-effects
	Builtin

	// Type keywords
	TypeDecl
	Error
	Any
	Struct
	Enum
	Map
	Set

	// Type interface
	Int     // i8, i16, i32, i64, i128
	Uint    // u8, u16, u32, u64, u128
	Float   // f16, f32, f64
	Complex // c32, c64, c128
	String  // ascii, utf8

	// Import keywords
	Package
	Import
	Export
	GoImport

	// Type qualifiers
	Variable // var

	EOF
)

func (t Type) String() string {
	switch t {
	case LParen:
		return "("
	case RParen:
		return ")"
	case LBrace:
		return "{"
	case RBrace:
		return "}"
	case LBracket:
		return "["
	case RBracket:
		return "]"
	case Comma:
		return ","
	case Dot:
		return "."
	case Colon:
		return ":"
	case Semicolon:
		return ";"
	case Plus:
		return "+"
	case Minus:
		return "-"
	case Asterisk:
		return "*"
	case Divide:
		return "/"
	case Question:
		return "?"
	case Tilde:
		return "~"
	case Pipe:
		return "|"
	case Not:
		return "!"
	case NotEqual:
		return "!="
	case Assign:
		return "="
	case Equal:
		return "=="
	case GT:
		return ">"
	case GTEqual:
		return ">="
	case LT:
		return "<"
	case LTEqual:
		return "<="
	case Declaration:
		return ":="
	case BitAnd:
		return "&"
	case BitXor:
		return "^"
	case And:
		return "&&"
	case Or:
		return "||"
	case Identifier:
		return "identifier"
	case Bool:
		return "bool"
	case StringLiteral:
		return "string_literal"
	case IntLiteral:
		return "int_literal"
	case FloatLiteral:
		return "float_literal"
	case ASCII:
		return "ascii"
	case UTF8:
		return "utf8"
	case Uint8:
		return "uint8"
	case Uint16:
		return "uint16"
	case Uint32:
		return "uint32"
	case Uint64:
		return "uint64"
	case Uint128:
		return "uint128"
	case Int8:
		return "int8"
	case Int16:
		return "int16"
	case Int32:
		return "int32"
	case Int64:
		return "int64"
	case Int128:
		return "int128"
	case Float16:
		return "float16"
	case Float32:
		return "float32"
	case Float64:
		return "float64"
	case Complex32:
		return "complex32"
	case Complex64:
		return "complex64"
	case Complex128:
		return "complex128"
	case True:
		return "true"
	case False:
		return "false"
	case If:
		return "if"
	case Else:
		return "else"
	case For:
		return "for"
	case Switch:
		return "switch"
	case Select:
		return "select"
	case Case:
		return "case"
	case Default:
		return "default"
	case Return:
		return "return"
	case Break:
		return "break"
	case Continue:
		return "continue"
	case Async:
		return "async"
	case Function:
		return "func"
	case Procedure:
		return "proc"
	case Builtin:
		return "@"
	case Error:
		return "error"
	case Any:
		return "any"
	case Struct:
		return "struct"
	case Enum:
		return "enum"
	case Map:
		return "map"
	case Set:
		return "set"
	case Int:
		return "int"
	case Uint:
		return "uint"
	case Float:
		return "float"
	case Complex:
		return "complex"
	case String:
		return "string"
	case Package:
		return "package"
	case Import:
		return "import"
	case Export:
		return "export"
	case GoImport:
		return "goimport"
	case Variable:
		return "var"
	case EOF:
		return "EOF"
	default:
		return "undefined"
	}
}
