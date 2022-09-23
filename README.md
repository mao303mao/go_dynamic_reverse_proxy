# 一个反向代理服务器
- 利用Go httpUtil编写的反向代理服务器
- 可以通过http api进行转发服务器的动态修改
- 可以通过http api进行路由的动态设置请求头

# 配置说明
 - sysConf.yml配置主http接口服务
 - reverseConf.yml这是反向代理服务及路由转发配置

# 编译
`go build`

# http api
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
  

# maybe todo
- [ ] 负载均衡-这个得进行反向代理服务负载、压力测试后进行
- [X] 更多灵活的转发配置
- [X] 支持删除请求头
- [ ] 启动后的代理转发放在内存中，重启后就丢失了：希望进行持久化，放入redis中
