package gostage

import (
	"context"
	"sync"
)

type Request struct {
	name  string
	flags map[string]string
	args  map[string]string
	st    *Stage
	//conn  net.Conn
	lock sync.Mutex
}

func NewRequest(st *Stage, name string, flags map[string]string, args map[string]string) *Request {

	return &Request{name: name, flags: flags, args: args, st: st, lock: sync.Mutex{}}
}

func (request *Request) Get(key string) (string, bool) {

	return request.st.Get(key)
}

func (request *Request) Set(key string, value string) {

	request.st.Set(key, value)
}

func (request *Request) Remove(key string) {

	request.st.Remove(key)
}

func (request *Request) GetObj(key any) (any, bool) {

	return request.st.GetObj(key)
}

func (request *Request) SetObj(key any, value any) {

	request.st.SetObj(key, value)
}

func (request *Request) RemoveObj(key any) {

	request.st.RemoveObj(key)
}

func (request *Request) GetFlag(key string) string {

	request.lock.Lock()

	defer request.lock.Unlock()

	return request.flags[key]

}

func (request *Request) GetFlags() map[string]string {

	return request.flags
}

func (request *Request) GetArg(key string) string {

	request.lock.Lock()

	defer request.lock.Unlock()

	return request.args[key]

}

func (request *Request) GetArgs() map[string]string {

	return request.args
}

func (request *Request) GetCxt() context.Context {

	return request.st.GetCxt()
}
