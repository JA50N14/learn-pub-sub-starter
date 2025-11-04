package main

import (
	"fmt"

	"github.com/JA50N14/learn-pub-sub-starter/internal/routing"
	"github.com/JA50N14/learn-pub-sub-starter/internal/pubsub"
	"github.com/JA50N14/learn-pub-sub-starter/internal/gamelogic"
)

func handlerLog() func(gl routing.GameLog) pubsub.Acktype {
	return func(gl routing.GameLog) pubsub.Acktype {
		defer fmt.Printf("> ")
		err := gamelogic.WriteLog(gl)
		if err != nil {
			fmt.Printf("error: %s\n", err)
			return pubsub.NackRequeue
		}
		return pubsub.Ack
	}
}