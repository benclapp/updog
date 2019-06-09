package main

import (
	"fmt"
	"os"
	"time"

	"github.com/streadway/amqp"
)

// var rabbitClients = rabbitClientsList{}
var hostname = ""

func initRabbit() {
	hn, err := os.Hostname()
	if err != nil {
		logger.Log("msg", "failure getting hostname", "error", err)
	}
	hostname = hn

	for _, bunny := range config.Dependencies.RabbitMQ {
		logger.Log("dependency_type", "rabbitmq", "dependency_name", string(bunny.Name), "db_type", "RabbitMQ")

		healthCheckDependencyDuration.WithLabelValues(bunny.Name, "RabbitMQ").Observe(0)
		healthChecksTotal.WithLabelValues(bunny.Name, "RabbitMQ").Add(0)
		healthChecksFailuresTotal.WithLabelValues(bunny.Name, "RabbitMQ").Add(0)
	}
}

func checkRabbit(dsn, name string, ch chan<- rabbitResult) {
	start := time.Now()
	err := check(dsn, name)
	elapsed := time.Since(start).Seconds()

	healthCheckDependencyDuration.WithLabelValues(name, "RabbitMQ").Observe(elapsed)
	healthChecksTotal.WithLabelValues(name, "RabbitMQ").Inc()

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "dependency", name, "err", err)
		healthChecksFailuresTotal.WithLabelValues(name, "RabbitMQ").Inc()
		ch <- rabbitResult{Name: name, Success: false, Duration: elapsed, Err: err}
	} else {
		ch <- rabbitResult{Name: name, Success: true, Duration: elapsed}
	}
}

func check(dsn, name string) error {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		logRabbitErr("Failed on dial", err)
		return err
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
		return err
	}

	defer func() {
		if err := channel.Close(); err != nil {
			logRabbitErr("Failed closing channel", err)
		}
		logger.Log("dependency", "RabbitMQ", "msg", "Closing Channel")
	}()

	if err := channel.ExchangeDeclare("UpdogExchange", "topic", true, false, false, false, nil); err != nil {
		logRabbitErr("Failed declaring exchange", err)
		return err
	}

	if _, err := channel.QueueDeclare("UpdogQueue", false, false, false, false, nil); err != nil {
		logRabbitErr("Failed declaring queue", err)
		return err
	}

	if err := channel.QueueBind("UpdogQueue", hostname, "UpdogExchange", false, nil); err != nil {
		logRabbitErr("Failed during binding", err)
		return err
	}

	messages, err := channel.Consume("UpdogQueue", "", true, false, false, false, nil)
	if err != nil {
		logRabbitErr("Failed while consuming", err)
		return err
	}

	fin := make(chan struct{})

	go func() {
		<-messages
		close(fin)
		for range messages {
		}
	}()

	msg := amqp.Publishing{Body: []byte(time.Now().Format(time.RFC3339Nano))}
	if err := channel.Publish("UpdogExchange", hostname, false, false, msg); err != nil {
		logRabbitErr("Fail while publishing", err)
		return err
	}

	for {
		select {
		case <-time.After(depTimeout):
			logger.Log("dependency", "RabbitMQ", "msg", "Timed out waiting for message")
			return fmt.Errorf("Timed out while waiting for message")
		case <-fin:
			return nil
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
