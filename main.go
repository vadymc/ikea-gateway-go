package main

import (
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	gw "github.com/vadymc/ikea-gateway-go/m/gateway-handler"
	"github.com/vadymc/ikea-gateway-go/m/ikea"
)

const (
	clientID = "111-222-111"

	ikeaGwIP           = "IKEA_GW_IP"
	ikeaGwPSK          = "IKEA_GW_PSK"
	ikeaGwSecurityCode = "IKEA_GW_SECURITY_CODE" // from the back of gateway device
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
	dbStorage := gw.NewDBStorage()
	firebaseStorage := gw.NewFirebaseStorage()
	h := gw.NewHandler(tc, dbStorage, firebaseStorage)
	ticker := time.NewTicker(1 * time.Minute)
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
		log.Error("Both clientID and psk args must be specified when performing key exchange")
		os.Exit(1)
	}

	done := make(chan bool)
	defer func() { done <- true }()
	go func() {
		select {
		case <-time.After(time.Second * 5):
			log.Info("(Please note that the key exchange may appear to be stuck at \"Connecting to peer at\" if the PSK from the bottom of your Gateway is not entered correctly.)")
		case <-done:
		}
	}()

	// Note that we hard-code "Client_identity" here before creating the DTLS client,
	// required when performing token exchange
	dtlsClient := ikea.NewTradfriClient(gatewayAddress, "Client_identity", psk)

	authToken, err := dtlsClient.AuthExchange(clientID)
	if err != nil {
		log.WithError(err).Error("Failed AuthExchange")
		os.Exit(1)
	}
	os.Setenv(ikeaGwPSK, authToken.Token)
	log.WithField("env var", ikeaGwPSK).Info("Have set PSK token to environment variable, make sure it is being saved between sessions")
}
