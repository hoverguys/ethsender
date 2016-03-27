package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"gopkg.in/cheggaaa/pb.v1"
)

var address = flag.String("address", "", "Adress of the GameCube that is running ethloader")
var payload = flag.String("payload", "", "Payload to send")
var bufferSize = flag.Uint("buffer", 1024, "Size of package buffer, recommended <= 1024")

func main() {
	// Read in configuration
	flag.Parse()
	if *address == "" {
		fmt.Println("Please provide a target address: ip:port")
		return
	}

	if *payload == "" {
		fmt.Println("Please provide a payload: ./path/to/file.dol")
		return
	}

	// Read DOL into memory
	data, err := ioutil.ReadFile(*payload)
	if err != nil {
		fmt.Println("Error reading payload")
		return
	}

	// Create connection with ethloader on GameCube
	conn, err := net.Dial("tcp4", *address)
	if err != nil {
		fmt.Printf("Error dailing address %s\n%s\n", *address, err)
		return
	}
	defer conn.Close()

	// Get file size
	filesize := int64(len(data))

	// Send length of incoming payload
	lengthdata := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthdata, uint32(filesize))
	_, err = conn.Write(lengthdata)
	if err != nil {
		fmt.Printf("Error write size %s\n", err)
	}

	sizedata := make([]byte, 4)
	binary.BigEndian.PutUint32(sizedata, uint32(*bufferSize))
	_, err = conn.Write(sizedata)
	if err != nil {
		fmt.Printf("Error buffer size %s\n", err)
	}

	// Setup progressbars
	bar := pb.New(int(filesize))
	bar.SetUnits(pb.U_BYTES)
	bar.Prefix(fmt.Sprintf("%d bytes", *bufferSize))
	bar.Start()

	// Send payload
	ackdata := make([]byte, 4)
	totalwritten := int64(0)
send:
	for {
		limit := totalwritten + int64(*bufferSize)
		if limit >= filesize {
			limit = filesize
		}
		written, err := conn.Write(data[totalwritten:limit])
		if err != nil {
			log.Printf("Error transferring %s\n", err)
			return
		}
		bar.Add(written)
		totalwritten += int64(written)

		conn.Read(ackdata)

		if written == 0 {
			break send
		}
	}

	bar.FinishPrint("File transferred")
}
