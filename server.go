package main

import (
  "github.com/gorilla/websocket" 
  "github.com/codegangsta/martini"
  "net"
  "net/http"
  "log"
)

var ActiveClients = make(map[ClientConn]int)

type ClientConn struct {
   websocket *websocket.Conn
   clientIP  net.Addr
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
  m.Get("/sock", func (w http.ResponseWriter, r *http.Request) {
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
    ActiveClients[sockCli] = 0 

    for {
      messageType, p, err := ws.ReadMessage()
      if err != nil {
        return
      }
      for client, _ := range ActiveClients {
        if err := client.websocket.WriteMessage(messageType, p); err != nil {
          return
        }
      }
    }
  })
  m.Run()
}
