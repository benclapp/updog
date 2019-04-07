package main

import (
	"os"
	"time"

	"github.com/streadway/amqp"
)

var rabbitClients = rabbitClientsList{}

func initRabbit() {
	for _, rabbit := range config.Dependencies.RabbitMQ {
		key, err := os.Hostname()
		if err != nil {
			logger.Log("msg", "failure getting hostname", "error", err)
		}

		conn, err := amqp.Dial(rabbit.DSN)
		if err != nil {
			logRabbitErr("Failed on dial", err)
		}

		defer func() {
			if err := conn.Close(); err != nil {
				logRabbitErr("RabbitMQ health check failed to close connection", err)
			}
			logger.Log("dependency", "RabbitMQ", "msg", "Close rabbitmq connection")
		}()

		channel, err := conn.Channel()
		if err != nil {
			logRabbitErr("Failed getting channel", err)
		}

		defer func() {
			if err := channel.Close(); err != nil {
				logRabbitErr("Failed closing channel", err)
			}
			logger.Log("dependency", "RabbitMQ", "msg", "Closing Channel")
		}()

		if err := channel.ExchangeDeclare("UpdogExchange", "topic", true, false, false, false, nil); err != nil {
			logRabbitErr("Failed declaring exchange", err)
		}

		if _, err := channel.QueueDeclare("UpdogQueue", false, false, false, false, nil); err != nil {
			logRabbitErr("Failed declaring queue", err)
		}

		if err := channel.QueueBind("UpdogQueue", key, "UpdogExchange", false, nil); err != nil {
			logRabbitErr("Failed during binding", err)
		}

		rabbitClients = append(
			rabbitClients,
			rabbitClient{
				Name:    rabbit.Name,
				Channel: channel,
				Key:     key,
			},
		)
	}
}

func checkRabbit(bunny rabbitClient) {
	logger.Log("dependency", "RabbitMQ", "msg", "Starting to check dependency")
	start := time.Now()

	messages, err := bunny.Channel.Consume("UpdogQueue", "", true, false, false, false, nil)
	if err != nil {
		logRabbitErr("Failed while consuming", err)
	}

	fin := make(chan struct{})

	go func() {
		<-messages
		logger.Log("dependency", "RabbitMQ", "time", time.Since(start).Seconds(), "msg", "message received")
		close(fin)
		for range messages {
		}
	}()

	msg := amqp.Publishing{Body: []byte(time.Now().Format(time.RFC3339Nano))}
	if err := bunny.Channel.Publish("UpdogExchange", bunny.Key, false, false, msg); err != nil {
		logRabbitErr("Fail while publishing", err)
	}

	for {
		select {
		case <-time.After(depTimeout):
			logger.Log("dependency", "RabbitMQ", "msg", "Timed out waiting for message")
			return
		case <-fin:
			return
		}
	}
}

func logRabbitErr(msg string, err error) {
	logger.Log("dependency", "RabbitMQ", "msg", msg, "err", err)
}

type rabbitResult struct {
	Name     string  `json:"name"`
	Success  bool    `json:"success"`
	Duration float64 `json:"duration"`
	Err      error   `json:"error,omitempty"`
}

type rabbitClientsList []rabbitClient

type rabbitClient struct {
	Name    string
	Channel *amqp.Channel
	Key     string
}
