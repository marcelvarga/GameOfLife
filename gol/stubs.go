package gol

var WorldEvolution = "SecretGolOperations.Evolve"

type Request struct {

	InitialWorld [][]byte
	P            Params
	C            DistributorChannels
}

type Response struct {
	OutputWorld [][]byte
}

