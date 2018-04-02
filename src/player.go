package main

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"encoding/json"
	"time"
	"math/rand"
	"fmt"
)

const maxMessageSize = 4096 // Maximum message size allowed from peer.

const ROLE_CITIZEN = 1
const ROLE_MAFIA = 2
const ROLE_DOCTOR = 3
const ROLE_GIRL = 4
const ROLE_SHERIFF = 5

const STATUS_OK = 1
const STATUS_ERR = 2

type Message struct {
	Status    int         `json:"status"`
	Iteration int         `json:"iteration"`
	Event     string      `json:"event"`
	Action    string      `json:"action"`
	Data      interface{} `json:"data"`
}

func NewEventMessage(event IEvent, action string) *Message {
	return &Message{
		Status:    STATUS_OK,
		Iteration: event.Iteration(),
		Event:     event.Name(),
		Action:    action,
	}
}

type Player struct {
	id        int
	name      string
	role      int
	game      *Game
	master    bool
	addr      string
	version   string
	device    string
	url       string
	createdAt time.Time
	conn      *websocket.Conn
	out       bool
	send      chan []byte
}

func NewPlayer() *Player {
	player := &Player{
		id:        rand.Intn(999999999),
		createdAt: time.Now(),
		send:      make(chan []byte, 0),
		out:       false,
	}

	return player
}

func (p *Player) SendMessage(message *Message) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Send message id: %d, err: %v", p.Id, err)
		}
	}()

	msg, err := json.Marshal(message)

	if err != nil {
		log.Errorf("Marshal message: id: %d, err: %v", p.Id, err)
		return
	}

	p.send <- msg
}

func (p *Player) SetGame(game *Game) {
	p.game = game
}

func (p *Player) Game() *Game {
	return p.game
}

func (p *Player) Id() int {
	return p.id
}

func (p *Player) SetRole(role int) {
	p.role = role
}

func (p *Player) Role() int {
	return p.role
}

func (p *Player) SetName(name string) {
	p.name = name
}

func (p *Player) Name() string {
	return p.name
}

func (p *Player) SetMaster(master bool) {
	p.master = master
}

func (p *Player) Master() bool {
	return p.master
}

func (p *Player) SetOut(out bool) {
	p.out = out
}

func (p *Player) Out() bool {
	return p.out
}

func (player *Player) SetConnection(conn *websocket.Conn) {
	player.conn = conn
	go player.readLoop()
	go player.writeLoop()
}

func (p *Player) readLoop() {
	log.Debugf("readLoop %s", p.Id())
	defer func() {

		if err := recover(); err != nil {
			log.Errorf("Close readLoop id=%d err=%v", p.Id(), err)
		}

		log.Debugf("Closing on read end: %d", p.Id())
		p.conn.Close()

	}()

	p.conn.SetReadLimit(maxMessageSize)

	for {
		_, message, err := p.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Infof("error: %v", err)
			}
			break
		}

		msg := &Message{}
		err = json.Unmarshal(message, msg)
		if err != nil {
			log.Errorf("error on msg decode {msg:%v, err:%v, id:%s", string(message), err, p.Id())
			break
		}

		log.Debugf("rcv msg %s %#v", p.Id(), msg)

		p.OnMessage(msg)
	}
}

func (p *Player) OnMessage(msg *Message) {
	if p.Game() == nil {

		switch msg.Action {
		case ACTION_CREATE:
			game := NewGame()
			game.Run()
			Games[game.Id] = game
			p.game = game
			p.master = true
			break
		case ACTION_JOIN:
			data := msg.Data.(map[string]interface{})
			gameId := int(data["game"].(float64))

			game, ok := Games[gameId]

			if ok {
				p.game = game
				break
			}

			if !ok {
				rmsg := &Message{
					Status: STATUS_ERR,
					Data:   "invalid gameId",
				}
				p.SendMessage(rmsg)
			}
			break
		default:
			rmsg := &Message{
				Status: STATUS_ERR,
				Data:   "invalid gameId",
			}
			p.SendMessage(rmsg)
			break
		}
	}

	if p.Game() == nil {
		fmt.Errorf("Player has not gameId")
		return
	}

	actions := p.game.Event.Actions()
	if action, ok := actions[msg.Action]; ok && p.Game() != nil {
		err := action(p.game.Players, p.game.EventsHistory, p, msg)
		if err != nil {
			log.Errorf("error on action: %s, id: %d, err: %v", msg.Action, p.Id(), err)
		}
	} else {
		log.Errorf("undefined message type: %s, id: %d", msg.Action, p.Id())
	}
}

func (p *Player) writeLoop() {
	log.Debugf("writePump %s", p.Id)
	defer func() {

		if err := recover(); err != nil {
			log.Errorf("Close writeLoop id: %d, err:%v", p.Id(), err)
		}

		log.Debugf("Closing on write end: %d", p.Id())
		p.conn.Close()
	}()

	for {
		select {
		case message, ok := <-p.send:

			msg := &Message{}
			errUnmarshal := json.Unmarshal(message, msg)
			if errUnmarshal != nil {
				log.Errorf("error on msg unmarshal id: %d, err: %v, msg: %s", p.Id(), errUnmarshal, string(message))
			}

			if !ok {
				p.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := p.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				log.Errorf("send message : error can't get writer connection id: %d, msg: %#v, err: %v", p.Id(), msg, err)
				return
			}

			log.Debugf("snd msg %s %#v", p.Id, msg)

			if _, err = w.Write(message); err != nil {
				log.Infof("send message : error on w.Write() id: %d, msg: %#v, err: %v", p.Id(), msg, err)
				return
			}

			if err := w.Close(); err != nil {
				log.Infof("send message : error on w.Close() writer connection id: %d, msg: %#v, err: %v", p.Id(), msg, err)
				return
			}
		}
	}
}

func (p *Player) CloseConnection() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Close client channel id: %d, err:%v", p.Id(), err)
		}
	}()

	close(p.send)
}

/*
 Players
 */
type Players struct {
	data []*Player
}

func NewPlayers() *Players {
	return &Players{data: make([]*Player, 0)}
}

func (p *Players) FindOneById(id int) *Player {
	for _, player := range p.data {
		if player.Id() == id && !player.Out() {
			return player
		}
	}

	return nil
}

func (p *Players) FindByRole(role int) []*Player {
	players := make([]*Player, 0)
	for _, player := range p.data {
		if player.Role() == role && !player.Out() {
			players = append(players, player)
		}
	}
	return players
}

func (p *Players) FindOneByRole(role int) *Player {
	for _, player := range p.data {
		if player.Role() == role && !player.Out() {
			return player
		}
	}
	return nil
}

func (p *Players) FindOneByUsername(username string) *Player {
	for _, player := range p.data {
		if player.Name() == username && !player.Out() {
			return player
		}
	}
	return nil
}

func (p *Players) FindAll() []*Player {
	players := make([]*Player, 0)
	for _, player := range p.data {
		if !player.Out() {
			players = append(players, player)
		}
	}
	return players
}

func (p *Players) FindAllWithOut() []*Player {
	return p.data
}

func (p *Players) Add(player *Player) {
	p.data = append(p.data, player)
}
