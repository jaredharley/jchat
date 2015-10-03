package main

import (
	"fmt"
	"net"
	"container/list"
	"bytes"
)

type Client struct {
	Name string
	Incoming chan string
	Outgoing chan string
	Conn net.Conn
	Quit chan bool
	ClientList *list.List
}

// Reads
func (c *Client) Read(buffer []byte) bool {
	bytesRead, err := c.Conn.Read(buffer)
	if err != nil {
		c.Close()
		Log(err)
		return false
	}
	Log("Read ", bytesRead, " bytes")
	return true
}

// Closes client connection
func (c *Client) Close() {
	c.Quit <- true
	c.Conn.Close()
	c.RemoveMe()
}

// Equal compares clients
func (c *Client) Equal(other *Client) bool {
	if bytes.Equal([]byte(c.Name), []byte(other.Name)) {
		if c.Conn == other.Conn {
			return true
		}
	}
	return false
}

// Removes a client from the list
func (c *Client) RemoveMe() {
	for entry := c.ClientList.Front(); entry != nil; entry = entry.Next() {
		client := entry.Value.(Client)
		if c.Equal(&client) {
			Log("RemoveMe: ", c.Name)
			c.ClientList.Remove(entry)
		}
	}
}

// Logging function
func Log(v ...interface{}) {
	fmt.Println(v...)
}

// IOHandler
func IOHandler(Incoming <- chan	string, clientList *list.List) {
	for {
		Log("IOHandler: Waiting for input.")
		input := <-Incoming
		Log("IOHandler: Handling ", input)
		for e := clientList.Front(); e != nil; e = e.Next() {
			client := e.Value.(Client)
			client.Incoming <-input
		}
	}
}

// ClientReader
func ClientReader (client *Client) {
	buffer := make([]byte, 2048)

	for client.Read(buffer) {
		if bytes.Equal(buffer, []byte("/quit")) {
			client.Close()
			break
		}

		Log("ClientReader: received ", client.Name, "> ", string(buffer))
		send := client.Name+"> " + string(buffer)
		client.Outgoing <- send
		for i := 0; i < 2048; i++ {
			buffer[i] = 0x00
		}
	}

	client.Outgoing <- client.Name + " has left the chat"
	Log("ClientReader stopped for ", client.Name)
}

// ClientSender
func ClientSender(client *Client) {
	for {
		select {
		case buffer := <- client.Incoming:
			Log("ClientSender sending ", string(buffer), " to ", client.Name)
			count := 0
			for i := 0; i < len(buffer); i++ {
				if buffer[i] == 0x00 {
					break
				}
				count++
			}
			Log("Send size: ", count)
			client.Conn.Write([]byte(buffer)[0:count])
		case <- client.Quit:
			Log("Client ", client.Name, " quitting")
			client.Conn.Close()
			break
		}
	}
}

// ClientHandler
func ClientHandler(conn net.Conn, ch chan string, clientList *list.List) {
	buffer := make([]byte, 1024)
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		Log("Client connection error: ", err)
	}

	name := string(buffer[0:bytesRead])
	newClient := &Client{name, make(chan string), ch, conn, make(chan bool), clientList}

	go ClientSender(newClient)
	go ClientReader(newClient)

	clientList.PushBack(*newClient)
	ch <-string(name + " has joined the chat.")
}

// Main
func main() {
	Log("Hello server!")

	clientList := list.New()
	in := make(chan string)
	go IOHandler(in, clientList)

	service := ":9988"
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	if err != nil {
		Log("ERROR: Could not resolve address")
	} else {
		netListen, err := net.Listen(tcpAddr.Network(), tcpAddr.String())
		if err != nil {
			Log("ERROR", err)
		} else {
			defer netListen.Close()

			for {
				Log("Waiting for clients")
				connection, err := netListen.Accept()
				if err != nil {
					Log("Client error: ", err)
				} else {
					go ClientHandler(connection, in, clientList)
				}
			}
		}
	}
}