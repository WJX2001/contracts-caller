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

// Scan 方法：用于从数据库扫描数据并设置到目标值
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

	// 十六进制解码，把 0x 开头的 十六进制字符串 转化为 []byte
	b, err := hexutil.Decode(hexStr)
	if err != nil {
		return fmt.Errorf("failed to decode database value: %w", err)
	}

	// 构建字段值，创建原始类型的对象
	// 处理了字段本身是指针的情况，如果字段是 *T, 那么 reflect.New(field.Fieldtype) **T，代码会再向下取一层并为其分配内存，最多支持到双指针，超过则报错
	fieldValue := reflect.New(field.FieldType)
	fieldInterface := fieldValue.Interface()

	// 如果字段本身是一个指针类型，进入这个分支
	// 如果字段本身是一个指针类型，进入这个分支
	if field.FieldType.Kind() == reflect.Pointer {
		nestedField := fieldValue.Elem() // *T
		if nestedField.Elem().Kind() == reflect.Pointer {
			// 如果 T 还是一个指针，比如定义了 **SomeType 就相当于 ***SomeType 了
			// 直接报错 这里最多只能支持到双指针
			return fmt.Errorf("double pointers are the max depth supported: %T", fieldValue)
		}

		// field.FieldType.Elem() 表示去掉一层指针，比如 *common.Hash -> common.Hash
		// reflect.New(common.Hash) -> *common.Hash
		// 于是执行 Set，相当于 *T = new(T), 也就是说，给这根 *T 指针分配了一块内存
		nestedField.Set(reflect.New(field.FieldType.Elem()))
		// 便于后续断言判断
		fieldInterface = nestedField.Interface()
	}
	// 接口断言，必须实现 SetBytes([]byte) 方法
	fieldSetBytes, ok := fieldInterface.(SetBytesInterface)
	if !ok {
		return fmt.Errorf("field does not satisfy the `SetBytes([]byte)` interface: %T", fieldInterface)
	}
	// 调用自定义类型的 SetBytes 方法，把字节放进去
	fieldSetBytes.SetBytes(b)
	// 最终用 field.ReflectValueOf 获取到真实的 struct 字段位置并赋值
	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return nil
}

func (BytesSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue == nil || (field.FieldType.Kind() == reflect.Pointer && reflect.ValueOf(fieldValue).IsNil()) {
		return nil, nil
	}

	fieldBytes, ok := fieldValue.(BytesInterface)
	if !ok {
		return nil, fmt.Errorf("field does not satisfy the `Bytes() []byte` interface")
	}
	// 把字节编码成十六进制字符串
	hexStr := hexutil.Encode(fieldBytes.Bytes())
	return hexStr, nil
}
