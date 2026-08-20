package entity

type Pair struct {
	ID        int
	Base      Symbol
	Quote     Symbol
	LotSize   int64
	PriceTick int64
}

type Symbol struct {
	ID        int
	Code      string
	Precision int
}
