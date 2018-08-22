package main

import (
	"flag"
	"fmt"
	"github.com/brian-armstrong/gpio"
	"github.com/tarm/serial"
	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
	"log"
	/*	"net" */
	"os"
	"os/signal"
	"strings"
	"time"
)

const ringPin = 17

type Cinfo struct {
	Name   string
	Number string
	Time   string
}

func checksum_valid(msg_data []byte) bool {
	msg_len := msg_data[1]
	var sum byte
	var j byte

	for j = 0; j < msg_len+3; j++ {
		sum = sum + msg_data[j]
	}

	if (sum) == 0 {
		return true
	} else {
		return false
	}
}

func parse_MDMF(msg_data []byte, callinfo *Cinfo, verbose bool) {
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
			callinfo.Time = time
			if verbose {
				fmt.Printf("Time: %s\n", time)
			}
		case 0x02:
			/* ID */
			id = string(msg_data[idx : idx+int(dlen)])
			callinfo.Number = id
			if verbose {
				fmt.Printf("Number: %s\n", id)
			}
		case 0x03:
			/* Reserved for Dialable DN */
			if verbose {
				fmt.Printf("parsed reserved value %02d\n", dtype)
			}
		case 0x04:
			/* Reason for absense of DN */
			/* O = blocked */
			dn_reason = string(msg_data[idx : idx+int(dlen)])
			callinfo.Number = dn_reason
			if verbose {
				if strings.Compare(dn_reason, string("O")) != 0 {
					fmt.Printf("dn Reason: %s\n", dn_reason)
				} else {
					fmt.Printf("Number Blocked\n")
				}
			}
		case 0x05:
			/* Reserved for Redirection */
			if verbose {
				fmt.Printf("parsed reserved value %02d\n", dtype)
			}
		case 0x06:
			/* Call Qualifier */
			if verbose {
				fmt.Printf("parsed call qualifier value %02d\n", dtype)
			}
		case 0x07:
			/* Name */
			name = string(msg_data[idx : idx+int(dlen)])
			callinfo.Name = name
			if verbose {
				fmt.Printf("Name: %s\n", name)
			}
		case 0x08:
			/* Reason for absence of Name */
			/* P = private */
			name_reason = string(msg_data[idx : idx+int(dlen)])
			callinfo.Name = name_reason
			if verbose {
				fmt.Printf("Name Reason: %s\n", name_reason)
			}
		case 0x0B:
			/* Message Waiting */
			if verbose {
				fmt.Printf("parsed Message Waiting\n")
			}
		default:
			if verbose {
				fmt.Printf("parsed unrecognized value %02d\n", dtype)
			}

		}
		bleft = bleft - dlen
		idx = idx + int(dlen)
	}
	if verbose {
		fmt.Printf("========================\n")
	}
}

func main() {

	var ipaddress_string string
	var serialport_string string
	var verbose = flag.Bool("v", false, "Enable verbose output")
	flag.StringVar(&ipaddress_string, "ip", "172.31.0.51", "ipv4 address of the mqtt server")
	flag.StringVar(&serialport_string, "port", "/dev/ttyAMA0", "path for the serial port device")

	flag.Parse()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	cli := client.New(&client.Options{
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})

	defer cli.Terminate()

	ipstr := ipaddress_string + ":1883"
	if *verbose {
		fmt.Printf("MQTT server addr: %s\n", ipstr)
	}

	// Connect to the MQTT Server.
	err := cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  ipstr,
		ClientID: []byte("callerid-client"),
	})
	if err != nil {
		panic(err)
	}

	if *verbose {
		fmt.Printf("Serial device path: %s\n", serialport_string)
	}
	c := &serial.Config{Name: serialport_string, Baud: 1200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	go func() {
		/* Listen to the serial port and parse caller ID data */
		const IDLE = 0
		const SYNC = 1
		const READING_LENGTH = 2
		const READING_DATA = 3
		const READING_CHECKSUM = 4

		const SYNC_NEED = 16
		const SYNC_HOLD = 16

		var callinfo Cinfo
		var inp byte
		var num_sync byte
		var sync_holding byte
		var msg_type byte
		var msg_len byte
		var working_len byte
		buf := make([]byte, 128)
		msg_data := make([]byte, 300)

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
					msg_type = buf[idx]
					if msg_type == 0x80 { /* Our BGW210 router/VOIP interface always sends type 0x80 */
						msg_data[data_idx] = buf[idx]
						data_idx = data_idx + 1
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
						msg_data[data_idx] = buf[idx]
						data_idx = data_idx + 1
						state = READING_DATA
					} else {
						state = IDLE
						data_idx = 0
					}
				case READING_DATA:
					msg_data[data_idx] = buf[idx]
					data_idx = data_idx + 1
					working_len = working_len - 1
					if working_len == 0 {
						state = READING_CHECKSUM
					}
				case READING_CHECKSUM:
					msg_data[data_idx] = buf[idx]
					data_idx = data_idx + 1
					if checksum_valid(msg_data) {
						/* do something */
						parse_MDMF(msg_data[2:msg_len+2], &callinfo, *verbose)
						textinfo := fmt.Sprintf("name:%s, time:%s, number:%s", callinfo.Name, callinfo.Time, callinfo.Number)
						err = cli.Publish(&client.PublishOptions{
							QoS:       mqtt.QoS1,
							TopicName: []byte("home-assistant/phone/callerid"),
							Message:   []byte(textinfo),
						})
						if err != nil {
							log.Fatal(err)
						}
					} else {
						if *verbose {
							fmt.Printf("CSUM NOT valid\n")
						}
					}
					state = IDLE
					data_idx = 0
				default:
					if *verbose {
						fmt.Printf("Invalid State, back to IDLE\n")
					}
					state = IDLE
					data_idx = 0
				}
				n = n - 1
				idx = idx + 1
			}
		}
	}()

	go func() {
		/* Monitor the Ring indicator GPIO pin and publish to the MQTT topic
				 * Ring indicator is active low on ringPin
				 * U.S. ring cadence is 2 seconds on and 4 off
		         *
		         * We periodically get repeated indications that the Value is 1 even when it hasn't changed
		*/
		var ringing bool
		var notify gpio.WatcherNotification
		tmr := time.NewTimer(10)
		tmr.Stop()
		watcher := gpio.NewWatcher()
		watcher.AddPin(ringPin)
		defer watcher.Close()

		ringing = false
		for {
			select {
			case notify = <-watcher.Notification:
				if notify.Pin == ringPin {
					if notify.Value == 0 {
						/* Ring Start */
						if !ringing {
							ringing = true
							/* This is the beginning of a set of rings */
							/* Send a topic notification that we're ringing */
							textinfo := fmt.Sprintf("Yes")
							err = cli.Publish(&client.PublishOptions{
								QoS:       mqtt.QoS1,
								TopicName: []byte("home-assistant/phone/ringing"),
								Message:   []byte(textinfo),
							})
							if err != nil {
								log.Fatal(err)
							}
						}
						tmr = time.NewTimer(7 * time.Second)
					}
				}
			case <-tmr.C:
				ringing = false
				// Send a topic notification that we're NOT ringing
				textinfo := fmt.Sprintf("No")
				err = cli.Publish(&client.PublishOptions{
					QoS:       mqtt.QoS1,
					TopicName: []byte("home-assistant/phone/ringing"),
					Message:   []byte(textinfo),
				})
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		/*
		 */
	}()

	go func() {
		/* Subscribe to the control topic and control the relay */
		/* relay outputs are active low on GPIO 22, 23, 24, 25 */
		/*
			err = cli.Subscribe(&client.SubscribeOptions{
				SubReqs: []*client.SubReq{
					&client.SubReq{
						TopicFilter: []byte("home-assistant/phone/mode"),
						QoS:         mqtt.QoS0,
						// Define the processing of the message handler.
						Handler: func(topicName, message []byte) {
							fmt.Println(string(topicName), string(message))
						},
					},
				},
			})
			if err != nil {
				panic(err)
			}
		*/
		for {
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	for {
		// hang out here and look for signals and mailbox messages
		// Wait for receiving a signal.
		<-sigc

		// Disconnect the Network Connection.
		if err := cli.Disconnect(); err != nil {
			panic(err)
		}
		// Unsubscribe from topics.
		err = cli.Unsubscribe(&client.UnsubscribeOptions{
			TopicFilters: [][]byte{
				[]byte("home-assistant/phone/mode"),
			},
		})
		if err != nil {
			panic(err)
		}

	}
}
