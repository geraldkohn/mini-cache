package cache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// 提供被其他节点访问的能力（基于HTTP）


const defaultBasePath = "/api/cache/"

type HttpPool struct {
	// 记录URL地址，主机名、IP、端口号。
	self string
	// 节点通讯地址的前缀，self+basePath开头的请求，就用于节点之间的访问。
	basePath string
}

// 初始化节点的HTTPPool
func NewHttpPool(self string) *HttpPool {
	return &HttpPool{
		self: self,
		basePath: defaultBasePath,
	}
}

// 日志
func (p *HttpPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
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

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

