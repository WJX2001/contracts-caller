package serializers

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	"github.com/jackc/pgtype"
	"gorm.io/gorm/schema"
)

// 把数据库中的数值和 Go 中的 *big.Int 类型（尤其是以太坊常用的 uint256 大整数互相转换）

/*
	数据库 Postgres 里的 NUMERIC 类型数值，和 Go 中的 *big.Int 并不是天然兼容的
	定义一个 自定义序列化器，让 GORM 可以：
		- Scan 反序列化：把数据库里的数值（NUMERIC/DECIMAL）读到 Go 的 *big.Int
		- Value(序列化)：把Go 的 *big.Int 存回数据库
*/

var (
	big10              = big.NewInt(10)
	u256BigIntOverflow = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
)

type U256Serializer struct{}

func init() {
	schema.RegisterSerializer("u256", U256Serializer{})
}

// 数据库 -> Go
func (U256Serializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	// 类型必须是 *big.Int 否则报错
	if dbValue == nil {
		return nil
	} else if field.FieldType != reflect.TypeOf((*big.Int)(nil)) {
		return fmt.Errorf("can only deserialize into a *big.Int: %T", field.FieldType)
	}

	// 用 pgtype.Numeric 解析 dbValue
	// numeric.Int 是整数部分
	// numeric.Exp 是指数部分
	numeric := new(pgtype.Numeric)
	err := numeric.Scan(dbValue)
	if err != nil {
		return err
	}

	bigInt := numeric.Int
	if numeric.Exp > 0 {
		factor := new(big.Int).Exp(big10, big.NewInt(int64(numeric.Exp)), nil)
		// 实际数据库值实际上是 bigInt * 10 ^ Exp
		// 数据库值 123e2 → 123 × 10^2 = 12300
		bigInt.Mul(bigInt, factor)
	}

	if bigInt.Cmp(u256BigIntOverflow) >= 0 {
		return fmt.Errorf("deserialized number larger than u256 can hold: %s", bigInt)
	}

	field.ReflectValueOf(ctx, dst).Set(reflect.ValueOf(bigInt))
	return nil
}

func (U256Serializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue == nil || (field.FieldType.Kind() == reflect.Pointer && reflect.ValueOf(fieldValue).IsNil()) {
		return nil, nil
	} else if field.FieldType != reflect.TypeOf((*big.Int)(nil)) {
		return nil, fmt.Errorf("can only serialize a *big.Int: %T", field.FieldType)
	}

	// 转成 pgtype.Numeric,接收 *big.Int  标记Status: pgtype.Present 表示非空
	numeric := pgtype.Numeric{Int: fieldValue.(*big.Int), Status: pgtype.Present}
	return numeric.Value()
}
