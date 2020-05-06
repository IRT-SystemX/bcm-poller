package ingest

import (
    "container/list"
)

type ForkWatcher struct {
    maxForkSize int
    chain *list.List
}

func NewForkWatcher(maxForkSize int) *ForkWatcher {
    return &ForkWatcher{ maxForkSize: maxForkSize, chain: list.New() }
}

func (fork *ForkWatcher) apply(blockEvent *BlockEvent) {
    if fork.chain.Len() >= fork.maxForkSize {
        fork.chain.Remove(fork.chain.Front())
    }
    fork.chain.PushBack(blockEvent)
}

