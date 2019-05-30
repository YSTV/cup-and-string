package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/dchote/gumble/gumble"
	_ "github.com/dchote/gumble/opus"
	"github.com/dchote/talkiepi"
)

// ConfigFile stores values from config file
type ConfigFile struct {
	Server      string `json:"server"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Insecure    bool   `json:"insecure"`
	Certificate string `json:"certificate"`
	Channel     string `json:"channel"`
}

func main() {
	// Command line flags
	server := flag.String("server", "", "the server to connect to")
	username := flag.String("username", "", "the username of the client")
	password := flag.String("password", "", "the password of the server")
	insecure := flag.Bool("insecure", true, "skip server certificate verification")
	certificate := flag.String("certificate", "", "PEM encoded certificate and private key")
	channel := flag.String("channel", "", "mumble channel to join by default")
	config := flag.String("config", "", "path to config file (takes priority over flags)")

	flag.Parse()

	/**
	 * Config priority:
	 * 	1. Config File
	 *	2. Command line args
	 */

	if *config != "" {
		fmt.Println(*config)
		jsonFile, err := os.Open(*config)
		if err != nil {
			fmt.Println(err)
		}
		defer jsonFile.Close()

		byteValue, err := ioutil.ReadAll(jsonFile)

		var configFile ConfigFile

		json.Unmarshal(byteValue, &configFile)

		*server = configFile.Server
		*username = configFile.Username
		*password = configFile.Password
		*insecure = configFile.Insecure
		*certificate = configFile.Certificate
		*channel = configFile.Channel
	}

	// Initialize
	b := talkiepi.Talkiepi{
		Config:      gumble.NewConfig(),
		Address:     *server,
		ChannelName: *channel,
	}

	// if no username specified, lets just autogen a random one
	if len(*username) == 0 {
		buf := make([]byte, 6)
		_, err := rand.Read(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}

		buf[0] |= 2
		b.Config.Username = fmt.Sprintf("talkiepi-%02x%02x%02x%02x%02x%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
	} else {
		b.Config.Username = *username
	}

	b.Config.Password = *password

	if *insecure {
		b.TLSConfig.InsecureSkipVerify = true
	}
	if *certificate != "" {
		cert, err := tls.LoadX509KeyPair(*certificate, *certificate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		b.TLSConfig.Certificates = append(b.TLSConfig.Certificates, cert)
	}

	b.Init()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	exitStatus := 0

	<-sigs
	b.CleanUp()

	os.Exit(exitStatus)
}
