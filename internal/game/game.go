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

	Limit = 3
)

const (
	Villager      = "villager"
	Demon         = "demon"
	Monk          = "monk"
	Chef          = "chef"
	FortuneTeller = "fortune teller"
	Imp           = "imp"
	ScarletWoman  = "scarlet woman"
	Undertaker    = "undertaker"
	Dreamer       = "dreamer"
)

var roleDefinitions = map[string]Role{
	Demon: {
		Name: Demon,
		Team: "evil",
		Ability: Ability{
			Name:        "kill",
			Description: "kill another player at night",
		},
	},
	Monk: {
		Name:    Monk,
		Team:    "poble",
		Ability: Ability{Description: "protect a player at night"},
	},
	Villager: {
		Name:    Villager,
		Team:    "poble",
		Ability: Ability{Description: "none"},
	},
	FortuneTeller: {
		Name:    FortuneTeller,
		Team:    "poble",
		Ability: Ability{Description: "Each night, choose 2 players: you learn if either is a Demon. There is a good player that registers as a Demon to you."},
	},
}

type Player struct {
	ID    string
	Name  string
	Role  Role
	Alive bool
	Ready bool
}

type Role struct {
	Name    string
	Team    string
	Ability Ability
	IsDemon bool
}

type Ability struct {
	Name        string
	Description string
}

type Game struct {
	ID          string
	Phase       string
	DayNumber   int
	Ready       int
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
	if len(g.Players) < Limit {
		return fmt.Errorf("at least %d players are required to start", Limit)
	}
	if g.Ready < len(g.Players) {
		return errors.New("all players must be ready to start")
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

	for i, player := range players {
		switch {
		case i == 0:
			player.Role = roleDefinitions[Demon]
		case i == 1:
			player.Role = roleDefinitions[Monk]
		default:
			player.Role = roleDefinitions[Villager]
		}
	}

	g.Players = players
	g.Phase = PhaseNight
	g.DayNumber = 1
	g.NightTarget = make(map[string]string)
	g.StartedAt = time.Now()
	g.Events = []string{"A long time ago in the sleepy town of Ravenswood Bluff, during a hellish thunderstorm, on the stroke of midnight... you hear a scream. Rushing to the Town Square to investigate, you find your beloved Storyteller, myself, has been murdered... impaled on the hour hand of the clocktower, blood dripping onto the cobblestones below. You assume that this is the work of a Demon, and you are correct— a Demon that kills by night and takes on human form by day."}
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

	if actor.Role.Name == Demon {
		g.NightTarget[playerID] = target.ID
		g.Events = append(g.Events, fmt.Sprintf("%s targeted %s at night.", actor.Name, target.Name))
		return nil
	}

	if actor.Role.Name == Monk {
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
		if actor.Role.Name == Demon {
			killer = targetID
		}
		if actor.Role.Name == Monk {
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
