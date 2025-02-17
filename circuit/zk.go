package main

import (
	"github.com/consensys/gnark-crypto/hash"
	"github.com/consensys/gnark/frontend"
	"math/big"
)

type Voting struct {
	PkArray    []frontend.Variable `gnark:",public"`
	MerkleRoot frontend.Variable   `gnark:",public"`

	PkIndex    frontend.Variable
	PrivateKey frontend.Variable
}

func (circuit *Voting) Define(api frontend.API) error {

	//hFunc, err := mimc.NewMiMC(api)
	//if err != nil {
	//	return errors.New("mimc.NewMiMC failed")
	//}

	// 1. 验证PkArray 计算出来的root 与 mp相等
	// 2. 验证sk =》 pk 在 array中
	return nil
}

func ComputeHashBytes(preImages []string) string {
	f := hash.MIMC_BN254.New()
	for _, preImage := range preImages {
		f.Write([]byte(preImage))
	}
	hash := f.Sum(nil)
	hashInt := big.NewInt(0).SetBytes(hash)
	return hashInt.String()
}

func main() {

}
