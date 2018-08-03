package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
)

func main() {
	// Set up channel on which to send signal notifications.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	// Create an MQTT Client.
	cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})

	// Terminate the Client.
	defer cli.Terminate()

	// Connect to the MQTT Server.
	err := cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  "172.31.0.51:1883",
		ClientID: []byte("phone-client"),
	})
	if err != nil {
		panic(err)
	}
	/*
		// Subscribe to topics.
		err = cli.Subscribe(&client.SubscribeOptions{
			SubReqs: []*client.SubReq{
				&client.SubReq{
					TopicFilter: []byte("foo"),
					QoS:         mqtt.QoS0,
					// Define the processing of the message handler.
					Handler: func(topicName, message []byte) {
						fmt.Println(string(topicName), string(message))
					},
				},
				&client.SubReq{
					TopicFilter: []byte("bar/#"),
					QoS:         mqtt.QoS1,
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

	// Publish a message.
	err = cli.Publish(&client.PublishOptions{
		QoS:       mqtt.QoS1,
		TopicName: []byte("home-assistant/phone/callerid"),
		Message:   []byte("name:Unavailable, time:010203, number:1234567890"),
	})
	if err != nil {
		panic(err)
	}

	/*
		// Unsubscribe from topics.
		err = cli.Unsubscribe(&client.UnsubscribeOptions{
			TopicFilters: [][]byte{
				[]byte("foo"),
			},
		})
		if err != nil {
			panic(err)
		}
	*/
	/*
		// Wait for receiving a signal.
		<-sigc

		// Disconnect the Network Connection.
		if err := cli.Disconnect(); err != nil {
			panic(err)
		}
	*/
}
