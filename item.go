package gostage

type item struct {
	fun      func(request *Request) (string, error)
	flags    []*Flag
	args     []*Arg
	name     string
	st       *Stage
	help     string
	isNormal bool
	hide     bool
}

func NewItem(name string, fun func(request *Request) (string, error), st *Stage, help string) *item {

	return &item{fun: fun, st: st, help: help, flags: []*Flag{}, args: []*Arg{}, name: name}
}

func (i *item) Flag(name string, help string) *Flag {

	f := NewFlag(name, help, i)

	i.flags = append(i.flags, f)

	return f

}

func (i *item) Arg(name string, help string) *Arg {

	a := NewArg(name, help, i)

	i.args = append(i.args, a)

	return a
}

func (i *item) IsNormal() {

	i.isNormal = true
}

func (i *item) Hide() *item {

	i.hide = true

	return i
}
