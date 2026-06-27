package main

import (
	"image/color"

	"charm.land/bubbles/v2/textinput"
	lipgloss "charm.land/lipgloss/v2"
)

type palette struct {
	accent  color.Color
	fg      color.Color
	subtle  color.Color
	success color.Color
	warning color.Color
	danger  color.Color
	border  color.Color
}

func newPalette(isDark bool) palette {
	ld := lipgloss.LightDark(isDark)
	return palette{
		accent:  ld(lipgloss.Color("#5A4FCF"), lipgloss.Color("#A89BFF")),
		fg:      ld(lipgloss.Color("#1C1C28"), lipgloss.Color("#E6E6F0")),
		subtle:  ld(lipgloss.Color("#7A7A8C"), lipgloss.Color("#8C8CA0")),
		success: ld(lipgloss.Color("#1F8A4C"), lipgloss.Color("#5BD68A")),
		warning: ld(lipgloss.Color("#9A6B00"), lipgloss.Color("#E6C44D")),
		danger:  ld(lipgloss.Color("#C0392B"), lipgloss.Color("#FF7A7A")),
		border:  ld(lipgloss.Color("#D5D5DE"), lipgloss.Color("#3A3A50")),
	}
}

type styles struct {
	p palette

	app      lipgloss.Style
	title    lipgloss.Style
	subtitle lipgloss.Style

	itemName   lipgloss.Style
	itemDetail lipgloss.Style
	selName    lipgloss.Style
	selDetail  lipgloss.Style
	selBar     lipgloss.Style

	barFilled lipgloss.Style
	barEmpty  lipgloss.Style

	pagination lipgloss.Style

	formBox   lipgloss.Style
	formLabel lipgloss.Style

	confirmBox lipgloss.Style

	statusInfo lipgloss.Style
	statusOK   lipgloss.Style
	statusWarn lipgloss.Style
	statusErr  lipgloss.Style

	help    lipgloss.Style
	helpKey lipgloss.Style
	helpSep lipgloss.Style
	empty   lipgloss.Style
}

func newStyles(isDark bool) styles {
	p := newPalette(isDark)
	base := lipgloss.NewStyle().Foreground(p.fg)

	return styles{
		p: p,

		app:      lipgloss.NewStyle().Padding(1, 2),
		title:    lipgloss.NewStyle().Foreground(p.accent).Bold(true),
		subtitle: lipgloss.NewStyle().Foreground(p.subtle),

		itemName:   base,
		itemDetail: lipgloss.NewStyle().Foreground(p.subtle),
		selName:    lipgloss.NewStyle().Foreground(p.accent).Bold(true),
		selDetail:  lipgloss.NewStyle().Foreground(p.subtle),
		selBar:     lipgloss.NewStyle().Foreground(p.accent),

		barFilled: lipgloss.NewStyle().Foreground(p.accent),
		barEmpty:  lipgloss.NewStyle().Foreground(p.border),

		pagination: lipgloss.NewStyle().Foreground(p.subtle).PaddingLeft(2),

		formBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.border).
			Padding(1, 2),
		formLabel: lipgloss.NewStyle().Foreground(p.subtle),

		confirmBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.danger).
			Padding(1, 2),

		statusInfo: lipgloss.NewStyle().Foreground(p.subtle),
		statusOK:   lipgloss.NewStyle().Foreground(p.success),
		statusWarn: lipgloss.NewStyle().Foreground(p.warning),
		statusErr:  lipgloss.NewStyle().Foreground(p.danger),

		help:    lipgloss.NewStyle().Foreground(p.subtle),
		helpKey: lipgloss.NewStyle().Foreground(p.fg),
		helpSep: lipgloss.NewStyle().Foreground(p.border),
		empty:   lipgloss.NewStyle().Foreground(p.subtle),
	}
}

func (s styles) textinputStyles() textinput.Styles {
	st := textinput.DefaultStyles(true)
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(s.p.accent)
	st.Focused.Text = lipgloss.NewStyle().Foreground(s.p.fg)
	st.Focused.Placeholder = lipgloss.NewStyle().Foreground(s.p.subtle)
	st.Blurred.Prompt = lipgloss.NewStyle().Foreground(s.p.subtle)
	st.Blurred.Text = lipgloss.NewStyle().Foreground(s.p.subtle)
	st.Blurred.Placeholder = lipgloss.NewStyle().Foreground(s.p.subtle)
	st.Cursor.Color = s.p.accent
	return st
}
