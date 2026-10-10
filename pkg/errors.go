package pkg

type FieldsError struct {
	Fields map[string]any
	Err    error
}

func (e FieldsError) Error() string {
	return e.Err.Error()
}

func (e FieldsError) Unwrap() error {
	return e.Err
}
