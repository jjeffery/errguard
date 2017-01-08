package testdata

import "context"

//go:generate errguard-gen --type Service

type Service interface {
	DoSomething(context.Context, *DoSomethingInput) (*DoSomethingOutput, error)
	Varargs(<-chan string) error
}

type DoSomethingInput struct {
}

type DoSomethingOutput struct {
}
