package parser

type Scope uint8

const (
	GlobalScope Scope = iota + 1
	ScanScope
	LocalScope
	ImportScope
	GoImportScope
	StructScope
	EnumScope
)
