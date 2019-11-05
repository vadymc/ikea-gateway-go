package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	gw "github.com/vadymc/ikea-gateway-go/m/gateway-handler"
	"github.com/vadymc/ikea-gateway-go/m/ikea"
)

const (
	clientID = "111-222-111"

	ikeaGwIP           = "ikeaGwIP"
	ikeaGwPSK          = "ikeaGwPSK"
	ikeaGwSecurityCode = "ikeaGwSecurityCode" // from the back of gateway device
)

func main() {
	gwIP := os.Getenv(ikeaGwIP)
	gwAddr := gwIP + ":5684"
	psk := os.Getenv(ikeaGwPSK)

	if psk == "" {
		securityCode := os.Getenv(ikeaGwSecurityCode)
		authenticate(gwAddr, clientID, securityCode)
		os.Exit(1)
	}

	tc := ikea.NewTradfriClient(gwAddr, clientID, psk)
	storage := &gw.DBStorage{}
	h := gw.NewHandler(tc, storage)
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				h.PollAndSaveDevicesState()
			}
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func authenticate(gatewayAddress, clientID, psk string) {
	if len(clientID) < 1 || len(psk) < 10 {
		fail("Both clientID and psk args must be specified when performing key exchange")
	}

	done := make(chan bool)
	defer func() { done <- true }()
	go func() {
		select {
		case <-time.After(time.Second * 5):
			fmt.Println("(Please note that the key exchange may appear to be stuck at \"Connecting to peer at\" if the PSK from the bottom of your Gateway is not entered correctly.)")
		case <-done:
		}
	}()

	// Note that we hard-code "Client_identity" here before creating the DTLS client,
	// required when performing token exchange
	dtlsClient := ikea.NewTradfriClient(gatewayAddress, "Client_identity", psk)

	authToken, err := dtlsClient.AuthExchange(clientID)
	if err != nil {
		fail(err.Error())
	}
	os.Setenv(ikeaGwPSK, authToken.Token)
	fmt.Println("Have set PSK token to environment variable, make sure it is being saved between sessions", ikeaGwPSK)
}

func fail(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}
