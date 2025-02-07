package stf

import "errors"

// StOptions can be used to create a customized stream.
type StOptions struct {

	// Name is an optional name label
	Name string

	// Discrete indicates I/O operate on sequence of length/data blocks
	Discrete bool

	// MaxLen if set, operation checks discrete data length is not above the limit
	MaxLen int

	// MaxNb if set, operation stops the number of discrete data blocks processed after the limit is raised
	MaxNb int
}

// StOption is a function on the options for a stream.
type StOption func(*StOptions) error

// IstOptions can be used to create a customized input stream.
type IstOptions struct {
	StOptions

	// BSize overload the default input buffer size
	BSize int

	// BSet enables to provide continuously the data read to the client.
	BSet func([]byte) error

	// Unmarshaller in Discrete mode converts received raw data to structured data.
	Unmarshaller func(data []byte, v any) error

	// NewASet in Discrete mode provide structured data unmarshal receiver.
	NewASet func() any

	// ASet enables to provide continuously the structured data read to the client.
	ASet func(any) error
}

// IstOption is a function on the options for an input stream.
type IstOption func(*IstOptions) error

// IstName is a IstOption to set the stream name.
func IstName(name string) IstOption {
	return func(o *IstOptions) error {
		o.Name = name
		return nil
	}
}

// IstDiscrete is a IstOption to set the stream discrete mode.
func IstDiscrete(b bool) IstOption {
	return func(o *IstOptions) error {
		o.Discrete = b
		return nil
	}
}

// IstMaxLen is a IstOption to set the stream discrete max block length.
func IstMaxLen(i int) IstOption {
	return func(o *IstOptions) error {
		o.MaxLen = i
		return nil
	}
}

// IstMaxNb is a IstOption to set the stream discrete limit for number of blocks processed.
func IstMaxNb(i int) IstOption {
	return func(o *IstOptions) error {
		o.MaxNb = i
		return nil
	}
}

// IstBsize is a IstOption to set the input buffer size.
func IstBsize(s int) IstOption {
	return func(o *IstOptions) error {
		o.BSize = s
		return nil
	}
}

// IstBSet is a IstOption to set the function providing the data read to the client.
func IstBSet(bset func([]byte) error) IstOption {
	return func(o *IstOptions) error {
		if bset == nil {
			return errors.New("no BSet function")
		}
		o.BSet = bset
		return nil
	}
}

// IstASet is a IstOption to set the functions unmarshalling and providing the structured data read to the client.
func IstASet(unmarshal func(data []byte, v any) error, na func() any, aset func(any) error) IstOption {
	return func(o *IstOptions) error {
		var err error
		if unmarshal == nil {
			err = errors.Join(err, errors.New("no Unmarshal function"))
		}
		if na == nil {
			err = errors.Join(err, errors.New("no NewASet function"))
		}
		if aset == nil {
			err = errors.Join(err, errors.New("no ASet function"))
		}
		if err != nil {
			return err
		}
		o.Unmarshaller = unmarshal
		o.NewASet = na
		o.ASet = aset
		return nil
	}
}

// OstOptions can be used to create a customized output stream.
type OstOptions struct {
	StOptions

	// BGet enables to request on demand the data to be written from the client.
	BGet func() ([]byte, error)

	// AGet in discrete mode enables to request on demand the structured data to be written from the client.
	AGet func() (any, error)

	// Marshaller in discrete mode converts structured data to raw data to be written.
	Marshaller func(any) ([]byte, error)
}

// OstOption is a function on the options for an output stream.
type OstOption func(*OstOptions) error

// OstName is a OstOption to set the stream name.
func OstName(name string) OstOption {
	return func(o *OstOptions) error {
		o.Name = name
		return nil
	}
}

// OstDiscrete is a OstOption to set the stream discrete mode.
func OstDiscrete(b bool) OstOption {
	return func(o *OstOptions) error {
		o.Discrete = b
		return nil
	}
}

// OstMaxLen is a OstOption to set the stream discrete max block length.
func OstMaxLen(i int) OstOption {
	return func(o *OstOptions) error {
		o.MaxLen = i
		return nil
	}
}

// OstMaxNb is a OstOption  to set the stream discrete limit for number of blocks processed.
func OstMaxNb(i int) OstOption {
	return func(o *OstOptions) error {
		o.MaxNb = i
		return nil
	}
}

// OstBGet is a OstOption to set the function requesting on demand the data to be written from the client.
func OstBGet(bget func() ([]byte, error)) OstOption {
	return func(o *OstOptions) error {
		if bget == nil {
			return errors.New("no BGet function")
		}
		o.BGet = bget
		return nil
	}
}

// OstAGet is a OstOption to set the functions requesting on demand the structured data to be written from the client, and the marshaller to convert it to raw data.
func OstAGet(aget func() (any, error), marshaller func(any) ([]byte, error)) OstOption {
	return func(o *OstOptions) error {
		var err error
		if aget == nil {
			err = errors.Join(err, errors.New("no AGet function"))
		}
		if marshaller == nil {
			err = errors.Join(err, errors.New("no Marshaller function"))
		}
		if err != nil {
			return err
		}
		o.AGet = aget
		o.Marshaller = marshaller
		return nil
	}
}
