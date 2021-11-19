package gol

import (
	"fmt"
	"net/rpc"
	"time"
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

var x = true

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

func makeCall(server rpc.Client, world [][]byte, p Params, c distributorChannels) [][]byte {
	req := Request{
		InitialWorld: world,
		P:            p,
		Events:       c.events,
	}
	resp := new(Response)
	err := server.Call(WorldEvolution, req, resp)

	if err != nil {
		panic(err)
	}
	return resp.OutputWorld
}
func requestAliveCells(server rpc.Client, ticker <-chan time.Time, c distributorChannels, stop chan bool) {
	req := RequestAliveCells{}
	res := new(ReportAliveCells)
	done := make(chan *rpc.Call, 1)
	for {
		select {
		case <-stop:
			return
		case <-ticker:
			server.Go(AliveCellsEvent, req, res, done)
			<-done
			c.events <- res.AliveCellsCountEv
		default:

		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
// Passes keypresses to dealWithKey
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	ticker := time.Tick(2 * time.Second)

	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	initialWorld := generateBoard(p, c)
	world := initialWorld

	server, err := rpc.Dial("tcp", p.Server)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Dialed successfully")
	defer server.Close()
	stop := make(chan bool)
	go requestAliveCells(*server, ticker, c, stop)
	world = makeCall(*server, world, p, c)
	stop <- true
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
