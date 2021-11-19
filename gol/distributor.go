package gol

import (
	"fmt"
	"net/rpc"
	//"uk.ac.bris.cs/gameoflife/gol/server"
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
		fmt.Println("Connection error")
		quit(world, c, 0)
		return
	}
	fmt.Println("Dialed successfully")
	defer client.Close()
	world = makeCall(*client, world, p)
	if &world == &initialWorld {
		fmt.Println("Pointer hasn't changed")
	}
	screenShot(world, c, filename, p.Turns)
	quit(world, c, p.Turns)
}

func makeCall(client rpc.Client, world [][]byte, p Params) [][]byte {
	req := Request{InitialWorld: world,
		P: p}
	res := new(Response)
	client.Call(WorldEvolution, req, res)
	return res.OutputWorld
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

// Closes channels and sends Quitting event to SDL
func quit(world [][]byte, c distributorChannels, turn int) {
	alive := calculateAliveCells(world)
	finalTurn := FinalTurnComplete{CompletedTurns: turn, Alive: alive}

	//Send the final state on the events channel
	c.events <- finalTurn

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

// Returns a slice with all the alive cells
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
func screenShot(world [][]byte, c distributorChannels, filename string, turn int) {
	c.ioCommand <- ioOutput
	filename = filename + fmt.Sprintf("x%v", turn)
	c.ioFilename <- filename

	for i := range world {
		for j := range world[i] {
			c.ioOutput <- world[i][j]
		}
	}
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
}
