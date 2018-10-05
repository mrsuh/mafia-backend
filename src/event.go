package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

const NOT_IN_PROCESS = 1
const IN_PROCESS = 2
const PROCESSED = 3

const EVENT_GAME = "game"
const EVENT_GAME_START = "game_start"
const EVENT_GAME_OVER = "game_over"
const EVENT_DAY = "day"
const EVENT_NIGHT = "night"
const EVENT_NIGHT_RESULT = "night_result"
const EVENT_COURT = "court"
const EVENT_COURT_RESULT = "court_result"
const EVENT_MAFIA = "mafia"
const EVENT_DOCTOR = "doctor"
const EVENT_SHERIFF = "sheriff"
const EVENT_SHERIFF_RESULT = "sheriff_result"
const EVENT_GIRL = "girl"
const EVENT_GREET_MAFIA = "greet_mafia"
const EVENT_GREET_CITIZENS = "greet_citizen"

const ACTION_CREATE = "create"
const ACTION_RECONNECT = "reconnect"
const ACTION_JOIN = "join"
const ACTION_START = "start"
const ACTION_END = "end"
const ACTION_OVER = "over"
const ACTION_ROLE = "role"
const ACTION_PLAYERS = "players"
const ACTION_ACCEPT = "accept"
const ACTION_VOTE = "vote"
const ACTION_CHOICE = "choice"
const ACTION_OUT = "out"

type IEvent interface {
	AddAction(name string, f func(players *Players, history *EventHistory, player *Player, msg *Message) error)
	Actions() map[string]func(players *Players, history *EventHistory, player *Player, msg *Message) error
	Status() int
	SetStatus(status int)
	SetName(name string)
	Name() string
	Iteration() int
	Process(players *Players, history *EventHistory) error
	Action(players *Players, history *EventHistory, player *Player, msg *Message) error
}

type Event struct {
	status    int
	event     string
	iteration int
	actions   map[string]func(players *Players, history *EventHistory, player *Player, msg *Message) error
}

func NewEvent() Event {
	return Event{
		status:    NOT_IN_PROCESS,
		iteration: 1,
		actions:   make(map[string]func(players *Players, history *EventHistory, player *Player, msg *Message) error, 0),
	}
}

func (e *Event) AddAction(name string, f func(players *Players, history *EventHistory, player *Player, msg *Message) error) {
	e.actions[name] = f
}

func (e *Event) Actions() map[string]func(players *Players, history *EventHistory, player *Player, msg *Message) error {
	return e.actions
}

func (e *Event) Status() int {
	return e.status
}

func (e *Event) SetStatus(status int) {
	e.status = status
}

func (e *Event) SetName(name string) {
	e.event = name
}

func (e *Event) Name() string {
	return e.event
}

func (e *Event) Iteration() int {
	return e.iteration
}

func (e *Event) Process(players *Players, history *EventHistory) error {
	e.status = IN_PROCESS
	return nil
}

func (e *Event) Action(players *Players, history *EventHistory, player *Player, msg *Message) error {
	return nil
}

/*
	EventChoice
*/
type IEventChoice interface {
	SetChoice(choice *Player)
	Choice() *Player
}

type EventChoice struct {
	choice *Player
}

func (e *EventChoice) SetChoice(choice *Player) {
	e.choice = choice
}

func (e *EventChoice) Choice() *Player {
	return e.choice
}

/*
	EventVote
*/

type IEventVote interface {
	AddVoted(player *Player, vote *Player)
	IsAllVoted(players []*Player) bool
	FindVotedById(id int) *Player
	Votes() []*Player
	SetCandidate(player *Player)
	Candidate() *Player
}

type EventVote struct {
	voted     map[*Player]*Player
	candidate *Player
}

func (e *EventVote) AddVoted(player *Player, vote *Player) {
	e.voted[player] = vote
}

func (e *EventVote) IsAllVoted(players []*Player) bool {
	for _, player := range players {
		if e.FindVotedById(player.Id()) == nil {
			return false
		}
	}

	return true
}

func (e *EventVote) FindVotedById(id int) *Player {
	for player := range e.voted {
		if player.Id() == id {
			return player
		}
	}

	return nil
}

func (e *EventVote) Votes() []*Player {
	votes := make([]*Player, 0)
	for _, vote := range e.voted {
		votes = append(votes, vote)
	}

	return votes
}

func (e *EventVote) SetCandidate(player *Player) {
	e.candidate = player
}

func (e *EventVote) Candidate() *Player {
	return e.candidate
}

/*
	EventQueue
*/
type EventQueue struct {
	data []IEvent
}

func NewEventQueue() *EventQueue {
	return &EventQueue{data: make([]IEvent, 0)}
}

func (e *EventQueue) Push(event IEvent) {
	e.data = append(e.data, event)
}

func (e *EventQueue) Pop() IEvent {
	if len(e.data) == 0 {
		return nil
	}

	event := e.data[0]
	e.data = e.data[1:]

	return event
}

func (e *EventQueue) Len() int {
	return len(e.data)
}

func (e *EventQueue) Clear() {
	e.data = make([]IEvent, 0)
}

/*
	EventHistory
*/
type EventHistory struct {
	data []IEvent
}

func NewEventHistory() *EventHistory {
	return &EventHistory{data: make([]IEvent, 0)}
}

func (e *EventHistory) Push(event IEvent) {
	e.data = append(e.data, event)
}

func (e *EventHistory) FindEventChoice(eventName string, iteration int) IEventChoice {
	for _, event := range e.data {
		if event.Name() == eventName && event.Iteration() == iteration {
			eventChoice, ok := event.(IEventChoice)

			if !ok {
				continue
			}
			return eventChoice
		}
	}

	return nil
}

func (e *EventHistory) FindEventVote(eventName string, iteration int) IEventVote {
	for _, event := range e.data {
		if event.Name() == eventName && event.Iteration() == iteration {
			eventVote, ok := event.(IEventVote)

			if !ok {
				continue
			}

			return eventVote
		}
	}

	return nil
}

/*
AcceptEvent
*/
type AcceptEvent struct {
	Event
	accepted []*Player
	action   string
}

func NewAcceptEvent(iter int, event string, action string) *AcceptEvent {
	e := &AcceptEvent{}
	e.Event = NewEvent()
	e.accepted = make([]*Player, 0)
	e.status = NOT_IN_PROCESS
	e.iteration = iter
	e.event = event
	e.action = action
	e.AddAction(action, e.AcceptAction)
	return e
}

func (event *AcceptEvent) AddAccepted(player *Player) {
	event.accepted = append(event.accepted, player)
}

func (event *AcceptEvent) IsAllAccepted(players []*Player) bool {
	for _, player := range players {
		if event.FindAcceptedById(player.Id()) == nil {
			return false
		}
	}

	return true
}

func (event *AcceptEvent) FindAcceptedById(id int) *Player {
	for _, player := range event.accepted {
		if player.Id() == id {
			return player
		}
	}

	return nil
}

func (event *AcceptEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS
	for _, player := range players.FindAll() {
		rmsg := NewEventMessage(event, event.action)
		player.SendMessage(rmsg)
	}

	return nil
}

func (event *AcceptEvent) AcceptAction(players *Players, history *EventHistory, player *Player, msg *Message) error {
	event.AddAccepted(player)

	if event.IsAllAccepted(players.FindAll()) {
		event.SetStatus(PROCESSED)
	}

	return nil
}

/*
 CourtEvent
*/
type CourtEvent struct {
	Event
	EventVote
}

func NewCourtEvent(iter int) *CourtEvent {
	e := &CourtEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.iteration = iter
	e.event = EVENT_COURT
	e.AddAction(ACTION_VOTE, e.VoteAction)
	e.voted = make(map[*Player]*Player, 0)
	return e
}

func (event *CourtEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS

	playersInfo := make([]interface{}, 0)
	for _, player := range players.FindAll() {
		playerInfo := map[string]interface{}{
			"username": player.Name(),
			"id":       player.Id(),
		}
		playersInfo = append(playersInfo, playerInfo)
	}

	response := NewEventMessage(event, ACTION_PLAYERS)
	response.Data = playersInfo

	for _, player := range players.FindAll() {
		player.SendMessage(response)
	}

	return nil
}

func (event *CourtEvent) VoteAction(players *Players, history *EventHistory, player *Player, msg *Message) error {

	voteId := int(msg.Data.(float64))
	vote := players.FindOneById(voteId)

	if vote == nil {
		rmsg := NewEventMessage(event, ACTION_PLAYERS)
		rmsg.Status = STATUS_ERR
		err := "invalid player id"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	rmsg := NewEventMessage(event, ACTION_VOTE)
	rmsg.Data = map[string]interface{}{"player": player.Name(), "vote": vote.Name()}

	for _, pl := range players.FindAll() {
		pl.SendMessage(rmsg)
	}

	event.AddVoted(player, vote)

	if event.IsAllVoted(players.FindAll()) {
		event.SetStatus(PROCESSED)
	}

	return nil
}

/*
CourtResult
*/
type CourtResultEvent struct {
	Event
	AcceptEvent
}

func NewCourtResultEvent(iter int) *CourtResultEvent {
	e := &CourtResultEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_COURT_RESULT
	e.iteration = iter
	e.AddAction(ACTION_ACCEPT, e.AcceptAction)
	e.accepted = make([]*Player, 0)
	return e
}

func (event *CourtResultEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS

	eventCourt := history.FindEventVote(EVENT_COURT, event.iteration)

	if eventCourt == nil {
		event.SetStatus(PROCESSED)
		return fmt.Errorf("court has not event")
	}

	maxVotes := 0
	votes := make(map[*Player]int, 0)
	for _, vote := range eventCourt.Votes() {
		if _, ok := votes[vote]; !ok {
			votes[vote] = 0
		}
		votes[vote]++

		if votes[vote] > maxVotes {
			maxVotes = votes[vote]
		}
	}

	candidates := make([]*Player, 0)

	for candidate, vote := range votes {
		if vote == maxVotes {
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) > 1 {
		rmsg := NewEventMessage(event, ACTION_OUT)
		for _, player := range players.FindAll() {
			player.SendMessage(rmsg)
		}
		return fmt.Errorf("Too many candidates")
	}

	if len(candidates) == 0 {
		rmsg := NewEventMessage(event, ACTION_OUT)
		for _, player := range players.FindAll() {
			player.SendMessage(rmsg)
		}
		return fmt.Errorf("Too few candidates")
	}

	courtCandidate := candidates[0]

	rmsg := NewEventMessage(event, ACTION_OUT)
	rmsg.Data = map[string]interface{}{"id": courtCandidate.Id(), "username": courtCandidate.Name()}

	playersFor := players.FindAll()
	courtCandidate.SetOut(true)
	for _, player := range playersFor {
		player.SendMessage(rmsg)
	}

	return nil
}

func (event *CourtResultEvent) AcceptAction(players *Players, history *EventHistory, player *Player, msg *Message) error {
	event.AddAccepted(player)

	if event.IsAllAccepted(players.FindAll()) {
		event.SetStatus(PROCESSED)
	}

	return nil
}

/*
DoctorEvent
*/
type DoctorEvent struct {
	Event
	EventChoice
}

func NewDoctorEvent(iter int) *DoctorEvent {
	e := &DoctorEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_DOCTOR
	e.iteration = iter
	e.AddAction(ACTION_CHOICE, e.ChoiceAction)
	return e
}

func (event *DoctorEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS

	doctor := players.FindOneByRole(ROLE_DOCTOR)

	if doctor == nil {
		event.status = PROCESSED
		return fmt.Errorf("Player is not active")
	}

	playersInfo := make([]interface{}, 0)
	for _, player := range players.FindAll() {
		playerInfo := map[string]interface{}{
			"username": player.Name(),
			"id":       player.Id(),
		}
		playersInfo = append(playersInfo, playerInfo)
	}

	response := NewEventMessage(event, ACTION_PLAYERS)
	response.Data = playersInfo

	doctor.SendMessage(response)

	return nil
}

func (event *DoctorEvent) ChoiceAction(players *Players, history *EventHistory, player *Player, msg *Message) error {

	if player.Role() != ROLE_DOCTOR {
		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "player have wrong role for this action"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	choiceId := int(msg.Data.(float64))
	choice := players.FindOneById(choiceId)

	if choice == nil {
		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "invalid player id"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	prevEvent := history.FindEventChoice(event.Name(), event.Iteration()-1)

	if prevEvent != nil &&
		prevEvent.Choice() != nil &&
		prevEvent.Choice().Id() == choice.Id() {

		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "you can not do this action with this player several times in a row"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	event.SetChoice(choice)
	event.SetStatus(PROCESSED)

	return nil
}

/*
GameEvent
*/
type GameEvent struct {
	Event
}

func NewGameEvent() *GameEvent {
	e := &GameEvent{}
	e.Event = NewEvent()
	e.SetName(EVENT_GAME)
	e.AddAction(ACTION_CREATE, e.CreateAction)
	e.AddAction(ACTION_JOIN, e.JoinAction)
	e.AddAction(ACTION_START, e.StartAction)
	return e
}

func (event *GameEvent) CreateAction(players *Players, history *EventHistory, player *Player, msg *Message) error {

	data := msg.Data.(map[string]interface{})

	username := data["username"].(string)

	if players.FindOneByUsername(username) != nil {
		rmsg := NewEventMessage(event, ACTION_CREATE)
		rmsg.Status = STATUS_ERR
		err := "username already exists"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	player.SetName(username)
	players.Add(player)

	response := NewEventMessage(event, ACTION_CREATE)
	response.Data = map[string]interface{}{"username": player.Name(), "id": player.Id(), "game": player.Game().Id}

	player.SendMessage(response)

	event.sendPlayersInfo(players)

	return nil
}

func (event *GameEvent) JoinAction(players *Players, history *EventHistory, player *Player, msg *Message) error {

	data := msg.Data.(map[string]interface{})

	username := data["username"].(string)

	if players.FindOneByUsername(username) != nil {
		rmsg := NewEventMessage(event, ACTION_JOIN)
		rmsg.Status = STATUS_ERR
		err := "username already exists"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	player.SetName(username)
	players.Add(player)

	response := NewEventMessage(event, ACTION_JOIN)
	response.Data = map[string]interface{}{"username": player.Name(), "id": player.Id(), "game": player.Game().Id}

	player.SendMessage(response)

	event.sendPlayersInfo(players)

	return nil
}

func (event *GameEvent) sendPlayersInfo(players *Players) {
	playersInfo := make([]interface{}, 0)
	for _, player := range players.FindAll() {
		playerInfo := map[string]interface{}{
			"username": player.Name(),
			"id":       player.Id(),
		}
		playersInfo = append(playersInfo, playerInfo)
	}

	responseForAll := NewEventMessage(event, ACTION_PLAYERS)
	responseForAll.Data = playersInfo
	for _, player := range players.FindAll() {
		player.SendMessage(responseForAll)
	}
}

func (event *GameEvent) StartAction(players *Players, history *EventHistory, player *Player, msg *Message) error {

	if !player.Master() {
		rmsg := NewEventMessage(event, ACTION_START)
		rmsg.Status = STATUS_ERR
		err := "you have not rights to start game"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	if len(players.FindAll()) < 3 {
		rmsg := NewEventMessage(event, ACTION_START)
		rmsg.Status = STATUS_ERR
		err := "too few players to start game"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	event.SetStatus(PROCESSED)

	return nil
}

/*
GirlEvent
*/
type GirlEvent struct {
	Event
	EventChoice
}

func NewGirlEvent(iter int) *GirlEvent {
	e := &GirlEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_GIRL
	e.iteration = iter
	e.AddAction(ACTION_CHOICE, e.ChoiceAction)
	return e
}

func (event *GirlEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS

	player := players.FindOneByRole(ROLE_GIRL)

	if player == nil {
		event.status = PROCESSED
		return fmt.Errorf("Player is not active")
	}

	playersInfo := make([]interface{}, 0)
	for _, player := range players.FindAll() {
		playerInfo := map[string]interface{}{
			"username": player.Name(),
			"id":       player.Id(),
		}
		playersInfo = append(playersInfo, playerInfo)
	}

	response := NewEventMessage(event, ACTION_PLAYERS)
	response.Data = playersInfo

	player.SendMessage(response)

	return nil
}

func (event *GirlEvent) ChoiceAction(players *Players, history *EventHistory, player *Player, msg *Message) error {

	if player.Role() != ROLE_GIRL {
		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "player have wrong role for this action"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	choiceId := int(msg.Data.(float64))
	choice := players.FindOneById(choiceId)

	if choice == nil {
		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "invalid player id"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	prevEvent := history.FindEventChoice(event.Name(), event.Iteration())

	if prevEvent != nil &&
		prevEvent.Choice() != nil &&
		prevEvent.Choice().Id() == choice.Id() {
		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "you can not do this action with this player several times in a row"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	event.SetChoice(choice)
	event.SetStatus(PROCESSED)

	return nil
}

/*
GreetCitizensEvent
*/
type GreetCitizensEvent struct {
	Event
	AcceptEvent
}

func NewGreetCitizensEvent(iter int) *GreetCitizensEvent {
	e := &GreetCitizensEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_GREET_CITIZENS
	e.iteration = iter
	e.AddAction(ACTION_ACCEPT, e.AcceptAction)
	e.accepted = make([]*Player, 0)
	return e
}

func Shuffle(vals []int) []int {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]int, len(vals))
	n := len(vals)
	for i := 0; i < n; i++ {
		randIndex := r.Intn(len(vals))
		ret[i] = vals[randIndex]
		vals = append(vals[:randIndex], vals[randIndex+1:]...)
	}
	return ret
}

func (event *GreetCitizensEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS

	playersCount := len(players.FindAll())
	roles := Shuffle(event.getRoles(playersCount))
	for index, player := range players.FindAll() {
		player.SetRole(roles[index])

		rmsg := NewEventMessage(event, ACTION_ROLE)
		rmsg.Data = player.Role()
		player.SendMessage(rmsg)
	}

	return nil
}

func (event *GreetCitizensEvent) AcceptAction(players *Players, history *EventHistory, player *Player, msg *Message) error {
	event.AddAccepted(player)

	if event.IsAllAccepted(players.FindAll()) {
		event.SetStatus(PROCESSED)
	}

	return nil
}

func (event *GreetCitizensEvent) getRoles(playersCount int) []int {
	roles := make([]int, 0)

	mafia := 0
	girl := 0
	sheriff := 0
	doctor := 0
	citizens := 0

	switch true {
	case playersCount >= 5:
		mafia = int(math.Floor(float64(playersCount / 3)))
		girl = 1
		sheriff = 1
		doctor = 1
		citizens = playersCount - (girl + sheriff + doctor + mafia)
		break
	case playersCount == 3:
		mafia = 1
		doctor = 1
		citizens = 1
		break
	case playersCount == 4:
		mafia = 1
		girl = 1
		doctor = 1
		citizens = 1
		break
	}

	for i := 1; i <= mafia; i++ {
		roles = append(roles, ROLE_MAFIA)
	}

	for i := 1; i <= citizens; i++ {
		roles = append(roles, ROLE_CITIZEN)
	}

	if girl != 0 {
		roles = append(roles, ROLE_GIRL)
	}

	if sheriff != 0 {
		roles = append(roles, ROLE_SHERIFF)
	}

	if doctor != 0 {
		roles = append(roles, ROLE_DOCTOR)
	}

	return roles
}

/*
GreetMafiaEvent
*/
type GreetMafiaEvent struct {
	Event
	AcceptEvent
}

func NewGreetMafiaEvent(iter int) *GreetMafiaEvent {
	e := &GreetMafiaEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_GREET_MAFIA
	e.iteration = iter
	e.AddAction(ACTION_ACCEPT, e.AcceptAction)
	e.accepted = make([]*Player, 0)
	return e
}

func (event *GreetMafiaEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS
	rmsg := NewEventMessage(event, ACTION_PLAYERS)

	playersInfo := make([]interface{}, 0)
	for _, player := range players.FindAll() {
		if player.Role() != ROLE_MAFIA {
			continue
		}
		playerInfo := map[string]interface{}{
			"username": player.Name(),
			"id":       player.Id(),
		}
		playersInfo = append(playersInfo, playerInfo)
	}
	rmsg.Data = playersInfo

	for _, player := range players.FindByRole(ROLE_MAFIA) {
		player.SendMessage(rmsg)
	}
	return nil
}

func (event *GreetMafiaEvent) AcceptAction(players *Players, history *EventHistory, player *Player, msg *Message) error {
	event.AddAccepted(player)

	if event.IsAllAccepted(players.FindByRole(ROLE_MAFIA)) {
		event.SetStatus(PROCESSED)
	}

	return nil
}

/*
MafiaEvent
*/
type MafiaEvent struct {
	Event
	EventVote
}

func NewMafiaEvent(iter int) *MafiaEvent {
	e := &MafiaEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_MAFIA
	e.iteration = iter
	e.AddAction(ACTION_VOTE, e.VoteAction)
	e.voted = make(map[*Player]*Player, 0)
	return e
}

func (event *MafiaEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS

	playersInfo := make([]interface{}, 0)
	for _, player := range players.FindAll() {
		if player.Role() == ROLE_MAFIA {
			continue
		}
		playerInfo := map[string]interface{}{
			"username": player.Name(),
			"id":       player.Id(),
		}
		playersInfo = append(playersInfo, playerInfo)
	}

	response := NewEventMessage(event, ACTION_PLAYERS)
	response.Data = playersInfo

	for _, player := range players.FindByRole(ROLE_MAFIA) {
		player.SendMessage(response)
	}

	return nil
}

func (event *MafiaEvent) VoteAction(players *Players, history *EventHistory, player *Player, msg *Message) error {

	if player.Role() != ROLE_MAFIA {
		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "player have wrong role for this action"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	voteId := int(msg.Data.(float64))
	vote := players.FindOneById(voteId)

	if vote == nil {
		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "invalid player id"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	event.AddVoted(player, vote)

	if !event.IsAllVoted(players.FindByRole(ROLE_MAFIA)) {
		return nil
	}

	maxVotes := 0
	votes := make(map[*Player]int, 0)
	for _, vote := range event.Votes() {
		if _, ok := votes[vote]; !ok {
			votes[vote] = 0
		}
		votes[vote]++

		if votes[vote] > maxVotes {
			maxVotes = votes[vote]
		}
	}

	candidates := make([]*Player, 0)

	for candidate, vote := range votes {
		if vote == maxVotes {
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) > 1 {
		event.SetStatus(PROCESSED)
		return fmt.Errorf("Too many candidates")
	}

	if len(candidates) == 0 {
		event.SetStatus(PROCESSED)
		return fmt.Errorf("Too few candidates")
	}

	event.SetCandidate(candidates[0])
	event.SetStatus(PROCESSED)

	return nil
}

/*
NightResultEvent
*/
type NightResultEvent struct {
	Event
	AcceptEvent
}

func NewNightResultEvent(iter int) *NightResultEvent {
	e := &NightResultEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_NIGHT_RESULT
	e.iteration = iter
	e.AddAction(ACTION_ACCEPT, e.AcceptAction)
	e.accepted = make([]*Player, 0)
	return e
}

func (event *NightResultEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS

	if event.Iteration() == 1 {
		log.Debugf("event has iteration %d", event.Iteration())
		event.SetStatus(PROCESSED)
		return nil
	}

	eventMafia := history.FindEventVote(EVENT_MAFIA, event.iteration)
	eventDoctor := history.FindEventChoice(EVENT_DOCTOR, event.iteration)
	eventGirl := history.FindEventChoice(EVENT_GIRL, event.iteration)

	var mafiaCandidate *Player
	var doctorChoice *Player
	var girlChoice *Player

	if eventMafia != nil {
		mafiaCandidate = eventMafia.Candidate()
	}

	if eventDoctor != nil {
		doctorChoice = eventDoctor.Choice()
	}

	if eventGirl != nil {
		girlChoice = eventGirl.Choice()
	}

	if mafiaCandidate == nil {
		rmsg := NewEventMessage(event, ACTION_OUT)
		for _, player := range players.FindAll() {
			player.SendMessage(rmsg)
		}
		return fmt.Errorf("mafia has not candidate")
	}

	if girlChoice != nil && girlChoice.Id() == mafiaCandidate.Id() {
		rmsg := NewEventMessage(event, ACTION_OUT)
		for _, player := range players.FindAll() {
			player.SendMessage(rmsg)
		}
		return fmt.Errorf("mafia killed no one becouse of the girl")
	}

	if doctorChoice != nil && doctorChoice.Id() == mafiaCandidate.Id() {
		rmsg := NewEventMessage(event, ACTION_OUT)
		for _, player := range players.FindAll() {
			player.SendMessage(rmsg)
		}
		return fmt.Errorf("mafia killed no one becouse of the doctor")
	}

	rmsg := NewEventMessage(event, ACTION_OUT)
	rmsg.Data = map[string]interface{}{"id": mafiaCandidate.Id(), "username": mafiaCandidate.Name()}

	for _, player := range players.FindAll() {
		player.SendMessage(rmsg)
	}

	mafiaCandidate.SetOut(true)

	return nil
}

func (event *NightResultEvent) AcceptAction(players *Players, history *EventHistory, player *Player, msg *Message) error {
	event.AddAccepted(player)

	if event.IsAllAccepted(players.FindAll()) {
		event.SetStatus(PROCESSED)
	}

	return nil
}

/*
SheriffEvent
*/
type SheriffEvent struct {
	Event
	EventChoice
}

func NewSheriffEvent(iter int) *SheriffEvent {
	e := &SheriffEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_SHERIFF
	e.iteration = iter
	e.AddAction(ACTION_CHOICE, e.ChoiceAction)
	return e
}

func (event *SheriffEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS

	player := players.FindOneByRole(ROLE_SHERIFF)

	if player == nil {
		event.status = PROCESSED
		return fmt.Errorf("Player is not active")
	}

	playersInfo := make([]interface{}, 0)
	for _, player := range players.FindAll() {
		if player.Role() == ROLE_SHERIFF {
			continue
		}
		playerInfo := map[string]interface{}{
			"username": player.Name(),
			"id":       player.Id(),
		}
		playersInfo = append(playersInfo, playerInfo)
	}

	response := NewEventMessage(event, ACTION_PLAYERS)
	response.Data = playersInfo

	player.SendMessage(response)

	return nil
}

func (event *SheriffEvent) ChoiceAction(players *Players, history *EventHistory, player *Player, msg *Message) error {

	if player.Role() != ROLE_SHERIFF {
		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "player have wrong role for this action"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	choiceId := int(msg.Data.(float64))
	choice := players.FindOneById(choiceId)

	if choice == nil {
		rmsg := NewEventMessage(event, ACTION_CHOICE)
		rmsg.Status = STATUS_ERR
		err := "invalid player id"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	event.SetChoice(choice)
	event.SetStatus(PROCESSED)

	return nil
}

/*
SheriffResultEvent
*/
type SheriffResultEvent struct {
	Event
}

func NewSheriffResultEvent(iter int) *SheriffResultEvent {
	e := &SheriffResultEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_SHERIFF_RESULT
	e.iteration = iter
	e.AddAction(ACTION_ACCEPT, e.AcceptAction)
	return e
}

func (event *SheriffResultEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS
	sheriff := players.FindOneByRole(ROLE_SHERIFF)

	sheriffEvent := history.FindEventChoice("sheriff", event.Iteration())

	if sheriffEvent == nil {
		event.status = PROCESSED
		return fmt.Errorf("has not event")
	}

	if sheriffEvent.Choice() == nil {
		event.status = PROCESSED
		return fmt.Errorf("has not choice")
	}

	rmsg := NewEventMessage(event, ACTION_ROLE)
	rmsg.Data = map[string]interface{}{"username": sheriffEvent.Choice().Name(), "role": sheriffEvent.Choice().Role()}
	sheriff.SendMessage(rmsg)

	return nil
}

func (event *SheriffResultEvent) AcceptAction(players *Players, history *EventHistory, player *Player, msg *Message) error {

	if player.Role() != ROLE_SHERIFF {
		rmsg := NewEventMessage(event, ACTION_ACCEPT)
		rmsg.Status = STATUS_ERR
		err := "player have wrong role for this action"
		rmsg.Data = err
		player.SendMessage(rmsg)
		return fmt.Errorf(err)
	}

	event.SetStatus(PROCESSED)

	return nil
}

/*
GameOverEvent
*/
type GameOverEvent struct {
	Event
	AcceptEvent
	winner int
}

func NewGameOverEvent(iter int, winner int) *GameOverEvent {
	e := &GameOverEvent{}
	e.Event = NewEvent()
	e.status = NOT_IN_PROCESS
	e.event = EVENT_GAME_OVER
	e.iteration = iter
	e.AddAction(ACTION_ACCEPT, e.AcceptAction)
	e.winner = winner
	return e
}

func (event *GameOverEvent) Process(players *Players, history *EventHistory) error {
	event.status = IN_PROCESS

	rmsg := NewEventMessage(event, ACTION_OVER)
	rmsg.Data = event.winner

	for _, player := range players.FindAllWithOut() {
		player.SendMessage(rmsg)
	}

	return nil
}

func (event *GameOverEvent) AcceptAction(players *Players, history *EventHistory, player *Player, msg *Message) error {
	event.AddAccepted(player)

	if event.IsAllAccepted(players.FindAllWithOut()) {
		event.SetStatus(PROCESSED)
	}

	return nil
}
