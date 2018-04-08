package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"gopkg.in/cheggaaa/pb.v1"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <payload.dol>\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	var address = flag.String("address", "", "Address of the GameCube that is running ethloader (must be specified if using -nodiscover)")
	var nodiscovery = flag.Bool("nodiscovery", false, "Disable service discovery")
	var bufferSize = flag.Uint("buffer", 1024, "Size of package buffer, recommended <= 1024")

	// Read in configuration
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "No payload specified")
		flag.Usage()
		os.Exit(1)
	}

	if *address == "" {
		if *nodiscovery {
			fmt.Fprintln(os.Stderr, "No target address specified")
			flag.Usage()
			os.Exit(1)
		}
		*address = lookupProbe()
	}

	payload := flag.Arg(0)

	// Read DOL into memory
	data, err := ioutil.ReadFile(payload)
	checkErr(err, "Error reading payload")

	// Add the default port on the address if not specified
	if strings.Index(*address, ":") < 0 {
		*address += ":8856"
	}

	// Create connection with ethloader on GameCube
	conn, err := net.Dial("tcp4", *address)
	checkErr(err, "Error dialing address %s", *address)
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
		checkErr(err, "Error trasferring payload")
		bar.Add(written)
		totalwritten += int64(written)

		conn.Read(ackdata)

		if written == 0 {
			break send
		}
	}

	bar.FinishPrint("File transferred")
}

func lookupProbe() string {
	// Search for gamecube using service discovery probe
	serverConn, err := net.ListenMulticastUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(239, 1, 9, 14),
		Port: 8890,
	})
	checkErr(err, "Error listening for probe")

	serverConn.SetReadBuffer(16)

	fmt.Println("Looking for probe...")
	buf := make([]byte, 16)
	n, addr, err := serverConn.ReadFromUDP(buf)
	checkErr(err, "Error getting probe")
	fmt.Println("Received ", string(buf[0:n]), " from ", addr)

	return addr.String()
}

func checkErr(err error, msg string, fmtargs ...interface{}) {
	if err != nil {
		fmt.Fprintf(os.Stderr, msg+":\n    ", fmtargs...)
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
