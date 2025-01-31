package stf

// FcOptions can be used to create a customized function.
type FcOptions struct {

	// Name is an optional name label
	Name string
}

// FcOption is a function on the options for a function.
type FcOption func(*FcOptions) error

// FcName is a FcOption to set the function name.
func FcName(name string) FcOption {
	return func(o *FcOptions) error {
		o.Name = name
		return nil
	}
}
