package main

import (
	"fmt"
	"log"
	"time"

	"github.com/JA50N14/learn-pub-sub-starter/internal/gamelogic"
	"github.com/JA50N14/learn-pub-sub-starter/internal/pubsub"
	"github.com/JA50N14/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("could not connect to rabbitmq server: %v", err)
	}
	defer conn.Close()

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}

	gs := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		string(routing.ExchangePerilTopic),
		string(routing.ArmyMovesPrefix)+"."+gs.GetUsername(),
		string(routing.ArmyMovesPrefix)+".*",
		pubsub.SimpleQueueTransient,
		handlerMoves(gs, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to army_moves: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		string(routing.ExchangePerilTopic),
		string(routing.WarRecognitionsPrefix),
		string(routing.WarRecognitionsPrefix)+".*",
		pubsub.SimpleQueueDurable,
		handlerWar(gs, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to war declarations: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.GetUsername(),
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "move":
			mv, err := gs.CommandMove(words)
			if err != nil {
				fmt.Println(err)
				continue
			}
			//publish the move
			err = pubsub.PublishJSON(publishCh, routing.ExchangePerilTopic, string(routing.ArmyMovesPrefix)+"."+mv.Player.Username, mv)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				continue
			}
			fmt.Printf("Moved %v units to %s\n", len(mv.Units), mv.ToLocation)
		case "spawn":
			err := gs.CommandSpawn(words)
			if err != nil {
				fmt.Println(err)
				continue
			}
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			//TODO: publish n malicious logs
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Unknown command")
		}
	}
}

func publishGameLog(publishCh *amqp.Channel, username, msg string) error {
	return pubsub.PublishGob(
		publishCh,
		string(routing.ExchangePerilTopic),
		routing.GameLogSlug+"."+username,
		routing.GameLog{
			Username:    username,
			CurrentTime: time.Now(),
			Message:     msg,
		},
	)
}

//Pause Flow:
// 1. client/main.go is where individual players connect to the game
// 2. Each client creates a connection to the RabbitMQ server
// 3. Each player gets their own unique queue (e.g., pause.suntzu, pause.napoleon, etc.)
// 4. All players use the same exchange (peril_direct)
// 5. All players bind their queue with the same routing key (pause)
// 6. When one player pauses, all other players receive the pause message in their personal queues
// Small clarification on the exchange type:

// You're using a direct exchange (peril_direct). In a direct exchange, messages are routed to queues whose binding key exactly matches the routing key of the message.

// So when someone publishes a pause message:

// They publish to the peril_direct exchange with routing key pause
// The exchange looks at all queues bound with the key pause
// All those queues (one per player) receive a copy of the message
// Why each player needs their own queue:

// Each player has their own transient queue so they can independently consume pause messages. If they all shared one queue, only one player would get each message (queues distribute messages round-robin to consumers). With separate queues, every player gets every pause message.

// Please correct any mistakes in my understanding as to the control flow that we programmed. In the server main.go we created a publish channel and when the game gets paused, we call PublishJSON, which sends the pause message to the exchange. We also passed a key, so every queue that is binded to that exchange with the corresponding key will receive the message. For every instance of the client main.go, we created a queue that is binded to the same pause exchange within the server main.go, which we did this through the SubscribeJSON function. Within SubscribeJSON, our client creates a channel and binds the queue to the pause exchange, then runs a go routine, which blocks until it receives a pause message. When the go routine receives a pause message, it unmarshalls the body and then calls the handler funciton we passed into SubscribeJSON, which changes the the game state to a pause. When the game state is in the pause state, all functions associated with move, spawn, etc. will not allow it to occur. Are there any flaws in my understanding?

// Boots
// You’ve got the big picture right. A few precise tweaks:

// The server’s PublishJSON sends to an exchange with a routing key; any queue bound to that exchange with a matching binding key gets the message. Correct.

// On the client, SubscribeJSON should:

// Declare/bind the queue to the same exchange with the key. Correct.
// Call Channel.Consume to get a deliveries channel. Correct.
// Start a goroutine that ranges over deliveries, unmarshals into T, calls the handler, then Ack(false). Correct.
// Minor nuance: the goroutine itself doesn’t “block”; ranging over the deliveries channel blocks between messages. The program keeps running (other goroutines keep working).

// Each client instance typically uses a distinct queue name (e.g., pause.username), so messages are delivered to the intended user, not broadcast to all clients unless you bind multiple queues with the same key. Sounds like you did that.

// Paused behavior: your handler should update state via GameState.HandlePause. Then command handlers (move/spawn/etc.) should consult that state and reject actions. Correct.

// So each client has a game state, which is manipulated by the server when it sends a pause message?

// Boots
// Mostly. The server doesn’t directly manipulate client state—it just publishes pause/resume messages. Each client:

// Subscribes to its own queue (e.g., pause.username).
// Consumes messages and runs its local handler.
// The handler updates the client’s local GameState (e.g., HandlePause/HandleResume).
// So the server “signals,” and each client chooses to update its own state based on that signal.
