package main

import (
	/*"flag" */
	"github.com/tarm/serial"
	"log"
	"os"
)

func main() {
	/*
		var verbose = flag.Bool("v", false, "Enable verbose output")
		flag.Parse()
	*/
	/* Read the serial data until we receive caller id info */
	c := &serial.Config{Name: "/dev/ttyUSB0", Baud: 1200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Printf("serial open error\n")
		log.Fatal(err)
	}
	buf := make([]byte, 128)

	f, err := os.Create("./logdata")
	if err != nil {
		log.Printf("create error\n")
		log.Fatal(err)
	}
	defer f.Close()

	for {
		n, err := s.Read(buf)
		if err != nil {
			log.Printf("read error\n")
			log.Fatal(err)
		}

		n2, err := f.Write(buf[:n])
		if err != nil {
			log.Printf("write error\n")
			log.Fatal(err)
		}
		if n2 != n {
			log.Printf("length mismatch\n")
			log.Fatal(err)
		}
	}
}
