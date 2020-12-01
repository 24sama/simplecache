package SimpleCache

type PeerPicker interface {
	// 根据key，选择节点，返回节点对应的http客户端
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// 从远程节点获取缓存值
	Get(group string, key string) ([]byte, error)
}
