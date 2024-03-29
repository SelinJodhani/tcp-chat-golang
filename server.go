package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

type server struct {
	rooms    map[string]*room
	commands chan command
}

func newServer() *server {
	return &server{
		rooms:    make(map[string]*room),
		commands: make(chan command),
	}
}

func (s *server) run() {
	for cmd := range s.commands {
		switch cmd.id {
		case CMD_NICK:
			s.nick(cmd.client, cmd.args)
		case CMD_JOIN:
			s.join(cmd.client, cmd.args)
		case CMD_MSG:
			s.msg(cmd.client, cmd.args)
		case CMD_ROOMS:
			s.listRooms(cmd.client)
		case CMD_QUIT:
			s.quit(cmd.client)
		}
	}
}

func (s *server) nick(c *client, args []string) {
	c.nick = args[1]
	c.msg(fmt.Sprintf("all right, i will call you %s\n", c.nick))
}

func (s *server) join(c *client, args []string) {
	roomName := args[1]

	r, ok := s.rooms[roomName]
	if !ok {
		r = &room{
			name:    roomName,
			members: make(map[net.Addr]*client),
		}
		s.rooms[roomName] = r
	}

	r.members[c.conn.RemoteAddr()] = c

	s.quitCurrentRoom(c)

	c.room = r

	r.broadcast(c, fmt.Sprintf("%s has joined the room\n", c.nick))
	c.msg(fmt.Sprintf("welcome to %s\n", r.name))
}

func (s *server) msg(c *client, args []string) {
	if c.room == nil {
		c.err(fmt.Errorf("you must join room first to communicate"))
	}

	r, ok := s.rooms[c.room.name]

	if !ok {
		c.err(fmt.Errorf("room %s is deleted", r.name))
	}

	r.broadcast(c, fmt.Sprintf("%s: %s", c.nick, strings.Join(args[1:], " ")))
}

func (s *server) listRooms(c *client) {
	var rooms []string

	for name := range s.rooms {
		rooms = append(rooms, name)
	}

	c.msg(fmt.Sprintf("available rooms are: %s\n", strings.Join(rooms, ", ")))
}

func (s *server) quit(c *client) {
	log.Printf("client has disconnected: %s\n", c.conn.RemoteAddr().String())

	c.msg("sad to see you go :(")
	c.conn.Close()

	if c.room != nil {
		delete(c.room.members, c.conn.RemoteAddr())
		c.room.broadcast(c, fmt.Sprintf("%s has left the room!\n", c.nick))
	}
}

func (s *server) quitCurrentRoom(c *client) {
	if c.room != nil {
		delete(c.room.members, c.conn.RemoteAddr())
		c.room.broadcast(c, fmt.Sprintf("%s has left the room!\n", c.nick))
	}
}

func (s *server) newClient(conn net.Conn) {
	log.Printf("new client has connection: %s\n", conn.RemoteAddr().String())

	c := &client{
		conn:     conn,
		nick:     "anonymous",
		commands: s.commands,
	}

	c.readInput()
}
