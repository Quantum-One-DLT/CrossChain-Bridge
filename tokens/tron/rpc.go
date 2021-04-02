package tron

import (
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"

	"github.com/anyswap/CrossChain-Bridge/log"
)

var GRPC_TIMEOUT = time.Second * 15

func (b *Bridge) getClients() []*client.GrpcClient {
	endpoints := b.GatewayConfig.APIAddress
	clis = make([]*client.GrpcClient, 0)
	for _, endpoint := range endpoints {
		cli := client.NewGrpcClientWithTimeout(endpoint, GRPC_TIMEOUT)
		if cli != nil {
			clis = append(clis, cli)
		}
	}
}

type RPCError struct {
	errs   []error
	method string
}

func (e *RPCError) log(msg error) {
	log.Warn("[Solana RPC error]", "method", e.method, "msg", msg)
	if len(e.errs) < 1 {
		e.errs = make([]error, 1)
	}
	e.errs = append(e.errs, msg)
}

func (e *RPCError) Error() error {
	return fmt.Errorf("[Solana RPC error] method: %v errors:%+v", e.method, e.errs)
}

// GetLatestBlockNumber returns current finalized block height
func (b *Bridge) GetLatestBlockNumber() (height uint64, err error) {
	rpcError := &RPCError{[]error{}, "GetLatestBlockNumber"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		res, err := cli.GetNowBlock()
		if err == nil {
			if res.BlockHeader.RawData.Number > 0 {
				height = uint64(res.BlockHeader.RawData.Number)
				cli.Stop()
				break
			}
		} else {
			rpcError.log(err)
		}
		cli.Stop()
	}
	if height > 0 {
		return height, nil
	}
	return 0, rpcError.Error()
}

// GetLatestBlockNumberOf returns current finalized block height from given node
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	rpcError := &RPCError{[]error{}, "GetLatestBlockNumberOf"}
	cli := client.NewGrpcClientWithTimeout(apiAddress, GRPC_TIMEOUT)
	if cli == nil {
		rpcError.log(errors.New("New client failed"))
		return 0, rpcError.Error()
	}
	err := cli.Start(grpc.WithInsecure())
	if err != nil {
		rpcError.log(err)
		return 0, rpcError.Error()
	}
	res, err := cli.GetNowBlock()
	if err != nil {
		rpcError.log(err)
		return 0, rpcError.Error()
	}
	return uint64(res.BlockHeader.RawData.Number), nil
}

// GetBalance gets SOL token balance
func (b *Bridge) GetBalance(account string) (balance *big.Int, err error) {
	rpcError := &RPCError{[]error{}, "GetBalance"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		res, err := cli.GetAccount(account)
		if err == nil {
			if res.Balance > 0 {
				balance = big.NewInt(int64(res.Balance))
				cli.Stop()
				break
			}
		} else {
			rpcError.log(err)
		}
		cli.Stop()
	}
	if balance.Cmp(big.NewInt(0)) > 0 {
		return balance, nil
	}
	return big.NewInt(0), rpcError.Error()
}

func (b *Bridge) GetTokenBalance(tokenType, tokenAddress, accountAddress string) (balance *big.Int, err error) {
	switch strings.ToUpper(tokenType) {
	case TRC20TokenType:
		return b.GetTrc20Balance(tokenAddress, accountAddress)
	case TRC10TokenType:
		return nil, fmt.Errorf("[%v] can not get token balance of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	default:
		return nil, fmt.Errorf("[%v] can not get token balance of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	}
}

// GetTrc20Balance gets balance for given ERC20 token
func (b *Bridge) GetTrc20Balance(tokenAddress, accountAddress string) (balance *big.Int, err error) {
	rpcError := &RPCError{[]error{}, "GetTrc20Balance"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		res, err := cli.TRC20ContractBalance(accountAddress, tokenAddress)
		if err == nil {
			balance = res
			cli.Stop()
			break
		} else {
			rpcError.log(err)
		}
		cli.Stop()
	}
	if balance.Cmp(big.NewInt(0)) > 0 {
		return balance, nil
	}
	return big.NewInt(0), rpcError.Error()
}

// GetTokenSupply impl
func (b *Bridge) GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error) {
	switch strings.ToUpper(tokenType) {
	case TRC20TokenType:
		return b.GetErc20TotalSupply(tokenAddress)
	case TRC10TokenType:
		return nil, fmt.Errorf("[%v] can not get token supply of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	default:
		return nil, fmt.Errorf("[%v] can not get token supply of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	}
}

// GetTokenSupply not supported
func (b *Bridge) GetErc20TotalSupply(tokenAddress string) (totalSupply *big.Int, err error) {
	totalSupplyMethodSignature := "0x18160ddd"
	rpcError := &RPCError{[]error{}, "GetErc20TotalSupply"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		result, err := cli.TRC20Call("", tokenAddress, totalSupplyMethodSignature, true, 0)
		if err == nil {
			totalSupply = new(big.Int).SetBytes(result.GetConstantResult()[0])
			cli.Stop()
			break
		} else {
			rpcError.log(err)
		}
		cli.Stop()
	}
	if totalSupply.Cmp(big.NewInt(0)) > 0 {
		return balance, nil
	}
	return big.NewInt(0), rpcError.Error()
}

// GetTransaction gets tx by hash, returns sdk.Tx
func (b *Bridge) GetTransaction(txHash string) (tx interface{}, err error) {
	rpcError := &RPCError{[]error{}, "GetTransaction"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		tx, err = cli.GetTransactionInfoByID(txHash)
		if err == nil {
			cli.Stop()
			break
		}
		cli.Stop()
	}
	if err != nil {
		return nil, rpcError.Error()
	}
	return
}

// GetTransactionStatus returns tx status
func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus) {
	status = &tokens.TxStatus{}
	var tx *troncore.Transaction
	for _, cli := range b.getClients() {
		err := cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		tx, err = cli.GetTransactionInfoByID(txHash)
		if err == nil {
			cli.Stop()
			break
		}
		cli.Stop()
	}
	if err != nil {
		return nil, rpcError.Error()
	}
	status.Receipt = tx.Receipt
	status.PrioriFinalized = false
	status.BlockNumber = tx.BlockNumber
	status.BlockTime = tx.BlockTimeStamp / 1000

	if latest, err := b.GetLatestBlockNumber(); err == nil {
		status.Confirmations = latest - status.BlockHeight
	}
	return
}

// BuildTransfer returns an unsigned tron transfer tx
func (b *Bridge) BuildTransfer(from, to string amount *big.NewInt, input []byte) (tx *core.Transaction, err error) {
	n, _ := new(big.Int).SetString("18446740000000000000", 0)
	if amount.Cmp(n) > 0 {
		return nil, errors.New("Amount exceed max uint64")
	}
	contract := &core.TransferContract{}
	contract.OwnerAddress, err = common.DecodeCheck(from)
	if err != nil {
		return nil, err
	}
	contract.ToAddress, err = common.DecodeCheck(to)
	if err != nil {
		return nil, err
	}
	rpcError := &RPCError{[]error{}, "BuildTransfer"}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.Log(err)
			continue
		}
		tx, err = cli.Client.CreateTransaction2(ctx, contract)
		if err == nil {
			cli.Stop()
			break
		}
		rpcError(err)
		cli.Stop()
	}
	if err != nil {
		return rpcError.Error()
	}
	return tx, nil
}

// BuildTRC20Transfer returns an unsigned trc20 transfer tx
func (b *Bridge) BuildTRC20Transfer(from, to, tokenAddress string amount *big.NewInt) (tx *core.Transaction, err error) {
	n, _ := new(big.Int).SetString("18446740000000000000", 0)
	if amount.Cmp(n) > 0 {
		return nil, errors.New("Amount exceed max uint64")
	}
	contract := &core.TransferContract{}
	contract.OwnerAddress, err = common.DecodeCheck(from)
	if err != nil {
		return nil, err
	}
	contract.ToAddress, err = common.DecodeCheck(to)
	if err != nil {
		return nil, err
	}
	rpcError := &RPCError{[]error{}, "BuildTRC20Transfer"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.Log(err)
			continue
		}
		txext, err1 := cli.TRC20Send(from, to, tokenAddress, amount)
		err = err1
		if err == nil {
			tx = txext.Transaction
			cli.Stop()
			break
		}
		rpcError(err)
		cli.Stop()
	}
	if err != nil {
		return rpcError.Error()
	}
	return tx, nil
}

// BuildSwapinTx returns an unsigned mapping asset minting tx
func (b *Bridge) BuildSwapinTx(from, to, tokenAddress string amount *big.NewInt, txhash string) (tx *core.Transaction, err error) {
	n, _ := new(big.Int).SetString("18446740000000000000", 0)
	if amount.Cmp(n) > 0 {
		return nil, errors.New("Amount exceed max uint64")
	}
	param := fmt.Sprintf(`[{"string":"%s"},{"address":"%s"},{"uint256":"%v"}]`, txhash, to, amount.Uint64())
	rpcError := &RPCError{[]error{}, "BuildSwapinTx"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.Log(err)
			continue
		}
		txext, err1 := cli.TriggerConstantContract(from, contract, method, param)
		err = err1
		if err == nil {
			tx = txext.Transaction
			cli.Stop()
			break
		}
		rpcError.log(err)
		cli.Stop()
	}
	if err != nil {
		return rpcError.Error()
	}
	return tx, nil
}

// GetCode returns contract bytecode
func (b *Bridge) GetCode(contractAddress string) (data []byte, err error) {
	contractDesc, err := tronaddress.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}
	message := new(api.BytesMessage)
	message.Value = contractDesc
	rpcError := &RPCError{[]error{}, "GetCode"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.Log(err)
			continue
		}
		sm, err := cli.Client.GetContract(ctx, message)
		if err == nil {
			data = sm.Bytecode
			cli.Stop()
			break
		}
		cli.Stop()
	}
	if err != nil {
		return nil, rpcError.Error()
	}
	return data, nil
}