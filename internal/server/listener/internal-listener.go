package listener

import (
	"fmt"
	"log"
	"net"
	"proxy/internal/server/connection"
	"proxy/internal/server/enum"
	"proxy/internal/server/inter"
	"proxy/internal/server/pack"
	"proxy/pkg/communication"
	"proxy/pkg/key"
	"sync"
)

type InternalListener struct {
	port                                  string
	connections                           map[string]inter.ListenSendWithHeadersConnection
	chanAddConnection                     chan pack.ChanInternalConnection
	chanRemoveConnection                  chan string
	chanCloseExternalConnection           chan<- string
	chanReceivedInternalMessageToExternal chan pack.ChanProxyMessageToExternal
}

func NewInternalListener(
	port string,
	chanMsgToInternal <-chan pack.ChanProxyMessageToInternal,
	chanMsgToExternal chan<- pack.ChanProxyMessageToExternal,
	chanCloseExternalConnection chan<- string,
) *InternalListener {
	l := &InternalListener{
		port:                                  port,
		connections:                           make(map[string]inter.ListenSendWithHeadersConnection),
		chanAddConnection:                     make(chan pack.ChanInternalConnection),
		chanRemoveConnection:                  make(chan string),
		chanReceivedInternalMessageToExternal: make(chan pack.ChanProxyMessageToExternal),
		chanCloseExternalConnection:           chanCloseExternalConnection,
	}
	go func() {
		mu := sync.Mutex{}
		for {
			select {
			case msgToExternal := <-l.chanReceivedInternalMessageToExternal:
				chanMsgToExternal <- msgToExternal
			case sendMessage := <-chanMsgToInternal:
				mu.Lock()
				if con, ok := l.connections[sendMessage.Host]; ok {
					switch sendMessage.Type {
					case enum.MessageExternalToInternalMessageType:
						err := con.Send(sendMessage.ExternalConnectionID, sendMessage.Content)
						if err != nil {
							log.Print(err)
						}
					case enum.CloseConnectionExternalToInternalMessageType:
						headers := communication.BytesHeader{
							key.MessageTypeBytesHeader: key.CloseExternalPersistentConnectionMessageType,
						}
						err := con.SendWithHeaders(sendMessage.ExternalConnectionID, headers, sendMessage.Content)
						if err != nil {
							log.Print(err)
						}
					}
				}
				mu.Unlock()
			case addConnection := <-l.chanAddConnection:
				l.connections[addConnection.Host] = addConnection.Connection
				log.Printf("internal connection: %s added", addConnection.Host)
			case removeConnection := <-l.chanRemoveConnection:
				mu.Lock()
				delete(l.connections, removeConnection)
				log.Printf("internal connection: %s removed", removeConnection)
				mu.Unlock()
			}
			log.Printf("internal connections: %d", len(l.connections))
		}
	}()
	return l
}

func (l *InternalListener) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", l.port))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			con, err := listener.Accept()
			if err != nil {
				log.Fatal(err)
			}
			ec := connection.NewInternalConnection(con, l.chanRemoveConnection, l.chanCloseExternalConnection, l.chanAddConnection, l.chanReceivedInternalMessageToExternal)
			go ec.Listen()
		}
	}()
}
