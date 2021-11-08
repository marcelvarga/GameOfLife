package gol

import "fmt"

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
	c.ioFilename <- filename
	// TODO: Create a 2D slice to store the world.
	a := make([][]byte, p.ImageHeight)
	for i := range a {
		a[i] = make([]byte, p.ImageWidth)
	}

	turn := 0
	// TODO: Execute all turns of the Game of Life.
	for ; turn < p.Turns; turn++ {
		// Split work between p.Threads threads
		// Get work back
		// TODO: Use bitshift instead of modulo
		// n % 2^i = n & (2^i - 1)
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
