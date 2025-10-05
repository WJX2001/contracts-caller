package serializers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"gorm.io/gorm/schema"
)

type RLPSerializer struct{}

func init() {
	schema.RegisterSerializer("rlp", RLPSerializer{})
}

func (RLPSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	if dbValue == nil {
		return nil
	}

	// 数据库读出来的值应该是字符串 (hex 编码的 RLP 字节)
	hexStr, ok := dbValue.(string)
	if !ok {
		return fmt.Errorf("expected hex string as the database value: %T", dbValue)
	}

	// 先把 hex 字符串解析成二进制
	b, err := hexutil.Decode(hexStr)
	if err != nil {
		return fmt.Errorf("failed to decode database value: %w", err)
	}

	// 为目标字段生成一个空值
	fieldValue := reflect.New(field.FieldType)

	// 用 RLP 解码，把二进制还原成 GO struct
	if err := rlp.DecodeBytes(b, fieldValue.Interface()); err != nil {
		return fmt.Errorf("failed to decode rlp bytes: %w", err)
	}

	// 把解码后的值设置到目标字段里
	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return nil
}

func (RLPSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue == nil || (field.FieldType.Kind() == reflect.Pointer && reflect.ValueOf(fieldValue).IsNil()) {
		return nil, nil
	}

	rlpBytes, err := rlp.EncodeToBytes(fieldValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encode rlp bytes: %w", err)
	}

	hexStr := hexutil.Encode(rlpBytes)
	return hexStr, nil
}
