package size

import "github.com/urfave/cli/v3"

type Flag = cli.FlagBase[Size, cli.NoConfig, valueCreator]

type valueCreator struct {
}

func (v valueCreator) Create(t Size, t2 *Size, c cli.NoConfig) cli.Value {
	*t2 = t
	return &value{size: t}
}

func (v valueCreator) ToString(t Size) string {
	return t.String()
}

type value struct {
	size Size
}

func (v *value) String() string {
	return v.size.String()
}

func (v *value) Set(s string) error {
	tmp, err := FromString(s)
	if err != nil {
		return err
	}
	v.size = tmp
	return nil
}

func (v *value) Get() any {
	return v.size
}
