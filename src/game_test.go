package main

import (
	"testing"
	"time"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
)

func loggerInit() {

	f, err := os.OpenFile("/Users/newuser/web/go/src/mafia-backend/log/out.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Open log file error %v\n", err)
		os.Exit(1)
	}
	log.SetFormatter(&LogFormatter{})
	log.SetOutput(f)
	log.SetLevel(log.Level(5))
}

func (p *Player) Run(t *testing.T) {
	go func(){
		for {
			select {
			case message := <-p.send:
				//t.Logf("%s", message)
				fmt.Sprint(message)
			}
		}
	}()
}

func TestGameCreate(t *testing.T) {

	player := NewPlayer()
	player.Run(t)

	msg := &Message{
		Event:EVENT_GAME,
		Action: ACTION_CREATE,
		Iteration: 0,
		Data: map[string]interface{}{"username": "anton"},
	}

	player.OnMessage(msg)

	if !player.Master() {
		t.Errorf("Player is not a master")
	}

	if player.Game() == nil {
		t.Errorf("Player has not game")
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
		Event:EVENT_GAME,
		Action: ACTION_JOIN,
		Iteration: 0,
		Data: map[string]interface{}{"username": "anton2", "game": float64(game.Id)},
	}

	player2.OnMessage(msg)
	time.Sleep(5 * time.Millisecond)

	//t.Logf("%s", <- player2.send)
	//t.Logf("%s", <- player2.send)
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
	msg.Data = citizen.Id()
	mafia.OnMessage(msg)

	time.Sleep(5 * time.Millisecond)
	if game.Event.Name() != EVENT_DAY {
		t.Errorf("Game has wrong event: %s, must be: %s, iteration: %d", game.Event.Name(), EVENT_DAY, game.Event.Iteration())
	}

	game.Event.SetStatus(PROCESSED)
	time.Sleep(5 * time.Millisecond)

	if game.Event.Name() != EVENT_NIGHT_RESULT {
		t.Errorf("Game has wrong event: %s, must be: %s, iteration: %d", game.Event.Name(), EVENT_NIGHT_RESULT, game.Event.Iteration())
	}

	if !citizen.Out() {
		t.Errorf("night result is wrong")
	}
}

func TestGameEventLoopFirstLoop(t *testing.T) {
	game := NewGame()
	game.Run()
	game.Event = NewGameEvent()

	for i := 0; i < 10 ; i++ {
		player := NewPlayer()
		player.Run(t)
		player.SetGame(game)
		game.Players.Add(player)
	}

	events := []string{
		EVENT_GREET_CITIZENS,//start
		EVENT_GREET_CITIZENS,
		EVENT_GREET_CITIZENS,//end
		EVENT_NIGHT,
		EVENT_GREET_MAFIA,//start
		EVENT_GREET_MAFIA,
		EVENT_GREET_MAFIA,//end
		EVENT_DAY,
		EVENT_COURT,//start
		EVENT_COURT,
		EVENT_COURT_RESULT,
		EVENT_COURT,//end
		EVENT_NIGHT,
		EVENT_MAFIA,//start
		EVENT_MAFIA,
		EVENT_MAFIA,//end
	}

	for _, eventName := range events {
		game.Event.SetStatus(PROCESSED)
		time.Sleep(2 * time.Millisecond)
		t.Logf("Check %s, current %s, iteration %d", eventName, game.Event.Name(), game.Event.Iteration())
		if game.Event.Name() != eventName {
			t.Errorf("Event has wrong name %s", game.Event.Name())
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
		EVENT_MAFIA,//start
		EVENT_MAFIA,
		EVENT_MAFIA,//end
		EVENT_DOCTOR,//start
		EVENT_DOCTOR,
		EVENT_DOCTOR,//end
		EVENT_SHERIFF,//start
		EVENT_SHERIFF,
		//EVENT_SHERIFF_RESULT, sheriff has not choice
		EVENT_SHERIFF,//end
		EVENT_GIRL,//start
		EVENT_GIRL,
		EVENT_GIRL,//end
		EVENT_DAY,
		EVENT_NIGHT_RESULT,
		EVENT_COURT,//start
		EVENT_COURT,
		EVENT_COURT_RESULT,
		EVENT_COURT,//end
		EVENT_NIGHT,//start
	}

	game.Event = NewAcceptEvent(game.Iteration, EVENT_NIGHT, ACTION_ACCEPT)
	for _, eventName := range events {
		game.Event.SetStatus(PROCESSED)
		time.Sleep(10 * time.Millisecond)
		t.Logf("Check: %s, current: %s, iteration: %d", eventName, game.Event.Name(), game.Event.Iteration())
		if game.Event.Name() != eventName {
			t.Errorf("Event has wrong name, check %s, current: %s", eventName, game.Event.Name())
		}
	}
}