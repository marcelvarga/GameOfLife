package gol

import (
	"fmt"
	"net/rpc"
	//"uk.ac.bris.cs/gameoflife/server"
	"uk.ac.bris.cs/gameoflife/util"
)

type DistributorChannels struct {
	Events    chan<- Event
	ioCommand chan<- ioCommand
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
func makeCall(client rpc.Client, world [][]byte, p Params, c DistributorChannels){
	request := Request{InitialWorld: world,
						P: p,
						C: c}
	response := new(Response)
	fmt.Println("inside the call function")
	client.Call(WorldEvolution, request, response)
	world = response.OutputWorld
}

// distributor divides the work between workers and interacts with other goroutines.
// Passes keypresses to dealWithKey
func distributor(p Params, c DistributorChannels, keyPresses <-chan rune) {
	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- filename
	world := generateBoard(p,c)
	server :="127.0.0.1:8030"
	client, _ := rpc.Dial("tcp", server)
	fmt.Println("dialed successfully")
	defer client.Close()
	makeCall(*client,world,p,c)

	quit(world, c, p.Turns)


}
func generateBoard(p Params,c DistributorChannels) [][]byte{
	world := make([][]byte, p.ImageHeight)

	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
		for j := range world[i] {
			world[i][j] = <-c.ioInput
			if world[i][j] == Alive {
				c.Events <- CellFlipped{
					Cell:           util.Cell{X: j, Y: i},
					CompletedTurns: 0,
				}
			}
		}
	}
	return world
}
// Closes channels and sends Quitting event to SDL
func quit(world [][]byte, c DistributorChannels, turn int) {

	c.Events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.Events)
}
