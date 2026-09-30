package highlight

import "github.com/twind-dev/twind/twi/theme"

type Palette [kindEnd]theme.Token

func DefaultPalette() Palette {
	return Palette{
		Text: theme.Foreground, Keyword: theme.SyntaxKeyword, String: theme.SyntaxString, Escape: theme.SyntaxConstant,
		Number: theme.SyntaxNumber, Comment: theme.SyntaxComment, Function: theme.SyntaxFunction, Operator: theme.SyntaxKeyword,
		Punctuation: theme.SyntaxPunctuation, Identifier: theme.Foreground, Property: theme.SyntaxConstant, Boolean: theme.SyntaxConstant,
		Variable: theme.SyntaxParameter, Builtin: theme.SyntaxConstant, Regex: theme.SyntaxString, Datetime: theme.SyntaxNumber,
		TableHeader: theme.SyntaxNamespace, Namespace: theme.SyntaxNamespace, Parameter: theme.SyntaxParameter, Constant: theme.SyntaxConstant,
	}
}
