package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleutil"
	_ "layeh.com/gumble/opus"
)

var (
	// Build vars
	version string
	commit  string
	date    string
)

func main() {
	fmt.Println("Mumble-Discord-Bridge")
	fmt.Println("v" + version + " " + commit + " " + date)

	godotenv.Load()

	mumbleAddr := flag.String("mumble-address", lookupEnvOrString("MUMBLE_ADDRESS", ""), "MUMBLE_ADDRESS, mumble server address, example example.com, required")
	mumblePort := flag.Int("mumble-port", lookupEnvOrInt("MUMBLE_PORT", 64738), "MUMBLE_PORT, mumble port, (default 64738)")
	mumbleUsername := flag.String("mumble-username", lookupEnvOrString("MUMBLE_USERNAME", "Echo"), "MUMBLE_USERNAME, mumble username, (default: discord)")
	mumblePassword := flag.String("mumble-password", lookupEnvOrString("MUMBLE_PASSWORD", ""), "MUMBLE_PASSWORD, mumble password, optional")
	mumbleInsecure := flag.Bool("mumble-insecure", lookupEnvOrBool("MUMBLE_INSECURE", false), " MUMBLE_INSECURE, mumble insecure, optional")
	mumbleCertificate := flag.String("mumble-certificate", lookupEnvOrString("MUMBLE_CERTIFICATE", ""), "MUMBLE_CERTIFICATE, client certificate to use when connecting to the Mumble server")

	pinger := flag.Bool("pinger", lookupEnvOrBool("PINGER", false), "Send ping message perodically")

	flag.Parse()
	log.Printf("app.config %v\n", getConfig(flag.CommandLine))

	if *mumbleAddr == "" {
		log.Fatalln("missing mumble address")
	}
	if *mumbleUsername == "" {
		log.Fatalln("missing mumble username")
	}

	// MUMBLE SETUP
	MumbleConfig := gumble.NewConfig()
	MumbleConfig.Username = *mumbleUsername
	MumbleConfig.Password = *mumblePassword
	MumbleConfig.AudioInterval = time.Millisecond * 10

	MumbleListener := &MumbleListener{}

	MumbleConfig.Attach(gumbleutil.Listener{
		Connect:     MumbleListener.mumbleConnect,
		UserChange:  MumbleListener.mumbleUserChange,
		TextMessage: MumbleListener.onTextMessage,
	})

	var tlsConfig tls.Config
	if *mumbleInsecure {
		tlsConfig.InsecureSkipVerify = true
	}

	if *mumbleCertificate != "" {
		keyFile := *mumbleCertificate
		if certificate, err := tls.LoadX509KeyPair(keyFile, keyFile); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
			os.Exit(1)
		} else {
			tlsConfig.Certificates = append(tlsConfig.Certificates, certificate)
		}
	}

	MumbleAddr := *mumbleAddr + ":" + strconv.Itoa(*mumblePort)

	log.Println("Attempting to join Mumble")
	MumbleClient, err := gumble.DialWithDialer(new(net.Dialer), MumbleAddr, MumbleConfig, &tlsConfig)
	if err != nil {
		log.Panicln(err)
	}

	if *pinger {
		go func() {
			pingTicker := time.NewTicker(30 * time.Second)
			for {
				<-pingTicker.C
				MumbleClient.Do(func() {
					MumbleClient.Self.Channel.Send("Ping", false)
					pingSentCount++
				})
			}
		}()
	}

	// Monitor Mumble
	// Shutdown on OS signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			if MumbleClient == nil || MumbleClient.State() != 2 {
				if MumbleClient != nil {
					log.Println("Lost mumble connection " + strconv.Itoa(int(MumbleClient.State())))
					return
				} else {
					log.Println("Lost mumble connection due to bridge dieing")
				}
			}
		case <-sc:
			log.Println("OS Signal. Bot shutting down")
			return
		}
	}
}
