# 用于FCI中切换外部系统回调对应版本应用的插件及工具的第2个版本
- 利用Go httpUtil重写了反向代理服务器，取消了nginx层面
- 为了配合fci的极简部署，利用http接口来控制各种路由的转发及请求头设置(这个nginx插件不具备)，且无需重启服务
- 单一的转发目标服务

# callback2.vue
这个是fci前端的插件，放入FCI，配置route即可

# callback_switch
这个是已经编译好的linux应用
 - sysConf.yml配置主http接口服务
 - reverseConf.yml这是反向代理服务及路由转发配置

# 后端代码
    main.go
    reverseProxy.go

# maybe todo
- [ ] 负载均衡-这个得进行反向代理服务负载、压力测试后进行
- [ ] 更多灵活的转发配置
- [X] 支持删除请求头
- [ ] 启动后的代理转发放在内存中，重启后就丢失了：希望进行持久化，放入redis中
