package SimpleCache

import (
	"SimpleCache/consistenthash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_simplecache/"
	defaultReplicas = 50
)

// 服务端
type HTTPPool struct {
	self        string // 记录自己的地址，包括主机IP和端口
	basePath    string // 作为节点间通讯地址的前缀，例：http://example.com/_simplecache/开头的请求，均用于节点间访问
	mu          sync.Mutex
	peers       *consistenthash.Map    // 一致性哈希算法的map，用来根据key选择节点
	httpGetters map[string]*HttpGetter // 映射远程节点与对应的httpGetter。每一个节点对应一个httpGetter。例："http://10.0.0.2:8008"
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s]%s", p.self, fmt.Sprintf(format, v))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// 约定的访问路径 /<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.Log("Hit")
	w.Header().Set("Content-Type", "application/octet-stream")
	// 将缓存的值作为httpResponse返回
	_, _ = w.Write(view.ByteSlice())
}

// 实例化一致性哈希算法，并且添加传入的节点
// 并为每个节点创建一个http客户端
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*HttpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &HttpGetter{baseURL: peer + p.basePath}
	}
}

// 包装一致性哈希算法的Get()方法，根据key返回对应节点的HTTP客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// HTTP客户端
type HttpGetter struct {
	// 将要访问的远程节点地址
	baseURL string
}

// 创建HTTP客户端从远程节点获取缓存值
func (h *HttpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// 利用强制类型转换，在编译期检查结构体HttpGetter是否实现了PeerGetter接口，
// 将空值nil转换为*HttpGetter类型，再转换为PeerGetter接口，如果转换失败，则说明HttpGetter并没有实现PeerGetter所有方法
var _ PeerGetter = (*HttpGetter)(nil)
var _ PeerPicker = (*HTTPPool)(nil)
