package mymiddleware

import (
	"github.com/go-playground/validator/v10"
	"reflect"
	"strconv"
	"sync"
)

var DefaultValidator = &Validator{}

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
	v := reflect.ValueOf(p).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() == reflect.Struct {
			setDefaults(f.Addr().Interface())
			continue
		}
		tag := t.Field(i).Tag.Get("default")
		if tag != "" && f.Interface() == reflect.Zero(f.Type()).Interface() {
			switch f.Kind() {
			case reflect.String:
				f.SetString(tag)
			case reflect.Int:
				if v, err := strconv.Atoi(tag); err == nil {
					f.SetInt(int64(v))
				}
			case reflect.Float64:
				//if  v, err := strconv.Atoi(tag); err == nil{
				//	f.SetFloat(v)
				//}
				// handle other types
			}
		}
	}
}
