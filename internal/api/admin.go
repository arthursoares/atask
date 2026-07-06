package api

import (
	"context"
	"embed"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/store"
)

//go:embed admin_templates/*.html
var adminFS embed.FS

const (
	// adminSessionCookie is the name of the server-side-session cookie the admin
	// UI issues at login and reads on every protected request.
	adminSessionCookie = "admin_session"
	// adminCookiePath scopes the session cookie to the admin UI only. The browser
	// never sends it to the JSON API routes.
	adminCookiePath = "/admin/"
	// adminSessionMaxAge is the cookie lifetime in seconds (8h).
	adminSessionMaxAge = 3600 * 8
)

// AdminHandler renders the server-side admin UI (Go html/template) and owns the
// admin authentication surface: password login, a rotating server-side session,
// and single-use CSRF tokens on every mutation form.
//
// html/template (NOT text/template) is used deliberately: it context-escapes
// interpolated values so a user-controlled name/email cannot inject markup.
type AdminHandler struct {
	auth      auth.AuthProvider
	db        *store.DB
	templates *template.Template
	csrf      *csrfStore
	sessions  *sessionStore
	// secure controls the Secure cookie flag. It must be false for local
	// http:// testing (a Secure cookie is never sent over plain HTTP, which
	// would break login), and true in production behind https://. RegisterRoutes
	// derives it from the configured BaseURL scheme.
	secure bool
}

// NewAdminHandler parses the embedded templates and wires the shared CSRF and
// session stores. The same store instances must be shared with requireAdmin and
// requireCSRF (see RegisterRoutes) so tokens/sessions issued here validate there.
// db backs the Task 22 orphan-data check surfaced on the dashboard.
func NewAdminHandler(authProvider auth.AuthProvider, db *store.DB, csrf *csrfStore, sessions *sessionStore, secure bool) *AdminHandler {
	tmpl := template.Must(template.ParseFS(adminFS, "admin_templates/*.html"))
	return &AdminHandler{
		auth:      authProvider,
		db:        db,
		templates: tmpl,
		csrf:      csrf,
		sessions:  sessions,
		secure:    secure,
	}
}

// render executes a named template and logs (rather than silently ignores) any
// execution error. A mid-write template error cannot un-write already-flushed
// bytes, but it must at least be observable.
func (h *AdminHandler) render(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("admin template render failed", "template", name, "err", err)
	}
}

// sessionCSRF issues a fresh single-use CSRF token for the request's session, or
// "" when there is no session cookie (a page rendered pre-login). Every GET that
// renders a mutation form calls this and injects the token into the form.
func (h *AdminHandler) sessionCSRF(r *http.Request) string {
	c, err := r.Cookie(adminSessionCookie)
	if err != nil {
		return ""
	}
	return h.csrf.issue(c.Value)
}

// setSessionCookie writes the admin session cookie with the security flags from
// spec §5.2: HttpOnly, SameSite=Strict, Path=/admin/, 8h, and Secure gated on
// the deployment scheme.
func (h *AdminHandler) setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    value,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
		Path:     adminCookiePath,
		MaxAge:   maxAge,
	})
}

// LoginPage renders the public login form (GET /admin/login).
func (h *AdminHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "login.html", map[string]any{})
}

// LoginSubmit authenticates admin credentials and, on success, rotates to a
// fresh server-side session (POST /admin/login).
func (h *AdminHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")

	_, user, err := h.auth.AuthWithPassword(email, password)
	if err != nil || user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		h.render(w, "login.html", map[string]any{"Error": "invalid credentials"})
		return
	}
	// Only admins may hold an admin session. requireAdmin re-checks the role on
	// every request (defense in depth), but rejecting here avoids minting a
	// session a non-admin could never use.
	if user.Role != "admin" {
		w.WriteHeader(http.StatusForbidden)
		h.render(w, "login.html", map[string]any{"Error": "not an admin account"})
		return
	}

	// SECURITY: rotate the session ID at the auth boundary. If the
	// unauthenticated browser already presented an admin_session cookie, discard
	// its server-side session and CSRF tokens and mint a brand-new session tied
	// to the now-authenticated identity. Defeats session-fixation.
	if old, oerr := r.Cookie(adminSessionCookie); oerr == nil {
		h.csrf.clear(old.Value)
		h.sessions.Delete(old.Value)
	}

	sessionID := generateSessionID()
	h.sessions.Set(sessionID, user.ID)
	h.setSessionCookie(w, sessionID, adminSessionMaxAge)
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

// Logout clears the session's CSRF tokens and server-side session, expires the
// cookie, and returns to the login page (GET /admin/logout).
func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(adminSessionCookie); err == nil {
		h.csrf.clear(c.Value)
		h.sessions.Delete(c.Value)
	}
	h.setSessionCookie(w, "", -1)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// Dashboard shows the user count, a user table, and (Task 22) an orphaned-data
// warning banner (GET /admin/).
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	users, total, err := h.auth.ListUsers("", 1, 100)
	if err != nil {
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}

	// Surface pre-multi-user orphaned rows (user_id = '') so an admin who
	// upgraded a single-user deployment sees why data looks "missing" instead
	// of just an empty task list. A failure here must not break the dashboard
	// — it only degrades to omitting the banner.
	var orphanTotal int
	if h.db != nil {
		counts, oerr := store.OrphanCounts(r.Context(), h.db.DB)
		if oerr != nil {
			slog.Error("orphan check failed", "err", oerr)
		} else {
			orphanTotal = store.OrphanTotal(counts)
		}
	}

	h.render(w, "dashboard.html", map[string]any{
		"UserCount":   total,
		"Users":       users,
		"OrphanTotal": orphanTotal,
	})
}

// ListUsers renders the full user list (GET /admin/users).
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, total, err := h.auth.ListUsers("", 1, 50)
	if err != nil {
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}
	h.render(w, "users.html", map[string]any{
		"Users": users,
		"Total": total,
	})
}

// CreateUser renders the new-user form (GET) and creates a user (POST) at
// /admin/users/new. The CSRF token was already verified+consumed by
// requireCSRF, so any re-rendered form on a business-logic error MUST mint a
// fresh token (renderUserForm does).
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.renderUserForm(w, r, "")
		return
	}
	// POST — form already parsed and CSRF-verified by requireCSRF.
	_, err := h.auth.CreateUser(
		r.FormValue("email"),
		r.FormValue("password"),
		r.FormValue("name"),
		r.FormValue("role"),
	)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderUserForm(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// renderUserForm renders user_form.html with a FRESH single-use CSRF token. It
// is called both for the initial GET and for the failure-path re-render, so the
// user's next submit always carries an unconsumed token.
func (h *AdminHandler) renderUserForm(w http.ResponseWriter, r *http.Request, errMsg string) {
	data := map[string]any{"CSRFToken": h.sessionCSRF(r)}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, "user_form.html", data)
}

// EditUser renders the edit form for a single user (GET) and applies updates
// (POST) at /admin/users/{id}.
func (h *AdminHandler) EditUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if r.Method == http.MethodGet {
		user, err := h.auth.FindUserByID(id)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		h.render(w, "user_edit.html", map[string]any{
			"CSRFToken": h.sessionCSRF(r),
			"User":      user,
		})
		return
	}

	// POST — form already parsed and CSRF-verified by requireCSRF.
	updates := map[string]any{
		"name": r.FormValue("name"),
		"role": r.FormValue("role"),
	}
	if err := h.auth.UpdateUser(id, updates); err != nil {
		h.renderEditError(w, r, id, err.Error())
		return
	}

	// A "disabled" checkbox toggles account access via the dedicated
	// enable/disable methods (which the AuthProvider models explicitly).
	if r.FormValue("disabled") == "on" {
		if err := h.auth.DisableUser(id); err != nil {
			h.renderEditError(w, r, id, err.Error())
			return
		}
	} else {
		if err := h.auth.EnableUser(id); err != nil {
			h.renderEditError(w, r, id, err.Error())
			return
		}
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// renderEditError re-renders the edit form with a fresh CSRF token and an error.
func (h *AdminHandler) renderEditError(w http.ResponseWriter, r *http.Request, id, errMsg string) {
	user, err := h.auth.FindUserByID(id)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	h.render(w, "user_edit.html", map[string]any{
		"CSRFToken": h.sessionCSRF(r),
		"User":      user,
		"Error":     errMsg,
	})
}

// requireAdmin gates the protected admin routes. It resolves the session cookie
// to a user ID via the server-side session store, loads the user, and requires
// Role=="admin" and not disabled. No/unknown session → redirect to the login
// page; an authenticated-but-non-admin (or disabled) user → 403.
func requireAdmin(authProvider auth.AuthProvider, sessions *sessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(adminSessionCookie)
			if err != nil {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			userID, ok := sessions.Get(c.Value)
			if !ok {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			user, ferr := authProvider.FindUserByID(userID)
			if ferr != nil {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			if user.Role != "admin" || user.Disabled {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
