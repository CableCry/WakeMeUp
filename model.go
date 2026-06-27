package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type uiState int

const (
	stateList uiState = iota
	stateForm
	stateConfirm
)

type statusKind int

const (
	statusNone statusKind = iota
	statusInfo
	statusOK
	statusWarn
	statusErr
)

type statusMsg struct {
	kind statusKind
	text string
}

const statusTimeout = 4 * time.Second

type wolResultMsg struct {
	name string
	err  error
}

type clearStatusMsg struct{ gen int }

// wake animation messages
type (
	wakeTickMsg  struct{ gen int }
	clearAnimMsg struct{ gen int }
)

const (
	wakeFrame = time.Second / 30 // animation frame interval
	wakeStep  = 0.08             // fill added per frame (~0.45s to fill)
	wakeHold  = 900 * time.Millisecond
)

func sendWOLCmd(d Device) tea.Cmd {
	return func() tea.Msg {
		return wolResultMsg{name: d.Name, err: SendMagicPacket(d)}
	}
}

func clearStatusCmd(gen int) tea.Cmd {
	return tea.Tick(statusTimeout, func(time.Time) tea.Msg {
		return clearStatusMsg{gen: gen}
	})
}

func wakeTickCmd(gen int) tea.Cmd {
	return tea.Tick(wakeFrame, func(time.Time) tea.Msg {
		return wakeTickMsg{gen: gen}
	})
}

func clearAnimCmd(gen int) tea.Cmd {
	return tea.Tick(wakeHold, func(time.Time) tea.Msg {
		return clearAnimMsg{gen: gen}
	})
}

type deviceItem struct{ dev Device }

func (i deviceItem) FilterValue() string { return i.dev.Name + " " + i.dev.MAC }

// wakeAnim is the transient "sending…" animation shown beneath a device while
// its magic packet is dispatched. It is held behind a pointer shared by the
// model and the list delegate so per-frame updates are visible to both without
// re-installing the delegate every tick.
type wakeAnim struct {
	active   bool
	name     string  // name of the device being woken
	percent  float64 // 0..1 fill progress
	gotReply bool    // the send result has come back
	failed   bool
}

const wakeBarWidth = 18

type deviceDelegate struct {
	s    styles
	anim *wakeAnim
}

func (d deviceDelegate) Height() int                             { return 2 }
func (d deviceDelegate) Spacing() int                            { return 1 }
func (d deviceDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d deviceDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(deviceItem)
	if !ok {
		return
	}
	selected := index == m.Index()
	waking := d.anim != nil && d.anim.active && d.anim.name == it.dev.Name

	prefix := "  "
	nameStyle, detailStyle := d.s.itemName, d.s.itemDetail
	if selected {
		prefix = d.s.selBar.Render("│ ")
		nameStyle, detailStyle = d.s.selName, d.s.selDetail
	}

	name := nameStyle.Render(it.dev.Name)

	var detail string
	if waking {
		detail = d.wakeBar()
	} else {
		detail = detailStyle.Render(strings.ToUpper(it.dev.MAC) + "  →  " + it.dev.target())
	}

	fmt.Fprintf(w, "%s%s\n%s%s", prefix, name, prefix, detail)
}

// wakeBar renders the in-progress fill plus a short status label.
func (d deviceDelegate) wakeBar() string {
	filled := int(d.anim.percent*wakeBarWidth + 0.5)
	if filled > wakeBarWidth {
		filled = wakeBarWidth
	}
	bar := d.s.barFilled.Render(strings.Repeat("█", filled)) +
		d.s.barEmpty.Render(strings.Repeat("░", wakeBarWidth-filled))

	var label string
	switch {
	case d.anim.percent >= 1 && d.anim.gotReply && d.anim.failed:
		label = d.s.statusErr.Render("  ✕ failed")
	case d.anim.percent >= 1 && d.anim.gotReply:
		label = d.s.statusOK.Render("  ✓ sent")
	default:
		label = d.s.statusInfo.Render("  waking…")
	}
	return bar + label
}

type model struct {
	devices []Device
	state   uiState

	list   list.Model
	styles styles
	isDark bool

	width, height int

	inputs    []textinput.Model
	focus     int
	editIndex int
	formErr   string

	deleteIndex int

	status    statusMsg
	statusGen int

	anim    *wakeAnim
	animGen int

	quitting bool
}

const (
	fieldName = iota
	fieldMAC
	fieldIP
	fieldPort
	fieldCount
)

func newModel(devices []Device) model {
	m := model{
		devices:     devices,
		state:       stateList,
		isDark:      true,
		editIndex:   -1,
		deleteIndex: -1,
	}
	m.styles = newStyles(true)
	m.anim = &wakeAnim{}

	l := list.New(nil, deviceDelegate{s: m.styles, anim: m.anim}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.PaginationStyle = m.styles.pagination
	m.list = l
	m.refreshList()

	placeholders := []string{
		"Living-room PC",
		"AA:BB:CC:DD:EE:FF",
		"192.168.1.255 (optional)",
		"9 (optional)",
	}
	limits := []int{40, 32, 45, 5}

	tis := m.styles.textinputStyles()
	m.inputs = make([]textinput.Model, fieldCount)
	for i := range m.inputs {
		t := textinput.New()
		t.Prompt = ""
		t.Placeholder = placeholders[i]
		t.CharLimit = limits[i]
		t.SetWidth(36)
		t.SetStyles(tis)
		m.inputs[i] = t
	}

	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.RequestBackgroundColor)
}

func (m *model) applyTheme(isDark bool) {
	m.isDark = isDark
	m.styles = newStyles(isDark)
	m.list.SetDelegate(deviceDelegate{s: m.styles, anim: m.anim})
	m.list.Styles.PaginationStyle = m.styles.pagination
	tis := m.styles.textinputStyles()
	for i := range m.inputs {
		m.inputs[i].SetStyles(tis)
	}
}

func (m *model) setSize() {
	const (
		hPad   = 4
		vPad   = 2
		chrome = 6
	)
	w := m.width - hPad
	if w < 10 {
		w = 10
	}
	h := m.height - vPad - chrome
	if h < 3 {
		h = 3
	}
	m.list.SetSize(w, h)
}

func (m *model) refreshList() {
	items := make([]list.Item, len(m.devices))
	for i, d := range m.devices {
		items[i] = deviceItem{d}
	}
	m.list.SetItems(items)
}

func (m *model) setStatus(kind statusKind, text string) tea.Cmd {
	m.status = statusMsg{kind: kind, text: text}
	m.statusGen++
	return clearStatusCmd(m.statusGen)
}

func (m *model) openForm(idx int) tea.Cmd {
	m.editIndex = idx
	m.formErr = ""
	m.focus = 0

	vals := []string{"", "", "", ""}
	if idx >= 0 && idx < len(m.devices) {
		d := m.devices[idx]
		vals[fieldName] = d.Name
		vals[fieldMAC] = d.MAC
		vals[fieldIP] = d.IP
		if d.Port != 0 {
			vals[fieldPort] = strconv.Itoa(d.Port)
		}
	}
	for i := range m.inputs {
		m.inputs[i].SetValue(vals[i])
		m.inputs[i].SetCursor(len(vals[i]))
		m.inputs[i].Blur()
	}
	m.state = stateForm
	return m.inputs[0].Focus()
}

func (m *model) focusField(i int) tea.Cmd {
	n := len(m.inputs)
	i = (i%n + n) % n
	m.focus = i
	var cmds []tea.Cmd
	for j := range m.inputs {
		if j == i {
			cmds = append(cmds, m.inputs[j].Focus())
		} else {
			m.inputs[j].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.applyTheme(msg.IsDark())
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.setSize()
		return m, nil

	case clearStatusMsg:
		if msg.gen == m.statusGen {
			m.status = statusMsg{}
		}
		return m, nil

	case wolResultMsg:
		if m.anim.active && m.anim.name == msg.name {
			m.anim.gotReply = true
			m.anim.failed = msg.err != nil
		}
		if msg.err != nil {
			// The bar shows "✕ failed"; the status line carries the detail.
			return m, m.setStatus(statusErr, fmt.Sprintf("Couldn't wake %s: %v", msg.name, msg.err))
		}
		return m, nil

	case wakeTickMsg:
		if msg.gen != m.animGen || !m.anim.active {
			return m, nil
		}
		if m.anim.percent < 1 {
			m.anim.percent += wakeStep
			if m.anim.percent > 1 {
				m.anim.percent = 1
			}
			return m, wakeTickCmd(m.animGen)
		}
		// Fill complete: hold the result briefly, then clear.
		return m, clearAnimCmd(m.animGen)

	case clearAnimMsg:
		if msg.gen == m.animGen {
			m.anim.active = false
		}
		return m, nil
	}

	switch m.state {
	case stateForm:
		return m.updateForm(msg)
	case stateConfirm:
		return m.updateConfirm(msg)
	default:
		return m.updateList(msg)
	}
}

func (m model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	if m.list.SettingFilter() {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	switch key.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "a":
		return m, m.openForm(-1)

	case "e":
		if it, ok := m.list.SelectedItem().(deviceItem); ok {
			if idx := findDevice(m.devices, it.dev.Name); idx >= 0 {
				return m, m.openForm(idx)
			}
		}
		return m, nil

	case "d", "x", "delete":
		if it, ok := m.list.SelectedItem().(deviceItem); ok {
			if idx := findDevice(m.devices, it.dev.Name); idx >= 0 {
				m.deleteIndex = idx
				m.state = stateConfirm
			}
		}
		return m, nil

	case "enter":
		if it, ok := m.list.SelectedItem().(deviceItem); ok {
			m.status = statusMsg{} // let the inline bar carry the feedback
			m.animGen++
			*m.anim = wakeAnim{active: true, name: it.dev.Name}
			return m, tea.Batch(sendWOLCmd(it.dev), wakeTickCmd(m.animGen))
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			m.state = stateList
			return m, nil
		case "tab", "down":
			return m, m.focusField(m.focus + 1)
		case "shift+tab", "up":
			return m, m.focusField(m.focus - 1)
		case "enter":
			return m.submitForm()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m model) submitForm() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	macRaw := m.inputs[fieldMAC].Value()
	ipRaw := strings.TrimSpace(m.inputs[fieldIP].Value())
	portRaw := strings.TrimSpace(m.inputs[fieldPort].Value())

	if name == "" {
		m.formErr = "Name is required."
		return m, m.focusField(fieldName)
	}
	mac, err := normalizeMAC(macRaw)
	if err != nil {
		m.formErr = err.Error()
		return m, m.focusField(fieldMAC)
	}
	if ipRaw != "" && net.ParseIP(ipRaw) == nil {
		m.formErr = "Target IP is not a valid address."
		return m, m.focusField(fieldIP)
	}
	port := 0
	if portRaw != "" {
		p, err := strconv.Atoi(portRaw)
		if err != nil || p < 1 || p > 65535 {
			m.formErr = "Port must be a number between 1 and 65535."
			return m, m.focusField(fieldPort)
		}
		port = p
	}
	if existing := findDevice(m.devices, name); existing != -1 && existing != m.editIndex {
		m.formErr = fmt.Sprintf("A device named %q already exists.", name)
		return m, m.focusField(fieldName)
	}

	dev := Device{Name: name, MAC: mac, IP: ipRaw, Port: port}

	next := make([]Device, len(m.devices))
	copy(next, m.devices)
	verb := "Added"
	selectIdx := -1
	if m.editIndex >= 0 && m.editIndex < len(next) {
		next[m.editIndex] = dev
		verb = "Updated"
		selectIdx = m.editIndex
	} else {
		next = append(next, dev)
		selectIdx = len(next) - 1
	}

	if err := SaveDevices(next); err != nil {
		m.formErr = "Could not save: " + err.Error()
		return m, nil
	}

	m.devices = next
	m.refreshList()
	if selectIdx >= 0 {
		m.list.Select(selectIdx)
	}
	m.state = stateList
	return m, m.setStatus(statusOK, fmt.Sprintf("%s %s.", verb, dev.Name))
}

func (m model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "y", "Y", "enter":
		return m.deleteSelected()
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	default:
		m.state = stateList
		return m, nil
	}
}

func (m model) deleteSelected() (tea.Model, tea.Cmd) {
	idx := m.deleteIndex
	if idx < 0 || idx >= len(m.devices) {
		m.state = stateList
		return m, nil
	}
	name := m.devices[idx].Name

	next := make([]Device, 0, len(m.devices)-1)
	next = append(next, m.devices[:idx]...)
	next = append(next, m.devices[idx+1:]...)

	if err := SaveDevices(next); err != nil {
		m.state = stateList
		return m, m.setStatus(statusErr, "Could not save: "+err.Error())
	}

	m.devices = next
	m.refreshList()
	m.state = stateList
	m.deleteIndex = -1
	return m, m.setStatus(statusOK, fmt.Sprintf("Removed %s.", name))
}

func (m model) View() tea.View {
	var body string
	switch m.state {
	case stateForm:
		body = m.formView()
	case stateConfirm:
		body = m.confirmView()
	default:
		body = m.listView()
	}

	v := tea.NewView(m.styles.app.Render(body))
	v.AltScreen = true
	v.WindowTitle = "WakeMeUp"
	return v
}

func (m model) header() string {
	n := len(m.devices)
	noun := "devices"
	if n == 1 {
		noun = "device"
	}
	title := m.styles.title.Render("WakeMeUp")
	sub := m.styles.subtitle.Render(fmt.Sprintf("%d %s", n, noun))
	return title + "  " + sub
}

func (m model) listView() string {
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n\n")
	if len(m.devices) == 0 {
		b.WriteString(m.emptyView())
	} else {
		b.WriteString(m.list.View())
	}
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	b.WriteString("\n")
	b.WriteString(m.listHelp())
	return b.String()
}

func (m model) emptyView() string {
	line1 := m.styles.empty.Render("  No devices saved yet.")
	line2 := m.styles.empty.Render("  Press ") +
		m.styles.helpKey.Render("a") +
		m.styles.empty.Render(" to add your first one.")
	return line1 + "\n\n" + line2
}

func (m model) statusLine() string {
	switch m.status.kind {
	case statusInfo:
		return m.styles.statusInfo.Render("• " + m.status.text)
	case statusOK:
		return m.styles.statusOK.Render("✓ " + m.status.text)
	case statusWarn:
		return m.styles.statusWarn.Render("! " + m.status.text)
	case statusErr:
		return m.styles.statusErr.Render("✕ " + m.status.text)
	default:
		return ""
	}
}

func (m model) hint(key, label string) string {
	return m.styles.helpKey.Render(key) + " " + m.styles.help.Render(label)
}

func (m model) listHelp() string {
	if m.list.SettingFilter() {
		return m.styles.help.Render("type to filter   ") +
			m.hint("enter", "apply") + m.styles.helpSep.Render("   ") +
			m.hint("esc", "clear")
	}

	sep := m.styles.helpSep.Render("   ")
	hints := []string{
		m.hint("↑/↓", "move"),
		m.hint("enter", "wake"),
		m.hint("a", "add"),
		m.hint("e", "edit"),
		m.hint("d", "remove"),
	}
	if len(m.devices) > 0 {
		hints = append(hints, m.hint("/", "filter"))
	}
	hints = append(hints, m.hint("q", "quit"))
	return strings.Join(hints, sep)
}

func (m model) formView() string {
	titleText := "Add device"
	if m.editIndex >= 0 {
		titleText = "Edit device"
	}

	labels := []string{"Name", "MAC", "Target IP", "Port"}
	rows := make([]string, len(m.inputs))
	for i := range m.inputs {
		rows[i] = m.formRow(i, labels[i])
	}

	parts := []string{
		m.styles.title.Render(titleText),
		"",
		strings.Join(rows, "\n"),
	}
	if m.formErr != "" {
		parts = append(parts, "", m.styles.statusErr.Render("✕ "+m.formErr))
	}
	parts = append(parts, "", m.formHelp())

	box := m.styles.formBox.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))

	hintLine := m.styles.subtitle.Render("Target IP and Port are optional — they default to the subnet broadcast on port 9.")
	return box + "\n\n" + hintLine
}

func (m model) formRow(i int, label string) string {
	indicator := "  "
	if i == m.focus {
		indicator = m.styles.title.Render("› ")
	}
	lbl := m.styles.formLabel.Render(fmt.Sprintf("%-11s", label))
	return indicator + lbl + m.inputs[i].View()
}

func (m model) formHelp() string {
	sep := m.styles.helpSep.Render("   ")
	return strings.Join([]string{
		m.hint("tab", "next field"),
		m.hint("enter", "save"),
		m.hint("esc", "cancel"),
	}, sep)
}

func (m model) confirmView() string {
	name := "this device"
	if m.deleteIndex >= 0 && m.deleteIndex < len(m.devices) {
		name = m.devices[m.deleteIndex].Name
	}

	parts := []string{
		m.styles.statusErr.Render("Remove " + name + "?"),
		"",
		m.styles.subtitle.Render("It will be deleted from your saved devices."),
		"",
		strings.Join([]string{
			m.hint("y", "remove"),
			m.hint("n", "cancel"),
		}, m.styles.helpSep.Render("   ")),
	}
	return m.styles.confirmBox.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}
