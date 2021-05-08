package eventually

// Command is a Message representing an action being performed by something
// or somebody.
//
// In order to enforce this concept, it is suggested to name Command types
// using "present tense".
type Command Message
