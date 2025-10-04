package sessions

import (
	"github.com/gorilla/sessions"
	"net/http"
)

var (
	store       *sessions.CookieStore
	sessionName = "sse-session"
)

func Initialize(secret []byte) {
	store = sessions.NewCookieStore(secret)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}

func Get(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, sessionName)
}

func SetUserSession(w http.ResponseWriter, r *http.Request, userID string) error {
	session, err := Get(r)
	if err != nil {
		return err
	}

	session.Values["authenticated"] = true
	session.Values["user_id"] = userID
	return session.Save(r, w)
}

func ClearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := Get(r)
	if err != nil {
		return err
	}

	session.Values = make(map[interface{}]interface{})
	return session.Save(r, w)
}
