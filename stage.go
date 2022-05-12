package gostage

type Stage struct {
	server *Server
	list   map[string]func(st *Stage)
}

func NewStage() *Stage {

	return &Stage{}
}

func (st *Stage) Add(param string, f func(st *Stage)) {

	st.list[param] = f

}

func Run() {

}
