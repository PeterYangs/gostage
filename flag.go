package gostage

type Flag struct {
	name     string
	value    string
	help     string
	required bool
	short    rune
	isBool   bool
}

func NewFlag(name string, help string) *Flag {

	return &Flag{name: name, help: help}
}

func (flag *Flag) Required() *Flag {

	flag.required = true

	return flag

}

func (flag *Flag) Short(name rune) *Flag {

	flag.short = name

	return flag

}

func (flag *Flag) Bool() *Flag {

	flag.isBool = true

	return flag

}
