package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// 请求头设置
type ReqHeader struct {
	Name  string `yaml:"Name"`
	Value string `yaml:"Value"`
}

// 路由设置
type Route struct {
	Name        string       `yaml:"Name"`
	Host        string       `yaml:"Host"`
	PathPattern string       `yaml:"PathPattern"`
	RePath      string       `yaml:"RePath"`
	ReqHeaders  []*ReqHeader `yaml:"ReqHeaders"`
}

func (this *Route) IsRouteEmpty() bool {
	if strings.TrimSpace(this.PathPattern) == "" || strings.TrimSpace(this.RePath) == "" {
		return true
	}
	return false
}

// 单个反向设置
type SigleReverse struct {
	Name   string   `yaml:"Name"`
	Target string   `yaml:"Target"`
	Routes []*Route `yaml:"Routes"`
}

func (this *SigleReverse) IsSigleReverseEmpty() bool {
	if strings.TrimSpace(this.Target) == "" || len(this.Routes) == 0 {
		return true
	}
	return false
}

// 简单反向代理设置
type ReverseConf struct {
	ProxyServ string          `yaml:"ProxyServ"`
	Confs     []*SigleReverse `yaml:"Confs"`
}

func (this *ReverseConf) IsReverseConfEmpty() bool {

	if strings.TrimSpace(this.ProxyServ) == "" || len(this.Confs) == 0 {
		return true
	}
	return false
}

// 拷贝原版代码
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// 拷贝原版代码
func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

// 初始化反向代理配置
func InitReverseConf(confPath string) error {
	yamlBytes, err := ioutil.ReadFile(confPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlBytes, RvConf)

	if err != nil {
		return err
	}
	return nil
}

// 构建反向代理服务
func NewReverseProxy(defaultTarget string) (*httputil.ReverseProxy, error) {
	baseTarget, err := url.Parse(defaultTarget)
	if err != nil {
		return nil, err
	}
	rp := httputil.NewSingleHostReverseProxy(baseTarget)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //忽略SSL证书
	}
	rp.Transport = transport
	return rp, nil
}

// 目标路由结构，由于存放路由查找结果
type TargetRoute struct {
	TargetUrl string
	NewPath   string
	RouteConf *Route
	sync.Mutex
}

func (this *TargetRoute) Init() {
	this.TargetUrl = ""
	this.NewPath = ""
	this.RouteConf = nil
}

var tRoute = &TargetRoute{}

// 分治找route
func FindRoute(wg *sync.WaitGroup, sr *SigleReverse, sourcePath string) {
	defer func() {
		wg.Done()
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	if sr.IsSigleReverseEmpty() {
		log.Printf("[reverseConf.yml]中有[conf]为空, 请检查。")
		return
	}
	if len(sr.Routes) > 0 {
		for _, rt := range sr.Routes {
			if tRoute.TargetUrl != "" { // 已经找到，中止
				return
			}
			if rt.IsRouteEmpty() { // 当前route配额为空
				log.Printf("[reverseConf.yml]中有[conf:%s]有[route]为空, 请检查。\n", sr.Name)
				continue
			}

			rexp, err := regexp.Compile(rt.PathPattern)
			if err != nil {
				continue
			}
			matches := rexp.FindStringSubmatch(sourcePath)
			if len(matches) == 0 {
				continue
			}
			tRoute.Lock() // 匹配后，读写双锁，并结束查找
			defer tRoute.Unlock()
			if strings.TrimSpace(tRoute.TargetUrl) == "" {
				newPath := rt.RePath
				for i, m := range matches {
					if i == 0 {
						continue
					}
					newPath = strings.ReplaceAll(newPath, fmt.Sprintf("{$%d}", i), m)
				}
				tRoute.TargetUrl = sr.Target // 设置新的转发地址
				tRoute.NewPath = newPath     // 设置新的路径
				tRoute.RouteConf = rt        // 设置对应的路由配置
			}
			return
		}
	}
}

// 运行反向代理
func RunReverseProxyServ() {
	err := InitReverseConf("reverseConf.yml")
	if err != nil {
		log.Fatalln(err)
	}
	if RvConf.IsReverseConfEmpty() {
		log.Panicln("[reverseConf.yml]有误, 请检查。")
		return
	}
	rp, _ := NewReverseProxy("http://127.0.0.1:80")                     //此处选用一个默认80端口做一个初始化而已，无任何特殊作用
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { // 支持正则路由版本--多协程查找route
		var wg sync.WaitGroup
		// 进行锁定保护
		tRoute.Init()
		for _, sr := range RvConf.Confs {
			go FindRoute(&wg, sr, r.URL.Path)
			wg.Add(1)
		}
		wg.Wait()
		// fmt.Printf("tRoute: %v\n", tRoute)
		if strings.TrimSpace(tRoute.TargetUrl) != "" { // 再次检查下匹配的转发route
			rp.Director = func(req *http.Request) {
				target, err := url.Parse(strings.TrimSpace(tRoute.TargetUrl))
				if err != nil {
					log.Printf("转发的URL有误：%s", tRoute.TargetUrl)
				}
				// log.Printf("转发的URL：%s", tRoute.TargetUrl)
				targetQuery := target.RawQuery
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)
				if targetQuery == "" || req.URL.RawQuery == "" {
					req.URL.RawQuery = targetQuery + req.URL.RawQuery
				} else {
					req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
				}
				if _, ok := req.Header["User-Agent"]; !ok {
					req.Header.Set("User-Agent", "")
				}
				xff := strings.TrimSpace(req.Header.Get("X-Forwarded-For"))
				if xff != "" {
					xff = xff + "," + strings.Split(req.RemoteAddr, ":")[0]
				} else {
					xff = strings.Split(req.RemoteAddr, ":")[0]
				}
				req.Header.Set("X-Forwarded-For", xff)
				for _, srh := range tRoute.RouteConf.ReqHeaders {
					req.Header.Set(srh.Name, srh.Value)
				}
				if strings.TrimSpace(tRoute.RouteConf.Host) != "" {
					req.Host = tRoute.RouteConf.Host
				}
				req.URL.Path = tRoute.NewPath
				log.Printf("url:%s, host:%s, X-Forwarded-For:%s\n", req.URL.String(), req.Host, req.Header.Get("X-Forwarded-For"))
			}
		}
		rp.ServeHTTP(w, r)
	})
	server := &http.Server{Addr: RvConf.ProxyServ, Handler: nil}
	log.Fatal(server.ListenAndServe())
}
