package gol

var WorldEvolution = "GolOperations.Evolve"

type Request struct {
	InitialWorld [][]byte
	P            Params
}

type Response struct {
	//OutputWorld [][]byte
	Message string
}
