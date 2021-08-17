package main

import (
	"fmt"
	"github.com/Factom-Asset-Tokens/factom"
	"github.com/pegnet/pegnet/modules/opr"
)

var (
	OPRChain                              = factom.NewBytes32("a642a8674f46696cc47fdb6b65f9c87b2a19c5ea8123b3d2f0c13b6f33a9d5ef")
	GradingV2Activation            uint32 = 210330
	PEGFreeFloatingPriceActivation uint32 = 222270
	V4OPRUpdate                    uint32 = 231620
)

type MinerType struct {
	miner string
	count int64
}

func main() {
	cl := factom.NewClient()
	cl.FactomdServer = "https://api.factomd.net/v2"

	heights := new(factom.Heights)
	err := heights.Get(nil, cl)
	if err != nil {
		fmt.Println("factom height is not getting correctly")
	}

	for height:=291780; height <= int(heights.DirectoryBlock); height++ {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%", height)
		dblock := new(factom.DBlock)
		dblock.Height = uint32(height)

		if err := dblock.Get(nil, cl); err != nil {
			fmt.Println("error: ", err)
			return
		}

		oprEBlock := dblock.EBlock(OPRChain)
		if oprEBlock != nil {
			if err := multiFetch(oprEBlock, cl); err != nil {
				fmt.Println("error: ", err)
				return
			}
		}
		var miners []string
		for i, entry := range oprEBlock.Entries {
			extids := make([][]byte, len(entry.ExtIDs))
			for i := range entry.ExtIDs {
				extids[i] = entry.ExtIDs[i]
			}

			o2, err := opr.ParseV2Content(entry.Content)
			if err != nil {
				fmt.Println("parsing error...", err)
			}
			o := &opr.V5Content{V2Content: *o2}
			//fmt.Println("FactomID:", o.GetID())
			miners = append(miners, o.GetID())
			fmt.Println("miner", i, ": ", o.GetID(), "==========================")
			fmt.Println(o.Assets)
			fmt.Println("")
		}

		var uniqueList []MinerType
		for _, miner := range miners {
			isExist := false
			for _, unique := range uniqueList {
				if unique.miner == miner {
					isExist = true
					break
				}
			}
			if !isExist {
				var item MinerType
				item.miner = miner
				item.count = 0
				uniqueList = append(uniqueList, item)
			}
		}

		for i, uniqueItem := range uniqueList {
			count := 0
			for _, miner := range miners {
				if uniqueItem.miner == miner {
					count ++
				}
			}
			uniqueList[i].count = int64(count)
		}
		fmt.Println(height, uniqueList)
	}
}

func multiFetch(eblock *factom.EBlock, c *factom.Client) error {
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
