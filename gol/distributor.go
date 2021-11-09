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

//Calculates the neighbours of a given cell
/*
func neighbours(world [][]byte,i int,j int,n int,m int) int{
s:=0
for x:= i-1;x<=i+1;x++{
	for y:=j-1;y<=j+1;y++{
		if !(i==x && j==y){
			if world[(x+n)%n][(y+m)%m]==255{
				s++
			}
		}
	}
 }
return s
}*/

//Decides the state of a cell
/*
func deadOrAlive (s int, world [][]byte,i int,j int) byte{

	var aliveness byte=0
	switch {
	case s>3 && world[i][j] == 255 :
		aliveness = 0
	case s<2 && world[i][j] == 255 :
		aliveness = 0
	case s==3 && world[i][j] == 0 :
		aliveness = 255
	case (s==2 || s==3) && world[i][j] == 255 :
		aliveness = 255
	}
	return aliveness
}*/

// GoL Algorithm implementation
/*
func GoL(world,newWorld [][]byte) [][]byte{
	n := len(world)
	m :=len(world[0])
	//Instantiate the new world with the same values as the old one
	for i:=range world{
		for j := range world[i]{
			newWorld[i][j] = world[i][j]
		}
	}
    //Iterates through the 2-D slice, calculates the neighbours and then decides the state of the cell
	for i:= 0; i<n; i++{
		for j := 0; j<m; j++{
			s := neighbours(world,i,j,n,m)
			newWorld[i][j] =deadOrAlive(s,world,i,j)
		}
	}
	return newWorld
}*/

//Returns a slice of the Alive Cells as (x,y) coordinates
func calculateAliveCells(world [][]byte) []util.Cell {
	cells := make([]util.Cell, 0)
	n := len(world)
	m := len(world[0])
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			if world[i][j] == 255 {
				cells = append(cells, util.Cell{X: j, Y: i})
			}
		}
	}
	return cells
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioCommand <- 1
	c.ioFilename <- filename

	// TODO: Create world 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}
	for i := range world {
		for j := range world[i] {
			world[i][j] = <-c.ioInput
		}
	}
	newWorld := make([][]byte, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}
	turn := 0
	// TODO: Execute all turns of the Game of Life.
	for ; turn < p.Turns; turn++ {
		// Split work between p.Threads threads
		// Get work back
		// TODO: Use bitshift instead of modulo
		// n % 2^i = n & (2^i - 1)

		//Update the current world to be the new one returned by the GoL function
		//world = GoL(world,newWorld)
	}
	// TODO: Report the final state using FinalTurnCompleteEvent.
	//TODO: Look at the event.go file and see how the interface is implemented by different structs
	Alive := calculateAliveCells(world)
	FinalTurn := FinalTurnComplete{CompletedTurns: p.Turns,
		Alive: Alive}
	//Send the final state on the events channel
	c.events <- FinalTurn
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
