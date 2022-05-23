package gostage

type item struct {
	fun   func(request *Request) (string, error)
	flags []*Flag
	args  []*Arg
	name  string
	st    *Stage
	help  string
}

func NewItem(fun func(request *Request) (string, error), st *Stage, help string) *item {

	return &item{fun: fun, st: st, help: help, flags: []*Flag{}, args: []*Arg{}}
}

func (i *item) Flag(name string, help string) *Flag {

	f := NewFlag(name, help)

	i.flags = append(i.flags, f)

	return f

}

func (i *item) Arg(name string, help string) *Arg {

	a := NewArg(name, help)

	i.args = append(i.args, a)

	return a
}
