package mymiddleware

import (
	"reflect"
	"strconv"
	"sync"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	once     sync.Once
	validate *validator.Validate
}

func (c *Validator) Validate(i any) error {
	c.once.Do(func() {
		c.validate = validator.New()
	})
	setDefaults(i)
	return c.validate.Struct(i)
}

// 获取传入结构体的反射值和类型
// 递归处理嵌套结构体字段
// 获取每个字段的tag
// 判断字段值是否为默认零值
// 如果是,根据字段类型设置为tag中的default值
// 增加对更多字段类型的处理
// 使用这个函数时，传进来的结构体字段一定要导出（大写）
func setDefaults(p any) {
	if p == nil {
		return
	}

	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	setStructDefaults(v)
}

func setStructDefaults(v reflect.Value) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		switch field.Kind() {
		case reflect.Struct:
			setStructDefaults(field)
		case reflect.Pointer:
			if !field.IsNil() && field.Elem().Kind() == reflect.Struct {
				setStructDefaults(field.Elem())
			}
		}

		if !field.CanSet() {
			continue
		}

		tag := fieldType.Tag.Get("default")
		if tag == "" || !field.IsZero() {
			continue
		}

		_ = setDefaultValue(field, tag)
	}
}

func setDefaultValue(field reflect.Value, tag string) bool {
	switch field.Kind() {
	case reflect.String:
		field.SetString(tag)
		return true
	case reflect.Bool:
		if v, err := strconv.ParseBool(tag); err == nil {
			field.SetBool(v)
			return true
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v, err := strconv.ParseInt(tag, 10, field.Type().Bits()); err == nil {
			field.SetInt(v)
			return true
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if v, err := strconv.ParseUint(tag, 10, field.Type().Bits()); err == nil {
			field.SetUint(v)
			return true
		}
	case reflect.Float32, reflect.Float64:
		if v, err := strconv.ParseFloat(tag, field.Type().Bits()); err == nil {
			field.SetFloat(v)
			return true
		}
	case reflect.Pointer:
		elem := reflect.New(field.Type().Elem()).Elem()
		if !setDefaultValue(elem, tag) {
			return false
		}
		ptr := reflect.New(field.Type().Elem())
		ptr.Elem().Set(elem)
		field.Set(ptr)
		return true
	}

	return false
}
