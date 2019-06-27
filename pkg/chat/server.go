package chat

import (
	"fmt"
	"net"
	"os"
	"time"
)

const (
	CLIENT_DISCONN_REASON_LEFT    = "left"
	CLIENT_DISCONN_REASON_TIMEOUT = "timed out"
	CLIENT_CONN_REASON_JOIN       = "joined"
	CLIENT_TIME_OUT               = 60 //in seconds
	SERVER_MESSAGES_QUEUE_LEN     = 10
)

// Chat server
type Server struct {
	Messages  []*CompleteMessage
	Clients   map[int]*Client
	AddCh     chan *Client
	DelCh     chan *Client
	SendAllCh chan *CompleteMessage
	DoneCh    chan bool
	ErrCh     chan error
}

// Create new chat server.
func NewServer() *Server {
	return &Server{
		make([]*CompleteMessage, 0, 15), // messages, 0 len and 15 cap, we use only 10 messages
		make(map[int]*Client),           // clients
		make(chan *Client),              // addCh
		make(chan *Client),              // delCh
		make(chan *CompleteMessage),     // sendAllCh
		make(chan bool),                 // doneCh
		make(chan error),                // errCh
	}
}

//Implement OnFillOutUpdater interface: broadcast new complete mesg
func (s *Server) FillOutUpdate(cMesg *CompleteMessage) {
	s.SendMesgToAllCh(cMesg)
}

//Add new client to client list
func (s *Server) Add(c *Client) {
	s.AddCh <- c
}

//Remove client from client list
func (s *Server) Del(c *Client) {
	s.DelCh <- c
}

//Send mesg to all ch
func (s *Server) SendMesgToAllCh(msg *CompleteMessage) {
	s.SendAllCh <- msg
}

//Server job is done
func (s *Server) Done() {
	s.DoneCh <- true
}

//Error handling
func (s *Server) Err(err error) {
	s.ErrCh <- err
}

//Send last MESSAGES_QUEUE_LEN messages for new clients
func (s *Server) sendPastMessages(c *Client) {
	//fmt.Println(fmt.Sprintf("Server: send past messages to client [%d] %s",c.Id,c.GetIdentity()))
	s.truncMessages()

	for _, msg := range s.Messages {
		//fmt.Println("Server: send past: to client [",c.Id,"]->>>>",k, msg)
		//Write to client's buffered channel. Size of buffer is 11
		//len(s.Messages) shouldn't be > that 10 otherwise client block would be blocked
		c.Write(msg)
	}
}

//Send a mesg to all clients
func (s *Server) sendAll(msg *CompleteMessage) {
	for _, c := range s.Clients {
		//fmt.Println(fmt.Sprintf("Server sendALL: send %s to client [%d] %s",msg,c.Id,c.GetIdentity()))
		c.Write(msg)
	}
}

//Handle PART event: when client is disconnected: TCP conn is closed gracefully or by timeout
//or when joined to chat
func (s *Server) handleClientEvent(identity, action string) {
	//fmt.Println(fmt.Sprintf("Server: called handleClientDisconnEvent with client %s",identity))
	body := fmt.Sprintf("Client %s has %s", identity, action)
	cm := &CompleteMessage{body, "**SERVER**", time.Now()}

	//Nofity all conn clients about event
	s.SendMesgToAllCh(cm)
}

//Handle new accept client connection: create new client and listen for his activity
func (s *Server) handleRequest(conn net.Conn) {
	client := NewClient(conn, s)
	s.Add(client)
	client.Listen()
}

//Find timed out connections and close them
func (s *Server) findAndKickTimedOutClients() {
	for {
		select {
		case <-s.DoneCh:
			//fmt.Println("Server: got signal in Done channel in findAndKickTimedOutClients")
			return

		default:
			for key, c := range s.Clients {
				if time.Now().After(c.LastActivity.Add(CLIENT_TIME_OUT * time.Second)) {
					//fmt.Println(fmt.Sprintf("Server: client ident %s with Id %d is staled!",c.GetIdentity(),c.Id))
					//Closing connection causes listenRead client's loop to drop client's job with timeout error

					//User direct access to  s.Clients[key], becuase c is JUST A COPY
					s.Clients[key].DisconnAction = CLIENT_DISCONN_REASON_TIMEOUT
					c.CloseConnection()
				}
			}
		}
		//Scan every second
		time.Sleep(time.Second)
	}
}

////Leave only last SERVER_MESSAGES_QUEUE_LEN-1 items in messages and join event at last
func (s *Server) truncMessages() {
	if len(s.Messages) >= SERVER_MESSAGES_QUEUE_LEN {
		s.Messages = s.Messages[(len(s.Messages)-SERVER_MESSAGES_QUEUE_LEN)+1:]
	}
}

//Check all channels for events
func (s *Server) loopOverChannels() {
	for {
		select {
		// Add new client
		case c := <-s.AddCh:
			//fmt.Println(fmt.Sprintf("Server: added new client ident %s with Id %d", c.GetIdentity(), c.Id))
			s.Clients[c.Id] = c
			
			//Send last 10 mesg to clients
			s.sendPastMessages(c)
			//And add new complete message about join event and broadcast it
			go s.handleClientEvent(c.GetIdentity(),CLIENT_CONN_REASON_JOIN)
			

		// Del a client
		case c := <-s.DelCh:
			//fmt.Println(fmt.Sprintf("Server: identity %s with Id %d removed from client list", c.GetIdentity(), c.Id))
			//Fire Part Event, client left chat
			
			//Broadcast client was disconnected
			go s.handleClientEvent(c.GetIdentity(), c.DisconnAction)
			
			//Remove client from list
			delete(s.Clients, c.Id)

		// Broadcast message for all clients
		case msg := <-s.SendAllCh:
			//Add new message to queue
			s.Messages = append(s.Messages, msg)
			//Leave only last 10 items in messages
			s.truncMessages()
			//Send msg to all connected clients
			s.sendAll(msg)

		case err := <-s.ErrCh:
			fmt.Println("Server: got error:", err.Error())

		case <-s.DoneCh:
			fmt.Println("Server: got signal in Done channel")
			return
		}
	}
}

// Listen and accept clients
func (s *Server) Listen(port string) {
	// Listen for incoming connections.
	l, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println("Server: error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Server: listening on ", port)

	//Sniff all channel data
	go s.loopOverChannels()

	//Remove timed-out clients
	go s.findAndKickTimedOutClients()

	//Neverending loop waits for new connections
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Server: error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle new connection
		go s.handleRequest(conn)
	}
}
