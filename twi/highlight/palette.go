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
		Selector: theme.SyntaxNamespace, SelectorClass: theme.SyntaxFunction, SelectorID: theme.SyntaxFunction,
		SelectorPseudo: theme.SyntaxKeyword, Attribute: theme.SyntaxConstant, CSSVariable: theme.SyntaxParameter, Unit: theme.SyntaxKeyword,
		Null: theme.SyntaxConstant, CharEscape: theme.SyntaxConstant, HardBreak: theme.SyntaxPunctuation, Heading: theme.SyntaxFunction,
		HeadingMarker: theme.SyntaxPunctuation, Bold: theme.Foreground, Italic: theme.Foreground, Strike: theme.SyntaxComment,
		Code: theme.SyntaxString, LinkText: theme.SyntaxFunction, Autolink: theme.SyntaxString, URL: theme.SyntaxString,
		URLLink: theme.SyntaxPunctuation, URLTitle: theme.SyntaxString, Entity: theme.SyntaxConstant, CodeBlock: theme.SyntaxString,
		CodeFence: theme.SyntaxPunctuation, CodeLanguage: theme.SyntaxKeyword, RawCodeBlock: theme.Foreground,
		FrontMatterMarker: theme.SyntaxPunctuation, RawFrontMatter: theme.SyntaxComment, BlockquoteMarker: theme.SyntaxPunctuation,
		ListMarker: theme.SyntaxKeyword, TaskMarker: theme.SyntaxKeyword, ThematicBreak: theme.SyntaxPunctuation,
		Prompt: theme.SyntaxKeyword, PromptPrefix: theme.SyntaxComment, Output: theme.Foreground, Template: theme.SyntaxString,
		Decorator: theme.SyntaxFunction, Type: theme.SyntaxNamespace, ClassName: theme.SyntaxNamespace, TagName: theme.SyntaxKeyword,
		AttrName: theme.SyntaxParameter, Doctype: theme.SyntaxComment,
	}
}
