package network

type Notifiee interface {
	Connected(Network, NetworkConn) // called when a connection opened
}

type NotifyBundle struct {
	ConnectedF func(Network, NetworkConn)
}

var _ Notifiee = (*NotifyBundle)(nil)

// Connected calls ConnectedF if it is not null.
func (nb *NotifyBundle) Connected(n Network, c NetworkConn) {
	if nb.ConnectedF != nil {
		nb.ConnectedF(n, c)
	}
}

type NoopNotifiee struct{}

var _ Notifiee = (*NoopNotifiee)(nil)

func (nn *NoopNotifiee) Connected(n Network, c NetworkConn) {}
