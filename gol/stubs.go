package gol

var WorldEvolution = "GolOperations.Evolve"

type Request struct {
	InitialWorld [][]byte
	P            Params
	Events       chan<- Event
}

type Response struct {
	OutputWorld [][]byte
}
