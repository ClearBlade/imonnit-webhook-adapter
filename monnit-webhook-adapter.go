package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	cb "github.com/clearblade/Go-SDK"
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

const digicertGlobalRootCA = `-----BEGIN CERTIFICATE-----
MIIDrzCCApegAwIBAgIQCDvgVpBCRrGhdWrJWZHHSjANBgkqhkiG9w0BAQUFADBh
MQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3
d3cuZGlnaWNlcnQuY29tMSAwHgYDVQQDExdEaWdpQ2VydCBHbG9iYWwgUm9vdCBD
QTAeFw0wNjExMTAwMDAwMDBaFw0zMTExMTAwMDAwMDBaMGExCzAJBgNVBAYTAlVT
MRUwEwYDVQQKEwxEaWdpQ2VydCBJbmMxGTAXBgNVBAsTEHd3dy5kaWdpY2VydC5j
b20xIDAeBgNVBAMTF0RpZ2lDZXJ0IEdsb2JhbCBSb290IENBMIIBIjANBgkqhkiG
9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4jvhEXLeqKTTo1eqUKKPC3eQyaKl7hLOllsB
CSDMAZOnTjC3U/dDxGkAV53ijSLdhwZAAIEJzs4bg7/fzTtxRuLWZscFs3YnFo97
nh6Vfe63SKMI2tavegw5BmV/Sl0fvBf4q77uKNd0f3p4mVmFaG5cIzJLv07A6Fpt
43C/dxC//AH2hdmoRBBYMql1GNXRor5H4idq9Joz+EkIYIvUX7Q6hL+hqkpMfT7P
T19sdl6gSzeRntwi5m3OFBqOasv+zbMUZBfHWymeMr/y7vrTC0LUq7dBMtoM1O/4
gdW7jVg/tRvoSSiicNoxBN33shbyTApOB6jtSj1etX+jkMOvJwIDAQABo2MwYTAO
BgNVHQ8BAf8EBAMCAYYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUA95QNVbR
TLtm8KPiGxvDl7I90VUwHwYDVR0jBBgwFoAUA95QNVbRTLtm8KPiGxvDl7I90VUw
DQYJKoZIhvcNAQEFBQADggEBAMucN6pIExIK+t1EnE9SsPTfrgT1eXkIoyQY/Esr
hMAtudXH/vTBH1jLuG2cenTnmCmrEbXjcKChzUyImZOMkXDiqw8cvpOp/2PV5Adg
06O/nVsJ8dWO41P0jmP6P6fbtGbfYmbW0W5BjfIttep3Sp+dWOIrWcBAI+0tKIJF
PnlUkiaY4IBIqDfv8NZ5YBberOgOzW6sRBc4L0na4UU+Krk2U886UAb3LujEV0ls
YSEY1QSteDwsOoBrp+uvFRTp2InBuThs4pFsiv9kuXclVzDAGySj4dzp30d8tbQk
CAUw7C29C79Fv1C5qfPrmAESrciIxpg0X40KPMbp1ZWVbd4=
-----END CERTIFICATE-----`

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

	rootCA := x509.NewCertPool()
	loadCA := rootCA.AppendCertsFromPEM([]byte(digicertGlobalRootCA))
	if !loadCA {
		log.Fatalf("Failed to load digicert global root ca")
	}

	tlsConfig := &tls.Config{RootCAs: rootCA}

	if err := deviceClient.InitializeMQTT("webhookadapter_"+deviceName, "", 30, tlsConfig, nil); err != nil {
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
