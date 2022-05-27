package gostage

type Arg struct {
	name     string
	value    string
	help     string
	required bool
	item     *item
}

func NewArg(name string, help string, item *item) *Arg {

	return &Arg{name: name, help: help, item: item}
}

func (arg *Arg) Required() *Arg {

	arg.required = true

	return arg

}
