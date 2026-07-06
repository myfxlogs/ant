package service

type AccountStatus string

const (
	StatusConnecting   AccountStatus = "connecting"
	StatusConnected    AccountStatus = "connected"
	StatusDisconnected AccountStatus = "disconnected"
	StatusFrozen       AccountStatus = "frozen"
)

func (s AccountStatus) String() string { return string(s) }
