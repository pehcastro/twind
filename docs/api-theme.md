# twi/theme

Every exported name in `twi/theme`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.

## Theme

type Token uint8\
const Background Token\
const Foreground Token\
const Card Token\
const CardForeground Token\
const Popover Token\
const PopoverForeground Token\
const Primary Token\
const PrimaryForeground Token\
const Secondary Token\
const SecondaryForeground Token\
const Muted Token\
const MutedForeground Token\
const Accent Token\
const AccentForeground Token\
const Destructive Token\
const Border Token\
const Input Token\
const Ring Token\
const Sidebar Token\
const SidebarForeground Token\
const SidebarPrimary Token\
const SidebarPrimaryForeground Token\
const SidebarAccent Token\
const SidebarAccentForeground Token\
const SidebarBorder Token\
const SidebarRing Token\
const Chart1 Token\
const Chart2 Token\
const Chart3 Token\
const Chart4 Token\
const Chart5 Token\
const Selection Token\
const SelectionForeground Token\
const SyntaxKeyword Token\
const SyntaxString Token\
const SyntaxNumber Token\
const SyntaxComment Token\
const SyntaxFunction Token\
const SyntaxConstant Token\
const SyntaxNamespace Token\
const SyntaxParameter Token\
const SyntaxPunctuation Token\
const DestructiveForeground Token\
const Primary50 Token\
const Primary100 Token\
const Primary200 Token\
const Primary300 Token\
const Primary400 Token\
const Primary500 Token\
const Primary600 Token\
const Primary700 Token\
const Primary800 Token\
const Primary900 Token\
const Primary950 Token\
const Accent50 Token\
const Accent100 Token\
const Accent200 Token\
const Accent300 Token\
const Accent400 Token\
const Accent500 Token\
const Accent600 Token\
const Accent700 Token\
const Accent800 Token\
const Accent900 Token\
const Accent950 Token\
func ParseToken(name string) (Token, bool)\
func (Token) String() string\
type Scheme uint8\
const Light Scheme\
const Dark Scheme\
type Tokens \[tokenEnd\]color.Color\
type Theme struct\
func Builtin() \[\]Theme\
func Default() Theme\
Theme.Name string\
Theme.Scheme Scheme\
Theme.Tokens Tokens\
Theme.Schemes \[Dark + 1\]Tokens\
func (Theme) WithScheme(s Scheme) Theme
