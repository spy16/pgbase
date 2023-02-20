package auth

import (
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/markbates/goth"

	"github.com/spy16/pgbase/errors"
	"github.com/spy16/pgbase/httputils"
	"github.com/spy16/pgbase/strutils"
)

const redirectToParam = "redirect_to"

// Routes installs auth module routes onto the given router.
func (auth *Auth) Routes(r chi.Router) {
	r.Post("/register", auth.handleRegister)
	r.Post("/login", auth.handleLogin)
	r.Get("/logout", auth.handleLogout)

	r.Get("/oauth2", auth.handleOAuth2Redirect)
	r.Get("/oauth2/cb", auth.handleOAuth2Callback)

	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticate())

		r.Get("/me", httputils.HandlerFuncE(auth.handleWhoAmI))
	})
}

func (auth *Auth) handleRegister(w http.ResponseWriter, r *http.Request) {
	var creds userCreds
	if err := creds.readFrom(r); err != nil {
		respond(w, r, 0, err)
		return
	} else if !strutils.IsValidEmail(creds.Email) {
		respond(w, r, 0, errors.MissingAuth.Hintf("invalid email"))
		return
	}

	pwdHash, err := HashPassword(creds.Password)
	if err != nil {
		respond(w, r, 0, err)
		return
	}

	u := NewUser(creds.Kind, creds.Username, creds.Email)
	u.PwdHash = &pwdHash

	registeredU, err := auth.RegisterUser(r.Context(), u, nil)
	if err != nil {
		if errors.OneOf(err, []error{errors.Conflict, errors.InvalidInput}) {
			respond(w, r, 0, err)
			return
		}
		respond(w, r, 0, errors.InternalIssue.CausedBy(err))
		return
	}

	respond(w, r, http.StatusCreated, registeredU.Clone(true))
}

func (auth *Auth) handleOAuth2Redirect(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userKind := q.Get("kind")
	providerID := q.Get("p")

	if userKind == "" {
		userKind = defaultUserKind
	}
	if !strutils.OneOf(userKind, auth.cfg.EnabledKinds) {
		respond(w, r, 0,
			errors.InvalidInput.Coded("invalid_kind").Hintf("user kind '%s' is not valid", userKind))
		return
	}

	p, err := goth.GetProvider(providerID)
	if err != nil {
		respond(w, r, 0, errors.InvalidInput.Coded("invalid_provider").CausedBy(err))
		return
	}

	state := strutils.RandStr(10)

	sess, err := p.BeginAuth(state)
	if err != nil {
		respond(w, r, 0, errors.InternalIssue.CausedBy(err))
		return
	}

	authURL, err := sess.GetAuthURL()
	if err != nil {
		respond(w, r, 0, errors.InternalIssue.CausedBy(err))
		return
	}

	setOAuthFlowState(w, &oauth2FlowState{
		UserKind:   userKind,
		Provider:   p.Name(),
		Session:    sess.Marshal(),
		RedirectTo: r.FormValue(redirectToParam),
	})

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (auth *Auth) handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	var errInvalidCB = errors.InvalidInput.Coded("invalid_callback")

	flowState := popOAuthState(w, r)
	if flowState == nil {
		respond(w, r, 0, errInvalidCB.Hintf("oauth2 flow state is nil"))
		return
	}
	_ = r.ParseForm()
	r.Form.Set(redirectToParam, flowState.RedirectTo)

	p, err := goth.GetProvider(flowState.Provider)
	if err != nil {
		respond(w, r, 0, errInvalidCB.CausedBy(err))
		return
	}

	sess, err := p.UnmarshalSession(flowState.Session)
	if err != nil {
		respond(w, r, 0, errInvalidCB.CausedBy(err))
		return
	}

	q := r.URL.Query()
	if !checkCallbackState(sess, q.Get("state")) {
		respond(w, r, 0, errInvalidCB.Hintf("state value mismatch"))
		return
	}

	if _, err := sess.Authorize(p, q); err != nil {
		respond(w, r, 0, errors.InternalIssue.CausedBy(err))
		return
	}

	gothUser, err := p.FetchUser(sess)
	if err != nil {
		respond(w, r, 0, errors.InternalIssue.CausedBy(err))
		return
	}

	loginKeyID := NewAuthKey(gothUser.Provider, gothUser.UserID)
	exU, err := auth.GetUser(r.Context(), loginKeyID)
	if err != nil && !errors.Is(err, errors.NotFound) {
		respond(w, r, 0, errors.InternalIssue.CausedBy(err))
		return
	}

	if exU == nil {
		// new user registration
		newU := NewUser(flowState.UserKind, "", gothUser.Email)
		newU.Data = userDataFromGothUser(gothUser)

		loginKey := Key{
			Key: loginKeyID,
			Attribs: map[string]any{
				"user_id":       gothUser.UserID,
				"expires_at":    gothUser.ExpiresAt.Unix(),
				"access_token":  gothUser.AccessToken,
				"refresh_token": gothUser.RefreshToken,
				"raw_data":      gothUser.RawData,
			},
		}

		exU, err = auth.RegisterUser(r.Context(), newU, []Key{loginKey})
		if err != nil {
			if !errors.OneOf(err, []error{errors.Conflict}) {
				err = errors.InternalIssue.CausedBy(err)
			}
			respond(w, r, 0, err)
			return
		}
	} else {
		// update existing user
	}

	auth.finishLogin(w, r, *exU)
}

func (auth *Auth) handleVerify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	id := q.Get("id")
	verifyToken := q.Get("token")

	u, err := auth.VerifyUser(r.Context(), id, verifyToken)
	if err != nil {
		if errors.Is(err, errors.NotFound) {
			err = errors.MissingAuth
		}
		respond(w, r, 0, err)
		return
	}

	auth.finishLogin(w, r, *u)
}

func (auth *Auth) handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds userCreds
	if err := creds.readFrom(r); err != nil {
		respond(w, r, 0, err)
		return
	}

	keyKind := KeyKindUsername
	keyValue := creds.Username
	if creds.Email != "" {
		keyKind = KeyKindEmail
		keyValue = creds.Email
	}

	u, err := auth.GetUser(r.Context(), NewAuthKey(keyKind, keyValue))
	if err != nil {
		if errors.Is(err, errors.NotFound) {
			err = errors.MissingAuth.Hintf("user not found")
		}
		respond(w, r, 0, err)
		return
	} else if !u.CheckPassword(creds.Password) {
		respond(w, r, 0, errors.MissingAuth.Hintf("password mismatch"))
		return
	} else if creds.Email != "" && creds.Email != u.Email {
		respond(w, r, 0, errors.MissingAuth.Hintf("email mismatch"))
		return
	} else if u.Kind != creds.Kind {
		respond(w, r, 0, errors.MissingAuth.Hintf("user kind mismatch"))
		return
	}

	auth.finishLogin(w, r, *u)
}

func (auth *Auth) handleWhoAmI(w http.ResponseWriter, r *http.Request) error {
	session := CurSession(r.Context())
	if session == nil {
		return errors.MissingAuth
	}

	u, err := auth.GetUser(r.Context(), NewAuthKey(KeyKindID, session.UserID))
	if err != nil {
		if errors.Is(err, errors.NotFound) {
			return errors.MissingAuth
		}
		return errors.InternalIssue.CausedBy(err)
	}

	httputils.WriteJSON(w, r, http.StatusOK, u.Clone(true))
	return nil
}

func (auth *Auth) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.cfg.SessionCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Now(),
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	respond(w, r, http.StatusNoContent, nil)
}

func (auth *Auth) finishLogin(w http.ResponseWriter, r *http.Request, user User) {
	session, err := auth.CreateSession(r.Context(), user)
	if err != nil {
		respond(w, r, 0, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.cfg.SessionCookie,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	respond(w, r, http.StatusOK, map[string]any{
		"user":   user.Clone(true),
		"token":  session.Token,
		"expiry": session.ExpiresAt,
	})
}

func respond(w http.ResponseWriter, r *http.Request, status int, val any) {
	redirectTo := r.FormValue(redirectToParam)

	if redirectTo != "" {
		u, parseErr := url.Parse(redirectTo)
		if parseErr == nil {
			if err, ok := val.(error); ok {
				e := errors.E(err)
				q := url.Values{
					"err_code": {e.Code},
				}
				u.RawQuery = q.Encode()
			}
			http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
			return
		}
	}

	if e, ok := val.(error); ok {
		httputils.WriteErr(w, r, e)
	} else {
		httputils.WriteJSON(w, r, status, val)
	}
	return
}

func userDataFromGothUser(gu goth.User) UserData {
	// TODO: extract more from raw data?
	return map[string]any{
		"name":      gu.Name,
		"picture":   gu.AvatarURL,
		"location":  gu.Location,
		"nick_name": gu.NickName,
	}
}
