package ui

import (
	"context"
	"errors"
)

func beginStream(parent context.Context, bus *messageBus, class streamClass, generation *requestID, cancel *context.CancelFunc) (context.Context, requestID) {
	stopStream(generation, cancel)
	*generation++
	id := *generation
	bus.AdvanceStream(class, id)
	ctx, nextCancel := context.WithCancel(parent)
	*cancel = nextCancel
	return ctx, id
}

func stopStream(generation *requestID, cancel *context.CancelFunc) {
	if *cancel == nil {
		return
	}
	(*cancel)()
	*cancel = nil
	*generation++
}

func streamCanceled(err error) bool {
	return errors.Is(err, context.Canceled)
}
