package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gowon-irc/go-gowon"
	"github.com/jessevdk/go-flags"
	bolt "go.etcd.io/bbolt"
)

type Options struct {
	Prefix string `short:"P" long:"prefix" env:"GOWON_PREFIX" default:"." description:"prefix for commands"`
	Broker string `short:"b" long:"broker" env:"GOWON_BROKER" default:"localhost:1883" description:"mqtt broker"`
	APIKey string `short:"k" long:"api-key" env:"GOWON_TRAKT_API_KEY" required:"true" description:"trakt api key"`
	KVPath string `short:"K" long:"kv-path" env:"GOWON_TRAKT_KV_PATH" default:"kv.db" description:"path to kv db"`
}

const (
	moduleName               = "trakt"
	mqttConnectRetryInternal = 5
	mqttDisconnectTimeout    = 1000
)

func setUser(kv *bolt.DB, nick, user []byte) error {
	err := kv.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("trakt"))
		return b.Put([]byte(nick), []byte(user))
	})
	return err
}

func getUser(kv *bolt.DB, nick []byte) (user []byte, err error) {
	err = kv.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("trakt"))
		v := b.Get([]byte(nick))
		user = v
		return nil
	})
	return user, err
}

func genTraktHandler(apiKey string, kv *bolt.DB) func(m gowon.Message) (string, error) {
	return func(m gowon.Message) (string, error) {
		fields := strings.Fields(m.Args)

		if len(fields) >= 2 && fields[0] == "set" {
			err := setUser(kv, []byte(m.Nick), []byte(fields[1]))
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("set %s's user to %s", m.Nick, fields[1]), nil
		}

		if len(fields) >= 1 {
			user := strings.Fields(m.Args)[0]
			return trakt(user, apiKey)
		}

		user, err := getUser(kv, []byte(m.Nick))
		if err != nil {
			return "", err
		}

		if len(user) > 0 {
			return trakt(string(user), apiKey)
		}

		return "Error: username needed", nil
	}
}

func defaultPublishHandler(c mqtt.Client, msg mqtt.Message) {
	log.Printf("unexpected message:  %s\n", msg)
}

func onConnectionLostHandler(c mqtt.Client, err error) {
	log.Println("connection to broker lost")
}

func onRecconnectingHandler(c mqtt.Client, opts *mqtt.ClientOptions) {
	log.Println("attempting to reconnect to broker")
}

func onConnectHandler(c mqtt.Client) {
	log.Println("connected to broker")
}

func main() {
	log.Printf("%s starting\n", moduleName)

	opts := Options{}
	if _, err := flags.Parse(&opts); err != nil {
		log.Fatal(err)
	}

	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(fmt.Sprintf("tcp://%s", opts.Broker))
	mqttOpts.SetClientID(fmt.Sprintf("gowon_%s", moduleName))
	mqttOpts.SetConnectRetry(true)
	mqttOpts.SetConnectRetryInterval(mqttConnectRetryInternal * time.Second)
	mqttOpts.SetAutoReconnect(true)

	mqttOpts.DefaultPublishHandler = defaultPublishHandler
	mqttOpts.OnConnectionLost = onConnectionLostHandler
	mqttOpts.OnReconnecting = onRecconnectingHandler
	mqttOpts.OnConnect = onConnectHandler

	kv, err := bolt.Open(opts.KVPath, 0666, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer kv.Close()

	err = kv.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("trakt"))
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	mr := gowon.NewMessageRouter()
	mr.AddCommand("trakt", genTraktHandler(opts.APIKey, kv))
	mr.Subscribe(mqttOpts, moduleName)

	log.Print("connecting to broker")

	c := mqtt.NewClient(mqttOpts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	log.Print("connected to broker")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	log.Println("signal caught, exiting")
	c.Disconnect(mqttDisconnectTimeout)
	log.Println("shutdown complete")
}
