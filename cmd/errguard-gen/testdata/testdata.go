package testdata

import (
	bbb "bytes"
	"context"
	"time"
)

//go:generate errguard-gen Service

type Service interface {
	DoSomething(context.Context, *DoSomethingInput) (*DoSomethingOutput, error)
	NoArgs() error
	ExternPackage(time.Time) (*bbb.Buffer, error)
}

type DoSomethingInput struct {
}

type DoSomethingOutput struct {
}
