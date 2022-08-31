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

	"gopkg.in/yaml.v3"
)

// 请求头设置
type ReqHeader struct {
	Name  string `yaml:"Name"`
	Value string `yaml:"Value"`
}

// 路由设置
type Route struct {
	Name       string       `yaml:"Name"`
	Host       string       `yaml:"Host"`
	Path       string       `yaml:"Path"`
	RePath     string       `yaml:"RePath"`
	ReqHeaders []*ReqHeader `yaml:"ReqHeaders"`
}

func (this *Route) IsRouteEmpty() bool {
	if strings.TrimSpace(this.Path) == "" || strings.TrimSpace(this.RePath) == "" {
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

// 支持正则路由版本
func ProxyRequestHandler2(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, sr := range RvConf.Confs {
			if sr.IsSigleReverseEmpty() {
				log.Printf("[reverseConf.yml]中有[conf]为空, 请检查。")
				continue
			}
			for _, rt := range sr.Routes {
				if rt.IsRouteEmpty() {
					log.Printf("[reverseConf.yml]中有[conf:%s]有[route]为空, 请检查。\n", sr.Name)
					continue
				}
				rexp, err := regexp.Compile(rt.Path)
				if err != nil {
					continue
				}
				matches := rexp.FindStringSubmatch(r.URL.Path)
				if len(matches) == 0 {
					continue
				}
				newPath := rt.RePath
				for i, m := range matches {
					if i == 0 {
						continue
					}

					newPath = strings.ReplaceAll(newPath, fmt.Sprintf("{$%d}", i), m)

				} // 优化原版转发代码
				//fmt.Println(newPath)
				proxy.Director = func(req *http.Request) {
					target, err := url.Parse(strings.TrimSpace(sr.Target))
					if err != nil {
						log.Printf("转发的URL有误：%s", sr.Target)
					}
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
					for _, srh := range rt.ReqHeaders {
						req.Header.Set(srh.Name, srh.Value)
					}
					if strings.TrimSpace(rt.Host) != "" {
						req.Host = rt.Host
					}
					req.URL.Path = newPath
					log.Printf("url:%s, host:%s, xff:%s\n", req.URL.String(), req.Host, req.Header.Get("X-Forwarded-For"))
				}
				proxy.ServeHTTP(w, r)
				goto end //只找1个路由
			}
		}
	end:
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
	rp, _ := NewReverseProxy("http://127.0.0.1:80")
	if err != nil {
		log.Printf("启动反向代理异常[http://127.0.0.1:80]：%s", err.Error())
	}
	http.HandleFunc("/", ProxyRequestHandler2(rp))
	server := &http.Server{Addr: RvConf.ProxyServ, Handler: nil}
	log.Fatal(server.ListenAndServe())
}
