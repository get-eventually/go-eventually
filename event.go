package eventually

// Event is a Message representing some Domain information that has happened
// in the past, which is of vital information to the Domain itself.
//
// Event type names should be phrased in the past tense, to enforce the notion
// of "information happened in the past".
type Event Message
