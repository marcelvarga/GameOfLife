package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/gol"
)

type GolOperations struct{}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&GolOperations{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	fmt.Println("Server is up and running. Listening on port " + *pAddr)
	rpc.Accept(listener)
}

func (golOperation *GolOperations) Evolve(req gol.Request, res *gol.Response) (err error) {
	if req.InitialWorld == nil {
		fmt.Println("Empty message")
		return
	}
	fmt.Println("Got World")
	fmt.Println(req.P)

	turn := 0
	fmt.Println("Proceeding to do the evolution")
	world := req.InitialWorld
	boardHeight := len(world)
	//ticker := time.Tick(2 * time.Second)
	for ; turn < req.P.Turns; turn++ {
		if turn%10 == 0 {
			fmt.Printf("Processing turn %d\n", turn)
		}
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
		world = newWorld
	}

	res.OutputWorld = world
	fmt.Printf("Finished evolution of %d turns and sent response\n", turn)
	return
}

// If the ticker signalises that 2 seconds have passed, send an AliveCellsCount event down the c.events channel containing the number of alive cells
/*func reportAliveCells(world [][]byte, ticker <-chan time.Time, c gol.DistributorChannels, turn int) {
	select {
	case <-ticker:
		aliveCells := len(calculateAliveCells(world))

		c.events <- gol.AliveCellsCount{
			CellsCount:     aliveCells,
			CompletedTurns: turn,
		}
	default:
		return
	}
}*/

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
			return gol.Dead
		}
		if (aliveNeighbours == 2) || aliveNeighbours == 3 {
			return gol.Alive
		}
	} else {
		if aliveNeighbours == 3 {
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
