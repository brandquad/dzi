package main

//
//import (
//	"encoding/json"
//	"github.com/brandquad/dzi"
//	"log"
//)
//
//type worker1payload struct {
//	AssetId       int    `json:"asset_id"`
//	Url           string `json:"url"`
//	Debug         bool   `json:"debug"`
//	SplitChannels bool   `json:"split_channels"`
//	Resolution    int    `json:"resolution"`
//}
//
//type worker1body struct {
//	AssetId int    `json:"asset_id"`
//	Url     string `json:"url"`
//	Debug   bool   `json:"debug"`
//}
//
//func main() {
//	ch, err := dzi.RabbitMQConn.Channel()
//	dzi.OrPanic(err)
//	defer ch.Close()
//
//	q, err := ch.QueueDeclare(
//		"w-01", // name
//		false,  // durable
//		false,  // delete when unused
//		false,  // exclusive
//		false,  // no-wait
//		nil,    // arguments
//	)
//	dzi.OrPanic(err)
//
//	msgs, err := ch.Consume(
//		q.Name, // queue
//		"",     // consumer
//		true,   // auto-ack
//		false,  // exclusive
//		false,  // no-local
//		false,  // no-wait
//		nil,    // args
//	)
//
//	dzi.OrPanic(err)
//	var forever chan struct{}
//
//	go func() {
//		for d := range msgs {
//			var payload worker1payload
//			dzi.OrPanic(json.Unmarshal(d.Body, &payload))
//			log.Printf(" [>] url %s with asset id %d  (debug %t)", payload.Url, payload.AssetId, payload.Debug)
//		}
//	}()
//
//	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
//	<-forever
//}
