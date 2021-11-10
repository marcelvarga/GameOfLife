package gol

import (
	"fmt"
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

func mod(x, m int) int {
	return x & (m - 1)

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	// TODO: Create world 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
		for j := range world[i] {
			world[i][j] = <-c.ioInput
		}
	}

	//newWorld := make([][]byte, p.ImageHeight)
	//for i := range newWorld {
	//	newWorld[i] = make([]byte, p.ImageWidth)
	//}

	turn := 0
	// TODO: Execute all turns of the Game of Life.
	for ; turn < p.Turns; turn++ {

		world = calculateNextState(world)
		// Split work between p.Threads threads
		// Get work back

		//Update the current world to be the new one returned by the GoL function
		//world = GoL(world,newWorld)
	}
	//TODO: Report the final state using FinalTurnCompleteEvent.
	//TODO: Look at the event.go file and see how the interface is implemented by different structs

	alive := calculateAliveCells(world)
	finalTurn := FinalTurnComplete{CompletedTurns: turn, Alive: alive}

	//Send the final state on the events channel
	c.events <- finalTurn
	// Make sure that the Io has finished output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateNextState(world [][]byte) [][]byte {

	var n = len(world)
	var m = len(world[0])

	// New 2D that stores the next state
	newWorld := make([][]byte, n)
	for i := range newWorld {
		newWorld[i] = make([]byte, m)
		for j := range newWorld[i] {
			newWorld[i][j] = world[i][j]
		}
	}

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			newWorld[i][j] = newCellValue(world, i, j, n, m)
		}
	}

	return newWorld
}

func newCellValue(world [][]byte, x int, y int, rows int, cols int) byte {

	aliveNeighbours := 0

	// Iterate through the neighbours
	for i := x - 1; i <= x+1; i++ {
		for j := y - 1; j <= y+1; j++ {
			if !(i == x && j == y) {
				if world[(i+rows)%rows][(j+cols)%cols] == 255 {
					aliveNeighbours++
				}
			}
		}
	}

	if world[x][y] == 255 {
		if aliveNeighbours < 2 {
			return 0
		}
		if (aliveNeighbours == 2) || aliveNeighbours == 3 {
			return 255
		}
		if aliveNeighbours > 3 {
			return 0
		}
	}
	if aliveNeighbours == 3 {
		return 255
	}
	return 0
}

func calculateAliveCells(world [][]byte) []util.Cell {
	aliveCells := make([]util.Cell, 0)
	for i := range world {
		for j := range world[i] {
			if world[i][j] == 255 {
				aliveCells = append(aliveCells, util.Cell{X: j, Y: i})
			}
		}
	}
	return aliveCells
}
