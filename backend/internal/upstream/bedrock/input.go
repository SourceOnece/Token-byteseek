// RouteInput 是已完成提供商模型映射的当次区域投影，不携带凭据。
package bedrock

type RouteInput struct {
	Region      string
	ForceGlobal bool
	Model       string
}
