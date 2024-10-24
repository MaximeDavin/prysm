package network

type Notifiee interface {
	Connected(Network, Conn) // called when a connection opened
}

type NotifyBundle struct {
	ConnectedF func(Network, Conn)
}

var _ Notifiee = (*NotifyBundle)(nil)

// Connected calls ConnectedF if it is not null.
func (nb *NotifyBundle) Connected(n Network, c Conn) {
	if nb.ConnectedF != nil {
		nb.ConnectedF(n, c)
	}
}

type NoopNotifiee struct{}

var _ Notifiee = (*NoopNotifiee)(nil)

func (nn *NoopNotifiee) Connected(n Network, c Conn) {}
