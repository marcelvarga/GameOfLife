package gol

var WorldEvolution = "GolOperations.Evolve"

type Request struct {

	InitialWorld [][]byte
	P            Params
	C            DistributorChannels
}

type Response struct {
	OutputWorld [][]byte
}

