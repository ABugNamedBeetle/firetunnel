package worker

import (
	"encoding/base64"
	"encoding/json"
	"firetunnel/config"
	"firetunnel/modal"
	slg "firetunnel/util/slg"
	"log/slog"
	"os"

	"github.com/r3labs/sse"
)

var (
	simplelogHandler = slg.CreateHandler(os.Stdout, "rtlistener", &config.SlgOpts)
	logger           = slog.New(simplelogHandler)
)

func FireListener(url string, reschan chan []byte) {
	logger.Info("Fire Listner Started", "url", url)
	client := sse.NewClient(url)

	client.Headers["Accept"] = "text/event-stream"
	client.Subscribe("messages", func(msg *sse.Event) {
		processMessageEvent(msg, reschan, url)
	})
	client.OnDisconnect(func(c *sse.Client) {
		logger.Info("Fire Listner CLosed", "url", url)

	})

}

type ChangeEvent struct {
	Path string `json:"path"`
	Data string `json:"data"`
}

func processMessageEvent(msg *sse.Event, reschan chan []byte, url string) {

	datastring := string(msg.Data)

	if datastring != "null" {
		// logger.Info("Change Detected on response:", "message", datastring)
		logger.Debug("Change Detected on response")

		var ce ChangeEvent
		err := json.Unmarshal([]byte(datastring), &ce)
		if err != nil {
			logger.Error("JOSN decode error while reading received packet " + err.Error())
			return
		}

		if ce.Data != "" {
			decodedData, err := base64.StdEncoding.DecodeString(ce.Data)
			if err != nil {
				logger.Error("Decoding Failed for received data string to bytes")
				return
			}
			var fp modal.FirePacket
			err = json.Unmarshal(decodedData, &fp)
			if err != nil {
				logger.Error("Json Decoding Failed for received data")
				return
			}
			logger.Debug("Packet Received in Local", "Request", fp.Request, "Packet", fp.Counter, "Url", url)
			reschan <- fp.Data
		}

	}

}
