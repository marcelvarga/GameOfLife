package gol

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
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

const (
	Dead  = 0
	Alive = 255
)

func makeCall(server *rpc.Client, world [][]byte, p Params, c distributorChannels) ([][]byte, int) {
	req := Request{
		InitialWorld: world,
		P:            p,
		Events:       c.events,
	}
	res := new(Result)
	_ = server.Call(WorldEvolution, req, res)
	return res.OutputWorld, res.Turn
}

func requestAliveCells(server *rpc.Client, ticker <-chan time.Time, c distributorChannels, stop chan bool, pause chan bool, turns int) {
	req := RequestAliveCells{}
	res := new(ReportAliveCells)
	for {
		select {
		case <-pause:
			<-ticker
			<-pause
		case <-stop:
			return
		case <-ticker:
			err := server.Call(AliveCellsEvent, req, res)
			if err != nil {
				return
			}
			if res.AliveCellsCountEv.CompletedTurns >= turns {
				select {
				case <-stop:
				}
				return
			}
			c.events <- res.AliveCellsCountEv
		}
	}
}

func dealWithKey(server *rpc.Client, keyPresses <-chan rune, c distributorChannels, filename string, pause chan bool) {
	req := RequestForKey{}
	res := new(ReceiveFromKey)
	for {
		select {
		case key := <-keyPresses:
			switch key {
			case sdl.K_q:
				err := server.Call(Quit, req, res)
				handleErr(err)
				return
			case sdl.K_s:
				err := server.Call(Save, req, res)
				handleErr(err)

				screenShot(res.ScreenshotWorld, c, filename, res.Turn)
			case sdl.K_k:
				err := server.Call(Quit, req, res)
				handleErr(err)

				_ = server.Call(Kill, req, res)

				fmt.Println("Killing server and closing")

			case sdl.K_p:
				pause <- true
				err := server.Call(Pause, req, res)
				handleErr(err)

				fmt.Println("Pausing. Completed turns: ", res.Turn)
				dealWithPause(keyPresses, pause)

				err = server.Call(Resume, req, res)
				handleErr(err)
			}

		}
	}

}
func dealWithPause(keyPresses <-chan rune, pause chan bool) {
	for {
		select {
		case key := <-keyPresses:
			switch key {
			case sdl.K_p:
				fmt.Println("Continuing")
				pause <- false
				return
			}
		}
	}
}

// controller divides the work between workers and interacts with other goroutines.
// Passes keypresses to dealWithKey
func controller(p Params, c distributorChannels, keyPresses <-chan rune) {
	ticker := time.Tick(2 * time.Second)

	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	initialWorld := generateBoard(p, c)
	world := initialWorld

	server, err := rpc.Dial("tcp", p.Server)
	handleErr(err)

	stop := make(chan bool)
	pause := make(chan bool)
	go dealWithKey(server, keyPresses, c, filename, pause)
	go requestAliveCells(server, ticker, c, stop, pause, p.Turns)
	var turn int
	world, turn = makeCall(server, world, p, c)
	stop <- true

	screenShot(world, c, filename, turn)
	quit(world, c, turn)
	err = server.Close()
	handleErr(err)
}

func quit(world [][]byte, c distributorChannels, turn int) {
	alive := calculateAliveCells(world)
	finalTurn := FinalTurnComplete{CompletedTurns: turn, Alive: alive}
	//Send the final state on the events channel
	c.events <- finalTurn
	c.events <- StateChange{turn, Quitting}
	close(c.events)

}
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
func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
