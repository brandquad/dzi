package dzi

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitSettings struct {
	Host string `envconfig:"RABBITMQ_HOST" required:"true" default:"localhost"`
	Port string `envconfig:"RABBITMQ_PORT" required:"true" default:"5672"`
	User string `envconfig:"RABBITMQ_USER" required:"true" default:"admin"`
	Pass string `envconfig:"RABBITMQ_PASS" required:"true" default:"admin"`
}

var RabbitMQConn *amqp.Connection

func init() {
	var c rabbitSettings
	OrPanic(envconfig.Process("", &c))

	_c, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", c.User, c.Pass, c.Host, c.Port))
	OrPanic(err)

	RabbitMQConn = _c
}

func disconnect() {
	RabbitMQConn.Close()
}
