package cache

import (
	"distributed-cache/consistenthash"
	pb "distributed-cache/proto"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
)

// 提供被其他节点访问的能力（基于HTTP）

const (
	defaultBasePath = "/api/cache/"
	defaultReplicas = 50
)

// HTTP Server Pool
type HttpPool struct {
	// 记录URL地址，主机名、IP、端口号。
	selfPath string
	// 节点通讯地址的前缀，self+basePath开头的请求，就用于节点之间的访问。
	basePath string
	// 互斥锁
	mu sync.Mutex
	// 一致性Hash算法控制类
	consistentHashPool *consistenthash.Pool
	// 映射远程节点的的http client。keyed by e.g. "http://10.0.0.2:8008"
	httpClient map[string]*httpClient
}

// 初始化节点的HTTPPool
func NewHttpPool(selfPath string) *HttpPool {
	return &HttpPool{
		selfPath: selfPath,
		basePath: defaultBasePath,
	}
}

// 日志
func (p *HttpPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.selfPath, fmt.Sprintf(format, v...))
}

// ServeHttp处理所有的请求
func (p *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		log.Println("HTTPPool serving unexpected path: " + r.URL.Path)
		return
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group, ok := GetGroup(groupName)
	if !ok {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// 添加新节点，需要更新映射
func (p *HttpPool) Set(peersPath ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 一致性Hash算法初始化
	p.consistentHashPool = consistenthash.New(defaultReplicas, nil)
	p.consistentHashPool.Add(peersPath...)
	// 为每一个节点都初始化一个Http客户端
	p.httpClient = make(map[string]*httpClient, len(peersPath))
	for _, peerPath := range peersPath {
		p.httpClient[peerPath] = &httpClient{baseURL: peerPath + p.basePath}
	}
}

// PickerPeer() 包装了一致性哈希算法的 Get() 方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端。
func (p *HttpPool) PickPeer(key string) (PeerService, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 根据虚拟节点的key来查找真实节点的位置。
	if peerPath := p.consistentHashPool.Get(key); peerPath != "" && peerPath != p.selfPath {
		p.Log("Pick Peer %s", peerPath)
		return p.httpClient[peerPath], true
	}
	return nil, false
}

// HTTP客户端类
type httpClient struct {
	baseURL string
}

// 实现HTTP客户端接口
func (h *httpClient) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf(
		"%v/%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	// 获取返回值
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	// 返回体为[]byte
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}
