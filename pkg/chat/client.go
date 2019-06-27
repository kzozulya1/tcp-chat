package chat

import (
	"app/pkg/util"
	"crypto/sha1"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
	"time"
)

//Buffer for store write mesgs, defined in server.go
//const channelBufSize = SERVER_MESSAGES_QUEUE_LEN

//Client order id
var maxId int = 0

// Chat client
type Client struct {
	Id            int
	Conn          net.Conn
	Server        *Server
	Ch            chan *CompleteMessage
	DoneCh        chan bool
	LastActivity  time.Time
	PartMesgBuf   *PartMessageBuffer
	DisconnAction string
	Sha1Identity  string
}

// Create new chat client
func NewClient(conn net.Conn, server *Server) *Client {
	if conn == nil || server == nil {
		panic("Client: invalid params for NewClient routine")
	}
	//Client id
	maxId++

	//Prepare client resources
	ch := make(chan *CompleteMessage, SERVER_MESSAGES_QUEUE_LEN)
	doneCh := make(chan bool)

	client := &Client{maxId, conn, server, ch, doneCh, time.Now(), nil, "", ""}
	client.PartMesgBuf = NewPartMessageBuffer(server)
	return client
}

//Keep client alive
func (c *Client) TouchLastActivity() {
	c.LastActivity = time.Now()
}

//Client write
func (c *Client) Write(msg *CompleteMessage) {
	select {
	case c.Ch <- msg: //Add msg to data channel
	default:
		//27.06.2019 fix

		// c.Server.Del(c)
		// err := fmt.Errorf("Client [%d] %s: was disconnected.", c.Id, c.GetIdentity())
		// c.Server.Err(err)
		fmt.Println("Client's channel can't receive new message.Check buffer")
	}
}

//Close connection
func (c *Client) CloseConnection() {
	c.Conn.Close()
}

// Listen write and read request
func (c *Client) Listen() {
	//Listen for new data in client's data channel and send data to client endpoint; quit if done channel was touched
	go c.listenWrite()
	//Infinite loop for data reading from client endpoint
	c.listenRead()

	//fmt.Println(fmt.Sprintf("Client [%d] %s: gonna close connection", c.Id, c.GetIdentity()))
}

// Listen read requests
func (c *Client) listenRead() {
	//fmt.Println(fmt.Sprintf("Client [%d] %s: ran listenRead loop", c.Id, c.GetIdentity()))
	// Make a buffer to hold incoming data.
	buf := make([]byte, 2048)

	for {
		// Read the incoming connection into the buffer.
		readLen, err := c.Conn.Read(buf)
		if err != nil {
			fmt.Println(fmt.Sprintf("Client [%d] %s: listenRead got error: %s. Send signal to Done ch", c.Id, c.GetIdentity(), err.Error()))

			if strings.Contains(err.Error(), syscall.ECONNRESET.Error()) {
				//Fired when client was disconnected by his initiative
				c.DisconnAction = CLIENT_DISCONN_REASON_LEFT
			} else if c.DisconnAction == "" {
				//Some kind of anoter error
				c.DisconnAction = err.Error()
			}
			//Send signal for listenWrite loop for quit
			c.DoneCh <- true
			return
		}
		//Keep alive routine
		c.TouchLastActivity()

		//Copy new data and add in part buffer
		//fmt.Println(fmt.Sprintf("Client [%d] %s: listenRead read %d bytes", c.Id, c.GetIdentity(),readLen))
		newData := string(buf[:readLen])

		//fmt.Println(fmt.Sprintf("Client [%d] %s: listenRead got new data: %s", c.Id, c.GetIdentity(),newData))

		//Sanitize payload
		msg := util.SkipSpecialChars(string(newData))
		//Add non empty mesg to buffer
		if msg != "" {
			c.PartMesgBuf.Add(msg, c.GetIdentity())
		}
	}
}

//Listen for new data in client's data channel and send data to client endpoint;
//remove client from server list and quit if done channel was touched
func (c *Client) listenWrite() {
	//fmt.Println(fmt.Sprintf("Client [%d] %s: ran listenWrite loop", c.Id, c.GetIdentity()))
	for {
		select {
		// send message to the client
		case msg := <-c.Ch:
			strMsg := fmt.Sprintf("%s", msg.String())
			//fmt.Println(fmt.Sprintf("Client [%d] %s: got new msg in data ch %s", c.Id, c.GetIdentity(),strMsg))
			//Send data to client's endpoint
			c.Conn.Write([]byte(strMsg))

			// receive done request
		case <-c.DoneCh:
			//fmt.Println(fmt.Sprintf("Client [%d] %s: listenWrite loop got signal in Done ch. Quit loop", c.Id, c.GetIdentity()))
			c.Server.Del(c)
			return
		}
	}
}

//Get client SHA1 encrypted IP:PORT
func (c *Client) GetIdentity() string {
	if c.Sha1Identity == "" {
		remoteAddr := c.Conn.RemoteAddr().String()
		remoteAddrParts := strings.Split(remoteAddr, ":")

		h := sha1.New()
		io.WriteString(h, remoteAddrParts[0])
		io.WriteString(h, ":")
		io.WriteString(h, remoteAddrParts[1])
		c.Sha1Identity = fmt.Sprintf("%x", h.Sum(nil))
	}
	return c.Sha1Identity
}
