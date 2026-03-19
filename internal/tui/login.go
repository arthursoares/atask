package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/atask/atask/internal/client"
)

// LoginResult is returned when the login screen completes successfully.
type LoginResult struct {
	Token string
}

const (
	modeLogin    = 0
	modeRegister = 1
)

const (
	fieldEmail    = 0
	fieldPassword = 1
	fieldName     = 2 // register only
)

// Login is a standalone bubbletea model for the login/register screen.
type Login struct {
	client *client.Client
	mode   int // modeLogin or modeRegister
	focus  int // which field is focused
	width  int
	height int

	email    textinput.Model
	password textinput.Model
	name     textinput.Model // register only

	err    string
	result *LoginResult // set when login/register succeeds
}

// NewLogin creates the login screen model.
func NewLogin(c *client.Client) Login {
	email := textinput.New()
	email.Placeholder = "you@example.com"
	email.Focus()

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword

	name := textinput.New()
	name.Placeholder = "Your Name"

	return Login{
		client:   c,
		mode:     modeLogin,
		focus:    fieldEmail,
		email:    email,
		password: password,
		name:     name,
	}
}

// Result returns the login result if authentication succeeded.
func (l Login) Result() *LoginResult {
	return l.result
}

func (l Login) Init() tea.Cmd {
	return textinput.Blink
}

func (l Login) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.width = msg.Width
		l.height = msg.Height
		return l, nil

	case loginSuccessMsg:
		l.result = &LoginResult{Token: msg.token}
		return l, tea.Quit

	case loginErrorMsg:
		l.err = msg.err.Error()
		return l, nil

	case tea.KeyPressMsg:
		l.err = "" // clear error on any key

		if isEscape(msg) {
			return l, tea.Quit
		}

		if isTab(msg) || isShiftTab(msg) {
			if isTab(msg) {
				l = l.nextField()
			} else {
				l = l.prevField()
			}
			return l, nil
		}

		// Ctrl+R toggles login/register
		if msg.Mod&tea.ModCtrl != 0 && msg.Code == 'r' {
			if l.mode == modeLogin {
				l.mode = modeRegister
			} else {
				l.mode = modeLogin
			}
			l.focus = fieldEmail
			l = l.updateFocus()
			return l, nil
		}

		if isEnter(msg) {
			return l, l.submit()
		}

		// Route to focused input
		return l.updateInput(msg)
	}

	return l, nil
}

func (l Login) View() tea.View {
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Cyan).Render("atask")
	subtitle := DimmedItem.Render("AI-first task manager")

	// Mode toggle
	var modeText string
	if l.mode == modeLogin {
		modeText = ActiveTab.Render("Login") + "  " + InactiveTab.Render("Register")
	} else {
		modeText = InactiveTab.Render("Login") + "  " + ActiveTab.Render("Register")
	}

	// Fields
	emailLabel := "  Email:"
	passLabel := "  Password:"

	// Error
	var errLine string
	if l.err != "" {
		errLine = "\n" + ErrorStyle.Render("  "+l.err)
	}

	// Build form
	b.WriteString(title + "  " + subtitle + "\n\n")
	b.WriteString(modeText + "\n\n")
	b.WriteString(emailLabel + "\n")
	b.WriteString("  " + l.email.View() + "\n\n")
	b.WriteString(passLabel + "\n")
	b.WriteString("  " + l.password.View() + "\n")

	if l.mode == modeRegister {
		b.WriteString("\n  Name:\n")
		b.WriteString("  " + l.name.View() + "\n")
	}

	b.WriteString(errLine + "\n\n")
	b.WriteString(DimmedItem.Render("  [Enter] submit  [Tab] next field  [Ctrl+R] toggle login/register  [Esc] quit"))

	content := OverlayStyle.Width(50).Render(b.String())

	// Center on screen
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)
	padLeft := (l.width - contentWidth) / 2
	padTop := (l.height - contentHeight) / 3
	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	centered := strings.Repeat("\n", padTop) +
		lipgloss.NewStyle().PaddingLeft(padLeft).Render(content)

	return tea.NewView(centered)
}

func (l Login) submit() tea.Cmd {
	email := l.email.Value()
	password := l.password.Value()

	if email == "" || password == "" {
		return func() tea.Msg {
			return loginErrorMsg{err: fmt.Errorf("email and password required")}
		}
	}

	if l.mode == modeRegister {
		name := l.name.Value()
		if name == "" {
			return func() tea.Msg {
				return loginErrorMsg{err: fmt.Errorf("name required for registration")}
			}
		}
		return func() tea.Msg {
			_, err := l.client.Register(context.Background(), email, password, name)
			if err != nil {
				return loginErrorMsg{err: fmt.Errorf("registration failed: %w", err)}
			}
			// After register, login to get token
			token, err := l.client.Login(context.Background(), email, password)
			if err != nil {
				return loginErrorMsg{err: fmt.Errorf("login after register failed: %w", err)}
			}
			return loginSuccessMsg{token: token}
		}
	}

	return func() tea.Msg {
		token, err := l.client.Login(context.Background(), email, password)
		if err != nil {
			return loginErrorMsg{err: fmt.Errorf("login failed: %w", err)}
		}
		return loginSuccessMsg{token: token}
	}
}

func (l Login) nextField() Login {
	maxField := fieldPassword
	if l.mode == modeRegister {
		maxField = fieldName
	}
	l.focus++
	if l.focus > maxField {
		l.focus = fieldEmail
	}
	l = l.updateFocus()
	return l
}

func (l Login) prevField() Login {
	maxField := fieldPassword
	if l.mode == modeRegister {
		maxField = fieldName
	}
	l.focus--
	if l.focus < fieldEmail {
		l.focus = maxField
	}
	l = l.updateFocus()
	return l
}

func (l Login) updateFocus() Login {
	l.email.Blur()
	l.password.Blur()
	l.name.Blur()
	switch l.focus {
	case fieldEmail:
		l.email.Focus()
	case fieldPassword:
		l.password.Focus()
	case fieldName:
		l.name.Focus()
	}
	return l
}

func (l Login) updateInput(msg tea.KeyPressMsg) (Login, tea.Cmd) {
	var cmd tea.Cmd
	switch l.focus {
	case fieldEmail:
		l.email, cmd = l.email.Update(msg)
	case fieldPassword:
		l.password, cmd = l.password.Update(msg)
	case fieldName:
		l.name, cmd = l.name.Update(msg)
	}
	return l, cmd
}

type loginSuccessMsg struct{ token string }
type loginErrorMsg struct{ err error }
