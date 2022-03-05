package connection

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"proxy/internal/server/pack"
	"strings"

	"github.com/hashicorp/go-uuid"
)

type ExternalConnection struct {
	ID                   string
	connection           net.Conn
	chanRemoveConnection chan<- string
	chanMsgToInternal    chan<- pack.ChanProxyMessageToInternal
}

func NewExternalConnection(con net.Conn, chanRemoveConnection chan<- string, chanMsgToInternal chan<- pack.ChanProxyMessageToInternal) *ExternalConnection {
	id, _ := uuid.GenerateUUID()
	c := &ExternalConnection{
		ID:                   id,
		connection:           con,
		chanRemoveConnection: chanRemoveConnection,
		chanMsgToInternal:    chanMsgToInternal,
	}
	return c
}

func (c *ExternalConnection) Listen() {
	log.Printf("New connection: %s", c.connection.RemoteAddr().String())

	msgBytes := make([]byte, 0)
	for {
		b := make([]byte, 1024)
		bl, err := c.connection.Read(b)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		log.Printf("Recv bytes: %d", bl)
		msgBytes = append(msgBytes, b[:bl]...)
		if bl < len(b) {
			break
		}
	}

	br := bufio.NewReader(bytes.NewReader(msgBytes))
	httpRequest, err := http.ReadRequest(br)
	if err != nil {
		c.chanRemoveConnection <- c.ID
		log.Fatal(err)
	}
	hostArr := strings.Split(httpRequest.Host, ":") // <hostname>:<port>
	log.Printf("HOST: %s", hostArr[0])
	c.chanMsgToInternal <- pack.ChanProxyMessageToInternal{
		Host:    hostArr[0],
		Content: msgBytes,
	}
	log.Printf("End receiving")

	responseMessage := `
HTTP/1.1 200 OK
Date: Sun, 10 Oct 2010 23:26:07 GMT
Server: Proxy Server 
Last-Modified: Sun, 26 Sep 2010 22:04:35 GMT
ETag: "45b6-834-49130cc1182c0"
Accept-Ranges: bytes
Content-Length: 12
Connection: close
Content-Type: text/html

OK!`
	_, err = c.connection.Write([]byte(responseMessage))
	if err != nil {
		c.chanRemoveConnection <- c.ID
		log.Fatal(err)
	}
	err = c.connection.Close()
	if err != nil {
		c.chanRemoveConnection <- c.ID
		log.Fatal(err)
	}
	c.chanRemoveConnection <- c.ID
}
