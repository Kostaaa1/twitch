package main

import (
	"context"
	"crypto/rand"
	"os"
	"time"

	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type Unit struct {
	title string
	err   error
	id    any
}

func (u Unit) GetID() any {
	return u.id
}
func (u Unit) GetError() error {
	return u.err
}
func (u Unit) GetTitle() string {
	return u.title
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	uuid := uuid.NewString()
	unit := Unit{title: "This is the title of the spinnerspinnerspinnerspinnerspinnerspinner spinnerspinner spinnerspinnerspinnerspinnerspinner spinnerspinnerspinner spinnerspinner", id: uuid}
	units := []Unit{unit}

	spin := spinner.New(ctx, units, cancel)

	g.Go(func() error {
		spin.Run()
		return nil
	})

	f, err := os.CreateTemp("", "spinner_example.*.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	g.Go(func() error {
		for i := 0; i < 5; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1 * time.Second):
			}

			buf := make([]byte, 1_048_576)
			_, err := rand.Read(buf)
			if err != nil {
				panic(err)
			}

			n, err := f.Write(buf)
			if err != nil {
				panic(err)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case spin.C <- spinner.Message{
				ID:    unit.id,
				Bytes: int64(n),
			}:
			}
		}

		cancel()
		return nil
	})

	g.Wait()
	close(spin.C)
}
