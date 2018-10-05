package main

import (
	"fmt"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

var Games = make(map[int]*Game, 0)

type Game struct {
	Id            int
	Players       *Players
	EventsQueue   *EventQueue
	EventsHistory *EventHistory
	Event         IEvent
	Iteration     int
	Winner        int
}

func NewGame() *Game {
	return &Game{
		Id:            rand.Intn(99),
		Players:       NewPlayers(),
		EventsQueue:   NewEventQueue(),
		EventsHistory: NewEventHistory(),
		Iteration:     1,
		Event:         NewGameEvent(),
	}
}

func (game *Game) Run() {
	go game.EventLoop()
}

func (game *Game) isOver() bool {

	if game.Event.Name() == EVENT_GAME ||
		game.Event.Name() == EVENT_GAME_START ||
		game.Event.Name() == EVENT_GREET_CITIZENS ||
		game.Event.Name() == EVENT_GREET_MAFIA {
		return false
	}

	mafia := len(game.Players.FindByRole(ROLE_MAFIA))
	citizens := len(game.Players.FindByRole(ROLE_CITIZEN))
	sheriff := len(game.Players.FindByRole(ROLE_SHERIFF))
	girl := len(game.Players.FindByRole(ROLE_GIRL))
	doctor := len(game.Players.FindByRole(ROLE_DOCTOR))

	if citizens+sheriff+girl+doctor == 0 {
		game.Winner = ROLE_MAFIA
	}

	if mafia == 0 {
		game.Winner = ROLE_CITIZEN
	}

	return game.Winner != 0
}

func (game *Game) SetNextEvent() error {

	if game.EventsQueue.Len() == 0 {
		game.initEventQueue()
	}

	event := game.EventsQueue.Pop()

	if event != nil {
		game.EventsHistory.Push(game.Event)
		game.Event = event
		return nil
	}

	return fmt.Errorf("Can not get next event")
}

func (game *Game) initEventQueue() error {
	queue := game.EventsQueue
	eventName := game.Event.Name()
	for {

		if game.isOver() {
			queue.Push(NewGameOverEvent(game.Iteration, game.Winner))
			return nil
		}

		switch eventName {
		case EVENT_GAME:
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_GAME_START, ACTION_START))
			return nil
		case EVENT_GAME_START:
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_GREET_CITIZENS, ACTION_START))
			queue.Push(NewGreetCitizensEvent(game.Iteration))
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_GREET_CITIZENS, ACTION_END))
			return nil
		case EVENT_GREET_CITIZENS:
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_NIGHT, ACTION_START))
			return nil
		case EVENT_NIGHT:
			if game.Iteration == 1 {
				queue.Push(NewAcceptEvent(game.Iteration, EVENT_GREET_MAFIA, ACTION_START))
				queue.Push(NewGreetMafiaEvent(game.Iteration))
				queue.Push(NewAcceptEvent(game.Iteration, EVENT_GREET_MAFIA, ACTION_END))
				return nil
			}

			queue.Push(NewAcceptEvent(game.Iteration, EVENT_MAFIA, ACTION_START))
			queue.Push(NewMafiaEvent(game.Iteration))
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_MAFIA, ACTION_END))
			return nil
		case EVENT_GREET_MAFIA:
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_DAY, ACTION_START))
			return nil
		case EVENT_MAFIA:
			if game.Players.FindOneByRole(ROLE_DOCTOR) != nil {
				queue.Push(NewAcceptEvent(game.Iteration, EVENT_DOCTOR, ACTION_START))
				queue.Push(NewDoctorEvent(game.Iteration))
				queue.Push(NewAcceptEvent(game.Iteration, EVENT_DOCTOR, ACTION_END))
				return nil
			}
			eventName = EVENT_DOCTOR
			break
		case EVENT_DOCTOR:
			if game.Players.FindOneByRole(ROLE_SHERIFF) != nil {
				queue.Push(NewAcceptEvent(game.Iteration, EVENT_SHERIFF, ACTION_START))
				queue.Push(NewSheriffEvent(game.Iteration))
				queue.Push(NewSheriffResultEvent(game.Iteration))
				queue.Push(NewAcceptEvent(game.Iteration, EVENT_SHERIFF, ACTION_END))
				return nil
			}
			eventName = EVENT_SHERIFF
			break
		case EVENT_SHERIFF:
			if game.Players.FindOneByRole(ROLE_GIRL) != nil {
				queue.Push(NewAcceptEvent(game.Iteration, EVENT_GIRL, ACTION_START))
				queue.Push(NewGirlEvent(game.Iteration))
				queue.Push(NewAcceptEvent(game.Iteration, EVENT_GIRL, ACTION_END))
				return nil
			}
			eventName = EVENT_GIRL
			break
		case EVENT_GIRL:
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_DAY, ACTION_START))
			return nil
		case EVENT_DAY:
			if game.Iteration != 1 {
				queue.Push(NewNightResultEvent(game.Iteration))
				return nil
			}

			eventName = EVENT_NIGHT_RESULT
			break
		case EVENT_NIGHT_RESULT:
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_COURT, ACTION_START))
			queue.Push(NewCourtEvent(game.Iteration))
			queue.Push(NewCourtResultEvent(game.Iteration))
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_COURT, ACTION_END))
			return nil
		case EVENT_COURT:
			game.Iteration++
			queue.Push(NewAcceptEvent(game.Iteration, EVENT_NIGHT, ACTION_START))
			return nil
		}
	}

	return nil
}

func (game *Game) EventLoop() {
	ticker := time.NewTicker(1 * time.Millisecond)
	for range ticker.C {
		switch game.Event.Status() {
		case NOT_IN_PROCESS:
			err := game.Event.Process(game.Players, game.EventsHistory)
			if err != nil {
				log.Warningf("Event: %s, err: %v", game.Event.Name(), err)
			}
			break
		case PROCESSED:
			game.SetNextEvent()
			break
		}
	}
}
