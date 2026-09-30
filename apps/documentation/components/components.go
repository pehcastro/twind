package components

import (
	"embed"
	"io/fs"

	"github.com/twind-dev/twind/twi"
)

//go:embed *.go
var source embed.FS

type Catalog struct {
	Source fs.FS
	Demos  map[string]func(rt *twi.Runtime) func() twi.Node
}

func All() Catalog {
	return Catalog{source, map[string]func(rt *twi.Runtime) func() twi.Node{
		"accordion-demo":       AccordionDemo,
		"alert-demo":           AlertDemo,
		"alert-dialog-demo":    AlertDialogDemo,
		"aspect-ratio-demo":    AspectRatioDemo,
		"avatar-demo":          AvatarDemo,
		"badge-demo":           BadgeDemo,
		"breadcrumb-demo":      BreadcrumbDemo,
		"button-demo":          ButtonDemo,
		"button-variants":      ButtonVariants,
		"button-sizes":         ButtonSizes,
		"button-group-demo":    ButtonGroupDemo,
		"calendar-demo":        CalendarDemo,
		"card-demo":            CardDemo,
		"checkbox-demo":        CheckboxDemo,
		"collapsible-demo":     CollapsibleDemo,
		"combobox-demo":        ComboboxDemo,
		"command-demo":         CommandDemo,
		"command-dialog-demo":  CommandDialogDemo,
		"context-menu-demo":    ContextMenuDemo,
		"dialog-demo":          DialogDemo,
		"drawer-demo":          DrawerDemo,
		"dropdown-menu-demo":   DropdownMenuDemo,
		"empty-demo":           EmptyDemo,
		"field-demo":           FieldDemo,
		"hover-card-demo":      HoverCardDemo,
		"input-demo":           InputDemo,
		"input-group-demo":     InputGroupDemo,
		"input-otp-demo":       InputOTPDemo,
		"item-demo":            ItemDemo,
		"kbd-demo":             KbdDemo,
		"label-demo":           LabelDemo,
		"menubar-demo":         MenubarDemo,
		"native-select-demo":   NativeSelectDemo,
		"navigation-menu-demo": NavigationMenuDemo,
		"pagination-demo":      PaginationDemo,
		"popover-demo":         PopoverDemo,
		"progress-demo":        ProgressDemo,
		"radio-group-demo":     RadioGroupDemo,
		"scroll-area-demo":     ScrollAreaDemo,
		"select-demo":          SelectDemo,
		"separator-demo":       SeparatorDemo,
		"sheet-demo":           SheetDemo,
		"sidebar-demo":         SidebarDemo,
		"skeleton-demo":        SkeletonDemo,
		"slider-demo":          SliderDemo,
		"spinner-demo":         SpinnerDemo,
		"switch-demo":          SwitchDemo,
		"table-demo":           TableDemo,
		"tabs-demo":            TabsDemo,
		"textarea-demo":        TextareaDemo,
		"toaster-demo":         ToasterDemo,
		"toggle-demo":          ToggleDemo,
		"toggle-group-demo":    ToggleGroupDemo,
		"tooltip-demo":         TooltipDemo,
		"layout-flex":          LayoutFlex,
		"layout-grid":          LayoutGrid,
		"layout-position":      LayoutPosition,
		"text-wrap":            TextWrap,
		"motion-transition":    MotionTransition,
		"motion-presence":      MotionPresence,
		"events-demo":          EventsDemo,
		"theming-tokens":       ThemingTokens,
	}}
}
