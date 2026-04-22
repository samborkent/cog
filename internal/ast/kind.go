package ast

type NodeKind uint8

const KindNone = NodeKind(0)

const (
	KindArrayLiteral NodeKind = iota + 1
	KindASCIILiteral
	KindAssignment
	KindBlock
	KindBoolLiteral
	KindBranch
	KindBuiltin
	KindCall
	KindCase
	KindComment
	KindComplex128Literal
	KindComplex32Literal
	KindComplex64Literal
	KindDeclaration
	KindDefault
	KindEitherLiteral
	KindExpressionStatement
	KindFile
	KindFloat16Literal
	KindFloat32Literal
	KindFloat64Literal
	KindForStatement
	KindGoCallExpression
	KindGoImport
	KindIdentifier
	KindIfStatement
	KindImport
	KindIndex
	KindInfix
	KindInt128Literal
	KindInt16Literal
	KindInt32Literal
	KindInt64Literal
	KindInt8Literal
	KindLabel
	KindMapLiteral
	KindMatch
	KindMatchCase
	KindMethod
	KindPackage
	KindParameter
	KindPrefix
	KindProcedureLiteral
	KindResultLiteral
	KindReturn
	KindSelector
	KindSetLiteral
	KindSliceLiteral
	KindStructLiteral
	KindSuffix
	KindSwitch
	KindTupleLiteral
	KindType
	KindUint128Literal
	KindUint16Literal
	KindUint32Literal
	KindUint64Literal
	KindUint8Literal
	KindUTF8Literal
)

func (k NodeKind) IsStatement() bool {
	return k == KindAssignment ||
		k == KindBranch ||
		k == KindExpressionStatement ||
		k == KindForStatement ||
		k == KindIfStatement ||
		k == KindMatch ||
		k == KindReturn ||
		k == KindSwitch ||
		k == KindDeclaration ||
		k == KindImport ||
		k == KindMethod ||
		k == KindPackage ||
		k == KindParameter ||
		k == KindType
}

func (k NodeKind) IsLiteral() bool {
	return k == KindArrayLiteral ||
		k == KindASCIILiteral ||
		k == KindBoolLiteral ||
		k == KindComplex128Literal ||
		k == KindComplex32Literal ||
		k == KindComplex64Literal ||
		k == KindEitherLiteral ||
		k == KindFloat16Literal ||
		k == KindFloat32Literal ||
		k == KindFloat64Literal ||
		k == KindInt128Literal ||
		k == KindInt16Literal ||
		k == KindInt32Literal ||
		k == KindInt64Literal ||
		k == KindInt8Literal ||
		k == KindMapLiteral ||
		k == KindProcedureLiteral ||
		k == KindResultLiteral ||
		k == KindSetLiteral ||
		k == KindSliceLiteral ||
		k == KindStructLiteral ||
		k == KindTupleLiteral ||
		k == KindUint128Literal ||
		k == KindUint16Literal ||
		k == KindUint32Literal ||
		k == KindUint64Literal ||
		k == KindUint8Literal ||
		k == KindUTF8Literal
}

func (k NodeKind) IsExpression() bool {
	return !k.IsStatement()
}
