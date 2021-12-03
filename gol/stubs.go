package gol

var WorldEvolution = "GolOperations.Evolve"
var AliveCellsEvent = "GolOperations.ReportAliveCellsCount"
var Save = "GolOperations.Save"
var Quit = "GolOperations.Quit"
var Kill = "GolOperations.Kill"
var Pause = "GolOperations.Pause"
var Resume = "GolOperations.Resume"

type Request struct {
	InitialWorld [][]byte
	P            Params
	Events       chan<- Event
}

type Result struct {
	OutputWorld [][]byte
	Turn        int
}

type RequestAliveCells struct {
}

type ReportAliveCells struct {
	AliveCellsCountEv AliveCellsCount
}

type RequestForKey struct {
}
type ReceiveFromKey struct {
	ScreenshotWorld [][]byte
	Turn            int
}
