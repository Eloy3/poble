package game

import "testing"

func TestNewGameAndJoinPlayer(t *testing.T) {
	game := NewGame()
	player, err := game.AddPlayer("Alice")
	if err != nil {
		t.Fatalf("AddPlayer returned error: %v", err)
	}
	if player.Name != "Alice" {
		t.Fatalf("expected Alice, got %q", player.Name)
	}
	if game.PlayerCount() != 1 {
		t.Fatalf("expected 1 player, got %d", game.PlayerCount())
	}
}

func TestStartGameAssignsRolesAndStartsNight(t *testing.T) {
	game := NewGame()
	for _, name := range []string{"Alice", "Bob", "Charlie", "Dana"} {
		player, err := game.AddPlayer(name)
		if err != nil {
			t.Fatalf("AddPlayer(%q) returned error: %v", name, err)
		}
		player.Ready = true
	}
	game.Ready = game.PlayerCount()

	if err := game.StartGame(); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}

	if game.Phase != PhaseNight {
		t.Fatalf("expected phase %q, got %q", PhaseNight, game.Phase)
	}

	roles := map[string]int{}
	for _, p := range game.Players {
		roles[p.Role.Name]++
	}
	if roles[Demon] != 1 {
		t.Fatalf("expected exactly 1 demon, got %d", roles[Demon])
	}
}

func TestStartGameRequiresFourPlayers(t *testing.T) {
	game := NewGame()
	for _, name := range []string{"Alice", "Bob", "Charlie"} {
		player, err := game.AddPlayer(name)
		if err != nil {
			t.Fatalf("AddPlayer(%q) returned error: %v", name, err)
		}
		player.Ready = true
	}
	game.Ready = game.PlayerCount()

	if err := game.StartGame(); err == nil {
		t.Fatal("expected StartGame to require four players")
	}
	if game.Phase != PhaseLobby {
		t.Fatalf("expected phase %q, got %q", PhaseLobby, game.Phase)
	}
}

func TestStartGameRequiresAllPlayersReady(t *testing.T) {
	game := NewGame()
	for _, name := range []string{"Alice", "Bob", "Charlie", "Dana"} {
		player, err := game.AddPlayer(name)
		if err != nil {
			t.Fatalf("AddPlayer(%q) returned error: %v", name, err)
		}
		if name != "Dana" {
			player.Ready = true
			game.Ready++
		}
	}

	if err := game.StartGame(); err == nil {
		t.Fatal("expected StartGame to require all players to be ready")
	}
	if game.Phase != PhaseLobby {
		t.Fatalf("expected phase %q, got %q", PhaseLobby, game.Phase)
	}
}

func TestResolveNightActionRejectsInvalidTarget(t *testing.T) {
	game := NewGame()
	for _, name := range []string{"Alice", "Bob", "Charlie", "Dana"} {
		player, err := game.AddPlayer(name)
		if err != nil {
			t.Fatalf("AddPlayer(%q) returned error: %v", name, err)
		}
		player.Ready = true
	}
	game.Ready = game.PlayerCount()
	if err := game.StartGame(); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}

	player := game.FindPlayerByName("Alice")
	if player == nil {
		t.Fatal("expected Alice to exist")
	}

	if err := game.ResolveNightAction(player.ID, "Ghost"); err == nil {
		t.Fatal("expected invalid target to fail")
	}
}
