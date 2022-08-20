package cache

/*
	缓存值的抽象与封装
*/

// ByteView 用来表示缓存值，只读
type ByteView struct {
	b []byte
}

// 返回长度, 实现了lru中的 value 接口
func (v ByteView) Len() int {
	return len(v.b)
}

// 返回一个byte数组的切片
func (v ByteView) ByteSlice() []byte {
	c := make([]byte, len(v.b))
	copy(c, v.b)
	return c
}

// 返回string类型的值
func (v ByteView) String() string {
	return string(v.b)
}