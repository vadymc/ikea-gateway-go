package main

import (
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vadymc/telegram-client-go/v2"

	"github.com/robfig/cron/v3"

	gw "github.com/vadymc/ikea-gateway-go/m/handler"
	"github.com/vadymc/ikea-gateway-go/m/ikea"
	"github.com/vadymc/ikea-gateway-go/m/sql"
	"github.com/vadymc/ikea-gateway-go/m/stat"
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
	telegramClient := telegram.NewTelegramClient()

	// configure gateway state polling
	tc := ikea.NewTradfriClient(gwAddr, clientID, psk, telegramClient)
	dbStorage := sql.NewDBStorage()
	h := gw.NewHandler(tc, dbStorage)

	// init
	stat.CalcQuantiles(dbStorage)

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		h.PollAndSaveDevicesState()
		for {
			select {
			case <-ticker.C:
				h.PollAndSaveDevicesState()
			}
		}
	}()

	// configure cron jobs
	cr := cron.New(cron.WithLocation(time.UTC))
	cr.AddFunc("@midnight", func() { tc.RebootGateway() })
	cr.AddFunc("30 0 * * *", func() { stat.CalcQuantiles(dbStorage) })
	cr.Start()

	go func() { telegramClient.SendMessage("Ikea GW", "Started Ikea Gateway") }()
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
	dtlsClient := ikea.NewTradfriClient(gatewayAddress, "Client_identity", psk, nil)

	authToken, err := dtlsClient.AuthExchange(clientID)
	if err != nil {
		log.WithError(err).Error("Failed AuthExchange")
		os.Exit(1)
	}
	os.Setenv(ikeaGwPSK, authToken.Token)
	log.WithField("env var", ikeaGwPSK).Info("Have set PSK token to environment variable, make sure it is being saved between sessions")
}
