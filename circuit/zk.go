package main

import (
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/hash"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/std/hash/mimc"
	"errors"
	"fmt"
	"math/big"

	/*"github.com/consensys/gnark-crypto/hash"
	"github.com/consensys/gnark/frontend"
	"math/big"*/
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

	// 1. 用私钥生成公钥（使用MiMC哈希）
	mimcHash, _ := mimc.NewMiMC(api)
	mimcHash.Write(circuit.PrivateKey)
	computedPk := mimcHash.Sum()

	// 2. 验证公钥在指定索引位置
	api.AssertIsEqual(circuit.PkArray[circuit.PkIndex], computedPk)

	// 3. 构建Merkle树验证根节点
	merkleRoot := computeMerkleRoot(api, circuit.PkArray)
	api.AssertIsEqual(merkleRoot, circuit.MerkleRoot)

	// 1. 验证PkArray 计算出来的root 与 mp相等
	// 2. 验证sk =》 pk 在 array中
	return nil
}

// 计算Merkle根
func computeMerkleRoot(api frontend.API, nodes []frontend.Variable) frontend.Variable {
	if len(nodes) == 1 {
		return nodes[0]
	}

	var nextLevel []frontend.Variable
	for i := 0; i < len(nodes); i += 2 {
		h, _ := mimc.NewMiMC(api)
		h.Write(nodes[i], nodes[i+1])
		nextLevel = append(nextLevel, h.Sum())
	}
	return computeMerkleRoot(api, nextLevel)
}

// 生成证明
func GenerateProof(circuit *Voting) (groth16.Proof, groth16.VerifyingKey, error) {
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	if err != nil {
		return nil, nil, fmt.Errorf("编译电路失败: %v", err)
	}

	pk, vk, err := groth16.Setup(cs)
	if err != nil {
		return nil, nil, fmt.Errorf("初始化证明系统失败: %v", err)
	}

	witness, err := frontend.NewWitness(circuit, ecc.BN254.ScalarField())
	if err != nil {
		return nil, nil, fmt.Errorf("生成证据失败: %v", err)
	}

	proof, err := groth16.Prove(cs, pk, witness)
	if err != nil {
		return nil, nil, fmt.Errorf("生成证明失败: %v", err)
	}

	return proof, vk, nil
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
