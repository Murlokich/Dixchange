package login

type UseCase struct {
}

func (u *UseCase) Execute(_ Input) (Output, error) {
	return Output{}, nil
}
