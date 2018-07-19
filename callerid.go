package main

import (
	/*	"flag" */
	"fmt"
	"log"
	"strings"
	/*	"unicode" */
	"github.com/tarm/serial"
	/*	"net" */ /*"os" */)

/*
 * About Caller ID format
 *
 * http://melabs.com/resources/callerid.htm
 *
 * SDMF not shown here since we don't receive it from our provider
 *
 * MDMF - Multiple Data Message Format
 *
 *  Message Type: 0x80 is MDMF
 *  Length of data
 *  [repeat]
 *    Data Type [1: date and time, 2: phone number, 4: number not present, 7: name, 8: name not present
 *    Length of data
 *    data
 *  Checksum
 *
 *
 * And info about MDMF types came from here:
 * http://www.holtek.com.tw/documents/10179/116745/an0053e.pdf
 *
 * 0x01 - Time
 * 0x02 - Calling Line Identification
 * 0x03 - Reserved (for Dialable Directory Number (DN))
 * 0x04 - Reason for absence of DN
 * 0x05 - Reserved (for Reason for Redirection)
 * 0x06 - Call Qualifier
 * 0x07 - Name
 * 0x08 - Reason for absence of name
 * 0x0B - Message waiting notification
 */

/*
type CIDinfo struct {
	Name string
}
*/

func checksum_valid(msg_type byte, msg_data []byte, msg_len byte, msg_csum byte) bool {
	var sum byte
	sum = msg_type
	sum = sum + msg_len
	var j byte
	for j = 0; j < msg_len; j++ {
		sum = sum + msg_data[j]
	}

	if (sum + msg_csum) == 0 {
		return true
	} else {
		return false
	}
}

func parse_MDMF(msg_data []byte) {
	/* Data Type [1: date and time, 2: phone number, 4: number not present, 7: name, 8: name not present */
	var idx int
	var bleft byte
	var time string
	var id string
	var name string
	var dn_reason string
	var name_reason string

	bleft = byte(len(msg_data))
	for bleft > 2 {

		dtype := msg_data[idx]
		dlen := msg_data[idx+1]
		bleft = bleft - 2
		idx = idx + 2

		switch dtype {
		case 0x01:
			/* Time mmddHHMM*/
			time = string(msg_data[idx : idx+int(dlen)])
			fmt.Printf("Time: %s\n", time)
		case 0x02:
			/* ID */
			id = string(msg_data[idx : idx+int(dlen)])
			fmt.Printf("Number: %s\n", id)
		case 0x03:
			/* Reserved for Dialable DN */
			fmt.Printf("parsed reserved value %02d\n", dtype)
		case 0x04:
			/* Reason for absense of DN */
			/* O = blocked */
			dn_reason = string(msg_data[idx : idx+int(dlen)])
			if strings.Compare(dn_reason, string("O")) != 0 {
				fmt.Printf("dn Reason: %s\n", dn_reason)
			} else {
				fmt.Printf("Number Blocked\n")
			}
		case 0x05:
			/* Reserved for Redirection */
			fmt.Printf("parsed reserved value %02d\n", dtype)
		case 0x06:
			/* Call Qualifier */
			fmt.Printf("parsed call qualifier value %02d\n", dtype)
		case 0x07:
			/* Name */
			name = string(msg_data[idx : idx+int(dlen)])
			fmt.Printf("Name: %s\n", name)
		case 0x08:
			/* Reason for absence of Name */
			/* P = private */
			name_reason = string(msg_data[idx : idx+int(dlen)])
			fmt.Printf("Name Reason: %s\n", name_reason)
		case 0x0B:
			/* Message Waiting */
			fmt.Printf("parsed Message Waiting\n")
		default:
			fmt.Printf("parsed unrecognized value %02d\n", dtype)

		}
		bleft = bleft - dlen
		idx = idx + int(dlen)
	}
	fmt.Printf("========================\n")
}

func main() {
	/*
		var verbose = flag.Bool("v", false, "Enable verbose output")
		flag.Parse()
	*/
	const IDLE = 0
	const SYNC = 1
	const READING_LENGTH = 2
	const READING_DATA = 3
	const READING_CHECKSUM = 4

	const SYNC_NEED = 16
	const SYNC_HOLD = 16

	var inp byte
	var num_sync byte
	var sync_holding byte
	var msg_type byte
	var msg_len byte
	var working_len byte
	msg_data := make([]byte, 256)
	var msg_csum byte

	/* Read the serial data until we receive caller id info */
	c := &serial.Config{Name: "/dev/ttyUSB0", Baud: 1200}
	s, err := serial.OpenPort(c)
	/*	s, err := os.Open("./logdata") */
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	buf := make([]byte, 128)

	state := IDLE
	num_sync = 0
	data_idx := 0
	for {
		n, err := s.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		idx := 0
		for n > 0 {
			/* process the byte depending on state */
			switch state {
			case IDLE:
				inp = buf[idx]
				if inp == 'U' {
					num_sync = num_sync + 1
					if num_sync > SYNC_NEED {
						state = SYNC
						sync_holding = SYNC_HOLD
						num_sync = 0
					}
				} else {
					num_sync = 0
				}
			case SYNC:
				/*				fmt.Printf("Examining: 0x%0X\n", buf[idx]) */
				msg_type = buf[idx]
				if msg_type == 0x80 { /* Our BGW210 router/VOIP interface always sends type 0x80 */
					state = READING_LENGTH
				} else {
					sync_holding = sync_holding - 1
					if sync_holding == 0 {
						state = IDLE
					}
				}
			case READING_LENGTH:
				msg_len = buf[idx]
				if msg_len > 0 {
					working_len = msg_len
					state = READING_DATA
					data_idx = 0
				} else {
					state = IDLE
				}
			case READING_DATA:
				msg_data[data_idx] = buf[idx]
				/*				fmt.Printf("idx = %d, data_idx = %d ", idx, data_idx)
								fmt.Printf("Data = %d (0x%0x)\n", buf[idx], buf[idx]) */
				data_idx = data_idx + 1
				working_len = working_len - 1
				if working_len == 0 {
					state = READING_CHECKSUM
				}
			case READING_CHECKSUM:
				msg_csum = buf[idx]
				/* fmt.Printf("Checksum = %d (0x%0x)\n", msg_csum, msg_csum) */
				if checksum_valid(msg_type, msg_data, msg_len, msg_csum) {
					/* do something */
					/*					fmt.Printf("CSUM valid, Type = %d, Length = %d (0x%0x)\n", msg_type, msg_len, msg_len) */
					parse_MDMF(msg_data[:msg_len])
					/* fmt.Printf("%v\n", msg_data) */
				} else {
					/* fmt.Printf("CSUM NOT valid\n") */
				}
				state = IDLE
			default:
				fmt.Printf("Invalid State, back to IDLE\n")
				state = IDLE
			}
			n = n - 1
			idx = idx + 1
		}
	}
}
