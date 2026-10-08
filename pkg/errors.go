package pkg

type ErrorWithFields struct {
	Fields map[string]any
	Err    error
}

func (e ErrorWithFields) Error() string {
	return e.Err.Error()
}

func (e ErrorWithFields) Unwrap() error {
	return e.Err
}
