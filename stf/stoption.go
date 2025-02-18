package stf

import (
	"context"
	"errors"
)

// StOptions can be used to create a customized stream.
type StOptions struct {

	// Id is an optional id label
	Id string

	// Discrete indicates I/O operate on sequence of length/data blocks
	Discrete bool

	// MaxLen if set, operation checks discrete data length is not above the limit
	MaxLen int

	// MaxNb if set, operation stops the number of discrete data blocks processed after the limit is raised
	MaxNb int

	// Encryption identities, if any
	EncIds []string

	// Encryption recipients, if any
	EncRcps []string
}

// StOption is a function on the options for a stream.
type StOption func(*StOptions) error

// IstOptions can be used to create a customized input stream.
type IstOptions struct {
	StOptions

	// BSize overload the default input buffer size
	BSize int

	// BSet enables to provide continuously the data read to the client until EOF.
	BSet func(context.Context, []byte, bool) error

	// Unmarshaller in Discrete mode converts received raw data to structured data.
	Unmarshaller func(data []byte, v any) error

	// NewASet in Discrete mode provide structured data unmarshal receiver.
	NewASet func() any

	// ASet enables to provide continuously the structured data read to the client.
	ASet func(context.Context, any) error
}

// IstOption is a function on the options for an input stream.
type IstOption func(*IstOptions) error

// IstId is a IstOption to set the stream id.
func IstId(id string) IstOption {
	return func(o *IstOptions) error {
		o.Id = id
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

// IstEncIds is a IstOption to set the stream encryption Ids (private keys)
func IstEncIds(ids []string) IstOption {
	return func(o *IstOptions) error {
		o.EncIds = ids
		return nil
	}
}

// IstEncRcps is a IstOption to set the stream encryption recipients (punlic keys)
func IstEncRcps(rcps []string) IstOption {
	return func(o *IstOptions) error {
		o.EncRcps = rcps
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
func IstBSet(bset func(context.Context, []byte, bool) error) IstOption {
	return func(o *IstOptions) error {
		if bset == nil {
			return errors.New("no BSet function")
		}
		o.BSet = bset
		return nil
	}
}

// IstASet is a IstOption to set the functions unmarshalling and providing the structured data read to the client.
func IstASet(unmarshal func(data []byte, v any) error, na func() any, aset func(context.Context, any) error) IstOption {
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

	// CloseOnTerminate if set and if embedded stream supports Closer interface,
	// calls close() on stream termination
	CloseOnTerminate bool

	// BGet enables to request on demand the data to be written from the client.
	BGet func(context.Context) ([]byte, error)

	// AGet in discrete mode enables to request on demand the structured data to be written from the client.
	AGet func(context.Context) (any, error)

	// Marshaller in discrete mode converts structured data to raw data to be written.
	Marshaller func(any) ([]byte, error)
}

// OstOption is a function on the options for an output stream.
type OstOption func(*OstOptions) error

// OstId is a OstOption to set the stream id.
func OstId(id string) OstOption {
	return func(o *OstOptions) error {
		o.Id = id
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

// OstMaxNb is a OstOption to set the stream discrete limit for number of blocks processed.
func OstMaxNb(i int) OstOption {
	return func(o *OstOptions) error {
		o.MaxNb = i
		return nil
	}
}

// OstEncIds is a OstOption to set the stream encryption Ids (private keys)
func OstEncIds(ids []string) OstOption {
	return func(o *OstOptions) error {
		o.EncIds = ids
		return nil
	}
}

// OstEncRcps is a OstOption to set the stream encryption Recipients (public keys)
func OstEncRcps(rcps []string) OstOption {
	return func(o *OstOptions) error {
		o.EncRcps = rcps
		return nil
	}
}

// OstCloseOnTerminate is a OstOption to close the embedded stream on termination
func OstCloseOnTerminate(b bool) OstOption {
	return func(o *OstOptions) error {
		o.CloseOnTerminate = b
		return nil
	}
}

// OstBGet is a OstOption to set the function requesting on demand the data to be written from the client.
func OstBGet(bget func(context.Context) ([]byte, error)) OstOption {
	return func(o *OstOptions) error {
		if bget == nil {
			return errors.New("no BGet function")
		}
		o.BGet = bget
		return nil
	}
}

// OstAGet is a OstOption to set the functions requesting on demand the structured data to be written from the client, and the marshaller to convert it to raw data.
func OstAGet(aget func(context.Context) (any, error), marshaller func(any) ([]byte, error)) OstOption {
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
