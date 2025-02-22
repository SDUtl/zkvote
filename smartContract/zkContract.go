package main

// package 必须是main

import (
	"encoding/json"
	"errors"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sandbox"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sdk"
	"log"
)

// 定义合约状态键
const (
	ConfigKey      = "voting_config"
	VotesKey       = "votes"
	UsedProofsKey  = "used_proofs"
	AdminKey       = "admin"
	VoteCountKey   = "vote_count"
)

// VotingConfig 投票配置
type VotingConfig struct {
	Title         string   `json:"title"`
	Options       []string `json:"options"`
	ValidPks      []string `json:"valid_pks"`     // 合格参与者公钥数组
	MerkleRoot    string   `json:"merkle_root"`   // 默克尔根
	ZKPVerifier   string   `json:"zkp_verifier"`  // ZKP验证合约名称
	StartTime     int64    `json:"start_time"`
	EndTime       int64    `json:"end_time"`
}

// VoteRecord 投票记录
type VoteRecord struct {
	Proof       string `json:"proof"`       // ZKP证明
	Choice      int    `json:"choice"`      // 投票选项
	Timestamp   int64  `json:"timestamp"`   // 投票时间
	VoterHash   string `json:"voter_hash"`  // 选民身份哈希
}


// FactContract 合约结构体，合约名称需要写入main()方法当中
type VotingContract struct {
}

/*type VotingParams struct {
	VotingTitle  string `json:"voting_title"`
	TimeStamp    string `json:"time_stamp"`
	VotingChoice string `json:"voting_choice"`
	// ...
}*/

func (c *VotingContract) InitContract() protogo.Response {
	// 设置部署者为管理员
	caller := sdk.Instance.GetSender()
	if err := sdk.Instance.PutStateByte(AdminKey, "", []byte(caller)); err != nil {
		return sdk.Error("init admin failed: " + err.Error())
	}
	return sdk.Success([]byte("Init success"))
}


func (c *VotingContract) UpgradeContract() protogo.Response {
	return sdk.Success([]byte("Upgrade success"))
}

// QueryVotingParams 查询投票参数
func (c *VotingContract) QueryVotingParams() protogo.Response {
	configBytes, err := sdk.Instance.GetStateByte(ConfigKey, "")
	if err != nil || len(configBytes) == 0 {
		return sdk.Error("voting not initialized")
	}
	return sdk.Success([]byte("Query success"))
}

// ======================== 管理员方法 ========================   仅管理员可调用
func (c *VotingContract) InitialVotingParams() protogo.Response {
	// 权限验证
	if !c.isAdmin() {
		return sdk.Error("permission denied")
	}

	// 参数解析
	params := struct {
		Title      string   `json:"title"`
		Options    []string `json:"options"`
		ValidPks   []string `json:"valid_pks"`
		MerkleRoot string   `json:"merkle_root"`
		StartTime  int64    `json:"start_time"`
		EndTime    int64    `json:"end_time"`
	}{}

	if err := sdk.Instance.GetArgsJson(&params); err != nil {
		return sdk.Error("invalid params: " + err.Error())
	}

	// 参数校验
	if len(params.Options) < 2 {
		return sdk.Error("at least 2 options required")
	}

	// 存储配置
	config := VotingConfig{
		Title:       params.Title,
		Options:     params.Options,
		ValidPks:    params.ValidPks,
		MerkleRoot:  params.MerkleRoot,
		StartTime:   params.StartTime,
		EndTime:     params.EndTime,
	}

	configBytes, _ := json.Marshal(config)
	if err := sdk.Instance.PutStateByte(ConfigKey, "", configBytes); err != nil {
		return sdk.Error("save config failed: " + err.Error())
	}

	// 初始化投票计数器
	for i := range params.Options {
		if err := sdk.Instance.PutStateByte(VoteCountKey, string(rune(i)), []byte("0")); err != nil {
			return sdk.Error("init vote count failed")
		}
	}
	return sdk.Success([]byte("Set success"))
}

func (c *VotingContract) Vote() protogo.Response {
	/*
		 1. 首先进行zk proof 的验证
			若 验证通过，且未存储，则允许投票，否则不允许
		 2. 投票成功之后 将zk prooof存储至链上
	*/
	// 1. 基础校验
	if c.isVotingClosed() {
		return sdk.Error("voting is closed")
	}

	// 2. 解析参数
	args := struct {
		Proof  string `json:"proof"`
		Choice int    `json:"choice"`
	}{}
	if err := sdk.Instance.GetArgsJson(&args); err != nil {
		return sdk.Error("invalid params: " + err.Error())
	}

	// 3. 验证ZKP证明
	if valid, err := c.verifyZKP(args.Proof); !valid || err != nil {
		return sdk.Error("invalid zkp proof")
	}

	// 4. 检查重复投票
	if exists, _ := sdk.Instance.GetStateByte(UsedProofsKey, args.Proof); exists != nil {
		return sdk.Error("duplicate voting")
	}

	// 5. 记录投票
	record := VoteRecord{
		Proof:     args.Proof,
		Choice:    args.Choice,
		Timestamp: sdk.Instance.GetBlockTimestamp(),
		VoterHash: hashIdentity(sdk.Instance.GetSender()),
	}

	recordBytes, _ := json.Marshal(record)
	if err := sdk.Instance.PutStateByte(VotesKey, args.Proof, recordBytes); err != nil {
		return sdk.Error("save vote failed")
	}

	// 6. 更新计数器
	current, _ := sdk.Instance.GetStateByte(VoteCountKey, string(rune(args.Choice)))
	newCount := big.NewInt(0).SetBytes(current).Add(big.NewInt(0).SetBytes(current), big.NewInt(1))
	sdk.Instance.PutStateByte(VoteCountKey, string(rune(args.Choice)), newCount.Bytes())

	// 7. 标记已使用的证明
	sdk.Instance.PutStateByte(UsedProofsKey, args.Proof, []byte{1})
	return sdk.Success([]byte("Vote success"))
}


// ======================== 辅助方法 ========================
func (c *VotingContract) isAdmin() bool {
	admin, _ := sdk.Instance.GetStateByte(AdminKey, "")
	return bytes.Equal(admin, []byte(sdk.Instance.GetSender()))
}

func (c *VotingContract) isVotingClosed() bool {
	configBytes, _ := sdk.Instance.GetStateByte(ConfigKey, "")
	var config VotingConfig
	json.Unmarshal(configBytes, &config)
	
	now := sdk.Instance.GetBlockTimestamp()
	return now < config.StartTime || now > config.EndTime
}

func (c *VotingContract) verifyZKP(proof string) (bool, error) {
	// 调用ZKP验证合约（需要根据实际部署调整）
	result := sdk.Instance.CallContract(
		"zkp_verifier", 
		"VerifyProof", 
		map[string][]byte{
			"proof": []byte(proof),
		},
	)
	
	return result.Status == protogo.OK, nil
}

func hashIdentity(identity string) string {
	// 实际应使用密码学哈希函数
	return sdk.Instance.Sha256([]byte(identity))
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
