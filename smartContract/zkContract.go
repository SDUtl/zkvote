package main

// package 必须是main

import (
	"chainmaker.org/chainmaker/contract-sdk-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sandbox"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sdk"
	"log"
)

// FactContract 合约结构体，合约名称需要写入main()方法当中
type VotingContract struct {
}

type VotingParams struct {
	VotingTitle  string `json:"voting_title"`
	TimeStamp    string `json:"time_stamp"`
	VotingChoice string `json:"voting_choice"`
	// ...
}

func (c *VotingContract) InitContract() protogo.Response {
	return sdk.Success([]byte("Init success"))
}

func (c *VotingContract) UpgradeContract() protogo.Response {
	return sdk.Success([]byte("Upgrade success"))
}

// QueryVotingParams 查询投票参数
func (c *VotingContract) QueryVotingParams() protogo.Response {
	return sdk.Success([]byte("Query success"))
}

// 仅管理员可调用
func (c *VotingContract) InitialVotingParams() protogo.Response {
	return sdk.Success([]byte("Set success"))
}

func (c *VotingContract) Vote() protogo.Response {
	/*
		 1. 首先进行zk proof 的验证
			若 验证通过，且未存储，则允许投票，否则不允许
		 2. 投票成功之后 将zk prooof存储至链上
	*/
	return sdk.Success([]byte("Vote success"))
}

// InvokeContract 用于合约的调用
// @param method: 交易请求调用的方法
// @return: 	合约返回结果，包括Success和Error
func (v *VotingContract) InvokeContract(method string) protogo.Response {
	switch method {
	case "initialVotingParams":
		return v.InitialVotingParams()
	case "queryVotingParams":
		return v.QueryVotingParams()
	case "getVotingParams":
		return v.QueryVotingParams()
	case "vote":
		return v.Vote()
	default:
		return sdk.Error("invalid method")
	}
}

// sdk代码中，有且仅有一个main()方法
func main() {
	// main()方法中，下面的代码为必须代码，不建议修改main()方法当中的代码
	// 其中，TestContract为用户实现合约的具体名称
	err := sandbox.Start(new(VotingContract))
	if err != nil {
		log.Fatal(err)
	}
}
