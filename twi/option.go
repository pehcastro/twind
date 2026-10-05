package twi

import (
	"io"
	"time"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/terminal"
	"github.com/pehcastro/twind/twi/theme"
)

type Option interface{ applyRun(*config) }

type RenderOption interface{ applyRender(*config) }

type Setting interface {
	Option
	RenderOption
}

type config struct {
	width      int
	profile    color.Profile
	profileSet bool
	sheet      style.Sheet
	theme      *theme.Theme
	graphics   *terminal.Graphics
	backend    Terminal
	clock      Clock
}

type setting func(*config)

func (s setting) applyRun(c *config) { s(c) }

func (s setting) applyRender(c *config) { s(c) }

type renderSetting func(*config)

func (s renderSetting) applyRender(c *config) { s(c) }

type runSetting func(*config)

func (s runSetting) applyRun(c *config) { s(c) }

func Width(cells int) RenderOption { return renderSetting(func(c *config) { c.width = cells }) }

func ColorProfile(p color.Profile) Setting {
	return setting(func(c *config) { c.profile, c.profileSet = p, true })
}

func Styles(sheet style.Sheet) Setting { return setting(func(c *config) { c.sheet = sheet }) }

func Theme(t theme.Theme) Setting { return setting(func(c *config) { c.theme = &t }) }

func Graphics(mode terminal.Graphics) Setting {
	return setting(func(c *config) { c.graphics = &mode })
}

type Terminal interface {
	io.Writer
	Events() <-chan input.Event
	Size() (width, height int, err error)
	Capabilities() terminal.Capabilities
	Exit() error
}

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

func Backend(t Terminal, c Clock) Option {
	return runSetting(func(cfg *config) { cfg.backend, cfg.clock = t, c })
}
