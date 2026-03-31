package opener

import (
	"context"
	"os/exec"
)

type Opener interface {
	Open(ctx context.Context, rawURL string) error
}

type defaultOpener struct{}

func New() Opener {
	return defaultOpener{}
}

func (defaultOpener) Open(ctx context.Context, rawURL string) error {
	name, args, err := platformCommand(rawURL)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}
