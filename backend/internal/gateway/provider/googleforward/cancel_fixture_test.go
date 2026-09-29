package googleforward_test

import "context"

// cancelReadCloser 模拟读取层直接返回取消错误。
type cancelReadCloser struct{}

func (cancelReadCloser) Read([]byte) (int, error) { return 0, context.Canceled }
func (cancelReadCloser) Close() error             { return nil }
