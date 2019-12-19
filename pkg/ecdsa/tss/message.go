package tss

import (
	"fmt"

	"github.com/keep-network/keep-tecdsa/pkg/ecdsa/tss/gen/pb"
)

// TSSProtocolMessage is a network message used to transport messages generated in
// TSS protocol execution. It is a wrapper over a message generated by underlying
// implementation of the protocol.
type TSSProtocolMessage struct {
	SenderID    MemberID
	Payload     []byte
	IsBroadcast bool
}

// Type returns a string type of the `TSSMessage` so that it conforms to
// `net.Message` interface.
func (m *TSSProtocolMessage) Type() string {
	return "ecdsa/tss_message"
}

// Marshal converts this message to a byte array suitable for network communication.
func (m *TSSProtocolMessage) Marshal() ([]byte, error) {
	return (&pb.TSSProtocolMessage{
		SenderID:    m.SenderID.string(),
		Payload:     m.Payload,
		IsBroadcast: m.IsBroadcast,
	}).Marshal()
}

// Unmarshal converts a byte array produced by Marshal to a message.
func (m *TSSProtocolMessage) Unmarshal(bytes []byte) error {
	pbMsg := &pb.TSSProtocolMessage{}
	if err := pbMsg.Unmarshal(bytes); err != nil {
		return err
	}

	m.SenderID = MemberID(pbMsg.SenderID)
	m.Payload = pbMsg.Payload
	m.IsBroadcast = pbMsg.IsBroadcast

	return nil
}

// JoinMessage is a network message used to notify peer members about readiness
// to start protool execution.
type JoinMessage struct {
	SenderID MemberID
}

// Type returns a string type of the `JoinMessage`.
func (m *JoinMessage) Type() string {
	return fmt.Sprintf("%T", m)
}

// Marshal converts this message to a byte array suitable for network communication.
func (m *JoinMessage) Marshal() ([]byte, error) {
	return (&pb.JoinMessage{
		SenderID: m.SenderID.string(),
	}).Marshal()
}

// Unmarshal converts a byte array produced by Marshal to a message.
func (m *JoinMessage) Unmarshal(bytes []byte) error {
	pbMsg := &pb.JoinMessage{}
	if err := pbMsg.Unmarshal(bytes); err != nil {
		return err
	}

	m.SenderID = MemberID(pbMsg.SenderID)

	return nil
}
