package main

import (
	"github.com/codegangsta/martini"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/http"
	"sync"
)

/*
  Go is a great langauge but you have to be aware of what happens when you are mutating state from multiple goroutines.
  Everytime a web request comes in, in your /sock handler down below Go will create a new goroutine to service that
  request, this means, you could potentially have a lot of new gourtines concurrently running that are modifiying the
  ActiveClients map defined below this text.  By design, maps are not concurrent-safe, meaning that if they are being
  modified by multiple goroutines concurrently, they can easily get into an invalid state.

  If all the /sock handler was doing, was only reading to but not writing the ActiveClients, then you would be fine.
  But in this script you are also writing to the ActiveClients therefore you must synchronize access to map.

  You can do that one of two ways: Using channels, or using mutexes.  These topics are fairly big topics on their own
  but here is how you can use a mutex to make sure that are safely mutating the state of the map.

  See this page for further reading: http://blog.golang.org/go-maps-in-action

*/

var ActiveClients = make(map[ClientConn]int)
var ActiveClientsRWMutex sync.RWMutex

type ClientConn struct {
	websocket *websocket.Conn
	clientIP  net.Addr
}

func addClient(cc ClientConn) {
	ActiveClientsRWMutex.Lock()
	ActiveClients[cc] = 0
	ActiveClientsRWMutex.Unlock()
}

func deleteClient(cc ClientConn) {
	ActiveClientsRWMutex.Lock()
	delete(ActiveClients, cc)
	ActiveClientsRWMutex.Unlock()
}

func broadcastMessage(messageType int, message []byte) {
	ActiveClientsRWMutex.RLock()
	defer ActiveClientsRWMutex.RUnlock()

	for client, _ := range ActiveClients {
		if err := client.websocket.WriteMessage(messageType, message); err != nil {
			return
		}
	}
}

func main() {
	m := martini.Classic()
	m.Get("/", func() string {
		return `<html><body><script src='//ajax.googleapis.com/ajax/libs/jquery/1.10.2/jquery.min.js'></script>
    <ul id=messages></ul><form><input id=message><input type="submit" id=send value=Send></form>
    <script>
    var c=new WebSocket('ws://localhost:3000/sock');
    c.onopen = function(){
      c.onmessage = function(response){
        console.log(response.data);
        var newMessage = $('<li>').text(response.data);
        $('#messages').append(newMessage);
        $('#message').val('');
      };
      $('form').submit(function(){
        c.send($('#message').val());
        return false;
      });
    }
    </script></body></html>`
	})
	m.Get("/sock", func(w http.ResponseWriter, r *http.Request) {
		log.Println(ActiveClients)
		ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if _, ok := err.(websocket.HandshakeError); ok {
			http.Error(w, "Not a websocket handshake", 400)
			return
		} else if err != nil {
			log.Println(err)
			return
		}
		client := ws.RemoteAddr()
		sockCli := ClientConn{ws, client}
		addClient(sockCli)

		for {
			log.Println(len(ActiveClients), ActiveClients)
			messageType, p, err := ws.ReadMessage()
			if err != nil {
				deleteClient(sockCli)
				log.Println("bye")
				log.Println(err)
				return
			}
			broadcastMessage(messageType, p)
		}
	})
	m.Run()
}
