package realtime

// SignalRMessage is the wire envelope for every server-push message.
// It matches the format expected by Sonarr clients (LunaSea, Notifiarr, etc.):
//
//	{"H":"MessageHub","M":"receiveMessage","A":["series",{...}]}
//
// Clients look for M == "receiveMessage" to dispatch.
type SignalRMessage struct {
	H string `json:"H"` // Hub name: always "MessageHub"
	M string `json:"M"` // Method: always "receiveMessage"
	A []any  `json:"A"` // Arguments: [name, body]
}

// NegotiateResponse is returned by the /signalr/messages/negotiate endpoint.
type NegotiateResponse struct {
	ConnectionID        string          `json:"connectionId"`
	AvailableTransports []TransportInfo `json:"availableTransports"`
}

// TransportInfo describes a single available transport in a NegotiateResponse.
type TransportInfo struct {
	Transport       string   `json:"transport"`
	TransferFormats []string `json:"transferFormats"`
}
