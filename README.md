# 一个反向代理服务器
- 利用Go httpUtil编写的反向代理服务器
- 可以通过http api进行转发服务器的动态修改
- 可以通过http api进行路由的动态设置请求头

# 配置说明
 - sysConf.yml配置主http接口服务，比如http api的服务端口,比如8088
 - reverseConf.yml这是反向代理服务的端口及路由转发配置,比如82。
   repath中{$1}、{$2}...标识path正则匹配中的子匹配1、子匹配2...

# 编译
`cmd
go get "github.com/kataras/iris"
go get "gopkg.in/yaml.v3"
`

`go build`

# 动态修改转发及设置请求头相关的http api
- 查看当前的转发配置

  http://127.0.0.1:8088/getReverseInfo
- 设置指定服务名的新的转发地址
  
  http://127.0.0.1:8088/setNewTarget/{revsName:string}?newTarget={newTargetHostPort:string-urlencode}
- 设置指定服务名的新的转发地址和1个请求头
  
  http://127.0.0.1:8088/setFciReveser/{revsName:string}?newTarget={newTargetHostPort:string-urlencode}&headerName={headerName:string-urlencode}&headerValue={headerValue:string-urlencode}
- 设置指定服务名的新的转发地址和删除1个请求头
  
  http://127.0.0.1:8088/setFciReveser/{revsName:string}?newTarget={newTargetHostPort:string-urlencode}&headerName={headerName:string-urlencode}
- 设置指定服务名和路由的请求头

  http://127.0.0.1:8088/setNewHeader/{revsName:string}/{routeName:string}?headerName={headerName:string-urlencode}&headerValue={headerValue:string-urlencode}
- 设置指定服务名和路由，删除请求头

  http://127.0.0.1:8088/setNewHeader/{revsName:string}/{routeName:string}?headerName={headerName:string-urlencode}
  
- 举例（服务器ip为192.168.52.56），get请求，修改dd_social_pay的新转发地址为http://10.110.2.254:80 设置请求头Environment-Label为test_4：
  
  http://192.168.52.56:8088/setFciReveser/dd_social_pay?newTarget=http://10.110.2.254:80&headerName=Environment-Label&headerValue=test_4
  

# maybe todo
- [ ] 负载均衡-这个得进行反向代理服务负载、压力测试后进行
- [X] 更多灵活的转发配置
- [X] 支持删除请求头
- [ ] 启动后的代理转发放在内存中，重启后就丢失了：希望进行持久化，放入redis中
