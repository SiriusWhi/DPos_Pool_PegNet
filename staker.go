package main

import (
	"fmt"
	"github.com/Factom-Asset-Tokens/factom"
	"github.com/FactomProject/factomd/common/primitives"
	"github.com/pegnet/pegnet/modules/spr"
)

var (
	SPRChain = factom.NewBytes32("e3b1668158026b2450d123ba993aca5367a8b96c6018f63640101a28b8ab5bc7")
)

func getDelegatorsAddress(delegatorData []byte, signature []byte, signer string) ([]string, error) {
	if len(signature) != 96 {
		return nil, fmt.Errorf("Invalid signature length")
	}
	dPubKey := signature[:32]
	dSignData := signature[32:]

	err3 := primitives.VerifySignature(delegatorData, dPubKey[:], dSignData[:])
	if err3 != nil {
		return nil, fmt.Errorf("Invalid signature")
	}

	var listOfDelegatorsAddress []string
	for bI := 0; bI < len(delegatorData); bI += 148 {
		delegator := delegatorData[bI : bI+148]
		addressOfDelegator := delegator[:52]
		signDataOfDelegator := delegator[52:116]
		pubKeyOfDelegator := delegator[116:]

		err2 := primitives.VerifySignature([]byte(signer), pubKeyOfDelegator[:], signDataOfDelegator[:])
		if err2 != nil {
			continue
		}
		listOfDelegatorsAddress = append(listOfDelegatorsAddress, string(addressOfDelegator[:]))
	}
	return listOfDelegatorsAddress, nil
}

func main() {
	cl := factom.NewClient()
	cl.FactomdServer = "http://localhost:8088/v2"

	heights := new(factom.Heights)
	err := heights.Get(nil, cl)
	if err != nil {
		fmt.Println("factom height is not getting correctly")
	}
	for height := 194000; height <= int(heights.DirectoryBlock); height++ {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%", height)
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

			for _, entry := range sprEBlock.Entries {
				extids := make([][]byte, len(entry.ExtIDs))
				for i := range entry.ExtIDs {
					extids[i] = entry.ExtIDs[i]
				}
				o2, errP := spr.ParseS1Content(entry.Content)
				if errP != nil && len(extids) == 5 && len(extids[0]) == 1 && extids[0][0] == 8 {
					listOfDelegatorsAddress, err := getDelegatorsAddress(extids[3], extids[4], o2.Address)
					if err != nil {
						fmt.Println("listOfDelegatorsAddress:", listOfDelegatorsAddress)
					}
				}
			}
			/*
				for i, entry := range sprEBlock.Entries {
					extids := make([][]byte, len(entry.ExtIDs))
					for i := range entry.ExtIDs {
						extids[i] = entry.ExtIDs[i]
					}

					o2, err := spr.ParseS1Content(entry.Content)
					if err != nil {
						fmt.Println("parsing error...", err)
					}
					fmt.Println("staker", i, ": ", o2.Address, "================================================================================")
					//fmt.Println(extids)


					// Validations

					if len(extids) != 5 {
						fmt.Println("Invalid extid count")
						//return nil, NewValidateError("Invalid extid count")
						break
					}

					if len(extids[0]) != 1 || extids[0][0] != 7 {
						fmt.Println("Invalid version")
						//return nil, NewValidateError("Invalid version")
						break
					}
					// Verify Signature
					listOfDelegatorsAddress, err := getDelegatorsAddress(extids[3], extids[4], o2.Address)
					if err != nil {
						break
					}
					fmt.Println("listOfDelegatorsAddress:", listOfDelegatorsAddress)
				}*/
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
