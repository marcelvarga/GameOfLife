package gol

var WorldEvolution = "GolOperations.Evolve"
var AliveCellsEvent = "GolOperations.ReportAliveCellsCount"

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
