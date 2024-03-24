package main

import (
	"bytes"
	"encoding/base64"
	"firetunnel/config"
	"firetunnel/modal"
	slg "firetunnel/util/slg"
	"firetunnel/worker"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	slgOpts = slg.Options{
		Ansi:  true,
		Level: slog.LevelDebug,
	}
	simplelogHandler = slg.CreateHandler(os.Stdout, "main", &config.SlgOpts)
	logger           = slog.New(simplelogHandler)
)

var peerurl = ".com/peer/%v.json"   // data will sent
var localurl = ".com/local/%v.json" //data will come

var maxRequest = 10
var maxByteRead = 1024 * 10000

func main() {
	logger.Info("Application Started")
	mode, localport, _ := worker.GetArgs()

	if mode == "local" {

		startLocalListner(localport)
	}

	if mode == "peer" {
		//listen on the peer url to read request from peer
		//reqId := make(chan int)

		///go peerworker.FireListener(masterpeerurl, reqId)

		wg := sync.WaitGroup{}
		wg.Add(1)
		for i := 0; i < 10; i++ {
			go startPeerDialer2(localport, i)
		}
		wg.Wait()
	}

}
func startDialer(localport int) (pcon *net.Conn, err error) {
	pconn, err := net.Dial("tcp", ":"+strconv.Itoa(int(localport)))
	if err != nil {
		return nil, err
	}
	return &pconn, nil
}

func startPeerDialer2(localport int, requestId int) {

	// Loop body

	logger.Info("Request Dialer Created", "request", requestId)
	var localToPeer = make(chan []byte) // put rest
	var peerToLocal = make(chan []byte) //get sse

	reqPerPeerUrl := fmt.Sprintf(peerurl, requestId)
	wg := sync.WaitGroup{}
	go worker.FireListener(reqPerPeerUrl, peerToLocal)

	for {
		// Dial to the local port

		closesignal := make(chan string)
		packetCounter := 0
		//request submiter
		wg.Add(1)
		go func(cs chan string, wgp *sync.WaitGroup) {
		reqsub_loop:
			for {
				select {
				case data := <-peerToLocal: //data will from fire listener
					{
						logger.Info("Connectioned Dialed...", "Request", requestId)
						conn, conErr := net.Dial("tcp", ":"+strconv.Itoa(localport))

						if conErr != nil {
							// Handle error
							logger.Error("Error dialing:", conErr)
							time.Sleep(time.Second * 2) // Wait before trying again
							// return
							return
						}

						go PeerDataForward(conn, localToPeer, "local -> peer", requestId, closesignal)

						_, err := conn.Write(data)
						if err != nil {
							// Handle error
							logger.Error("Error writing to Dialed Conn:", err)
							//clear the peer
							//sendRequest(peerurl, []byte(""))
							break reqsub_loop
						}
						logger.Debug("<--- Packet Written for Dialed Conn", "Request", requestId)
						//start the read routine
					}

				case data := <-localToPeer:
					{

						packetCounter += 1
						fp := modal.FirePacket{Data: data, Counter: packetCounter, Request: requestId}
						bytedata, err := fp.GetBytePacket()
						if err != nil {
							logger.Error("Error while sending the packet to peer")
							return
						}
						reqPerLocalUrl := fmt.Sprintf(localurl, requestId)
						sendRequest(reqPerLocalUrl, bytedata)
						logger.Debug("---> Packet Sent to Peer", "Request", requestId, "counter", fp.Counter)
					}

				case <-cs:
					{
						logger.Error("Close Signal received")
						//break reqsub_loop
					}

				}
			}
			logger.Debug("Req Sub Loop Closed", "Request", requestId)
			wgp.Done()
		}(closesignal, &wg)

		//

		wg.Wait()
		logger.Info("Connection closed", "Request", requestId)
	}

}

func PeerDataForward(src net.Conn, ch chan []byte, direction string, requestId int, cs chan string) {

reader_loop:
	for {
		// var buf bytes.Buffer
		data := make([]byte, 1024)

		// var buf bytes.Buffer

		n, err := src.Read(data)

		logger.Debug("Packet Length Read From Dialer", "request", requestId, "Packet Size", len(data))
		if err != nil {
			if err == io.EOF {
				logger.Error("Connection closed while Reading,", "direction", direction, "request", requestId)
			} else {
				logger.Error("Error reading from", "request", requestId, direction, err.Error())
			}
			//src.Close()
			cs <- "close"
			break reader_loop
		}
		if n > 0 {
			ch <- data[:n]
		}
	}
	logger.Debug("Reader Loop Closed", "request", requestId)
}

// func startPeerDialer(localport int, localToPeer chan []byte, peerToLocal chan []byte) {
// 	//start the dialer
// 	conn, err := startDialer(localport)
// 	if err != nil {
// 		logger.Error("Error creating a dialer to", "port", localport, "error", err.Error())
// 		return
// 	}
// 	logger.Info("Diater for port started", "port", localport)
// 	var packetCounter = 0
//
// 	go forwardData(conn, localToPeer, "local -> peer")
//
// 	for {
// 		select {
// 		case data := <-localToPeer:
// 			//prepare the data
// 			packetCounter += 1
// 			fp := modal.FirePacket{Data: data, Counter: packetCounter}
// 			bytedata, err := fp.GetBytePacket()
// 			logger.Debug("Packet Sent to Peer", "counter", fp.Counter)
// 			if err != nil {
// 				logger.Error("Error while sending the packet to peer")
// 				return
// 			}
// 			sendRequest(localurl, bytedata)

// 		case data := <-peerToLocal:
// 			logger.Info("Packet received peer -> local")
// 			for {

// 				_, err := (*conn).Write(data)
// 				if err != nil {
// 					logger.Error("Error writing to client:", "error", err.Error())
// 					logger.Info("Starting dialer again...")
// 					newconn, err := startDialer(localport)
// 					if err != nil {
// 						logger.Error("Error creating a dialer Again to", "port", localport, "error", err.Error())

// 					}
// 					conn = newconn
// 					continue

// 				}
// 				break
// 			}
// 		}
// 	}

// 	//port in the request will daled
// 	//send the data form local to peer

// }

func startLocalListner(localport int) {

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(int(localport)))
	if err != nil {
		fmt.Println("Error creating listener:", err)
		return
	}
	defer listener.Close()

	logger.Info("Listening on ", "port", localport)

	peerToLocalChanArray := make([]chan []byte, maxRequest)
	idelWorkerGroup := make(chan int, maxRequest)
	for i := range peerToLocalChanArray {
		// fill idel with zero, 1 mean taken
		idelWorkerGroup <- i
		peerToLocalChanArray[i] = make(chan []byte)
		reqPerLocalUrl := fmt.Sprintf(localurl, i)
		go worker.FireListener(reqPerLocalUrl, peerToLocalChanArray[i])
	}

	for {
		// Accept an incoming connection from a client
		conn, err := listener.Accept()
		//find the worker with free with 0

		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		iw := <-idelWorkerGroup
		logger.Info("idel worker selced", "idel", iw)
		// Handle the connection in a separate goroutine
		go processClient(conn, iw, peerToLocalChanArray[iw], idelWorkerGroup)
	}
}

func processClient(conn net.Conn, requestCounter int, peerToLocal chan []byte, iwkg chan int) {
	logger.Info("Request Received", "counter", requestCounter)
	defer conn.Close()
	//empty
	var localToPeer = make(chan []byte) // put rest
	// var peerToLocal = make(chan []byte) //get sse

	// Start goroutines for forwarding data
	go forwardData(conn, localToPeer, requestCounter, "local -> peer") //reqest

	// go forwardData(conn, serverToClient, "Server -> Client") //response
	var packetCounter = 0

outter:
	for {
		select {
		case data := <-localToPeer:
			//prepare the data
			packetCounter += 1
			fp := modal.FirePacket{Data: data, Counter: packetCounter, Request: requestCounter}
			bytedata, err := fp.GetBytePacket()
			if err != nil {
				logger.Info("Error while sending the packet to peer")
				break outter
			}
			reqPerPeerUrl := fmt.Sprintf(peerurl, requestCounter)
			sendRequest(reqPerPeerUrl, bytedata)
			logger.Debug("---> Packet Sent to Peer", "Request", requestCounter, "counter", fp.Counter)

		case data := <-peerToLocal:

			_, err := conn.Write(data)
			if err != nil {
				fmt.Println("Error writing to local client:", "Request", requestCounter, "Err", err)
				break outter
			}
			logger.Debug("<--- Packet Received From Peer", "Request", requestCounter)
		}

	}
	logger.Warn("Request Processor closed", "Request", requestCounter)
	iwkg <- requestCounter

}

func sendRequest(url string, buffer []byte) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer([]byte("\""+base64.StdEncoding.EncodeToString(buffer)+"\"")))
	if err != nil {
		return
	}
	http.DefaultClient.Do(req)
	// resBody, err := io.ReadAll(res.Body)
	// if err != nil {

	// }
	// fmt.Printf("client: response body: %s\n", resBody)
}

func forwardData(src net.Conn, ch chan []byte, requestCounter int, direction string) {
	logger.Debug("Forward Loop started for", "request", requestCounter)
	for {

		data := make([]byte, 1024)

		// var buf bytes.Buffer

		n, err := src.Read(data)
		if err != nil {
			if err == io.EOF {
				logger.Error("Connection closed while Reading,", "request", requestCounter, "direction", direction)
			} else {
				logger.Error("Error reading from", "request", requestCounter, direction, err.Error())
			}
			break
		}
		if n > 0 {
			logger.Debug("Packet Length for Local to Peer", "request", requestCounter, "Packet Size", len(data), "n", n)
			ch <- data[:n]
		}
	}
	logger.Debug("Forward Loop Closed", "request", requestCounter)

}

// func doSOme() {
// 	// Create a TCP listener on port 5000
// 	listener, err := net.Listen("tcp", ":5000")
// 	if err != nil {
// 		fmt.Println("Error creating listener:", err)
// 		return
// 	}
// 	defer listener.Close()

// 	fmt.Println("Listening on :5000...")

// 	for {
// 		// Accept an incoming connection from a client
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			fmt.Println("Error accepting connection:", err)
// 			continue
// 		}

// 		// Handle the connection in a separate goroutine
// 		// go processClient(conn)
// 	}
// }

// // Encode the byte array to a base64 string
// encodedString := base64.StdEncoding.EncodeToString(data)
// fmt.Println("Encoded:", encodedString)

// // Decode the base64 string back to a byte array
// decodedData, err := base64.StdEncoding.DecodeString(encodedString)
// fmt.Println("Decoded:", decodedData)
