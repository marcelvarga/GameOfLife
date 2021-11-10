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

const (
	dead  = 0
	alive = 255
)

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
		for j := range world[i] {
			world[i][j] = <-c.ioInput
		}
	}

	boardHeight := len(world)
	turn := 0
	for ; turn < p.Turns; turn++ {
		if p.Threads == 1 {
			world = calculateNextState(world, 0, boardHeight)
		} else {
			channels := make([]chan [][]byte, p.Threads)
			for i := range channels {
				channels[i] = make(chan [][]byte)
			}
			workerHeight := boardHeight / p.Threads

			for i := 0; i < p.Threads; i++ {
				go calculateNextState(world, i*workerHeight, (i+1)*workerHeight)
			}

			var newWorld [][]byte
			for i := 0; i < p.Threads; i++ {
				newWorld = append(newWorld, <-channels[i]...)
			}
			world = newWorld
		}
		// TODO Split work between p.Threads threads
		// Get work back
	}

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

func calculateNextState(world [][]byte, startY, endY int) [][]byte {
	height := endY - startY
	totalHeight := len(world)
	width := len(world[0])

	// New 2D that stores the next state
	newWorld := make([][]byte, height)
	for i := range newWorld {
		newWorld[i] = make([]byte, width)
		for j := range newWorld[i] {
			newWorld[i][j] = world[i+startY][j]
		}
	}

	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			newWorld[i][j] = newCellValue(world, i, j, totalHeight, width)
		}
	}

	return newWorld
}

// Function used to wrap around the closed domain board
// Uses optimization for the modulo operation if n is a power of two
func wrap(x, n int) int {
	x += n
	if n != 0 && (n&(n-1)) == 0 {
		return x & (n - 1)
	}
	return x % n
}

func newCellValue(world [][]byte, y int, x int, rows int, cols int) byte {
	aliveNeighbours := 0

	// Iterate through the neighbours and count how many of them are alive
	for i := y - 1; i <= y+1; i++ {
		for j := x - 1; j <= x+1; j++ {
			if !(i == y && j == x) {
				if world[wrap(i, rows)][wrap(j, cols)] == alive {
					aliveNeighbours++
				}
			}
		}
	}

	if world[y][x] == alive {
		if aliveNeighbours < 2 {
			return dead
		}
		if (aliveNeighbours == 2) || aliveNeighbours == 3 {
			return alive
		}
		if aliveNeighbours > 3 {
			return dead
		}
	}
	if aliveNeighbours == 3 {
		return alive
	}
	return dead
}

func calculateAliveCells(world [][]byte) []util.Cell {
	aliveCells := make([]util.Cell, 0)
	for i := range world {
		for j := range world[i] {
			if world[i][j] == alive {
				aliveCells = append(aliveCells, util.Cell{X: j, Y: i})
			}
		}
	}
	return aliveCells
}
