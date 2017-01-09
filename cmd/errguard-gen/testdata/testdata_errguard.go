package testdata

// AUTOMATICALLY GENERATED -- DO NOT MODIFY

import (
	"bytes"
	"context"
	"time"

	"github.com/jjeffery/errguard"
)

type guardService struct {
	inner Service
}

func newGuardService(inner Service) Service {
	return &guardService{inner: inner}
}

func (g *guardService) DoSomething(ctx context.Context, input *DoSomethingInput) (output *DoSomethingOutput, err error) {
	var guard errguard.Guard
	err = guard.Run(ctx, func() error {
		output, err = g.inner.DoSomething(ctx, input)
		return err
	})
	return output, err
}

func (g *guardService) NoArgs() (err error) {
	var guard errguard.Guard
	err = guard.Run(context.TODO(), func() error {
		err = g.inner.NoArgs()
		return err
	})
	return err
}

func (g *guardService) ExternPackage(a time.Time) (a1 *bytes.Buffer, err error) {
	var guard errguard.Guard
	err = guard.Run(context.TODO(), func() error {
		a1, err = g.inner.ExternPackage(a)
		return err
	})
	return a1, err
}
