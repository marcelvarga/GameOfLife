package gol

var WorldEvolution = "GolOperations.Evolve"
var AliveCellsEvent = "GolOperations.ReportAliveCellsCount"
var SendFLip="GolOperations.GetFLip"
var Save ="GolOperations.Saving"
var Quit ="GolOperations.Quitting"
var Shut ="GolOperations.ShutDown"
var Pause ="GolOperations.Pausing"
var ContinueFromPause ="GolOperations.Resume"
type Request struct {
	InitialWorld [][]byte
	P            Params
	Events       chan<- Event
}

type Response struct {
	OutputWorld [][]byte
}

type RequestAliveCells struct {
}

type ReportAliveCells struct {
	AliveCellsCountEv AliveCellsCount
}

type RequestCellFlip struct {

}
type GetCellFlip struct {
	Flip CellFlipped
}
type RequestForKey struct {

}
type ReceiveFromKey struct {
	ScreenshotWorld [][]byte
	Turn int
}