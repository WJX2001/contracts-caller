package serializers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"gorm.io/gorm/schema"
)

/*
	在以太坊中很多类型 common.Hash common.Address 等，本质都是二进制数据，但是数据库一般习惯用十六进制字符串来存储 0x1234..
	如果直接使用 []byte 存储到数据库，读取和查看都很不方便
	自定义序列化器：
		- 写入数据库时（Value 方法）：把字段的 []byte 转成 0x... 形式的十六进制字符串
		- 读取数据库时（Scan 方法）：把十六进制字符串解析成 []byte 并赋值回对应的结构体字段
*/

type BytesSerializer struct{}
type BytesInterface interface{ Bytes() []byte }
type SetBytesInterface interface{ SetBytes([]byte) }

func (BytesSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	// 空值检查
	if dbValue == nil {
		return nil
	}

	// 类型断言 - 期望十六进制字符串
	hexStr, ok := dbValue.(string)
	if !ok {
		return fmt.Errorf("expected hex string as the database value: %T", dbValue)
	}

	// 十六进制解码
	b, err := hexutil.Decode(hexStr)
	if err != nil {
		return fmt.Errorf("failed to decode database value: %w", err)
	}

	// 创建字段值
	fieldValue := reflect.New(field.FieldType)
	fieldInterface := fieldValue.Interface()

	// 处理指针类型
	if field.FieldType.Kind() == reflect.Pointer {

	}

}
