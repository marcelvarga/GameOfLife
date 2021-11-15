package gol

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
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
	actionRequest := make(chan int)
	resumeCh := make(chan bool)
	world := make([][]byte, p.ImageHeight)

	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
		for j := range world[i] {
			world[i][j] = <-c.ioInput
			if world[i][j] == alive {
				c.events <- CellFlipped{
					Cell:           util.Cell{X: j, Y: i},
					CompletedTurns: 0,
				}
			}
		}
	}

	boardHeight := len(world)
	turn := 0
	turnRequest := make(chan int)
	ticker := time.Tick(2 * time.Second)
	go dealWithKey(keyPresses, turnRequest, actionRequest, resumeCh)

	for ; turn < p.Turns; turn++ {
		var newWorld [][]byte
		var workerHeight int

		threads := p.Threads

		channels := make([]chan [][]byte, threads)
		for i := range channels {
			channels[i] = make(chan [][]byte)
		}
		workerHeight = boardHeight / threads
		i := 0
		for ; i < threads-1; i++ {
			go worker(world, i*workerHeight, (i+1)*workerHeight, channels[i], c, turn)
		}
		go worker(world, i*workerHeight, boardHeight, channels[i], c, turn)
		for i := 0; i < threads; i++ {
			newWorld = append(newWorld, <-channels[i]...)
		}

		reportAliveCells(world, ticker, c, turn)

		requestedAction := actOrReturn(actionRequest)
		resume := true

		if requestedAction == pause {
			turnRequest <- turn
			secondAction := noAction
			for secondAction != pause && secondAction != quitAndSave {
				secondAction = <-actionRequest
				if secondAction == save {
					screenShot(world, c, filename, turn)
				} else {
					resume = <-resumeCh
					if resume == false {
						secondAction = quitAndSave
					}
				}
			}
		}
		if requestedAction == save {
			screenShot(world, c, filename, turn)
			resume = <-resumeCh
		}
		if requestedAction == quitAndSave || resume == false {
			screenShot(world, c, filename, turn)
			quit(world, c, turn)
			return
		}

		world = newWorld
		complete := TurnComplete{CompletedTurns: turn}
		c.events <- complete

	}

	screenShot(world, c, filename, turn)
	quit(world, c, turn)

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

// Helper function that checks if actionCh has any actions in it
// Returns noAction otherwise
func actOrReturn(actionCh chan int) int {
	select {
	case requestedAction := <-actionCh:
		return requestedAction
	default:
		return noAction
	}
}

// Runs concurrently with distributor and deals with keypresses
// Calls keyPressesOnPause if the execution is paused
func dealWithKey(keyPresses <-chan rune, turnRequest, actionCh chan int, resumeCh chan bool) {
	var turn int
	for {
		select {
		case key := <-keyPresses:
			switch key {
			case sdl.K_q:
				fmt.Println("Saving board and quitting")
				actionCh <- quitAndSave
			case sdl.K_s:
				actionCh <- save
				resumeCh <- true
			case sdl.K_p:
				actionCh <- pause
				turn = <-turnRequest
				fmt.Printf("Completed turns %d     Paused\n", turn)
				keyPressesOnPause(keyPresses, resumeCh, actionCh)
			}

		}
	}

}

func keyPressesOnPause(keyPresses <-chan rune, resumeCh chan bool, actionCh chan int) {
	for {
		select {
		case key := <-keyPresses:
			switch key {
			case sdl.K_q:
				fmt.Println("Saving board and quitting")
				actionCh <- quitAndSave
				resumeCh <- false
				return
			case sdl.K_s:
				actionCh <- save
			case sdl.K_p:
				fmt.Println("Continuing...")
				actionCh <- pause
				resumeCh <- true
				return
			}
		}
	}
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

// Outputs world state to pgm file
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

// If the ticker signalises that 2 seconds have passed, send an AliveCellsCount event down the c.events channel containing the number of alive cells
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
func worker(world [][]byte, startY, endY int, out chan<- [][]byte, c distributorChannels, turn int) {
	partialWorld := calculateNextState(world, startY, endY, c, turn)
	out <- partialWorld
}

// Makes a transition between the Y coordinates given and returns a 2D slice containing the updated cells
func calculateNextState(world [][]byte, startY, endY int, c distributorChannels, turn int) [][]byte {
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
			newWorld[i][j] = newCellValue(world, i+startY, j, totalHeight, width, c, turn)
		}
	}

	return newWorld
}

// Computes the value of a particular cell based on its neighbours
// Sends CellFlipped events to notify the GUI about a change of state of a cell
func newCellValue(world [][]byte, y int, x int, rows int, cols int, c distributorChannels, turn int) byte {
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
		if aliveNeighbours < 2 || aliveNeighbours > 3 {
			c.events <- CellFlipped{
				Cell:           util.Cell{X: x, Y: y},
				CompletedTurns: turn,
			}
			return dead
		}
		if (aliveNeighbours == 2) || aliveNeighbours == 3 {
			return alive
		}
	}
	if aliveNeighbours == 3 {
		c.events <- CellFlipped{
			Cell:           util.Cell{X: x, Y: y},
			CompletedTurns: turn,
		}
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
