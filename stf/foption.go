package stf

// FcOptions can be used to create a customized function.
type FcOptions struct {

	// Id is an optional identifier label
	Id string

	// Terminable enable to send Terminate message to a backgroup function
	Terminable bool
}

// FcOption is a function on the options for a function.
type FcOption func(*FcOptions) error

// FcId is a FcOption to set the function id.
func FcId(id string) FcOption {
	return func(o *FcOptions) error {
		o.Id = id
		return nil
	}
}

// FcTerminable is a FcOption to make the function terminable.
func FcTerminable(b bool) FcOption {
	return func(o *FcOptions) error {
		o.Terminable = b
		return nil
	}
}
