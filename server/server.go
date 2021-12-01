package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/util"
)

type GolOperations struct{}

var mutex sync.Mutex
var world [][]byte
var turn = 0

var pause = make(chan bool)
var kill = make(chan bool)
var quit = make(chan bool)
var queue = make([]gol.CellFlipped, 0)

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&GolOperations{})
	listener, err := net.Listen("tcp", ":"+*pAddr)

	if err != nil {
		panic(err)
	}

	fmt.Println("Server is up and running. Listening on port " + *pAddr)
	go rpc.Accept(listener)
	<-kill
	time.Sleep(2 * time.Second)
	defer listener.Close()

}

func (golOperation *GolOperations) Evolve(req gol.Request, res *gol.Result) (err error) {
	if req.InitialWorld == nil {
		fmt.Println("Empty message")
		return
	}
	fmt.Println("Got World")
	fmt.Println(req.P)

	fmt.Println("Proceeding to do the evolution")
	world = req.InitialWorld
	boardHeight := len(world)

	turn = 0
	for turn < req.P.Turns {
		var newWorld [][]byte
		var workerHeight int

		threads := req.P.Threads

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

		select {
		case <-quit:
			fmt.Println("Quitting connection to client")
			mutex.Lock()
			turn = 0
			mutex.Unlock()
			return
		case <-pause:
			fmt.Println("Receive the pause in evolve, waiting for the second one")
			<-pause
			fmt.Println("Got the second pause, going to continue the evolution of the game")
		default:

		}
		mutex.Lock()
		world = newWorld
		turn++
		mutex.Unlock()
	}

	res.OutputWorld = world
	fmt.Printf("Finished evolution of %d turns and sent response\n", turn)
	return
}

func (golOperation *GolOperations) ReportAliveCellsCount(req gol.RequestAliveCells, res *gol.ReportAliveCells) (err error) {
	mutex.Lock()
	aliveCells := len(calculateAliveCells(world))
	turnCount := turn
	mutex.Unlock()

	res.AliveCellsCountEv = gol.AliveCellsCount{
		CellsCount:     aliveCells,
		CompletedTurns: turnCount,
	}
	return
}
func (golOperation *GolOperations) GetFlip(req gol.RequestCellFlip, res *gol.GetCellFlip) (err error) {
	if len(queue) > 0 {
		res.Flip = queue[0]
		queue = queue[1:]
	}
	return
}
func (golOperation *GolOperations) Kill(req gol.RequestForKey, res *gol.ReceiveFromKey) (err error) {

	kill <- true
	quit <- true
	return
}
func (golOperation *GolOperations) Pause(req gol.RequestForKey, res *gol.ReceiveFromKey) (err error) {
	fmt.Println("Pausing")
	mutex.Lock()
	pause <- true
	res.Turn = turn
	res.ScreenshotWorld = world
	mutex.Unlock()
	return
}
func (golOperation *GolOperations) Resume(req gol.RequestForKey, res *gol.ReceiveFromKey) (err error) {
	fmt.Println("execution will resume now")
	pause <- false
	return
}
func (golOperation *GolOperations) Save(req gol.RequestForKey, res *gol.ReceiveFromKey) (err error) {
	mutex.Lock()
	res.Turn = turn
	res.ScreenshotWorld = world
	mutex.Unlock()
	return
}

func (golOperation *GolOperations) Quit(req gol.RequestForKey, res *gol.ReceiveFromKey) (err error) {
	mutex.Lock()
	res.Turn = turn
	res.ScreenshotWorld = world
	quit <- true
	mutex.Unlock()
	return
}
func calculateAliveCells(world [][]byte) []util.Cell {
	aliveCells := make([]util.Cell, 0)
	for i := range world {
		for j := range world[i] {
			if world[i][j] == gol.Alive {
				aliveCells = append(aliveCells, util.Cell{X: j, Y: i})
			}
		}
	}
	return aliveCells
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

// Computes the value of a particular cell based on its neighbours
// Sends CellFlipped events to notify the GUI about a change of state of a cell
func newCellValue(world [][]byte, y int, x int, rows int, cols int) byte {
	aliveNeighbours := 0

	// Iterate through the neighbours and count how many of them are alive
	for i := y - 1; i <= y+1; i++ {
		for j := x - 1; j <= x+1; j++ {
			if !(i == y && j == x) {
				if world[wrap(i, rows)][wrap(j, cols)] == gol.Alive {
					aliveNeighbours++
				}
			}
		}
	}

	if world[y][x] == gol.Alive {
		if aliveNeighbours < 2 || aliveNeighbours > 3 {
			/*queue= append(queue, gol.CellFlipped{
				Cell:           util.Cell{X: x, Y: y},
				CompletedTurns: turn,
			})*/
			return gol.Dead
		}
		if (aliveNeighbours == 2) || aliveNeighbours == 3 {
			return gol.Alive
		}
	} else {
		if aliveNeighbours == 3 {
			/*queue= append(queue, gol.CellFlipped{
				Cell:           util.Cell{X: x, Y: y},
				CompletedTurns: turn,
			})*/
			return gol.Alive
		}
	}
	return gol.Dead
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
