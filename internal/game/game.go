package game

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
)

const (
	PhaseLobby = "lobby"
	PhaseNight = "night"
	PhaseDay   = "day"
	PhaseEnded = "ended"
)

const (
	RoleVillager = "villager"
	RoleDemon    = "demon"
	RoleMonk     = "monk"
)

type Player struct {
	ID    string
	Name  string
	Role  Role
	Alive bool
}

type Role struct {
	Name    string
	Team    string
	Ability string
}

type Game struct {
	ID          string
	Phase       string
	DayNumber   int
	Players     []*Player
	Events      []string
	NightTarget map[string]string
	StartedAt   time.Time
}

func NewGame() *Game {
	return &Game{
		ID:          fmt.Sprintf("game-%d", time.Now().UnixNano()),
		Phase:       PhaseLobby,
		Players:     []*Player{},
		NightTarget: make(map[string]string),
	}
}

func (g *Game) PlayerCount() int {
	return len(g.Players)
}

func (g *Game) FindPlayerByName(name string) *Player {
	trimmed := strings.TrimSpace(name)
	for _, p := range g.Players {
		if strings.EqualFold(p.Name, trimmed) {
			return p
		}
	}
	return nil
}

func (g *Game) FindPlayerByID(id string) *Player {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (g *Game) AddPlayer(name string) (*Player, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, errors.New("player name is required")
	}
	if g.FindPlayerByName(trimmed) != nil {
		return nil, fmt.Errorf("player %q already exists", trimmed)
	}

	player := &Player{
		ID:    fmt.Sprintf("player-%d", time.Now().UnixNano()+int64(len(g.Players)+1)),
		Name:  trimmed,
		Alive: true,
	}
	g.Players = append(g.Players, player)
	return player, nil
}

func (g *Game) StartGame() error {
	if len(g.Players) < 3 {
		return errors.New("at least 3 players are required to start")
	}
	if g.Phase != PhaseLobby {
		return errors.New("game already started")
	}

	players := append([]*Player{}, g.Players...)
	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})

	for i := range players {
		players[i].Alive = true
	}

	assignments := map[string]Role{
		RoleDemon:    {Name: RoleDemon, Team: "evil", Ability: "kill another player at night"},
		RoleMonk:     {Name: RoleMonk, Team: "good", Ability: "protect a player at night"},
		RoleVillager: {Name: RoleVillager, Team: "good", Ability: "none"},
	}

	for i, player := range players {
		switch {
		case i == 0:
			player.Role = assignments[RoleDemon]
		case i == 1:
			player.Role = assignments[RoleMonk]
		default:
			player.Role = assignments[RoleVillager]
		}
	}

	g.Players = players
	g.Phase = PhaseNight
	g.DayNumber = 1
	g.NightTarget = make(map[string]string)
	g.StartedAt = time.Now()
	g.Events = []string{"The game begins. Night has fallen."}
	return nil
}

func (g *Game) ResolveNightAction(playerID, targetName string) error {
	if g.Phase != PhaseNight {
		return errors.New("it is not night")
	}

	actor := g.FindPlayerByID(playerID)
	if actor == nil {
		return errors.New("player not found")
	}
	if !actor.Alive {
		return errors.New("player is dead")
	}
	if targetName == "" {
		return errors.New("target is required")
	}

	target := g.FindPlayerByName(targetName)
	if target == nil || !target.Alive {
		return errors.New("target is not a valid alive player")
	}

	if actor.Role.Name == RoleDemon {
		g.NightTarget[playerID] = target.ID
		g.Events = append(g.Events, fmt.Sprintf("%s targeted %s at night.", actor.Name, target.Name))
		return nil
	}

	if actor.Role.Name == RoleMonk {
		g.NightTarget[playerID] = target.ID
		g.Events = append(g.Events, fmt.Sprintf("%s protected %s at night.", actor.Name, target.Name))
		return nil
	}

	return errors.New("this role cannot act at night")
}

func (g *Game) ResolveNight() {
	if g.Phase != PhaseNight {
		return
	}

	killer := ""
	protected := ""
	for playerID, targetID := range g.NightTarget {
		actor := g.FindPlayerByID(playerID)
		if actor == nil || !actor.Alive {
			continue
		}
		if actor.Role.Name == RoleDemon {
			killer = targetID
		}
		if actor.Role.Name == RoleMonk {
			protected = targetID
		}
	}

	if killer != "" && killer != protected {
		victim := g.FindPlayerByID(killer)
		if victim != nil {
			victim.Alive = false
			g.Events = append(g.Events, fmt.Sprintf("%s was killed by the demon.", victim.Name))
		}
	}

	g.Phase = PhaseDay
	g.DayNumber++
	g.Events = append(g.Events, fmt.Sprintf("Day %d begins.", g.DayNumber))
	g.NightTarget = make(map[string]string)
}

func (g *Game) ActivePlayers() []*Player {
	result := []*Player{}
	for _, p := range g.Players {
		if p.Alive {
			result = append(result, p)
		}
	}
	sort.Slice(result, func(i, j int) bool { return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name) })
	return result
}
