package coap

import (
	"os"
	"time"

	"github.com/dustin/go-coap"
	"github.com/eriklupander/dtls"
	log "github.com/sirupsen/logrus"
)

// DtlsClient provides an domain-agnostic CoAP-client with DTLS transport.
// Copy of https://github.com/eriklupander/tradfri-go/blob/master/dtlscoap/dtlscoap-client.go with removed excessive logging.
type DtlsClient struct {
	peer           *dtls.Peer
	msgID          uint16
	gatewayAddress string
	clientID       string
	psk            string
}

// NewDtlsClient acts as factory function, returns a pointer to a connected (or will panic) DtlsClient.
func NewDtlsClient(gatewayAddress, clientID, psk string) *DtlsClient {
	client := &DtlsClient{
		gatewayAddress: gatewayAddress,
		clientID:       clientID,
		psk:            psk,
	}
	client.Connect()
	return client
}

func (dc *DtlsClient) Connect() {
	dc.setupKeystore()

	listener, err := dtls.NewUdpListener(":0", time.Second*900)
	if err != nil {
		panic(err.Error())
	}

	peerParams := &dtls.PeerParams{
		Addr:             dc.gatewayAddress,
		Identity:         dc.clientID,
		HandshakeTimeout: time.Second * 15}
	log.WithField("GW address", dc.gatewayAddress).Info("Connecting to peer")

	dc.peer, err = listener.AddPeerWithParams(peerParams)
	if err != nil {
		log.WithField("GW address", dc.gatewayAddress).WithError(err).Error("Unable to connect to Gateway")
		os.Exit(1)
	}
	dc.peer.UseQueue(true)
	log.WithField("GW address", dc.gatewayAddress).Info("DTLS connection established")
}

// Call writes the supplied coap.Message to the peer
func (dc *DtlsClient) Call(req coap.Message) (coap.Message, error) {
	data, err := req.MarshalBinary()
	if err != nil {
		return coap.Message{}, err
	}
	err = dc.peer.Write(data)

	if err != nil {
		return coap.Message{}, err
	}

	respData, err := dc.peer.Read(time.Second)
	if err != nil {
		return coap.Message{}, err
	}

	msg, err := coap.ParseMessage(respData)
	if err != nil {
		return coap.Message{}, err
	}

	return msg, nil
}

// BuildGETMessage produces a CoAP GET message with the next msgID set.
func (dc *DtlsClient) BuildGETMessage(path string) coap.Message {
	dc.msgID++
	req := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.GET,
		MessageID: dc.msgID,
	}
	req.SetPathString(path)
	return req
}

// BuildPUTMessage produces a CoAP PUT message with the next msgID set.
func (dc *DtlsClient) BuildPUTMessage(path string, payload string) coap.Message {
	dc.msgID++

	req := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.PUT,
		MessageID: dc.msgID,
		Payload:   []byte(payload),
	}
	req.SetPathString(path)

	return req
}

// BuildPOSTMessage produces a CoAP POST message with the next msgID set.
func (dc *DtlsClient) BuildPOSTMessage(path string, payload string) coap.Message {
	dc.msgID++

	req := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.POST,
		MessageID: dc.msgID,
		Payload:   []byte(payload),
	}
	req.SetPathString(path)

	return req
}

func (dc *DtlsClient) setupKeystore() {
	mks := dtls.NewKeystoreInMemory()
	dtls.SetKeyStores([]dtls.Keystore{mks})
	mks.AddKey(dc.clientID, []byte(dc.psk))
}
