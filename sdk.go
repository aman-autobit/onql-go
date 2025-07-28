package onqlsdk

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// Message defines the structure for communication
type Message struct {
	Command string            `json:"command"`
	Args    string            `json:"args"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// Response defines standard reply format
type Response struct {
	Status  string `json:"status"` // "ok" / "error"
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// ONQLsdk is the main SDK structure
type ONQLsdk struct {
	nc      *nats.Conn
	subject string

	// Lifecycle callbacks
	onInstall   func()
	onUninstall func()
	onDelete    func()
	onActive    func()
}

// Singleton instance
var globalSDK *ONQLsdk

// Init initializes the SDK and NATS connection
func Init(subject string) {
	nc, err := nats.Connect("nats://host.docker.internal:4222")
	if err != nil {
		log.Fatal("❌ Failed to connect", err)
	}

	globalSDK = &ONQLsdk{
		nc:      nc,
		subject: "onql." + subject,
	}

	// Run lifecycle hooks if defined
	if globalSDK.onInstall != nil {
		globalSDK.onInstall()
	}
	if globalSDK.onActive != nil {
		globalSDK.onActive()
	}
}

// Request sends a request to another subject
func Request(toSubject string, msg Message) (Response, error) {
	data, _ := json.Marshal(msg)
	m, err := globalSDK.nc.Request(toSubject, data, nats.DefaultTimeout)
	if err != nil {
		return Response{}, err
	}
	var resp Response
	json.Unmarshal(m.Data, &resp)
	return resp, nil
}

// Respond sends a response back to the client
func Respond(original Message, resp Response) {
	data, _ := json.Marshal(resp)
	err := globalSDK.nc.Publish(globalSDK.subject+".response", data)
	if err != nil {
		fmt.Println("❌ Failed to respond:", err)
	}
}

// Subscribe to a topic and handle incoming messages
//
//	func Subscribe(subject string, handler func(Message)) error {
//		_, err := globalSDK.nc.Subscribe(subject, func(m *nats.Msg) {
//			var msg Message
//			json.Unmarshal(m.Data, &msg)
//			handler(msg)
//		})
//		return err
//	}
func Subscribe(subject string, handler func(Message) Response) error {
	_, err := globalSDK.nc.Subscribe(subject, func(m *nats.Msg) {
		var msg Message
		json.Unmarshal(m.Data, &msg)

		resp := handler(msg)
		data, _ := json.Marshal(resp)
		m.Respond(data) // 🔥 use reply subject from NATS
	})
	return err
}

// Lifecycle hook setters
func OnInstall(f func())   { globalSDK.onInstall = f }
func OnUninstall(f func()) { globalSDK.onUninstall = f }
func OnDelete(f func())    { globalSDK.onDelete = f }
func OnActive(f func())    { globalSDK.onActive = f }

// Wait blocks forever (for main thread)
func Wait() {
	select {}
}
