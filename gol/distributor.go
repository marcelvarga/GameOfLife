package gol

import (
	"fmt"

	"net/rpc"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

const (
	Dead  = 0
	Alive = 255
)
const (
	noAction    = 0
	pause       = 1
	save        = 2
	quitAndSave = 3
)

func makeCall(client rpc.Client, world [][]byte, p Params, c distributorChannels) [][]byte {
	req := Request{
		InitialWorld: world,
		P:            p,
		Events:       c.events,
	}
	resp := new(Response)
	err := client.Call(WorldEvolution, req, resp)
	if err != nil {
		panic(err)
	}
	return resp.OutputWorld
}

// distributor divides the work between workers and interacts with other goroutines.
// Passes keypresses to dealWithKey
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	initialWorld := generateBoard(p, c)
	world := initialWorld

	client, err := rpc.Dial("tcp", p.Server)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Dialed successfully")
	defer client.Close()
	world = makeCall(*client, world, p, c)
	quit(world, c, p.Turns)
}
func quit(world [][]byte, c distributorChannels, turn int) {
	alive := calculateAliveCells(world)
	finalTurn := FinalTurnComplete{CompletedTurns: turn, Alive: alive}

	//Send the final state on the events channel
	c.events <- finalTurn
	c.events <- StateChange{turn, Quitting}
	close(c.events)

}
func calculateAliveCells(world [][]byte) []util.Cell {
	aliveCells := make([]util.Cell, 0)
	for i := range world {
		for j := range world[i] {
			if world[i][j] == Alive {
				aliveCells = append(aliveCells, util.Cell{X: j, Y: i})
			}
		}
	}
	return aliveCells
}
func generateBoard(p Params, c distributorChannels) [][]byte {
	world := make([][]byte, p.ImageHeight)

	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
		for j := range world[i] {
			world[i][j] = <-c.ioInput
			if world[i][j] == Alive {
				c.events <- CellFlipped{
					Cell:           util.Cell{X: j, Y: i},
					CompletedTurns: 0,
				}
			}
		}
	}
	return world
}
