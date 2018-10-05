package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func (p *Player) Run(t *testing.T) {
	go func() {
		for {
			select {
			case message := <-p.send:
				fmt.Sprint(message)
			}
		}
	}()
}

func (p *Player) ReceiveMessage(t *testing.T, event string, action string) bool {
	msg := &Message{}
	json.Unmarshal(<-p.send, msg)

	if msg.Event != event {
		t.Errorf("Player receive wrong message event, {id: %d, rcv: %s, must: %s}", p.Id(), msg.Event, event)
		return false
	}

	if msg.Action != action {
		t.Errorf("Player receive wrong message action, {id: %d, rcv: %s, must: %s}", p.Id(), msg.Action, action)
		return false
	}

	return true
}

type EventChecker struct {
	Players       []*Player
	T             *testing.T
	Event         string
	ActionSend    string
	ActionReceive string
	Data          interface{}
}

func (e *EventChecker) Check() {
	for _, player := range e.Players {
		if !player.ReceiveMessage(e.T, e.Event, e.ActionSend) {
			return
		}
	}

	for _, player := range e.Players {
		msg := &Message{
			Event:  e.Event,
			Action: e.ActionReceive,
			Data:   e.Data,
		}

		for {

			if len(player.send) == 0 {
				break
			}

			<-player.send
		}

		player.OnMessage(msg)

		for _, rcvPlayer := range e.Players {
			for {

				if len(rcvPlayer.send) == 0 {
					break
				}

				<-rcvPlayer.send
			}
		}
	}

	time.Sleep(5 * time.Millisecond)
}

func TestGameCreate(t *testing.T) {

	player := NewPlayer()
	player.Run(t)

	msg := &Message{
		Event:     EVENT_GAME,
		Action:    ACTION_CREATE,
		Iteration: 0,
		Data:      map[string]interface{}{"username": "anton"},
	}

	player.OnMessage(msg)

	if !player.Master() {
		t.Errorf("Player is not a master")
		return
	}

	if player.Game() == nil {
		t.Errorf("Player has not game")
		return
	}
}

func TestGameJoin(t *testing.T) {
	game := NewGame()
	game.Run()

	Games[game.Id] = game

	player := NewPlayer()
	player.SetName("anton")
	player.SetMaster(true)
	player.Run(t)

	game.Players.Add(player)

	player2 := NewPlayer()
	player2.Run(t)

	msg := &Message{
		Event:     EVENT_GAME,
		Action:    ACTION_JOIN,
		Iteration: 0,
		Data:      map[string]interface{}{"username": "anton2", "game": float64(game.Id)},
	}

	player2.OnMessage(msg)
}

func TestAcceptEvent(t *testing.T) {
	game := NewGame()
	game.Event = NewAcceptEvent(game.Iteration, EVENT_GREET_CITIZENS, ACTION_END)
	game.Run()
	Games[game.Id] = game

	mafia := NewPlayer()
	mafia.Run(t)
	mafia.SetGame(game)
	mafia.SetRole(ROLE_MAFIA)
	game.Players.Add(mafia)

	citizen := NewPlayer()
	citizen.Run(t)
	citizen.SetGame(game)
	citizen.SetRole(ROLE_CITIZEN)
	game.Players.Add(citizen)

	doctor := NewPlayer()
	doctor.Run(t)
	doctor.SetGame(game)
	doctor.SetRole(ROLE_DOCTOR)
	game.Players.Add(doctor)

	girl := NewPlayer()
	girl.Run(t)
	girl.SetGame(game)
	girl.SetRole(ROLE_GIRL)
	game.Players.Add(girl)

	sheriff := NewPlayer()
	sheriff.Run(t)
	sheriff.SetGame(game)
	sheriff.SetRole(ROLE_SHERIFF)
	game.Players.Add(sheriff)

	msg := NewEventMessage(game.Event, ACTION_END)
	mafia.OnMessage(msg)
	citizen.OnMessage(msg)
	doctor.OnMessage(msg)
	girl.OnMessage(msg)
	sheriff.OnMessage(msg)

	time.Sleep(10 * time.Millisecond)

	if game.Event.Name() != EVENT_NIGHT {
		t.Errorf("Game has wrong event")
	}
}

func TestMafiaResultEvent(t *testing.T) {
	game := NewGame()
	game.Iteration = 2
	game.Event = NewMafiaEvent(game.Iteration)
	game.Run()
	Games[game.Id] = game

	mafia := NewPlayer()
	mafia.Run(t)
	mafia.SetGame(game)
	mafia.SetRole(ROLE_MAFIA)
	game.Players.Add(mafia)

	citizen := NewPlayer()
	citizen.Run(t)
	citizen.SetGame(game)
	citizen.SetRole(ROLE_CITIZEN)
	game.Players.Add(citizen)

	msg := NewEventMessage(game.Event, ACTION_VOTE)
	msg.Data = float64(citizen.Id())
	mafia.OnMessage(msg)

	time.Sleep(5 * time.Millisecond)
	if game.Event.Name() != EVENT_DAY {
		t.Errorf("Game has wrong event: %s, must be: %s, iteration: %d", game.Event.Name(), EVENT_DAY, game.Event.Iteration())
		return
	}

	game.Event.SetStatus(PROCESSED)
	time.Sleep(5 * time.Millisecond)

	if game.Event.Name() != EVENT_NIGHT_RESULT {
		t.Errorf("Game has wrong event: %s, must be: %s, iteration: %d", game.Event.Name(), EVENT_NIGHT_RESULT, game.Event.Iteration())
		return
	}

	if !citizen.Out() {
		t.Errorf("night result is wrong")
		return
	}
}

func TestGameEventLoopFirstLoop(t *testing.T) {
	game := NewGame()
	game.Run()
	game.Event = NewGameEvent()

	for i := 0; i < 10; i++ {
		player := NewPlayer()
		player.Run(t)
		player.SetGame(game)
		game.Players.Add(player)
	}

	events := []string{
		EVENT_GAME_START,     //start
		EVENT_GREET_CITIZENS, //start
		EVENT_GREET_CITIZENS,
		EVENT_GREET_CITIZENS, //end
		EVENT_NIGHT,
		EVENT_GREET_MAFIA, //start
		EVENT_GREET_MAFIA,
		EVENT_GREET_MAFIA, //end
		EVENT_DAY,
		EVENT_COURT, //start
		EVENT_COURT,
		EVENT_COURT_RESULT,
		EVENT_COURT, //end
		EVENT_NIGHT,
		EVENT_MAFIA, //start
		EVENT_MAFIA,
		EVENT_MAFIA, //end
	}

	for _, eventName := range events {
		game.Event.SetStatus(PROCESSED)
		time.Sleep(2 * time.Millisecond)
		t.Logf("Check %s, current %s, iteration %d", eventName, game.Event.Name(), game.Event.Iteration())
		if game.Event.Name() != eventName {
			t.Errorf("Event has wrong name %s", game.Event.Name())
			return
		}
	}
}

func TestGameEventLoopSecondIteration(t *testing.T) {

	game := NewGame()
	game.Iteration = 2
	game.Run()
	game.Event = NewGameEvent()

	mafia := NewPlayer()
	mafia.Run(t)
	mafia.SetGame(game)
	mafia.SetRole(ROLE_MAFIA)
	game.Players.Add(mafia)

	citizen := NewPlayer()
	citizen.Run(t)
	citizen.SetGame(game)
	citizen.SetRole(ROLE_CITIZEN)
	game.Players.Add(citizen)

	doctor := NewPlayer()
	doctor.Run(t)
	doctor.SetGame(game)
	doctor.SetRole(ROLE_DOCTOR)
	game.Players.Add(doctor)

	girl := NewPlayer()
	girl.Run(t)
	girl.SetGame(game)
	girl.SetRole(ROLE_GIRL)
	game.Players.Add(girl)

	sheriff := NewPlayer()
	sheriff.Run(t)
	sheriff.SetGame(game)
	sheriff.SetRole(ROLE_SHERIFF)
	game.Players.Add(sheriff)

	events := []string{
		EVENT_MAFIA, //start
		EVENT_MAFIA,
		EVENT_MAFIA,  //end
		EVENT_DOCTOR, //start
		EVENT_DOCTOR,
		EVENT_DOCTOR,  //end
		EVENT_SHERIFF, //start
		EVENT_SHERIFF,
		//EVENT_SHERIFF_RESULT, sheriff has not choice
		EVENT_SHERIFF, //end
		EVENT_GIRL,    //start
		EVENT_GIRL,
		EVENT_GIRL, //end
		EVENT_DAY,
		EVENT_NIGHT_RESULT,
		EVENT_COURT, //start
		EVENT_COURT,
		EVENT_COURT_RESULT,
		EVENT_COURT, //end
		EVENT_NIGHT, //start
	}

	game.Event = NewAcceptEvent(game.Iteration, EVENT_NIGHT, ACTION_ACCEPT)
	for _, eventName := range events {
		game.Event.SetStatus(PROCESSED)
		time.Sleep(10 * time.Millisecond)
		t.Logf("Check: %s, current: %s, iteration: %d", eventName, game.Event.Name(), game.Event.Iteration())
		if game.Event.Name() != eventName {
			t.Errorf("Event has wrong name, check %s, current: %s", eventName, game.Event.Name())
			return
		}
	}
}

func TestGameEvents(t *testing.T) {
	playerMaster := NewPlayer()

	msg := &Message{
		Event:  EVENT_GAME,
		Action: ACTION_CREATE,
		Data:   map[string]interface{}{"username": strconv.Itoa(playerMaster.Id())},
	}
	playerMaster.OnMessage(msg)

	if !playerMaster.ReceiveMessage(t, EVENT_GAME, ACTION_CREATE) {
		return
	}
	if !playerMaster.ReceiveMessage(t, EVENT_GAME, ACTION_PLAYERS) {
		return
	}

	players := playerMaster.Game().Players

	if playerMaster.Game() == nil {
		t.Errorf("Player has no game")
		return
	}

	for i := 0; i < 100; i++ {
		player := NewPlayer()
		msg := &Message{
			Event:  EVENT_GAME,
			Action: ACTION_JOIN,
			Data:   map[string]interface{}{"username": strconv.Itoa(player.Id()), "game": float64(playerMaster.Game().Id)},
		}
		player.OnMessage(msg)
		if !player.ReceiveMessage(t, EVENT_GAME, ACTION_JOIN) {
			return
		}
		for _, innerPlayer := range players.FindAll() {
			if !innerPlayer.ReceiveMessage(t, EVENT_GAME, ACTION_PLAYERS) {
				return
			}
		}
	}

	msg = &Message{
		Event:  EVENT_GAME,
		Action: ACTION_START,
	}
	playerMaster.OnMessage(msg)

	time.Sleep(5 * time.Millisecond)

	if playerMaster.Game().Event.Name() != EVENT_GAME_START {
		t.Errorf("Invalid event: %s", playerMaster.Game().Event.Name())
		return
	}

	ch := &EventChecker{}
	ch.T = t

	ch.Players = players.FindAll()
	ch.Event = EVENT_GAME_START
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_GREET_CITIZENS
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_GREET_CITIZENS
	ch.ActionSend = ACTION_ROLE
	ch.ActionReceive = ACTION_ACCEPT
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_GREET_CITIZENS
	ch.ActionSend = ACTION_END
	ch.ActionReceive = ACTION_END
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_NIGHT
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_GREET_MAFIA
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindByRole(ROLE_MAFIA)
	ch.Event = EVENT_GREET_MAFIA
	ch.ActionSend = ACTION_PLAYERS
	ch.ActionReceive = ACTION_ACCEPT
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_GREET_MAFIA
	ch.ActionSend = ACTION_END
	ch.ActionReceive = ACTION_END
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_DAY
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_COURT
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	candidates := players.FindByRole(ROLE_CITIZEN)
	candidate := candidates[0]

	ch.Players = players.FindAll()
	ch.Event = EVENT_COURT
	ch.ActionSend = ACTION_PLAYERS
	ch.ActionReceive = ACTION_VOTE
	ch.Data = float64(candidate.Id())

	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_COURT_RESULT
	ch.ActionSend = ACTION_OUT
	ch.ActionReceive = ACTION_ACCEPT
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_COURT
	ch.ActionSend = ACTION_END
	ch.ActionReceive = ACTION_END
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_NIGHT
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_MAFIA
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	candidates = players.FindByRole(ROLE_CITIZEN)
	candidate = candidates[0]

	ch.Players = players.FindByRole(ROLE_MAFIA)
	ch.Event = EVENT_MAFIA
	ch.ActionSend = ACTION_PLAYERS
	ch.ActionReceive = ACTION_VOTE
	ch.Data = float64(candidate.Id())
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_MAFIA
	ch.ActionSend = ACTION_END
	ch.ActionReceive = ACTION_END
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_DOCTOR
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindByRole(ROLE_DOCTOR)
	ch.Event = EVENT_DOCTOR
	ch.ActionSend = ACTION_PLAYERS
	ch.ActionReceive = ACTION_CHOICE
	ch.Data = float64(candidate.Id())
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_DOCTOR
	ch.ActionSend = ACTION_END
	ch.ActionReceive = ACTION_END
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_SHERIFF
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindByRole(ROLE_SHERIFF)
	ch.Event = EVENT_SHERIFF
	ch.ActionSend = ACTION_PLAYERS
	ch.ActionReceive = ACTION_CHOICE
	ch.Data = float64(candidate.Id())
	ch.Check()

	ch.Players = players.FindByRole(ROLE_SHERIFF)
	ch.Event = EVENT_SHERIFF_RESULT
	ch.ActionSend = ACTION_ROLE
	ch.ActionReceive = ACTION_ACCEPT
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_SHERIFF
	ch.ActionSend = ACTION_END
	ch.ActionReceive = ACTION_END
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_GIRL
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindByRole(ROLE_GIRL)
	ch.Event = EVENT_GIRL
	ch.ActionSend = ACTION_PLAYERS
	ch.ActionReceive = ACTION_CHOICE
	ch.Data = float64(candidate.Id())
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_GIRL
	ch.ActionSend = ACTION_END
	ch.ActionReceive = ACTION_END
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_DAY
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = players.FindAll()
	ch.Event = EVENT_NIGHT_RESULT
	ch.ActionSend = ACTION_OUT
	ch.ActionReceive = ACTION_ACCEPT
	ch.Check()
}

func TestGameOver(t *testing.T) {
	game := NewGame()
	game.Iteration = 2

	game.Event = NewMafiaEvent(game.Iteration)

	mafia := NewPlayer()
	mafia.SetGame(game)
	mafia.SetRole(ROLE_MAFIA)
	game.Players.Add(mafia)
	mafia.game = game

	citizen := NewPlayer()
	citizen.SetGame(game)
	citizen.SetRole(ROLE_CITIZEN)
	game.Players.Add(citizen)
	citizen.game = game

	game.Run()

	ch := &EventChecker{}
	ch.T = t

	candidates := game.Players.FindByRole(ROLE_CITIZEN)
	candidate := candidates[0]

	ch.Players = game.Players.FindByRole(ROLE_MAFIA)
	ch.Event = EVENT_MAFIA
	ch.ActionSend = ACTION_PLAYERS
	ch.ActionReceive = ACTION_VOTE
	ch.Data = float64(candidate.Id())
	ch.Check()

	ch.Players = game.Players.FindAll()
	ch.Event = EVENT_DAY
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	ch.Players = game.Players.FindAll()
	ch.Event = EVENT_NIGHT_RESULT
	ch.ActionSend = ACTION_OUT
	ch.ActionReceive = ACTION_ACCEPT
	ch.Check()

	if !game.isOver() {
		t.Errorf("Game is not over")
		return
	}
}

func TestReconnect(t *testing.T) {

	game := NewGame()
	game.Iteration = 2

	game.Event = NewAcceptEvent(game.Iteration, EVENT_DAY, ACTION_START)

	mafia := NewPlayer()
	mafia.SetGame(game)
	mafia.SetRole(ROLE_MAFIA)
	game.Players.Add(mafia)
	mafia.game = game

	citizen := NewPlayer()
	citizen.SetGame(game)
	citizen.SetRole(ROLE_CITIZEN)
	game.Players.Add(citizen)
	citizen.game = game

	Games[game.Id] = game
	game.Run()

	ch := &EventChecker{}
	ch.T = t

	ch.Players = game.Players.FindAll()
	ch.Event = EVENT_DAY
	ch.ActionSend = ACTION_START
	ch.ActionReceive = ACTION_START
	ch.Check()

	newCitizen := NewPlayer()

	msg := &Message{
		Event:  EVENT_GAME,
		Action: ACTION_RECONNECT,
		Data:   map[string]interface{}{"game": float64(game.Id), "player": float64(citizen.Id())},
	}

	newCitizen.OnMessage(msg)

	if !newCitizen.ReceiveMessage(t, EVENT_NIGHT_RESULT, ACTION_OUT) {
		return
	}
}
