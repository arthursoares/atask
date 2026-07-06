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
	// adminRecentEventsLimit caps the recent-activity tables (dashboard and
	// per-user overview) at a fixed, positive value. RecentEvents/
	// RecentEventsByUser do not clamp their limit argument themselves (a
	// negative value means "unlimited" to the underlying SQLite query), so
	// this handler must always pass a positive constant.
	adminRecentEventsLimit = 20
	// adminGrowthDays is the window (in days) for the dashboard's
	// creation-growth chart.
	adminGrowthDays = 30
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

// dashboardUserRow merges one auth.User with its per-account entity counts
// and activity summary for the dashboard's user table. Built here in Go
// (rather than looked up by ID via template map-indexing) per the design:
// the handler already builds a users list from ListUsers, so it attaches
// each user's counts + last-active by user.ID directly onto the row.
//
// Counts/Activity default to their zero value when the underlying stats
// query fails or a user has no data — indistinguishable from "0 items" /
// "never active", which is the same soft-degrade discipline the existing
// OrphanTotal block already uses.
type dashboardUserRow struct {
	User       *auth.User
	Counts     store.EntityCounts
	Activity   store.Activity
	TotalCount int
}

// growthBarChartWidth/Height/BarWidth/BarGap size the inline-SVG
// creation-growth chart on the dashboard. Bar coordinates are precomputed in
// Go (buildGrowthBars) rather than computed with template arithmetic, since
// html/template has no built-in math helpers.
const (
	growthBarChartHeight = 60
	growthBarWidth       = 8
	growthBarGap         = 2
)

// growthBar is one day's pre-scaled bar for the creation-growth SVG panel.
type growthBar struct {
	Date   string
	Count  int
	X      int
	Y      int
	Height int
}

// buildGrowthBars scales CreationGrowth's zero-filled day buckets into SVG
// rect coordinates. Guards the divide-by-zero case where every bucket is
// empty (max == 0): every bar then renders at zero height instead of
// panicking or producing NaN/Inf.
func buildGrowthBars(buckets []store.DayBucket) []growthBar {
	max := 0
	for _, b := range buckets {
		if b.Count > max {
			max = b.Count
		}
	}

	bars := make([]growthBar, len(buckets))
	for i, b := range buckets {
		h := 0
		if max > 0 {
			h = b.Count * growthBarChartHeight / max
			if h == 0 && b.Count > 0 {
				h = 2 // keep a visible sliver for a nonzero day
			}
		}
		bars[i] = growthBar{
			Date:   b.Date,
			Count:  b.Count,
			X:      i * (growthBarWidth + growthBarGap),
			Y:      growthBarChartHeight - h,
			Height: h,
		}
	}
	return bars
}

// Dashboard shows the user count, system/per-user statistics, a
// creation-growth chart, a recent-activity feed, a user table, and (Task 22)
// an orphaned-data warning banner (GET /admin/).
//
// Every statistics query is independently non-fatal: on error it logs via
// slog and the dashboard simply omits that one panel (or, for the merged
// per-user row fields, leaves them at their zero value) — mirroring the
// existing orphan-count discipline immediately below. A stats failure must
// never turn into a 500 or a blank dashboard.
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
		counts, oerr := store.OrphanCounts(ctx, h.db.DB)
		if oerr != nil {
			slog.Error("orphan check failed", "err", oerr)
		} else {
			orphanTotal = store.OrphanTotal(counts)
		}
	}

	data := map[string]any{
		"UserCount":   total,
		"OrphanTotal": orphanTotal,
	}

	var perUserCounts map[string]store.EntityCounts
	var userActivity map[string]store.Activity
	if h.db != nil {
		if pc, pcErr := store.PerUserCounts(ctx, h.db.DB); pcErr != nil {
			slog.Error("per-user counts failed", "err", pcErr)
		} else {
			perUserCounts = pc
		}

		if ua, uaErr := store.UserActivity(ctx, h.db.DB); uaErr != nil {
			slog.Error("user activity failed", "err", uaErr)
		} else {
			userActivity = ua
		}
	}

	rows := make([]dashboardUserRow, 0, len(users))
	for _, u := range users {
		row := dashboardUserRow{User: u}
		row.Counts = perUserCounts[u.ID]  // zero value if missing/failed
		row.Activity = userActivity[u.ID] // zero value if missing/failed
		row.TotalCount = row.Counts.Tasks + row.Counts.Projects + row.Counts.Areas + row.Counts.Tags
		rows = append(rows, row)
	}
	data["UserRows"] = rows

	if h.db != nil {
		if sys, sErr := store.SystemStats(ctx, h.db.DB); sErr != nil {
			slog.Error("system stats failed", "err", sErr)
		} else {
			data["SystemStats"] = sys
		}

		if events, eErr := store.RecentEvents(ctx, h.db.DB, adminRecentEventsLimit); eErr != nil {
			slog.Error("recent events failed", "err", eErr)
		} else {
			data["RecentEvents"] = events
		}

		if growth, gErr := store.CreationGrowth(ctx, h.db.DB, adminGrowthDays); gErr != nil {
			slog.Error("creation growth failed", "err", gErr)
		} else {
			data["GrowthBars"] = buildGrowthBars(growth)
			data["GrowthChartWidth"] = len(growth) * (growthBarWidth + growthBarGap)
			data["GrowthChartHeight"] = growthBarChartHeight
		}
	}

	h.render(w, "dashboard.html", data)
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

// addUserOverview attaches the per-account statistics overview (entity
// counts, activity summary, recent events) to the edit-user page data, keyed
// by userID. Counts/Activity are always set to a zero value first so the
// template can access nested fields (e.g. `.Counts.Tasks`) unconditionally
// even when h.db is nil or a stats query fails — each query is independently
// non-fatal: a failure logs via slog and leaves that piece at its zero
// value/omitted, mirroring the Dashboard discipline. Join date is not part of
// this: it comes from the already-loaded auth.User.CreatedAt, which cannot
// fail once FindUserByID has succeeded.
func (h *AdminHandler) addUserOverview(ctx context.Context, data map[string]any, userID string) {
	data["Counts"] = store.EntityCounts{}
	data["Activity"] = store.Activity{}
	if h.db == nil {
		return
	}

	if counts, err := store.PerUserCounts(ctx, h.db.DB); err != nil {
		slog.Error("per-user counts failed", "err", err)
	} else {
		data["Counts"] = counts[userID]
	}

	if activity, err := store.UserActivity(ctx, h.db.DB); err != nil {
		slog.Error("user activity failed", "err", err)
	} else {
		data["Activity"] = activity[userID]
	}

	if events, err := store.RecentEventsByUser(ctx, h.db.DB, userID, adminRecentEventsLimit); err != nil {
		slog.Error("recent events by user failed", "err", err)
	} else {
		data["RecentEvents"] = events
	}
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
		data := map[string]any{
			"CSRFToken": h.sessionCSRF(r),
			"User":      user,
		}
		h.addUserOverview(r.Context(), data, id)
		h.render(w, "user_edit.html", data)
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

// renderEditError re-renders the edit form with a fresh CSRF token and an
// error. It rebuilds the overview data too (addUserOverview), since
// user_edit.html unconditionally reads .Counts/.Activity — leaving them
// unset here would panic the template on the error-repaint path.
func (h *AdminHandler) renderEditError(w http.ResponseWriter, r *http.Request, id, errMsg string) {
	user, err := h.auth.FindUserByID(id)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	data := map[string]any{
		"CSRFToken": h.sessionCSRF(r),
		"User":      user,
		"Error":     errMsg,
	}
	h.addUserOverview(r.Context(), data, id)
	w.WriteHeader(http.StatusUnprocessableEntity)
	h.render(w, "user_edit.html", data)
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
