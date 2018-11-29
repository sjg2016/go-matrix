//1543489817.7738225
//1543489068.9726374
//1543488409.0887682
//1543487620.2182815
//1543486819.843277
//1543485998.761771
// Copyright (c) 2018 The MATRIX Authors 
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php


package types

import (
	"container/heap"
	"errors"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/common/hexutil"
	"github.com/matrix/go-matrix/crypto"
	"github.com/matrix/go-matrix/rlp"
	"github.com/matrix/go-matrix/params"
)

//go:generate gencodec -type txdata -field-override txdataMarshaling -out gen_tx_json.go

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

func init()  {
	rlp.InterfaceConstructorMap[uint16(NormalTxIndex)] = func() interface{} {
		return &Transaction{}
	}
	rlp.InterfaceConstructorMap[uint16(BroadCastTxIndex)] = func() interface{} {
		return &TransactionBroad{}
	}
}
// deriveSigner makes a *best* guess about which signer to use.
//func deriveSigner(V *big.Int) Signer {
//	if V.Sign() != 0 && isProtectedV(V) {
//		return NewEIP155Signer(deriveChainId(V))
//	} else {
//		return HomesteadSigner{}
//	}
//}

type Transaction struct {
	data txdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
	entrustfrom atomic.Value
	// by hezi
	N []uint32
}

//YY
type Transaction_Mx struct {
	Data       txdata
	TxType_Mx  byte
	LockHeight uint64  `json:"lockHeight" gencodec:"required"`
	ExtraTo    []Tx_to `json:"extra_to" gencodec:"required"`
}

//YY
type ExtraTo_tr struct {
	To_tr    *common.Address `json:"to"`
	Value_tr *hexutil.Big    `json:"value"`
	Input_tr *hexutil.Bytes  `json:"input"`
}

//YY
type Tx_to struct {
	Recipient *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount    *big.Int        `json:"value"    gencodec:"required"`
	Payload   []byte          `json:"input"    gencodec:"required"`
}

//YY
type Matrix_Extra struct {
	TxType     byte    `json:"txType" gencodec:"required"`
	LockHeight uint64  `json:"lockHeight" gencodec:"required"`
	ExtraTo    []Tx_to `json:"extra_to" gencodec:"required"`
}

//YY 用于洪泛（传输）
type Floodtxdata struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`

	// Signature values
	V     *big.Int       `json:"v" gencodec:"required"`
	R     *big.Int       `json:"r" gencodec:"required"`
	TxEnterType common.TxTypeInt
	IsEntrustTx bool  `json:"TxEnterType" gencodec:"required"`//是否是委托
	Extra []Matrix_Extra ` rlp:"tail"`
}

type txdata struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash  *common.Hash   `json:"hash" rlp:"-"`
	TxEnterType common.TxTypeInt  `json:"TxEnterType" gencodec:"required"`//入池类型
	IsEntrustTx bool  `json:"TxEnterType" gencodec:"required"`//是否是委托
	Extra []Matrix_Extra ` rlp:"tail"` //YY
}

type txdataMarshaling struct {
	AccountNonce hexutil.Uint64
	Price        *hexutil.Big
	GasLimit     hexutil.Uint64
	Amount       *hexutil.Big
	Payload      hexutil.Bytes
	V            *hexutil.Big
	R            *hexutil.Big
	S            *hexutil.Big
}

func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, &to, amount, gasLimit, gasPrice, data)
}

func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, nil, amount, gasLimit, gasPrice, data)
}

//YY
func NewTransactions(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, ex []*ExtraTo_tr, localtime uint64, txType byte) *Transaction {
	return newTransactions(nonce, &to, amount, gasLimit, gasPrice, data, ex, localtime, txType)
}

//YY
func newTransactions(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, ex []*ExtraTo_tr, localtime uint64, txType byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
		TxEnterType: NormalTxIndex,
		Extra:        make([]Matrix_Extra, 0),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}
	//YY
	matrixEx := new(Matrix_Extra)
	arrayTx := make([]Tx_to, 0)
	if len(ex) > 0 {
		for _, extro := range ex {
			var input []byte
			if extro.Input_tr == nil {
				input = make([]byte, 0)
			} else {
				input = *extro.Input_tr
			}
			txto := new(Tx_to)

			txto.Amount = (*big.Int)(extro.Value_tr)
			txto.Recipient = extro.To_tr
			txto.Payload = input
			arrayTx = append(arrayTx, *txto)
		}
	}
	matrixEx.TxType = txType
	matrixEx.LockHeight = localtime
	matrixEx.ExtraTo = arrayTx
	d.Extra = append(d.Extra, *matrixEx)
	tx:=&Transaction{data: d}
	return tx
}

func newTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
		TxEnterType: NormalTxIndex,
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}
	tx:=&Transaction{data: d}
	return tx
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (tx *Transaction) ChainId() *big.Int {
	return deriveChainId(tx.data.V)
}

// Protected returns whether the transaction is protected from replay protection.
func (tx *Transaction) Protected() bool {
	return isProtectedV(tx.data.V)
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 are considered unprotected
	return true
}
type EncodeTx struct {
	Data txdata
	From common.Address
}
// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	etx := &EncodeTx{
				Data:tx.data,
			}
	if tx.GetMatrixType() == common.ExtraUnGasTxType{
		etx.From = tx.From()
	}
	return rlp.Encode(w,etx)
	//if tx.GetMatrixType() == common.ExtraUnGasTxType{
	//	return rlp.Encode(w, &EncodeTx{
	//		Data:tx.data,
	//		From:tx.From(),
	//	})
	//}else{
	//	return rlp.Encode(w, &tx.data)
	//}
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	var err error
	entx := new(EncodeTx)
	err = s.Decode(&entx)
	tx.data = entx.Data
	if tx.GetMatrixType() == common.ExtraUnGasTxType{
		tx.SetFromLoad(entx.From)
	}
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}
	//_, size, _ := s.Kind()
	//var err error
	//if tx.GetMatrixType() == common.ExtraUnGasTxType{
	//	entx := new(EncodeTx)
	//	err = s.Decode(&entx)
	//	tx.data = entx.Data
	//	tx.SetFromLoad(entx.From)
	//}else{
	//	err = s.Decode(&tx.data)
	//}
	//if err == nil {
	//	tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	//}

	return err
}

// MarshalJSON encodes the web3 RPC transaction format.
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()
	data := tx.data
	data.Hash = &hash
	return data.MarshalJSON()
}

// UnmarshalJSON decodes the web3 RPC transaction format.
func (tx *Transaction) UnmarshalJSON(input []byte) error {
	var dec txdata
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}
	var V byte
	if isProtectedV(dec.V) {
		chainID := deriveChainId(dec.V).Uint64()
		V = byte(dec.V.Uint64() - 35 - 2*chainID)
	} else {
		V = byte(dec.V.Uint64() - 27)
	}
	if !crypto.ValidateSignatureValues(V, dec.R, dec.S, false) {
		return ErrInvalidSig
	}
	*tx = Transaction{data: dec}
	return nil
}
func (tx *Transaction)GetConstructorType()uint16{
	return uint16(NormalTxIndex)
}
func (tx *Transaction) Data() []byte       { return common.CopyBytes(tx.data.Payload) }
func (tx *Transaction) Gas() uint64        { return tx.data.GasLimit }
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.Price) }
func (tx *Transaction) Value() *big.Int    { return new(big.Int).Set(tx.data.Amount) }
func (tx *Transaction) Nonce() uint64      { return tx.data.AccountNonce }
func (tx *Transaction) CheckNonce() bool   { return true }

func (tx *Transaction) GetTxHashStruct() {
	
} 
func (tx *Transaction)Call() error{
	return nil
}
func (tx *Transaction) CoinType()string{
	//TODO 返回交易中的币种
	return "MAN"
}
func (tx *Transaction) SetCoinType(typ string){
	//TODO 设置交易中的币种

}
func (tx *Transaction) TxType() common.TxTypeInt		{ return tx.data.TxEnterType}
//YY
func (tx *Transaction) GetMatrix_EX() []Matrix_Extra { return tx.data.Extra }

func (tx *Transaction) GetMatrixType() byte {
	if tx.data.Extra != nil && len(tx.data.Extra)>0{
		return tx.data.Extra[0].TxType
	}
	return common.ExtraNormalTxType
}

//YYY 为了兼容去掉的Message结构体
func (tx *Transaction) From() common.Address {
	addr, err := tx.GetTxFrom()
	if err != nil {
		return common.Address{}
	}
	return addr
}
func (tx *Transaction)GetFromLoad() interface{}  {
	return tx.from.Load()
}
func (tx *Transaction)SetFromLoad(x interface{})  {
	tx.from.Store(x)
}
//func (tx *Transaction)GetentrustFrom()(x interface{}){
//	return tx.entrustfrom.Load()
//}
func (tx *Transaction)Setentrustfrom(x interface{}){
	tx.entrustfrom.Store(x)
}
func (tx *Transaction)GasFrom() (from common.Address){
	//TODO 需要去委托中要💊 from
	tmp,ok := tx.from.Load().(sigCache)
	if !ok{
		tmpfrom,isok :=tx.from.Load().(common.Address)
		if !isok{
			return common.Address{}
		}
		from = tmpfrom
	}else {
		from = tmp.from
	}
	return
}
func (tx *Transaction)AmontFrom() (from common.Address){
	//TODO from 要改为entrustfrom
	tmp,ok := tx.from.Load().(sigCache)
	if !ok{
		tmpfrom,isok :=tx.from.Load().(common.Address)
		if !isok{
			return common.Address{}
		}
		from = tmpfrom
	}else {
		from = tmp.from
	}
	return
}
//YY
func (tx *Transaction) GetTxFrom() (from common.Address,err error) {
	if tx.from.Load() == nil{
		//如果交易没有做过验签则err不为空。
		return common.Address{},errors.New("Address is Nil")
	}
	var tf common.Address
	//如果交易做过验签则err为空。
	tmp,ok := tx.from.Load().(sigCache)
	if !ok{
		tmpfrom,isok :=tx.from.Load().(common.Address)
		if !isok{
			return common.Address{},errors.New("load Address is Nil")
		}
		if tmpfrom != tf{
			from = tmpfrom
		}else {
			return common.Address{},errors.New("load Address is Nil")
		}
	}else {
		from = tmp.from
	}
	return
}
//YY// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) CostALL() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Amount)
	for _, extra := range tx.data.Extra[0].ExtraTo {
		total.Add(total, extra.Amount)
	}
	return total
}
func (tx *Transaction)GetTxNLen()int{
	return len(tx.N)
}
//YY
func (tx *Transaction) GetTxV() *big.Int { return tx.data.V }
func (tx *Transaction) SetTxV(v *big.Int)  { tx.data.V = v}
func (tx *Transaction) SetTxR(r *big.Int)  { tx.data.R = r}
//YY
func (tx *Transaction) GetTxS() *big.Int { return tx.data.S }
//YY
func (tx *Transaction) GetTxR() *big.Int { return tx.data.R }
func (tx *Transaction) GetTxN(index int) uint32{
	return tx.N[index]
}
//YY 在传递交易时用来操作Nonce
func (tx *Transaction) SetNonce(nc uint64) {
	tx.data.AccountNonce = nc
}

//YY
func GetFloodData(tx *Transaction) *Floodtxdata {

	floodtx := &Floodtxdata{
		AccountNonce:tx.data.AccountNonce & params.NonceSubOne,
		Price:tx.data.Price,
		GasLimit:tx.data.GasLimit,
		Recipient:tx.data.Recipient,
		Amount:tx.data.Amount,
		Payload:tx.data.Payload,
		// Signature values
		V:     tx.data.V,
		R:     tx.data.R,
		TxEnterType : tx.data.TxEnterType,
		IsEntrustTx : tx.data.IsEntrustTx,
		Extra: tx.data.Extra,
	}
	return floodtx
}

//YY
func  SetFloodData(floodtx *Floodtxdata) *Transaction{
	tx:=&Transaction{}
	tx.data.AccountNonce = floodtx.AccountNonce | params.NonceAddOne
	tx.data.Price = floodtx.Price
	tx.data.GasLimit = floodtx.GasLimit
	tx.data.Recipient = floodtx.Recipient
	tx.data.Amount = floodtx.Amount
	tx.data.Payload = floodtx.Payload
	// Signature values
	tx.data.V = floodtx.V
	tx.data.R = floodtx.R
	tx.data.TxEnterType = floodtx.TxEnterType
	tx.data.IsEntrustTx = floodtx.IsEntrustTx
	tx.data.Extra = floodtx.Extra
	return tx
}
//YY
//func (tx *Transaction) SetTransactionMx(tx_Mx *Transaction_Mx)(txer SelfTransaction ){
//	txd := txdata{
//		AccountNonce:tx_Mx.Data.AccountNonce | params.NonceAddOne,
//		Price:tx_Mx.Data.Price,
//		GasLimit:tx_Mx.Data.GasLimit,
//		Recipient:tx_Mx.Data.Recipient,
//		Amount:tx_Mx.Data.Amount,
//		Payload:tx_Mx.Data.Payload,
//		// Signature values
//		V:     tx_Mx.Data.V,
//		R:     tx_Mx.Data.R,
//		S:     tx_Mx.Data.S,
//		TxEnterType : tx_Mx.Data.TxEnterType,
//		IsEntrustTx : tx_Mx.Data.IsEntrustTx,
//		Extra: tx_Mx.Data.Extra,
//	}
//	if len(tx_Mx.ExtraTo) > 0 {
//		mx := Matrix_Extra{
//			TxType:     tx_Mx.TxType_Mx,
//			LockHeight: tx_Mx.LockHeight,
//			ExtraTo:    tx_Mx.ExtraTo,
//		}
//		if mx.TxType == 0 {
//			mx.LockHeight = tx_Mx.LockHeight
//		}
//		txd.Extra = append(txd.Extra, mx)
//	}
//	newtx := &Transaction{data: txd}
//	txer = newtx
//	return
//}
//
////YY
//func (tx *Transaction)GetTransactionMx(stx SelfTransaction) *Transaction_Mx {
//	btx,ok:=stx.(*Transaction)
//	if !ok {
//		return nil
//	}
//	tx_Mx:=&Transaction_Mx{}
//	tx_Mx.Data.AccountNonce = btx.data.AccountNonce & params.NonceSubOne
//	tx_Mx.Data.Price = btx.data.Price
//	tx_Mx.Data.GasLimit = btx.data.GasLimit
//	tx_Mx.Data.Recipient = btx.data.Recipient
//	tx_Mx.Data.Amount = btx.data.Amount
//	tx_Mx.Data.Payload = btx.data.Payload
//	// Signature values
//	tx_Mx.Data.V = btx.data.V
//	tx_Mx.Data.R = btx.data.R
//	tx_Mx.Data.S = btx.data.S
//	tx_Mx.Data.TxEnterType = btx.data.TxEnterType
//	tx_Mx.Data.IsEntrustTx = btx.data.IsEntrustTx
//	tx_Mx.Data.Extra = btx.data.Extra
//	//tx_Mx.Data.Extra = append(tx_Mx.Data.Extra,tx.data.Extra[])
//	if len(btx.data.Extra) > 0 {
//		tx_Mx.TxType_Mx = btx.data.Extra[0].TxType
//		tx_Mx.LockHeight = btx.data.Extra[0].LockHeight
//		tx_Mx.ExtraTo = btx.data.Extra[0].ExtraTo
//	}
//	return tx_Mx
//}
func  ConvTxtoMxtx(txer SelfTransaction) *Transaction_Mx{
	tx,ok:=txer.(*Transaction)
	if !ok {
		return nil
	}
	tx_Mx:=&Transaction_Mx{}
	tx_Mx.Data.AccountNonce = tx.data.AccountNonce & params.NonceSubOne
	tx_Mx.Data.Price = tx.data.Price
	tx_Mx.Data.GasLimit = tx.data.GasLimit
	tx_Mx.Data.Recipient = tx.data.Recipient
	tx_Mx.Data.Amount = tx.data.Amount
	tx_Mx.Data.Payload = tx.data.Payload
	// Signature values
	tx_Mx.Data.V = tx.data.V
	tx_Mx.Data.R = tx.data.R
	tx_Mx.Data.S = tx.data.S
	tx_Mx.Data.TxEnterType = tx.data.TxEnterType
	tx_Mx.Data.IsEntrustTx = tx.data.IsEntrustTx
	tx_Mx.Data.Extra = tx.data.Extra
	//tx_Mx.Data.Extra = append(tx_Mx.Data.Extra,tx.data.Extra[])
	if len(tx.data.Extra) > 0 {
		tx_Mx.TxType_Mx = tx.data.Extra[0].TxType
		tx_Mx.LockHeight = tx.data.Extra[0].LockHeight
		tx_Mx.ExtraTo = tx.data.Extra[0].ExtraTo
	}
	return tx_Mx
}

func ConvMxtotx(tx_Mx *Transaction_Mx) *Transaction {
	txd := txdata{
		AccountNonce:tx_Mx.Data.AccountNonce | params.NonceAddOne,
		Price:tx_Mx.Data.Price,
		GasLimit:tx_Mx.Data.GasLimit,
		Recipient:tx_Mx.Data.Recipient,
		Amount:tx_Mx.Data.Amount,
		Payload:tx_Mx.Data.Payload,
		// Signature values
		V:     tx_Mx.Data.V,
		R:     tx_Mx.Data.R,
		S:     tx_Mx.Data.S,
		TxEnterType : tx_Mx.Data.TxEnterType,
		IsEntrustTx : tx_Mx.Data.IsEntrustTx,
		Extra: tx_Mx.Data.Extra,
	}
	if len(tx_Mx.ExtraTo) > 0 {
		mx := Matrix_Extra{
			TxType:     tx_Mx.TxType_Mx,
			LockHeight: tx_Mx.LockHeight,
			ExtraTo:    tx_Mx.ExtraTo,
		}
		if mx.TxType == 0 {
			mx.LockHeight = tx_Mx.LockHeight
		}
		txd.Extra = append(txd.Extra, mx)
	}
	tx := &Transaction{data: txd}
	return tx
}

//hezi
func (tx *Transaction) SetTxS(S *big.Int) { tx.data.S = S }

//func (tx *Transaction) SetTxN(N uint32) {tx.data.N = N}
//func (tx *Transaction) GetTxN() uint32{return tx.data.N}
//func (tx *Transaction) GetTxIsFlood() bool{return tx.data.IsFlood}

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *Transaction) To() *common.Address {
	//return tx.data.Recipient
	if tx.data.Recipient == nil {
		return nil
	}
	to := *tx.data.Recipient
	return &to
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (tx *Transaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// AsMessage returns the transaction as a core.Message.
//
// AsMessage requires a signer to derive the sender.
//
// XXX Rename message to something less arbitrary?

//YYY ================begin=========================
//func (tx *Transaction) AsMessage(s Signer) (Message, error) {
//	msg := Message{
//		nonce:      tx.data.AccountNonce,
//		gasLimit:   tx.data.GasLimit,
//		gasPrice:   new(big.Int).Set(tx.data.Price),
//		to:         tx.data.Recipient,
//		amount:     tx.data.Amount,
//		data:       tx.data.Payload,
//		checkNonce: true,
//	}
//	//YY
//	if len(tx.data.Extra) > 0 {
//		msg.extra = tx.data.Extra[0]
//	}
//	var err error
//	//YY ========begin=========
//	from, addrerr := tx.GetTxFrom()
//	if addrerr != nil {
//		msg.from, err = Sender(s, tx)
//	} else {
//		msg.from = from
//	}
//	//===========end=============
//	return msg, err
//}
//=======================end===============================

// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (SelfTransaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	//YY
	if len(cpy.data.Extra) > 0 {
		cpy.data.V.Add(cpy.data.V, big.NewInt(128))
	}
	return cpy, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Amount)
	return total
}

func (tx *Transaction) RawSignatureValues() (*big.Int, *big.Int, *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

// Transactions is a Transaction slice type for basic sorting.
type SelfTransactions []SelfTransaction

// Len returns the length of s.
func (s SelfTransactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s SelfTransactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (s SelfTransactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// TxDifference returns a new set t which is the difference between a to b.
func TxDifference(a, b SelfTransactions) (keep SelfTransactions) {
	keep = make(SelfTransactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce []SelfTransaction

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce() < s[j].Nonce() }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s *TxByNonce) Push(x interface{}) {
	//*s = append(*s, x.(*Transaction))
}

func (s *TxByNonce) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}


// TxByPrice implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type TxByPrice SelfTransactions

func (s TxByPrice) Len() int           { return len(s) }
func (s TxByPrice) Less(i, j int) bool { return s[i].GasPrice().Cmp(s[j].GasPrice()) > 0 }
func (s TxByPrice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *TxByPrice) Push(x interface{}) {
	//*s = append(*s, x.(*Transaction))
}

func (s *TxByPrice) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// TransactionsByPriceAndNonce represents a set of transactions that can return
// transactions in a profit-maximizing sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type TransactionsByPriceAndNonce struct {
	txs    map[common.Address]SelfTransactions // Per account nonce-sorted list of transactions
	//heads  TxByPrice                       // Next transaction for each unique account (price heap)
	heads TxByNonce
	signer Signer                          // Signer for the set of transactions
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
func NewTransactionsByPriceAndNonce(signer Signer, txs map[common.Address]SelfTransactions) *TransactionsByPriceAndNonce {
	// Initialize a price based heap with the head transactions
	heads := make(TxByNonce, 0, len(txs))
	for from, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := Sender(signer, accTxs[0])
		txs[acc] = accTxs[1:]
		if from != acc {
			delete(txs, from)
		}
	}
	//heap.Init(&heads)

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Peek returns the next transaction by price.
func (t *TransactionsByPriceAndNonce) Peek() SelfTransaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TransactionsByPriceAndNonce) Shift() {
	acc, _ := Sender(t.signer, t.heads[0])
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		t.heads[0], t.txs[acc] = txs[0], txs[1:]
		heap.Fix(&t.heads, 0)
	} else {
		heap.Pop(&t.heads)
	}
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *TransactionsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}

// Message is a fully derived transaction and implements core.Message
//
// NOTE: In a future PR this will be removed.

//YYY ================begin=========================
//type Message struct {
//	to         *common.Address
//	from       common.Address
//	nonce      uint64
//	amount     *big.Int
//	gasLimit   uint64
//	gasPrice   *big.Int
//	data       []byte
//	checkNonce bool
//	extra      Matrix_Extra //YY
//}
//
//func NewMessage(from common.Address, to *common.Address, nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, checkNonce bool) Message {
//	return Message{
//		from:       from,
//		to:         to,
//		nonce:      nonce,
//		amount:     amount,
//		gasLimit:   gasLimit,
//		gasPrice:   gasPrice,
//		data:       data,
//		checkNonce: checkNonce,
//	}
//}
//
//func (m Message) From() common.Address { return m.from }
//func (m Message) To() *common.Address  { return m.to }
//func (m Message) GasPrice() *big.Int   { return m.gasPrice }
//func (m Message) Value() *big.Int      { return m.amount }
//func (m Message) Gas() uint64          { return m.gasLimit }
//func (m Message) Nonce() uint64        { return m.nonce }
//func (m Message) Data() []byte         { return m.data }
//func (m Message) CheckNonce() bool     { return m.checkNonce }
//func (m Message) Extra() Matrix_Extra  { return m.extra } //YY
//YYY ====================end===================================
