# Golang Session Manager

Simple golang session manager library based on [OWASP 
Session Management Guide](https://www.owasp.org/index.php/Session_Management_Cheat_Sheet). 

## Installation

**gosm** is splitted by design into small single-purpose packages for 
ease of use. You should install the `session` package and choice of
storage engine, currently only in-memory storage engine is supported.

So it is

```
go get github.com/Flynston/gosm/memory
go get github.com/Flynston/gosm/session
```

## Usage


Create session manager using `session.NewManager`:

```go
//generates new session manager
func NewManager(provider Provider, cookieName string, maxlifetime int64) (*Manager, error)
```

After that start them:

```
//start session manager
func (manager *Manager) Init()
```

Example:

```go
import (
	"github.com/Flynston/gosm/memory"
	"github.com/Flynston/gosm/session"
)

var globalPrivateSessions *session.Manager
var globalAnonymousSessions *session.Manager

...

func initSessionManagers() {
	var err error

	//create anonymous session manager
	anonymousMemoryProvider := memory.NewProvider()
	globalAnonymousSessions, err = session.NewManager(anonymousMemoryProvider, "sessionid", 1*60)

	if err != nil {
		panic(err.Error())
	}

	//create private session manager
	privateMemoryProvider := memory.NewProvider()
	globalPrivateSessions, err = session.NewManager(privateMemoryProvider, "session_id", 5*60)

	if err != nil {
		panic(err.Error())
	}

	//start session managers
	globalPrivateSessions.Init()
	globalAnonymousSessions.Init()
}
```

Retrieve session:

```go
func (manager *Manager) GetSession(r *http.Request) (session Session) 
```

example:

```go
session := globalPrivateSessions.GetSession(r)
if session == nil {
	...
}
```

Retrieve key/value from session:

```go
func (st *SessionStore) Get(key interface{}) interface{} 
```

Set key/value to session:

```go
func (st *SessionStore) Set(key, value interface{}) error
```

Delete key from session:

```go
func (st *SessionStore) Delete(key interface{}) error
```

full example:

```go
func login(w http.ResponseWriter, r *http.Request) {

	//check if user authenticated
	session := globalPrivateSessions.GetSession(r)
	if session != nil {
		http.Redirect(w, r, "/", 302)
	}

	if r.Method == "POST" {
		username := template.HTMLEscapeString(r.PostFormValue("username"))
		password := template.HTMLEscapeString(r.PostFormValue("password"))
		//retrieve user struct from database with username
		user, err := data.UserByName(username)

		if err != nil {
			danger(err.Error())
		}

		if user.Password == data.Encrypt(password) {

			//destroy anonymous session
			globalAnonymousSessions.DestroySession(w, r)
			//start new private session
			session = globalPrivateSessions.StartSession(w)
			session.Set("user", user)
			info("user authenticated: ", user.Name)
			http.Redirect(w, r, "/", 302)
			return
		}
	}

	//start new unauthorized session
	session = globalAnonymousSessions.GetSession(r)
	if session == nil {
		globalAnonymousSessions.StartSession(w)
	}
	t, _ := template.ParseFiles("templates/login.html")
	t.Execute(w, nil)
}
```

just see source code
