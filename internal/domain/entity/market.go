package entity

type Pair struct {
	ID    int
	Base  Symbol
	Quote Symbol
}

type Symbol struct {
	ID   int
	Code string
}
