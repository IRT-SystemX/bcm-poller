package ingest

import (
    "log"
    "math/big"
    //"reflect"
    "errors"
    //"context"
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

func (fork *ForkWatcher) checkFork(blockEvent *BlockEvent) {
    if fork.last() != nil {
        if new(big.Int).Add(fork.last().Number, one).Cmp(blockEvent.Number) != 0 { // TODO: test parentHash instead
            log.Printf("Detect fork block #%s != #%s", fork.last().Number, blockEvent.Number)
            blockEvent.Fork = true
            err := fork.revert(blockEvent)
            if err != nil {
                log.Println(err)
            }
        }
        if fork.last() != nil && fork.last().Timestamp != 0 {
            blockEvent.Interval = fork.last().Timestamp - blockEvent.Timestamp
        }
    }
}

func (fork *ForkWatcher) last() *BlockEvent {
    if fork.chain.Len() > 0 {
        return fork.chain.Back().Value.(*BlockEvent)
    } else {
        return nil
    }
}

func (fork *ForkWatcher) revert(blockEvent *BlockEvent) error {
    main := list.New()
    last := fork.walkChain(main, fork.chain.Front())
    if main.Len() == 0 {
        return errors.New("Fork revert error : out of scope")
    }
    number := last.Value.(*BlockEvent).Number
    for last.Next() != nil {
        /*
        if fork.connector != nil && !reflect.ValueOf(fork.connector).IsNil() {
            fork.connector.Revert(last.Value.(*BlockEvent))
        }*/
        last = last.Next()
    }
    fork.chain = main
    for i := new(big.Int).Set(number); i.Cmp(blockEvent.Number) < 0; i.Add(i, one) {
        //fork.apply(fork.process(i))
    }
    return nil
}

func (fork *ForkWatcher) walkChain(main *list.List, elem *list.Element) *list.Element {
    /*block := elem.Value.(*BlockEvent)
    header, err := fork.client.HeaderByNumber(context.Background(), block.Number)
    if err != nil {
        log.Fatal(err)
    }
    if block.Hash == header.Hash().Hex() {
        main.PushBack(block)
        if elem.Next() != nil {
            return fork.walkChain(main, elem.Next())
        }
    } */
    return elem
}

