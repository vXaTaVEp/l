package l

type Config interface {
	Level() string
	Path() string
	Console() bool
	Async() bool // 是否异步写入
}
