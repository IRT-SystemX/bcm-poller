package ingest

import (
	"container/list"
	"log"
	"reflect"
)

type ForkWatcher struct {
	engine      *Engine
	maxForkSize int
	chain       *list.List
}

func NewForkWatcher(engine *Engine, maxForkSize int) *ForkWatcher {
	return &ForkWatcher{engine: engine, maxForkSize: maxForkSize, chain: list.New()}
}

func (fork *ForkWatcher) last() *BlockEvent {
	if fork.chain.Len() > 0 {
		return fork.chain.Back().Value.(*BlockEvent)
	} else {
		return nil
	}
}

func (fork *ForkWatcher) apply(blockEvent *BlockEvent) {
	if fork.chain.Len() >= fork.maxForkSize {
		fork.chain.Remove(fork.chain.Front())
	}
	fork.chain.PushBack(blockEvent)
}

func (fork *ForkWatcher) revert(elem *list.Element) {
	if fork.chain.Len() > 0 {
		fork.chain.Remove(elem)
	}
	if fork.engine.connector != nil && !reflect.ValueOf(fork.engine.connector).IsNil() {
		fork.engine.connector.Revert(elem.Value.(*BlockEvent))
	}
}

func (fork *ForkWatcher) checkFork(blockEvent *BlockEvent) {
	if fork.last() != nil {
		if fork.last().Hash != blockEvent.ParentHash {
			blockEvent.Fork = true
			if fork.last().Hash == blockEvent.Hash {
				//log.Printf("Detect block update")
				fork.revert(fork.chain.Back())
			} else {
				//log.Printf("Detect fork block %s != %s", blockEvent.ParentHash, fork.last().Hash)
				//log.Printf("Detect fork block #%s != #%s", blockEvent.Number, fork.last().Number)
				if blockEvent.Number.Cmp(fork.last().Number) <= 0 {
					//log.Printf("Number <= Last")
					toRevert := list.New()
					for elem := fork.chain.Back(); elem != nil && blockEvent.Number.Cmp(elem.Value.(*BlockEvent).Number) <= 0; elem = elem.Prev() {
						toRevert.PushBack(elem)
					}
					for elem := toRevert.Front(); elem != nil; elem = elem.Next() {
						fork.revert(elem.Value.(*list.Element))
					}
				} else {
					//log.Printf("Number > Last")
					//fork.debugChain()
				}
			}
		}
	}
}

func (fork *ForkWatcher) debugChain() {
	i := 0
	for e := fork.chain.Back(); e != nil; e = e.Prev() {
		log.Printf("%x %s", i, e.Value.(*BlockEvent).Hash)
		i++
	}
}
