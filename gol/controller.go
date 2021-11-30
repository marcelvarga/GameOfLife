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

var x = true

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

func makeCall(server rpc.Client, world [][]byte, p Params, c distributorChannels) [][]byte {
	req := Request{
		InitialWorld: world,
		P:            p,
		Events:       c.events,
	}
	resp := new(Response)
	err := server.Call(WorldEvolution, req, resp)

	if err != nil {
		panic(err)
	}
	return resp.OutputWorld
}
func requestAliveCells(server rpc.Client, ticker <-chan time.Time, c distributorChannels, stop chan bool,pause chan bool) {
	req := RequestAliveCells{}
	res := new(ReportAliveCells)
	done := make(chan *rpc.Call, 1)
	for {
		select {
		case <-pause:
			<-ticker
			<-pause
		case <-stop:
			return
		case <-ticker:
			server.Go(AliveCellsEvent, req, res, done)
			<-done
			c.events <- res.AliveCellsCountEv
		default:

		}
	}
}
func getFlip(server rpc.Client,c distributorChannels,stop chan bool){
    time.Sleep(1*time.Second)
	req := RequestCellFlip{}
	res := new(GetCellFlip)
	done := make(chan *rpc.Call,1)
	for{
		select {
		case <-stop:
			return
			default:
				server.Go(SendFLip,req,res,done)
				<-done
				c.events <-res.Flip
		}
	}
}
func dealWithKey(server *rpc.Client,keyPresses <-chan rune,c distributorChannels,filename string,pause chan bool) {
    req := RequestForKey{}
	res := new(ReceiveFromKey)
	done := make(chan *rpc.Call,1)
	for {
		select {
		case key := <-keyPresses:
			switch key {
			case sdl.K_q:
            server.Go(Quit,req,res,done)
			<-done
			fmt.Println(res.Turn)
			quit(res.ScreenshotWorld,c,res.Turn)
			closeClient(server)
			return
			case sdl.K_s:
            server.Go(Save,req,res,done)
			<-done
			screenShot(res.ScreenshotWorld,c,filename,res.Turn)
			case sdl.K_k:
				server.Go(Quit,req,res,done)
				<-done
				var turn = res.Turn
				var ssworld = res.ScreenshotWorld
				server.Go(Shut,req,res,done)
				//<-done
				fmt.Println("returned from rpc")
				//fmt.Println(res.Turn)
				screenShot(res.ScreenshotWorld,c,filename,res.Turn)
				quit(ssworld,c,turn)
				 closeClient(server)
			case sdl.K_p:
				pause <-true
				server.Go(Pause,req,res,done)
				<- done
				fmt.Println(res.Turn)
				screenShot(res.ScreenshotWorld,c,filename,res.Turn)
				dealWithPausing(server,keyPresses,req,res,pause)
				server.Go(ContinueFromPause,req,res,done)
				<-done

			}

		}
	}

}
func dealWithPausing(server *rpc.Client, keyPresses <-chan rune,req RequestForKey,res *ReceiveFromKey,pause chan bool){
	//done := make(chan *rpc.Call,1)
   for{
	   select {
	   case key := <-keyPresses:
		   switch key {
		   case sdl.K_p:
			   fmt.Println("Continuing")
			   pause <-false
			   return
		   default:


		   }
	   }
   }
}
// distributor divides the work between workers and interacts with other goroutines.
// Passes keypresses to dealWithKey
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	ticker := time.Tick(2 * time.Second)

	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	initialWorld := generateBoard(p, c)
	world := initialWorld

	server, err := rpc.Dial("tcp", p.Server)
	if err != nil {
		fmt.Println(err)
	}

	stop := make(chan bool)
	pause := make(chan bool)
	//stopFlip :=make(chan bool)
	go dealWithKey(server,keyPresses,c,filename,pause)
	go requestAliveCells(*server, ticker, c, stop,pause)
	//go getFlip(*server,c,stopFlip)
	world = makeCall(*server, world, p, c)
	stop <- true
	//stopFlip <-true
	screenShot(world,c,filename,p.Turns)
	quit(world, c, p.Turns)
    closeClient(server)
}
func closeClient(server *rpc.Client ){
	server.Close()
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
