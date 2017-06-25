package memory

import (
	"container/list"
	"errors"
	"github.com/Flynston/gosm/session"
	"sync"
	"time"
)

type SessionStore struct {
	sid          string                      //unique session id
	timeAccessed time.Time                   //last access time
	value        map[interface{}]interface{} //session value stored inside
	provider     *Provider                   //memory provider associated with session
}

//generate new memory provider
func NewProvider() *Provider {
	return &Provider{
		list:     list.New(),
		sessions: make(map[string]*list.Element, 0),
	}
}

type Provider struct {
	lock     sync.Mutex               //lock
	sessions map[string]*list.Element //save in memory
	list     *list.List               //gc
}

func (st *SessionStore) Set(key, value interface{}) error {
	st.value[key] = value
	st.provider.SessionUpdate(st.sid)
	return nil
}

func (st *SessionStore) Get(key interface{}) interface{} {
	st.provider.SessionUpdate(st.sid)
	if v, ok := st.value[key]; ok {
		return v
	}
	return nil
}

func (st *SessionStore) Delete(key interface{}) error {
	delete(st.value, key)
	st.provider.SessionUpdate(st.sid)
	return nil
}

func (st *SessionStore) SessionID() string {
	return st.sid
}

func (pder *Provider) SessionInit(sid string) (session.Session, error) {
	pder.lock.Lock()
	defer pder.lock.Unlock()

	//check if sid in sessions map
	_, ok := pder.sessions[sid]
	if ok {
		return nil, errors.New("session sid not unique!")
	}

	v := make(map[interface{}]interface{}, 0)
	newsess := &SessionStore{sid: sid,
		timeAccessed: time.Now(),
		value:        v,
		provider:     pder}
	element := pder.list.PushBack(newsess)
	pder.sessions[sid] = element
	return newsess, nil
}

func (pder *Provider) SessionRead(sid string) (session.Session, error) {
	if element, ok := pder.sessions[sid]; ok {
		return element.Value.(*SessionStore), nil
	}
	return nil, errors.New("session not valid!")
}

func (pder *Provider) SessionDestroy(sid string) error {
	if element, ok := pder.sessions[sid]; ok {
		delete(pder.sessions, sid)
		pder.list.Remove(element)
	}
	return nil
}

func (pder *Provider) SessionGC(maxlifetime int64) {
	pder.lock.Lock()
	defer pder.lock.Unlock()

	for {
		element := pder.list.Back()
		if element == nil {
			break
		}

		if element.Value.(*SessionStore).timeAccessed.Unix()+maxlifetime < time.Now().Unix() {
			pder.list.Remove(element)
			delete(pder.sessions, element.Value.(*SessionStore).sid)
		} else {
			break
		}
	}
}

func (pder *Provider) SessionUpdate(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()

	if element, ok := pder.sessions[sid]; ok {
		element.Value.(*SessionStore).timeAccessed = time.Now()
		pder.list.MoveToFront(element)
	}
	return nil
}
