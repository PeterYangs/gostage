package gostage

type Arg struct {
	name     string
	value    string
	help     string
	required bool
}

func NewArg(name string, help string) *Arg {

	return &Arg{name: name, help: help}
}

func (arg *Arg) Required() *Arg {

	arg.required = true

	return arg

}
