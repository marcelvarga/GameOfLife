package gol

var WorldEvolution = "SecretGolOperations.Evolve"

type Response struct {
	initialWorld [][]byte
}

type Request struct {
	outputWorld [][]byte
}
