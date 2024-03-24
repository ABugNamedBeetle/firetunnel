package peerworker

import (
	"encoding/base64"
	"encoding/json"
	"firetunnel/config"
	"firetunnel/modal"
	slg "firetunnel/util/slg"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/r3labs/sse"
)

var (
	slgOpts = slg.Options{
		Ansi:  true,
		Level: slog.LevelDebug,
	}
	simplelogHandler = slg.CreateHandler(os.Stdout, "peerlistener", &config.SlgOpts)
	logger           = slog.New(simplelogHandler)
)

func FireListener(url string, requestIdChan chan int) {
	logger.Info("Fire Listner Started", "url", url)
	client := sse.NewClient(url)
	client.Headers["Accept"] = "text/event-stream"
	client.Subscribe("messages", func(msg *sse.Event) {
		processMessageEvent(msg, requestIdChan)
	})
	client.OnDisconnect(func(c *sse.Client) {
		logger.Info("Disconnected")
		// wg.Done()
	})

}

type ChangeEvent struct {
	Path string `json:"path"`
	Data string `json:"data"`
}

func processMessageEvent(msg *sse.Event, requestIdChan chan int) {

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
			logger.Debug("Packet Received in Local", "counter", fp.Counter)

			numberStr := strings.TrimPrefix(ce.Path, "/")
			number, err := strconv.Atoi(numberStr)
			if err != nil {
				logger.Error("Received request path is not a number string", "num str", numberStr, "error", err.Error())
				return
			}
			requestIdChan <- number
		}
	}

}
