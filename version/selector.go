package version

var SelectFromBeginning = Selector{From: 0}

type Selector struct {
	From uint64
}
