package version

// Version is the type to specify Event Stream versions.
// Versions should be starting from 1, as they represent the length of a single Event Stream.
type Version uint32

// SelectFromBeginning is a Selector value that will return all Domain Events in an Event Stream.
var SelectFromBeginning = Selector{From: 0}

// Selector specifies which slice of the Event Stream to select when streaming Domain Events
// from the Event Store.
type Selector struct {
	From Version
}
