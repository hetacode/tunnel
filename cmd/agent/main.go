package main

import (
	"flag"
	"log"
	"proxy/internal/agent"
	"proxy/pkg/enum"
)

var (
	serverAddress      *string
	hostnameListener   *string
	destinationAddress *string
)

func main() {
	serverAddress = flag.String("server", "localhost:5050", "Server address")
	hostnameListener = flag.String("hostname", "proxy.local", "Hostname use as recognizer of given flow")
	destinationAddress = flag.String("destination", "http://localhost:4321", "Address of where exists web application")
	connectionType := flag.String("type", "http", "Choose connection type: http|ws")
	flag.Parse()

	agent := agent.NewAgent(*serverAddress, *hostnameListener, *destinationAddress, enum.AgentConnectionType(*connectionType))
	log.Println("agent is starting")
	agent.Start()

	log.Println("agent is stopped")
}
