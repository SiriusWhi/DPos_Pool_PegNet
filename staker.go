package main

import (
	"fmt"
	"github.com/Factom-Asset-Tokens/factom"
	"github.com/pegnet/pegnet/modules/spr"
)

var (
	SPRChain                              = factom.NewBytes32("e3b1668158026b2450d123ba993aca5367a8b96c6018f63640101a28b8ab5bc7")
)

type StakerType struct {
	staker string
	count int64
}

func main() {
	cl := factom.NewClient()
	cl.FactomdServer = "http://localhost:8088/v2"

	heights := new(factom.Heights)
	err := heights.Get(nil, cl)
	if err != nil {
		fmt.Println("factom height is not getting correctly")
	}

	for height:=194269; height <= int(heights.DirectoryBlock); height++ {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%", height)
		dblock := new(factom.DBlock)
		dblock.Height = uint32(height)

		if err := dblock.Get(nil, cl); err != nil {
			fmt.Println("error: ", err)
			return
		}

		sprEBlock := dblock.EBlock(SPRChain)
		if sprEBlock != nil {
			if err := multiFetch2(sprEBlock, cl); err != nil {
				fmt.Println("error: ", err)
				return
			}
		}
		var stakers []string
		for i, entry := range sprEBlock.Entries {
			extids := make([][]byte, len(entry.ExtIDs))
			for i := range entry.ExtIDs {
				extids[i] = entry.ExtIDs[i]
			}

			o2, err := spr.ParseS1Content(entry.Content)
			if err != nil {
				fmt.Println("parsing error...", err)
			}
			//fmt.Println("FactomID:", o.GetID())
			stakers = append(stakers, o2.Address)
			fmt.Println("staker", i, ": ", o2.Address, "================================================================================")
			fmt.Println(o2.Assets)
			fmt.Println("")
		}
	}
}


func multiFetch2(eblock *factom.EBlock, c *factom.Client) error {
	err := eblock.Get(nil, c)
	if err != nil {
		return err
	}

	work := make(chan int, len(eblock.Entries))
	defer close(work)
	errs := make(chan error)
	defer close(errs)

	for i := 0; i < 8; i++ {
		go func() {
			// TODO: Fix the channels such that a write on a closed channel never happens.
			//		For now, just kill the worker go routine
			defer func() {
				recover()
			}()

			for j := range work {
				errs <- eblock.Entries[j].Get(nil, c)
			}
		}()
	}

	for i := range eblock.Entries {
		work <- i
	}

	count := 0
	for e := range errs {
		count++
		if e != nil {
			// If we return, we close the errs channel, and the working go routine will
			// still try to write to it.
			return e
		}
		if count == len(eblock.Entries) {
			break
		}
	}

	return nil
}
