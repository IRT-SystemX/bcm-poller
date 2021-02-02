package utils

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"math"
	"math/big"
	"strconv"
)

func FloatToString(val float64) string {
	return strconv.FormatFloat(val, 'g', 1, 64)
}

func IntToString(val int64) string {
	return strconv.FormatInt(val, 10)
}

func StringToFloat(val string) float64 {
	if len(val) == 0 {
		return 0
	}
	value, err := strconv.ParseFloat(val, 64)
	if err != nil {
		log.Fatal(err)
	}
	return value
}

func Percent(val float64, limit float64) string {
	return strconv.FormatInt(int64(math.Abs(val*100/limit)), 10) + "%%"
}

func Decode(res string) *big.Int {
	val, err := hexutil.DecodeBig(res)
	if err != nil {
		log.Fatal(err)
	}
	return val
}

func GetFunctionId(value string) string {
	return crypto.Keccak256Hash([]byte(value)).Hex()[:10]
}
