package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string) error
	SessionGC(maxLifeTime int64)
}

type Session interface {
	Set(key, value interface{}) error //set session value
	Get(key interface{}) interface{}  //get session value
	Delete(key interface{}) error     //delete session value
	SessionID() string                // retrieve current SessionID
}

type Manager struct {
	lock        sync.Mutex
	CookieName  string //cookie name
	maxlifetime int64
	provider    Provider
}

//generates new session manager
func NewManager(provider Provider, cookieName string, maxlifetime int64) (*Manager, error) {

	return &Manager{provider: provider,
		CookieName:  cookieName,
		maxlifetime: maxlifetime}, nil
}

//generating new session id
func (manager *Manager) sessionId() (string, error) {

	buf := make([]byte, 256)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}

//generate new unique session
func (manager *Manager) generateNewSession() (session Session, sid string) {

	//iterate until we get unique session id
	var err error
	for {
		sid, _ = manager.sessionId()
		session, err = manager.provider.SessionInit(sid)
		if err == nil {
			break
		}
	}
	return
}

//start new unique session
func (manager *Manager) StartSession(w http.ResponseWriter) (session Session) {

	//generate new session
	var sid string
	session, sid = manager.generateNewSession()
	cookie := http.Cookie{
		Name:     manager.CookieName,
		Value:    url.QueryEscape(sid),
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	return
}

//retrieve session
func (manager *Manager) GetSession(r *http.Request) (session Session) {

	cookie, err := r.Cookie(manager.CookieName)
	if err != nil || cookie.Value == "" {
		return nil
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session, err = manager.provider.SessionRead(sid)
		//check if session still valid
		if err != nil {
			return nil
		}
	}
	return
}

//destroy sessionid
//for example, used while log out
func (manager *Manager) DestroySession(w http.ResponseWriter,
	r *http.Request) {

	cookie, err := r.Cookie(manager.CookieName)
	if err != nil || cookie.Value == "" {
		return
	} else {

		manager.lock.Lock()
		defer manager.lock.Unlock()

		manager.provider.SessionDestroy(cookie.Value)

		expiration := time.Now()
		cookie := http.Cookie{
			Name:     manager.CookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Expires:  expiration,
			MaxAge:   -1,
		}
		http.SetCookie(w, &cookie)
	}
}

//garbage collect expired sessions
func (manager *Manager) GC() {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	fmt.Println(manager.CookieName, "GC called!")
	manager.provider.SessionGC(manager.maxlifetime)

	time.AfterFunc(time.Duration(manager.maxlifetime*int64(time.Second)),
		func() { manager.GC() })
}

func (manager *Manager) Init() {
	go manager.GC()
}
