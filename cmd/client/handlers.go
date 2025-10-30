package main

import (
	"fmt"

	"github.com/JA50N14/learn-pub-sub-starter/internal/gamelogic"
	"github.com/JA50N14/learn-pub-sub-starter/internal/routing"
)

func handlerMoves(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(move)
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}
