package yamux

import "github.com/hashicorp/yamux"

type Stream struct{ *yamux.Stream }

func NewStream(s *yamux.Stream) *Stream {
	return &Stream{s}
}

// TODO(yamux)
func (s *Stream) CloseWrite() error {
	return nil
}

// TODO(yamux)
func (s *Stream) Reset() error {
	return nil
}
