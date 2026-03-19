package tui

import tea "charm.land/bubbletea/v2"

// isRune reports whether the key message corresponds to the given rune.
// In bubbletea v2, printable character keys have their rune in Key.Code.
func isRune(msg tea.KeyPressMsg, r rune) bool {
	return msg.Code == r
}

// isKey reports whether the key message matches the given special key code.
// Use this for non-printable keys (Enter, Escape, Tab, arrow keys, etc.).
func isKey(msg tea.KeyPressMsg, code rune) bool {
	return msg.Code == code
}

// isCtrl reports whether the key message is Ctrl+r for the given rune.
func isCtrl(msg tea.KeyPressMsg, r rune) bool {
	return msg.Code == r && msg.Mod&tea.ModCtrl != 0
}

// isEnter reports whether the key message is the Enter key.
func isEnter(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyEnter
}

// isEscape reports whether the key message is the Escape key.
func isEscape(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyEscape
}

// isTab reports whether the key message is the Tab key.
func isTab(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyTab
}

// isShiftTab reports whether the key message is Shift+Tab.
func isShiftTab(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyTab && msg.Mod&tea.ModShift != 0
}

// isBackspace reports whether the key message is the Backspace key.
func isBackspace(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyBackspace
}

// isUp reports whether the key message is the Up arrow key.
func isUp(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyUp
}

// isDown reports whether the key message is the Down arrow key.
func isDown(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyDown
}

// isLeft reports whether the key message is the Left arrow key.
func isLeft(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyLeft
}

// isRight reports whether the key message is the Right arrow key.
func isRight(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyRight
}

// isDelete reports whether the key message is the Delete key.
func isDelete(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyDelete
}

// isQuit reports whether the key message is q or Ctrl+c.
func isQuit(msg tea.KeyPressMsg) bool {
	return msg.Code == 'q' || (msg.Code == 'c' && msg.Mod&tea.ModCtrl != 0)
}
