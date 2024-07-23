package message

import "encoding/json"

// MessageDetails represents a WebSocket message.
type MessageDetails struct {
	OriginID string `json:"origin_id"`
	HubID    string `json:"hub_id"`
	SenderID string `json:"sender_id"`
	Message  []byte `json:"message"`
}

// NewMessageDetails creates a new MessageDetails instance.
func NewMessageDetails(originID, hubID, senderID string, message []byte) MessageDetails {
	return MessageDetails{
		OriginID: originID,
		HubID:    hubID,
		SenderID: senderID,
		Message:  message,
	}
}

// IsFromPubSub checks if the message is from the Pub/Sub channel.
func (md *MessageDetails) IsFromPubSub(pubSubChannel string) bool {
	return md.SenderID == pubSubChannel
}

// ShouldBroadcastToClient checks if the message should be broadcast to a given client.
func (md *MessageDetails) ShouldBroadcastToClient(clientID string) bool {
	return md.OriginID != clientID
}

// ToJSON converts the MessageDetails to a JSON string.
func (md *MessageDetails) ToJSON() ([]byte, error) {
	return json.Marshal(md)
}

// FromJSON populates the MessageDetails from a JSON string.
func (md *MessageDetails) FromJSON(data []byte) error {
	return json.Unmarshal(data, md)
}
