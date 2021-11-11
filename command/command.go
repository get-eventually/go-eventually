package command

import "github.com/get-eventually/go-eventually"

// Command is a Message representing an action being performed by something
// or somebody.
//
// In order to enforce this concept, it is suggested to name Command types
// using "present tense".
type Command eventually.Message

// Type is a marker interface that represents the type of a Domain Command.
type Type eventually.Payload
