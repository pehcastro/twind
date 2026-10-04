# twi/ui

Every exported name in `twi/ui`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.

## Accordion

type AccordionType uint8\
const Single AccordionType\
const Multiple AccordionType\
type Accordion struct\
func NewAccordion(rt \*twi.Runtime) \*Accordion\
Accordion.Type AccordionType\
Accordion.Collapsible bool\
Accordion.Value \[\]string\
Accordion.OnChange func(\[\]string)\
func (\*Accordion) Content(value string, children ...twi.NodeOption) twi.Node\
func (\*Accordion) Item(value string, children ...twi.NodeOption) twi.Node\
func (\*Accordion) Node(children ...twi.NodeOption) twi.Node\
func (\*Accordion) Trigger(value string, children ...twi.NodeOption) twi.Node\
Accordion.Disabled bool\
Accordion.Invalid bool\
type Collapsible struct\
func NewCollapsible(rt \*twi.Runtime) \*Collapsible\
func (\*Collapsible) AsTrigger() twi.NodeOption\
func (\*Collapsible) Content(children ...twi.NodeOption) twi.Node\
func (\*Collapsible) Node(children ...twi.NodeOption) twi.Node\
func (\*Collapsible) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node\
Collapsible.Key string\
Collapsible.Open bool\
Collapsible.OnOpenChange func(bool)\
Collapsible.Disabled bool\
Collapsible.Invalid bool

## Alert

func Alert(v Variant, children ...twi.NodeOption) twi.Node\
func AlertTitle(children ...twi.NodeOption) twi.Node\
func AlertDescription(children ...twi.NodeOption) twi.Node

## AspectRatio

func AspectRatio(children ...twi.NodeOption) twi.Node

## Attachment

type Upload uint8\
const Done Upload\
const Idle Upload\
const Uploading Upload\
const Processing Upload\
const Failed Upload\
func Attachment(u Upload, o Orientation, children ...twi.NodeOption) twi.Node\
func AttachmentMedia(v Variant, children ...twi.NodeOption) twi.Node\
func AttachmentContent(children ...twi.NodeOption) twi.Node\
func AttachmentTitle(children ...twi.NodeOption) twi.Node\
func AttachmentDescription(children ...twi.NodeOption) twi.Node\
func AttachmentActions(children ...twi.NodeOption) twi.Node\
func AttachmentAction(children ...twi.NodeOption) twi.Node\
func AttachmentTrigger(children ...twi.NodeOption) twi.Node\
func AttachmentGroup(children ...twi.NodeOption) twi.Node

## Avatar

func Avatar(s Size, children ...twi.NodeOption) twi.Node\
func AvatarFallback(children ...twi.NodeOption) twi.Node

## Badge

func Badge(v Variant, children ...twi.NodeOption) twi.Node

## Breadcrumb

func Breadcrumb(children ...twi.NodeOption) twi.Node\
func BreadcrumbList(children ...twi.NodeOption) twi.Node\
func BreadcrumbItem(children ...twi.NodeOption) twi.Node\
func BreadcrumbLink(children ...twi.NodeOption) twi.Node\
func BreadcrumbPage(children ...twi.NodeOption) twi.Node\
func BreadcrumbSeparator(children ...twi.NodeOption) twi.Node\
func BreadcrumbEllipsis(children ...twi.NodeOption) twi.Node

## Bubble

func BubbleGroup(children ...twi.NodeOption) twi.Node\
func Bubble(v Variant, a Alignment, children ...twi.NodeOption) twi.Node\
func BubbleContent(children ...twi.NodeOption) twi.Node\
func BubbleReactions(s Side, a Alignment, children ...twi.NodeOption) twi.Node

## Button

func Button(v Variant, s Size, children ...twi.NodeOption) twi.Node

## ButtonGroup

func ButtonGroup(o Orientation, children ...twi.NodeOption) twi.Node\
func ButtonGroupSeparator(o Orientation, children ...twi.NodeOption) twi.Node\
func ButtonGroupText(children ...twi.NodeOption) twi.Node

## Calendar

type Calendar struct\
func NewCalendar(rt \*twi.Runtime) \*Calendar\
Calendar.Month time.Time\
Calendar.Selected time.Time\
Calendar.Today time.Time\
Calendar.OnSelect func(time.Time)\
func (\*Calendar) Node(options ...twi.NodeOption) twi.Node\
Calendar.Disabled bool\
Calendar.Invalid bool

## Card

func Card(children ...twi.NodeOption) twi.Node\
func CardHeader(children ...twi.NodeOption) twi.Node\
func CardTitle(children ...twi.NodeOption) twi.Node\
func CardDescription(children ...twi.NodeOption) twi.Node\
func CardAction(children ...twi.NodeOption) twi.Node\
func CardContent(children ...twi.NodeOption) twi.Node\
func CardFooter(children ...twi.NodeOption) twi.Node

## Carousel

type Carousel struct\
func NewCarousel(rt \*twi.Runtime) \*Carousel\
Carousel.Orientation Orientation\
Carousel.Index int\
Carousel.OnChange func(int)\
func (\*Carousel) Content(children ...twi.NodeOption) twi.Node\
func (\*Carousel) Item(children ...twi.NodeOption) twi.Node\
func (\*Carousel) Next(children ...twi.NodeOption) twi.Node\
func (\*Carousel) Node(children ...twi.NodeOption) twi.Node\
func (\*Carousel) Previous(children ...twi.NodeOption) twi.Node\
Carousel.Disabled bool\
Carousel.Invalid bool

## Checkbox

type Checkbox struct\
func NewCheckbox(rt \*twi.Runtime) \*Checkbox\
Checkbox.Checked bool\
Checkbox.OnChange func(bool)\
func (\*Checkbox) Node(options ...twi.NodeOption) twi.Node\
Checkbox.Disabled bool\
Checkbox.Invalid bool

## Combobox

type Combobox struct\
func NewCombobox(rt \*twi.Runtime) \*Combobox\
Combobox.Value string\
Combobox.Placeholder string\
Combobox.Empty string\
Combobox.ShowClear bool\
Combobox.OnChange func(string)\
func (\*Combobox) Content(children ...twi.NodeOption) twi.Node\
func (\*Combobox) Input(options ...twi.NodeOption) twi.Node\
func (\*Combobox) Item(value, label string, children ...twi.NodeOption) twi.Node\
func (\*Combobox) Node(children ...twi.NodeOption) twi.Node\
func (\*Combobox) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node\
Combobox.Side Side\
Combobox.Align Alignment\
Combobox.Key string\
Combobox.Open bool\
Combobox.OnOpenChange func(bool)\
Combobox.Disabled bool\
Combobox.Invalid bool

## Command

type CommandItem struct\
type CommandGroup struct\
type Command struct\
func NewCommand(rt \*twi.Runtime) \*Command\
Command.Empty string\
Command.OnSelect func(string)\
func (\*Command) Group(heading string, items ...CommandItem) CommandGroup\
func (\*Command) Input(placeholder string) twi.Node\
func (\*Command) Item(value string, children ...twi.NodeOption) CommandItem\
func (\*Command) List(groups ...CommandGroup) twi.Node\
func (\*Command) Node(children ...twi.NodeOption) twi.Node\
func (\*Command) Search(s string)\
func (\*Command) Separator() CommandGroup\
func CommandShortcut(children ...twi.NodeOption) twi.Node\
type CommandDialog struct\
func NewCommandDialog(rt \*twi.Runtime) \*CommandDialog\
CommandDialog embeds \*Dialog\
CommandDialog embeds \*Command\
CommandDialog.Hotkey rune\
func (\*CommandDialog) Node(children ...twi.NodeOption) twi.Node\
func (CommandDialog) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node\
func (\*CommandDialog) Close(v Variant, s Size, children ...twi.NodeOption) twi.Node\
func (\*CommandDialog) Content(children ...twi.NodeOption) twi.Node\
func (\*CommandDialog) Description(children ...twi.NodeOption) twi.Node\
func (\*CommandDialog) Footer(children ...twi.NodeOption) twi.Node\
func (\*CommandDialog) Header(children ...twi.NodeOption) twi.Node\
func (\*CommandDialog) Title(children ...twi.NodeOption) twi.Node\
CommandDialog.Key string\
CommandDialog.Open bool\
CommandDialog.OnOpenChange func(bool)\
CommandDialog.Disabled bool\
CommandDialog.Invalid bool\
CommandDialog.Empty string\
CommandDialog.OnSelect func(string)\
func (\*CommandDialog) Group(heading string, items ...CommandItem) CommandGroup\
func (\*CommandDialog) Input(placeholder string) twi.Node\
func (\*CommandDialog) Item(value string, children ...twi.NodeOption) CommandItem\
func (\*CommandDialog) List(groups ...CommandGroup) twi.Node\
func (\*CommandDialog) Search(s string)\
func (\*CommandDialog) Separator() CommandGroup

## Dialog

type Dialog struct\
func NewAlertDialog(rt \*twi.Runtime) \*Dialog\
func NewDialog(rt \*twi.Runtime) \*Dialog\
func NewDrawer(rt \*twi.Runtime, side Side) \*Dialog\
func NewSheet(rt \*twi.Runtime, side Side) \*Dialog\
func (\*Dialog) Close(v Variant, s Size, children ...twi.NodeOption) twi.Node\
func (\*Dialog) Content(children ...twi.NodeOption) twi.Node\
func (\*Dialog) Description(children ...twi.NodeOption) twi.Node\
func (\*Dialog) Footer(children ...twi.NodeOption) twi.Node\
func (\*Dialog) Header(children ...twi.NodeOption) twi.Node\
func (\*Dialog) Title(children ...twi.NodeOption) twi.Node\
func (\*Dialog) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node\
Dialog.Key string\
Dialog.Open bool\
Dialog.OnOpenChange func(bool)\
Dialog.Disabled bool\
Dialog.Invalid bool

## Empty

func Empty(children ...twi.NodeOption) twi.Node\
func EmptyHeader(children ...twi.NodeOption) twi.Node\
func EmptyMedia(v Variant, children ...twi.NodeOption) twi.Node\
func EmptyTitle(children ...twi.NodeOption) twi.Node\
func EmptyDescription(children ...twi.NodeOption) twi.Node\
func EmptyContent(children ...twi.NodeOption) twi.Node

## Field

func FieldSet(children ...twi.NodeOption) twi.Node\
func FieldLegend(children ...twi.NodeOption) twi.Node\
func FieldGroup(children ...twi.NodeOption) twi.Node\
func Field(o Orientation, children ...twi.NodeOption) twi.Node\
func FieldLabel(children ...twi.NodeOption) twi.Node\
func FieldDescription(children ...twi.NodeOption) twi.Node\
func FieldError(children ...twi.NodeOption) twi.Node\
func FieldSeparator(children ...twi.NodeOption) twi.Node

## Form

type FormValues struct\
FormValues.Text map\[string\]string\
FormValues.Checked map\[string\]bool\
type Form struct\
func NewForm(rt \*twi.Runtime) \*Form\
Form.OnSubmit func(FormValues)\
func (\*Form) Checkbox(key string, c \*Checkbox, rules ...func(bool) string)\
func (\*Form) Control(key string, options ...twi.NodeOption) twi.Node\
func (\*Form) Input(key string, in \*Input, rules ...func(string) string)\
func (\*Form) Item(key string, children ...twi.NodeOption) twi.Node\
func (\*Form) Label(key string, children ...twi.NodeOption) twi.Node\
func (\*Form) Reset()\
func (\*Form) Select(key string, s \*Select, rules ...func(string) string)\
func (\*Form) Submit()\
func (\*Form) Textarea(key string, t \*Textarea, rules ...func(string) string)

## Input

type Input struct\
func NewInput(rt \*twi.Runtime) \*Input\
func (\*Input) Group(addons ...Addon) twi.Node\
func (\*Input) Node(options ...twi.NodeOption) twi.Node\
Input embeds edit.Buffer\
Input.Key string\
Input.Placeholder string\
Input.Submit func(string)\
Input.Mode edit.Mode\
Input.Wrap int\
Input.Widths text.Widths\
Input.Now func() time.Time\
func (\*Input) Apply(k input.KeyEvent) bool\
func (\*Input) At(row, column int) int\
func (\*Input) Cursor() (row, column int)\
func (\*Input) Drag(at int)\
func (\*Input) Expand(s string) string\
func (\*Input) Insert(s string)\
func (\*Input) Paste(s string)\
func (\*Input) Press(at int, u edit.Unit, extend bool)\
func (\*Input) Remember(s string)\
func (\*Input) Rows() \[\]edit.Row\
func (\*Input) Selection() (start, end int)\
func (\*Input) Set(s string)\
func (\*Input) Value() string\
Input.Disabled bool\
Input.Invalid bool\
type Textarea struct\
func NewTextarea(rt \*twi.Runtime) \*Textarea\
func (\*Textarea) Group(addons ...Addon) twi.Node\
func (\*Textarea) Node(options ...twi.NodeOption) twi.Node\
Textarea embeds edit.Buffer\
Textarea.Key string\
Textarea.Placeholder string\
Textarea.Submit func(string)\
Textarea.Mode edit.Mode\
Textarea.Wrap int\
Textarea.Widths text.Widths\
Textarea.Now func() time.Time\
func (\*Textarea) Apply(k input.KeyEvent) bool\
func (\*Textarea) At(row, column int) int\
func (\*Textarea) Cursor() (row, column int)\
func (\*Textarea) Drag(at int)\
func (\*Textarea) Expand(s string) string\
func (\*Textarea) Insert(s string)\
func (\*Textarea) Paste(s string)\
func (\*Textarea) Press(at int, u edit.Unit, extend bool)\
func (\*Textarea) Remember(s string)\
func (\*Textarea) Rows() \[\]edit.Row\
func (\*Textarea) Selection() (start, end int)\
func (\*Textarea) Set(s string)\
func (\*Textarea) Value() string\
Textarea.Disabled bool\
Textarea.Invalid bool\
type Align uint8\
const InlineStart Align\
const InlineEnd Align\
const BlockStart Align\
const BlockEnd Align\
type Addon struct\
func InputGroupAddon(a Align, children ...twi.NodeOption) Addon\
func InputGroupText(children ...twi.NodeOption) twi.Node\
func InputGroupButton(children ...twi.NodeOption) twi.Node

## InputOTP

type InputOTP struct\
func NewInputOTP(rt \*twi.Runtime, length int) \*InputOTP\
InputOTP.Length int\
InputOTP.Value string\
InputOTP.OnChange func(string)\
func (\*InputOTP) Node(options ...twi.NodeOption) twi.Node\
func (\*InputOTP) Slot(i int) twi.Node\
InputOTP.Disabled bool\
InputOTP.Invalid bool\
func InputOTPGroup(children ...twi.NodeOption) twi.Node\
func InputOTPSeparator() twi.Node

## Item

func Item(v Variant, s Size, children ...twi.NodeOption) twi.Node\
func ItemMedia(v Variant, children ...twi.NodeOption) twi.Node\
func ItemContent(children ...twi.NodeOption) twi.Node\
func ItemTitle(children ...twi.NodeOption) twi.Node\
func ItemDescription(children ...twi.NodeOption) twi.Node\
func ItemActions(children ...twi.NodeOption) twi.Node\
func ItemGroup(children ...twi.NodeOption) twi.Node\
func ItemSeparator(children ...twi.NodeOption) twi.Node

## Kbd

func Kbd(children ...twi.NodeOption) twi.Node\
func KbdGroup(children ...twi.NodeOption) twi.Node

## Label

func Label(children ...twi.NodeOption) twi.Node

## Marker

func Marker(v Variant, children ...twi.NodeOption) twi.Node\
func MarkerIcon(children ...twi.NodeOption) twi.Node\
func MarkerContent(children ...twi.NodeOption) twi.Node

## menu.go

type DropdownMenu struct\
func NewDropdownMenu(rt \*twi.Runtime) \*DropdownMenu\
DropdownMenu.OnSelect func(string)\
func (\*DropdownMenu) CheckboxItem(text string, checked \*bool, children ...twi.NodeOption) twi.Node\
func (\*DropdownMenu) Content(children ...twi.NodeOption) twi.Node\
func (\*DropdownMenu) Item(text string, children ...twi.NodeOption) twi.Node\
func (\*DropdownMenu) Node(children ...twi.NodeOption) twi.Node\
func (\*DropdownMenu) RadioItem(text string, value \*string, children ...twi.NodeOption) twi.Node\
func (\*DropdownMenu) Sub() \*DropdownMenuSub\
func (\*DropdownMenu) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node\
DropdownMenu.Side Side\
DropdownMenu.Align Alignment\
DropdownMenu.Key string\
DropdownMenu.Open bool\
DropdownMenu.OnOpenChange func(bool)\
DropdownMenu.Disabled bool\
DropdownMenu.Invalid bool\
type ContextMenu struct\
func NewContextMenu(rt \*twi.Runtime) \*ContextMenu\
ContextMenu embeds DropdownMenu\
func (\*ContextMenu) CheckboxItem(text string, checked \*bool, children ...twi.NodeOption) twi.Node\
func (\*ContextMenu) Content(children ...twi.NodeOption) twi.Node\
func (\*ContextMenu) Item(text string, children ...twi.NodeOption) twi.Node\
func (\*ContextMenu) Node(children ...twi.NodeOption) twi.Node\
func (\*ContextMenu) RadioItem(text string, value \*string, children ...twi.NodeOption) twi.Node\
func (\*ContextMenu) Sub() \*DropdownMenuSub\
func (\*ContextMenu) Trigger(children ...twi.NodeOption) twi.Node\
ContextMenu.OnSelect func(string)\
ContextMenu.Side Side\
ContextMenu.Align Alignment\
ContextMenu.Key string\
ContextMenu.Open bool\
ContextMenu.OnOpenChange func(bool)\
ContextMenu.Disabled bool\
ContextMenu.Invalid bool\
type DropdownMenuSub struct\
DropdownMenuSub.Open bool\
func (\*DropdownMenuSub) CheckboxItem(text string, checked \*bool, children ...twi.NodeOption) twi.Node\
func (\*DropdownMenuSub) Content(children ...twi.NodeOption) twi.Node\
func (\*DropdownMenuSub) Item(text string, children ...twi.NodeOption) twi.Node\
func (\*DropdownMenuSub) Node(children ...twi.NodeOption) twi.Node\
func (\*DropdownMenuSub) RadioItem(text string, value \*string, children ...twi.NodeOption) twi.Node\
func (\*DropdownMenuSub) Sub() \*DropdownMenuSub\
func (\*DropdownMenuSub) Trigger(text string, children ...twi.NodeOption) twi.Node\
func DropdownMenuLabel(children ...twi.NodeOption) twi.Node\
func DropdownMenuSeparator() twi.Node\
func DropdownMenuShortcut(children ...twi.NodeOption) twi.Node

## Menubar

type Menubar struct\
func NewMenubar(rt \*twi.Runtime) \*Menubar\
func (\*Menubar) Menu() \*MenubarMenu\
func (\*Menubar) Node(children ...twi.NodeOption) twi.Node\
Menubar.Disabled bool\
Menubar.Invalid bool\
type MenubarMenu struct\
MenubarMenu embeds DropdownMenu\
func (\*MenubarMenu) CheckboxItem(text string, checked \*bool, children ...twi.NodeOption) twi.Node\
func (\*MenubarMenu) Content(children ...twi.NodeOption) twi.Node\
func (\*MenubarMenu) Item(text string, children ...twi.NodeOption) twi.Node\
func (\*MenubarMenu) RadioItem(text string, value \*string, children ...twi.NodeOption) twi.Node\
func (\*MenubarMenu) Sub() \*DropdownMenuSub\
func (\*MenubarMenu) Trigger(children ...twi.NodeOption) twi.Node\
MenubarMenu.OnSelect func(string)\
func (\*MenubarMenu) Node(children ...twi.NodeOption) twi.Node\
MenubarMenu.Side Side\
MenubarMenu.Align Alignment\
MenubarMenu.Key string\
MenubarMenu.Open bool\
MenubarMenu.OnOpenChange func(bool)\
MenubarMenu.Disabled bool\
MenubarMenu.Invalid bool

## Merge

func Merge(classes ...string) string

## Message

func MessageGroup(children ...twi.NodeOption) twi.Node\
func Message(a Alignment, children ...twi.NodeOption) twi.Node\
func MessageAvatar(children ...twi.NodeOption) twi.Node\
func MessageContent(children ...twi.NodeOption) twi.Node\
func MessageHeader(children ...twi.NodeOption) twi.Node\
func MessageFooter(children ...twi.NodeOption) twi.Node

## NativeSelect

type NativeSelect struct\
func NewNativeSelect(rt \*twi.Runtime) \*NativeSelect\
NativeSelect.Options \[\]string\
NativeSelect.Value string\
NativeSelect.OnChange func(string)\
func (\*NativeSelect) Node(options ...twi.NodeOption) twi.Node\
NativeSelect.Disabled bool\
NativeSelect.Invalid bool

## NavigationMenu

type NavigationMenu struct\
func NewNavigationMenu(rt \*twi.Runtime) \*NavigationMenu\
NavigationMenu.Value string\
NavigationMenu.OnSelect func(string)\
func (\*NavigationMenu) Item(value string) \*NavigationMenuItem\
func (\*NavigationMenu) List(children ...twi.NodeOption) twi.Node\
func (\*NavigationMenu) Node(children ...twi.NodeOption) twi.Node\
NavigationMenu.Disabled bool\
NavigationMenu.Invalid bool\
type NavigationMenuItem struct\
func (\*NavigationMenuItem) Content(children ...twi.NodeOption) twi.Node\
func (\*NavigationMenuItem) Link(href string, children ...twi.NodeOption) twi.Node\
func (\*NavigationMenuItem) Node(children ...twi.NodeOption) twi.Node\
func (\*NavigationMenuItem) Trigger(children ...twi.NodeOption) twi.Node

## overlay.go

type Side uint8\
const Bottom Side\
const Top Side\
const Right Side\
const Left Side\
type Alignment uint8\
const Center Alignment\
const Start Alignment\
const End Alignment\
type Popover struct\
func NewPopover(rt \*twi.Runtime) \*Popover\
func (\*Popover) Content(children ...twi.NodeOption) twi.Node\
func (\*Popover) Node(children ...twi.NodeOption) twi.Node\
func (\*Popover) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node\
Popover.Side Side\
Popover.Align Alignment\
Popover.Key string\
Popover.Open bool\
Popover.OnOpenChange func(bool)\
Popover.Disabled bool\
Popover.Invalid bool\
type Tooltip struct\
func NewTooltip(rt \*twi.Runtime) \*Tooltip\
func (\*Tooltip) Content(children ...twi.NodeOption) twi.Node\
func (\*Tooltip) Node(children ...twi.NodeOption) twi.Node\
func (\*Tooltip) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node\
Tooltip.Side Side\
Tooltip.Align Alignment\
Tooltip.Key string\
Tooltip.Open bool\
Tooltip.OnOpenChange func(bool)\
Tooltip.Disabled bool\
Tooltip.Invalid bool\
type HoverCard struct\
func NewHoverCard(rt \*twi.Runtime) \*HoverCard\
func (\*HoverCard) Content(children ...twi.NodeOption) twi.Node\
func (\*HoverCard) Node(children ...twi.NodeOption) twi.Node\
func (\*HoverCard) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node\
HoverCard.Side Side\
HoverCard.Align Alignment\
HoverCard.Key string\
HoverCard.Open bool\
HoverCard.OnOpenChange func(bool)\
HoverCard.Disabled bool\
HoverCard.Invalid bool

## Pagination

func Pagination(children ...twi.NodeOption) twi.Node\
func PaginationContent(children ...twi.NodeOption) twi.Node\
func PaginationItem(children ...twi.NodeOption) twi.Node\
func PaginationLink(isActive bool, children ...twi.NodeOption) twi.Node\
func PaginationPrevious(children ...twi.NodeOption) twi.Node\
func PaginationNext(children ...twi.NodeOption) twi.Node\
func PaginationEllipsis(children ...twi.NodeOption) twi.Node

## Progress

func Progress(value int, children ...twi.NodeOption) twi.Node

## RadioGroup

type RadioGroup struct\
func NewRadioGroup(rt \*twi.Runtime) \*RadioGroup\
RadioGroup.Value string\
RadioGroup.OnChange func(string)\
func (\*RadioGroup) Item(value string, options ...twi.NodeOption) twi.Node\
func (\*RadioGroup) Node(options ...twi.NodeOption) twi.Node\
RadioGroup.Disabled bool\
RadioGroup.Invalid bool

## Resizable

type Resizable struct\
func NewResizable(rt \*twi.Runtime) \*Resizable\
Resizable.Orientation Orientation\
Resizable.Sizes \[\]int\
Resizable.OnResize func(\[\]int)\
func (\*Resizable) Handle(withHandle bool, children ...twi.NodeOption) twi.Node\
func (\*Resizable) Node(children ...twi.NodeOption) twi.Node\
func (\*Resizable) Panel(children ...twi.NodeOption) twi.Node\
Resizable.Disabled bool\
Resizable.Invalid bool

## scroller.go

type MessageScroller struct\
func NewMessageScroller(rt \*twi.Runtime) \*MessageScroller\
func (\*MessageScroller) Button(children ...twi.NodeOption) twi.Node\
func (\*MessageScroller) Item(children ...twi.NodeOption) twi.Node\
func (\*MessageScroller) Node(children ...twi.NodeOption) twi.Node\
func (\*MessageScroller) Viewport(children ...twi.NodeOption) twi.Node

## Select

type Select struct\
func NewSelect(rt \*twi.Runtime) \*Select\
Select.Value string\
Select.Placeholder string\
Select.OnChange func(string)\
func (\*Select) Content(children ...twi.NodeOption) twi.Node\
func (\*Select) Item(value, label string, children ...twi.NodeOption) twi.Node\
func (\*Select) Node(children ...twi.NodeOption) twi.Node\
func (\*Select) Trigger(children ...twi.NodeOption) twi.Node\
Select.Side Side\
Select.Align Alignment\
Select.Key string\
Select.Open bool\
Select.OnOpenChange func(bool)\
Select.Disabled bool\
Select.Invalid bool\
func SelectLabel(children ...twi.NodeOption) twi.Node\
func SelectSeparator() twi.Node

## Separator

func Separator(o Orientation, children ...twi.NodeOption) twi.Node

## Sidebar

type Sidebar struct\
func NewSidebar(rt \*twi.Runtime) \*Sidebar\
Sidebar.Open bool\
Sidebar.OnOpenChange func(bool)\
Sidebar.Hotkey rune\
func (\*Sidebar) Node(children ...twi.NodeOption) twi.Node\
func (\*Sidebar) Provider(children ...twi.NodeOption) twi.Node\
func (\*Sidebar) Toggle()\
func (\*Sidebar) Trigger(children ...twi.NodeOption) twi.Node\
Sidebar.Disabled bool\
Sidebar.Invalid bool\
func SidebarInset(children ...twi.NodeOption) twi.Node\
func SidebarHeader(children ...twi.NodeOption) twi.Node\
func SidebarFooter(children ...twi.NodeOption) twi.Node\
func SidebarSeparator() twi.Node\
func SidebarContent(children ...twi.NodeOption) twi.Node\
func SidebarGroup(children ...twi.NodeOption) twi.Node\
func SidebarGroupLabel(children ...twi.NodeOption) twi.Node\
func SidebarGroupContent(children ...twi.NodeOption) twi.Node\
func SidebarMenu(children ...twi.NodeOption) twi.Node\
func SidebarMenuItem(children ...twi.NodeOption) twi.Node\
func SidebarMenuButton(s Size, active bool, children ...twi.NodeOption) twi.Node\
func SidebarMenuBadge(children ...twi.NodeOption) twi.Node\
func SidebarMenuSub(children ...twi.NodeOption) twi.Node\
func SidebarMenuSubItem(children ...twi.NodeOption) twi.Node\
func SidebarMenuSubButton(active bool, children ...twi.NodeOption) twi.Node\
func ScrollArea(children ...twi.NodeOption) twi.Node

## Skeleton

func Skeleton(children ...twi.NodeOption) twi.Node

## Slider

type Slider struct\
func NewSlider(rt \*twi.Runtime) \*Slider\
Slider.Value int\
Slider.Min int\
Slider.Max int\
Slider.Step int\
Slider.OnChange func(int)\
func (\*Slider) Node(options ...twi.NodeOption) twi.Node\
Slider.Disabled bool\
Slider.Invalid bool

## Spinner

type Spinner struct\
func NewSpinner(rt \*twi.Runtime) \*Spinner\
func (\*Spinner) Node(options ...twi.NodeOption) twi.Node

## Switch

type Switch struct\
func NewSwitch(rt \*twi.Runtime) \*Switch\
Switch.Checked bool\
Switch.OnChange func(bool)\
func (\*Switch) Node(options ...twi.NodeOption) twi.Node\
Switch.Disabled bool\
Switch.Invalid bool

## Table

func Table(children ...twi.NodeOption) twi.Node\
func TableHeader(children ...twi.NodeOption) twi.Node\
func TableBody(children ...twi.NodeOption) twi.Node\
func TableFooter(children ...twi.NodeOption) twi.Node\
func TableRow(children ...twi.NodeOption) twi.Node\
func TableHead(children ...twi.NodeOption) twi.Node\
func TableCell(children ...twi.NodeOption) twi.Node\
func TableCaption(children ...twi.NodeOption) twi.Node

## Tabs

type Tabs struct\
func NewTabs(rt \*twi.Runtime) \*Tabs\
Tabs.Orientation Orientation\
Tabs.Value string\
Tabs.OnChange func(string)\
func (\*Tabs) Content(value string, children ...twi.NodeOption) twi.Node\
func (\*Tabs) List(children ...twi.NodeOption) twi.Node\
func (\*Tabs) Node(children ...twi.NodeOption) twi.Node\
func (\*Tabs) Trigger(value string, children ...twi.NodeOption) twi.Node\
Tabs.Disabled bool\
Tabs.Invalid bool

## toast.go

type ToastKind uint8\
const ToastDefault ToastKind\
const ToastSuccess ToastKind\
const ToastError ToastKind\
type ToastAction struct\
ToastAction.Label string\
ToastAction.OnClick func()\
type Toaster struct\
func NewToaster(rt \*twi.Runtime) \*Toaster\
Toaster.Duration time.Duration\
func (\*Toaster) Avoid(panels ...\*Dialog)\
func (\*Toaster) Error(title, description string, action ToastAction)\
func (\*Toaster) Node() twi.Node\
func (\*Toaster) Show(title, description string, action ToastAction)\
func (\*Toaster) Success(title, description string, action ToastAction)

## Toggle

type Toggle struct\
func NewToggle(rt \*twi.Runtime) \*Toggle\
Toggle.Variant Variant\
Toggle.Size Size\
Toggle.Pressed bool\
Toggle.OnChange func(bool)\
func (\*Toggle) Node(options ...twi.NodeOption) twi.Node\
Toggle.Disabled bool\
Toggle.Invalid bool\
type ToggleGroup struct\
func NewToggleGroup(rt \*twi.Runtime) \*ToggleGroup\
ToggleGroup.Variant Variant\
ToggleGroup.Size Size\
ToggleGroup.Multiple bool\
ToggleGroup.Value \[\]string\
ToggleGroup.OnChange func(\[\]string)\
func (\*ToggleGroup) Item(value string, options ...twi.NodeOption) twi.Node\
func (\*ToggleGroup) Node(options ...twi.NodeOption) twi.Node\
ToggleGroup.Disabled bool\
ToggleGroup.Invalid bool

## ui.go

type Variant uint8\
const Default Variant\
const Destructive Variant\
const Outline Variant\
const Secondary Variant\
const Ghost Variant\
const Link Variant\
const Icon Variant\
const Muted Variant\
const Image Variant\
const Tinted Variant\
const Ruled Variant\
const Bordered Variant\
type Size uint8\
const SizeDefault Size\
const SizeXS Size\
const SizeSM Size\
const SizeLG Size\
const SizeIcon Size\
type Orientation uint8\
const Horizontal Orientation\
const Vertical Orientation\
type ItemIcon struct\
ItemIcon embeds twi.Node
