package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	cb "github.com/clearblade/Go-SDK"
	mqtt "github.com/clearblade/paho.mqtt.golang"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	platURL      string
	messURL      string
	sysKey       string
	sysSec       string
	deviceName   string
	activeKey    string
	listenPort   string
	topicName    string
	enableTLS    bool
	tlsCertPath  string
	tlsKeyPath   string
	deviceClient *cb.DeviceClient
)

func init() {
	flag.StringVar(&sysKey, "systemKey", "", "system key (required)")
	flag.StringVar(&sysSec, "systemSecret", "", "system secret (required)")
	flag.StringVar(&platURL, "platformURL", "", "platform url (required)")
	flag.StringVar(&messURL, "messagingURL", "", "messaging URL")
	flag.StringVar(&deviceName, "deviceName", "", "name of device (required)")
	flag.StringVar(&activeKey, "activeKey", "", "active key (password) for device (required)")
	flag.StringVar(&listenPort, "receiverPort", "", "receiver port for adapter (required)")
	flag.StringVar(&topicName, "topicName", "monnit-webhook-adapter/<sensor_id>", "topic name to publish received HTTP requests too, <sensor_id> will be replaced with the specific id from the incoming request")
	flag.BoolVar(&enableTLS, "enableTLS", false, "enable TLS on http listener (must provide tlsCertPath and tlsKeyPath params if enabled)")
	flag.StringVar(&tlsCertPath, "tlsCertPath", "", "path to TLS .crt file (required if enableTLS flag is set)")
	flag.StringVar(&tlsKeyPath, "tlsKeyPath", "", "path to TLS .key file (required if enableTLS flag is set)")
}

type incomingMonnitWebhook struct {
	SensorMessages []map[string]interface{} `json:"sensorMessages"`
	GatewayMessage map[string]interface{}   `json:"gatewayMessage"`
}

type monnitData struct {
	SensorMessage  map[string]interface{} `json:"sensor_message"`
	GatewayMessage map[string]interface{} `json:"gateway_message"`
	TimeReceived   string                 `json:"time_received"`
}

func usage() {
	log.Printf("Usage: webhook-adapter [options]\n\n")
	flag.PrintDefaults()
}

func handleRequest(rw http.ResponseWriter, r *http.Request) {
	timeReceived := time.Now().UTC().Format(time.RFC3339)

	log.Println("Received a http request!")
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading body of request: %s\n", err.Error())
		return
	}

	var bodyJSON incomingMonnitWebhook
	if err := json.Unmarshal(body, &bodyJSON); err != nil {
		log.Printf("Unexpect JSON from monnit webhook: %s\n", err.Error())
		return
	}

	for _, sensorMsg := range bodyJSON.SensorMessages {
		msg := &monnitData{
			SensorMessage:  sensorMsg,
			GatewayMessage: bodyJSON.GatewayMessage,
			TimeReceived:   timeReceived,
		}
		b, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to convert monnit data into a string %s\n", err.Error())
			continue
		}
		sensorID := sensorMsg["sensorID"].(string)
		if err := deviceClient.Publish(strings.Replace(topicName, "<sensor_id>", sensorID, -1), b, 2); err != nil {
			log.Printf("Unable to publish request: %s\n", err.Error())
			continue
		}
		log.Println("Message published")
	}
}

func validateFlags() {
	flag.Parse()
	if sysKey == "" || sysSec == "" || platURL == "" || deviceName == "" || activeKey == "" || listenPort == "" {
		log.Printf("Missing required flags\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if _, err := strconv.Atoi(listenPort); err != nil {
		log.Printf("receiverPort must be numeric\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if enableTLS && (tlsCertPath == "" || tlsKeyPath == "") {
		log.Printf("tlsCertPath and tlsKeyPath are required if TLS is enabled\n")
		flag.Usage()
		os.Exit(1)
	}

}

func main() {
	mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New(os.Stdout, "", 0)
	flag.Usage = usage
	validateFlags()

	deviceClient = cb.NewDeviceClient(sysKey, sysSec, deviceName, activeKey)

	if platURL != "" {
		log.Println("Setting custom platform URL to: ", platURL)
		deviceClient.HttpAddr = platURL
	}

	if messURL != "" {
		log.Println("Setting custom messaging URL to: ", messURL)
		deviceClient.MqttAddr = messURL
	}

	log.Println("Authenticating to platform with device: ", deviceName)

	if err := deviceClient.Authenticate(); err != nil {
		log.Fatalf("Error authenticating: %s\n", err.Error())
	}

	if err := deviceClient.InitializeMQTT("webhookadapter_"+deviceName, "", 30, &tls.Config{}, nil); err != nil {
		log.Fatalf("Unable to initialize MQTT: %s\n", err.Error())
	}
	log.Printf("MQTT connected and adapter about to listen on port: %s\n", listenPort)

	http.HandleFunc("/", handleRequest)

	if enableTLS {
		log.Fatal(http.ListenAndServeTLS(":"+listenPort, tlsCertPath, tlsKeyPath, nil))
	} else {
		log.Fatal(http.ListenAndServe(":"+listenPort, nil))
	}
}
