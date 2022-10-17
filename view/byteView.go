package view

// 缓存值的抽象与封装

// ByteView 用来表示缓存值，只读
type ByteView struct {
	B []byte
}

// 返回长度, 实现了lru中的 value 接口
func (v ByteView) Len() uint64 {
	return uint64(len(v.B))
}

// 返回一个byte数组的切片
func (v ByteView) ByteSlice() []byte {
	c := make([]byte, len(v.B))
	copy(c, v.B)
	return c
}

// 返回string类型的值
func (v ByteView) String() string {
	return string(v.B)
}
