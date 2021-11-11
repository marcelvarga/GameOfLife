package gol

import (
	"fmt"
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

	ticker := time.Tick(2 * time.Second)
	for ; turn < p.Turns; turn++ {
		var newWorld [][]byte
		var workerHeight int

		if p.Threads == 1 {
			reportAliveCells(world, ticker, c, turn)
			world = calculateNextState(world, 0, boardHeight)
			complete := TurnComplete{CompletedTurns: turn}
			c.events <- complete

		} else {

			threads := p.Threads

			channels := make([]chan [][]byte, threads)
			for i := range channels {
				channels[i] = make(chan [][]byte)
			}
			workerHeight = boardHeight / threads
			i := 0
			for ; i < threads-1; i++ {
				go worker(world, i*workerHeight, (i+1)*workerHeight, channels[i])
			}
			go worker(world, i*workerHeight, boardHeight, channels[i])
			for i := 0; i < threads; i++ {
				newWorld = append(newWorld, <-channels[i]...)

			}
			reportAliveCells(world, ticker, c, turn)
			world = newWorld
			complete := TurnComplete{CompletedTurns: turn}
			c.events <- complete
		}
		// TODO Split work between p.Threads threads
		// Get work back

	}

	alive := calculateAliveCells(world)
	finalTurn := FinalTurnComplete{CompletedTurns: turn, Alive: alive}

	//Send the final state on the events channel
	c.events <- finalTurn
	// Make sure that the Io has finished output before exiting.
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	/*for i:=range world{
		for j:=range world[i]{
			c.ioOutput <- world[i][j]
		}
	}*/
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {

			c.ioOutput <- world[i][j]
		}
	}
	c.ioCommand <- ioCheckIdle

	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
func reportAliveCells(world [][]byte, ticker <-chan time.Time, c distributorChannels, turn int) {
	select {
	case <-ticker:
		aliveCells := len(calculateAliveCells(world))

		c.events <- AliveCellsCount{
			CellsCount:     aliveCells,
			CompletedTurns: turn,
		}
	default:
		return
	}
}

// Function used for splitting work between multiple threads
// worker makes a "calculateNextState" call
func worker(world [][]byte, startY, endY int, out chan<- [][]byte) {
	partialWorld := calculateNextState(world, startY, endY)
	out <- partialWorld
}

// Makes a transition between the Y coordinates given and returns a 2D slice containing the updated cells
func calculateNextState(world [][]byte, startY, endY int) [][]byte {
	height := endY - startY
	totalHeight := len(world)
	width := len(world)
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
			newWorld[i][j] = newCellValue(world, i+startY, j, totalHeight, width)
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

// Computes the value of a particular cell based on its neighbours
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

// Returns a slice with all the alive cells
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
