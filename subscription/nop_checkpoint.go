package subscription

import "context"

type NopCheckpointer struct{}

func (nc NopCheckpointer) Get(context.Context, string) (int64, error) {
	return 0, nil
}

func (nc NopCheckpointer) Store(context.Context, string, int64) error {
	return nil
}
